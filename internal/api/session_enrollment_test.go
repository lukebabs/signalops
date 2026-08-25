package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSessionEnrollmentAllowsAuthenticatedUserWithoutExistingAccess(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo, SubscriberB2CTenantID: "tenant-b2c"})
	token := fixture.token(t, map[string]any{
		"tenant_id":      "tenant-b2c",
		"email_verified": false,
		"realm_access":   map[string]any{"roles": []string{}},
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"email_verification_required"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.upsertCount != 0 {
		t.Fatalf("unexpected access provisioning count = %d", repo.upsertCount)
	}
}

func TestSessionEnrollmentProvisionsB2CExplorerAccessAfterEmailVerification(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo, SubscriberB2CTenantID: "tenant-b2c"})
	token := fixture.token(t, map[string]any{
		"tenant_id":      "tenant-b2c",
		"email_verified": true,
		"realm_access":   map[string]any{"roles": []string{}},
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"marketops_ready"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.upsertCount != 1 {
		t.Fatalf("access provisioning count = %d", repo.upsertCount)
	}
	if repo.lastUpsert.TenantID != "tenant-b2c" || repo.lastUpsert.Subject != "user-123" || repo.lastUpsert.AppID != "marketops" || repo.lastUpsert.Permission != "read" {
		t.Fatalf("provisioned access = %+v", repo.lastUpsert)
	}
	if !strings.Contains(recorder.Body.String(), `"marketops":"read"`) || !strings.Contains(recorder.Body.String(), `"marketops_access"`) {
		t.Fatalf("body=%s", recorder.Body.String())
	}
}

func TestSessionEnrollmentDoesNotSelfProvisionNonB2CTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo, SubscriberB2CTenantID: "tenant-b2c"})
	token := fixture.token(t, map[string]any{"email_verified": true})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"tenant_access_missing"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.upsertCount != 0 {
		t.Fatalf("unexpected access provisioning count = %d", repo.upsertCount)
	}
}

func TestSessionEnrollmentDoesNotRequireSubscriptionWhenEnforcementDisabled(t *testing.T) {
	fixture := newTestAuthFixture(t)
	accessRepo := &accessManagementTestRepository{subjectAccess: []storage.TenantUserAccessRecord{{TenantID: "tenant-local", Subject: "user-123", AppID: "marketops", Permission: "read"}}}
	subscriptionRepo := &subscriberSubscriptionAPIFake{err: storage.ErrNotFound}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: accessRepo, SubscriberSubscriptionRepository: subscriptionRepo, SubscriberB2CTenantID: "tenant-b2c"})
	token := fixture.token(t, map[string]any{"email_verified": true})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"marketops_ready"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
