package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListMarketOpsEODEvents reads persisted EOD history for the requested assets.
// It must not use a global event cap: heavily backfilled symbols would otherwise
// crowd newer universe members out of the history window.
func (r *Repository) ListMarketOpsEODEvents(ctx context.Context, tenantID string, symbols []string) ([]storage.NormalizedEventLedgerRecord, error) {
	keys := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if key := strings.ToUpper(strings.TrimSpace(symbol)); key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return []storage.NormalizedEventLedgerRecord{}, nil
	}
	rows, err := r.temporal().QueryContext(ctx, normalizedEventSelect+` WHERE tenant_id=$1 AND app_id='marketops' AND dataset='equity_eod_prices' AND UPPER(normalized_payload::jsonb->>'symbol') = ANY(string_to_array($2, ',')) ORDER BY created_at ASC`, strings.TrimSpace(tenantID), strings.Join(keys, ","))
	if err != nil {
		return nil, fmt.Errorf("list marketops EOD history: %w", err)
	}
	defer rows.Close()
	out := make([]storage.NormalizedEventLedgerRecord, 0)
	for rows.Next() {
		x, err := scanNormalizedEventLedger(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marketops EOD history: %w", err)
	}
	return out, nil
}
