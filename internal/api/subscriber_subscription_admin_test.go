package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSubscriptionAdministrationFake struct {
	snapshot           storage.SubscriberSubscriptionAdministrationSnapshot
	product            storage.SubscriberSubscriptionProductMutation
	productBilling     storage.SubscriberSubscriptionProductBillingMutation
	subject            storage.SubscriberSubjectSubscriptionMutation
	subjectBilling     storage.SubscriberSubjectSubscriptionBillingMutation
	tenant             storage.SubscriberTenantSubscriptionMutation
	tenantBilling      storage.SubscriberTenantSubscriptionBillingMutation
	seat               storage.SubscriberSubscriptionSeatMutation
	stripeWebhook      storage.SubscriberStripeWebhookMutation
	activity           storage.SubscriberUserActivityRecordInput
	upgradeInteraction storage.SubscriberUpgradeInteractionInput
	checkoutSession    storage.SubscriberCheckoutSessionInput
	refundCreate       storage.SubscriberRefundRequestInput
	refundUpdate       storage.SubscriberRefundRequestMutation
	activityFilter     storage.SubscriberUserActivityFilter
	activitySnapshot   storage.SubscriberUserActivitySnapshot
}

func (f *subscriberSubscriptionAdministrationFake) ListSubscriberSubscriptionAdministration(_ context.Context, filter storage.SubscriberSubscriptionAdministrationFilter) (storage.SubscriberSubscriptionAdministrationSnapshot, error) {
	if f.snapshot.TenantID == "" {
		f.snapshot.TenantID = filter.TenantID
	}
	return f.snapshot, nil
}
func (f *subscriberSubscriptionAdministrationFake) ListSubscriberUserActivity(_ context.Context, filter storage.SubscriberUserActivityFilter) (storage.SubscriberUserActivitySnapshot, error) {
	f.activityFilter = filter
	if f.activitySnapshot.TenantID == "" {
		f.activitySnapshot.TenantID = filter.TenantID
	}
	return f.activitySnapshot, nil
}
func (f *subscriberSubscriptionAdministrationFake) RecordSubscriberUserActivity(_ context.Context, input storage.SubscriberUserActivityRecordInput) error {
	f.activity = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) RecordSubscriberUpgradeInteraction(_ context.Context, input storage.SubscriberUpgradeInteractionInput) error {
	f.upgradeInteraction = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) CreateSubscriberCheckoutSession(_ context.Context, input storage.SubscriberCheckoutSessionInput) error {
	f.checkoutSession = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpdateSubscriberSubscriptionProduct(_ context.Context, input storage.SubscriberSubscriptionProductMutation) error {
	f.product = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpdateSubscriberSubscriptionProductBilling(_ context.Context, input storage.SubscriberSubscriptionProductBillingMutation) error {
	f.productBilling = input
	return nil
}

func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberSubjectSubscription(_ context.Context, input storage.SubscriberSubjectSubscriptionMutation) error {
	f.subject = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpdateSubscriberSubjectSubscriptionBilling(_ context.Context, input storage.SubscriberSubjectSubscriptionBillingMutation) error {
	f.subjectBilling = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberTenantSubscription(_ context.Context, input storage.SubscriberTenantSubscriptionMutation) error {
	f.tenant = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpdateSubscriberTenantSubscriptionBilling(_ context.Context, input storage.SubscriberTenantSubscriptionBillingMutation) error {
	f.tenantBilling = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberSubscriptionSeat(_ context.Context, input storage.SubscriberSubscriptionSeatMutation) error {
	f.seat = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) ProcessSubscriberStripeWebhook(_ context.Context, input storage.SubscriberStripeWebhookMutation) (storage.SubscriberBillingWebhookEventRecord, error) {
	f.stripeWebhook = input
	return storage.SubscriberBillingWebhookEventRecord{ProviderEventID: input.ProviderEventID, EventType: input.EventType, ProcessingStatus: "processed"}, nil
}
func (f *subscriberSubscriptionAdministrationFake) CreateSubscriberRefundRequest(_ context.Context, input storage.SubscriberRefundRequestInput) (storage.SubscriberRefundRequestRecord, error) {
	f.refundCreate = input
	return storage.SubscriberRefundRequestRecord{
		RefundRequestID: "subrefund-test", TenantID: input.TenantID, Subject: input.Subject, SubjectEmail: "pilot@example.com",
		SubscriptionID: "sub-subscription-test", ProductKey: "explorer", DisplayName: "Explorer",
		StripeCustomerID: "cus_secret", StripeSubscriptionID: "sub_secret", StripeSessionID: "cs_secret",
		RequestedAmountCents: input.RequestedAmountCents, Currency: input.Currency, Reason: input.Reason, Status: "requested", RequestedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, nil
}
func (f *subscriberSubscriptionAdministrationFake) UpdateSubscriberRefundRequest(_ context.Context, input storage.SubscriberRefundRequestMutation) (storage.SubscriberRefundRequestRecord, error) {
	f.refundUpdate = input
	return storage.SubscriberRefundRequestRecord{
		RefundRequestID: input.RefundRequestID, TenantID: input.TenantID, Subject: "pilot-sub", SubjectEmail: "pilot@example.com",
		SubscriptionID: "sub-subscription-test", ProductKey: "explorer", DisplayName: "Explorer",
		StripeCustomerID: "cus_admin", StripeSubscriptionID: "sub_admin", StripeSessionID: "cs_admin",
		Currency: "usd", Reason: "requested by customer", Status: input.Status, AdminNote: input.AdminNote, ActorSubject: input.ActorSubject, ActorEmail: "admin@example.com", RequestedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, nil
}

func TestSubscriberSubscriptionAdministrationFailsClosedWithoutPlatformRole(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/administration/subscriptions/subject", strings.NewReader(`{"tenant_id":"tenant-pilot-b","subject":"pilot","product_key":"professional","status":"trialing"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if store.subject.TenantID != "" {
		t.Fatal("unauthenticated request reached provisioning repository")
	}
}

func TestSubscriberSubscriptionAdministrationAllowsDedicatedPlatformRoleAcrossTenants(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/administration/subscriptions/subject", strings.NewReader(`{"tenant_id":"tenant-pilot-b","subject":"pilot","product_key":"professional","status":"trialing","correlation_id":"test-correlation"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if store.subject.TenantID != "tenant-pilot-b" || store.subject.Subject != "pilot" || store.subject.ActorSubject != "subscription-admin" || store.subject.CorrelationID != "test-correlation" {
		t.Fatalf("unexpected mutation: %+v", store.subject)
	}
}

func TestSubscriberSubscriptionAdministrationListsTenantGovernanceSnapshot(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{snapshot: storage.SubscriberSubscriptionAdministrationSnapshot{
		TenantID:             "tenant-pilot-b",
		Products:             []storage.SubscriberSubscriptionProductRecord{{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", FeaturePolicyJSON: []byte(`{"value_intelligence":true}`), LimitPolicyJSON: []byte(`{"private_watchlists":20}`), Active: true, Revision: 2}},
		SubjectSubscriptions: []storage.SubscriberSubjectSubscriptionRecord{{TenantID: "tenant-pilot-b", Subject: "pilot", SubscriptionID: "sub-1", ProductKey: "professional", DisplayName: "Professional", Status: "active"}},
		UpgradeInteractions:  []storage.SubscriberUpgradeInteractionRecord{{InteractionID: "subupg-1", TenantID: "tenant-pilot-b", Subject: "pilot", InteractionType: "prompt_shown", SourceFeature: "value_intelligence", CurrentTier: "explorer", RequiredTier: "professional", OccurredAt: time.Now().UTC()}},
	}}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodGet, "/v1/administration/subscriptions?tenant_id=tenant-pilot-b", nil)
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "tenant-pilot-b") || !strings.Contains(response.Body.String(), "value_intelligence") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}

func TestSubscriberSubscriptionAdministrationUpdatesProductPolicy(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/subscriptions/products/professional", strings.NewReader(`{"display_name":"Professional","is_free":false,"trial_days":14,"feature_policy":{"value_intelligence":true},"limit_policy":{"private_watchlists":50},"active":true,"correlation_id":"policy-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.product.TenantID != "tenant-local" || store.product.ProductKey != "professional" || store.product.TrialDays != 14 || store.product.ActorSubject != "subscription-admin" || store.product.CorrelationID != "policy-test" {
		t.Fatalf("unexpected product mutation: %+v", store.product)
	}
}

func TestSubscriberSubscriptionAdministrationUpdatesProductBilling(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/subscriptions/products/professional/billing", strings.NewReader(`{"stripe_product_id":"prod_123","stripe_monthly_price_id":"price_m","stripe_annual_price_id":"price_a","correlation_id":"billing-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.productBilling.ProductKey != "professional" || store.productBilling.StripeProductID != "prod_123" || store.productBilling.StripeMonthlyPriceID != "price_m" || store.productBilling.StripeAnnualPriceID != "price_a" {
		t.Fatalf("unexpected billing mutation: %+v", store.productBilling)
	}
}

func TestSubscriberStripeWebhookRequiresValidSignature(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store, StripeWebhookSecret: "whsec_test"})
	body := `{"id":"evt_1","type":"customer.subscription.updated","data":{"object":{"id":"sub_123","customer":"cus_123","status":"active","current_period_end":1800000000,"metadata":{"checkout_ref":"subcheckout-test"}}}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/billing/stripe/webhook", strings.NewReader(body))
	request.Header.Set("Stripe-Signature", stripeTestSignature(body, "whsec_test"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.stripeWebhook.ProviderEventID != "evt_1" || store.stripeWebhook.StripeSubscriptionID != "sub_123" || store.stripeWebhook.StripeCustomerID != "cus_123" || store.stripeWebhook.Status != storage.SubscriberSubscriptionActive || store.stripeWebhook.CheckoutRef != "subcheckout-test" {
		t.Fatalf("unexpected webhook mutation: %+v", store.stripeWebhook)
	}
}

func TestSubscriberStripeWebhookAcceptsCheckoutSessionCompleted(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store, StripeWebhookSecret: "whsec_test"})
	body := `{"id":"evt_checkout","type":"checkout.session.completed","data":{"object":{"id":"cs_live_123","customer":"cus_123","subscription":"sub_123","status":"complete","payment_status":"paid","metadata":{"checkout_ref":"subcheckout-test","product_key":"explorer","billing_period":"monthly"}}}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/billing/stripe/webhook", strings.NewReader(body))
	request.Header.Set("Stripe-Signature", stripeTestSignature(body, "whsec_test"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.stripeWebhook.ProviderEventID != "evt_checkout" || store.stripeWebhook.EventType != "checkout.session.completed" || store.stripeWebhook.StripeSubscriptionID != "sub_123" || store.stripeWebhook.StripeCustomerID != "cus_123" || store.stripeWebhook.Status != storage.SubscriberSubscriptionActive || store.stripeWebhook.CheckoutRef != "subcheckout-test" {
		t.Fatalf("unexpected checkout webhook mutation: %+v", store.stripeWebhook)
	}
}

func TestSubscriberStripeWebhookRejectsBadSignature(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store, StripeWebhookSecret: "whsec_test"})
	request := httptest.NewRequest(http.MethodPost, "/v1/billing/stripe/webhook", strings.NewReader(`{"id":"evt_bad","type":"invoice.payment_failed","data":{"object":{}}}`))
	request.Header.Set("Stripe-Signature", "t=1,v1=bad")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if store.stripeWebhook.ProviderEventID != "" {
		t.Fatal("invalid signature reached repository")
	}
}

func stripeTestSignature(body string, secret string) string {
	timestamp := time.Now().UTC().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp, body)))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

func TestSubscriberUpgradeInteractionRecordsAuthenticatedPrincipalScope(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/marketops/subscriptions/upgrade-interactions", strings.NewReader(`{"interaction_type":"prompt_clicked","source_feature":"value_intelligence","source_route":"/marketops/valuation","current_tier":"explorer","required_tier":"professional","prompt_variant":"contextual_route_gate_v1","cta_label":"View upgrade options","correlation_id":"upgrade-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", response.Code, response.Body.String())
	}
	if store.upgradeInteraction.TenantID != "tenant-pilot-b" || store.upgradeInteraction.Subject != "pilot-sub" || store.upgradeInteraction.InteractionType != "prompt_clicked" || store.upgradeInteraction.SourceFeature != "value_intelligence" || store.upgradeInteraction.RequiredTier != "professional" {
		t.Fatalf("unexpected upgrade interaction: %+v", store.upgradeInteraction)
	}
}

func TestSubscriberSessionActivityRecordsAuthenticatedPrincipalScope(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/session/activity", strings.NewReader(`{"event_type":"feature_view","app_id":"marketops","feature_key":"dashboard","route_path":"/marketops/dashboard","correlation_id":"activity-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", response.Code, response.Body.String())
	}
	if store.activity.TenantID != "tenant-pilot-b" || store.activity.Subject != "pilot-sub" || store.activity.EventType != "feature_view" || store.activity.FeatureKey != "dashboard" || store.activity.RoutePath != "/marketops/dashboard" {
		t.Fatalf("unexpected activity record: %+v", store.activity)
	}
}

func TestSubscriberSubscriptionAdministrationListsUserActivity(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	store := &subscriberSubscriptionAdministrationFake{activitySnapshot: storage.SubscriberUserActivitySnapshot{
		TenantID:  "tenant-pilot-b",
		Summaries: []storage.SubscriberUserActivitySummaryRecord{{Subject: "pilot-sub", SubjectEmail: "pilot@example.com", LastActivityAt: &now, FeatureViewCount: 3}},
		Events:    []storage.SubscriberUserActivityEventRecord{{ActivityID: "subact-1", TenantID: "tenant-pilot-b", Subject: "pilot-sub", SubjectEmail: "pilot@example.com", AppID: "marketops", EventType: "login", FeatureKey: "session", OccurredAt: now}},
	}}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodGet, "/v1/administration/subscriptions/activity?tenant_id=tenant-pilot-b&q=pilot&limit=50", nil)
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "pilot@example.com") || !strings.Contains(response.Body.String(), "subact-1") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
	if store.activityFilter.TenantID != "tenant-pilot-b" || store.activityFilter.Query != "pilot" || store.activityFilter.Limit != 50 {
		t.Fatalf("unexpected activity filter: %+v", store.activityFilter)
	}
}

