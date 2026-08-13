package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerSubscriberCatalogRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/catalog", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		if _, enabled := cfg.SubscriberListsPilotTenants[tenantID]; !enabled {
			writeError(w, http.StatusNotFound, "subscriber_lists_not_enabled", "subscriber lists are not enabled for this tenant")
			return
		}
		if _, ok := requireRequestSubject(w, r, ""); !ok {
			return
		}
		if cfg.SubscriberCatalogRepository == nil || cfg.SubscriberEntitlementRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "subscriber catalog storage is unavailable")
			return
		}
		entitlement, err := cfg.SubscriberEntitlementRepository.GetSubscriberEntitlement(r.Context(), tenantID)
		if errors.Is(err, storage.ErrNotFound) || err == nil && !subscriberCatalogSearchEnabled(entitlement) {
			writeError(w, http.StatusForbidden, "subscriber_catalog_not_entitled", "tenant is not entitled to catalog search")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "subscriber catalog entitlement lookup failed")
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) > 120 {
			writeError(w, http.StatusBadRequest, "invalid_subscriber_catalog_query", "catalog query must be at most 120 characters")
			return
		}
		records, err := cfg.SubscriberCatalogRepository.SearchSubscriberCatalog(r.Context(), tenantID, query, queryLimit(r, 20))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "subscriber_catalog_query_failed", "subscriber catalog query failed")
			return
		}
		result := make([]map[string]any, 0, len(records))
		for _, record := range records {
			result = append(result, map[string]any{"global_asset_id": record.GlobalAssetID, "ticker": record.Ticker, "company_name": record.CompanyName, "asset_type": record.AssetType, "exchange": record.Exchange, "sector": record.Sector, "eligibility_status": record.EligibilityStatus, "coverage_state": record.CoverageState, "coverage_mode": record.CoverageMode})
		}
		writeJSON(w, http.StatusOK, map[string]any{"assets": result})
	})
}

func subscriberCatalogSearchEnabled(entitlement storage.SubscriberEntitlementRecord) bool {
	if entitlement.Status != storage.SubscriberEntitlementActive {
		return false
	}
	for _, capability := range entitlement.Capabilities {
		if capability.Capability == "catalog_search" && capability.Enabled && capability.QuotaLimit > 0 {
			return true
		}
	}
	return false
}
