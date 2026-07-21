package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type equityReconciliationRepository interface {
	storage.EquityReconciliationRepository
	storage.PublishRepository
	ListMarketOpsAssets(ctx context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error)
}

type equityReconciliationConfig struct {
	TenantID            string
	SourceID            string
	UniverseGroup       string
	Environment         string
	ObservationDate     time.Time
	DryRun              bool
	AcknowledgeWrites   bool
	MaxProviderAttempts int
	MaxProviderRequests int
	Deadline            time.Duration
	RetryBackoffs       []time.Duration
	NormalizationPoll   time.Duration
	LeaseDuration       time.Duration
	RequeueFailed       bool
}

type equityReconciliationReport struct {
	ObservationDate   string         `json:"observation_date"`
	DryRun            bool           `json:"dry_run"`
	UniverseGroup     string         `json:"universe_group"`
	UniverseAssets    int            `json:"universe_assets"`
	InitiallyComplete int            `json:"initially_complete"`
	MissingSymbols    []string       `json:"missing_symbols,omitempty"`
	Enqueued          int            `json:"enqueued"`
	Requeued          int            `json:"requeued"`
	Claimed           int            `json:"claimed"`
	RawReplays        int            `json:"raw_replays"`
	ProviderRequests  int            `json:"provider_requests"`
	Published         int            `json:"published"`
	FinalComplete     int            `json:"final_complete"`
	StatusCounts      map[string]int `json:"status_counts,omitempty"`
	Failures          []string       `json:"failures,omitempty"`
	DeadlineReached   bool           `json:"deadline_reached"`
}

