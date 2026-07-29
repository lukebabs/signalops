package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type fakeCyberOpsConnectRepository struct {
	listFilter       storage.CyberOpsConnectRawFilter
	record           storage.CyberOpsConnectRawRecord
	integrityFilter  storage.CyberOpsIntegrityFailureFilter
	integrityRecord  storage.CyberOpsIntegrityFailureRecord
	resolutionActor  string
	resolutionReason string
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
func (f *fakeCyberOpsConnectRepository) ListCyberOpsIntegrityFailures(_ context.Context, filter storage.CyberOpsIntegrityFailureFilter) ([]storage.CyberOpsIntegrityFailureRecord, error) {
	f.integrityFilter = filter
	return []storage.CyberOpsIntegrityFailureRecord{f.integrityRecord}, nil
}
func (f *fakeCyberOpsConnectRepository) ResolveCyberOpsIntegrityFailure(_ context.Context, _ string, _ string, status, actor, reason string, _ time.Time) (storage.CyberOpsIntegrityFailureRecord, error) {
	f.integrityRecord.ResolutionStatus, f.resolutionActor, f.resolutionReason = status, actor, reason
	return f.integrityRecord, nil
}

func TestCyberOpsIntegrityResolutionCapturesActorReasonAndTenant(t *testing.T) {
	repo := &fakeCyberOpsConnectRepository{integrityRecord: storage.CyberOpsIntegrityFailureRecord{FailureID: "cyberint-1", TenantID: "tenant-local", ResolutionStatus: "open"}}
	router := NewRouter(RouterConfig{CyberOpsConnectRepository: repo})
	request := httptest.NewRequest(http.MethodPost, "/v1/cyberops/integrity-failures/cyberint-1/resolve?tenant_id=tenant-local", bytes.NewBufferString(`{"resolution_status":"resolved_false_positive","reason":"canonical JSONB comparison repair"}`))
	request.Header.Set("X-SignalOps-Actor", "operator-test")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repo.resolutionActor != "operator-test" || repo.resolutionReason != "canonical JSONB comparison repair" || repo.integrityRecord.ResolutionStatus != "resolved_false_positive" {
		t.Fatalf("resolution actor=%q reason=%q record=%+v", repo.resolutionActor, repo.resolutionReason, repo.integrityRecord)
	}
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/v1/cyberops/integrity-failures?tenant_id=tenant-local&resolution_status=resolved_false_positive", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	if repo.integrityFilter.TenantID != "tenant-local" || repo.integrityFilter.ResolutionStatus != "resolved_false_positive" {
		t.Fatalf("integrity filter=%+v", repo.integrityFilter)
	}
}
