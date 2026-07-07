package massive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/pkg/broker"
	"github.com/lukebabs/signalops/pkg/contracts"
)

const (
	ScheduledPullRoute = "massive_scheduled_pull"

	defaultPublishTimeout = 5 * time.Second
	maxReportErrors       = 25
)

type ScheduledPullClient interface {
	GetEquityDailyBar(ctx context.Context, symbol string, date time.Time) (EquityEODPriceRecord, error)
	ListOptionContracts(ctx context.Context, underlying string, asOf time.Time, limit int) ([]OptionContractDailyRecord, error)
}

type ScheduledPullConfig struct {
	TenantID          string
	SourceID          string
	Environment       string
	RawTopic          string
	ObservationDate   time.Time
	ProcessingAt      time.Time
	Companies         []MegacapCompanySeed
	IncludeEquityEOD  bool
	IncludeOptions    bool
	OptionsLimit      int
	DryRun            bool
	ContinueOnError   bool
	PublishTimeout    time.Duration
	RequestDelay      time.Duration
	MaxRetries        int
	RetryBackoff      time.Duration
	CorrelationPrefix string
	TraceID           string
}

type ScheduledPullReport struct {
	ObservationDate  time.Time      `json:"observation_date"`
	DryRun           bool           `json:"dry_run"`
	Topic            string         `json:"topic"`
	Companies        int            `json:"companies"`
	EventsBuilt      int            `json:"events_built"`
	EventsPublished  int            `json:"events_published"`
	EventsByDataset  map[string]int `json:"events_by_dataset"`
	ProviderRequests int            `json:"provider_requests"`
	ProviderRetries  int            `json:"provider_retries"`
	Failures         int            `json:"failures"`
	Errors           []string       `json:"errors,omitempty"`
}

func RunScheduledPull(ctx context.Context, cfg ScheduledPullConfig, client ScheduledPullClient, publisher broker.Publisher) (ScheduledPullReport, error) {
	cfg = normalizeScheduledPullConfig(cfg)
	report := ScheduledPullReport{
		ObservationDate: cfg.ObservationDate,
		DryRun:          cfg.DryRun,
		Topic:           cfg.RawTopic,
		EventsByDataset: map[string]int{},
	}

	if client == nil {
		return report, errors.New("massive scheduled pull client is required")
	}
	if !cfg.DryRun && publisher == nil {
		return report, errors.New("broker publisher is required when dry run is disabled")
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return report, errors.New("tenant id is required")
	}
	if strings.TrimSpace(cfg.SourceID) == "" {
		return report, errors.New("source id is required")
	}
	if cfg.ObservationDate.IsZero() {
		return report, errors.New("observation date is required")
	}
	if !cfg.IncludeEquityEOD && !cfg.IncludeOptions {
		return report, errors.New("at least one scheduled dataset must be enabled")
	}

	companies := cfg.Companies
	if len(companies) == 0 {
		loaded, err := TopMegacapCompanies()
		if err != nil {
			return report, fmt.Errorf("load megacap seed universe: %w", err)
		}
		companies = loaded
	}
	report.Companies = len(companies)

	for _, company := range companies {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		symbol := normalizeSymbol(company.Ticker)
		if symbol == "" {
			if shouldStop := recordPullFailure(&report, cfg, "empty ticker in megacap seed"); shouldStop {
				return report, errors.New("empty ticker in megacap seed")
			}
			continue
		}

		if cfg.IncludeEquityEOD {
			record, err := fetchWithRetry(ctx, cfg, &report, func() (EquityEODPriceRecord, error) {
				return client.GetEquityDailyBar(ctx, symbol, cfg.ObservationDate)
			})
			if err != nil {
				if shouldStop := recordPullFailure(&report, cfg, fmt.Sprintf("%s equity eod: %v", symbol, err)); shouldStop {
					return report, err
				}
			} else if err := buildAndMaybePublish(ctx, cfg, publisher, &report, DatasetEquityEODPrices, func(adapterCfg AdapterConfig) (contracts.RawSignalEvent, error) {
				return BuildEquityEODPriceEvent(adapterCfg, record)
			}); err != nil {
				if shouldStop := recordPullFailure(&report, cfg, fmt.Sprintf("%s equity publish: %v", symbol, err)); shouldStop {
					return report, err
				}
			}
		}

		if cfg.IncludeOptions {
			records, err := fetchWithRetry(ctx, cfg, &report, func() ([]OptionContractDailyRecord, error) {
				return client.ListOptionContracts(ctx, symbol, cfg.ObservationDate, cfg.OptionsLimit)
			})
			if err != nil {
				if shouldStop := recordPullFailure(&report, cfg, fmt.Sprintf("%s option contracts: %v", symbol, err)); shouldStop {
					return report, err
				}
				continue
			}
			for _, record := range records {
				record.UnderlyingSymbol = firstNonEmptyString(record.UnderlyingSymbol, symbol)
				if err := buildAndMaybePublish(ctx, cfg, publisher, &report, DatasetOptionsContractsDaily, func(adapterCfg AdapterConfig) (contracts.RawSignalEvent, error) {
					return BuildOptionContractDailyEvent(adapterCfg, record)
				}); err != nil {
					if shouldStop := recordPullFailure(&report, cfg, fmt.Sprintf("%s option publish: %v", symbol, err)); shouldStop {
						return report, err
					}
				}
			}
		}
	}

	if report.Failures > 0 && !cfg.ContinueOnError {
		return report, errors.New("massive scheduled pull failed")
	}
	return report, nil
}

