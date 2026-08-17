package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberSubscriptionEnforcementFake struct {
	record storage.SubscriberEffectiveSubscriptionRecord
	err    error
}

func (f subscriberSubscriptionEnforcementFake) ListSubscriberSubscriptionProducts(context.Context) ([]storage.SubscriberSubscriptionProductRecord, error) {
	return nil, nil
}

func (f subscriberSubscriptionEnforcementFake) GetSubscriberEffectiveSubscription(context.Context, string, string) (storage.SubscriberEffectiveSubscriptionRecord, error) {
	return f.record, f.err
}

func TestSubscriptionFeatureMiddlewareHonorsProductPolicy(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	professional := storage.SubscriberEffectiveSubscriptionRecord{Product: storage.SubscriberSubscriptionProductRecord{
		ProductKey: "professional", Active: true,
		FeaturePolicyJSON: []byte(`{"value_intelligence":true,"signal_assurance_analytics":false}`),
	}}
	handler := subscriptionFeatureMiddleware(base, RouterConfig{SubscriberSubscriptionsEnabled: true, SubscriberSubscriptionRepository: subscriberSubscriptionEnforcementFake{record: professional}})

	valuation := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/valuation?symbol=AAPL", nil)
	valuation = valuation.WithContext(context.WithValue(valuation.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subject-1"}))
	valuationResponse := httptest.NewRecorder()
	handler.ServeHTTP(valuationResponse, valuation)
	if valuationResponse.Code != http.StatusNoContent {
		t.Fatalf("valuation status = %d, want %d", valuationResponse.Code, http.StatusNoContent)
	}

	assurance := httptest.NewRequest(http.MethodGet, "/v1/marketops/signal-assurance/effectiveness?tenant_id=tenant-local", nil)
	assurance = assurance.WithContext(context.WithValue(assurance.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subject-1"}))
	assuranceResponse := httptest.NewRecorder()
	handler.ServeHTTP(assuranceResponse, assurance)
	if assuranceResponse.Code != http.StatusPaymentRequired {
		t.Fatalf("assurance status = %d, want %d", assuranceResponse.Code, http.StatusPaymentRequired)
	}
}

func TestSubscriptionFeatureMiddlewareFailsClosedForUnprovisionedUser(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := subscriptionFeatureMiddleware(base, RouterConfig{SubscriberSubscriptionsEnabled: true, SubscriberSubscriptionRepository: subscriberSubscriptionEnforcementFake{err: storage.ErrNotFound}})
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/assets/NVDA/options/coverage", nil)
	request = request.WithContext(context.WithValue(request.Context(), authContextKey{}, Principal{TenantID: "tenant-local", Subject: "subject-1"}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
}

func TestSubscriptionFeatureMiddlewareLeavesExplorerRoutesAvailable(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := subscriptionFeatureMiddleware(base, RouterConfig{SubscriberSubscriptionsEnabled: true})
	request := httptest.NewRequest(http.MethodGet, "/v1/marketops/sectors/rankings?tenant_id=tenant-local", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
