package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberCatalogAPIFake struct {
	entitled bool
	tenant   string
}

func (f *subscriberCatalogAPIFake) GetSubscriberEntitlement(_ context.Context, tenant string) (storage.SubscriberEntitlementRecord, error) {
	f.tenant = tenant
	return storage.SubscriberEntitlementRecord{TenantID: tenant, Status: storage.SubscriberEntitlementActive, Capabilities: []storage.SubscriberEntitlementCapabilityRecord{{Capability: "catalog_search", Enabled: f.entitled, QuotaLimit: 10}}}, nil
}
func (f *subscriberCatalogAPIFake) UpsertSubscriberEntitlement(context.Context, storage.SubscriberEntitlementRecord) (storage.SubscriberEntitlementRecord, error) {
	return storage.SubscriberEntitlementRecord{}, nil
}
func (f *subscriberCatalogAPIFake) ReserveSubscriberQuota(context.Context, storage.SubscriberQuotaReservationRequest) (storage.SubscriberQuotaReservationRecord, storage.SubscriberEntitlementDecisionRecord, error) {
	return storage.SubscriberQuotaReservationRecord{}, storage.SubscriberEntitlementDecisionRecord{}, nil
}
func (f *subscriberCatalogAPIFake) FinalizeSubscriberQuotaReservation(context.Context, storage.SubscriberQuotaReservationLifecycleRequest) (storage.SubscriberQuotaReservationRecord, error) {
	return storage.SubscriberQuotaReservationRecord{}, nil
}

func (f *subscriberCatalogAPIFake) SearchSubscriberCatalog(_ context.Context, tenant, _ string, _ int) ([]storage.SubscriberCatalogProjectionRecord, error) {
	f.tenant = tenant
	return []storage.SubscriberCatalogProjectionRecord{{GlobalAssetID: "global-aapl", Ticker: "AAPL", CompanyName: "Apple", EligibilityStatus: "eligible", CoverageState: "active", CoverageMode: "shadow"}}, nil
}

func TestSubscriberCatalogSearchRequiresEntitlementAndTenantBinding(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberCatalogAPIFake{entitled: true}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberCatalogRepository: store, SubscriberEntitlementRepository: store})
	token := fixture.token(t, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/catalog?q=AAPL", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "global-aapl") || store.tenant != "tenant-local" {
		t.Fatalf("status=%d body=%s tenant=%s", recorder.Code, recorder.Body.String(), store.tenant)
	}
	foreign := httptest.NewRecorder()
	router.ServeHTTP(foreign, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-other/marketops/subscriber/catalog?q=AAPL", nil), token))
	if foreign.Code != http.StatusForbidden || !strings.Contains(foreign.Body.String(), "tenant_mismatch") {
		t.Fatalf("foreign status=%d body=%s", foreign.Code, foreign.Body.String())
	}
}

func TestSubscriberCatalogSearchFailsClosedWithoutCapability(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberCatalogAPIFake{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberCatalogRepository: store, SubscriberEntitlementRepository: store})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/catalog?q=AAPL", nil), fixture.token(t, nil)))
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "subscriber_catalog_not_entitled") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
