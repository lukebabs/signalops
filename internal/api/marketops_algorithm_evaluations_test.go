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
	run      storage.MarketOpsAlgorithmEvaluationRunRecord
	result   storage.MarketOpsAlgorithmEvaluationResultRecord
	outcome  storage.MarketOpsAlgorithmEvaluationOutcomeRecord
	campaign storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord
}

func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationRun(context.Context, storage.MarketOpsAlgorithmEvaluationRunRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) GetMarketOpsAlgorithmEvaluationRun(context.Context, string, string) (storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	return r.run, nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationRuns(context.Context, storage.MarketOpsAlgorithmEvaluationRunFilter) ([]storage.MarketOpsAlgorithmEvaluationRunRecord, error) {
	return []storage.MarketOpsAlgorithmEvaluationRunRecord{r.run}, nil
}
func (r *algorithmEvaluationReadRepository) InsertMarketOpsAlgorithmEvaluationResult(context.Context, storage.MarketOpsAlgorithmEvaluationResultRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationResults(context.Context, storage.MarketOpsAlgorithmEvaluationResultFilter) ([]storage.MarketOpsAlgorithmEvaluationResultRecord, error) {
	return []storage.MarketOpsAlgorithmEvaluationResultRecord{r.result}, nil
}
func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationOutcome(context.Context, storage.MarketOpsAlgorithmEvaluationOutcomeRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationOutcomes(context.Context, storage.MarketOpsAlgorithmEvaluationOutcomeFilter) ([]storage.MarketOpsAlgorithmEvaluationOutcomeRecord, error) {
	return []storage.MarketOpsAlgorithmEvaluationOutcomeRecord{r.outcome}, nil
}
func (r *algorithmEvaluationReadRepository) UpsertMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord) error {
	return nil
}
func (r *algorithmEvaluationReadRepository) GetMarketOpsAlgorithmEvaluationBackfillCampaign(context.Context, string, string) (storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
	return r.campaign, nil
}
func (r *algorithmEvaluationReadRepository) ListMarketOpsAlgorithmEvaluationBackfillCampaigns(context.Context, storage.MarketOpsAlgorithmEvaluationBackfillCampaignFilter) ([]storage.MarketOpsAlgorithmEvaluationBackfillCampaignRecord, error) {
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
