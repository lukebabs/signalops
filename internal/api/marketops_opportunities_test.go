package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (q *fakeQueryRepository) InsertMarketOpsOpportunityDisposition(_ context.Context, record storage.MarketOpsOpportunityDispositionRecord) error {
	for _, opportunity := range q.marketOpsOpportunities {
		if opportunity.TenantID == record.TenantID && opportunity.OpportunityID == record.OpportunityID {
			q.marketOpsOpportunityDispositions = append(q.marketOpsOpportunityDispositions, record)
			return nil
		}
	}
	return storage.ErrNotFound
}

func (q *fakeQueryRepository) ListMarketOpsOpportunityDispositions(_ context.Context, filter storage.MarketOpsOpportunityDispositionFilter) ([]storage.MarketOpsOpportunityDispositionRecord, error) {
	out := []storage.MarketOpsOpportunityDispositionRecord{}
	for _, record := range q.marketOpsOpportunityDispositions {
		if record.TenantID == filter.TenantID && (filter.OpportunityID == "" || record.OpportunityID == filter.OpportunityID) && (filter.Disposition == "" || record.Disposition == filter.Disposition) {
			out = append(out, record)
		}
	}
	return out, nil
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

func TestMarketOpsOpportunityDispositionAPIs(t *testing.T) {
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	repo := &fakeQueryRepository{marketOpsOpportunities: []storage.MarketOpsOpportunityRecord{{OpportunityID: "mopp-1", TenantID: "tenant-1"}}, marketOpsOpportunityDispositions: []storage.MarketOpsOpportunityDispositionRecord{{DispositionID: "disp-1", TenantID: "tenant-1", OpportunityID: "mopp-1", Disposition: storage.MarketOpsOpportunityDispositionWatch, Actor: "analyst-1", CreatedAt: now}}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/v1/marketops/opportunities/mopp-1/dispositions?tenant_id=tenant-1", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), "disp-1") {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
	post := httptest.NewRecorder()
	body := strings.NewReader(`{"tenant_id":"tenant-1","disposition":"needs_more_evidence","actor":"analyst-2","note":"await another session","metadata":{"ticket":"G146"}}`)
	router.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/v1/marketops/opportunities/mopp-1/dispositions", body))
	if post.Code != http.StatusCreated || len(repo.marketOpsOpportunityDispositions) != 2 {
		t.Fatalf("post status=%d body=%s records=%d", post.Code, post.Body.String(), len(repo.marketOpsOpportunityDispositions))
	}
	if repo.marketOpsOpportunityDispositions[1].Disposition != storage.MarketOpsOpportunityDispositionNeedsMoreEvidence {
		t.Fatalf("record=%+v", repo.marketOpsOpportunityDispositions[1])
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
