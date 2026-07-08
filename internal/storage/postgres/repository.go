package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
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
	return upsertIdempotencyRecord(ctx, r.db, record)
}

type statementExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func upsertIdempotencyRecord(ctx context.Context, executor statementExecutor, record storage.IdempotencyRecord) error {
	_, err := executor.ExecContext(ctx, `
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
	return upsertRawEventLedger(ctx, r.db, record)
}

func upsertRawEventLedger(ctx context.Context, executor statementExecutor, record storage.RawEventLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
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

// PersistPublishedRawEvent atomically records the raw ledger and idempotency
// state after the broker has acknowledged publication.
func (r *Repository) PersistPublishedRawEvent(ctx context.Context, ledger storage.RawEventLedgerRecord, idempotency storage.IdempotencyRecord) error {
	if err := validateRawEventLedger(ledger); err != nil {
		return err
	}
	if err := validateIdempotencyRecord(idempotency); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin published raw event transaction: %w", err)
	}
	defer tx.Rollback()
	if err := upsertRawEventLedger(ctx, tx, ledger); err != nil {
		return err
	}
	if err := upsertIdempotencyRecord(ctx, tx, idempotency); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit published raw event transaction: %w", err)
	}
	return nil
}

func (r *Repository) UpsertNormalizedEventLedger(ctx context.Context, record storage.NormalizedEventLedgerRecord) error {
	if err := validateNormalizedEventLedger(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO normalized_event_ledger (
 event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, schema_id, schema_version,
 observation_time, processing_time, confidence, raw_topic, raw_partition, raw_offset,
 normalized_topic, normalized_partition, normalized_offset, normalized_payload, entities, evidence, metadata, event
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
ON CONFLICT (event_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, source_adapter=EXCLUDED.source_adapter,
 dataset=EXCLUDED.dataset, idempotency_key=EXCLUDED.idempotency_key, schema_id=EXCLUDED.schema_id,
 schema_version=EXCLUDED.schema_version, observation_time=EXCLUDED.observation_time,
 processing_time=EXCLUDED.processing_time, confidence=EXCLUDED.confidence, raw_topic=EXCLUDED.raw_topic,
 raw_partition=EXCLUDED.raw_partition, raw_offset=EXCLUDED.raw_offset, normalized_topic=EXCLUDED.normalized_topic,
 normalized_partition=EXCLUDED.normalized_partition, normalized_offset=EXCLUDED.normalized_offset,
 normalized_payload=EXCLUDED.normalized_payload, entities=EXCLUDED.entities, evidence=EXCLUDED.evidence,
 metadata=EXCLUDED.metadata, event=EXCLUDED.event, updated_at=now()`,
		record.EventID, record.TenantID, record.SourceID, record.SourceAdapter, record.Dataset,
		record.IdempotencyKey, record.SchemaID, record.SchemaVersion, record.ObservationTime,
		record.ProcessingTime, record.Confidence, record.RawTopic, record.RawPartition, record.RawOffset,
		record.NormalizedTopic, record.NormalizedPartition, record.NormalizedOffset,
		jsonOrEmpty(record.NormalizedPayload), jsonArrayOrEmpty(record.EntitiesJSON),
		jsonArrayOrEmpty(record.EvidenceJSON), jsonOrEmpty(record.MetadataJSON), jsonOrEmpty(record.EventJSON))
	if err != nil {
		return fmt.Errorf("upsert normalized event ledger: %w", err)
	}
	return nil
}

