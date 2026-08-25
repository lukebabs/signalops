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

func TestSessionEnrollmentRequiresSubscriptionForB2CWhenEnforcementEnabled(t *testing.T) {
	fixture := newTestAuthFixture(t)
	accessRepo := &accessManagementTestRepository{}
	subscriptionRepo := &subscriberSubscriptionAPIFake{err: storage.ErrNotFound}
	adminRepo := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: accessRepo, SubscriberSubscriptionRepository: subscriptionRepo, SubscriberSubscriptionAdministrationRepository: adminRepo, SubscriberSubscriptionsEnabled: true, SubscriberB2CTenantID: "tenant-b2c"})
	token := fixture.token(t, map[string]any{"tenant_id": "tenant-b2c", "email_verified": true})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"state":"subscription_missing"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if accessRepo.upsertCount != 1 {
		t.Fatalf("access provisioning count = %d", accessRepo.upsertCount)
	}
	if adminRepo.subject.ProductKey != "" {
		t.Fatalf("unexpected subscription auto-activation: %+v", adminRepo.subject)
	}
	if !strings.Contains(recorder.Body.String(), `"marketops_access"`) || strings.Contains(recorder.Body.String(), `"explorer_subscription"`) {
		t.Fatalf("body=%s", recorder.Body.String())
	}
}

func TestSessionEnrollmentCanAutoActivateExplorerWhenExplicitlyEnabled(t *testing.T) {
	fixture := newTestAuthFixture(t)
	accessRepo := &accessManagementTestRepository{subjectAccess: []storage.TenantUserAccessRecord{{TenantID: "tenant-b2c", Subject: "user-123", AppID: "marketops", Permission: "read"}}}
	subscriptionRepo := &subscriberSubscriptionAPIFake{err: storage.ErrNotFound}
	adminRepo := &subscriberSubscriptionAdministrationFake{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: accessRepo, SubscriberSubscriptionRepository: subscriptionRepo, SubscriberSubscriptionAdministrationRepository: adminRepo, SubscriberSubscriptionsEnabled: true, SubscriberB2CTenantID: "tenant-b2c", SubscriberB2CAutoActivateExplorer: true})
	token := fixture.token(t, map[string]any{"tenant_id": "tenant-b2c", "email_verified": true})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/session/enrollment", nil), token))
	if adminRepo.subject.TenantID != "tenant-b2c" || adminRepo.subject.Subject != "user-123" || adminRepo.subject.ProductKey != "explorer" || adminRepo.subject.Status != storage.SubscriberSubscriptionActive {
		t.Fatalf("expected explicit explorer auto-activation, got %+v", adminRepo.subject)
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"explorer_subscription"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
