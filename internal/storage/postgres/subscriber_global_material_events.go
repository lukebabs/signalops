package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ResolveSubscriberGlobalCanonicalAssetID(ctx context.Context, symbol string) (string, error) {
	var globalAssetID string
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(resolution.canonical_global_asset_id, link.global_asset_id)
FROM subscriber_global_asset_source_links link
LEFT JOIN subscriber_global_asset_identity_resolutions resolution ON resolution.source_global_asset_id = link.global_asset_id
WHERE link.source_tenant_id = 'tenant-local'
  AND link.source_is_active
  AND upper(link.source_ticker) = upper($1)
ORDER BY link.global_asset_id
LIMIT 1`, strings.TrimSpace(symbol)).Scan(&globalAssetID)
	if err != nil {
		return "", fmt.Errorf("resolve subscriber global canonical asset %s: %w", strings.ToUpper(strings.TrimSpace(symbol)), err)
	}
	return globalAssetID, nil
}

// ListSubscriberGlobalMaterialEvents reads only central provider-captured
// events. The Gateway supplies watchlist-authorized canonical symbols.
func (r *Repository) ListSubscriberGlobalMaterialEvents(ctx context.Context, symbols []string, from time.Time, limit int) ([]storage.SubscriberGlobalMaterialEventRecord, error) {
	if len(symbols) == 0 {
		return []storage.SubscriberGlobalMaterialEventRecord{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT global_asset_id, symbol, event_id, session_date, observed_at, quality_state, payload::text
FROM subscriber_gateway_global_material_events
WHERE upper(symbol) = ANY($1)
  AND ($2::date IS NULL OR (payload->>'event_date')::date >= $2::date)
ORDER BY (payload->>'event_date')::date ASC, symbol ASC, observed_at DESC
LIMIT $3`, pqArray(symbols), nullTime(from), valuationLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list subscriber global material events: %w", err)
	}
	defer rows.Close()
	out := []storage.SubscriberGlobalMaterialEventRecord{}
	for rows.Next() {
		var item storage.SubscriberGlobalMaterialEventRecord
		if err := rows.Scan(&item.GlobalAssetID, &item.Symbol, &item.EventID, &item.SessionDate, &item.ObservedAt, &item.QualityState, &item.PayloadJSON); err != nil {
			return nil, fmt.Errorf("scan subscriber global material event: %w", err)
		}
		item.Symbol = strings.ToUpper(item.Symbol)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber global material events: %w", err)
	}
	return out, nil
}
