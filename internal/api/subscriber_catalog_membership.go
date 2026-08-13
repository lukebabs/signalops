package api

import (
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
)

func registerSubscriberCatalogMembershipRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/catalog-memberships", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		if _, ok := cfg.SubscriberListsPilotTenants[tenant]; !ok {
			writeError(w, http.StatusNotFound, "subscriber_lists_not_enabled", "subscriber lists are not enabled for this tenant")
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		if cfg.SubscriberEntitlementRepository == nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "subscriber entitlement storage is unavailable")
			return
		}
		entitlement, err := cfg.SubscriberEntitlementRepository.GetSubscriberEntitlement(r.Context(), tenant)
		if err != nil || !subscriberCapabilityEnabled(entitlement, "eod_activation") {
			writeError(w, http.StatusForbidden, "subscriber_activation_not_entitled", "tenant is not entitled to catalog activation")
			return
		}
		store, ok := cfg.SubscriberCatalogMembershipRepository.(storage.SubscriberCatalogMembershipRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "subscriber_catalog_unavailable", "subscriber catalog membership storage is unavailable")
			return
		}
		request, ok := readSubscriberWatchlistRequest(w, r)
		if !ok {
			return
		}
		result, err := store.AddSubscriberPrivateCatalogMembership(r.Context(), storage.SubscriberWatchlistMembershipRequest{TenantID: tenant, ListID: r.PathValue("list_id"), GlobalAssetID: request.GlobalAssetID, ActorSubject: subject, CorrelationID: request.CorrelationID})
		if err != nil {
			writeSubscriberWatchlistMutationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"membership": subscriberWatchlistMembershipResponse(result.Membership), "activation_state": result.ActivationState})
	})
}
