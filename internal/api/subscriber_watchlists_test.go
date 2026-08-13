package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberWatchlistAPIFake struct {
	lastTenant, lastSubject, lastMutation string
}

func (f *subscriberWatchlistAPIFake) CreateSubscriberPrivateWatchlist(_ context.Context, request storage.SubscriberWatchlistCreateRequest) (storage.SubscriberWatchlistRecord, error) {
	f.lastTenant, f.lastSubject = request.TenantID, request.ActorSubject
	return storage.SubscriberWatchlistRecord{ListID: "private-a", TenantID: request.TenantID, ListKind: storage.SubscriberWatchlistKindPrivate, OwnerSubject: request.ActorSubject, ListName: request.ListName}, nil
}
func (f *subscriberWatchlistAPIFake) CreateSubscriberTenantDefaultWatchlist(_ context.Context, request storage.SubscriberWatchlistCreateRequest) (storage.SubscriberWatchlistRecord, error) {
	f.lastTenant, f.lastSubject = request.TenantID, request.ActorSubject
	return storage.SubscriberWatchlistRecord{ListID: "default-a", TenantID: request.TenantID, ListKind: storage.SubscriberWatchlistKindTenantDefault, ListName: request.ListName}, nil
}
func (f *subscriberWatchlistAPIFake) ListSubscriberWatchlists(_ context.Context, tenantID, subject string) ([]storage.SubscriberWatchlistRecord, error) {
	f.lastTenant, f.lastSubject = tenantID, subject
	return []storage.SubscriberWatchlistRecord{{ListID: "private-a", TenantID: tenantID, ListKind: storage.SubscriberWatchlistKindPrivate, OwnerSubject: subject, ListName: "Private"}}, nil
}
func (f *subscriberWatchlistAPIFake) GetSubscriberWatchlistContextPreference(context.Context, string, string) (storage.SubscriberWatchlistContextPreference, error) { return storage.SubscriberWatchlistContextPreference{}, storage.ErrNotFound }
func (f *subscriberWatchlistAPIFake) SetSubscriberWatchlistContextPreference(_ context.Context, preference storage.SubscriberWatchlistContextPreference) (storage.SubscriberWatchlistContextPreference, error) { return preference, nil }
func (f *subscriberWatchlistAPIFake) ListSubscriberWatchlistMemberships(context.Context, string, string, string) ([]storage.SubscriberWatchlistMembershipRecord, error) {
	return nil, nil
}
func (f *subscriberWatchlistAPIFake) ListSubscriberWatchlistItems(_ context.Context, tenantID, subject, listID string) ([]storage.SubscriberWatchlistItemRecord, error) {
	f.lastTenant, f.lastSubject = tenantID, subject
	return []storage.SubscriberWatchlistItemRecord{{TenantID: tenantID, ListID: listID, ListKind: storage.SubscriberWatchlistKindTenantDefault, ListName: "Default", GlobalAssetID: "global-a", Ticker: "AAPL", CompanyName: "Apple", EligibilityStatus: "eligible", CoverageState: "active", CoverageMode: "shadow"}}, nil
}
func (f *subscriberWatchlistAPIFake) AddSubscriberPrivateWatchlistMembership(_ context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberWatchlistMembershipRecord, error) {
	return storage.SubscriberWatchlistMembershipRecord{TenantID: request.TenantID, ListID: request.ListID, GlobalAssetID: request.GlobalAssetID, AddedBySubject: request.ActorSubject}, nil
}
func (f *subscriberWatchlistAPIFake) AddSubscriberTenantDefaultWatchlistMembership(_ context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberWatchlistMembershipRecord, error) {
	return storage.SubscriberWatchlistMembershipRecord{TenantID: request.TenantID, ListID: request.ListID, GlobalAssetID: request.GlobalAssetID, AddedBySubject: request.ActorSubject}, nil
}
func (f *subscriberWatchlistAPIFake) RemoveSubscriberPrivateWatchlistMembership(_ context.Context, request storage.SubscriberWatchlistMembershipRequest) error {
	f.lastTenant, f.lastSubject, f.lastMutation = request.TenantID, request.ActorSubject, "remove_private"
	return nil
}
func (f *subscriberWatchlistAPIFake) RemoveSubscriberTenantDefaultWatchlistMembership(context.Context, storage.SubscriberWatchlistMembershipRequest) error {
	return nil
}

