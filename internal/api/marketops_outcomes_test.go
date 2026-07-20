package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (q *fakeQueryRepository) ListMarketOpsSignalOutcomes(_ context.Context, filter storage.MarketOpsSignalOutcomeFilter) ([]storage.MarketOpsSignalOutcomeRecord, error) {
	q.lastOutcomeFilter = filter
	return q.marketOpsOutcomes, nil
}

func (q *fakeQueryRepository) GetMarketOpsSignalOutcome(_ context.Context, tenantID, outcomeID string) (storage.MarketOpsSignalOutcomeRecord, error) {
	for _, record := range q.marketOpsOutcomes {
		if record.TenantID == tenantID && record.OutcomeID == outcomeID {
			return record, nil
		}
	}
	return storage.MarketOpsSignalOutcomeRecord{}, storage.ErrNotFound
}

func TestMarketOpsOutcomeReadAPIs(t *testing.T) {
	origin := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	matured := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
	value, hit := .04, true
	record := storage.MarketOpsSignalOutcomeRecord{
		OutcomeID: "moutcome-1", TenantID: "tenant-1", AppID: "marketops",
		SourceType: storage.MarketOpsOutcomeSourceHypothesisEvaluation, SourceID: "eval-1",
		HypothesisKey: "H001", HypothesisVersion: "v1", AssetID: "ticker:AAPL", Symbol: "AAPL",
		Direction: "upside", OriginSessionDate: origin, HorizonSessions: 5, MaturedSessionDate: &matured,
		OutcomeStatus: storage.MarketOpsOutcomeMatured, ForwardReturn: &value, DirectionalHit: &hit,
		OutcomeEventIDs: []string{"evt-1"}, OutcomePayloadJSON: []byte(`{"threshold":0.02}`),
		CalculationVersion: "marketops.forward_outcome.v1", CalculationRunID: "run-1", DeterministicKey: "key-1",
	}
	repo := &fakeQueryRepository{marketOpsOutcomes: []storage.MarketOpsSignalOutcomeRecord{record}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	for _, path := range []string{
		"/v1/marketops/outcomes?tenant_id=tenant-1&source_type=hypothesis_evaluation&hypothesis_key=H001&symbol=aapl&direction=upside&outcome_status=matured&horizon_sessions=5&session_start=2026-07-01&session_end=2026-07-31",
		"/v1/marketops/outcomes/moutcome-1?tenant_id=tenant-1",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
	if repo.lastOutcomeFilter.Symbol != "aapl" || repo.lastOutcomeFilter.HorizonSessions != 5 || repo.lastOutcomeFilter.OutcomeStatus != "matured" {
		t.Fatalf("filter=%+v", repo.lastOutcomeFilter)
	}
}

func TestMarketOpsOutcomeReadsRejectInvalidQueries(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	for _, path := range []string{
		"/v1/marketops/outcomes/moutcome-1",
		"/v1/marketops/outcomes?horizon_sessions=3",
		"/v1/marketops/outcomes?session_start=not-a-date",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}
