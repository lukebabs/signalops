package postgres

import (
	"context"
	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListRetentionPolicies(ctx context.Context, tenant string) ([]storage.RetentionPolicyRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,policy_id,app_id,domain,data_class,retention_days,mode,preservation_rule,description,updated_at FROM retention_policies WHERE tenant_id=$1 ORDER BY app_id,policy_id`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.RetentionPolicyRecord{}
	for rows.Next() {
		var x storage.RetentionPolicyRecord
		if err = rows.Scan(&x.TenantID, &x.PolicyID, &x.AppID, &x.Domain, &x.DataClass, &x.RetentionDays, &x.Mode, &x.PreservationRule, &x.Description, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) ListRetentionRuns(ctx context.Context, tenant string, limit int) ([]storage.RetentionRunRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,tenant_id,policy_id,mode,status,candidate_rows,affected_rows,oldest_candidate_at,newest_candidate_at,detail,started_at,completed_at FROM retention_runs WHERE tenant_id=$1 ORDER BY started_at DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storage.RetentionRunRecord{}
	for rows.Next() {
		var x storage.RetentionRunRecord
		if err = rows.Scan(&x.RunID, &x.TenantID, &x.PolicyID, &x.Mode, &x.Status, &x.CandidateRows, &x.AffectedRows, &x.OldestCandidateAt, &x.NewestCandidateAt, &x.DetailJSON, &x.StartedAt, &x.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
