package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeCyberOpsConnectRepository struct {
	listFilter storage.CyberOpsConnectRawFilter
	record     storage.CyberOpsConnectRawRecord
}

func (f *fakeCyberOpsConnectRepository) PersistCyberOpsConnectRaw(context.Context, storage.CyberOpsConnectRawRecord, storage.CyberOpsOutboxRecord, []byte) (storage.CyberOpsPersistResult, error) {
	return storage.CyberOpsPersistResult{}, nil
}
func (f *fakeCyberOpsConnectRepository) ListCyberOpsConnectRaw(_ context.Context, filter storage.CyberOpsConnectRawFilter) ([]storage.CyberOpsConnectRawRecord, error) {
	f.listFilter = filter
	return []storage.CyberOpsConnectRawRecord{f.record}, nil
}
func (f *fakeCyberOpsConnectRepository) GetCyberOpsConnectRaw(context.Context, string, string) (storage.CyberOpsConnectRawRecord, error) {
	return f.record, nil
}
func (f *fakeCyberOpsConnectRepository) ListPendingCyberOpsOutbox(context.Context, int) ([]storage.CyberOpsOutboxRecord, error) {
	return nil, nil
}
func (f *fakeCyberOpsConnectRepository) MarkCyberOpsOutboxPublished(context.Context, string, time.Time) error {
	return nil
}
func (f *fakeCyberOpsConnectRepository) MarkCyberOpsOutboxAttempt(context.Context, string) error {
	return nil
}

func TestCyberOpsEventsUsesTenantScopeAndBoundsSearch(t *testing.T) {
	repo := &fakeCyberOpsConnectRepository{record: storage.CyberOpsConnectRawRecord{ConnectIngressEventID: "ing-1", EventID: "ing-1", OccurredAt: time.Now().UTC()}}
	router := NewRouter(RouterConfig{CyberOpsConnectRepository: repo})
	request := httptest.NewRequest(http.MethodGet, "/v1/cyberops/events?tenant_id=tenant-local&from=2026-07-01T00:00:00Z&to=2026-07-02T00:00:00Z&search=blocked&limit=500", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repo.listFilter.TenantID != "tenant-local" || repo.listFilter.Limit != 200 || repo.listFilter.Search != "blocked" {
		t.Fatalf("filter=%+v", repo.listFilter)
	}
}
func TestCyberOpsEventsRejectsUnboundedSearch(t *testing.T) {
	router := NewRouter(RouterConfig{CyberOpsConnectRepository: &fakeCyberOpsConnectRepository{}})
	request := httptest.NewRequest(http.MethodGet, "/v1/cyberops/events?tenant_id=tenant-local&search=blocked", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}