func TestSubscriberWatchlistRoutesAreDisabledByDefault(t *testing.T) {
	router := NewRouter(RouterConfig{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/lists", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("disabled route status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberWatchlistRoutesBindTenantAndSubject(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberWatchlistAPIFake{}
	router := NewRouter(RouterConfig{
		Auth: fixture.authCfg, SubscriberListsEnabled: true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-local": {}},
		SubscriberWatchlistRepository: store,
	})
	token := fixture.token(t, nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/lists", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastTenant != "tenant-local" || store.lastSubject != "user-123" {
		t.Fatalf("repository scope tenant=%q subject=%q", store.lastTenant, store.lastSubject)
	}

	foreign := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-other/marketops/subscriber/lists", nil)
	foreignRecorder := httptest.NewRecorder()
	router.ServeHTTP(foreignRecorder, withBearer(foreign, token))
	if foreignRecorder.Code != http.StatusForbidden || !strings.Contains(foreignRecorder.Body.String(), "tenant_mismatch") {
		t.Fatalf("foreign status=%d body=%s", foreignRecorder.Code, foreignRecorder.Body.String())
	}
}

func TestSubscriberWatchlistDefaultMutationRequiresAdmin(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{
		Auth: fixture.authCfg, SubscriberListsEnabled: true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-local": {}},
		SubscriberWatchlistRepository: &subscriberWatchlistAPIFake{},
	})
	token := fixture.token(t, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/tenant-default-list", strings.NewReader(`{"list_name":"Default"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "tenant_admin_required") {
		t.Fatalf("default mutation status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubscriberWatchlistPrivateMembershipRemovalBindsSubject(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberWatchlistAPIFake{}
	router := NewRouter(RouterConfig{
		Auth: fixture.authCfg, SubscriberListsEnabled: true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-local": {}},
		SubscriberWatchlistRepository: store,
	})
	token := fixture.token(t, nil)
	request := httptest.NewRequest(http.MethodDelete, "/v1/tenants/tenant-local/marketops/subscriber/lists/private-a/memberships/global-a?list_kind=private", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("private removal status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastTenant != "tenant-local" || store.lastSubject != "user-123" || store.lastMutation != "remove_private" {
		t.Fatalf("private removal scope tenant=%q subject=%q mutation=%q", store.lastTenant, store.lastSubject, store.lastMutation)
	}
}

func TestSubscriberWatchlistPrivateMutationAllowsMarketOpsReadGrant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberWatchlistAPIFake{}
	access := &accessManagementTestRepository{subjectAccess: []storage.TenantUserAccessRecord{{
		TenantID: "tenant-local", Subject: "user-123", AppID: "marketops", Permission: "read",
	}}}
	router := NewRouter(RouterConfig{
		Auth:                          fixture.authCfg,
		AccessRepository:              access,
		SubscriberListsEnabled:        true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-local": {}},
		SubscriberWatchlistRepository: store,
	})
	request := httptest.NewRequest(http.MethodPost, "/v1/tenants/tenant-local/marketops/subscriber/private-lists", strings.NewReader(`{"list_name":"Research"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, fixture.token(t, nil)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("private mutation status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastTenant != "tenant-local" || store.lastSubject != "user-123" {
		t.Fatalf("repository scope tenant=%q subject=%q", store.lastTenant, store.lastSubject)
	}
}

func TestSubscriberWatchlistItemsRouteBindsTenantAndSubject(t *testing.T) {
	fixture := newTestAuthFixture(t)
	store := &subscriberWatchlistAPIFake{}
	router := NewRouter(RouterConfig{
		Auth:                          fixture.authCfg,
		SubscriberListsEnabled:        true,
		SubscriberListsPilotTenants:   map[string]struct{}{"tenant-local": {}},
		SubscriberWatchlistRepository: store,
	})
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/lists/default-a/items", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, fixture.token(t, nil)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"ticker":"AAPL"`) {
		t.Fatalf("items status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastTenant != "tenant-local" || store.lastSubject != "user-123" {
		t.Fatalf("repository scope tenant=%q subject=%q", store.lastTenant, store.lastSubject)
	}
}