func runEquityReconciliation(
	ctx context.Context,
	cfg equityReconciliationConfig,
	repo equityReconciliationRepository,
	client massive.ScheduledPullClient,
	publisher broker.Publisher,
) (equityReconciliationReport, error) {
	cfg = normalizeEquityReconciliationConfig(cfg)
	report := equityReconciliationReport{
		ObservationDate: cfg.ObservationDate.UTC().Format("2006-01-02"),
		DryRun:          cfg.DryRun,
		UniverseGroup:   cfg.UniverseGroup,
		StatusCounts:    map[string]int{},
	}
	if repo == nil {
		return report, errors.New("reconciliation repository is required")
	}
	if !cfg.DryRun && !cfg.AcknowledgeWrites {
		return report, errors.New("reconciliation writes require --acknowledge-writes")
	}
	if !cfg.DryRun && (client == nil || publisher == nil) {
		return report, errors.New("provider client and broker publisher are required for reconciliation writes")
	}

	assets, err := repo.ListMarketOpsAssets(ctx, cfg.TenantID, cfg.UniverseGroup, true, 50)
	if err != nil {
		return report, fmt.Errorf("load active equity universe: %w", err)
	}
	report.UniverseAssets = len(assets)
	if len(assets) != 50 {
		return report, fmt.Errorf("active universe %s contains %d assets; expected 50", cfg.UniverseGroup, len(assets))
	}

	now := time.Now().UTC()
	for _, asset := range assets {
		found, err := repo.HasNormalizedEquity(ctx, cfg.TenantID, cfg.SourceID, asset.Ticker, cfg.ObservationDate)
		if err != nil {
			return report, err
		}
		if found {
			report.InitiallyComplete++
			continue
		}
		report.MissingSymbols = append(report.MissingSymbols, asset.Ticker)
		if cfg.DryRun {
			continue
		}
		nextAttemptAt := now.Add(cfg.RetryBackoffs[0])
		if _, rawErr := repo.FindRawEquityEvent(ctx, cfg.TenantID, cfg.SourceID, asset.Ticker, cfg.ObservationDate); rawErr == nil {
			nextAttemptAt = now
		} else if !errors.Is(rawErr, storage.ErrNotFound) {
			return report, rawErr
		}
		task, err := repo.EnqueueEquityReconciliationTask(ctx, storage.EquityReconciliationTaskRecord{
			TaskID:              stableReconciliationTaskID(cfg, asset.Ticker),
			TenantID:            cfg.TenantID,
			SourceID:            cfg.SourceID,
			UniverseGroup:       cfg.UniverseGroup,
			Dataset:             massive.DatasetEquityEODPrices,
			ObservationDate:     cfg.ObservationDate,
			Symbol:              asset.Ticker,
			UniverseRank:        asset.Rank,
			Status:              storage.EquityReconciliationStatusQueued,
			MaxProviderAttempts: cfg.MaxProviderAttempts,
			NextAttemptAt:       nextAttemptAt,
		})
		if err != nil {
			return report, fmt.Errorf("enqueue %s: %w", asset.Ticker, err)
		}
		if task.CreatedAt.Equal(task.UpdatedAt) {
			report.Enqueued++
		}
	}
	if cfg.DryRun {
		report.FinalComplete = report.InitiallyComplete
		return report, nil
	}
	if cfg.RequeueFailed {
		report.Requeued, err = repo.RequeueFailedEquityReconciliationTasks(ctx, cfg.TenantID, cfg.SourceID, cfg.UniverseGroup, cfg.ObservationDate, now.Add(cfg.RetryBackoffs[0]))
		if err != nil {
			return report, err
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, cfg.Deadline)
	defer cancel()
	assetBySymbol := make(map[string]storage.MarketOpsAssetRecord, len(assets))
	for _, asset := range assets {
		assetBySymbol[strings.ToUpper(asset.Ticker)] = asset
	}

	for {
		if runCtx.Err() != nil {
			report.DeadlineReached = true
			break
		}
		task, claimErr := repo.ClaimNextEquityReconciliationTask(
			runCtx, cfg.TenantID, cfg.SourceID, cfg.UniverseGroup,
			cfg.ObservationDate, time.Now().UTC(), cfg.LeaseDuration,
		)
		if claimErr != nil {
			if !errors.Is(claimErr, storage.ErrNotFound) {
				return report, claimErr
			}
			tasks, err := repo.ListEquityReconciliationTasks(runCtx, cfg.TenantID, cfg.SourceID, cfg.UniverseGroup, cfg.ObservationDate)
			if err != nil {
				return report, err
			}
			wait, active := nextReconciliationWait(tasks, time.Now().UTC(), cfg.NormalizationPoll)
			if !active {
				break
			}
			if err := sleepReconciliation(runCtx, wait); err != nil {
				report.DeadlineReached = true
				break
			}
			continue
		}
		report.Claimed++
		asset, ok := assetBySymbol[strings.ToUpper(task.Symbol)]
		if !ok {
			failReconciliationTask(runCtx, repo, &task, fmt.Sprintf("symbol %s is no longer active", task.Symbol))
			continue
		}
		if err := processEquityReconciliationTask(runCtx, cfg, repo, client, publisher, asset, &task, &report); err != nil {
			report.Failures = appendBounded(report.Failures, fmt.Sprintf("%s: %v", task.Symbol, err), 25)
		}
	}

	tasks, err := repo.ListEquityReconciliationTasks(ctx, cfg.TenantID, cfg.SourceID, cfg.UniverseGroup, cfg.ObservationDate)
	if err != nil {
		return report, err
	}
	for _, task := range tasks {
		report.StatusCounts[task.Status]++
	}
	for _, asset := range assets {
		found, checkErr := repo.HasNormalizedEquity(ctx, cfg.TenantID, cfg.SourceID, asset.Ticker, cfg.ObservationDate)
		if checkErr != nil {
			return report, checkErr
		}
		if found {
			report.FinalComplete++
		}
	}
	if report.DeadlineReached {
		return report, fmt.Errorf("equity reconciliation deadline reached with %d/50 normalized", report.FinalComplete)
	}
	if report.FinalComplete != len(assets) {
		return report, fmt.Errorf("equity reconciliation incomplete: %d/%d normalized", report.FinalComplete, len(assets))
	}
	return report, nil
}

func processEquityReconciliationTask(
	ctx context.Context,
	cfg equityReconciliationConfig,
	repo equityReconciliationRepository,
	client massive.ScheduledPullClient,
	publisher broker.Publisher,
	asset storage.MarketOpsAssetRecord,
	task *storage.EquityReconciliationTaskRecord,
	report *equityReconciliationReport,
) error {
	if found, err := repo.HasNormalizedEquity(ctx, cfg.TenantID, cfg.SourceID, task.Symbol, cfg.ObservationDate); err != nil {
		return err
	} else if found {
		return succeedReconciliationTask(ctx, repo, task)
	}

	raw, err := repo.FindRawEquityEvent(ctx, cfg.TenantID, cfg.SourceID, task.Symbol, cfg.ObservationDate)
	if err == nil {
		task.RawEventID = raw.EventID
		task.IdempotencyKey = raw.IdempotencyKey
		if task.ReplayCount >= 1 {
			return failReconciliationTask(ctx, repo, task, "raw event remained unnormalized after one replay")
		}
		if err := publishReconciliationReplay(ctx, cfg, publisher, *task, raw); err != nil {
			return rescheduleOrFailReconciliation(ctx, cfg, repo, task, err, false)
		}
		task.ReplayCount++
		report.RawReplays++
		if err := awaitTaskNormalization(ctx, cfg, repo, task); err != nil {
			return err
		}
		return nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if report.ProviderRequests >= cfg.MaxProviderRequests {
		return failReconciliationTask(ctx, repo, task, fmt.Sprintf("global provider request budget %d exhausted", cfg.MaxProviderRequests))
	}
	if task.ProviderAttempts >= task.MaxProviderAttempts {
		return failReconciliationTask(ctx, repo, task, "provider attempts exhausted")
	}

	task.ProviderAttempts++
	report.ProviderRequests++
	pullReport, pullErr := massive.RunScheduledPull(ctx, massive.ScheduledPullConfig{
		TenantID:        cfg.TenantID,
		SourceID:        cfg.SourceID,
		Environment:     cfg.Environment,
		ObservationDate: cfg.ObservationDate,
		Companies: []massive.MegacapCompanySeed{{
			Rank: asset.Rank, Ticker: asset.Ticker, TickerKey: asset.TickerKey,
			Company: asset.Company, CompanyKey: asset.CompanyKey,
			Sector: asset.Sector, SectorKey: asset.SectorKey,
			Industry: asset.Industry, IndustryKey: asset.IndustryKey,
		}},
		IncludeEquityEOD:    true,
		IncludeOptions:      false,
		MaxRetries:          0,
		MaxProviderRequests: 1,
		MaxEventsBuilt:      1,
		MaxEventsPublished:  1,
		DryRun:              false,
		ContinueOnError:     false,
		PublishRepository:   repo,
	}, client, publisher)
	if pullErr != nil || pullReport.Failures > 0 || pullReport.EventsPublished != 1 {
		if pullErr == nil {
			pullErr = fmt.Errorf("provider pull published %d event with %d failure", pullReport.EventsPublished, pullReport.Failures)
		}
		return rescheduleOrFailReconciliation(ctx, cfg, repo, task, pullErr, true)
	}
	report.Published++
	raw, rawErr := repo.FindRawEquityEvent(ctx, cfg.TenantID, cfg.SourceID, task.Symbol, cfg.ObservationDate)
	if rawErr == nil {
		task.RawEventID = raw.EventID
		task.IdempotencyKey = raw.IdempotencyKey
	}
	if err := awaitTaskNormalization(ctx, cfg, repo, task); err != nil {
		return err
	}
	return nil
}

func awaitTaskNormalization(ctx context.Context, cfg equityReconciliationConfig, repo equityReconciliationRepository, task *storage.EquityReconciliationTaskRecord) error {
	lease := time.Now().UTC().Add(cfg.LeaseDuration)
	task.Status = storage.EquityReconciliationStatusAwaitingNormalization
	task.LeaseExpiresAt = &lease
	task.LastError = ""
	if err := repo.UpdateEquityReconciliationTask(ctx, *task); err != nil {
		return err
	}
	waitLimit := cfg.RetryBackoffs[0]
	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	for {
		found, err := repo.HasNormalizedEquity(waitCtx, cfg.TenantID, cfg.SourceID, task.Symbol, cfg.ObservationDate)
		if err == nil && found {
			return succeedReconciliationTask(ctx, repo, task)
		}
		if err != nil {
			return err
		}
		if err := sleepReconciliation(waitCtx, cfg.NormalizationPoll); err != nil {
			break
		}
	}
	task.Status = storage.EquityReconciliationStatusQueued
	task.NextAttemptAt = time.Now().UTC()
	task.LeaseExpiresAt = nil
	task.LastError = "normalization not observed before poll window"
	return repo.UpdateEquityReconciliationTask(ctx, *task)
}

func publishReconciliationReplay(ctx context.Context, cfg equityReconciliationConfig, publisher broker.Publisher, task storage.EquityReconciliationTaskRecord, raw storage.RawEventLedgerRecord) error {
	var payload map[string]any
	if err := json.Unmarshal(raw.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("decode raw replay payload: %w", err)
	}
	replayID := "equity-reconcile:" + task.TaskID
	payload["replay_job_id"] = replayID
	metadata, _ := payload["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["replay"] = map[string]any{
		"reconciliation_task_id": task.TaskID,
		"replayed_at":            time.Now().UTC().Format(time.RFC3339Nano),
	}
	payload["metadata"] = metadata
	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode raw replay payload: %w", err)
	}
	topic := strings.TrimSpace(raw.BrokerTopic)
	if topic == "" {
		topic = broker.TopicName(cfg.Environment, broker.RawTopic)
	}
	_, err = publisher.Publish(ctx, broker.Message{
		Topic: topic,
		Key:   raw.IdempotencyKey,
		Value: value,
		Headers: map[string]string{
			"content_type":                     "application/json",
			"signalops_replay_job_id":          replayID,
			"signalops_replay_source_kind":     storage.ReplaySourceRaw,
			"signalops_reconciliation_task_id": task.TaskID,
		},
		CorrelationID: replayID,
		CausationID:   raw.EventID,
		TraceID:       replayID,
		PublishedAt:   time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("publish raw replay: %w", err)
	}
	return nil
}

func rescheduleOrFailReconciliation(ctx context.Context, cfg equityReconciliationConfig, repo equityReconciliationRepository, task *storage.EquityReconciliationTaskRecord, cause error, providerFailure bool) error {
	if providerFailure && task.ProviderAttempts < task.MaxProviderAttempts {
		index := task.ProviderAttempts
		if index >= len(cfg.RetryBackoffs) {
			index = len(cfg.RetryBackoffs) - 1
		}
		task.Status = storage.EquityReconciliationStatusQueued
		task.NextAttemptAt = time.Now().UTC().Add(cfg.RetryBackoffs[index])
		task.LeaseExpiresAt = nil
		task.LastError = cause.Error()
		if err := repo.UpdateEquityReconciliationTask(ctx, *task); err != nil {
			return err
		}
		return cause
	}
	return failReconciliationTask(ctx, repo, task, cause.Error())
}

func succeedReconciliationTask(ctx context.Context, repo equityReconciliationRepository, task *storage.EquityReconciliationTaskRecord) error {
	now := time.Now().UTC()
	task.Status = storage.EquityReconciliationStatusSucceeded
	task.CompletedAt = &now
	task.LeaseExpiresAt = nil
	task.NextAttemptAt = now
	task.LastError = ""
	return repo.UpdateEquityReconciliationTask(ctx, *task)
}

func failReconciliationTask(ctx context.Context, repo equityReconciliationRepository, task *storage.EquityReconciliationTaskRecord, message string) error {
	now := time.Now().UTC()
	task.Status = storage.EquityReconciliationStatusFailed
	task.CompletedAt = &now
	task.LeaseExpiresAt = nil
	task.NextAttemptAt = now
	task.LastError = message
	if err := repo.UpdateEquityReconciliationTask(ctx, *task); err != nil {
		return err
	}
	return errors.New(message)
}

func normalizeEquityReconciliationConfig(cfg equityReconciliationConfig) equityReconciliationConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.SourceID = strings.TrimSpace(cfg.SourceID)
	cfg.UniverseGroup = strings.TrimSpace(cfg.UniverseGroup)
	if cfg.UniverseGroup == "" {
		cfg.UniverseGroup = "top50_megacap"
	}
	if cfg.MaxProviderAttempts <= 0 {
		cfg.MaxProviderAttempts = 2
	}
	if cfg.MaxProviderRequests <= 0 {
		cfg.MaxProviderRequests = 100
	}
	if cfg.Deadline <= 0 {
		cfg.Deadline = 15 * time.Minute
	}
	if cfg.NormalizationPoll <= 0 {
		cfg.NormalizationPoll = 5 * time.Second
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 2 * time.Minute
	}
	if len(cfg.RetryBackoffs) == 0 {
		cfg.RetryBackoffs = []time.Duration{30 * time.Second, 2 * time.Minute}
	}
	return cfg
}

func parseRetryBackoffs(value string) ([]time.Duration, error) {
	parts := strings.Split(value, ",")
	backoffs := make([]time.Duration, 0, len(parts))
	for _, part := range parts {
		delay, err := time.ParseDuration(strings.TrimSpace(part))
		if err != nil || delay <= 0 {
			return nil, fmt.Errorf("invalid retry backoff %q", part)
		}
		backoffs = append(backoffs, delay)
	}
	if len(backoffs) == 0 {
		return nil, errors.New("at least one retry backoff is required")
	}
	return backoffs, nil
}

func nextReconciliationWait(tasks []storage.EquityReconciliationTaskRecord, now time.Time, poll time.Duration) (time.Duration, bool) {
	var wait time.Duration
	active := false
	for _, task := range tasks {
		switch task.Status {
		case storage.EquityReconciliationStatusQueued:
			active = true
			candidate := task.NextAttemptAt.Sub(now)
			if candidate < 0 {
				candidate = 0
			}
			if wait == 0 || candidate < wait {
				wait = candidate
			}
		case storage.EquityReconciliationStatusRunning, storage.EquityReconciliationStatusAwaitingNormalization:
			active = true
			candidate := poll
			if task.LeaseExpiresAt != nil {
				candidate = task.LeaseExpiresAt.Sub(now)
				if candidate < 0 {
					candidate = 0
				}
			}
			if wait == 0 || candidate < wait {
				wait = candidate
			}
		}
	}
	if active && wait <= 0 {
		wait = poll
	}
	if wait > poll {
		wait = poll
	}
	return wait, active
}

func stableReconciliationTaskID(cfg equityReconciliationConfig, symbol string) string {
	value := strings.Join([]string{
		cfg.TenantID, cfg.SourceID, cfg.UniverseGroup, massive.DatasetEquityEODPrices,
		cfg.ObservationDate.UTC().Format("2006-01-02"), strings.ToUpper(strings.TrimSpace(symbol)),
	}, "|")
	sum := sha256.Sum256([]byte(value))
	return "eqrec_" + hex.EncodeToString(sum[:16])
}

func sleepReconciliation(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func appendBounded(values []string, value string, limit int) []string {
	if len(values) >= limit {
		return values
	}
	return append(values, value)
}
