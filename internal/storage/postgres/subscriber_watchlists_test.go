package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSubscriberWatchlistValidation(t *testing.T) {
	if err := normalizeSubscriberWatchlistCreate(&storage.SubscriberWatchlistCreateRequest{TenantID: "tenant-test", ListName: "My list", ActorSubject: "subject-a"}, storage.SubscriberWatchlistKindPrivate); err != nil {
		t.Fatalf("valid private list rejected: %v", err)
	}
	if err := normalizeSubscriberWatchlistCreate(&storage.SubscriberWatchlistCreateRequest{TenantID: "tenant test", ListName: "My list", ActorSubject: "subject-a"}, storage.SubscriberWatchlistKindPrivate); err == nil {
		t.Fatal("invalid tenant accepted")
	}
	if err := normalizeSubscriberWatchlistMembership(&storage.SubscriberWatchlistMembershipRequest{TenantID: "tenant-test", ListID: "list-a", GlobalAssetID: "asset-a", ActorSubject: "subject-a"}); err != nil {
		t.Fatalf("valid membership rejected: %v", err)
	}
}

func TestSubscriberWatchlistsAgainstPostgres(t *testing.T) {
	repo := subscriberIntegrationRepository(t)
	defer repo.Close()
	ctx := context.Background()
	tenantID, listID, defaultID, assetID := "subscriber-watchlist-it", "sublist-it-private", "sublist-it-default", "subglobal-watchlist-it"
	defer func() {
		for _, statement := range []string{
			`DELETE FROM subscriber_watchlist_audit WHERE tenant_id=$1`,
			`DELETE FROM subscriber_watchlist_memberships WHERE tenant_id=$1`,
			`DELETE FROM subscriber_watchlists WHERE tenant_id=$1`,
		} {
			if _, err := repo.db.ExecContext(ctx, statement, tenantID); err != nil {
				t.Errorf("cleanup subscriber watchlist test: %v", err)
			}
		}
		if _, err := repo.db.ExecContext(ctx, `DELETE FROM subscriber_global_assets WHERE global_asset_id=$1`, assetID); err != nil {
			t.Errorf("cleanup subscriber global asset: %v", err)
		}
	}()

	if _, err := repo.db.ExecContext(ctx, `INSERT INTO subscriber_global_assets
  (global_asset_id, source_id, provider_symbol, canonical_symbol, eligibility_status)
VALUES ($1,'subscriber-watchlist-test','S3TEST','S3TEST','eligible')
ON CONFLICT (global_asset_id) DO NOTHING`, assetID); err != nil {
		t.Fatalf("seed global asset: %v", err)
	}

	privateList, err := repo.CreateSubscriberPrivateWatchlist(ctx, storage.SubscriberWatchlistCreateRequest{
		ListID: listID, TenantID: tenantID, ListName: "Private", ActorSubject: "subject-a", CorrelationID: "watchlist-test",
	})
	if err != nil {
		t.Fatalf("create private list: %v", err)
	}
	if privateList.OwnerSubject != "subject-a" || privateList.ListKind != storage.SubscriberWatchlistKindPrivate {
		t.Fatalf("private list=%+v", privateList)
	}
	if _, err := repo.CreateSubscriberTenantDefaultWatchlist(ctx, storage.SubscriberWatchlistCreateRequest{
		ListID: defaultID, TenantID: tenantID, ListName: "Default", ActorSubject: "admin-a",
	}); err != nil {
		t.Fatalf("create default list: %v", err)
	}
	addRequest := storage.SubscriberWatchlistMembershipRequest{
		TenantID: tenantID, ListID: listID, GlobalAssetID: assetID, ActorSubject: "subject-a",
	}
	if _, err := repo.AddSubscriberPrivateWatchlistMembership(ctx, addRequest); err != nil {
		t.Fatalf("add private membership: %v", err)
	}
	if _, err := repo.AddSubscriberPrivateWatchlistMembership(ctx, addRequest); err != nil {
		t.Fatalf("repeat private membership: %v", err)
	}
	var auditCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT count(*) FROM subscriber_watchlist_audit WHERE tenant_id=$1 AND list_id=$2`, tenantID, listID).Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("audit count after idempotent add=%d err=%v", auditCount, err)
	}
	if _, err := repo.AddSubscriberPrivateWatchlistMembership(ctx, storage.SubscriberWatchlistMembershipRequest{
		TenantID: tenantID, ListID: listID, GlobalAssetID: assetID, ActorSubject: "subject-b",
	}); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("foreign private mutation err=%v, want not found", err)
	}

	visibleA, err := repo.ListSubscriberWatchlists(ctx, tenantID, "subject-a")
	if err != nil || len(visibleA) != 2 {
		t.Fatalf("subject-a visible=%+v err=%v", visibleA, err)
	}
	visibleB, err := repo.ListSubscriberWatchlists(ctx, tenantID, "subject-b")
	if err != nil || len(visibleB) != 1 || visibleB[0].ListID != defaultID {
		t.Fatalf("subject-b visible=%+v err=%v", visibleB, err)
	}
	members, err := repo.ListSubscriberWatchlistMemberships(ctx, tenantID, "subject-a", listID)
	if err != nil || len(members) != 1 || members[0].GlobalAssetID != assetID {
		t.Fatalf("private members=%+v err=%v", members, err)
	}
	if _, err := repo.ListSubscriberWatchlistMemberships(ctx, tenantID, "subject-b", listID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("foreign private read err=%v, want not found", err)
	}
	if err := repo.RemoveSubscriberPrivateWatchlistMembership(ctx, addRequest); err != nil {
		t.Fatalf("remove private membership: %v", err)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT count(*) FROM subscriber_watchlist_audit WHERE tenant_id=$1 AND list_id=$2`, tenantID, listID).Scan(&auditCount); err != nil || auditCount != 3 {
		t.Fatalf("audit count after removal=%d err=%v", auditCount, err)
	}
}
