package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func (q *fakeQueryRepository) ListMarketOpsOpportunities(_ context.Context, filter storage.MarketOpsOpportunityFilter) ([]storage.MarketOpsOpportunityRecord, error) {
	q.lastOpportunityFilter = filter
	return q.marketOpsOpportunities, nil
}

func (q *fakeQueryRepository) GetMarketOpsOpportunity(_ context.Context, tenantID, opportunityID string) (storage.MarketOpsOpportunityRecord, error) {
	for _, record := range q.marketOpsOpportunities {
		if record.TenantID == tenantID && record.OpportunityID == opportunityID {
			return record, nil
		}
	}
	return storage.MarketOpsOpportunityRecord{}, storage.ErrNotFound
}

func TestMarketOpsOpportunityReadAPIs(t *testing.T) {
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	record := storage.MarketOpsOpportunityRecord{
		OpportunityID: "mopp-1", TenantID: "tenant-1", AppID: "marketops", AssetID: "ticker:AAPL", Symbol: "AAPL",
		OpenedSessionDate: now, LastEvaluatedDate: now, Direction: "downside", Horizon: "5_to_20_sessions",
		LifecycleStatus: storage.MarketOpsOpportunityActive, OpportunityScore: .8, ConfidenceScore: .75,
		DomainDiversityScore: .67, HypothesisEvaluationIDs: []string{"eval-1", "eval-2"},
		Summary: "AAPL downside opportunity.", OpportunityPayloadJSON: []byte(`{"research_only":true}`),
		Version: 1, ResearchOnly: true, BuildRunID: "run-1", DeterministicKey: "key-1",
	}
	repo := &fakeQueryRepository{marketOpsOpportunities: []storage.MarketOpsOpportunityRecord{record}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	for _, path := range []string{
		"/v1/marketops/opportunities?tenant_id=tenant-1&symbol=aapl&direction=downside&lifecycle_status=active&research_only=true&session_start=2026-07-01&session_end=2026-07-31",
		"/v1/marketops/opportunities/mopp-1?tenant_id=tenant-1",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
	if repo.lastOpportunityFilter.Symbol != "aapl" || repo.lastOpportunityFilter.ResearchOnly == nil || !*repo.lastOpportunityFilter.ResearchOnly || repo.lastOpportunityFilter.LifecycleStatus != "active" {
		t.Fatalf("filter=%+v", repo.lastOpportunityFilter)
	}
}

func TestMarketOpsOpportunityReadsRejectInvalidQueries(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	for _, path := range []string{
		"/v1/marketops/opportunities/mopp-1",
		"/v1/marketops/opportunities?research_only=maybe",
		"/v1/marketops/opportunities?session_start=not-a-date",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}