func TestMarketOpsMutationActivityCapturedBestEffort(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/private-lists", strings.NewReader(`{"list_name":"Research"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "user-sub", Roles: map[string]struct{}{roleViewer: {}}, Access: map[string]string{"marketops": "read"}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if store.activity.TenantID != "tenant-local" || store.activity.Subject != "user-sub" || store.activity.EventType != "api_mutation" || store.activity.HTTPMethod != http.MethodPost || store.activity.FeatureKey != "subscriber" || store.activity.RoutePath != "/v1/tenants/{tenant}/marketops/subscriber/private-lists" {
		t.Fatalf("unexpected captured activity: %+v; response=%d %s", store.activity, response.Code, response.Body.String())
	}
}

func TestSubscriberRefundRequestCreatesSafeSubscriberRecord(t *testing.T) {
	amount := 2499
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-pilot-b/marketops/subscription/refund-requests", strings.NewReader(`{"requested_amount_cents":2499,"currency":"USD","reason":"I meant to choose annual","correlation_id":"refund-create-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", response.Code, response.Body.String())
	}
	if store.refundCreate.TenantID != "tenant-pilot-b" || store.refundCreate.Subject != "pilot-sub" || store.refundCreate.Currency != "usd" || store.refundCreate.Reason == "" || store.refundCreate.CorrelationID != "refund-create-test" {
		t.Fatalf("unexpected refund create mutation: %+v", store.refundCreate)
	}
	if store.refundCreate.RequestedAmountCents == nil || *store.refundCreate.RequestedAmountCents != amount {
		t.Fatalf("unexpected requested amount: %+v", store.refundCreate.RequestedAmountCents)
	}
	body := response.Body.String()
	for _, forbidden := range []string{"cus_secret", "sub_secret", "cs_secret", "stripe_customer_id", "stripe_subscription_id", "stripe_session_id", "subscription_id"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("subscriber refund response leaked %q: %s", forbidden, body)
		}
	}
}

