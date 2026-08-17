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
	subject storage.SubscriberSubjectSubscriptionMutation
	tenant  storage.SubscriberTenantSubscriptionMutation
	seat    storage.SubscriberSubscriptionSeatMutation
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
