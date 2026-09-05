package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberEODRevisionDeltas joins the gateway-safe review projection to
// an active tenant asset. It cannot return a global revision row for another
// tenant's inactive or absent symbol.
func (r *Repository) ListSubscriberEODRevisionDeltas(ctx context.Context, tenantID, symbol string, limit int) ([]storage.SubscriberEODRevisionDeltaRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT review.canonical_symbol, review.session_date, review.field_name,
  review.initial_value, review.revised_value, review.delta_class, review.materiality,
  review.initial_observed_at, review.revised_observed_at,
  review.initial_source_event_id, review.revised_source_event_id,
  review.initial_source_run_id, review.revised_source_run_id,
  review.initial_payload_fingerprint, review.revised_payload_fingerprint,
  review.initial_algorithm_version, review.revised_algorithm_version
FROM marketops_universal_assets asset
JOIN subscriber_gateway_eod_revision_review review ON review.canonical_symbol = upper(asset.ticker)
WHERE asset.tenant_id=$1 AND asset.is_active=true AND upper(asset.ticker)=upper($2)
ORDER BY review.session_date DESC, review.materiality DESC, review.field_name ASC
LIMIT $3`, strings.TrimSpace(tenantID), strings.TrimSpace(symbol), limit)
	if err != nil {
		return nil, fmt.Errorf("list subscriber EOD revision deltas: %w", err)
	}
	defer rows.Close()
	results := []storage.SubscriberEODRevisionDeltaRecord{}
	for rows.Next() {
		var record storage.SubscriberEODRevisionDeltaRecord
		if err := rows.Scan(
			&record.Symbol, &record.SessionDate, &record.FieldName,
			&record.InitialValue, &record.RevisedValue, &record.DeltaClass, &record.Materiality,
			&record.InitialObservedAt, &record.RevisedObservedAt,
			&record.InitialSourceEventID, &record.RevisedSourceEventID,
			&record.InitialSourceRunID, &record.RevisedSourceRunID,
			&record.InitialPayloadFingerprint, &record.RevisedPayloadFingerprint,
			&record.InitialAlgorithmVersion, &record.RevisedAlgorithmVersion,
		); err != nil {
			return nil, fmt.Errorf("scan subscriber EOD revision delta: %w", err)
		}
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriber EOD revision deltas: %w", err)
	}
	return results, nil
}