func fetchWithRetry[T any](ctx context.Context, cfg ScheduledPullConfig, report *ScheduledPullReport, fetch func() (T, error)) (T, error) {
	var zero T
	attempts := cfg.MaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if err := waitForRequestSlot(ctx, cfg); err != nil {
			return zero, err
		}
		report.ProviderRequests++
		result, err := fetch()
		if err == nil {
			return result, nil
		}
		if attempt == attempts-1 {
			return zero, err
		}
		report.ProviderRetries++
		if err := waitForRetry(ctx, cfg, attempt+1); err != nil {
			return zero, err
		}
	}
	return zero, errors.New("provider fetch retry loop exhausted")
}

func waitForRequestSlot(ctx context.Context, cfg ScheduledPullConfig) error {
	if cfg.RequestDelay <= 0 {
		return ctx.Err()
	}
	return sleepContext(ctx, cfg.RequestDelay)
}

func waitForRetry(ctx context.Context, cfg ScheduledPullConfig, attempt int) error {
	if cfg.RetryBackoff <= 0 {
		return ctx.Err()
	}
	return sleepContext(ctx, time.Duration(attempt)*cfg.RetryBackoff)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func buildAndMaybePublish(ctx context.Context, cfg ScheduledPullConfig, publisher broker.Publisher, report *ScheduledPullReport, dataset string, build func(AdapterConfig) (contracts.RawSignalEvent, error)) error {
	event, err := build(AdapterConfig{
		TenantID:      cfg.TenantID,
		SourceID:      cfg.SourceID,
		CorrelationID: correlationIDFor(cfg, dataset),
		TraceID:       cfg.TraceID,
		ProcessingAt:  cfg.ProcessingAt,
	})
	if err != nil {
		return err
	}
	report.EventsBuilt++
	report.EventsByDataset[dataset]++
	if cfg.DryRun {
		return nil
	}
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal raw event: %w", err)
	}
	publishCtx := ctx
	cancel := func() {}
	if cfg.PublishTimeout > 0 {
		publishCtx, cancel = context.WithTimeout(ctx, cfg.PublishTimeout)
	}
	defer cancel()
	_, err = publisher.Publish(publishCtx, broker.Message{
		Topic:         cfg.RawTopic,
		Key:           event.IdempotencyKey,
		Value:         value,
		Headers:       scheduledPullHeaders(event),
		CorrelationID: event.CorrelationID,
		CausationID:   event.CausationID,
		TraceID:       event.TraceID,
		PublishedAt:   time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("publish raw event: %w", err)
	}
	report.EventsPublished++
	return nil
}

func normalizeScheduledPullConfig(cfg ScheduledPullConfig) ScheduledPullConfig {
	if strings.TrimSpace(cfg.Environment) == "" {
		cfg.Environment = broker.DefaultEnvironment
	}
	if strings.TrimSpace(cfg.RawTopic) == "" {
		cfg.RawTopic = broker.TopicName(cfg.Environment, broker.RawTopic)
	}
	if cfg.PublishTimeout == 0 {
		cfg.PublishTimeout = defaultPublishTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.ProcessingAt.IsZero() {
		cfg.ProcessingAt = time.Now().UTC()
	}
	if cfg.ObservationDate.IsZero() {
		cfg.ObservationDate = previousUTCDate(time.Now().UTC())
	} else {
		day, err := dayUTC(cfg.ObservationDate, "observation_date")
		if err == nil {
			cfg.ObservationDate = day
		}
	}
	return cfg
}

func previousUTCDate(now time.Time) time.Time {
	utc := now.UTC().AddDate(0, 0, -1)
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func scheduledPullHeaders(event contracts.RawSignalEvent) map[string]string {
	return map[string]string{
		"content_type":             "application/json",
		"signalops_event_id":       event.EventID,
		"signalops_idempotency":    event.IdempotencyKey,
		"signalops_ingest_route":   ScheduledPullRoute,
		"signalops_ingest_format":  "raw_signal_event.v1",
		"signalops_dataset":        event.Dataset,
		"signalops_source_adapter": event.SourceAdapter,
	}
}

func correlationIDFor(cfg ScheduledPullConfig, dataset string) string {
	prefix := strings.TrimSpace(cfg.CorrelationPrefix)
	if prefix == "" {
		prefix = "massive-scheduled"
	}
	return strings.Join([]string{prefix, dataset, dateKey(cfg.ObservationDate)}, ":")
}

func recordPullFailure(report *ScheduledPullReport, cfg ScheduledPullConfig, message string) bool {
	report.Failures++
	if len(report.Errors) < maxReportErrors {
		report.Errors = append(report.Errors, message)
	}
	return !cfg.ContinueOnError
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
