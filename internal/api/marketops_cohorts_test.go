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

func (q *fakeQueryRepository) UpsertMarketOpsIntelligenceCohortRun(_ context.Context, x storage.MarketOpsIntelligenceCohortRunRecord) error {
	q.cohortRuns = append(q.cohortRuns, x)
	return nil
}
func (q *fakeQueryRepository) UpsertMarketOpsIntelligenceCohortSymbolResult(_ context.Context, x storage.MarketOpsIntelligenceCohortSymbolResultRecord) error {
	q.cohortResults = append(q.cohortResults, x)
	return nil
}
func (q *fakeQueryRepository) ListMarketOpsIntelligenceCohortRuns(_ context.Context, _ storage.MarketOpsIntelligenceCohortRunFilter) ([]storage.MarketOpsIntelligenceCohortRunRecord, error) {
	return q.cohortRuns, nil
}
func (q *fakeQueryRepository) GetMarketOpsIntelligenceCohortRun(_ context.Context, tenantID, runID string) (storage.MarketOpsIntelligenceCohortRunRecord, error) {
	for _, x := range q.cohortRuns {
		if x.TenantID == tenantID && x.RunID == runID {
			return x, nil
		}
	}
	return storage.MarketOpsIntelligenceCohortRunRecord{}, storage.ErrNotFound
}
func (q *fakeQueryRepository) ListMarketOpsIntelligenceCohortSymbolResults(_ context.Context, tenantID, runID string) ([]storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	out := []storage.MarketOpsIntelligenceCohortSymbolResultRecord{}
	for _, x := range q.cohortResults {
		if x.TenantID == tenantID && x.RunID == runID {
			out = append(out, x)
		}
	}
	return out, nil
}
func (q *fakeQueryRepository) ListMarketOpsIntelligenceReadiness(_ context.Context, f storage.MarketOpsIntelligenceReadinessFilter) ([]storage.MarketOpsIntelligenceCohortSymbolResultRecord, error) {
	out := []storage.MarketOpsIntelligenceCohortSymbolResultRecord{}
	for _, x := range q.cohortResults {
		if x.TenantID == f.TenantID {
			out = append(out, x)
		}
	}
	return out, nil
}

func TestMarketOpsIntelligenceReadinessIsAggregateFirstAndFailClosed(t *testing.T) {
	session := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{cohortResults: []storage.MarketOpsIntelligenceCohortSymbolResultRecord{
		{ResultID: "cohortres-aapl", RunID: "cohort-1", TenantID: "tenant-1", UniverseGroup: "top50_megacap", Symbol: "AAPL", LatestMarketStateID: "mstate-1", LatestStateDate: &session, LatestStateQuality: "partial", LatestStateCompleteness: .14, CoverageState: "incomplete", EvaluationState: "blocked", GovernanceState: "research_only", CalibrationState: "below_minimum", OutcomeState: "unavailable", RolloutStatus: "blocked", ReadinessReasons: []string{"state quality blocks evaluation"}},
		{ResultID: "cohortres-msft", RunID: "cohort-1", TenantID: "tenant-1", UniverseGroup: "top50_megacap", Symbol: "MSFT", CoverageState: "unavailable", EvaluationState: "not_run", GovernanceState: "research_only", CalibrationState: "unavailable", OutcomeState: "unavailable", RolloutStatus: "not_observed", ReadinessReasons: []string{"no persisted market state"}},
	}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketops/intelligence/readiness?tenant_id=tenant-1&symbols=AAPL,MSFT&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	readiness := body["readiness"]
	aggregate := readiness["aggregate"].(map[string]any)
	if aggregate["production_ready_supported"] != false || aggregate["symbol_count"].(float64) != 2 {
		t.Fatalf("aggregate=%#v", aggregate)
	}
	rows := readiness["symbols"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows=%#v", rows)
	}
}