func TestSubscriberRefundRequestAdminUpdateRequiresRole(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/subscriptions/refund-requests/subrefund-test", strings.NewReader(`{"tenant_id":"tenant-pilot-b","status":"reviewing","admin_note":"Checking Stripe payment record"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-pilot-b", Subject: "pilot-sub", Roles: map[string]struct{}{roleViewer: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if store.refundUpdate.RefundRequestID != "" {
		t.Fatal("non-admin refund mutation reached repository")
	}
}

func TestSubscriberRefundRequestAdminUpdateRecordsDisposition(t *testing.T) {
	store := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{SubscriberListsEnabled: true, SubscriberSubscriptionAdministrationRepository: store})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/subscriptions/refund-requests/subrefund-test", strings.NewReader(`{"tenant_id":"tenant-pilot-b","status":"approved_for_manual_refund","admin_note":"Refund must be executed in Stripe Dashboard","correlation_id":"refund-admin-test"}`))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subscription-admin", Roles: map[string]struct{}{roleSubscriptionAdmin: {}}}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.refundUpdate.TenantID != "tenant-pilot-b" || store.refundUpdate.RefundRequestID != "subrefund-test" || store.refundUpdate.Status != "approved_for_manual_refund" || store.refundUpdate.ActorSubject != "subscription-admin" || store.refundUpdate.CorrelationID != "refund-admin-test" {
		t.Fatalf("unexpected refund update mutation: %+v", store.refundUpdate)
	}
	if !strings.Contains(response.Body.String(), "cus_admin") || !strings.Contains(response.Body.String(), "pilot@example.com") {
		t.Fatalf("admin response missing operational details: %s", response.Body.String())
	}
}
