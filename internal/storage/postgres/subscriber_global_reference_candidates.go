package postgres

import (
	"context"
	"fmt"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalReferenceCandidates exposes only platform-owned discovered
// identities to the catalog-sync workload; no browser projection is involved.
func (r *Repository) ListSubscriberGlobalReferenceCandidates(ctx context.Context, limit int) ([]storage.SubscriberGlobalReferenceCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT global_asset_id, source_id, provider_symbol, canonical_symbol
FROM subscriber_global_assets
WHERE eligibility_status='discovered'
ORDER BY source_id, provider_symbol, global_asset_id
LIMIT $1`, globalReferenceCandidateLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list global reference candidates: %w", err)
	}
	defer rows.Close()
	records := []storage.SubscriberGlobalReferenceCandidate{}
	for rows.Next() {
		var record storage.SubscriberGlobalReferenceCandidate
		if err := rows.Scan(&record.GlobalAssetID, &record.SourceID, &record.ProviderSymbol, &record.CanonicalSymbol); err != nil {
			return nil, fmt.Errorf("scan global reference candidate: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func globalReferenceCandidateLimit(limit int) int {
	if limit <= 0 || limit > 1000 {
		return 1000
	}
	return limit
}
