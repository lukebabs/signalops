package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

type algorithmEvaluationReadRepository struct {
	*fakeQueryRepository
	run                storage.MarketOpsAlgorithmEvaluationRunRecord
	result             storage.MarketOpsAlgorithmEvaluationResultRecord
	outcome            storage.MarketOpsAlgorithmEvaluationOutcomeRecord
	campaign           storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord
	lastRunFilter      storage.MarketOpsAlgorithmEvaluationRunFilter
	lastResultFilter   storage.MarketOpsAlgorithmEvaluationResultFilter
	lastOutcomeFilter  storage.MarketOpsAlgorithmEvaluationOutcomeFilter
	lastCampaignFilter storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter
	lastRunTenant      string
	lastCampaignTenant string
}

func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationRun(context.Context, storage.MarketOpsAlgorithmEvaluationRunRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) GetMarketOpsAlgorithmEvaluationRun(_ context.Context, tenantID, runID string) (storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	r.lastRunTenant = tenantID
	if tenantID != r.run.TenantID || runID != r.run.RunID {
		return storage.MarketOpsAlgorithmEvaluationRunRecord{}, storage.ErrNotFound
	}
	return r.run, nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationRuns(_ context.Context, filter storage.MarketOpsAlgorithmEvaluationRunFilter) ([]storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	r.lastRunFilter = filter
	if filter.TenantID != r.run.TenantID {
		return []storage.MarketOpsAlgorithmEvaluationRunRecord{}, nil
	}
	return []storage.MarketOpsAlgorithmEvaluationRunRecord{r.run}, nil
}
func (r *algorithmEvaluationReadRepository) InsertMarketOpsAlgorithmEvaluationResult(context.Context, storage.MarketOpsAlgorithmEvaluationResultRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationResults(_ context.Context, filter storage.MarketOpsAlgorithmEvaluationResultFilter) ([]storage.MarketOpsAlgorithmEvaluationResultRecord, error) {
	r.lastResultFilter = filter
	if filter.TenantID != r.result.TenantID {
		return []storage.MarketOpsAlgorithmEvaluationResultRecord{}, nil
	}
	return []storage.MarketOpsAlgorithmEvaluationResultRecord{r.result}, nil
}
func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationOutcome(context.Context, storage.MarketOpsAlgorithmEvaluationOutcomeRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationOutcomes(_ context.Context, filter storage.MarketOpsAlgorithmEvaluationOutcomeFilter) ([]storage.MarketOpsAlgorithmEvaluationOutcomeRecord, error) {
	r.lastOutcomeFilter = filter
	if filter.TenantID != r.outcome.TenantID {
		return []storage.MarketOpsAlgorithmEvaluationOutcomeRecord{}, nil
	}
	return []storage.MarketOpsAlgorithmEvaluationOutcomeRecord{r.outcome}, nil
}
func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) GetMarketOpsAlgorithmEvaluationBackfillCampaign(_ context.Context, tenantID, campaignID string) (storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	r.lastCampaignTenant = tenantID
	if tenantID != r.campaign.TenantID || campaignID != r.campaign.CampaignID {
		return storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{}, storage.ErrNotFound
	}
	return r.campaign, nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationBackfillCampaigns(_ context.Context, filter storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter) ([]storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	r.lastCampaignFilter = filter
	if filter.TenantID != r.campaign.TenantID {
		return []storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{}, nil
	}
	return []storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{r.campaign}, nil
}

func TestMarketOpsAlgorithmEvaluationReadAPIs(t *testing.T) {
	repo := &algorithmEvaluationReadRepository{
		fakeQueryRepository: &fakeQueryRepository{},
		run:                 storage.MarketOpsAlgorithmEvaluationRunRecord{RunID: "eval-1", TenantID: "tenant-1"},
		result:              storage.MarketOpsAlgorithmEvaluationResultRecord{EvaluationResultID: "result-1", RunID: "eval-1", TenantID: "tenant-1"},
		outcome:             storage.MarketOpsAlgorithmEvaluationOutcomeRecord{EvaluationOutcomeID: "outcome-1", RunID: "eval-1", TenantID: "tenant-1", HorizonSessions: 5},
		campaign:            storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{CampaignID: "campaign-1", TenantID: "tenant-1"},
	}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	for _, path := range []string{
		"/v1/marketops/algorithm-evaluations?tenant_id=tenant-1",
		"/v1/marketops/algorithm-evaluations/eval-1?tenant_id=tenant-1",
		"/v1/marketops/algorithm-evaluations/eval-1/results?tenant_id=tenant-1",
		"/v1/marketops/algorithm-evaluations/eval-1/outcomes?tenant_id=tenant-1&horizon_sessions=5",
		"/v1/marketops/algorithm-evaluation-backfills?tenant_id=tenant-1",
		"/v1/marketops/algorithm-evaluation-backfills/campaign-1?tenant_id=tenant-1",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestMarketOpsAlgorithmEvaluationReadAPIsRejectInvalidQueries(t *testing.T) {
	repo := &algorithmEvaluationReadRepository{fakeQueryRepository: &fakeQueryRepository{}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	for _, path := range []string{
		"/v1/marketops/algorithm-evaluations/eval-1",
		"/v1/marketops/algorithm-evaluations/eval-1/outcomes?tenant_id=tenant-1&horizon_sessions=3",
		"/v1/marketops/algorithm-evaluation-backfills/campaign-1",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestAuthenticatedAlgorithmEvaluationReadsBindPrincipalTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &algorithmEvaluationReadRepository{
		fakeQueryRepository: &fakeQueryRepository{},
		run:                 storage.MarketOpsAlgorithmEvaluationRunRecord{RunID: "eval-foreign", TenantID: "tenant-other"},
		result:              storage.MarketOpsAlgorithmEvaluationResultRecord{EvaluationResultID: "result-foreign", RunID: "eval-foreign", TenantID: "tenant-other"},
		outcome:             storage.MarketOpsAlgorithmEvaluationOutcomeRecord{EvaluationOutcomeID: "outcome-foreign", RunID: "eval-foreign", TenantID: "tenant-other"},
		campaign:            storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord{CampaignID: "campaign-foreign", TenantID: "tenant-other"},
	}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/algorithm-evaluations", nil), token))
	if list.Code != http.StatusOK || repo.lastRunFilter.TenantID != "tenant-local" {
		t.Fatalf("list status=%d tenant=%q body=%s", list.Code, repo.lastRunFilter.TenantID, list.Body.String())
	}

	foreign := httptest.NewRecorder()
	router.ServeHTTP(foreign, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/algorithm-evaluations/eval-foreign", nil), token))
	if foreign.Code != http.StatusNotFound || repo.lastRunTenant != "tenant-local" {
		t.Fatalf("foreign status=%d tenant=%q body=%s", foreign.Code, repo.lastRunTenant, foreign.Body.String())
	}

	for _, request := range []struct {
		path   string
		tenant func() string
	}{
		{path: "/v1/marketops/algorithm-evaluations/eval-foreign/results", tenant: func() string { return repo.lastResultFilter.TenantID }},
		{path: "/v1/marketops/algorithm-evaluations/eval-foreign/outcomes", tenant: func() string { return repo.lastOutcomeFilter.TenantID }},
		{path: "/v1/marketops/algorithm-evaluation-backfills", tenant: func() string { return repo.lastCampaignFilter.TenantID }},
		{path: "/v1/marketops/algorithm-evaluation-backfills/campaign-foreign", tenant: func() string { return repo.lastCampaignTenant }},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, request.path, nil), token))
		if recorder.Code != http.StatusOK && recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", request.path, recorder.Code, recorder.Body.String())
		}
		if request.tenant() != "tenant-local" {
			t.Fatalf("%s tenant=%q", request.path, request.tenant())
		}
	}
}
