package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lukebabs/signalops/internal/appmeta"
	"github.com/lukebabs/signalops/internal/storage"
)

const DriverName = "pgx"

type Repository struct {
	db          *sql.DB
	temporalDB  *sql.DB
	useTemporal bool
}

func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	return OpenWithTemporal(ctx, databaseURL, "")
}

func OpenWithTemporal(ctx context.Context, databaseURL string, temporalDatabaseURL string) (*Repository, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, errors.New("database url is required")
	}
	db, err := openDB(ctx, databaseURL, "postgres")
	if err != nil {
		return nil, err
	}
	repo := &Repository{db: db, temporalDB: db}
	temporalDatabaseURL = strings.TrimSpace(temporalDatabaseURL)
	if temporalDatabaseURL != "" && temporalDatabaseURL != databaseURL {
		temporalDB, err := openDB(ctx, temporalDatabaseURL, "temporal postgres")
		if err != nil {
			db.Close()
			return nil, err
		}
		repo.temporalDB = temporalDB
		repo.useTemporal = true
	}
	return repo, nil
}

func openDB(ctx context.Context, databaseURL string, label string) (*sql.DB, error) {
	db, err := sql.Open(DriverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", label, err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s database: %w", label, err)
	}
	return db, nil
}

func New(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("postgres db is required")
	}
	return &Repository{db: db, temporalDB: db}, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	if r.temporalDB != nil && r.temporalDB != r.db {
		if err := r.temporalDB.Close(); err != nil {
			return err
		}
	}
	return r.db.Close()
}

