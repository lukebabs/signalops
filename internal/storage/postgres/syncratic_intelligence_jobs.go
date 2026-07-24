package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const syncraticIntelligenceJobSelect = `SELECT job_id, tenant_id, app_id, use_case, subject_symbol, session_date,
 context_window_id, evidence_digest, status, attempts, max_attempts, COALESCE(lease_expires_at, 'epoch'::timestamptz),
 COALESCE(ask_query_id,''), COALESCE(syncratic_insight_id,''), COALESCE(error_code,''), COALESCE(error_message,''),
 created_at, updated_at, COALESCE(completed_at, 'epoch'::timestamptz) FROM syncratic_intelligence_jobs`

func (r *Repository) UpsertSyncraticIntelligenceJob(ctx context.Context, record storage.SyncraticIntelligenceJobRecord) error {
	if strings.TrimSpace(record.JobID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.SubjectSymbol) == "" || record.SessionDate.IsZero() || strings.TrimSpace(record.ContextWindowID) == "" || strings.TrimSpace(record.EvidenceDigest) == "" {
		return fmt.Errorf("syncratic intelligence job_id, tenant_id, subject_symbol, session_date, context_window_id, and evidence_digest are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO syncratic_intelligence_jobs (
 job_id, tenant_id, app_id, use_case, subject_symbol, session_date, context_window_id, evidence_digest, status, attempts, max_attempts
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'queued',0,$9)
ON CONFLICT (tenant_id, app_id, use_case, subject_symbol, session_date, evidence_digest) DO NOTHING`,
		record.JobID, strings.TrimSpace(record.TenantID), recordAppID(record.AppID), recordUseCase(record.UseCase), strings.ToUpper(strings.TrimSpace(record.SubjectSymbol)), record.SessionDate.UTC(), strings.TrimSpace(record.ContextWindowID), strings.TrimSpace(record.EvidenceDigest), syncraticJobMaxAttempts(record.MaxAttempts, 3))
	if err != nil { return fmt.Errorf("upsert syncratic intelligence job: %w", err) }
	return nil
}

func (r *Repository) ListSyncraticIntelligenceJobs(ctx context.Context, filter storage.SyncraticIntelligenceJobFilter) ([]storage.SyncraticIntelligenceJobRecord, error) {
	rows, err := r.db.QueryContext(ctx, syncraticIntelligenceJobSelect+` WHERE ($1='' OR tenant_id=$1) AND ($2='' OR status=$2) ORDER BY created_at DESC LIMIT $3`, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.Status), clampLimit(filter.Limit))
	if err != nil { return nil, fmt.Errorf("list syncratic intelligence jobs: %w", err) }
	defer rows.Close()
	var records []storage.SyncraticIntelligenceJobRecord
	for rows.Next() { record, err := scanSyncraticIntelligenceJob(rows); if err != nil { return nil, err }; records = append(records, record) }
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("list syncratic intelligence jobs rows: %w", err) }
	return records, nil
}

func (r *Repository) ClaimSyncraticIntelligenceJob(ctx context.Context, now time.Time, lease time.Duration) (storage.SyncraticIntelligenceJobRecord, error) {
	if lease <= 0 { lease = 90 * time.Second }
	row := r.db.QueryRowContext(ctx, `WITH candidate AS (
 SELECT job_id FROM syncratic_intelligence_jobs
 WHERE (status IN ('queued','retryable_failed') OR (status='running' AND lease_expires_at < $1)) AND attempts < max_attempts
 ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
) UPDATE syncratic_intelligence_jobs j SET status='running', attempts=j.attempts+1, lease_expires_at=$1+$2::interval,
 error_code=NULL, error_message=NULL, updated_at=$1 FROM candidate WHERE j.job_id=candidate.job_id
 RETURNING j.job_id, j.tenant_id, j.app_id, j.use_case, j.subject_symbol, j.session_date, j.context_window_id, j.evidence_digest,
 j.status, j.attempts, j.max_attempts, COALESCE(j.lease_expires_at, 'epoch'::timestamptz), COALESCE(j.ask_query_id,''),
 COALESCE(j.syncratic_insight_id,''), COALESCE(j.error_code,''), COALESCE(j.error_message,''), j.created_at, j.updated_at, COALESCE(j.completed_at, 'epoch'::timestamptz)`, now.UTC(), fmt.Sprintf("%f seconds", lease.Seconds()))
	record, err := scanSyncraticIntelligenceJob(row)
	if err == sql.ErrNoRows { return storage.SyncraticIntelligenceJobRecord{}, storage.ErrNotFound }
	return record, err
}

func (r *Repository) CompleteSyncraticIntelligenceJob(ctx context.Context, jobID, insightID, askQueryID string, completedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE syncratic_intelligence_jobs SET status='completed', syncratic_insight_id=$2, ask_query_id=$3, completed_at=$4, lease_expires_at=NULL, updated_at=$4 WHERE job_id=$1`, strings.TrimSpace(jobID), strings.TrimSpace(insightID), strings.TrimSpace(askQueryID), completedAt.UTC())
	if err != nil { return fmt.Errorf("complete syncratic intelligence job: %w", err) }
	n, _ := result.RowsAffected(); if n == 0 { return storage.ErrNotFound }; return nil
}

func (r *Repository) FailSyncraticIntelligenceJob(ctx context.Context, jobID, errorCode, errorMessage string, failedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE syncratic_intelligence_jobs SET status=CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'retryable_failed' END, error_code=$2, error_message=$3, lease_expires_at=NULL, updated_at=$4 WHERE job_id=$1`, strings.TrimSpace(jobID), strings.TrimSpace(errorCode), truncateSyncraticJobError(errorMessage), failedAt.UTC())
	if err != nil { return fmt.Errorf("fail syncratic intelligence job: %w", err) }
	n, _ := result.RowsAffected(); if n == 0 { return storage.ErrNotFound }; return nil
}

type syncraticIntelligenceJobScanner interface{ Scan(dest ...any) error }
func scanSyncraticIntelligenceJob(scanner syncraticIntelligenceJobScanner) (storage.SyncraticIntelligenceJobRecord, error) {
	var r storage.SyncraticIntelligenceJobRecord
	if err := scanner.Scan(&r.JobID, &r.TenantID, &r.AppID, &r.UseCase, &r.SubjectSymbol, &r.SessionDate, &r.ContextWindowID, &r.EvidenceDigest, &r.Status, &r.Attempts, &r.MaxAttempts, &r.LeaseExpiresAt, &r.AskQueryID, &r.SyncraticInsightID, &r.ErrorCode, &r.ErrorMessage, &r.CreatedAt, &r.UpdatedAt, &r.CompletedAt); err != nil { return storage.SyncraticIntelligenceJobRecord{}, mapScanError("scan syncratic intelligence job", err) }
	return r, nil
}

func syncraticJobMaxAttempts(value, fallback int) int { if value > 0 { return value }; return fallback }
func truncateSyncraticJobError(value string) string { value = strings.TrimSpace(value); if len(value) > 1000 { return value[:1000] }; return value }
