package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// SearchSubscriberCatalog invokes the bounded database projection rather than
// granting the subscriber gateway direct access to platform catalog tables.
func (r *Repository) SearchSubscriberCatalog(ctx context.Context, tenantID, query string, limit int) ([]storage.SubscriberCatalogProjectionRecord, error) {
	records := []storage.SubscriberCatalogProjectionRecord{}
	tenantID = strings.TrimSpace(tenantID)
	if !validSubscriberTenantID(tenantID) {
		return nil, fmt.Errorf("invalid subscriber tenant")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	err := r.WithSubscriberTenantScope(ctx, tenantID, func(ctx context.Context, tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT global_asset_id,ticker,company_name,asset_type,exchange,sector,eligibility_status,coverage_state,coverage_mode FROM subscriber_search_global_catalog($1,$2)`, strings.TrimSpace(query), limit)
		if err != nil {
			return fmt.Errorf("search subscriber catalog projection: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var record storage.SubscriberCatalogProjectionRecord
			if err := rows.Scan(&record.GlobalAssetID, &record.Ticker, &record.CompanyName, &record.AssetType, &record.Exchange, &record.Sector, &record.EligibilityStatus, &record.CoverageState, &record.CoverageMode); err != nil {
				return fmt.Errorf("scan subscriber catalog projection: %w", err)
			}
			records = append(records, record)
		}
		return rows.Err()
	})
	return records, err
}
