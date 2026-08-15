package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalMarketOpsMarketStates reads only the platform-owned
// Market State projection. The API authorizes symbols through watchlists.
func (r *Repository) ListSubscriberGlobalMarketOpsMarketStates(ctx context.Context, symbols []string, filter storage.MarketOpsMarketStateFilter) ([]storage.MarketOpsMarketStateRecord, error) {
	if len(symbols) == 0 {
		return []storage.MarketOpsMarketStateRecord{}, nil
	}
	rows, err := r.db.QueryContext(ctx, "SELECT market_state_id,global_asset_id,symbol,session_date,as_of_time,state_schema_version,COALESCE(payload->\x27state_payload\x27,jsonb_build_object())::text,COALESCE(payload->\x27feature_observation_ids\x27,jsonb_build_array())::text,COALESCE((payload->>\x27feature_count\x27)::integer,0),COALESCE((payload->>\x27required_feature_count\x27)::integer,0),COALESCE((payload->>\x27completeness_ratio\x27)::double precision,0),quality_state,NULLIF(payload->>\x27quality_score\x27,\x27\x27)::double precision,COALESCE(payload->\x27quality_summary\x27,jsonb_build_object())::text,COALESCE(payload->\x27eligible_hypotheses\x27,jsonb_build_array())::text,COALESCE(payload->>\x27build_run_id\x27,\x27\x27),evidence_fingerprint,as_of_time FROM subscriber_gateway_global_market_states WHERE upper(symbol)=ANY($1) AND ($2::timestamptz IS NULL OR session_date >= $2::date) AND ($3::timestamptz IS NULL OR session_date <= $3::date) AND ($4=\x27\x27 OR state_schema_version=$4) AND ($5=\x27\x27 OR quality_state=$5) ORDER BY session_date DESC,as_of_time DESC LIMIT $6", pqArray(symbols), nullTime(filter.SessionStart), nullTime(filter.SessionEnd), strings.TrimSpace(filter.StateSchemaVersion), strings.TrimSpace(filter.QualityState), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global market states: %w", err)
	}
	defer rows.Close()
	items := []storage.MarketOpsMarketStateRecord{}
	for rows.Next() {
		var item storage.MarketOpsMarketStateRecord
		var observationIDs, hypotheses string
		var qualityScore sql.NullFloat64
		if err := rows.Scan(&item.MarketStateID, &item.AssetID, &item.Symbol, &item.SessionDate, &item.AsOfTime, &item.StateSchemaVersion, &item.StatePayloadJSON, &observationIDs, &item.FeatureCount, &item.RequiredFeatureCount, &item.CompletenessRatio, &item.QualityState, &qualityScore, &item.QualitySummaryJSON, &hypotheses, &item.BuildRunID, &item.DeterministicKey, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber global market state: %w", err)
		}
		if err := json.Unmarshal([]byte(observationIDs), &item.FeatureObservationIDs); err != nil {
			return nil, fmt.Errorf("scan subscriber global market state feature ids: %w", err)
		}
		if err := json.Unmarshal([]byte(hypotheses), &item.EligibleHypotheses); err != nil {
			return nil, fmt.Errorf("scan subscriber global market state hypotheses: %w", err)
		}
		item.TenantID = "platform-global"
		item.AppID = "marketops"
		if qualityScore.Valid {
			value := qualityScore.Float64
			item.QualityScore = &value
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
