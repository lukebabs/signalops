package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSubjectSubscriptionRequest struct {
	TenantID      string `json:"tenant_id"`
	Subject       string `json:"subject"`
	ProductKey    string `json:"product_key"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}
type subscriberTenantSubscriptionRequest struct {
	TenantID      string `json:"tenant_id"`
	ProductKey    string `json:"product_key"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}
type subscriberSubscriptionSeatRequest struct {
	TenantID      string `json:"tenant_id"`
	Subject       string `json:"subject"`
	SeatRole      string `json:"seat_role"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id"`
}

// registerSubscriberSubscriptionAdministrationRoutes provides the controlled
// provisioning boundary for launch operations. It deliberately has no browser
// self-service upgrade operation: a caller needs the dedicated platform role
// signalops:subscription_admin (or an existing platform-super-admin role).
func registerSubscriberSubscriptionAdministrationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repository := cfg.SubscriberSubscriptionAdministrationRepository
	if repository == nil {
		return
	}
	mux.HandleFunc("POST /v1/administration/subscriptions/subject", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberSubjectSubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberSubjectSubscription(r.Context(), storage.SubscriberSubjectSubscriptionMutation{
			TenantID: strings.TrimSpace(request.TenantID), Subject: strings.TrimSpace(request.Subject), ProductKey: strings.TrimSpace(request.ProductKey), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
	mux.HandleFunc("POST /v1/administration/subscriptions/tenant", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberTenantSubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberTenantSubscription(r.Context(), storage.SubscriberTenantSubscriptionMutation{
			TenantID: strings.TrimSpace(request.TenantID), ProductKey: strings.TrimSpace(request.ProductKey), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
	mux.HandleFunc("PUT /v1/administration/subscriptions/seats", func(w http.ResponseWriter, r *http.Request) {
		actor, ok := requireSubscriptionAdministrator(w, r)
		if !ok {
			return
		}
		var request subscriberSubscriptionSeatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := repository.UpsertSubscriberSubscriptionSeat(r.Context(), storage.SubscriberSubscriptionSeatMutation{
			TenantID: strings.TrimSpace(request.TenantID), Subject: strings.TrimSpace(request.Subject), SeatRole: strings.TrimSpace(request.SeatRole), Status: strings.TrimSpace(request.Status), ActorSubject: actor, CorrelationID: subscriptionCorrelationID(r, request.CorrelationID),
		}); err != nil {
			writeSubscriberSubscriptionAdministrationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "provisioned"})
	})
}

func requireSubscriptionAdministrator(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, authenticated := principalFromContext(r.Context())
	if !authenticated || !isSubscriptionAdministrator(principal) || strings.TrimSpace(principal.Subject) == "" {
		writeError(w, http.StatusForbidden, "subscription_admin_required", "platform subscription administrator role is required")
		return "", false
	}
	return strings.TrimSpace(principal.Subject), true
}
func subscriptionCorrelationID(r *http.Request, requested string) string {
	return firstNonEmpty(strings.TrimSpace(requested), headerValue(r, "X-Correlation-ID"), newID("subcorr"))
}
func writeSubscriberSubscriptionAdministrationError(w http.ResponseWriter, err error) {
	if err == storage.ErrNotFound {
		writeError(w, http.StatusNotFound, "subscription_target_not_found", "subscription product or tenant contract was not found")
		return
	}
	if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "must use") {
		writeError(w, http.StatusBadRequest, "invalid_subscription_request", err.Error())
		return
	}
	writeError(w, http.StatusServiceUnavailable, "subscription_provisioning_failed", "subscription provisioning could not be completed")
}
