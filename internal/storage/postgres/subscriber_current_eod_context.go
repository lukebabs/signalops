package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// GetSubscriberCurrentEODContext reads the narrow gateway-safe global EOD
// projection. Its callers must first authorize the symbol through a tenant
// watchlist or the legacy tenant-universe path; this method never authorizes
// arbitrary catalog access on its own.
func (r *Repository) GetSubscriberCurrentEODContext(ctx context.Context, tenantID, symbol string) (storage.SubscriberCurrentEODContextRecord, error) {
	var record storage.SubscriberCurrentEODContextRecord
	err := r.db.QueryRowContext(ctx, `
SELECT current.global_asset_id, current.canonical_symbol, current.session_date,
  current.open, current.high, current.low, current.close, current.volume, current.vwap,
  current.provider, current.selected_observation_role, current.selection_policy_version,
  current.payload_fingerprint, current.source_event_id, current.source_run_id,
  current.algorithm_version, current.quality_state, current.as_of_time
FROM subscriber_gateway_current_eod_context current
WHERE upper(current.canonical_symbol)=upper($1)
ORDER BY current.session_date DESC, current.as_of_time DESC
LIMIT 1`, strings.TrimSpace(symbol)).Scan(
		&record.GlobalAssetID, &record.Symbol, &record.SessionDate,
		&record.Open, &record.High, &record.Low, &record.Close, &record.Volume, &record.VWAP,
		&record.Provider, &record.SelectedObservationRole, &record.SelectionPolicyVersion,
		&record.PayloadFingerprint, &record.SourceEventID, &record.SourceRunID,
		&record.AlgorithmVersion, &record.QualityState, &record.AsOfTime,
	)
	if err == sql.ErrNoRows {
		return storage.SubscriberCurrentEODContextRecord{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.SubscriberCurrentEODContextRecord{}, fmt.Errorf("get subscriber current EOD context: %w", err)
	}
	return record, nil
}

// ListSubscriberCurrentEODContexts reads the latest gateway-safe global EOD
// projection for each supplied, already-authorized symbol in one query.
func (r *Repository) ListSubscriberCurrentEODContexts(ctx context.Context, symbols []string) ([]storage.SubscriberCurrentEODContextRecord, error) {
	normalized := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		normalized = append(normalized, symbol)
	}
	if len(normalized) == 0 {
		return []storage.SubscriberCurrentEODContextRecord{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (upper(current.canonical_symbol))
  current.global_asset_id, current.canonical_symbol, current.session_date,
  current.open, current.high, current.low, current.close, current.volume, current.vwap,
  current.provider, current.selected_observation_role, current.selection_policy_version,
  current.payload_fingerprint, current.source_event_id, current.source_run_id,
  current.algorithm_version, current.quality_state, current.as_of_time
FROM subscriber_gateway_current_eod_context current
WHERE upper(current.canonical_symbol) = ANY($1)
ORDER BY upper(current.canonical_symbol), current.session_date DESC, current.as_of_time DESC`, pqArray(normalized))
	if err != nil {
		return nil, fmt.Errorf("list subscriber current EOD contexts: %w", err)
	}
	defer rows.Close()
	items := make([]storage.SubscriberCurrentEODContextRecord, 0, len(normalized))
	for rows.Next() {
		var record storage.SubscriberCurrentEODContextRecord
		if err := rows.Scan(
			&record.GlobalAssetID, &record.Symbol, &record.SessionDate,
			&record.Open, &record.High, &record.Low, &record.Close, &record.Volume, &record.VWAP,
			&record.Provider, &record.SelectedObservationRole, &record.SelectionPolicyVersion,
			&record.PayloadFingerprint, &record.SourceEventID, &record.SourceRunID,
			&record.AlgorithmVersion, &record.QualityState, &record.AsOfTime,
		); err != nil {
			return nil, fmt.Errorf("scan subscriber current EOD context: %w", err)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber current EOD contexts: %w", err)
	}
	return items, nil
}

// ListSubscriberGlobalRiskRewardSnapshots reads only the platform-owned
// gateway projection. The caller authorizes symbols via subscriber watchlists.
func (r *Repository) ListSubscriberGlobalRiskRewardSnapshots(ctx context.Context, symbols []string, sessionStart time.Time, limit int) ([]storage.MarketOpsRiskRewardSnapshotRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT snapshot_id,'platform-global','', '', symbol, session_date, observed_at,
  0, '', '', 0, usable_input_count, required_input_count, eligible, result_payload, '{}'::jsonb, observed_at
FROM subscriber_gateway_global_risk_reward_snapshots
WHERE upper(symbol) = ANY($1) AND session_date >= $2::date
ORDER BY session_date DESC, usable_input_count DESC, observed_at DESC
LIMIT $3`, pqArray(symbols), sessionStart.UTC(), riskRewardSnapshotLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global risk/reward snapshots: %w", err)
	}
	defer rows.Close()
	items := []storage.MarketOpsRiskRewardSnapshotRecord{}
	for rows.Next() {
		var item storage.MarketOpsRiskRewardSnapshotRecord
		if err := rows.Scan(&item.SnapshotID, &item.TenantID, &item.AlgorithmResultID, &item.ExecutionRequestID, &item.Symbol, &item.SessionDate, &item.ObservedAt, &item.TechnicalScore, &item.TechnicalDirection, &item.RiskLevel, &item.Confidence, &item.UsableInputCount, &item.RequiredInputCount, &item.Eligible, &item.ResultPayloadJSON, &item.InputSnapshotJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber global risk/reward snapshot: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
