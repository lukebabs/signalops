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
	snapshot       storage.SubscriberSubscriptionAdministrationSnapshot
	product        storage.SubscriberSubscriptionProductMutation
	productBilling storage.SubscriberSubscriptionProductBillingMutation
	subject        storage.SubscriberSubjectSubscriptionMutation
	subjectBilling storage.SubscriberSubjectSubscriptionBillingMutation
	tenant         storage.SubscriberTenantSubscriptionMutation
	tenantBilling  storage.SubscriberTenantSubscriptionBillingMutation
	seat           storage.SubscriberSubscriptionSeatMutation
	stripeWebhook  storage.SubscriberStripeWebhookMutation
}

func (f *subscriberSubscriptionAdministrationFake) ListSubscriberSubscriptionAdministration(_ context.Context, filter storage.SubscriberSubscriptionAdministrationFilter) (storage.SubscriberSubscriptionAdministrationSnapshot, error) {
	if f.snapshot.TenantID == "" {
		f.snapshot.TenantID = filter.TenantID
	}
	return f.snapshot, nil
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
	body := `{"id":"evt_1","type":"customer.subscription.updated","data":{"object":{"id":"sub_123","customer":"cus_123","status":"active","current_period_end":1800000000}}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/billing/stripe/webhook", strings.NewReader(body))
	request.Header.Set("Stripe-Signature", stripeTestSignature(body, "whsec_test"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if store.stripeWebhook.ProviderEventID != "evt_1" || store.stripeWebhook.StripeSubscriptionID != "sub_123" || store.stripeWebhook.StripeCustomerID != "cus_123" || store.stripeWebhook.Status != storage.SubscriberSubscriptionActive {
		t.Fatalf("unexpected webhook mutation: %+v", store.stripeWebhook)
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
