package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const equityReconciliationTaskSelect = `
SELECT task_id, tenant_id, source_id, universe_group, dataset, observation_date, symbol,
  universe_rank, status, provider_attempts, max_provider_attempts, replay_count,
  next_attempt_at, lease_expires_at, raw_event_id, idempotency_key, last_error,
  completed_at, created_at, updated_at
FROM marketops_equity_reconciliation_tasks`

type equityReconciliationTaskScanner interface{ Scan(dest ...any) error }

func scanEquityReconciliationTask(scanner equityReconciliationTaskScanner) (storage.EquityReconciliationTaskRecord, error) {
	var record storage.EquityReconciliationTaskRecord
	var leaseExpiresAt, completedAt sql.NullTime
	var rawEventID, idempotencyKey, lastError sql.NullString
	if err := scanner.Scan(
		&record.TaskID, &record.TenantID, &record.SourceID, &record.UniverseGroup,
		&record.Dataset, &record.ObservationDate, &record.Symbol, &record.UniverseRank,
		&record.Status, &record.ProviderAttempts, &record.MaxProviderAttempts,
		&record.ReplayCount, &record.NextAttemptAt, &leaseExpiresAt, &rawEventID,
		&idempotencyKey, &lastError, &completedAt, &record.CreatedAt, &record.UpdatedAt,
	); err != nil {
		return storage.EquityReconciliationTaskRecord{}, mapScanError("scan equity reconciliation task", err)
	}
	if leaseExpiresAt.Valid {
		record.LeaseExpiresAt = &leaseExpiresAt.Time
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}
	record.RawEventID = rawEventID.String
	record.IdempotencyKey = idempotencyKey.String
	record.LastError = lastError.String
	return record, nil
}

func (r *Repository) EnqueueEquityReconciliationTask(ctx context.Context, record storage.EquityReconciliationTaskRecord) (storage.EquityReconciliationTaskRecord, error) {
	if strings.TrimSpace(record.TaskID) == "" || strings.TrimSpace(record.TenantID) == "" ||
		strings.TrimSpace(record.SourceID) == "" || strings.TrimSpace(record.UniverseGroup) == "" ||
		strings.TrimSpace(record.Symbol) == "" || record.ObservationDate.IsZero() ||
		record.UniverseRank <= 0 || record.MaxProviderAttempts <= 0 {
		return storage.EquityReconciliationTaskRecord{}, errors.New("invalid equity reconciliation task")
	}
	if strings.TrimSpace(record.Dataset) == "" {
		record.Dataset = "equity_eod_prices"
	}
	if strings.TrimSpace(record.Status) == "" {
		record.Status = storage.EquityReconciliationStatusQueued
	}
	if record.NextAttemptAt.IsZero() {
		record.NextAttemptAt = time.Now().UTC()
	}
	row := r.db.QueryRowContext(ctx, `
INSERT INTO marketops_equity_reconciliation_tasks (
  task_id, tenant_id, source_id, universe_group, dataset, observation_date, symbol,
  universe_rank, status, provider_attempts, max_provider_attempts, replay_count, next_attempt_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (tenant_id, source_id, universe_group, dataset, observation_date, symbol)
DO UPDATE SET universe_rank=EXCLUDED.universe_rank,
  max_provider_attempts=EXCLUDED.max_provider_attempts, updated_at=now()
RETURNING task_id, tenant_id, source_id, universe_group, dataset, observation_date, symbol,
  universe_rank, status, provider_attempts, max_provider_attempts, replay_count,
  next_attempt_at, lease_expires_at, raw_event_id, idempotency_key, last_error,
  completed_at, created_at, updated_at`,
		record.TaskID, strings.TrimSpace(record.TenantID), strings.TrimSpace(record.SourceID),
		strings.TrimSpace(record.UniverseGroup), record.Dataset, record.ObservationDate,
		strings.ToUpper(strings.TrimSpace(record.Symbol)), record.UniverseRank, record.Status,
		record.ProviderAttempts, record.MaxProviderAttempts, record.ReplayCount, record.NextAttemptAt)
	return scanEquityReconciliationTask(row)
}

func (r *Repository) ListEquityReconciliationTasks(ctx context.Context, tenantID string, sourceID string, universeGroup string, observationDate time.Time) ([]storage.EquityReconciliationTaskRecord, error) {
	rows, err := r.db.QueryContext(ctx, equityReconciliationTaskSelect+`
WHERE tenant_id=$1 AND source_id=$2 AND universe_group=$3 AND observation_date=$4
ORDER BY universe_rank ASC`, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), strings.TrimSpace(universeGroup), observationDate)
	if err != nil {
		return nil, fmt.Errorf("list equity reconciliation tasks: %w", err)
	}
	defer rows.Close()
	records := []storage.EquityReconciliationTaskRecord{}
	for rows.Next() {
		record, err := scanEquityReconciliationTask(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list equity reconciliation task rows: %w", err)
	}
	return records, nil
}

