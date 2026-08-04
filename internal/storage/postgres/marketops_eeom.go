package postgres

import (
	"context"
	"fmt"
	"github.com/lukebabs/signalops/internal/storage"
	"strings"
)

func (r *Repository) UpsertMarketOpsEEOMResult(ctx context.Context, x storage.MarketOpsEEOMResultRecord) error {
	if strings.TrimSpace(x.ResultID) == "" || strings.TrimSpace(x.TenantID) == "" || strings.TrimSpace(x.Symbol) == "" || strings.TrimSpace(x.EarningsEventID) == "" || x.EarningsDate.IsZero() || x.SessionDate.IsZero() || strings.TrimSpace(x.ModelVersion) == "" {
		return fmt.Errorf("invalid EEOM result")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_eeom_results (result_id,tenant_id,symbol,earnings_event_id,earnings_date,session_date,model_version,score,posture,classification,evidence_quality,eligible,result_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (tenant_id,symbol,earnings_event_id,session_date,model_version) DO UPDATE SET score=EXCLUDED.score,posture=EXCLUDED.posture,classification=EXCLUDED.classification,evidence_quality=EXCLUDED.evidence_quality,eligible=EXCLUDED.eligible,result_json=EXCLUDED.result_json`, x.ResultID, strings.TrimSpace(x.TenantID), strings.ToUpper(strings.TrimSpace(x.Symbol)), strings.TrimSpace(x.EarningsEventID), x.EarningsDate.UTC(), x.SessionDate.UTC(), strings.TrimSpace(x.ModelVersion), x.Score, strings.TrimSpace(x.Posture), strings.TrimSpace(x.Classification), strings.TrimSpace(x.EvidenceQuality), x.Eligible, jsonOrEmpty(x.ResultJSON))
	if err != nil {
		return fmt.Errorf("upsert EEOM result: %w", err)
	}
	return nil
}
func (r *Repository) ListMarketOpsEEOMResults(ctx context.Context, f storage.MarketOpsEEOMFilter) ([]storage.MarketOpsEEOMResultRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT ON (symbol,earnings_event_id) result_id,tenant_id,symbol,earnings_event_id,earnings_date,session_date,model_version,score,posture,classification,evidence_quality,eligible,result_json,created_at FROM marketops_eeom_results WHERE tenant_id=$1 AND ($2='' OR symbol=$2) AND ($3::date IS NULL OR earnings_date >= $3::date) AND ($4::date IS NULL OR earnings_date <= $4::date) AND (NOT $5 OR eligible) ORDER BY symbol,earnings_event_id,session_date DESC,created_at DESC LIMIT $6`, strings.TrimSpace(f.TenantID), strings.ToUpper(strings.TrimSpace(f.Symbol)), nullTime(f.StartDate), nullTime(f.EndDate), f.EligibleOnly, valuationLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list EEOM results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsEEOMResultRecord{}
	for rows.Next() {
		var x storage.MarketOpsEEOMResultRecord
		if err := rows.Scan(&x.ResultID, &x.TenantID, &x.Symbol, &x.EarningsEventID, &x.EarningsDate, &x.SessionDate, &x.ModelVersion, &x.Score, &x.Posture, &x.Classification, &x.EvidenceQuality, &x.Eligible, &x.ResultJSON, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
