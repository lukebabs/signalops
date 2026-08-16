package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalEEOMResults reads only parity-approved global EEOM
// results. The Gateway supplies symbols authorized by the selected list.
func (r *Repository) ListSubscriberGlobalEEOMResults(ctx context.Context, symbols []string, f storage.MarketOpsEEOMFilter) ([]storage.MarketOpsEEOMResultRecord, error) {
	if len(symbols) == 0 {
		return []storage.MarketOpsEEOMResultRecord{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT result_id, global_asset_id, symbol, earnings_event_id, earnings_date,
  session_date, model_version, score, posture, classification, evidence_quality,
  eligible, result_json::text, created_at
FROM subscriber_gateway_global_eeom_results
WHERE upper(symbol) = ANY($1)
  AND ($2::date IS NULL OR earnings_date >= $2::date)
  AND ($3::date IS NULL OR earnings_date <= $3::date)
  AND (NOT $4 OR eligible)
ORDER BY earnings_date ASC, session_date DESC, score DESC, symbol ASC
LIMIT $5`, pqArray(symbols), nullTime(f.StartDate), nullTime(f.EndDate), f.EligibleOnly, valuationLimit(f.Limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global EEOM results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsEEOMResultRecord{}
	for rows.Next() {
		var item storage.MarketOpsEEOMResultRecord
		if err := rows.Scan(&item.ResultID, &item.TenantID, &item.Symbol, &item.EarningsEventID, &item.EarningsDate, &item.SessionDate, &item.ModelVersion, &item.Score, &item.Posture, &item.Classification, &item.EvidenceQuality, &item.Eligible, &item.ResultJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber global EEOM result: %w", err)
		}
		item.TenantID = "platform-global"
		item.Symbol = strings.ToUpper(item.Symbol)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber global EEOM results: %w", err)
	}
	return out, nil
}
