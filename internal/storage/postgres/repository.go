package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/storage"
)

const DriverName = "pgx"

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, errors.New("database url is required")
	}
	db, err := sql.Open(DriverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}
	return &Repository{db: db}, nil
}

func New(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("postgres db is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Repository) UpsertSchedulerRun(ctx context.Context, record storage.SchedulerRunRecord) error {
	if err := validateSchedulerRun(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO scheduler_runs (
  run_id, tenant_id, source_id, source_adapter, datasets, observation_date,
  dry_run, status, started_at, completed_at, events_built, events_published,
  provider_requests, provider_retries, failures, config, report, error_message,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12,
  $13, $14, $15, $16, $17, $18,
  now()
)
ON CONFLICT (run_id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  source_adapter = EXCLUDED.source_adapter,
  datasets = EXCLUDED.datasets,
  observation_date = EXCLUDED.observation_date,
  dry_run = EXCLUDED.dry_run,
  status = EXCLUDED.status,
  started_at = EXCLUDED.started_at,
  completed_at = EXCLUDED.completed_at,
  events_built = EXCLUDED.events_built,
  events_published = EXCLUDED.events_published,
  provider_requests = EXCLUDED.provider_requests,
  provider_retries = EXCLUDED.provider_retries,
  failures = EXCLUDED.failures,
  config = EXCLUDED.config,
  report = EXCLUDED.report,
  error_message = EXCLUDED.error_message,
  updated_at = now()`,
		record.RunID,
		record.TenantID,
		record.SourceID,
		record.SourceAdapter,
		pqArray(record.Datasets),
		record.ObservationDate,
		record.DryRun,
		record.Status,
		record.StartedAt,
		record.CompletedAt,
		record.EventsBuilt,
		record.EventsPublished,
		record.ProviderRequests,
		record.ProviderRetries,
		record.Failures,
		jsonOrEmpty(record.ConfigJSON),
		jsonOrEmpty(record.ReportJSON),
		nullString(record.ErrorMessage),
	)
	if err != nil {
		return fmt.Errorf("upsert scheduler run: %w", err)
	}
	return nil
}

func (r *Repository) InsertProviderUsage(ctx context.Context, record storage.ProviderUsageRecord) error {
	if err := validateProviderUsage(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO provider_usage_runs (
  usage_id, run_id, provider, dataset, request_count, retry_count, event_count, budget
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (usage_id) DO UPDATE SET
  run_id = EXCLUDED.run_id,
  provider = EXCLUDED.provider,
  dataset = EXCLUDED.dataset,
  request_count = EXCLUDED.request_count,
  retry_count = EXCLUDED.retry_count,
  event_count = EXCLUDED.event_count,
  budget = EXCLUDED.budget`,
		record.UsageID,
		record.RunID,
		record.Provider,
		record.Dataset,
		record.RequestCount,
		record.RetryCount,
		record.EventCount,
		jsonOrEmpty(record.BudgetJSON),
	)
	if err != nil {
		return fmt.Errorf("insert provider usage: %w", err)
	}
	return nil
}

func validateSchedulerRun(record storage.SchedulerRunRecord) error {
	if strings.TrimSpace(record.RunID) == "" {
		return errors.New("scheduler run id is required")
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("scheduler tenant id is required")
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return errors.New("scheduler source id is required")
	}
	if strings.TrimSpace(record.SourceAdapter) == "" {
		return errors.New("scheduler source adapter is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return errors.New("scheduler status is required")
	}
	if record.StartedAt.IsZero() {
		return errors.New("scheduler started at is required")
	}
	if record.ObservationDate.IsZero() {
		return errors.New("scheduler observation date is required")
	}
	return nil
}

func validateProviderUsage(record storage.ProviderUsageRecord) error {
	if strings.TrimSpace(record.UsageID) == "" {
		return errors.New("provider usage id is required")
	}
	if strings.TrimSpace(record.RunID) == "" {
		return errors.New("provider usage run id is required")
	}
	if strings.TrimSpace(record.Provider) == "" {
		return errors.New("provider is required")
	}
	if strings.TrimSpace(record.Dataset) == "" {
		return errors.New("provider usage dataset is required")
	}
	return nil
}

func jsonOrEmpty(value []byte) []byte {
	if len(value) == 0 {
		return []byte(`{}`)
	}
	return value
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func pqArray(values []string) stringArray {
	return stringArray(values)
}

type stringArray []string

func (a stringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	escaped := make([]string, 0, len(a))
	for _, value := range a {
		value = strings.ReplaceAll(value, `\`, `\\`)
		value = strings.ReplaceAll(value, `"`, `\"`)
		escaped = append(escaped, `"`+value+`"`)
	}
	return `{` + strings.Join(escaped, ",") + `}`, nil
}
