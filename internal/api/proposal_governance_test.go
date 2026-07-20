package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestHypothesisProposalIsVisibleAndMaterializationFailsClosed(t *testing.T) {
	proposal := storage.AlgorithmSignalProposalRecord{
		ProposalID: "msigprop-1", TenantID: "tenant-local",
		ProposalSource:         storage.SignalProposalSourceHypothesisEvaluation,
		HypothesisEvaluationID: "mhypeval-1", HypothesisKey: "H001", HypothesisVersion: "v1",
		HypothesisLifecycle: storage.MarketOpsHypothesisLifecycleCandidate,
		ProposedSignalType:  "marketops.hypothesis.h001.candidate",
		Status:              storage.AlgorithmSignalProposalStatusReviewed, Score: .8, Confidence: .75,
		Severity: "high", ResearchOnly: true, MaterializationEligible: false,
		ProposalPayloadJSON: []byte(`{}`), RationaleJSON: []byte(`{}`),
		EligibilitySnapshotJSON: []byte(`{"materialization_eligible":false}`),
		CorrelationID:           "run-1",
	}
	repo := &fakeQueryRepository{algorithmSignalProposals: []storage.AlgorithmSignalProposalRecord{proposal}}
	router := NewRouter(RouterConfig{QueryRepository: repo})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/v1/algorithms/signal-proposals?tenant_id=tenant-local&proposal_source=hypothesis_evaluation&hypothesis_key=H001", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"proposal_source":"hypothesis_evaluation"`) || !strings.Contains(list.Body.String(), `"hypothesis_evaluation_id":"mhypeval-1"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	if repo.lastAlgorithmProposalFilter.ProposalSource != storage.SignalProposalSourceHypothesisEvaluation || repo.lastAlgorithmProposalFilter.HypothesisKey != "H001" {
		t.Fatalf("filter=%+v", repo.lastAlgorithmProposalFilter)
	}

	materialize := httptest.NewRecorder()
	body := strings.NewReader(`{"tenant_id":"tenant-local","requested_by":"operator-1"}`)
	router.ServeHTTP(materialize, httptest.NewRequest(http.MethodPost, "/v1/algorithms/signal-proposals/msigprop-1/materializations", body))
	if materialize.Code != http.StatusConflict || !strings.Contains(materialize.Body.String(), "unsupported_materialization_source") {
		t.Fatalf("materialize status=%d body=%s", materialize.Code, materialize.Body.String())
	}
}
