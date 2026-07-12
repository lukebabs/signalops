package backtest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/marketops/dsm"
	"github.com/lukebabs/signalops/internal/signals"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type Config struct {
	RunID                string
	TenantID             string
	AppID                string
	Domain               string
	UseCase              string
	SourceID             string
	SourceAdapter        string
	Dataset              string
	DetectorID           string
	DetectorVersion      string
	RequestedBy          string
	WindowStart          time.Time
	WindowEnd            time.Time
	Symbols              []string
	MaxRecords           int
	BatchSize            int
	AutoAcceptConfidence float64
	PythonBin            string
}

type Metrics struct {
	RunID                string         `json:"run_id"`
	Scanned              int            `json:"scanned"`
	Signals              int            `json:"signals"`
	Artifacts            int            `json:"artifacts"`
	GraphProposals       int            `json:"graph_proposals"`
	PolicyResults        int            `json:"policy_results"`
	RecommendationCounts map[string]int `json:"recommendation_counts"`
	Batches              int            `json:"batches"`
	MaxRecords           int            `json:"max_records"`
	BatchSize            int            `json:"batch_size"`
	StartedAt            string         `json:"started_at"`
	CompletedAt          string         `json:"completed_at,omitempty"`
}

func execute(ctx context.Context, repo storage.MarketOpsBacktestRepository, cfg Config, metrics *Metrics) error {
	remaining := cfg.MaxRecords
	if remaining <= 0 {
		remaining = 50
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 || batchSize > remaining {
		batchSize = remaining
	}
	offset := 0
	evaluator := dsm.NewPolicyEvaluator(cfg.AutoAcceptConfidence)
	for remaining > 0 {
		limit := batchSize
		if remaining < limit {
			limit = remaining
		}
		events, err := repo.ListMarketOpsBacktestNormalizedEvents(ctx, storage.MarketOpsBacktestEventFilter{TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase, SourceID: cfg.SourceID, SourceAdapter: cfg.SourceAdapter, Dataset: cfg.Dataset, Symbols: cfg.Symbols, WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, Limit: limit, Offset: offset})
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}
		metrics.Batches++
		metrics.Scanned += len(events)
		signalValues, err := runDetectorBatch(ctx, cfg, events)
		if err != nil {
			return err
		}
		signalsBatch := []storage.MarketOpsBacktestSignalRecord{}
		artifactsBatch := []storage.MarketOpsBacktestArtifactRecord{}
		proposalsBatch := []storage.MarketOpsBacktestGraphProposalRecord{}
		policyBatch := []storage.MarketOpsBacktestPolicyResultRecord{}
		for i, value := range signalValues {
			record, err := signals.LedgerRecordFromEventJSON(value, broker.ConsumedMessage{Message: broker.Message{Topic: "marketops.backtest.signal.v1", Key: fmt.Sprintf("%s:%d", cfg.RunID, i), Value: value, CorrelationID: cfg.RunID}, Partition: -1, Offset: int64(metrics.Signals + i)})
			if err != nil {
				return err
			}
			signalsBatch = append(signalsBatch, storage.MarketOpsBacktestSignalRecord{RunID: cfg.RunID, SignalLedgerRecord: record})
			artifacts, err := dsm.ExtractArtifacts(record)
			if err != nil {
				return err
			}
			for _, artifact := range artifacts {
				artifactsBatch = append(artifactsBatch, storage.MarketOpsBacktestArtifactRecord{RunID: cfg.RunID, MarketOpsDSMArtifactRecord: artifact})
				proposals, err := dsm.ExtractGraphProposals(artifact)
				if err != nil {
					return err
				}
				for _, proposal := range proposals {
					proposalsBatch = append(proposalsBatch, storage.MarketOpsBacktestGraphProposalRecord{RunID: cfg.RunID, MarketOpsDSMGraphProposalRecord: proposal})
					policy, err := evaluator.Evaluate(cfg.RunID, proposal)
					if err != nil {
						return err
					}
					policyBatch = append(policyBatch, policy)
					metrics.RecommendationCounts[policy.Recommendation]++
				}
			}
		}
		if err := repo.PersistMarketOpsBacktestBatch(ctx, storage.MarketOpsBacktestRunRecord{RunID: cfg.RunID}, signalsBatch, artifactsBatch, proposalsBatch, policyBatch); err != nil {
			return err
		}
		metrics.Signals += len(signalsBatch)
		metrics.Artifacts += len(artifactsBatch)
		metrics.GraphProposals += len(proposalsBatch)
		metrics.PolicyResults += len(policyBatch)
		remaining -= len(events)
		offset += len(events)
	}
	return nil
}

