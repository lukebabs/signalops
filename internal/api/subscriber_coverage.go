package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerSubscriberCoverageRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/coverage-manifest", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		if cfg.SubscriberWatchlistRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_lists_unavailable", "subscriber list storage is unavailable")
			return
		}
		lists, err := cfg.SubscriberWatchlistRepository.ListSubscriberWatchlists(r.Context(), tenant, subject)
		if err != nil {
			writeQueryError(w, err, "subscriber_list_not_found", "subscriber default list was not found")
			return
		}
		var defaultList storage.SubscriberWatchlistRecord
		found := false
		for _, list := range lists {
			if list.ListKind == storage.SubscriberWatchlistKindTenantDefault {
				defaultList, found = list, true
				break
			}
		}
		if !found {
			writeError(w, http.StatusNotFound, "subscriber_default_list_not_found", "tenant default list was not found")
			return
		}
		items, err := cfg.SubscriberWatchlistRepository.ListSubscriberWatchlistItems(r.Context(), tenant, subject, defaultList.ListID)
		if err != nil {
			writeQueryError(w, err, "subscriber_list_not_found", "tenant default list was not found")
			return
		}
		assets := make([]map[string]any, 0, len(items))
		for _, item := range items {
			assets = append(assets, map[string]any{"tenant_id": item.TenantID, "list_id": item.ListID, "list_kind": item.ListKind, "list_name": item.ListName, "global_asset_id": item.GlobalAssetID, "ticker": item.Ticker, "company_name": item.CompanyName, "asset_type": item.AssetType, "exchange": item.Exchange, "sector": item.Sector, "eligibility_status": item.EligibilityStatus, "coverage_state": item.CoverageState, "coverage_mode": item.CoverageMode, "coverage_tier": item.CoverageMode, "source": "tenant_default", "added_at": item.AddedAt})
		}
		payload := map[string]any{"tenant_id": tenant, "manifest_schema": "subscriber.coverage-manifest.v1", "list_id": defaultList.ListID, "list_name": defaultList.ListName, "generated_at": nowUTC(), "asset_count": len(assets), "assets": assets}
		fingerprint, _ := json.Marshal(map[string]any{"tenant_id": tenant, "manifest_schema": "subscriber.coverage-manifest.v1", "list_id": defaultList.ListID, "list_name": defaultList.ListName, "asset_count": len(assets), "assets": assets})
		sum := sha256.Sum256(fingerprint)
		etag := `"` + hex.EncodeToString(sum[:]) + `"`
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "private, max-age=30")
		if strings.TrimSpace(r.Header.Get("If-None-Match")) == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscriber/admin/warm-catalog", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok || !requireTenantAdministrator(w, r) {
			return
		}
		repo := cfg.SubscriberCoverageRepository
		if repo == nil {
			repo, ok = cfg.SubscriberCatalogRepository.(storage.SubscriberCoverageRepository)
		}
		if repo == nil {
			repo, ok = cfg.SubscriberCatalogMembershipRepository.(storage.SubscriberCoverageRepository)
		}
		if repo == nil {
			ok = false
		}
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "warm catalog administration is unavailable")
			return
		}
		listID := strings.TrimSpace(r.URL.Query().Get("list_id"))
		if listID == "" {
			writeError(w, http.StatusBadRequest, "list_id_required", "list_id is required")
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		rows, err := repo.ListSubscriberAdminWarmCatalog(r.Context(), tenant, listID, r.URL.Query().Get("q"), r.URL.Query().Get("preset"), limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "subscriber_catalog_query_failed", "warm catalog query failed")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			out = append(out, map[string]any{"global_asset_id": row.GlobalAssetID, "ticker": row.Ticker, "company_name": row.CompanyName, "asset_type": row.AssetType, "exchange": row.Exchange, "sector": row.Sector, "eligibility_status": row.EligibilityStatus, "coverage_state": row.CoverageState, "coverage_tier": row.CoverageMode, "warm_rank": row.WarmRank, "market_cap_rank": row.MarketCapRank, "tenant_default_member": row.TenantDefaultMember, "legacy_protected": row.LegacyProtected})
		}
		writeJSON(w, http.StatusOK, map[string]any{"assets": out, "limit": limit, "offset": offset, "preset": r.URL.Query().Get("preset")})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscriber/admin/tenant-default-catalog-memberships", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok || !requireTenantAdministrator(w, r) {
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		repo := cfg.SubscriberCoverageRepository
		if repo == nil {
			repo, ok = cfg.SubscriberCatalogMembershipRepository.(storage.SubscriberCoverageRepository)
		}
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "tenant default catalog activation is unavailable")
			return
		}
		var request struct {
			ListID         string   `json:"list_id"`
			GlobalAssetIDs []string `json:"global_asset_ids"`
			Preset         string   `json:"preset"`
			CorrelationID  string   `json:"correlation_id"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
			return
		}
		if strings.TrimSpace(request.ListID) == "" {
			writeError(w, http.StatusBadRequest, "list_id_required", "list_id is required")
			return
		}
		if len(request.GlobalAssetIDs) == 0 {
			writeError(w, http.StatusBadRequest, "assets_required", "at least one global asset is required")
			return
		}
		if len(request.GlobalAssetIDs) > 400 {
			writeError(w, http.StatusBadRequest, "too_many_assets", "a maximum of 400 assets may be selected per request")
			return
		}
		results := make([]map[string]any, 0, len(request.GlobalAssetIDs))
		for _, id := range request.GlobalAssetIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			result, err := repo.AddSubscriberTenantDefaultCatalogMembership(r.Context(), storage.SubscriberWatchlistMembershipRequest{TenantID: tenant, ListID: request.ListID, GlobalAssetID: id, ActorSubject: subject, CorrelationID: request.CorrelationID})
			if err != nil {
				writeSubscriberWatchlistMutationError(w, err)
				return
			}
			results = append(results, map[string]any{"global_asset_id": result.Membership.GlobalAssetID, "activation_state": result.ActivationState})
		}
		writeJSON(w, http.StatusOK, map[string]any{"tenant_id": tenant, "list_id": request.ListID, "added": results, "preset": request.Preset})
	})
}

func nowUTC() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00") }
