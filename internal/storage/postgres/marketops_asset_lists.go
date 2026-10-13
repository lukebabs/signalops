package postgres

import (
	"context"
	"fmt"
	"strings"
)

// currentMarketOpsAssetList supplies a durable list reference to legacy callers
// that still submit symbol arrays. The symbol array remains a display snapshot;
// the list revision is the execution contract.
func (r *Repository) currentMarketOpsAssetList(ctx context.Context, tenantID string) (string, int64, string, error) {
	var id, fingerprint string
	var revision int64
	err := r.db.QueryRowContext(ctx, `
SELECT l.list_id, l.current_revision, COALESCE(r.membership_fingerprint, '')
FROM marketops_asset_lists l
LEFT JOIN marketops_asset_list_revisions r
  ON r.list_id=l.list_id AND r.revision=l.current_revision
WHERE l.tenant_id=$1 AND l.status='active'
ORDER BY l.updated_at DESC LIMIT 1`, strings.TrimSpace(tenantID)).Scan(&id, &revision, &fingerprint)
	if err != nil {
		return "", 0, "", fmt.Errorf("resolve current marketops asset list: %w", err)
	}
	return id, revision, fingerprint, nil
}