func runDetectorBatch(ctx context.Context, cfg Config, events []storage.NormalizedEventLedgerRecord) ([][]byte, error) {
	var stdin bytes.Buffer
	enc := json.NewEncoder(&stdin)
	for _, event := range events {
		mapped, err := normalizedEventForDetector(event)
		if err != nil {
			return nil, err
		}
		if err := enc.Encode(mapped); err != nil {
			return nil, err
		}
	}
	cmd := exec.CommandContext(ctx, cfg.PythonBin, "-m", "signalops_workers.backtest_detector", "--detector-id", cfg.DetectorID, "--worker-id", "marketops-backtest")
	cmd.Stdin = &stdin
	cmd.Env = pythonEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run python backtest detector: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	out := [][]byte{}
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		out = append(out, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizedEventForDetector(record storage.NormalizedEventLedgerRecord) (map[string]any, error) {
	payload := map[string]any{}
	if len(record.NormalizedPayload) > 0 {
		if err := json.Unmarshal(record.NormalizedPayload, &payload); err != nil {
			return nil, err
		}
	}
	entities := []any{}
	if len(record.EntitiesJSON) > 0 {
		_ = json.Unmarshal(record.EntitiesJSON, &entities)
	}
	evidence := []any{}
	if len(record.EvidenceJSON) > 0 {
		_ = json.Unmarshal(record.EvidenceJSON, &evidence)
	}
	return map[string]any{
		"event_id": record.EventID, "tenant_id": record.TenantID, "source_id": record.SourceID,
		"app_id": record.AppID, "domain": record.Domain, "use_case": record.UseCase, "source_domain": record.Domain,
		"source_adapter": record.SourceAdapter, "ingestion_mode": "scheduled_pull", "dataset": record.Dataset,
		"idempotency_key": record.IdempotencyKey, "correlation_id": record.EventID,
		"observation_time": record.ObservationTime.Format(time.RFC3339Nano), "effective_time": record.ObservationTime.Format(time.RFC3339Nano), "processing_time": time.Now().UTC().Format(time.RFC3339Nano),
		"normalized_payload": payload, "entities": entities, "evidence": evidence,
	}, nil
}

type Result struct {
	Run     storage.MarketOpsBacktestRunRecord
	Metrics Metrics
}

func Run(ctx context.Context, repo storage.MarketOpsBacktestRepository, cfg Config) (Result, error) {
	if repo == nil {
		return Result{}, errors.New("marketops backtest repository is required")
	}
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return Result{}, err
	}
	filters, _ := json.Marshal(map[string]any{"symbols": cfg.Symbols, "max_records": cfg.MaxRecords})
	params, _ := json.Marshal(map[string]any{"detector_id": cfg.DetectorID, "detector_version": cfg.DetectorVersion, "auto_accept_confidence": cfg.AutoAcceptConfidence})
	now := time.Now().UTC()
	runRecord := storage.MarketOpsBacktestRunRecord{
		RunID: cfg.RunID, TenantID: cfg.TenantID, AppID: cfg.AppID, Domain: cfg.Domain, UseCase: cfg.UseCase,
		SourceID: cfg.SourceID, SourceAdapter: cfg.SourceAdapter, Dataset: cfg.Dataset, DetectorID: cfg.DetectorID,
		DetectorVersion: cfg.DetectorVersion, Status: storage.RunStatusStarted, RequestedBy: cfg.RequestedBy,
		WindowStart: cfg.WindowStart, WindowEnd: cfg.WindowEnd, StartedAt: now, FiltersJSON: filters, ParametersJSON: params, MetricsJSON: []byte(`{}`),
	}
	if err := repo.CreateMarketOpsBacktestRun(ctx, runRecord); err != nil {
		return Result{}, err
	}
	metrics := Metrics{RunID: cfg.RunID, MaxRecords: cfg.MaxRecords, BatchSize: cfg.BatchSize, StartedAt: now.Format(time.RFC3339Nano), RecommendationCounts: map[string]int{}}
	runErr := execute(ctx, repo, cfg, &metrics)
	completedAt := time.Now().UTC()
	metrics.CompletedAt = completedAt.Format(time.RFC3339Nano)
	metricsJSON, marshalErr := json.Marshal(metrics)
	if marshalErr != nil {
		return Result{}, marshalErr
	}
	if runErr != nil {
		record, _ := repo.FailMarketOpsBacktestRun(context.Background(), cfg.RunID, completedAt, runErr.Error(), metricsJSON)
		return Result{Run: record, Metrics: metrics}, runErr
	}
	record, err := repo.CompleteMarketOpsBacktestRun(ctx, cfg.RunID, completedAt, metricsJSON)
	if err != nil {
		return Result{}, err
	}
	return Result{Run: record, Metrics: metrics}, nil
}

func pythonEnv(env []string) []string {
	for i, item := range env {
		if strings.HasPrefix(item, "PYTHONPATH=") {
			env[i] = item + string(os.PathListSeparator) + "python"
			return env
		}
	}
	return append(env, "PYTHONPATH=python")
}

func (cfg Config) withDefaults() Config {
	if strings.TrimSpace(cfg.AppID) == "" {
		cfg.AppID = "marketops"
	}
	if strings.TrimSpace(cfg.Domain) == "" {
		cfg.Domain = "market_data"
	}
	if strings.TrimSpace(cfg.UseCase) == "" {
		cfg.UseCase = "daily_market_surveillance"
	}
	if strings.TrimSpace(cfg.SourceAdapter) == "" {
		cfg.SourceAdapter = "market_data.massive"
	}
	if strings.TrimSpace(cfg.Dataset) == "" {
		cfg.Dataset = "equity_eod_prices"
	}
	if strings.TrimSpace(cfg.DetectorID) == "" {
		cfg.DetectorID = "marketops.dsm.taxonomy_v1"
	}
	if strings.TrimSpace(cfg.RequestedBy) == "" {
		cfg.RequestedBy = "operator-local"
	}
	if strings.TrimSpace(cfg.PythonBin) == "" {
		cfg.PythonBin = "python3"
	}
	if cfg.MaxRecords <= 0 {
		cfg.MaxRecords = 50
	}
	if cfg.BatchSize <= 0 || cfg.BatchSize > cfg.MaxRecords {
		cfg.BatchSize = cfg.MaxRecords
	}
	if cfg.AutoAcceptConfidence <= 0 || cfg.AutoAcceptConfidence > 1 {
		cfg.AutoAcceptConfidence = 0.75
	}
	for i, symbol := range cfg.Symbols {
		cfg.Symbols[i] = strings.ToUpper(strings.TrimSpace(symbol))
	}
	return cfg
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.RunID) == "" {
		return errors.New("run_id is required")
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if cfg.WindowStart.IsZero() || cfg.WindowEnd.IsZero() {
		return errors.New("window_start and window_end are required")
	}
	if !cfg.WindowEnd.After(cfg.WindowStart) {
		return errors.New("window_end must be after window_start")
	}
	if cfg.MaxRecords <= 0 || cfg.MaxRecords > 1000 {
		return errors.New("max_records must be between 1 and 1000")
	}
	return nil
}
