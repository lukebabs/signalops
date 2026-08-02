package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) LatestMarketOpsValuationSnapshot(ctx context.Context, tenantID, symbol, provider string) (storage.MarketOpsValuationSnapshotRecord, error) {
	var x storage.MarketOpsValuationSnapshotRecord
	err := r.db.QueryRowContext(ctx, `SELECT snapshot_id,tenant_id,symbol,session_date,available_at,sector,industry,provider,input_json,created_at FROM marketops_valuation_snapshots WHERE tenant_id=$1 AND symbol=$2 AND provider=$3 ORDER BY created_at DESC LIMIT 1`, strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(symbol)), strings.TrimSpace(provider)).Scan(&x.SnapshotID, &x.TenantID, &x.Symbol, &x.SessionDate, &x.AvailableAt, &x.Sector, &x.Industry, &x.Provider, &x.InputJSON, &x.CreatedAt)
	if err != nil {
		return x, fmt.Errorf("latest valuation snapshot: %w", err)
	}
	return x, nil
}

func (r *Repository) UpsertMarketOpsValuationSnapshot(ctx context.Context, x storage.MarketOpsValuationSnapshotRecord) error {
	if strings.TrimSpace(x.SnapshotID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Symbol) == "" || x.SessionDate.IsZero() || x.AvailableAt.IsZero() {
		return fmt.Errorf("invalid valuation snapshot")
	}
	var financialSnapshotID any
	if strings.TrimSpace(x.FinancialSnapshotID) != "" {
		financialSnapshotID = strings.TrimSpace(x.FinancialSnapshotID)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_valuation_snapshots (snapshot_id,financial_snapshot_id,tenant_id,symbol,session_date,available_at,sector,industry,provider,provider_request_ids,input_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (tenant_id,symbol,session_date,available_at) DO UPDATE SET financial_snapshot_id=EXCLUDED.financial_snapshot_id, sector=EXCLUDED.sector, industry=EXCLUDED.industry, provider=EXCLUDED.provider, provider_request_ids=EXCLUDED.provider_request_ids, input_json=EXCLUDED.input_json`, x.SnapshotID, financialSnapshotID, strings.TrimSpace(x.TenantID), strings.ToUpper(strings.TrimSpace(x.Symbol)), x.SessionDate.UTC(), x.AvailableAt.UTC(), strings.TrimSpace(x.Sector), strings.TrimSpace(x.Industry), strings.TrimSpace(x.Provider), pqArray(x.ProviderRequestIDs), jsonOrEmpty(x.InputJSON))
	if err != nil {
		return fmt.Errorf("upsert valuation snapshot: %w", err)
	}
	return nil
}

func (r *Repository) UpsertMarketOpsValuationResult(ctx context.Context, x storage.MarketOpsValuationResultRecord) error {
	if strings.TrimSpace(x.ResultID) == "" || strings.TrimSpace(x.SnapshotID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Symbol) == "" || strings.TrimSpace(x.AlgorithmID) == "" || x.SessionDate.IsZero() {
		return fmt.Errorf("invalid valuation result")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_valuation_results (result_id,snapshot_id,tenant_id,symbol,session_date,algorithm_id,model_version,score,fair_value,classification,confidence,confidence_label,evaluation_status,eligible,result_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT (snapshot_id,algorithm_id) DO UPDATE SET model_version=EXCLUDED.model_version, score=EXCLUDED.score, fair_value=EXCLUDED.fair_value, classification=EXCLUDED.classification, confidence=EXCLUDED.confidence, confidence_label=EXCLUDED.confidence_label, evaluation_status=EXCLUDED.evaluation_status, eligible=EXCLUDED.eligible, result_json=EXCLUDED.result_json`, x.ResultID, x.SnapshotID, strings.TrimSpace(x.TenantID), strings.ToUpper(strings.TrimSpace(x.Symbol)), x.SessionDate.UTC(), strings.TrimSpace(x.AlgorithmID), strings.TrimSpace(x.ModelVersion), x.Score, x.FairValue, strings.TrimSpace(x.Classification), x.Confidence, strings.TrimSpace(x.ConfidenceLabel), strings.TrimSpace(x.EvaluationStatus), x.Eligible, jsonOrEmpty(x.ResultJSON))
	if err != nil {
		return fmt.Errorf("upsert valuation result: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsValuationResults(ctx context.Context, f storage.MarketOpsValuationFilter) ([]storage.MarketOpsValuationResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM (SELECT DISTINCT ON (vr.symbol,vr.session_date,vr.algorithm_id) vr.result_id,vr.snapshot_id,vr.tenant_id,vr.symbol,vr.session_date,vr.algorithm_id,vr.model_version,vr.score,vr.fair_value,vr.classification,vr.confidence,vr.confidence_label,vr.evaluation_status,vr.eligible,vr.result_json,vr.created_at FROM marketops_valuation_results vr WHERE vr.tenant_id=$1 AND ($2='' OR vr.symbol=$2) AND ($3='' OR vr.algorithm_id=$3) AND ($4::date IS NULL OR vr.session_date=$4::date) AND (NOT $5 OR vr.eligible=true) ORDER BY vr.symbol,vr.session_date,vr.algorithm_id,vr.created_at DESC) latest ORDER BY session_date DESC,score DESC,symbol ASC LIMIT $6`, strings.TrimSpace(f.TenantID), strings.ToUpper(strings.TrimSpace(f.Symbol)), strings.TrimSpace(f.AlgorithmID), nullTime(f.SessionDate), f.EligibleOnly, valuationLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list valuation results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsValuationResultRecord{}
	for rows.Next() {
		var x storage.MarketOpsValuationResultRecord
		if err := rows.Scan(&x.ResultID, &x.SnapshotID, &x.TenantID, &x.Symbol, &x.SessionDate, &x.AlgorithmID, &x.ModelVersion, &x.Score, &x.FairValue, &x.Classification, &x.Confidence, &x.ConfidenceLabel, &x.EvaluationStatus, &x.Eligible, &x.ResultJSON, &x.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan valuation result: %w", err)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate valuation results: %w", err)
	}
	return out, nil
}
