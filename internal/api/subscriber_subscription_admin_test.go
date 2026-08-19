package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSubscriptionAdministrationFake struct {
	snapshot storage.SubscriberSubscriptionAdministrationSnapshot
	product  storage.SubscriberSubscriptionProductMutation
	subject  storage.SubscriberSubjectSubscriptionMutation
	tenant   storage.SubscriberTenantSubscriptionMutation
	seat     storage.SubscriberSubscriptionSeatMutation
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

func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberSubjectSubscription(_ context.Context, input storage.SubscriberSubjectSubscriptionMutation) error {
	f.subject = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberTenantSubscription(_ context.Context, input storage.SubscriberTenantSubscriptionMutation) error {
	f.tenant = input
	return nil
}
func (f *subscriberSubscriptionAdministrationFake) UpsertSubscriberSubscriptionSeat(_ context.Context, input storage.SubscriberSubscriptionSeatMutation) error {
	f.seat = input
	return nil
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
