package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalValuationResults reads only parity-approved global VC/DOSM
// valuation results. The Gateway supplies symbols authorized by the selected list.
func (r *Repository) ListSubscriberGlobalValuationResults(ctx context.Context, symbols []string, eligibleOnly bool, limit int) ([]storage.MarketOpsValuationResultRecord, error) {
	if len(symbols) == 0 {
		return []storage.MarketOpsValuationResultRecord{}, nil
	}
	rowLimit := valuationLimit(limit)
	if perSymbol := len(symbols) * 6; perSymbol > rowLimit {
		rowLimit = perSymbol
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT result_id, snapshot_id, global_asset_id, symbol, session_date, algorithm_id,
  model_version, score, fair_value, classification, confidence, confidence_label,
  evaluation_status, eligible, result_json::text, created_at
FROM (
  SELECT DISTINCT ON (upper(symbol), algorithm_id)
    result_id, snapshot_id, global_asset_id, symbol, session_date, algorithm_id,
    model_version, score, fair_value, classification, confidence, confidence_label,
    evaluation_status, eligible, result_json, created_at
  FROM subscriber_gateway_global_valuation_results
  WHERE upper(symbol) = ANY($1)
    AND (NOT $2 OR eligible)
  ORDER BY upper(symbol), algorithm_id, session_date DESC, created_at DESC, result_id DESC
) latest
ORDER BY session_date DESC, score DESC, symbol ASC
LIMIT $3`, pqArray(symbols), eligibleOnly, rowLimit)
	if err != nil {
		return nil, fmt.Errorf("list subscriber global valuation results: %w", err)
	}
	defer rows.Close()
	out := []storage.MarketOpsValuationResultRecord{}
	for rows.Next() {
		var item storage.MarketOpsValuationResultRecord
		if err := rows.Scan(&item.ResultID, &item.SnapshotID, &item.TenantID, &item.Symbol, &item.SessionDate, &item.AlgorithmID, &item.ModelVersion, &item.Score, &item.FairValue, &item.Classification, &item.Confidence, &item.ConfidenceLabel, &item.EvaluationStatus, &item.Eligible, &item.ResultJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber global valuation result: %w", err)
		}
		item.TenantID = "platform-global"
		item.Symbol = strings.ToUpper(item.Symbol)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber global valuation results: %w", err)
	}
	return out, nil
}
