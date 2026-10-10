package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

// registerSubscriberSubscriptionRoutes exposes commercial product context and
// constrained Stripe Checkout startup. Entitlement activation remains webhook-
// authoritative after payment completion.
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
		writeJSON(w, http.StatusOK, map[string]any{"products": response, "checkout_enabled": resolveStripeCheckoutClient(cfg) != nil})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscription/checkout", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		productRepo := cfg.SubscriberSubscriptionRepository
		adminRepo := cfg.SubscriberSubscriptionAdministrationRepository
		checkoutClient := resolveStripeCheckoutClient(cfg)
		if productRepo == nil || adminRepo == nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription storage is unavailable")
			return
		}
		if checkoutClient == nil {
			writeError(w, http.StatusServiceUnavailable, "stripe_checkout_disabled", "Stripe checkout is not configured")
			return
		}
		var request struct {
			ProductKey    string `json:"product_key"`
			BillingPeriod string `json:"billing_period"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_checkout_request", "checkout request must be JSON")
			return
		}
		productKey := strings.ToLower(strings.TrimSpace(request.ProductKey))
		billingPeriod := normalizeStripeBillingPeriod(request.BillingPeriod)
		if productKey != "explorer" && productKey != "professional" {
			writeError(w, http.StatusBadRequest, "unsupported_subscription_product", "self-service checkout supports Explorer and Professional only")
			return
		}
		if billingPeriod == "" {
			writeError(w, http.StatusBadRequest, "unsupported_billing_period", "billing_period must be monthly or annual")
			return
		}
		products, err := productRepo.ListSubscriberSubscriptionProducts(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription products are unavailable")
			return
		}
		var product storage.SubscriberSubscriptionProductRecord
		found := false
		for _, candidate := range products {
			if candidate.ProductKey == productKey {
				product = candidate
				found = true
				break
			}
		}
		if !found || !product.Active || product.BillingScope != "subject" {
			writeError(w, http.StatusConflict, "subscription_product_unavailable", "requested subscription product is unavailable")
			return
		}
		priceID := stripeCheckoutPrice(product, billingPeriod)
		if priceID == "" {
			writeError(w, http.StatusConflict, "subscription_price_unmapped", "requested subscription price is not mapped to Stripe")
			return
		}
		checkoutRef := newID("subcheckout")
		correlationID := subscriptionCorrelationID(r, checkoutRef)
		if err := adminRepo.CreateSubscriberCheckoutSession(r.Context(), storage.SubscriberCheckoutSessionInput{
			CheckoutRef: checkoutRef, TenantID: tenantID, Subject: subject, ProductKey: productKey, BillingPeriod: billingPeriod,
			StripePriceID: priceID, Status: "created", ActorSubject: subject, CorrelationID: correlationID,
		}); err != nil {
			writeError(w, http.StatusServiceUnavailable, "checkout_record_failed", "checkout session could not be recorded")
			return
		}
		session, err := checkoutClient.CreateCheckoutSession(r.Context(), stripeCheckoutSessionRequest{
			PriceID: priceID, SuccessURL: cfg.StripeCheckoutSuccessURL, CancelURL: cfg.StripeCheckoutCancelURL,
			CheckoutRef: checkoutRef, ProductKey: productKey, BillingPeriod: billingPeriod,
		})
		if err != nil {
			_ = adminRepo.CreateSubscriberCheckoutSession(r.Context(), storage.SubscriberCheckoutSessionInput{
				CheckoutRef: checkoutRef, TenantID: tenantID, Subject: subject, ProductKey: productKey, BillingPeriod: billingPeriod,
				StripePriceID: priceID, Status: "failed", ActorSubject: subject, CorrelationID: correlationID,
			})
			writeError(w, http.StatusServiceUnavailable, "stripe_checkout_failed", "Stripe checkout session could not be created")
			return
		}
		if err := adminRepo.CreateSubscriberCheckoutSession(r.Context(), storage.SubscriberCheckoutSessionInput{
			CheckoutRef: checkoutRef, TenantID: tenantID, Subject: subject, ProductKey: productKey, BillingPeriod: billingPeriod,
			StripePriceID: priceID, StripeSessionID: session.ID, Status: "checkout_started", ActorSubject: subject,
			CorrelationID: correlationID, CheckoutURLReturned: true,
		}); err != nil {
			writeError(w, http.StatusServiceUnavailable, "checkout_record_failed", "checkout session could not be recorded")
			return
		}
		_ = adminRepo.RecordSubscriberUpgradeInteraction(r.Context(), storage.SubscriberUpgradeInteractionInput{
			TenantID: tenantID, Subject: subject, AppID: "marketops", InteractionType: "checkout_started",
			SourceFeature: "subscription_checkout", SourceRoute: r.URL.Path, CurrentTier: "", RequiredTier: productKey,
			PromptVariant: "stripe_checkout_v1", CTALabel: "Start checkout", CorrelationID: correlationID,
			MetadataJSON: []byte(`{"checkout_ref":"` + checkoutRef + `","billing_period":"` + billingPeriod + `"}`),
		})
		writeJSON(w, http.StatusOK, map[string]any{"checkout_url": session.URL, "checkout_ref": checkoutRef, "stripe_session_id": session.ID})
	})

	mux.HandleFunc("POST /v1/tenants/{tenant_id}/marketops/subscription/portal", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		subject, ok := requireRequestSubject(w, r, "")
		if !ok {
			return
		}
		repo := cfg.SubscriberSubscriptionRepository
		portalClient := resolveStripePortalClient(cfg)
		if repo == nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription storage is unavailable")
			return
		}
		if portalClient == nil {
			writeError(w, http.StatusServiceUnavailable, "stripe_portal_disabled", "Stripe customer portal is not configured")
			return
		}
		subscription, err := repo.GetSubscriberEffectiveSubscription(r.Context(), tenantID, subject)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusConflict, "subscription_portal_unavailable", "an active subscription is required before opening the customer portal")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access is unavailable")
			return
		}
		if strings.TrimSpace(subscription.StripeCustomerID) == "" {
			writeError(w, http.StatusConflict, "subscription_portal_unavailable", "subscription is not backed by a Stripe customer")
			return
		}
		session, err := portalClient.CreateBillingPortalSession(r.Context(), stripeBillingPortalSessionRequest{
			CustomerID: subscription.StripeCustomerID,
			ReturnURL:  cfg.StripePortalReturnURL,
		})
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "stripe_portal_failed", "Stripe customer portal session could not be created")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"portal_url": session.URL, "stripe_session_id": session.ID})
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
			writeJSON(w, http.StatusOK, map[string]any{"access_state": "unprovisioned", "enforcement_enabled": cfg.SubscriberSubscriptionsEnabled, "subscription": nil})
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access is unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_state": "active", "enforcement_enabled": cfg.SubscriberSubscriptionsEnabled, "subscription": effectiveSubscriptionResponse(subscription)})
	})
}

func subscriptionProductResponse(product storage.SubscriberSubscriptionProductRecord) map[string]any {
	return map[string]any{
		"product_key":             product.ProductKey,
		"billing_scope":           product.BillingScope,
		"display_name":            product.DisplayName,
		"is_free":                 product.IsFree,
		"trial_days":              product.TrialDays,
		"stripe_product_id":       product.StripeProductID,
		"stripe_monthly_price_id": product.StripeMonthlyPriceID,
		"stripe_annual_price_id":  product.StripeAnnualPriceID,
		"monthly_display_price":   product.MonthlyDisplayPrice,
		"annual_display_price":    product.AnnualDisplayPrice,
		"feature_policy":          subscriptionJSON(product.FeaturePolicyJSON),
		"limit_policy":            subscriptionJSON(product.LimitPolicyJSON),
		"revision":                product.Revision,
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
