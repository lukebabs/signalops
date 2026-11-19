package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberCoverageAPIFake struct{ added []string }

func (f *subscriberCoverageAPIFake) ListSubscriberAdminWarmCatalog(context.Context, string, string, string, string, int, int) ([]storage.SubscriberAdminWarmCatalogRecord, error) {
	return []storage.SubscriberAdminWarmCatalogRecord{{GlobalAssetID: "global-a", Ticker: "AAPL", CompanyName: "Apple", CoverageState: "active", CoverageMode: "enabled", WarmRank: 1, MarketCapRank: 1}}, nil
}

func (f *subscriberCoverageAPIFake) AddSubscriberTenantDefaultCatalogMembership(_ context.Context, request storage.SubscriberWatchlistMembershipRequest) (storage.SubscriberCatalogMembershipResult, error) {
	f.added = append(f.added, request.GlobalAssetID)
	return storage.SubscriberCatalogMembershipResult{Membership: storage.SubscriberWatchlistMembershipRecord{TenantID: request.TenantID, ListID: request.ListID, GlobalAssetID: request.GlobalAssetID}, ActivationState: "active"}, nil
}

func TestSubscriberCoverageManifestSupportsETag(t *testing.T) {
	fixture := newTestAuthFixture(t)
	watchlists := &subscriberWatchlistAPIFake{lists: []storage.SubscriberWatchlistRecord{{ListID: "default-a", TenantID: "tenant-local", ListKind: storage.SubscriberWatchlistKindTenantDefault, ListName: "Default"}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-local": {}}, SubscriberWatchlistRepository: watchlists, SubscriberCoverageRepository: &subscriberCoverageAPIFake{}})
	token := fixture.token(t, nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/coverage-manifest", nil)
	first := httptest.NewRecorder()
	router.ServeHTTP(first, withBearer(request, token))
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" || !strings.Contains(first.Body.String(), "subscriber.coverage-manifest.v1") {
		t.Fatalf("manifest status=%d etag=%q body=%s", first.Code, first.Header().Get("ETag"), first.Body.String())
	}
	secondRequest := httptest.NewRequest(http.MethodGet, "/v1/tenants/tenant-local/marketops/subscriber/coverage-manifest", nil)
	secondRequest.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	router.ServeHTTP(second, withBearer(secondRequest, token))
	if second.Code != http.StatusNotModified {
		t.Fatalf("etag status=%d body=%s", second.Code, second.Body.String())
	}
}
