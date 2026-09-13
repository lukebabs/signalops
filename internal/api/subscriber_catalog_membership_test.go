package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberCatalogMembershipAPIFake struct {
	decision      string
	activation    string
	addCalls      int
	reservation   storage.SubscriberQuotaReservationRequest
	finalizations []storage.SubscriberQuotaReservationLifecycleRequest
}

func (f *subscriberCatalogMembershipAPIFake) GetSubscriberEntitlement(_ context.Context, tenant string) (storage.SubscriberEntitlementRecord, error) {
	return storage.SubscriberEntitlementRecord{TenantID: tenant, Status: storage.SubscriberEntitlementActive, Capabilities: []storage.SubscriberEntitlementCapabilityRecord{{Capability: "eod_activation", Enabled: true, QuotaLimit: 2}}}, nil
}

func (f *subscriberCatalogMembershipAPIFake) UpsertSubscriberEntitlement(context.Context, storage.SubscriberEntitlementRecord) (storage.SubscriberEntitlementRecord, error) {
	return storage.SubscriberEntitlementRecord{}, nil
}

func (f *subscriberCatalogMembershipAPIFake) ReserveSubscriberQuota(_ context.Context, request storage.SubscriberQuotaReservationRequest) (storage.SubscriberQuotaReservationRecord, storage.SubscriberEntitlementDecisionRecord, error) {
	f.reservation = request
	decision := f.decision
	if decision == "" {
		decision = "allowed"
	}
	record := storage.SubscriberQuotaReservationRecord{}
	if decision == "allowed" {
		record = storage.SubscriberQuotaReservationRecord{ReservationID: "subquota-test"}
	}
	return record, storage.SubscriberEntitlementDecisionRecord{DecisionReason: decision}, nil
}

func (f *subscriberCatalogMembershipAPIFake) FinalizeSubscriberQuotaReservation(_ context.Context, request storage.SubscriberQuotaReservationLifecycleRequest) (storage.SubscriberQuotaReservationRecord, error) {
	f.finalizations = append(f.finalizations, request)
	return storage.SubscriberQuotaReservationRecord{ReservationID: request.ReservationID, Status: request.Transition}, nil
}

func (f *subscriberCatalogMembershipAPIFake) AddSubscriberPrivateCatalogMembership(_ context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberCatalogMembershipResult, error) {
	f.addCalls++
	return storage.SubscriberCatalogMembershipResult{Membership: storage.SubscriberWatchlistMembershipRecord{TenantID: request.TenantID, ListID: request.ListID, GlobalAssetID: request.GlobalAssetID, AddedBySubject: request.ActorSubject}, ActivationState: f.activation}, nil
}

func TestSubscriberCatalogMembershipConsumesQuotaOnlyWhenActivationQueues(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberCatalogMembershipAPIFake{activation: "queued"}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberEntitlementRepository: store, SubscriberCatalogMembershipRepository: store})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/lists/private-1/catalog-memberships", strings.NewReader(`{"global_asset_id":"global-aapl","correlation_id":"corr-1"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, withBearer(req, fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || store.addCalls != 1 || store.reservation.Capability != "eod_activation" || store.reservation.RequestedUnits != 1 || !strings.HasPrefix(store.reservation.IdempotencyKey, "subscriber-eod-activation-v1:") {
		t.Fatalf("status=%d calls=%d reservation=%+v body=%s", recorder.Code, store.addCalls, store.reservation, recorder.Body.String())
	}
	if len(store.finalizations) != 1 || store.finalizations[0].Transition != storage.SubscriberQuotaConsumed {
		t.Fatalf("finalizations=%+v", store.finalizations)
	}
}

func TestSubscriberCatalogMembershipRejectsExhaustedQuotaBeforeMutation(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberCatalogMembershipAPIFake{decision: "deferred_quota", activation: "queued"}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberEntitlementRepository: store, SubscriberCatalogMembershipRepository: store})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/lists/private-1/catalog-memberships", strings.NewReader(`{"global_asset_id":"global-aapl"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, withBearer(req, fixture.token(t, nil)))
	if recorder.Code != http.StatusTooManyRequests || store.addCalls != 0 || !strings.Contains(recorder.Body.String(), "subscriber_activation_quota_exhausted") {
		t.Fatalf("status=%d calls=%d body=%s", recorder.Code, store.addCalls, recorder.Body.String())
	}
}

func TestSubscriberCatalogMembershipReleasesQuotaForWarmAsset(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberCatalogMembershipAPIFake{activation: "active"}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberEntitlementRepository: store, SubscriberCatalogMembershipRepository: store})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/lists/private-1/catalog-memberships", strings.NewReader(`{"global_asset_id":"global-aapl"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, withBearer(req, fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || len(store.finalizations) != 1 || store.finalizations[0].Transition != storage.SubscriberQuotaReleased {
		t.Fatalf("status=%d finalizations=%+v body=%s", recorder.Code, store.finalizations, recorder.Body.String())
	}
}
