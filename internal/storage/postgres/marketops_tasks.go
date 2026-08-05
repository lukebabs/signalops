package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsTaskWorkflow(ctx context.Context, x storage.MarketOpsTaskWorkflowRecord) error {
	if strings.TrimSpace(x.WorkflowID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.WorkflowType) == "" || strings.TrimSpace(x.Status) == "" || x.SessionDate.IsZero() {
		return fmt.Errorf("invalid marketops task workflow")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_task_workflows (workflow_id,tenant_id,session_date,workflow_type,status,schedule_job_id,coverage,failure_class,error_message,started_at,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (tenant_id,session_date,workflow_type) DO UPDATE SET status=EXCLUDED.status,schedule_job_id=EXCLUDED.schedule_job_id,coverage=EXCLUDED.coverage,failure_class=EXCLUDED.failure_class,error_message=EXCLUDED.error_message,started_at=COALESCE(EXCLUDED.started_at,marketops_task_workflows.started_at),completed_at=EXCLUDED.completed_at,updated_at=now()`, x.WorkflowID, x.TenantID, x.SessionDate.UTC(), x.WorkflowType, x.Status, x.ScheduleJobID, jsonOrEmpty(x.CoverageJSON), x.FailureClass, x.ErrorMessage, x.StartedAt, x.CompletedAt)
	return err
}

func (r *Repository) UpsertMarketOpsTaskItem(ctx context.Context, x storage.MarketOpsTaskItemRecord) error {
	if strings.TrimSpace(x.TaskID) == "" || strings.TrimSpace(x.WorkflowID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.TaskType) == "" || strings.TrimSpace(x.Status) == "" || x.SessionDate.IsZero() {
		return fmt.Errorf("invalid marketops task item")
	}
	if x.MaxAttempts == 0 {
		x.MaxAttempts = 3
	}
	if x.NextAttemptAt.IsZero() {
		x.NextAttemptAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_task_items (task_id,workflow_id,tenant_id,session_date,task_type,symbol,status,attempt_count,max_attempts,next_attempt_at,lease_expires_at,provider,failure_class,provider_status,error_message,result,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) ON CONFLICT (tenant_id,session_date,task_type,symbol) DO UPDATE SET workflow_id=EXCLUDED.workflow_id,status=EXCLUDED.status,attempt_count=EXCLUDED.attempt_count,max_attempts=EXCLUDED.max_attempts,next_attempt_at=EXCLUDED.next_attempt_at,lease_expires_at=EXCLUDED.lease_expires_at,provider=EXCLUDED.provider,failure_class=EXCLUDED.failure_class,provider_status=EXCLUDED.provider_status,error_message=EXCLUDED.error_message,result=EXCLUDED.result,completed_at=EXCLUDED.completed_at,updated_at=now()`, x.TaskID, x.WorkflowID, x.TenantID, x.SessionDate.UTC(), x.TaskType, strings.ToUpper(strings.TrimSpace(x.Symbol)), x.Status, x.AttemptCount, x.MaxAttempts, x.NextAttemptAt, x.LeaseExpiresAt, x.Provider, x.FailureClass, x.ProviderStatus, x.ErrorMessage, jsonOrEmpty(x.ResultJSON), x.CompletedAt)
	return err
}

func (r *Repository) ListMarketOpsTaskItems(ctx context.Context, f storage.MarketOpsTaskItemFilter) ([]storage.MarketOpsTaskItemRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT task_id,workflow_id,tenant_id,session_date,task_type,symbol,status,attempt_count,max_attempts,next_attempt_at,lease_expires_at,provider,failure_class,provider_status,error_message,result,completed_at,created_at,updated_at FROM marketops_task_items WHERE ($1='' OR tenant_id=$1) AND ($2='' OR workflow_id=$2) AND ($3='' OR task_type=$3) AND ($4='' OR symbol=$4) AND ($5='' OR status=$5) AND ($6::date IS NULL OR session_date=$6::date) ORDER BY session_date DESC, task_type, symbol LIMIT $7`, strings.TrimSpace(f.TenantID), strings.TrimSpace(f.WorkflowID), strings.TrimSpace(f.TaskType), strings.ToUpper(strings.TrimSpace(f.Symbol)), strings.TrimSpace(f.Status), nullTime(f.SessionDate), clampLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops task items: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsTaskItemRecord{}
	for rows.Next() {
		var x storage.MarketOpsTaskItemRecord
		var lease, done sql.NullTime
		var code sql.NullInt32
		if err := rows.Scan(&x.TaskID, &x.WorkflowID, &x.TenantID, &x.SessionDate, &x.TaskType, &x.Symbol, &x.Status, &x.AttemptCount, &x.MaxAttempts, &x.NextAttemptAt, &lease, &x.Provider, &x.FailureClass, &code, &x.ErrorMessage, &x.ResultJSON, &done, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		if lease.Valid {
			x.LeaseExpiresAt = &lease.Time
		}
		if done.Valid {
			x.CompletedAt = &done.Time
		}
		if code.Valid {
			v := int(code.Int32)
			x.ProviderStatus = &v
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
