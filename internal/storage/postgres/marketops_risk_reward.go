package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const marketOpsRiskRewardSnapshotSelect = `SELECT snapshot_id, tenant_id, algorithm_result_id, execution_request_id,
 symbol, session_date, observed_at, technical_score, technical_direction, risk_level, confidence,
 usable_input_count, required_input_count, eligible, result_payload, input_snapshot, created_at
FROM marketops_risk_reward_snapshots`

func (r *Repository) UpsertMarketOpsRiskRewardSnapshot(ctx context.Context, record storage.MarketOpsRiskRewardSnapshotRecord) error {
	if strings.TrimSpace(record.SnapshotID) == "" || strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.AlgorithmResultID) == "" || strings.TrimSpace(record.Symbol) == "" || record.SessionDate.IsZero() || record.ObservedAt.IsZero() || record.RequiredInputCount <= 0 || record.UsableInputCount < 0 {
		return fmt.Errorf("invalid risk/reward snapshot")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO marketops_risk_reward_snapshots (
 snapshot_id, tenant_id, algorithm_result_id, execution_request_id, symbol, session_date, observed_at,
 technical_score, technical_direction, risk_level, confidence, usable_input_count, required_input_count,
 eligible, result_payload, input_snapshot
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (tenant_id, algorithm_result_id) DO NOTHING`,
		strings.TrimSpace(record.SnapshotID), strings.TrimSpace(record.TenantID), strings.TrimSpace(record.AlgorithmResultID), strings.TrimSpace(record.ExecutionRequestID), strings.ToUpper(strings.TrimSpace(record.Symbol)), record.SessionDate.UTC(), record.ObservedAt.UTC(), record.TechnicalScore, strings.TrimSpace(record.TechnicalDirection), strings.TrimSpace(record.RiskLevel), record.Confidence, record.UsableInputCount, record.RequiredInputCount, record.Eligible, jsonOrEmpty(record.ResultPayloadJSON), jsonOrEmpty(record.InputSnapshotJSON))
	if err != nil {
		return fmt.Errorf("upsert risk/reward snapshot: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsRiskRewardSnapshots(ctx context.Context, filter storage.MarketOpsRiskRewardSnapshotFilter) ([]storage.MarketOpsRiskRewardSnapshotRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsRiskRewardSnapshotSelect+`
WHERE tenant_id=$1 AND ($2='' OR symbol=$2)
 AND (cardinality($3::text[]) = 0 OR symbol = ANY($3))
 AND ($4::timestamptz IS NULL OR session_date >= $4::date)
 AND ($5::timestamptz IS NULL OR session_date <= $5::date)
 AND (NOT $6 OR eligible=true)
ORDER BY session_date DESC, eligible DESC, usable_input_count DESC, created_at DESC
LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), pqArray(filter.Symbols), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), filter.EligibleOnly, riskRewardSnapshotLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list risk/reward snapshots: %w", err)
	}
	defer rows.Close()
	items := []storage.MarketOpsRiskRewardSnapshotRecord{}
	for rows.Next() {
		var item storage.MarketOpsRiskRewardSnapshotRecord
		if err := rows.Scan(&item.SnapshotID, &item.TenantID, &item.AlgorithmResultID, &item.ExecutionRequestID, &item.Symbol, &item.SessionDate, &item.ObservedAt, &item.TechnicalScore, &item.TechnicalDirection, &item.RiskLevel, &item.Confidence, &item.UsableInputCount, &item.RequiredInputCount, &item.Eligible, &item.ResultPayloadJSON, &item.InputSnapshotJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan risk/reward snapshot: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list risk/reward snapshot rows: %w", err)
	}
	return items, nil
}

func riskRewardSnapshotPayload(record storage.MarketOpsRiskRewardSnapshotRecord) map[string]any {
	payload := map[string]any{}
	_ = json.Unmarshal(record.ResultPayloadJSON, &payload)
	return payload
}
