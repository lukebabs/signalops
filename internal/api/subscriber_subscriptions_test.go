package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberCheckoutFake struct {
	request stripeCheckoutSessionRequest
	session stripeCheckoutSession
	err     error
}

func (f *subscriberCheckoutFake) CreateCheckoutSession(_ context.Context, request stripeCheckoutSessionRequest) (stripeCheckoutSession, error) {
	f.request = request
	if f.err != nil {
		return stripeCheckoutSession{}, f.err
	}
	return f.session, nil
}

type subscriberSubscriptionAPIFake struct {
	products []storage.SubscriberSubscriptionProductRecord
	record   storage.SubscriberEffectiveSubscriptionRecord
	err      error
	tenant   string
	subject  string
}

func (f *subscriberSubscriptionAPIFake) ListSubscriberSubscriptionProducts(context.Context) ([]storage.SubscriberSubscriptionProductRecord, error) {
	return f.products, f.err
}

func (f *subscriberSubscriptionAPIFake) GetSubscriberEffectiveSubscription(_ context.Context, tenantID, subject string) (storage.SubscriberEffectiveSubscriptionRecord, error) {
	f.tenant, f.subject = tenantID, subject
	return f.record, f.err
}

func TestSubscriberSubscriptionProductsExposePublicCommercialCatalog(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{products: []storage.SubscriberSubscriptionProductRecord{{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", TrialDays: 7, StripeProductID: "prod_test", StripeMonthlyPriceID: "price_monthly", StripeAnnualPriceID: "price_annual", FeaturePolicyJSON: []byte(`{"value_intelligence":true}`), LimitPolicyJSON: []byte(`{"private_watchlists":20}`), Revision: 1}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/subscription-products", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"professional"`) || !strings.Contains(recorder.Body.String(), `"stripe_monthly_price_id":"price_monthly"`) || !strings.Contains(recorder.Body.String(), `"checkout_enabled":false`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberSubscriptionBindsTenantAndSubject(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{record: storage.SubscriberEffectiveSubscriptionRecord{TenantID: "tenant-local", Subject: "user-123", SubscriptionID: "sub-professional", Status: storage.SubscriberSubscriptionTrialing, Source: "subject", Product: storage.SubscriberSubscriptionProductRecord{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", TrialDays: 7, FeaturePolicyJSON: []byte(`{"value_intelligence":true}`), LimitPolicyJSON: []byte(`{"private_watchlists":20}`), Revision: 1}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || store.tenant != "tenant-local" || store.subject != "user-123" || !strings.Contains(recorder.Body.String(), `"access_state":"active"`) || !strings.Contains(recorder.Body.String(), `"enforcement_enabled":false`) {
		t.Fatalf("status=%d tenant=%s subject=%s body=%s", recorder.Code, store.tenant, store.subject, recorder.Body.String())
	}
	foreign := httptest.NewRecorder()
	router.ServeHTTP(foreign, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-other/marketops/subscription", nil), fixture.token(t, nil)))
	if foreign.Code != http.StatusForbidden || !strings.Contains(foreign.Body.String(), "tenant_mismatch") {
		t.Fatalf("foreign status=%d body=%s", foreign.Code, foreign.Body.String())
	}
}

func TestSubscriberSubscriptionReturnsUnprovisionedWithoutFallback(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{err: storage.ErrNotFound}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"access_state":"unprovisioned"`) || !strings.Contains(recorder.Body.String(), `"enforcement_enabled":false`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	store.err = errors.New("database unavailable")
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if failed.Code != http.StatusServiceUnavailable || !strings.Contains(failed.Body.String(), "subscription_unavailable") {
		t.Fatalf("failed status=%d body=%s", failed.Code, failed.Body.String())
	}
}

func TestSubscriberSubscriptionReportsWhenEnforcementIsEnabled(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{err: storage.ErrNotFound}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"enforcement_enabled":true`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberProductsReportsCheckoutEnabledWhenStripeClientConfigured(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{products: []storage.SubscriberSubscriptionProductRecord{{ProductKey: "explorer", BillingScope: "subject", DisplayName: "Explorer", Active: true, FeaturePolicyJSON: []byte(`{}`), LimitPolicyJSON: []byte(`{}`), Revision: 1}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store, StripeCheckoutClient: &subscriberCheckoutFake{session: stripeCheckoutSession{ID: "cs_test_123", URL: "https://checkout.stripe.com/c/pay/cs_test_123"}}})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/subscription-products", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"checkout_enabled":true`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberCheckoutCreatesStripeSessionFromMappedPrice(t *testing.T) {
	products := []storage.SubscriberSubscriptionProductRecord{
		{ProductKey: "explorer", BillingScope: "subject", DisplayName: "Explorer", Active: true, StripeMonthlyPriceID: "price_explorer_month", StripeAnnualPriceID: "price_explorer_year", FeaturePolicyJSON: []byte(`{}`), LimitPolicyJSON: []byte(`{}`), Revision: 1},
		{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", Active: true, StripeMonthlyPriceID: "price_pro_month", StripeAnnualPriceID: "price_pro_year", FeaturePolicyJSON: []byte(`{}`), LimitPolicyJSON: []byte(`{}`), Revision: 1},
	}
	productRepo := &subscriberSubscriptionAPIFake{products: products}
	adminRepo := &subscriberSubscriptionAdministrationFake{}
	checkout := &subscriberCheckoutFake{session: stripeCheckoutSession{ID: "cs_test_123", URL: "https://checkout.stripe.com/c/pay/cs_test_123"}}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionRepository: productRepo, SubscriberSubscriptionAdministrationRepository: adminRepo, StripeCheckoutClient: checkout, StripeCheckoutSuccessURL: "https://signalops.syncratic.io/success", StripeCheckoutCancelURL: "https://signalops.syncratic.io/cancel"})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-pilot-b/marketops/subscription/checkout", strings.NewReader(`{"product_key":"professional","billing_period":"annual"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}, Access: map[string]string{"marketops": "read"}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "checkout.stripe.com") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if checkout.request.PriceID != "price_pro_year" || checkout.request.CheckoutRef == "" || checkout.request.SuccessURL == "" || checkout.request.CancelURL == "" {
		t.Fatalf("unexpected checkout request: %+v", checkout.request)
	}
	if adminRepo.checkoutSession.TenantID != "tenant-pilot-b" || adminRepo.checkoutSession.Subject != "pilot-sub" || adminRepo.checkoutSession.ProductKey != "professional" || adminRepo.checkoutSession.BillingPeriod != "annual" || adminRepo.checkoutSession.StripeSessionID != "cs_test_123" || !adminRepo.checkoutSession.CheckoutURLReturned {
		t.Fatalf("unexpected checkout ledger: %+v", adminRepo.checkoutSession)
	}
	if adminRepo.upgradeInteraction.InteractionType != "checkout_started" || adminRepo.upgradeInteraction.RequiredTier != "professional" {
		t.Fatalf("unexpected upgrade attribution: %+v", adminRepo.upgradeInteraction)
	}
}

func TestSubscriberCheckoutFailsClosedWithoutStripeClient(t *testing.T) {
	productRepo := &subscriberSubscriptionAPIFake{products: []storage.SubscriberSubscriptionProductRecord{{ProductKey: "explorer", BillingScope: "subject", DisplayName: "Explorer", Active: true, StripeMonthlyPriceID: "price_explorer_month", FeaturePolicyJSON: []byte(`{}`), LimitPolicyJSON: []byte(`{}`), Revision: 1}}}
	adminRepo := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionRepository: productRepo, SubscriberSubscriptionAdministrationRepository: adminRepo})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-pilot-b/marketops/subscription/checkout", strings.NewReader(`{"product_key":"explorer","billing_period":"monthly"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}, Access: map[string]string{"marketops": "read"}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "stripe_checkout_disabled") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if adminRepo.checkoutSession.CheckoutRef != "" {
		t.Fatalf("checkout should not be recorded when Stripe is disabled: %+v", adminRepo.checkoutSession)
	}
}

func TestSubscriberCheckoutRejectsUnmappedPrice(t *testing.T) {
	productRepo := &subscriberSubscriptionAPIFake{products: []storage.SubscriberSubscriptionProductRecord{{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", Active: true, StripeMonthlyPriceID: "price_pro_month", FeaturePolicyJSON: []byte(`{}`), LimitPolicyJSON: []byte(`{}`), Revision: 1}}}
	adminRepo := &subscriberSubscriptionAdministrationFake{}
	checkout := &subscriberCheckoutFake{session: stripeCheckoutSession{ID: "cs_test_123", URL: "https://checkout.stripe.com/c/pay/cs_test_123"}}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionRepository: productRepo, SubscriberSubscriptionAdministrationRepository: adminRepo, StripeCheckoutClient: checkout})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-pilot-b/marketops/subscription/checkout", strings.NewReader(`{"product_key":"professional","billing_period":"annual"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}, Access: map[string]string{"marketops": "read"}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "subscription_price_unmapped") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
