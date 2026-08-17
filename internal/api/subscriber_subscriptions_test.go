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

func TestSubscriberSubscriptionProductsExposeOnlyCommercialPolicy(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{products: []storage.SubscriberSubscriptionProductRecord{{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", TrialDays: 7, FeaturePolicyJSON: []byte(`{"value_intelligence":true}`), LimitPolicyJSON: []byte(`{"private_watchlists":20}`), Revision: 1}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/subscription-products", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"professional"`) || strings.Contains(recorder.Body.String(), "stripe_") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberSubscriptionBindsTenantAndSubject(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberSubscriptionAPIFake{record: storage.SubscriberEffectiveSubscriptionRecord{TenantID: "tenant-local", Subject: "user-123", SubscriptionID: "sub-professional", Status: storage.SubscriberSubscriptionTrialing, Source: "subject", Product: storage.SubscriberSubscriptionProductRecord{ProductKey: "professional", BillingScope: "subject", DisplayName: "Professional", TrialDays: 7, FeaturePolicyJSON: []byte(`{"value_intelligence":true}`), LimitPolicyJSON: []byte(`{"private_watchlists":20}`), Revision: 1}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberSubscriptionRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || store.tenant != "tenant-local" || store.subject != "user-123" || !strings.Contains(recorder.Body.String(), `"access_state":"active"`) {
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
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"access_state":"unprovisioned"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	store.err = errors.New("database unavailable")
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscription", nil), fixture.token(t, nil)))
	if failed.Code != http.StatusServiceUnavailable || !strings.Contains(failed.Body.String(), "subscription_unavailable") {
		t.Fatalf("failed status=%d body=%s", failed.Code, failed.Body.String())
	}
}
