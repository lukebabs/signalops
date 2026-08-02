package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) UpsertMarketOpsFinancialStatement(ctx context.Context, x storage.MarketOpsFinancialStatementRecord) error {
	if strings.TrimSpace(x.StatementID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Symbol) == "" || strings.TrimSpace(x.StatementType) == "" || x.PeriodEnd.IsZero() || x.AcceptedAt.IsZero() {
		return fmt.Errorf("invalid financial statement")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_financial_statements (statement_id,tenant_id,symbol,statement_type,fiscal_period_end,accepted_at,fiscal_period,normalized_json,raw_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (tenant_id,symbol,statement_type,fiscal_period_end,accepted_at) DO NOTHING`, x.StatementID, strings.TrimSpace(x.TenantID), strings.ToUpper(strings.TrimSpace(x.Symbol)), strings.TrimSpace(x.StatementType), x.PeriodEnd.UTC(), x.AcceptedAt.UTC(), strings.TrimSpace(x.Period), jsonOrEmpty(x.NormalizedJSON), jsonOrEmpty(x.RawJSON))
	if err != nil {
		return fmt.Errorf("upsert financial statement: %w", err)
	}
	return nil
}
func (r *Repository) UpsertMarketOpsFinancialSnapshot(ctx context.Context, x storage.MarketOpsFinancialSnapshotRecord) error {
	if strings.TrimSpace(x.FinancialSnapshotID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Symbol) == "" || strings.TrimSpace(x.SnapshotVersion) == "" || x.EvaluationDate.IsZero() || x.AvailableAt.IsZero() {
		return fmt.Errorf("invalid financial snapshot")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_financial_snapshots (financial_snapshot_id,tenant_id,symbol,snapshot_version,evaluation_date,available_at,statement_ids,input_json,derived_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (tenant_id,symbol,snapshot_version,evaluation_date,available_at) DO NOTHING`, x.FinancialSnapshotID, strings.TrimSpace(x.TenantID), strings.ToUpper(strings.TrimSpace(x.Symbol)), strings.TrimSpace(x.SnapshotVersion), x.EvaluationDate.UTC(), x.AvailableAt.UTC(), pqArray(x.StatementIDs), jsonOrEmpty(x.InputJSON), jsonOrEmpty(x.DerivedJSON))
	if err != nil {
		return fmt.Errorf("upsert financial snapshot: %w", err)
	}
	return nil
}
func (r *Repository) LatestMarketOpsFinancialSnapshot(ctx context.Context, tenantID, symbol string) (storage.MarketOpsFinancialSnapshotRecord, error) {
	var x storage.MarketOpsFinancialSnapshotRecord
	err := r.db.QueryRowContext(ctx, `SELECT financial_snapshot_id,tenant_id,symbol,snapshot_version,evaluation_date,available_at,statement_ids,input_json,derived_json FROM marketops_financial_snapshots WHERE tenant_id=$1 AND symbol=$2 ORDER BY created_at DESC LIMIT 1`, strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(symbol))).Scan(&x.FinancialSnapshotID, &x.TenantID, &x.Symbol, &x.SnapshotVersion, &x.EvaluationDate, &x.AvailableAt, pqArrayScan(&x.StatementIDs), &x.InputJSON, &x.DerivedJSON)
	if err != nil {
		return x, fmt.Errorf("latest financial snapshot: %w", err)
	}
	return x, nil
}
