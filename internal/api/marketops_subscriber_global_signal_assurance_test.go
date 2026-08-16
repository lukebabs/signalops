package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type subscriberGlobalSignalAssuranceFake struct {
	*signalAssuranceTenantBindingRepository
	symbols []string
}

func (f *subscriberGlobalSignalAssuranceFake) ListSubscriberGlobalSignalAssuranceEffectiveness(_ context.Context, symbols []string, _ storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
	f.symbols = append([]string(nil), symbols...)
	accuracy := 1.0
	return []storage.SignalAssuranceEffectivenessRecord{{EvidenceSource: "LEGACY", Dimension: "overall", DimensionValue: "all", SampleSize: 1, DirectionalHits: 1, DirectionalAccuracy: &accuracy, AsOf: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), MetricDefinitionVersion: "saf_effectiveness.v1"}}, nil
}

func (f *subscriberGlobalSignalAssuranceFake) ListSubscriberGlobalSignalAssuranceEffectivenessObservations(context.Context, []string, storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessObservationRecord, error) {
	return []storage.SignalAssuranceEffectivenessObservationRecord{}, nil
}

func (f *subscriberGlobalSignalAssuranceFake) ListSubscriberGlobalSignalAssuranceRecommendations(context.Context, []string, storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceRecommendationRecord, error) {
	return []storage.SignalAssuranceRecommendationRecord{}, nil
}

func TestSubscriberSignalAssuranceUsesGlobalWatchlistEvidence(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &subscriberGlobalSignalAssuranceFake{signalAssuranceTenantBindingRepository: &signalAssuranceTenantBindingRepository{fakeQueryRepository: &fakeQueryRepository{}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo, SubscriberListsEnabled: true, SubscriberListsPilotTenants: map[string]struct{}{"tenant-pilot-b": {}}, SubscriberWatchlistRepository: &subscriberWatchlistAPIFake{}})
	request := httptest.NewRequest(http.MethodGet, "/v1/marketops/signal-assurance/effectiveness?tenant_id=tenant-pilot-b", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(request, fixture.token(t, map[string]any{"tenant_id": "tenant-pilot-b"})))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.symbols) != 1 || repo.symbols[0] != "AAPL" {
		t.Fatalf("global Signal Assurance symbols=%v, want [AAPL]", repo.symbols)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["data_scope"] != "platform-global" || len(response["effectiveness"].([]any)) != 1 {
		t.Fatalf("subscriber Signal Assurance did not use global evidence: %#v", response)
	}
}
