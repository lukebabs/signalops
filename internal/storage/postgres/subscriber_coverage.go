package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func (r *Repository) ListSubscriberAdminWarmCatalog(ctx context.Context, tenantID, listID, query, preset string, limit, offset int) ([]storage.SubscriberAdminWarmCatalogRecord, error) {
	if !validSubscriberTenantID(strings.TrimSpace(tenantID)) || strings.TrimSpace(listID) == "" {
		return nil, fmt.Errorf("invalid subscriber coverage scope")
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rowsOut := []storage.SubscriberAdminWarmCatalogRecord{}
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT global_asset_id,ticker,company_name,asset_type,exchange,sector,eligibility_status,coverage_state,coverage_mode,warm_rank,market_cap_rank,tenant_default_member,legacy_protected FROM subscriber_search_global_warm_catalog($1,$2,$3,$4,$5,$6)`, tenantID, listID, strings.TrimSpace(query), strings.TrimSpace(preset), limit, offset)
		if err != nil {
			return fmt.Errorf("list subscriber warm catalog: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var item storage.SubscriberAdminWarmCatalogRecord
			if err := rows.Scan(&item.GlobalAssetID, &item.Ticker, &item.CompanyName, &item.AssetType, &item.Exchange, &item.Sector, &item.EligibilityStatus, &item.CoverageState, &item.CoverageMode, &item.WarmRank, &item.MarketCapRank, &item.TenantDefaultMember, &item.LegacyProtected); err != nil {
				return fmt.Errorf("scan subscriber warm catalog: %w", err)
			}
			rowsOut = append(rowsOut, item)
		}
		return rows.Err()
	})
	return rowsOut, err
}

func (r *Repository) AddSubscriberTenantDefaultCatalogMembership(ctx context.Context, request storage.SubscriberWatchlistMembershipRequest) (result storage.SubscriberCatalogMembershipResult, err error) {
	if err = normalizeSubscriberWatchlistMembership(&request); err != nil {
		return result, err
	}
	err = r.WithSubscriberTenantScope(ctx, request.TenantID, func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `SELECT tenant_id,list_id,global_asset_id,added_by_subject,added_at,activation_state FROM subscriber_add_tenant_default_catalog_membership($1,$2,$3,$4)`, request.ActorSubject, request.ListID, request.GlobalAssetID, request.CorrelationID).Scan(&result.Membership.TenantID, &result.Membership.ListID, &result.Membership.GlobalAssetID, &result.Membership.AddedBySubject, &result.Membership.AddedAt, &result.ActivationState)
	})
	if err == sql.ErrNoRows {
		return result, storage.ErrNotFound
	}
	if err != nil {
		return result, fmt.Errorf("add tenant default catalog membership: %w", err)
	}
	return result, nil
}
