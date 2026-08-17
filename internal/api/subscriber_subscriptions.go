package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lukebabs/signalops/internal/storage"
)

// registerSubscriberSubscriptionRoutes exposes read-only commercial context.
// Mutating billing, provisioning, and Stripe webhook routes remain disabled
// until their dedicated workload credential and reconciliation worker land.
func registerSubscriberSubscriptionRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/marketops/subscription-products", func(w http.ResponseWriter, r *http.Request) {
		repo := cfg.SubscriberSubscriptionRepository
		if repo == nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription storage is unavailable")
			return
		}
		products, err := repo.ListSubscriberSubscriptionProducts(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription products are unavailable")
			return
		}
		response := make([]map[string]any, 0, len(products))
		for _, product := range products {
			response = append(response, subscriptionProductResponse(product))
		}
		writeJSON(w, http.StatusOK, map[string]any{"products": response})
	})

	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/subscription", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		repo := cfg.SubscriberSubscriptionRepository
		if repo == nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription storage is unavailable")
			return
		}
		subscription, err := repo.GetSubscriberEffectiveSubscription(r.Context(), tenantID, subject)
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"access_state": "unprovisioned", "subscription": nil})
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access is unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_state": "active", "subscription": effectiveSubscriptionResponse(subscription)})
	})
}

func subscriptionProductResponse(product storage.SubscriberSubscriptionProductRecord) map[string]any {
	return map[string]any{
		"product_key":    product.ProductKey,
		"billing_scope":  product.BillingScope,
		"display_name":   product.DisplayName,
		"is_free":        product.IsFree,
		"trial_days":     product.TrialDays,
		"feature_policy": subscriptionJSON(product.FeaturePolicyJSON),
		"limit_policy":   subscriptionJSON(product.LimitPolicyJSON),
		"revision":       product.Revision,
	}
}

func effectiveSubscriptionResponse(subscription storage.SubscriberEffectiveSubscriptionRecord) map[string]any {
	response := subscriptionProductResponse(subscription.Product)
	response["subscription_id"] = subscription.SubscriptionID
	response["status"] = subscription.Status
	response["source"] = subscription.Source
	response["seat_role"] = subscription.SeatRole
	response["trial_ends_at"] = subscription.TrialEndsAt
	response["current_period_ends_at"] = subscription.CurrentPeriodEndsAt
	response["grace_ends_at"] = subscription.GraceEndsAt
	response["canceled_at"] = subscription.CanceledAt
	return response
}

func subscriptionJSON(raw []byte) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(raw, &result)
	return result
}
