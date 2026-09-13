package postgres

import (
	"context"
	"fmt"

	"github.com/lukebabs/signalops/internal/storage"
)

// ListSubscriberGlobalReferenceCandidates returns only undispositioned entries
// in the retained current ranking, ordered by source rank. This lets admission
// fill a 1,000-member qualified cohort without admitting alphabetical catalog
// entries ahead of higher-ranked candidates.
func (r *Repository) ListSubscriberGlobalReferenceCandidates(ctx context.Context, limit int) ([]storage.SubscriberGlobalReferenceCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT asset.global_asset_id, asset.source_id, asset.provider_symbol, asset.canonical_symbol
FROM subscriber_global_ranking_snapshots snapshot
JOIN subscriber_global_ranking_snapshot_entries entry ON entry.ranking_snapshot_id=snapshot.ranking_snapshot_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=entry.global_asset_id
WHERE snapshot.is_current
  AND asset.eligibility_status='discovered'
  AND NOT EXISTS (SELECT 1 FROM subscriber_global_asset_eligibility_decisions decision WHERE decision.global_asset_id=asset.global_asset_id AND decision.decision='deferred')
ORDER BY entry.source_rank, entry.selection_rank, asset.global_asset_id
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