func (r *Repository) ClaimNextEquityReconciliationTask(ctx context.Context, tenantID string, sourceID string, universeGroup string, observationDate time.Time, claimedAt time.Time, leaseDuration time.Duration) (storage.EquityReconciliationTaskRecord, error) {
	if leaseDuration <= 0 {
		leaseDuration = 2 * time.Minute
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE marketops_equity_reconciliation_tasks
SET status='queued', next_attempt_at=$5, lease_expires_at=NULL,
  last_error=CASE WHEN last_error IS NULL THEN 'recovered expired lease' ELSE last_error END,
  updated_at=now()
WHERE tenant_id=$1 AND source_id=$2 AND universe_group=$3 AND observation_date=$4
  AND status IN ('running','awaiting_normalization') AND lease_expires_at < $5`,
		strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), strings.TrimSpace(universeGroup), observationDate, claimedAt)
	if err != nil {
		return storage.EquityReconciliationTaskRecord{}, fmt.Errorf("recover equity reconciliation leases: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
WITH next_task AS (
  SELECT task_id FROM marketops_equity_reconciliation_tasks
  WHERE tenant_id=$1 AND source_id=$2 AND universe_group=$3 AND observation_date=$4
    AND status='queued' AND next_attempt_at <= $5
  ORDER BY universe_rank ASC
  FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE marketops_equity_reconciliation_tasks AS task
SET status='running', lease_expires_at=$6, updated_at=now()
FROM next_task WHERE task.task_id=next_task.task_id
RETURNING task.task_id, task.tenant_id, task.source_id, task.universe_group, task.dataset,
  task.observation_date, task.symbol, task.universe_rank, task.status,
  task.provider_attempts, task.max_provider_attempts, task.replay_count,
  task.next_attempt_at, task.lease_expires_at, task.raw_event_id, task.idempotency_key,
  task.last_error, task.completed_at, task.created_at, task.updated_at`,
		strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), strings.TrimSpace(universeGroup),
		observationDate, claimedAt, claimedAt.Add(leaseDuration))
	return scanEquityReconciliationTask(row)
}

func (r *Repository) UpdateEquityReconciliationTask(ctx context.Context, record storage.EquityReconciliationTaskRecord) error {
	if strings.TrimSpace(record.TaskID) == "" {
		return errors.New("equity reconciliation task id is required")
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE marketops_equity_reconciliation_tasks SET
  status=$2, provider_attempts=$3, max_provider_attempts=$4, replay_count=$5,
  next_attempt_at=$6, lease_expires_at=$7, raw_event_id=$8, idempotency_key=$9,
  last_error=$10, completed_at=$11, updated_at=now()
WHERE task_id=$1`, record.TaskID, record.Status, record.ProviderAttempts,
		record.MaxProviderAttempts, record.ReplayCount, record.NextAttemptAt,
		record.LeaseExpiresAt, nullString(record.RawEventID), nullString(record.IdempotencyKey),
		nullString(record.LastError), record.CompletedAt)
	if err != nil {
		return fmt.Errorf("update equity reconciliation task: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *Repository) RequeueFailedEquityReconciliationTasks(ctx context.Context, tenantID string, sourceID string, universeGroup string, observationDate time.Time, nextAttemptAt time.Time) (int, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE marketops_equity_reconciliation_tasks
SET status='queued', provider_attempts=0, replay_count=0, next_attempt_at=$5,
  lease_expires_at=NULL, last_error=NULL, completed_at=NULL, updated_at=now()
WHERE tenant_id=$1 AND source_id=$2 AND universe_group=$3 AND observation_date=$4
  AND status='failed'`, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID),
		strings.TrimSpace(universeGroup), observationDate, nextAttemptAt)
	if err != nil {
		return 0, fmt.Errorf("requeue failed equity reconciliation tasks: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count requeued equity reconciliation tasks: %w", err)
	}
	return int(affected), nil
}

func (r *Repository) HasNormalizedEquity(ctx context.Context, tenantID string, sourceID string, symbol string, observationDate time.Time) (bool, error) {
	var found bool
	err := r.temporal().QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM normalized_event_ledger
  WHERE tenant_id=$1 AND source_id=$2 AND dataset='equity_eod_prices'
    AND normalized_payload->>'symbol'=$3
    AND normalized_payload->>'observation_date'=$4
)`, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID), strings.ToUpper(strings.TrimSpace(symbol)), observationDate.UTC().Format("2006-01-02")).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check normalized equity: %w", err)
	}
	return found, nil
}

func (r *Repository) FindRawEquityEvent(ctx context.Context, tenantID string, sourceID string, symbol string, observationDate time.Time) (storage.RawEventLedgerRecord, error) {
	row := r.temporal().QueryRowContext(ctx, `
SELECT event_id, tenant_id, source_id, app_id, domain, use_case, source_adapter, dataset,
  idempotency_key, observation_time, processing_time, broker_topic, broker_partition,
  broker_offset, payload, entity_hints, created_at
FROM raw_event_ledger
WHERE tenant_id=$1 AND source_id=$2 AND dataset='equity_eod_prices'
  AND payload->'payload'->>'symbol'=$3
  AND payload->'payload'->>'observation_date'=$4
ORDER BY created_at DESC LIMIT 1`, strings.TrimSpace(tenantID), strings.TrimSpace(sourceID),
		strings.ToUpper(strings.TrimSpace(symbol)), observationDate.UTC().Format("2006-01-02"))
	return scanRawEventLedger(row)
}