func (r *Repository) temporal() *sql.DB {
	if r != nil && r.temporalDB != nil {
		return r.temporalDB
	}
	return r.db
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

func (r *Repository) UpsertReplayJob(ctx context.Context, record storage.ReplayJobRecord) error {
	if err := validateReplayJob(record); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO replay_jobs (
  replay_job_id, tenant_id, source_id, dataset, source_kind, replay_mode, status, requested_by,
  window_start, window_end, started_at, completed_at, filters, options, result, error_message, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, $16, now()
)
ON CONFLICT (replay_job_id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  dataset = EXCLUDED.dataset,
  source_kind = EXCLUDED.source_kind,
  replay_mode = EXCLUDED.replay_mode,
  status = EXCLUDED.status,
  requested_by = EXCLUDED.requested_by,
  window_start = EXCLUDED.window_start,
  window_end = EXCLUDED.window_end,
  started_at = EXCLUDED.started_at,
  completed_at = EXCLUDED.completed_at,
  filters = EXCLUDED.filters,
  options = EXCLUDED.options,
  result = EXCLUDED.result,
  error_message = EXCLUDED.error_message,
  updated_at = now()`,
		record.ReplayJobID, record.TenantID, nullString(record.SourceID), nullString(record.Dataset),
		record.SourceKind, record.ReplayMode, record.Status, record.RequestedBy, record.WindowStart,
		record.WindowEnd, record.StartedAt, record.CompletedAt, jsonOrEmpty(record.FiltersJSON),
		jsonOrEmpty(record.OptionsJSON), jsonOrEmpty(record.ResultJSON), nullString(record.ErrorMessage),
	)
	if err != nil {
		return fmt.Errorf("upsert replay job: %w", err)
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
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  use_case = EXCLUDED.use_case,
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
	if r.useTemporal {
		return upsertRawEventLedgerTemporal(ctx, r.temporal(), record)
	}
	return upsertRawEventLedger(ctx, r.db, record)
}

func upsertRawEventLedger(ctx context.Context, executor statementExecutor, record storage.RawEventLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO raw_event_ledger (
  event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key,
  observation_time, processing_time, broker_topic, broker_partition, broker_offset,
  payload, entity_hints
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9,
  $10, $11, $12, $13, $14,
  $15, $16
)
ON CONFLICT (event_id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  use_case = EXCLUDED.use_case,
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
		recordAppID(record.AppID),
		recordDomain(record.Domain),
		recordUseCase(record.UseCase),
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

func upsertRawEventLedgerTemporal(ctx context.Context, executor statementExecutor, record storage.RawEventLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO raw_event_ledger (
  event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key,
  observation_time, processing_time, broker_topic, broker_partition, broker_offset,
  payload, entity_hints
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9,
  $10, $11, $12, $13, $14,
  $15, $16
)
ON CONFLICT (event_id, observation_time) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  source_id = EXCLUDED.source_id,
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  use_case = EXCLUDED.use_case,
  source_adapter = EXCLUDED.source_adapter,
  dataset = EXCLUDED.dataset,
  idempotency_key = EXCLUDED.idempotency_key,
  processing_time = EXCLUDED.processing_time,
  broker_topic = EXCLUDED.broker_topic,
  broker_partition = EXCLUDED.broker_partition,
  broker_offset = EXCLUDED.broker_offset,
  payload = EXCLUDED.payload,
  entity_hints = EXCLUDED.entity_hints`,
		record.EventID,
		record.TenantID,
		record.SourceID,
		recordAppID(record.AppID),
		recordDomain(record.Domain),
		recordUseCase(record.UseCase),
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
		return fmt.Errorf("upsert temporal raw event ledger: %w", err)
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
	if r.useTemporal {
		if err := upsertRawEventLedgerTemporal(ctx, r.temporal(), ledger); err != nil {
			return err
		}
		return r.UpsertIdempotencyRecord(ctx, idempotency)
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
	conflict := "event_id"
	target := r.db
	if r.useTemporal {
		conflict = "event_id, observation_time"
		target = r.temporal()
	}
	_, err := target.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO normalized_event_ledger (
 event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key, schema_id, schema_version,
 observation_time, processing_time, confidence, raw_topic, raw_partition, raw_offset,
 normalized_topic, normalized_partition, normalized_offset, normalized_payload, entities, evidence, metadata, event
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
ON CONFLICT (%s) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_adapter=EXCLUDED.source_adapter,
 dataset=EXCLUDED.dataset, idempotency_key=EXCLUDED.idempotency_key, schema_id=EXCLUDED.schema_id,
 schema_version=EXCLUDED.schema_version, processing_time=EXCLUDED.processing_time,
 confidence=EXCLUDED.confidence, raw_topic=EXCLUDED.raw_topic,
 raw_partition=EXCLUDED.raw_partition, raw_offset=EXCLUDED.raw_offset, normalized_topic=EXCLUDED.normalized_topic,
 normalized_partition=EXCLUDED.normalized_partition, normalized_offset=EXCLUDED.normalized_offset,
 normalized_payload=EXCLUDED.normalized_payload, entities=EXCLUDED.entities, evidence=EXCLUDED.evidence,
 metadata=EXCLUDED.metadata, event=EXCLUDED.event, updated_at=now()`, conflict),
		record.EventID, record.TenantID, record.SourceID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), record.SourceAdapter, record.Dataset,
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

func (r *Repository) UpsertSignalLedger(ctx context.Context, record storage.SignalLedgerRecord) error {
	if err := validateSignalLedger(record); err != nil {
		return err
	}
	if r.useTemporal {
		return upsertSignalLedgerTemporal(ctx, r.temporal(), record)
	}
	return upsertSignalLedger(ctx, r.db, record)
}

func upsertSignalLedger(ctx context.Context, executor statementExecutor, record storage.SignalLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO signal_ledger (
 signal_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, ingestion_mode, dataset,
 event_ids, artifact_ids, signal_type, detector_id, detector_version, model_version, signal_time,
 observation_time, effective_time, processing_time, window_start, window_end, confidence, severity,
 entities, supporting_metrics, graph_targets, semantic_evidence, evidence, recommendation,
 correlation_id, trace_id, causation_id, replay_job_id, broker_topic, broker_partition, broker_offset, event
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38)
ON CONFLICT (signal_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_domain=EXCLUDED.source_domain,
 source_adapter=EXCLUDED.source_adapter, ingestion_mode=EXCLUDED.ingestion_mode, dataset=EXCLUDED.dataset,
 event_ids=EXCLUDED.event_ids, artifact_ids=EXCLUDED.artifact_ids, signal_type=EXCLUDED.signal_type,
 detector_id=EXCLUDED.detector_id, detector_version=EXCLUDED.detector_version, model_version=EXCLUDED.model_version,
 signal_time=EXCLUDED.signal_time, observation_time=EXCLUDED.observation_time, effective_time=EXCLUDED.effective_time,
 processing_time=EXCLUDED.processing_time, window_start=EXCLUDED.window_start, window_end=EXCLUDED.window_end,
 confidence=EXCLUDED.confidence, severity=EXCLUDED.severity, entities=EXCLUDED.entities,
 supporting_metrics=EXCLUDED.supporting_metrics, graph_targets=EXCLUDED.graph_targets,
 semantic_evidence=EXCLUDED.semantic_evidence, evidence=EXCLUDED.evidence, recommendation=EXCLUDED.recommendation,
 correlation_id=EXCLUDED.correlation_id, trace_id=EXCLUDED.trace_id, causation_id=EXCLUDED.causation_id,
 replay_job_id=EXCLUDED.replay_job_id, broker_topic=EXCLUDED.broker_topic,
 broker_partition=EXCLUDED.broker_partition, broker_offset=EXCLUDED.broker_offset,
 event=EXCLUDED.event, updated_at=now()`,
		record.SignalID, record.TenantID, record.SourceID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), record.SourceDomain, record.SourceAdapter,
		record.IngestionMode, record.Dataset, record.EventIDs, record.ArtifactIDs, record.SignalType,
		record.DetectorID, record.DetectorVersion, record.ModelVersion, record.SignalTime,
		record.ObservationTime, record.EffectiveTime, record.ProcessingTime, record.WindowStart, record.WindowEnd,
		record.Confidence, record.Severity, jsonArrayOrEmpty(record.EntitiesJSON), jsonOrEmpty(record.SupportingMetrics),
		jsonArrayOrEmpty(record.GraphTargetsJSON), jsonArrayOrEmpty(record.SemanticEvidenceJSON),
		jsonArrayOrEmpty(record.EvidenceJSON), jsonOrEmpty(record.RecommendationJSON), record.CorrelationID,
		nullString(record.TraceID), nullString(record.CausationID), nullString(record.ReplayJobID),
		record.BrokerTopic, record.BrokerPartition, record.BrokerOffset, jsonOrEmpty(record.EventJSON))
	if err != nil {
		return fmt.Errorf("upsert signal ledger: %w", err)
	}
	return nil
}

func upsertSignalLedgerTemporal(ctx context.Context, executor statementExecutor, record storage.SignalLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO signal_ledger (
 signal_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, ingestion_mode, dataset,
 event_ids, artifact_ids, signal_type, detector_id, detector_version, model_version, signal_time,
 observation_time, effective_time, processing_time, window_start, window_end, confidence, severity,
 entities, supporting_metrics, graph_targets, semantic_evidence, evidence, recommendation,
 correlation_id, trace_id, causation_id, replay_job_id, broker_topic, broker_partition, broker_offset, event
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38)
ON CONFLICT (signal_id, signal_time) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_domain=EXCLUDED.source_domain,
 source_adapter=EXCLUDED.source_adapter, ingestion_mode=EXCLUDED.ingestion_mode, dataset=EXCLUDED.dataset,
 event_ids=EXCLUDED.event_ids, artifact_ids=EXCLUDED.artifact_ids, signal_type=EXCLUDED.signal_type,
 detector_id=EXCLUDED.detector_id, detector_version=EXCLUDED.detector_version, model_version=EXCLUDED.model_version,
 observation_time=EXCLUDED.observation_time, effective_time=EXCLUDED.effective_time,
 processing_time=EXCLUDED.processing_time, window_start=EXCLUDED.window_start, window_end=EXCLUDED.window_end,
 confidence=EXCLUDED.confidence, severity=EXCLUDED.severity, entities=EXCLUDED.entities,
 supporting_metrics=EXCLUDED.supporting_metrics, graph_targets=EXCLUDED.graph_targets,
 semantic_evidence=EXCLUDED.semantic_evidence, evidence=EXCLUDED.evidence, recommendation=EXCLUDED.recommendation,
 correlation_id=EXCLUDED.correlation_id, trace_id=EXCLUDED.trace_id, causation_id=EXCLUDED.causation_id,
 replay_job_id=EXCLUDED.replay_job_id, broker_topic=EXCLUDED.broker_topic,
 broker_partition=EXCLUDED.broker_partition, broker_offset=EXCLUDED.broker_offset,
 event=EXCLUDED.event, updated_at=now()`,
		record.SignalID, record.TenantID, record.SourceID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), record.SourceDomain, record.SourceAdapter,
		record.IngestionMode, record.Dataset, record.EventIDs, record.ArtifactIDs, record.SignalType,
		record.DetectorID, record.DetectorVersion, record.ModelVersion, record.SignalTime,
		record.ObservationTime, record.EffectiveTime, record.ProcessingTime, record.WindowStart, record.WindowEnd,
		record.Confidence, record.Severity, jsonArrayOrEmpty(record.EntitiesJSON), jsonOrEmpty(record.SupportingMetrics),
		jsonArrayOrEmpty(record.GraphTargetsJSON), jsonArrayOrEmpty(record.SemanticEvidenceJSON),
		jsonArrayOrEmpty(record.EvidenceJSON), jsonOrEmpty(record.RecommendationJSON), record.CorrelationID,
		nullString(record.TraceID), nullString(record.CausationID), nullString(record.ReplayJobID),
		record.BrokerTopic, record.BrokerPartition, record.BrokerOffset, jsonOrEmpty(record.EventJSON))
	if err != nil {
		return fmt.Errorf("upsert temporal signal ledger: %w", err)
	}
	return nil
}

func (r *Repository) UpsertAlertLedger(ctx context.Context, record storage.AlertLedgerRecord) error {
	if err := validateAlertLedger(record); err != nil {
		return err
	}
	return upsertAlertLedger(ctx, r.db, record)
}

func upsertAlertLedger(ctx context.Context, executor statementExecutor, record storage.AlertLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO alert_ledger (
 alert_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, dataset, signal_id, detector_id,
 alert_type, severity, status, title, summary, confidence, event_ids, entities, evidence, recommendation,
 correlation_id, first_observed_at, last_observed_at, acknowledged_at, acknowledged_by, resolved_at, resolved_by, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
ON CONFLICT (alert_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_domain=EXCLUDED.source_domain,
 source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset, signal_id=EXCLUDED.signal_id,
 detector_id=EXCLUDED.detector_id, alert_type=EXCLUDED.alert_type, severity=EXCLUDED.severity,
 title=EXCLUDED.title, summary=EXCLUDED.summary, confidence=EXCLUDED.confidence, event_ids=EXCLUDED.event_ids,
 entities=EXCLUDED.entities, evidence=EXCLUDED.evidence, recommendation=EXCLUDED.recommendation,
 correlation_id=EXCLUDED.correlation_id, first_observed_at=LEAST(alert_ledger.first_observed_at, EXCLUDED.first_observed_at),
 last_observed_at=GREATEST(alert_ledger.last_observed_at, EXCLUDED.last_observed_at), metadata=EXCLUDED.metadata,
 updated_at=now()`, record.AlertID, record.TenantID, record.SourceID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase), record.SourceDomain, record.SourceAdapter,
		record.Dataset, record.SignalID, record.DetectorID, record.AlertType, record.Severity, record.Status,
		record.Title, record.Summary, record.Confidence, record.EventIDs, jsonArrayOrEmpty(record.EntitiesJSON),
		jsonArrayOrEmpty(record.EvidenceJSON), nullableJSON(record.RecommendationJSON), record.CorrelationID,
		record.FirstObservedAt, record.LastObservedAt, record.AcknowledgedAt, nullString(record.AcknowledgedBy),
		record.ResolvedAt, nullString(record.ResolvedBy), jsonOrEmpty(record.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert alert ledger: %w", err)
	}
	return nil
}

func (r *Repository) UpsertInsightLedger(ctx context.Context, record storage.InsightLedgerRecord) error {
	if err := validateInsightLedger(record); err != nil {
		return err
	}
	return upsertInsightLedger(ctx, r.db, record)
}

func upsertInsightLedger(ctx context.Context, executor statementExecutor, record storage.InsightLedgerRecord) error {
	_, err := executor.ExecContext(ctx, `
INSERT INTO insight_ledger (
 insight_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, dataset, signal_id, detector_id,
 insight_type, status, title, summary, confidence, severity, event_ids, entities, supporting_metrics,
 semantic_evidence, recommendation, correlation_id, observed_at, reviewed_at, reviewed_by, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
ON CONFLICT (insight_id) DO UPDATE SET
 tenant_id=EXCLUDED.tenant_id, source_id=EXCLUDED.source_id, app_id=EXCLUDED.app_id,
 domain=EXCLUDED.domain, use_case=EXCLUDED.use_case, source_domain=EXCLUDED.source_domain,
 source_adapter=EXCLUDED.source_adapter, dataset=EXCLUDED.dataset, signal_id=EXCLUDED.signal_id,
 detector_id=EXCLUDED.detector_id, insight_type=EXCLUDED.insight_type, title=EXCLUDED.title,
 summary=EXCLUDED.summary, confidence=EXCLUDED.confidence, severity=EXCLUDED.severity,
 event_ids=EXCLUDED.event_ids, entities=EXCLUDED.entities, supporting_metrics=EXCLUDED.supporting_metrics,
 semantic_evidence=EXCLUDED.semantic_evidence, recommendation=EXCLUDED.recommendation,
 correlation_id=EXCLUDED.correlation_id, observed_at=GREATEST(insight_ledger.observed_at, EXCLUDED.observed_at),
 metadata=EXCLUDED.metadata, updated_at=now()`, record.InsightID, record.TenantID, record.SourceID, recordAppID(record.AppID), recordDomain(record.Domain), recordUseCase(record.UseCase),
		record.SourceDomain, record.SourceAdapter, record.Dataset, record.SignalID, record.DetectorID,
		record.InsightType, record.Status, record.Title, record.Summary, record.Confidence, record.Severity,
		record.EventIDs, jsonArrayOrEmpty(record.EntitiesJSON), jsonOrEmpty(record.SupportingMetrics),
		jsonArrayOrEmpty(record.SemanticEvidenceJSON), nullableJSON(record.RecommendationJSON), record.CorrelationID,
		record.ObservedAt, record.ReviewedAt, nullString(record.ReviewedBy), jsonOrEmpty(record.MetadataJSON))
	if err != nil {
		return fmt.Errorf("upsert insight ledger: %w", err)
	}
	return nil
}

func (r *Repository) PersistSignalLifecycle(ctx context.Context, signal storage.SignalLedgerRecord, alerts []storage.AlertLedgerRecord, insights []storage.InsightLedgerRecord) error {
	if err := validateSignalLedger(signal); err != nil {
		return err
	}
	for _, alert := range alerts {
		if err := validateAlertLedger(alert); err != nil {
			return err
		}
	}
	for _, insight := range insights {
		if err := validateInsightLedger(insight); err != nil {
			return err
		}
	}
	artifacts, err := extractMarketOpsDSMArtifacts(signal)
	if err != nil {
		return err
	}
	graphProposals := []storage.MarketOpsDSMGraphProposalRecord{}
	for _, artifact := range artifacts {
		if err := validateMarketOpsDSMArtifact(artifact); err != nil {
			return err
		}
		proposals, err := extractMarketOpsDSMGraphProposals(artifact)
		if err != nil {
			return err
		}
		graphProposals = append(graphProposals, proposals...)
	}
	for _, proposal := range graphProposals {
		if err := validateMarketOpsDSMGraphProposal(proposal); err != nil {
			return err
		}
	}
	if r.useTemporal {
		if err := upsertSignalLedgerTemporal(ctx, r.temporal(), signal); err != nil {
			return err
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin signal lifecycle transaction: %w", err)
	}
	defer tx.Rollback()
	if err := upsertSignalLedger(ctx, tx, signal); err != nil {
		return err
	}
	for _, alert := range alerts {
		if err := upsertAlertLedger(ctx, tx, alert); err != nil {
			return err
		}
	}
	for _, insight := range insights {
		if err := upsertInsightLedger(ctx, tx, insight); err != nil {
			return err
		}
	}
	for _, artifact := range artifacts {
		if err := upsertMarketOpsDSMArtifact(ctx, tx, artifact); err != nil {
			return err
		}
	}
	for _, proposal := range graphProposals {
		if err := upsertMarketOpsDSMGraphProposal(ctx, tx, proposal); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit signal lifecycle transaction: %w", err)
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

func (r *Repository) ListReplayJobs(ctx context.Context, filter storage.ReplayJobFilter) ([]storage.ReplayJobRecord, error) {
	rows, err := r.db.QueryContext(ctx, replayJobSelect+`
WHERE ($1 = '' OR tenant_id = $1)
  AND ($2 = '' OR source_id = $2)
  AND ($3 = '' OR dataset = $3)
  AND ($4 = '' OR source_kind = $4)
  AND ($5 = '' OR status = $5)
ORDER BY created_at DESC
LIMIT $6`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.SourceID),
		strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.SourceKind), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list replay jobs: %w", err)
	}
	defer rows.Close()
	records := []storage.ReplayJobRecord{}
	for rows.Next() {
		record, err := scanReplayJob(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list replay jobs rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetReplayJob(ctx context.Context, replayJobID string) (storage.ReplayJobRecord, error) {
	record, err := scanReplayJob(r.db.QueryRowContext(ctx, replayJobSelect+` WHERE replay_job_id = $1`, strings.TrimSpace(replayJobID)))
	if err != nil {
		return storage.ReplayJobRecord{}, err
	}
	return record, nil
}

func (r *Repository) CountReplayJobsByStatus(ctx context.Context, tenantID string) ([]storage.ReplayJobStatusCount, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT status, count(*)
FROM replay_jobs
WHERE ($1 = '' OR tenant_id = $1)
GROUP BY status
ORDER BY status ASC`, strings.TrimSpace(tenantID))
	if err != nil {
		return nil, fmt.Errorf("count replay jobs by status: %w", err)
	}
	defer rows.Close()
	counts := []storage.ReplayJobStatusCount{}
	for rows.Next() {
		var record storage.ReplayJobStatusCount
		if err := rows.Scan(&record.Status, &record.Count); err != nil {
			return nil, fmt.Errorf("scan replay job status count: %w", err)
		}
		counts = append(counts, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count replay jobs by status rows: %w", err)
	}
	return counts, nil
}

func (r *Repository) UpsertReplayWorkerHeartbeat(ctx context.Context, record storage.ReplayWorkerHeartbeatRecord) error {
	workerID := strings.TrimSpace(record.WorkerID)
	if workerID == "" {
		return errors.New("replay worker id is required")
	}
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = "idle"
	}
	processStartedAt := record.ProcessStartedAt.UTC()
	if processStartedAt.IsZero() {
		processStartedAt = time.Now().UTC()
	}
	lastSeenAt := record.LastSeenAt.UTC()
	if lastSeenAt.IsZero() {
		lastSeenAt = time.Now().UTC()
	}
	metadata := record.MetadataJSON
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO replay_worker_heartbeats (
  worker_id, status, process_started_at, last_seen_at, last_claimed_at, last_claimed_replay_job_id,
  last_completed_at, last_completed_replay_job_id, last_error_at, last_error_message, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (worker_id) DO UPDATE SET
  status = EXCLUDED.status,
  process_started_at = EXCLUDED.process_started_at,
  last_seen_at = EXCLUDED.last_seen_at,
  last_claimed_at = COALESCE(EXCLUDED.last_claimed_at, replay_worker_heartbeats.last_claimed_at),
  last_claimed_replay_job_id = COALESCE(NULLIF(EXCLUDED.last_claimed_replay_job_id, ''), replay_worker_heartbeats.last_claimed_replay_job_id),
  last_completed_at = COALESCE(EXCLUDED.last_completed_at, replay_worker_heartbeats.last_completed_at),
  last_completed_replay_job_id = COALESCE(NULLIF(EXCLUDED.last_completed_replay_job_id, ''), replay_worker_heartbeats.last_completed_replay_job_id),
  last_error_at = COALESCE(EXCLUDED.last_error_at, replay_worker_heartbeats.last_error_at),
  last_error_message = COALESCE(NULLIF(EXCLUDED.last_error_message, ''), replay_worker_heartbeats.last_error_message),
  metadata = EXCLUDED.metadata,
  updated_at = now()`,
		workerID, status, processStartedAt, lastSeenAt, record.LastClaimedAt, strings.TrimSpace(record.LastClaimedReplayJobID),
		record.LastCompletedAt, strings.TrimSpace(record.LastCompletedReplayJobID), record.LastErrorAt, nullString(record.LastErrorMessage), jsonOrEmpty(metadata))
	if err != nil {
		return fmt.Errorf("upsert replay worker heartbeat: %w", err)
	}
	return nil
}

func (r *Repository) ListReplayWorkerHeartbeats(ctx context.Context, limit int) ([]storage.ReplayWorkerHeartbeatRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT worker_id, status, process_started_at, last_seen_at, last_claimed_at, last_claimed_replay_job_id,
  last_completed_at, last_completed_replay_job_id, last_error_at, last_error_message, metadata, created_at, updated_at
FROM replay_worker_heartbeats
ORDER BY last_seen_at DESC
LIMIT $1`, clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list replay worker heartbeats: %w", err)
	}
	defer rows.Close()
	records := []storage.ReplayWorkerHeartbeatRecord{}
	for rows.Next() {
		record, err := scanReplayWorkerHeartbeat(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list replay worker heartbeats rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ClaimNextReplayJob(ctx context.Context, workerID string, claimedAt time.Time) (storage.ReplayJobRecord, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		workerID = "signalops-replay-worker"
	}
	metadata, err := json.Marshal(map[string]any{"worker_id": workerID, "claimed_at": claimedAt.UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return storage.ReplayJobRecord{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("begin claim replay job: %w", err)
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `
WITH next_job AS (
  SELECT replay_job_id FROM replay_jobs
  WHERE status = 'queued'
  ORDER BY created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE replay_jobs
SET status = 'running', started_at = $1, result = COALESCE(result, '{}'::jsonb) || $2::jsonb, updated_at = now()
WHERE replay_job_id = (SELECT replay_job_id FROM next_job)
RETURNING replay_job_id, tenant_id, source_id, dataset, source_kind, replay_mode, status, requested_by,
 window_start, window_end, started_at, completed_at, filters, options, result, error_message, created_at, updated_at`, claimedAt.UTC(), jsonOrEmpty(metadata))
	record, err := scanReplayJob(row)
	if err != nil {
		return storage.ReplayJobRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("commit claim replay job: %w", err)
	}
	return record, nil
}

func (r *Repository) CompleteReplayJob(ctx context.Context, replayJobID string, completedAt time.Time, resultJSON []byte) (storage.ReplayJobRecord, error) {
	return r.updateReplayJobTerminal(ctx, replayJobID, storage.ReplayJobStatusSucceeded, completedAt, "", resultJSON)
}

func (r *Repository) FailReplayJob(ctx context.Context, replayJobID string, failedAt time.Time, errorMessage string, resultJSON []byte) (storage.ReplayJobRecord, error) {
	return r.updateReplayJobTerminal(ctx, replayJobID, storage.ReplayJobStatusFailed, failedAt, errorMessage, resultJSON)
}

func (r *Repository) CancelReplayJob(ctx context.Context, replayJobID string, actor string, canceledAt time.Time, reason string, resultJSON []byte) (storage.ReplayJobRecord, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "operator-local"
	}
	resultEnvelope := map[string]any{}
	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &resultEnvelope); err != nil {
			return storage.ReplayJobRecord{}, fmt.Errorf("decode cancel replay result: %w", err)
		}
	}
	resultEnvelope["canceled"] = map[string]any{
		"actor":       actor,
		"reason":      strings.TrimSpace(reason),
		"canceled_at": canceledAt.UTC().Format(time.RFC3339Nano),
	}
	metadata, err := json.Marshal(resultEnvelope)
	if err != nil {
		return storage.ReplayJobRecord{}, err
	}
	execResult, err := r.db.ExecContext(ctx, `
UPDATE replay_jobs
SET status = 'canceled', completed_at = $2, result = COALESCE(result, '{}'::jsonb) || $3::jsonb, error_message = $4, updated_at = now()
WHERE replay_job_id = $1 AND status IN ('queued', 'running', 'canceled')`, strings.TrimSpace(replayJobID), canceledAt.UTC(), jsonOrEmpty(metadata), nullString("canceled by "+actor))
	if err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("cancel replay job: %w", err)
	}
	changed, err := execResult.RowsAffected()
	if err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("cancel replay job rows affected: %w", err)
	}
	if changed == 0 {
		return r.GetReplayJob(ctx, replayJobID)
	}
	return r.GetReplayJob(ctx, replayJobID)
}

func (r *Repository) updateReplayJobTerminal(ctx context.Context, replayJobID string, status string, completedAt time.Time, errorMessage string, resultJSON []byte) (storage.ReplayJobRecord, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE replay_jobs SET status = $2, completed_at = $3, result = $4, error_message = $5, updated_at = now() WHERE replay_job_id = $1`, strings.TrimSpace(replayJobID), status, completedAt.UTC(), jsonOrEmpty(resultJSON), nullString(errorMessage))
	if err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("update replay job terminal: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return storage.ReplayJobRecord{}, fmt.Errorf("update replay job rows affected: %w", err)
	}
	if changed == 0 {
		return storage.ReplayJobRecord{}, storage.ErrNotFound
	}
	return r.GetReplayJob(ctx, replayJobID)
}

func (r *Repository) ListReplayRawEvents(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.RawEventLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, `
SELECT event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key, observation_time,
  processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE tenant_id = $1
  AND ($2 = '' OR source_id = $2)
  AND ($3 = '' OR dataset = $3)
  AND observation_time >= $4 AND observation_time < $5
ORDER BY observation_time ASC, event_id ASC
LIMIT $6 OFFSET $7`, strings.TrimSpace(job.TenantID), strings.TrimSpace(job.SourceID), strings.TrimSpace(job.Dataset), job.WindowStart, job.WindowEnd, clampLimit(limit), nonNegativeOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("list replay raw events: %w", err)
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
		return nil, fmt.Errorf("list replay raw events rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListReplayNormalizedEvents(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.NormalizedEventLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, normalizedEventSelect+`
WHERE tenant_id = $1 AND ($2 = '' OR source_id = $2) AND ($3 = '' OR dataset = $3)
  AND observation_time >= $4 AND observation_time < $5
ORDER BY observation_time ASC, event_id ASC LIMIT $6 OFFSET $7`, strings.TrimSpace(job.TenantID), strings.TrimSpace(job.SourceID), strings.TrimSpace(job.Dataset), job.WindowStart, job.WindowEnd, clampLimit(limit), nonNegativeOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("list replay normalized events: %w", err)
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
		return nil, fmt.Errorf("list replay normalized events rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ListReplaySignals(ctx context.Context, job storage.ReplayJobRecord, limit int, offset int) ([]storage.SignalLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, signalSelect+`
WHERE tenant_id = $1 AND ($2 = '' OR source_id = $2) AND ($3 = '' OR dataset = $3)
  AND signal_time >= $4 AND signal_time < $5
ORDER BY signal_time ASC, signal_id ASC LIMIT $6 OFFSET $7`, strings.TrimSpace(job.TenantID), strings.TrimSpace(job.SourceID), strings.TrimSpace(job.Dataset), job.WindowStart, job.WindowEnd, clampLimit(limit), nonNegativeOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("list replay signals: %w", err)
	}
	defer rows.Close()
	records := []storage.SignalLedgerRecord{}
	for rows.Next() {
		record, err := scanSignalLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list replay signals rows: %w", err)
	}
	return records, nil
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
	rows, err := r.temporal().QueryContext(ctx, `
SELECT event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key, observation_time,
  processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE ($1 = '' OR tenant_id = $1)
  AND ($2 = '' OR app_id = $2)
  AND ($3 = '' OR domain = $3)
  AND ($4 = '' OR use_case = $4)
  AND ($5 = '' OR source_id = $5)
  AND ($6 = '' OR dataset = $6)
ORDER BY created_at DESC
LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), clampLimit(filter.Limit))
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
	query := `
SELECT event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key, observation_time,
  processing_time, broker_topic, broker_partition, broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE event_id = $1`
	if r.useTemporal {
		query += ` ORDER BY observation_time DESC LIMIT 1`
	}
	row := r.temporal().QueryRowContext(ctx, query, strings.TrimSpace(eventID))
	record, err := scanRawEventLedger(row)
	if err != nil {
		return storage.RawEventLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListNormalizedEventLedger(ctx context.Context, filter storage.RawEventLedgerFilter) ([]storage.NormalizedEventLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, normalizedEventSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
AND ($5 = '' OR source_id = $5) AND ($6 = '' OR dataset = $6)
ORDER BY created_at DESC LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID), strings.TrimSpace(filter.Dataset), clampLimit(filter.Limit))
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
	record, err := scanNormalizedEventLedger(r.temporal().QueryRowContext(ctx, normalizedEventSelect+normalizedEventByIDWhere(r.useTemporal), strings.TrimSpace(eventID)))
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListSignalLedger(ctx context.Context, filter storage.SignalLedgerFilter) ([]storage.SignalLedgerRecord, error) {
	rows, err := r.temporal().QueryContext(ctx, signalSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
 AND ($5 = '' OR source_id = $5) AND ($6 = '' OR dataset = $6)
 AND ($7 = '' OR detector_id = $7) AND ($8 = '' OR severity = $8)
ORDER BY signal_time DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID),
		strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.DetectorID), strings.TrimSpace(filter.Severity), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list signal ledger: %w", err)
	}
	defer rows.Close()
	records := []storage.SignalLedgerRecord{}
	for rows.Next() {
		record, err := scanSignalLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list signal ledger rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetSignalLedger(ctx context.Context, signalID string) (storage.SignalLedgerRecord, error) {
	record, err := scanSignalLedger(r.temporal().QueryRowContext(ctx, signalSelect+signalByIDWhere(r.useTemporal), strings.TrimSpace(signalID)))
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) ListAlertLedger(ctx context.Context, filter storage.AlertLedgerFilter) ([]storage.AlertLedgerRecord, error) {
	rows, err := r.db.QueryContext(ctx, alertSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
 AND ($5 = '' OR source_id = $5) AND ($6 = '' OR dataset = $6)
 AND ($7 = '' OR severity = $7) AND ($8 = '' OR status = $8)
ORDER BY last_observed_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID),
		strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.Severity), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list alert ledger: %w", err)
	}
	defer rows.Close()
	records := []storage.AlertLedgerRecord{}
	for rows.Next() {
		record, err := scanAlertLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list alert ledger rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetAlertLedger(ctx context.Context, alertID string) (storage.AlertLedgerRecord, error) {
	record, err := scanAlertLedger(r.db.QueryRowContext(ctx, alertSelect+` WHERE alert_id = $1`, strings.TrimSpace(alertID)))
	if err != nil {
		return storage.AlertLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) MutateAlertLifecycle(ctx context.Context, mutation storage.AlertLifecycleMutation) (storage.AlertLedgerRecord, error) {
	if err := validateAlertLifecycleMutation(mutation); err != nil {
		return storage.AlertLedgerRecord{}, err
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE alert_ledger
SET status = $2,
    acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN $4 ELSE acknowledged_at END,
    acknowledged_by = CASE WHEN $2 = 'acknowledged' THEN $3 ELSE acknowledged_by END,
    resolved_at = CASE WHEN $2 = 'resolved' THEN $4 ELSE resolved_at END,
    resolved_by = CASE WHEN $2 = 'resolved' THEN $3 ELSE resolved_by END,
    metadata = COALESCE(metadata, '{}'::jsonb) || $5::jsonb,
    updated_at = now()
WHERE alert_id = $1`, strings.TrimSpace(mutation.AlertID), strings.TrimSpace(mutation.Status), strings.TrimSpace(mutation.Actor), mutation.MutatedAt.UTC(), jsonOrEmpty(mutation.MetadataJSON))
	if err != nil {
		return storage.AlertLedgerRecord{}, fmt.Errorf("mutate alert lifecycle: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return storage.AlertLedgerRecord{}, fmt.Errorf("mutate alert lifecycle rows affected: %w", err)
	}
	if changed == 0 {
		return storage.AlertLedgerRecord{}, storage.ErrNotFound
	}
	return r.GetAlertLedger(ctx, mutation.AlertID)
}

func (r *Repository) ListInsightLedger(ctx context.Context, filter storage.InsightLedgerFilter) ([]storage.InsightLedgerRecord, error) {
	rows, err := r.db.QueryContext(ctx, insightSelect+`
WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR app_id = $2) AND ($3 = '' OR domain = $3) AND ($4 = '' OR use_case = $4)
 AND ($5 = '' OR source_id = $5) AND ($6 = '' OR dataset = $6)
 AND ($7 = '' OR insight_type = $7) AND ($8 = '' OR status = $8)
ORDER BY observed_at DESC LIMIT $9`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.AppID), strings.TrimSpace(filter.Domain), strings.TrimSpace(filter.UseCase), strings.TrimSpace(filter.SourceID),
		strings.TrimSpace(filter.Dataset), strings.TrimSpace(filter.InsightType), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list insight ledger: %w", err)
	}
	defer rows.Close()
	records := []storage.InsightLedgerRecord{}
	for rows.Next() {
		record, err := scanInsightLedger(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list insight ledger rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetInsightLedger(ctx context.Context, insightID string) (storage.InsightLedgerRecord, error) {
	record, err := scanInsightLedger(r.db.QueryRowContext(ctx, insightSelect+` WHERE insight_id = $1`, strings.TrimSpace(insightID)))
	if err != nil {
		return storage.InsightLedgerRecord{}, err
	}
	return record, nil
}

func (r *Repository) MutateInsightLifecycle(ctx context.Context, mutation storage.InsightLifecycleMutation) (storage.InsightLedgerRecord, error) {
	if err := validateInsightLifecycleMutation(mutation); err != nil {
		return storage.InsightLedgerRecord{}, err
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE insight_ledger
SET status = $2,
    reviewed_at = CASE WHEN $2 IN ('reviewed', 'dismissed', 'archived') THEN $4 ELSE reviewed_at END,
    reviewed_by = CASE WHEN $2 IN ('reviewed', 'dismissed', 'archived') THEN $3 ELSE reviewed_by END,
    metadata = COALESCE(metadata, '{}'::jsonb) || $5::jsonb,
    updated_at = now()
WHERE insight_id = $1`, strings.TrimSpace(mutation.InsightID), strings.TrimSpace(mutation.Status), strings.TrimSpace(mutation.Actor), mutation.MutatedAt.UTC(), jsonOrEmpty(mutation.MetadataJSON))
	if err != nil {
		return storage.InsightLedgerRecord{}, fmt.Errorf("mutate insight lifecycle: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return storage.InsightLedgerRecord{}, fmt.Errorf("mutate insight lifecycle rows affected: %w", err)
	}
	if changed == 0 {
		return storage.InsightLedgerRecord{}, storage.ErrNotFound
	}
	return r.GetInsightLedger(ctx, mutation.InsightID)
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

func (r *Repository) ListMarketOpsAssets(ctx context.Context, tenantID string, universeGroup string, activeOnly bool, limit int) ([]storage.MarketOpsAssetRecord, error) {
	universeGroup = strings.TrimSpace(universeGroup)
	if universeGroup == "" {
		universeGroup = "top50_megacap"
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT tenant_id, app_id, domain, use_case, source_id, universe_group, rank, ticker, ticker_key,
  company, company_key, asset_type, exchange, sector, sector_key, industry, industry_key,
  is_active, metadata, created_at, updated_at
FROM marketops_asset_universe
WHERE tenant_id = $1
  AND universe_group = $2
  AND ($3 = false OR is_active = true)
ORDER BY rank ASC
LIMIT $4`, strings.TrimSpace(tenantID), universeGroup, activeOnly, clampLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops assets: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsAssetRecord{}
	for rows.Next() {
		record, err := scanMarketOpsAsset(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops assets rows: %w", err)
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

const replayJobSelect = `
SELECT replay_job_id, tenant_id, source_id, dataset, source_kind, replay_mode, status, requested_by,
 window_start, window_end, started_at, completed_at, filters, options, result, error_message, created_at, updated_at
FROM replay_jobs`

type replayJobScanner interface{ Scan(dest ...any) error }

func scanReplayJob(scanner replayJobScanner) (storage.ReplayJobRecord, error) {
	var record storage.ReplayJobRecord
	var sourceID, dataset, errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime
	if err := scanner.Scan(&record.ReplayJobID, &record.TenantID, &sourceID, &dataset, &record.SourceKind,
		&record.ReplayMode, &record.Status, &record.RequestedBy, &record.WindowStart, &record.WindowEnd,
		&startedAt, &completedAt, &record.FiltersJSON, &record.OptionsJSON, &record.ResultJSON,
		&errorMessage, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.ReplayJobRecord{}, mapScanError("scan replay job", err)
	}
	record.SourceID = sourceID.String
	record.Dataset = dataset.String
	record.ErrorMessage = errorMessage.String
	if startedAt.Valid {
		record.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	return record, nil
}

type replayWorkerHeartbeatScanner interface{ Scan(dest ...any) error }

func scanReplayWorkerHeartbeat(scanner replayWorkerHeartbeatScanner) (storage.ReplayWorkerHeartbeatRecord, error) {
	var record storage.ReplayWorkerHeartbeatRecord
	var lastClaimedAt sql.NullTime
	var lastClaimedReplayJobID sql.NullString
	var lastCompletedAt sql.NullTime
	var lastCompletedReplayJobID sql.NullString
	var lastErrorAt sql.NullTime
	var lastErrorMessage sql.NullString
	if err := scanner.Scan(
		&record.WorkerID, &record.Status, &record.ProcessStartedAt, &record.LastSeenAt, &lastClaimedAt, &lastClaimedReplayJobID,
		&lastCompletedAt, &lastCompletedReplayJobID, &lastErrorAt, &lastErrorMessage, &record.MetadataJSON, &record.CreatedAt, &record.UpdatedAt,
	); err != nil {
		return storage.ReplayWorkerHeartbeatRecord{}, mapScanError("scan replay worker heartbeat", err)
	}
	if lastClaimedAt.Valid {
		record.LastClaimedAt = &lastClaimedAt.Time
	}
	record.LastClaimedReplayJobID = lastClaimedReplayJobID.String
	if lastCompletedAt.Valid {
		record.LastCompletedAt = &lastCompletedAt.Time
	}
	record.LastCompletedReplayJobID = lastCompletedReplayJobID.String
	if lastErrorAt.Valid {
		record.LastErrorAt = &lastErrorAt.Time
	}
	record.LastErrorMessage = lastErrorMessage.String
	return record, nil
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

const signalSelect = `
SELECT signal_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, ingestion_mode, dataset,
 COALESCE(array_to_json(event_ids), '[]'::json)::text, COALESCE(array_to_json(artifact_ids), '[]'::json)::text,
 signal_type, detector_id, detector_version, model_version, signal_time, observation_time, effective_time,
 processing_time, window_start, window_end, confidence, severity, entities, supporting_metrics, graph_targets,
 semantic_evidence, evidence, recommendation, correlation_id, trace_id, causation_id, replay_job_id,
 broker_topic, broker_partition, broker_offset, event, created_at, updated_at FROM signal_ledger`

func signalByIDWhere(useTemporal bool) string {
	if useTemporal {
		return ` WHERE signal_id = $1 ORDER BY signal_time DESC LIMIT 1`
	}
	return ` WHERE signal_id = $1`
}

type signalLedgerScanner interface{ Scan(dest ...any) error }

func scanSignalLedger(scanner signalLedgerScanner) (storage.SignalLedgerRecord, error) {
	var record storage.SignalLedgerRecord
	var eventIDsJSON, artifactIDsJSON string
	var traceID, causationID, replayJobID sql.NullString
	if err := scanner.Scan(&record.SignalID, &record.TenantID, &record.SourceID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceDomain,
		&record.SourceAdapter, &record.IngestionMode, &record.Dataset, &eventIDsJSON, &artifactIDsJSON,
		&record.SignalType, &record.DetectorID, &record.DetectorVersion, &record.ModelVersion,
		&record.SignalTime, &record.ObservationTime, &record.EffectiveTime, &record.ProcessingTime,
		&record.WindowStart, &record.WindowEnd, &record.Confidence, &record.Severity, &record.EntitiesJSON,
		&record.SupportingMetrics, &record.GraphTargetsJSON, &record.SemanticEvidenceJSON,
		&record.EvidenceJSON, &record.RecommendationJSON, &record.CorrelationID, &traceID, &causationID,
		&replayJobID, &record.BrokerTopic, &record.BrokerPartition, &record.BrokerOffset, &record.EventJSON,
		&record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.SignalLedgerRecord{}, mapScanError("scan signal ledger", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.SignalLedgerRecord{}, fmt.Errorf("scan signal event ids: %w", err)
	}
	if err := json.Unmarshal([]byte(artifactIDsJSON), &record.ArtifactIDs); err != nil {
		return storage.SignalLedgerRecord{}, fmt.Errorf("scan signal artifact ids: %w", err)
	}
	record.TraceID, record.CausationID, record.ReplayJobID = traceID.String, causationID.String, replayJobID.String
	return record, nil
}

const alertSelect = `
SELECT alert_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, dataset, signal_id, detector_id,
 alert_type, severity, status, title, summary, confidence, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 entities, evidence, recommendation, correlation_id, first_observed_at, last_observed_at,
 acknowledged_at, acknowledged_by, resolved_at, resolved_by, metadata, created_at, updated_at FROM alert_ledger`

type alertLedgerScanner interface{ Scan(dest ...any) error }

func scanAlertLedger(scanner alertLedgerScanner) (storage.AlertLedgerRecord, error) {
	var record storage.AlertLedgerRecord
	var eventIDsJSON string
	var acknowledgedAt, resolvedAt sql.NullTime
	var acknowledgedBy, resolvedBy sql.NullString
	if err := scanner.Scan(&record.AlertID, &record.TenantID, &record.SourceID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceDomain,
		&record.SourceAdapter, &record.Dataset, &record.SignalID, &record.DetectorID, &record.AlertType,
		&record.Severity, &record.Status, &record.Title, &record.Summary, &record.Confidence, &eventIDsJSON,
		&record.EntitiesJSON, &record.EvidenceJSON, &record.RecommendationJSON, &record.CorrelationID,
		&record.FirstObservedAt, &record.LastObservedAt, &acknowledgedAt, &acknowledgedBy, &resolvedAt,
		&resolvedBy, &record.MetadataJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.AlertLedgerRecord{}, mapScanError("scan alert ledger", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.AlertLedgerRecord{}, fmt.Errorf("scan alert event ids: %w", err)
	}
	if acknowledgedAt.Valid {
		record.AcknowledgedAt = &acknowledgedAt.Time
	}
	if resolvedAt.Valid {
		record.ResolvedAt = &resolvedAt.Time
	}
	record.AcknowledgedBy, record.ResolvedBy = acknowledgedBy.String, resolvedBy.String
	return record, nil
}

const insightSelect = `
SELECT insight_id, tenant_id, source_id, app_id, domain, use_case, source_domain, source_adapter, dataset, signal_id, detector_id,
 insight_type, status, title, summary, confidence, severity, COALESCE(array_to_json(event_ids), '[]'::json)::text,
 entities, supporting_metrics, semantic_evidence, recommendation, correlation_id, observed_at,
 reviewed_at, reviewed_by, metadata, created_at, updated_at FROM insight_ledger`

type insightLedgerScanner interface{ Scan(dest ...any) error }

func scanInsightLedger(scanner insightLedgerScanner) (storage.InsightLedgerRecord, error) {
	var record storage.InsightLedgerRecord
	var eventIDsJSON string
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullString
	if err := scanner.Scan(&record.InsightID, &record.TenantID, &record.SourceID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceDomain,
		&record.SourceAdapter, &record.Dataset, &record.SignalID, &record.DetectorID, &record.InsightType,
		&record.Status, &record.Title, &record.Summary, &record.Confidence, &record.Severity, &eventIDsJSON,
		&record.EntitiesJSON, &record.SupportingMetrics, &record.SemanticEvidenceJSON, &record.RecommendationJSON,
		&record.CorrelationID, &record.ObservedAt, &reviewedAt, &reviewedBy, &record.MetadataJSON,
		&record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.InsightLedgerRecord{}, mapScanError("scan insight ledger", err)
	}
	if err := json.Unmarshal([]byte(eventIDsJSON), &record.EventIDs); err != nil {
		return storage.InsightLedgerRecord{}, fmt.Errorf("scan insight event ids: %w", err)
	}
	if reviewedAt.Valid {
		record.ReviewedAt = &reviewedAt.Time
	}
	record.ReviewedBy = reviewedBy.String
	return record, nil
}

const normalizedEventSelect = `
SELECT event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset, idempotency_key, schema_id, schema_version,
 observation_time, processing_time, confidence, raw_topic, raw_partition, raw_offset,
 normalized_topic, normalized_partition, normalized_offset, normalized_payload, entities, evidence, metadata, event,
 created_at, updated_at FROM normalized_event_ledger`

func normalizedEventByIDWhere(useTemporal bool) string {
	if useTemporal {
		return ` WHERE event_id = $1 ORDER BY observation_time DESC LIMIT 1`
	}
	return ` WHERE event_id = $1`
}

type normalizedLedgerScanner interface{ Scan(dest ...any) error }

func scanNormalizedEventLedger(scanner normalizedLedgerScanner) (storage.NormalizedEventLedgerRecord, error) {
	var record storage.NormalizedEventLedgerRecord
	if err := scanner.Scan(&record.EventID, &record.TenantID, &record.SourceID, &record.AppID, &record.Domain, &record.UseCase, &record.SourceAdapter,
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

type marketOpsAssetScanner interface {
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
		&record.AppID,
		&record.Domain,
		&record.UseCase,
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

func scanMarketOpsAsset(scanner marketOpsAssetScanner) (storage.MarketOpsAssetRecord, error) {
	var record storage.MarketOpsAssetRecord
	if err := scanner.Scan(
		&record.TenantID,
		&record.AppID,
		&record.Domain,
		&record.UseCase,
		&record.SourceID,
		&record.UniverseGroup,
		&record.Rank,
		&record.Ticker,
		&record.TickerKey,
		&record.Company,
		&record.CompanyKey,
		&record.AssetType,
		&record.Exchange,
		&record.Sector,
		&record.SectorKey,
		&record.Industry,
		&record.IndustryKey,
		&record.IsActive,
		&record.MetadataJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return storage.MarketOpsAssetRecord{}, mapScanError("scan marketops asset", err)
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

func recordAppID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return appmeta.DefaultAppID
	}
	return value
}

func recordDomain(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "custom"
	}
	return value
}

func recordUseCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return appmeta.DefaultUseCase
	}
	return value
}

func nonNegativeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
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

func validateReplayJob(record storage.ReplayJobRecord) error {
	if strings.TrimSpace(record.ReplayJobID) == "" {
		return errors.New("replay job id is required")
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return errors.New("replay tenant id is required")
	}
	if !allowedString(record.SourceKind, storage.ReplaySourceRaw, storage.ReplaySourceNormalized, storage.ReplaySourceSignals) {
		return errors.New("replay source kind is invalid")
	}
	if !allowedString(record.ReplayMode, storage.ReplayModeOriginal, storage.ReplayModeLatestCompatible, storage.ReplayModeExplicit) {
		return errors.New("replay mode is invalid")
	}
	if !allowedString(record.Status, storage.ReplayJobStatusQueued, storage.ReplayJobStatusRunning, storage.ReplayJobStatusSucceeded, storage.ReplayJobStatusFailed, storage.ReplayJobStatusCanceled) {
		return errors.New("replay status is invalid")
	}
	if strings.TrimSpace(record.RequestedBy) == "" {
		return errors.New("replay requested by is required")
	}
	if record.WindowStart.IsZero() || record.WindowEnd.IsZero() {
		return errors.New("replay window is required")
	}
	if !record.WindowEnd.After(record.WindowStart) {
		return errors.New("replay window end must be after start")
	}
	return nil
}

func allowedString(value string, allowed ...string) bool {
	value = strings.TrimSpace(value)
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
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

func validateSignalLedger(record storage.SignalLedgerRecord) error {
	for name, value := range map[string]string{
		"signal id": record.SignalID, "tenant id": record.TenantID, "source id": record.SourceID,
		"source domain": record.SourceDomain, "source adapter": record.SourceAdapter,
		"ingestion mode": record.IngestionMode, "dataset": record.Dataset, "signal type": record.SignalType,
		"detector id": record.DetectorID, "detector version": record.DetectorVersion,
		"model version": record.ModelVersion, "severity": record.Severity,
		"correlation id": record.CorrelationID, "broker topic": record.BrokerTopic,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("signal ledger %s is required", name)
		}
	}
	if len(record.EventIDs) == 0 {
		return errors.New("signal ledger event ids are required")
	}
	for name, value := range map[string]time.Time{"signal time": record.SignalTime, "observation time": record.ObservationTime,
		"effective time": record.EffectiveTime, "processing time": record.ProcessingTime,
		"window start": record.WindowStart, "window end": record.WindowEnd} {
		if value.IsZero() {
			return fmt.Errorf("signal ledger %s is required", name)
		}
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("signal ledger confidence must be between 0 and 1")
	}
	if len(record.EventJSON) == 0 {
		return errors.New("signal ledger event json is required")
	}
	return nil
}

func validateAlertLifecycleMutation(mutation storage.AlertLifecycleMutation) error {
	if strings.TrimSpace(mutation.AlertID) == "" {
		return errors.New("alert id is required")
	}
	switch strings.TrimSpace(mutation.Status) {
	case storage.AlertStatusAcknowledged, storage.AlertStatusResolved, storage.AlertStatusSuppressed:
	default:
		return errors.New("unsupported alert lifecycle status")
	}
	if strings.TrimSpace(mutation.Actor) == "" {
		return errors.New("actor is required")
	}
	if mutation.MutatedAt.IsZero() {
		return errors.New("mutated at is required")
	}
	return validateJSONObject("alert lifecycle metadata", jsonOrEmpty(mutation.MetadataJSON))
}

func validateInsightLifecycleMutation(mutation storage.InsightLifecycleMutation) error {
	if strings.TrimSpace(mutation.InsightID) == "" {
		return errors.New("insight id is required")
	}
	switch strings.TrimSpace(mutation.Status) {
	case storage.InsightStatusReviewed, storage.InsightStatusDismissed, storage.InsightStatusArchived:
	default:
		return errors.New("unsupported insight lifecycle status")
	}
	if strings.TrimSpace(mutation.Actor) == "" {
		return errors.New("actor is required")
	}
	if mutation.MutatedAt.IsZero() {
		return errors.New("mutated at is required")
	}
	return validateJSONObject("insight lifecycle metadata", jsonOrEmpty(mutation.MetadataJSON))
}

func validateAlertLedger(record storage.AlertLedgerRecord) error {
	for name, value := range map[string]string{
		"alert id": record.AlertID, "tenant id": record.TenantID, "source id": record.SourceID,
		"source domain": record.SourceDomain, "source adapter": record.SourceAdapter, "dataset": record.Dataset,
		"signal id": record.SignalID, "detector id": record.DetectorID, "alert type": record.AlertType,
		"severity": record.Severity, "status": record.Status, "title": record.Title, "summary": record.Summary,
		"correlation id": record.CorrelationID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("alert ledger %s is required", name)
		}
	}
	if len(record.EventIDs) == 0 {
		return errors.New("alert ledger event ids are required")
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("alert ledger confidence must be between 0 and 1")
	}
	if record.FirstObservedAt.IsZero() || record.LastObservedAt.IsZero() {
		return errors.New("alert ledger observed times are required")
	}
	return nil
}

func validateInsightLedger(record storage.InsightLedgerRecord) error {
	for name, value := range map[string]string{
		"insight id": record.InsightID, "tenant id": record.TenantID, "source id": record.SourceID,
		"source domain": record.SourceDomain, "source adapter": record.SourceAdapter, "dataset": record.Dataset,
		"signal id": record.SignalID, "detector id": record.DetectorID, "insight type": record.InsightType,
		"status": record.Status, "title": record.Title, "summary": record.Summary, "severity": record.Severity,
		"correlation id": record.CorrelationID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("insight ledger %s is required", name)
		}
	}
	if len(record.EventIDs) == 0 {
		return errors.New("insight ledger event ids are required")
	}
	if record.Confidence < 0 || record.Confidence > 1 {
		return errors.New("insight ledger confidence must be between 0 and 1")
	}
	if record.ObservedAt.IsZero() {
		return errors.New("insight ledger observed at is required")
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

func validateJSONObject(name string, value []byte) error {
	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return fmt.Errorf("%s must be a JSON object: %w", name, err)
	}
	if decoded == nil {
		return fmt.Errorf("%s must be a JSON object", name)
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

func nullableJSON(value []byte) any {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" || trimmed == "null" {
		return nil
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
