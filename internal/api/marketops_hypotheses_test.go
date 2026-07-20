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

func (q *fakeQueryRepository) ListMarketOpsHypothesisDefinitions(_ context.Context, filter storage.MarketOpsHypothesisDefinitionFilter) ([]storage.MarketOpsHypothesisDefinitionRecord, error) {
	q.lastHypothesisDefinitionFilter = filter
	return q.marketOpsHypothesisDefinitions, nil
}
func (q *fakeQueryRepository) GetMarketOpsHypothesisDefinition(_ context.Context, tenantID, key, version string) (storage.MarketOpsHypothesisDefinitionRecord, error) {
	for _, r := range q.marketOpsHypothesisDefinitions {
		if r.TenantID == tenantID && r.HypothesisKey == key && r.HypothesisVersion == version {
			return r, nil
		}
	}
	return storage.MarketOpsHypothesisDefinitionRecord{}, storage.ErrNotFound
}
func (q *fakeQueryRepository) ListMarketOpsHypothesisEvaluations(_ context.Context, filter storage.MarketOpsHypothesisEvaluationFilter) ([]storage.MarketOpsHypothesisEvaluationRecord, error) {
	q.lastHypothesisEvaluationFilter = filter
	return q.marketOpsHypothesisEvaluations, nil
}

func TestMarketOpsHypothesisReadAPIs(t *testing.T) {
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	score := .8
	definition := storage.MarketOpsHypothesisDefinitionRecord{TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v1", Title: "Term shift", Domain: "volatility_surface", Direction: "non_directional", RequiredFeaturesJSON: []byte(`[]`), RequiredTransitionsJSON: []byte(`[]`), QualityPolicyJSON: []byte(`{}`), EligibilityExpressionJSON: []byte(`{}`), TriggerExpressionJSON: []byte(`{}`), PersistenceRuleJSON: []byte(`{}`), CorroborationRuleJSON: []byte(`{}`), InvalidationRuleJSON: []byte(`{}`), ExpectedOutcomesJSON: []byte(`[]`), ScoringConfigJSON: []byte(`{}`), CalibrationPolicyJSON: []byte(`{"production_materialization_allowed":false}`), LifecycleStatus: storage.MarketOpsHypothesisLifecycleResearch}
	evaluation := storage.MarketOpsHypothesisEvaluationRecord{EvaluationID: "mhypeval-1", TenantID: "tenant-1", AppID: "marketops", HypothesisKey: "H004", HypothesisVersion: "v1", MarketStateID: "mstate-1", AssetID: "ticker:AAPL", Symbol: "AAPL", SessionDate: now, AsOfTime: now, Eligible: true, Triggered: false, QualityScore: &score, ReasonCodes: []string{"eligible_not_triggered"}, EvaluationPayloadJSON: []byte(`{"research_only":true}`), EvaluationRunID: "run-1", DeterministicKey: "key-1"}
	repo := &fakeQueryRepository{marketOpsHypothesisDefinitions: []storage.MarketOpsHypothesisDefinitionRecord{definition}, marketOpsHypothesisEvaluations: []storage.MarketOpsHypothesisEvaluationRecord{evaluation}}
	router := NewRouter(RouterConfig{QueryRepository: repo})
	for _, path := range []string{"/v1/marketops/hypotheses?tenant_id=tenant-1&lifecycle_status=research", "/v1/marketops/hypotheses/H004/v1?tenant_id=tenant-1", "/v1/marketops/hypothesis-evaluations?tenant_id=tenant-1&symbol=AAPL&eligible=true&triggered=false"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
	}
	if repo.lastHypothesisDefinitionFilter.LifecycleStatus != "research" || repo.lastHypothesisEvaluationFilter.Eligible == nil || !*repo.lastHypothesisEvaluationFilter.Eligible || repo.lastHypothesisEvaluationFilter.Triggered == nil || *repo.lastHypothesisEvaluationFilter.Triggered {
		t.Fatalf("filters definition=%+v evaluation=%+v", repo.lastHypothesisDefinitionFilter, repo.lastHypothesisEvaluationFilter)
	}
}

func TestMarketOpsHypothesisReadsRejectInvalidQueries(t *testing.T) {
	router := NewRouter(RouterConfig{QueryRepository: &fakeQueryRepository{}})
	for _, path := range []string{"/v1/marketops/hypotheses/H004/v1", "/v1/marketops/hypothesis-evaluations?eligible=maybe"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d", path, rec.Code)
		}
	}
}
