package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type accessManagementTestRepository struct {
	lastListTenant string
	lastUpsert     storage.TenantUserAccessRecord
	upsertCount    int
}

func (r *accessManagementTestRepository) ListTenantUserAccess(_ context.Context, tenantID string) ([]storage.TenantUserAccessRecord, error) {
	r.lastListTenant = tenantID
	return nil, nil
}

func (*accessManagementTestRepository) ListTenantUserAccessForSubject(context.Context, string, string) ([]storage.TenantUserAccessRecord, error) {
	return nil, nil
}

func (r *accessManagementTestRepository) UpsertTenantUserAccess(_ context.Context, record storage.TenantUserAccessRecord, _, _ string) (storage.TenantUserAccessRecord, error) {
	r.lastUpsert = record
	r.upsertCount++
	return record, nil
}

func (*accessManagementTestRepository) DeleteTenantUserAccess(context.Context, string, string, string, string, string) error {
	return nil
}

func (*accessManagementTestRepository) ListTenantUserAccessAudit(context.Context, string, string, int) ([]storage.TenantUserAccessAuditRecord, error) {
	return nil, nil
}

func TestAccessGrantBindsOmittedBodyTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/access", strings.NewReader(`{"subject":"subscriber-1","display_name":"Subscriber","app_id":"marketops","permission":"read"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.upsertCount != 1 {
		t.Fatalf("upsert count = %d", repo.upsertCount)
	}
	if repo.lastUpsert.TenantID != "tenant-local" {
		t.Fatalf("tenant id = %q", repo.lastUpsert.TenantID)
	}
	if repo.lastUpsert.GrantedBy != "user-123" {
		t.Fatalf("granted by = %q", repo.lastUpsert.GrantedBy)
	}
}

func TestAccessGrantRejectsBodyTenantMismatch(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})
	request := httptest.NewRequest(http.MethodPut, "/v1/administration/access", strings.NewReader(`{"tenant_id":"tenant-other","subject":"subscriber-1","display_name":"Subscriber","app_id":"marketops","permission":"read"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "tenant_mismatch") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if repo.upsertCount != 0 {
		t.Fatalf("unexpected upsert count = %d", repo.upsertCount)
	}
}

func TestAccessListBindsOmittedQueryTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &accessManagementTestRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, AccessRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/administration/access", nil), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastListTenant != "tenant-local" {
		t.Fatalf("listed tenant = %q", repo.lastListTenant)
	}
}