func (r *Repository) UpsertCatalogSource(ctx context.Context, record storage.CatalogSourceRecord) error {
	if err := validateCatalogSource(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO catalog_sources (
  tenant_id, source_id, source_domain, source_adapter, display_name, description,
  status, ingestion_modes, datasets, metadata, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, now()
)
ON CONFLICT (tenant_id, source_id) DO UPDATE SET
  source_domain = EXCLUDED.source_domain,
  source_adapter = EXCLUDED.source_adapter,
  display_name = EXCLUDED.display_name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  ingestion_modes = EXCLUDED.ingestion_modes,
  datasets = EXCLUDED.datasets,
  metadata = EXCLUDED.metadata,
  updated_at = now()`,
		record.TenantID,
		record.SourceID,
		record.SourceDomain,
		record.SourceAdapter,
		record.DisplayName,
		record.Description,
		record.Status,
		pqArray(record.IngestionModes),
		pqArray(record.Datasets),
		jsonOrEmpty(record.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog source: %w", err)
	}
	return nil
}

func (r *Repository) UpsertCatalogPipeline(ctx context.Context, record storage.CatalogPipelineRecord) error {
	if err := validateCatalogPipeline(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO catalog_pipelines (
  tenant_id, pipeline_id, source_id, source_domain, pipeline_name, description,
  status, stages, input_datasets, output_topics, metadata, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, now()
)
ON CONFLICT (tenant_id, pipeline_id) DO UPDATE SET
  source_id = EXCLUDED.source_id,
  source_domain = EXCLUDED.source_domain,
  pipeline_name = EXCLUDED.pipeline_name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  stages = EXCLUDED.stages,
  input_datasets = EXCLUDED.input_datasets,
  output_topics = EXCLUDED.output_topics,
  metadata = EXCLUDED.metadata,
  updated_at = now()`,
		record.TenantID,
		record.PipelineID,
		record.SourceID,
		record.SourceDomain,
		record.PipelineName,
		record.Description,
		record.Status,
		pqArray(record.Stages),
		pqArray(record.InputDatasets),
		pqArray(record.OutputTopics),
		jsonOrEmpty(record.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog pipeline: %w", err)
	}
	return nil
}

func (r *Repository) UpsertCatalogRule(ctx context.Context, record storage.CatalogRuleRecord) error {
	if err := validateCatalogRule(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO catalog_rules (
  tenant_id, rule_id, rule_name, description, rule_type, severity, status, version,
  source_id, pipeline_id, dataset_scope, entity_scope, expression, actions, metadata, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, now()
)
ON CONFLICT (tenant_id, rule_id) DO UPDATE SET
  rule_name = EXCLUDED.rule_name,
  description = EXCLUDED.description,
  rule_type = EXCLUDED.rule_type,
  severity = EXCLUDED.severity,
  status = EXCLUDED.status,
  version = EXCLUDED.version,
  source_id = EXCLUDED.source_id,
  pipeline_id = EXCLUDED.pipeline_id,
  dataset_scope = EXCLUDED.dataset_scope,
  entity_scope = EXCLUDED.entity_scope,
  expression = EXCLUDED.expression,
  actions = EXCLUDED.actions,
  metadata = EXCLUDED.metadata,
  updated_at = now()`,
		record.TenantID,
		record.RuleID,
		record.RuleName,
		record.Description,
		record.RuleType,
		record.Severity,
		record.Status,
		record.Version,
		nullString(record.SourceID),
		nullString(record.PipelineID),
		pqArray(record.DatasetScope),
		pqArray(record.EntityScope),
		jsonOrEmpty(record.ExpressionJSON),
		pqArray(record.Actions),
		jsonOrEmpty(record.MetadataJSON),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog rule: %w", err)
	}
	return nil
}

func (r *Repository) ListSchedulerRuns(ctx context.Context, limit int) ([]storage.SchedulerRunRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT run_id, tenant_id, source_id, source_adapter, COALESCE(array_to_json(datasets), '[]'::json)::text,
  observation_date, dry_run, status, started_at, completed_at, events_built, events_published,
  provider_requests, provider_retries, failures, config, report, error_message, created_at, updated_at
FROM scheduler_runs
ORDER BY started_at DESC
LIMIT $1`, clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list scheduler runs: %w", err)
	}
	defer rows.Close()
	records := []storage.SchedulerRunRecord{}
	for rows.Next() {
		record, err := scanSchedulerRun(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list scheduler runs rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetSchedulerRun(ctx context.Context, runID string) (storage.SchedulerRunRecord, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT run_id, tenant_id, source_id, source_adapter, COALESCE(array_to_json(datasets), '[]'::json)::text,
  observation_date, dry_run, status, started_at, completed_at, events_built, events_published,
  provider_requests, provider_retries, failures, config, report, error_message, created_at, updated_at
FROM scheduler_runs
WHERE run_id = $1`, strings.TrimSpace(runID))
	record, err := scanSchedulerRun(row)
	if err != nil {
		return storage.SchedulerRunRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListProviderUsage(ctx context.Context, runID string, limit int) ([]storage.ProviderUsageRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT usage_id, run_id, provider, dataset, request_count, retry_count, event_count, budget, created_at
FROM provider_usage_runs
WHERE ($1 = '' OR run_id = $1)
ORDER BY created_at DESC
LIMIT $2`, strings.TrimSpace(runID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list provider usage: %w", err)
	}
	defer rows.Close()
	records := []storage.ProviderUsageRecord{}
	for rows.Next() {
		var record storage.ProviderUsageRecord
		if err := rows.Scan(&record.UsageID, &record.RunID, &record.Provider, &record.Dataset, &record.RequestCount, &record.RetryCount, &record.EventCount, &record.BudgetJSON, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan provider usage: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list provider usage rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListRawEventLedger(ctx context.Context, filter storage.RawEventLedgerFilter) ([]storage.RawEventLedgerRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, observation_time,
  processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE ($1 = '' OR tenant_id = $1)
  AND ($2 = '' OR source_id = $2)
  AND ($3 = '' OR dataset = $3)
ORDER BY created_at DESC
LIMIT $4`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list raw event ledger: %w", err)
	}
	defer rows.Close()
	records := []storage.RawEventLedgerRecord{}
	for rows.Next() {
		record, err := scanRawEventLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list raw event ledger rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetRawEventLedger(ctx context.Context, eventID string) (storage.RawEventLedgerRecord, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, observation_time,
  processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE event_id = $1`, strings.TrimSpace(eventID))
	record, err := scanRawEventLedger(row)
	if err != nil {
		return storage.RawEventLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListNormalizedEventLedger(ctx context.Context, filter storage.RawEventLedgerFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	rows, err := r.db.QueryContext(ctx, normalizedEventSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR source_id = $2) AND ($3 = '' OR dataset = $3)
ORDER BY created_at DESC LIMIT $4`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list normalized event ledger: %w", err)
	}
	defer rows.Close()
	records := []storage.NormalizedEventLedgerRecord{}
	for rows.Next() {
		record, err := scanNormalizedEventLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list normalized event ledger rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetNormalizedEventLedger(ctx context.Context, eventID string) (storage.NormalizedEventLedgerRecord, error) {
	record, err := scanNormalizedEventLedger(r.db.QueryRowContext(ctx, normalizedEventSelect+` WHERE event_id = $1`, strings.TrimSpace(eventID)))
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) GetIdempotencyRecord(ctx context.Context, tenantID string, sourceID string, idempotencyKey string) (storage.IdempotencyRecord, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT tenant_id, source_id, idempotency_key, event_id, source_adapter, dataset, topic, partition,
  offset_value, payload_hash, status, metadata, first_seen_at, last_seen_at
FROM idempotency_records
WHERE tenant_id = $1 AND source_id = $2 AND idempotency_key = $3`, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), strings.TrimSpace(idempotencyKey))
	var record storage.IdempotencyRecord
	var topic sql.NullString
	var partition sql.NullInt32
	var offset sql.NullInt64
	var payloadHash sql.NullString
	if err := row.Scan(&record.TenantID, &record.SourceID, &record.IdempotencyKey, &record.EventID, &record.SourceAdapter, &record.Dataset, &topic, &partition, &offset, &payloadHash, &record.Status, &record.MetadataJSON, &record.FirstSeenAt, &record.LastSeenAt); err != nil {
		return storage.IdempotencyRecord{}, mapScanError("get idempotency record", err)
	}
	record.Topic = topic.String
	record.PayloadHash = payloadHash.String
	if partition.Valid {
		record.Partition = &partition.Int32
	}
	if offset.Valid {
		record.Offset = &offset.Int64
	}
	return record, nil
}

func (r *Repository) ListCatalogSources(ctx context.Context, tenantID string, limit int) ([]storage.CatalogSourceRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, source_id, source_domain, source_adapter, display_name, description, status,
  COALESCE(array_to_json(ingestion_modes), '[]'::json)::text,
  COALESCE(array_to_json(datasets), '[]'::json)::text,
  metadata, created_at, updated_at
FROM catalog_sources
WHERE tenant_id = $1
ORDER BY source_id ASC
LIMIT $2`, strings.TrimSpace(tenantID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list catalog sources: %w", err)
	}
	defer rows.Close()
	records := []storage.CatalogSourceRecord{}
	for rows.Next() {
		record, err := scanCatalogSource(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list catalog sources rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListCatalogPipelines(ctx context.Context, tenantID string, limit int) ([]storage.CatalogPipelineRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, pipeline_id, source_id, source_domain, pipeline_name, description, status,
  COALESCE(array_to_json(stages), '[]'::json)::text,
  COALESCE(array_to_json(input_datasets), '[]'::json)::text,
  COALESCE(array_to_json(output_topics), '[]'::json)::text,
  metadata, created_at, updated_at
FROM catalog_pipelines
WHERE tenant_id = $1
ORDER BY pipeline_id ASC
LIMIT $2`, strings.TrimSpace(tenantID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list catalog pipelines: %w", err)
	}
	defer rows.Close()
	records := []storage.CatalogPipelineRecord{}
	for rows.Next() {
		record, err := scanCatalogPipeline(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list catalog pipelines rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListCatalogRules(ctx context.Context, tenantID string, limit int) ([]storage.CatalogRuleRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, rule_id, rule_name, description, rule_type, severity, status, version,
  source_id, pipeline_id,
  COALESCE(array_to_json(dataset_scope), '[]'::json)::text,
  COALESCE(array_to_json(entity_scope), '[]'::json)::text,
  expression,
  COALESCE(array_to_json(actions), '[]'::json)::text,
  metadata, created_at, updated_at
FROM catalog_rules
WHERE tenant_id = $1
ORDER BY rule_id ASC
LIMIT $2`, strings.TrimSpace(tenantID), clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list catalog rules: %w", err)
	}
	defer rows.Close()
	records := []storage.CatalogRuleRecord{}
	for rows.Next() {
		record, err := scanCatalogRule(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list catalog rules rows: %w", err)
	}
	return records, nil
}

type schedulerScanner interface {
	Scan(dest ...any) error
}

type rawLedgerScanner interface {
	Scan(dest ...any) error
}

type catalogSourceScanner interface {
	Scan(dest ...any) error
}

const normalizedEventSelect = `
SELECT event_id, tenant_id, source_id, source_adapter, dataset, idempotency_key, schema_id, schema_version,
 observation_time, processing_time, confidence, raw_topic, raw_partition, raw_offset,
 normalized_topic, normalized_partition, normalized_offset, normalized_payload, entities, evidence, metadata, event,
 created_at, updated_at FROM normalized_event_ledger`

type normalizedLedgerScanner interface{ Scan(dest ...any) error }

func scanNormalizedEventLedger(scanner normalizedLedgerScanner) (storage.NormalizedEventLedgerRecord, error) {
	var record storage.NormalizedEventLedgerRecord
	if err := scanner.Scan(&record.EventID, &record.TenantID, &record.SourceID, &record.SourceAdapter,
		&record.Dataset, &record.IdempotencyKey, &record.SchemaID, &record.SchemaVersion,
		&record.ObservationTime, &record.ProcessingTime, &record.Confidence, &record.RawTopic,
		&record.RawPartition, &record.RawOffset, &record.NormalizedTopic, &record.NormalizedPartition,
		&record.NormalizedOffset, &record.NormalizedPayload, &record.EntitiesJSON, &record.EvidenceJSON,
		&record.MetadataJSON, &record.EventJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.NormalizedEventLedgerRecord{}, mapScanError("scan normalized event ledger", err)
	}
	return record, nil
}

type catalogPipelineScanner interface {
	Scan(dest ...any) error
}

type catalogRuleScanner interface {
	Scan(dest ...any) error
}

func scanSchedulerRun(scanner schedulerScanner) (storage.SchedulerRunRecord, error) {
	var record storage.SchedulerRunRecord
	var datasetsJSON string
	var completedAt sql.NullTime
	var errorMessage sql.NullString
	if err := scanner.Scan(
		&record.RunID,
		&record.TenantID,
		&record.SourceID,
		&record.SourceAdapter,
		&datasetsJSON,
		&record.ObservationDate,
		&record.DryRun,
		&record.Status,
		&record.StartedAt,
		&completedAt,
		&record.EventsBuilt,
		&record.EventsPublished,
		&record.ProviderRequests,
		&record.ProviderRetries,
		&record.Failures,
		&record.ConfigJSON,
		&record.ReportJSON,
		&errorMessage,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return storage.SchedulerRunRecord{}, mapScanError("scan scheduler run", err)
	}
	if err := json.Unmarshal([]byte(datasetsJSON), &record.Datasets); err != nil {
		return storage.SchedulerRunRecord{}, fmt.Errorf("scan scheduler run datasets: %w", err)
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	record.ErrorMessage = errorMessage.String
	return record, nil
}

func scanRawEventLedger(scanner rawLedgerScanner) (storage.RawEventLedgerRecord, error) {
	var record storage.RawEventLedgerRecord
	var topic sql.NullString
	var partition sql.NullInt32
	var offset sql.NullInt64
	if err := scanner.Scan(
		&record.EventID,
		&record.TenantID,
		&record.SourceID,
		&record.SourceAdapter,
		&record.Dataset,
		&record.IdempotencyKey,
		&record.ObservationTime,
		&record.ProcessingTime,
		&topic,
		&partition,
		&offset,
		&record.PayloadJSON,
		&record.EntityHintsJSON,
		&record.CreatedAt,
	); err != nil {
		return storage.RawEventLedgerRecord{}, mapScanError("scan raw event ledger", err)
	}
	record.BrokerTopic = topic.String
	if partition.Valid {
		record.BrokerPartition = &partition.Int32
	}
	if offset.Valid {
		record.BrokerOffset = &offset.Int64
	}
	return record, nil
}

func scanCatalogSource(scanner catalogSourceScanner) (storage.CatalogSourceRecord, error) {
	var record storage.CatalogSourceRecord
	var ingestionModesJSON string
	var datasetsJSON string
	if err := scanner.Scan(
		&record.TenantID,
		&record.SourceID,
		&record.SourceDomain,
		&record.SourceAdapter,
		&record.DisplayName,
		&record.Description,
		&record.Status,
		&ingestionModesJSON,
		&datasetsJSON,
		&record.MetadataJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return storage.CatalogSourceRecord{}, mapScanError("scan catalog source", err)
	}
	if err := json.Unmarshal([]byte(ingestionModesJSON), &record.IngestionModes); err != nil {
		return storage.CatalogSourceRecord{}, fmt.Errorf("scan catalog source ingestion modes: %w", err)
	}
	if err := json.Unmarshal([]byte(datasetsJSON), &record.Datasets); err != nil {
		return storage.CatalogSourceRecord{}, fmt.Errorf("scan catalog source datasets: %w", err)
	}
	return record, nil
}

func scanCatalogPipeline(scanner catalogPipelineScanner) (storage.CatalogPipelineRecord, error) {
	var record storage.CatalogPipelineRecord
	var stagesJSON string
	var inputDatasetsJSON string
	var outputTopicsJSON string
	if err := scanner.Scan(
		&record.TenantID,
		&record.PipelineID,
		&record.SourceID,
		&record.SourceDomain,
		&record.PipelineName,
		&record.Description,
		&record.Status,
		&stagesJSON,
		&inputDatasetsJSON,
		&outputTopicsJSON,
		&record.MetadataJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return storage.CatalogPipelineRecord{}, mapScanError("scan catalog pipeline", err)
	}
	if err := json.Unmarshal([]byte(stagesJSON), &record.Stages); err != nil {
		return storage.CatalogPipelineRecord{}, fmt.Errorf("scan catalog pipeline stages: %w", err)
	}
	if err := json.Unmarshal([]byte(inputDatasetsJSON), &record.InputDatasets); err != nil {
		return storage.CatalogPipelineRecord{}, fmt.Errorf("scan catalog pipeline input datasets: %w", err)
	}
	if err := json.Unmarshal([]byte(outputTopicsJSON), &record.OutputTopics); err != nil {
		return storage.CatalogPipelineRecord{}, fmt.Errorf("scan catalog pipeline output topics: %w", err)
	}
	return record, nil
}

func scanCatalogRule(scanner catalogRuleScanner) (storage.CatalogRuleRecord, error) {
	var record storage.CatalogRuleRecord
	var sourceID sql.NullString
	var pipelineID sql.NullString
	var datasetScopeJSON string
	var entityScopeJSON string
	var actionsJSON string
	if err := scanner.Scan(
		&record.TenantID,
		&record.RuleID,
		&record.RuleName,
		&record.Description,
		&record.RuleType,
		&record.Severity,
		&record.Status,
		&record.Version,
		&sourceID,
		&pipelineID,
		&datasetScopeJSON,
		&entityScopeJSON,
		&record.ExpressionJSON,
		&actionsJSON,
		&record.MetadataJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return storage.CatalogRuleRecord{}, mapScanError("scan catalog rule", err)
	}
	record.SourceID = sourceID.String
	record.PipelineID = pipelineID.String
	if err := json.Unmarshal([]byte(datasetScopeJSON), &record.DatasetScope); err != nil {
		return storage.CatalogRuleRecord{}, fmt.Errorf("scan catalog rule dataset scope: %w", err)
	}
	if err := json.Unmarshal([]byte(entityScopeJSON), &record.EntityScope); err != nil {
		return storage.CatalogRuleRecord{}, fmt.Errorf("scan catalog rule entity scope: %w", err)
	}
	if err := json.Unmarshal([]byte(actionsJSON), &record.Actions); err != nil {
		return storage.CatalogRuleRecord{}, fmt.Errorf("scan catalog rule actions: %w", err)
	}
	return record, nil
}

func mapScanError(operation string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return storage.ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
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

func validateNormalizedEventLedger(record storage.NormalizedEventLedgerRecord) error {
	for name, value := range map[string]string{
		"event id": record.EventID, "tenant id": record.TenantID, "source id": record.SourceID,
		"source adapter": record.SourceAdapter, "dataset": record.Dataset, "idempotency key": record.IdempotencyKey,
		"schema id": record.SchemaID, "schema version": record.SchemaVersion,
		"raw topic": record.RawTopic, "normalized topic": record.NormalizedTopic,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("normalized event ledger %s is required", name)
		}
	}
	if record.ObservationTime.IsZero() {
		return errors.New("normalized event ledger observation time is required")
	}
	if record.ProcessingTime.IsZero() {
		return errors.New("normalized event ledger processing time is required")
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("normalized event ledger confidence must be between 0 and 1")
	}
	if len(record.NormalizedPayload) == 0 || len(record.EventJSON) == 0 {
		return errors.New("normalized event ledger payload and event json are required")
	}
	return nil
}

func validateCatalogSource(record storage.CatalogSourceRecord) error {
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("catalog source tenant id is required")
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return errors.New("catalog source id is required")
	}
	if strings.TrimSpace(record.SourceDomain) == "" {
		return errors.New("catalog source domain is required")
	}
	if strings.TrimSpace(record.SourceAdapter) == "" {
		return errors.New("catalog source adapter is required")
	}
	if strings.TrimSpace(record.DisplayName) == "" {
		return errors.New("catalog source display name is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return errors.New("catalog source status is required")
	}
	return nil
}

func validateCatalogPipeline(record storage.CatalogPipelineRecord) error {
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("catalog pipeline tenant id is required")
	}
	if strings.TrimSpace(record.PipelineID) == "" {
		return errors.New("catalog pipeline id is required")
	}
	if strings.TrimSpace(record.SourceID) == "" {
		return errors.New("catalog pipeline source id is required")
	}
	if strings.TrimSpace(record.SourceDomain) == "" {
		return errors.New("catalog pipeline source domain is required")
	}
	if strings.TrimSpace(record.PipelineName) == "" {
		return errors.New("catalog pipeline name is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return errors.New("catalog pipeline status is required")
	}
	return nil
}

func validateCatalogRule(record storage.CatalogRuleRecord) error {
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("catalog rule tenant id is required")
	}
	if strings.TrimSpace(record.RuleID) == "" {
		return errors.New("catalog rule id is required")
	}
	if strings.TrimSpace(record.RuleName) == "" {
		return errors.New("catalog rule name is required")
	}
	if strings.TrimSpace(record.RuleType) == "" {
		return errors.New("catalog rule type is required")
	}
	if strings.TrimSpace(record.Severity) == "" {
		return errors.New("catalog rule severity is required")
	}
	if strings.TrimSpace(record.Status) == "" {
		return errors.New("catalog rule status is required")
	}
	if record.Version <= 0 {
		return errors.New("catalog rule version must be positive")
	}
	if len(record.ExpressionJSON) == 0 {
		return errors.New("catalog rule expression json is required")
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
