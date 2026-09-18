package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type administrationTenantBindingRepository struct {
	*fakeQueryRepository
	lastNotificationFilter storage.AdministrationNotificationFilter
	lastSMTPGetTenant      string
	lastSMTPUpsert         storage.AdministrationSMTPSettings
	smtpUpsertCount        int
}

func (*administrationTenantBindingRepository) UpsertAdministrationNotification(context.Context, storage.AdministrationNotificationRecord) (storage.AdministrationNotificationRecord, error) {
	return storage.AdministrationNotificationRecord{}, nil
}

func (r *administrationTenantBindingRepository) ListAdministrationNotifications(_ context.Context, filter storage.AdministrationNotificationFilter) ([]storage.AdministrationNotificationRecord, error) {
	r.lastNotificationFilter = filter
	return []storage.AdministrationNotificationRecord{{NotificationID: "notification-1"}}, nil
}

func (*administrationTenantBindingRepository) SetAdministrationNotificationInboxState(context.Context, storage.AdministrationNotificationInboxState) error {
	return nil
}

func (r *administrationTenantBindingRepository) GetAdministrationSMTPSettings(_ context.Context, tenantID string) (storage.AdministrationSMTPSettings, error) {
	r.lastSMTPGetTenant = tenantID
	return storage.AdministrationSMTPSettings{}, storage.ErrNotFound
}

func (r *administrationTenantBindingRepository) UpsertAdministrationSMTPSettings(_ context.Context, settings storage.AdministrationSMTPSettings) (storage.AdministrationSMTPSettings, error) {
	r.lastSMTPUpsert = settings
	r.smtpUpsertCount++
	return settings, nil
}

func TestAdministrationNotificationsBindOmittedTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &administrationTenantBindingRepository{fakeQueryRepository: &fakeQueryRepository{}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/administration/notifications", nil), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastNotificationFilter.TenantID != "tenant-local" {
		t.Fatalf("notification tenant = %q", repo.lastNotificationFilter.TenantID)
	}
}

func TestAdministrationNotificationStateBindsOmittedTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &administrationTenantBindingRepository{fakeQueryRepository: &fakeQueryRepository{}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})

	request := httptest.NewRequest(http.MethodPost, "/v1/administration/notifications/notification-1/state", strings.NewReader(`{"read":true}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastNotificationFilter.TenantID != "tenant-local" {
		t.Fatalf("notification state tenant = %q", repo.lastNotificationFilter.TenantID)
	}
}

func TestAdministrationSMTPBindsOmittedTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &administrationTenantBindingRepository{fakeQueryRepository: &fakeQueryRepository{}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})

	request := httptest.NewRequest(http.MethodPut, "/v1/administration/notification-email", strings.NewReader(`{"host":"smtp.example.test","port":587,"from_email":"ops@example.test","use_starttls":true}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastSMTPUpsert.TenantID != "tenant-local" {
		t.Fatalf("smtp tenant = %q", repo.lastSMTPUpsert.TenantID)
	}
}

func TestAdministrationSMTPReadBindsOmittedTenantToAuthenticatedPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &administrationTenantBindingRepository{fakeQueryRepository: &fakeQueryRepository{}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/administration/notification-email", nil), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastSMTPGetTenant != "tenant-local" {
		t.Fatalf("smtp read tenant = %q", repo.lastSMTPGetTenant)
	}
}
