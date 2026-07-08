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

func (r *Repository) UpsertIdempotencyRecord(ctx context.Context, record storage.IdempotencyRecord) error {
	if err := validateIdempotencyRecord(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO idempotency_records (
  tenant_id, source_id, idempotency_key, event_id, source_adapter, dataset,
  topic, partition, offset_value, payload_hash, status, metadata, last_seen_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12, now()
)
ON CONFLICT (tenant_id, source_id, idempotency_key) DO UPDATE SET
  event_id = EXCLUDED.event_id,
  source_adapter = EXCLUDED.source_adapter,
  dataset = EXCLUDED.dataset,
  topic = EXCLUDED.topic,
  partition = EXCLUDED.partition,
  offset_value = EXCLUDED.offset_value,
  payload_hash = EXCLUDED.payload_hash,
  status = EXCLUDED.status,
  metadata = EXCLUDED.metadata,
  last_seen_at = now()`,
		record.TenantID,
		record.SourceID,
		record.IdempotencyKey,
		record.EventID,
		record.SourceAdapter,
		record.Dataset,
		nullString(record.Topic),
		record.Partition,
		record.Offset,
		nullString(record.PayloadHash),
		record.Status,
		jsonOrEmpty(record.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert idempotency record: %w", err)
	}
	return nil
}

func (r *Repository) UpsertRawEventLedger(ctx context.Context, record storage.RawEventLedgerRecord) error {
	if err := validateRawEventLedger(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO raw_event_ledger (
  event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key,
  observation_time, processing_time, broker_topic, broker_partition, broker_offset,
  payload, entity_hints
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11,
  $12, $13
)
ON CONFLICT (event_id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  source_adapter = EXCLUDED.source_adapter,
  dataset = EXCLUDED.dataset,
  idempotency_key = EXCLUDED.idempotency_key,
  observation_time = EXCLUDED.observation_time,
  processing_time = EXCLUDED.processing_time,
  broker_topic = EXCLUDED.broker_topic,
  broker_partition = EXCLUDED.broker_partition,
  broker_offset = EXCLUDED.broker_offset,
  payload = EXCLUDED.payload,
  entity_hints = EXCLUDED.entity_hints`,
		record.EventID,
		record.TenantID,
		record.SourceID,
		record.SourceAdapter,
		record.Dataset,
		record.IdempotencyKey,
		record.ObservationTime,
		record.ProcessingTime,
		nullString(record.BrokerTopic),
		record.BrokerPartition,
		record.BrokerOffset,
		jsonOrEmpty(record.PayloadJSON),
		jsonArrayOrEmpty(record.EntityHintsJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert raw event ledger: %w", err)
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

func validateIdempotencyRecord(record storage.IdempotencyRecord) error {
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("idempotency tenant id is required")
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return errors.New("idempotency source id is required")
	}
	if strings.TrimSpace(record.IdempotencyKey) == "" {
		return errors.New("idempotency key is required")
	}
	if strings.TrimSpace(record.EventID) == "" {
		return errors.New("idempotency event id is required")
	}
	if strings.TrimSpace(record.SourceAdapter) == "" {
		return errors.New("idempotency source adapter is required")
	}
	if strings.TrimSpace(record.Dataset) == "" {
		return errors.New("idempotency dataset is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return errors.New("idempotency status is required")
	}
	return nil
}

func validateRawEventLedger(record storage.RawEventLedgerRecord) error {
	if strings.TrimSpace(record.EventID) == "" {
		return errors.New("raw event ledger event id is required")
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("raw event ledger tenant id is required")
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return errors.New("raw event ledger source id is required")
	}
	if strings.TrimSpace(record.SourceAdapter) == "" {
		return errors.New("raw event ledger source adapter is required")
	}
	if strings.TrimSpace(record.Dataset) == "" {
		return errors.New("raw event ledger dataset is required")
	}
	if strings.TrimSpace(record.IdempotencyKey) == "" {
		return errors.New("raw event ledger idempotency key is required")
	}
	if record.ObservationTime.IsZero() {
		return errors.New("raw event ledger observation time is required")
	}
	if record.ProcessingTime.IsZero() {
		return errors.New("raw event ledger processing time is required")
	}
	if len(record.PayloadJSON) == 0 {
		return errors.New("raw event ledger payload json is required")
	}
	return nil
}

func jsonOrEmpty(value []byte) []byte {
	if len(value) == 0 {
		return []byte(`{}`)
	}
	return value
}

func jsonArrayOrEmpty(value []byte) []byte {
	if len(value) == 0 {
		return []byte(`[]`)
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
