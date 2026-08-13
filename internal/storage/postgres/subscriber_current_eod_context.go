package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// GetSubscriberCurrentEODContext joins the gateway-safe global projection to
// the tenant's active MarketOps universe. The gateway never queries global EOD
// tables directly and cannot read a symbol outside that tenant's universe.
func (r *Repository) GetSubscriberCurrentEODContext(ctx context.Context, tenantID, symbol string) (storage.SubscriberCurrentEODContextRecord, error) {
	var record storage.SubscriberCurrentEODContextRecord
	err := r.db.QueryRowContext(ctx, `
SELECT current.global_asset_id, current.canonical_symbol, current.session_date,
  current.open, current.high, current.low, current.close, current.volume, current.vwap,
  current.provider, current.selected_observation_role, current.selection_policy_version,
  current.payload_fingerprint, current.source_event_id, current.source_run_id,
  current.algorithm_version, current.quality_state, current.as_of_time
FROM marketops_universal_assets asset
JOIN subscriber_gateway_current_eod_context current ON current.canonical_symbol = upper(asset.ticker)
WHERE asset.tenant_id=$1 AND asset.is_active=true AND upper(asset.ticker)=upper($2)
ORDER BY current.session_date DESC, current.as_of_time DESC
LIMIT 1`, strings.TrimSpace(tenantID), strings.TrimSpace(symbol)).Scan(
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
