package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) AddSubscriberPrivateCatalogMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) (result storage.SubscriberCatalogMembershipResult, err error) {
	if err = normalizeSubscriberWatchlistMembership(&request); err != nil {
		return result, err
	}
	err = r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT tenant_id,list_id,global_asset_id,added_by_subject,added_at,activation_state FROM subscriber_add_private_catalog_membership($1,$2,$3,$4)`, request.ActorSubject, request.ListID, request.GlobalAssetID, request.CorrelationID).Scan(&result.Membership.TenantID, &result.Membership.ListID, &result.Membership.GlobalAssetID, &result.Membership.AddedBySubject, &result.Membership.AddedAt, &result.ActivationState)
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("add private catalog membership: %w", err)
		}
		return nil
	})
	return result, err
}
