package api

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"time"
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
		reservation, decision, err := cfg.SubscriberEntitlementRepository.ReserveSubscriberQuota(r.Context(), storage.SubscriberQuotaReservationRequest{
			TenantID: tenant, Subject: subject, Capability: "eod_activation", RequestedUnits: 1,
			IdempotencyKey: subscriberEODActivationIdempotencyKey(tenant, subject, r.PathValue("list_id"), request.GlobalAssetID),
			CorrelationID:  request.CorrelationID, RequestedAt: time.Now().UTC(),
		})
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_activation_unavailable", "subscriber activation quota storage is unavailable")
			return
		}
		if decision.DecisionReason != "allowed" || reservation.ReservationID == "" {
			writeError(w, http.StatusTooManyRequests, "subscriber_activation_quota_exhausted", "tenant activation quota is exhausted")
			return
		}
		result, err := store.AddSubscriberPrivateCatalogMembership(r.Context(), storage.SubscriberWatchlistMembershipRequest{TenantID: tenant, ListID: r.PathValue("list_id"), GlobalAssetID: request.GlobalAssetID, ActorSubject: subject, CorrelationID: request.CorrelationID})
		if err != nil {
			_, _ = cfg.SubscriberEntitlementRepository.FinalizeSubscriberQuotaReservation(r.Context(), storage.SubscriberQuotaReservationLifecycleRequest{TenantID: tenant, ReservationID: reservation.ReservationID, ActorSubject: subject, Transition: storage.SubscriberQuotaReleased, CorrelationID: request.CorrelationID, OccurredAt: time.Now().UTC()})
			writeSubscriberWatchlistMutationError(w, err)
			return
		}
		transition := storage.SubscriberQuotaReleased
		if result.ActivationState == "queued" {
			transition = storage.SubscriberQuotaConsumed
		}
		if _, err := cfg.SubscriberEntitlementRepository.FinalizeSubscriberQuotaReservation(r.Context(), storage.SubscriberQuotaReservationLifecycleRequest{TenantID: tenant, ReservationID: reservation.ReservationID, ActorSubject: subject, Transition: transition, CorrelationID: request.CorrelationID, OccurredAt: time.Now().UTC()}); err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscriber_activation_unavailable", "subscriber activation quota finalization is unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"membership": subscriberWatchlistMembershipResponse(result.Membership), "activation_state": result.ActivationState})
	})
}

func subscriberEODActivationIdempotencyKey(tenant, subject, listID, assetID string) string {
	sum := sha256.Sum256([]byte(tenant + "\x00" + subject + "\x00" + listID + "\x00" + assetID))
	return "subscriber-eod-activation-v1:" + hex.EncodeToString(sum[:])
}
