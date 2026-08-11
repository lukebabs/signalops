package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestAuthenticatedReplayJobsBindTenantAndHideForeignRecords(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &fakeQueryRepository{replayJobs: []storage.ReplayJobRecord{{ReplayJobID: "replay-foreign", TenantID: "tenant-other"}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})

	body := bytes.NewBufferString(`{"source_id":"src-massive","dataset":"equity_eod_prices","source_kind":"raw_events","window_start":"2026-07-09T00:00:00Z","window_end":"2026-07-10T00:00:00Z","requested_by":"untrusted-body"}`)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/replay/jobs", body), token))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("create status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	created := repo.replayJobs[1]
	if created.TenantID != "tenant-local" || created.RequestedBy != "operator-auth" {
		t.Fatalf("created replay job = %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/replay/jobs", nil), token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	if repo.lastReplayFilter.TenantID != "tenant-local" {
		t.Fatalf("list tenant = %q", repo.lastReplayFilter.TenantID)
	}

	foreignRecorder := httptest.NewRecorder()
	router.ServeHTTP(foreignRecorder, withBearer(httptest.NewRequest(http.MethodGet, "/v1/replay/jobs/replay-foreign", nil), token))
	if foreignRecorder.Code != http.StatusNotFound {
		t.Fatalf("foreign detail status = %d body = %s", foreignRecorder.Code, foreignRecorder.Body.String())
	}
}

func TestAuthenticatedCalibrationBaselineBindsOmittedTenantToPrincipal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	summary := validMarketOpsBacktestCalibrationSummaryRecord()
	summary.TenantID = "tenant-local"
	repo := &fakeQueryRepository{backtestCalibrationSummaries: []storage.MarketOpsBacktestCalibrationSummaryRecord{summary}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})

	body := bytes.NewBufferString(`{"baseline_id":"btbase-local","name":"Tenant baseline","summary_id":"btcal-1","created_by":"untrusted-body","scope":{}}`)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-calibration-baselines", body), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if len(repo.backtestCalibrationBaselines) != 1 || repo.backtestCalibrationBaselines[0].TenantID != "tenant-local" {
		t.Fatalf("baselines = %+v", repo.backtestCalibrationBaselines)
	}
	if repo.backtestCalibrationBaselines[0].CreatedBy != "operator-auth" {
		t.Fatalf("created by = %q", repo.backtestCalibrationBaselines[0].CreatedBy)
	}
}

func TestAuthenticatedPromotionDecisionRejectsForeignCandidate(t *testing.T) {
	fixture := newTestAuthFixture(t)
	candidate := validMarketOpsBacktestPromotionCandidateRecord()
	candidate.TenantID = "tenant-other"
	repo := &fakeQueryRepository{backtestPromotionCandidates: []storage.MarketOpsBacktestPromotionCandidateRecord{candidate}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/marketops/backtest-promotion-candidates/btpromo-1/decision", bytes.NewBufferString(`{"status":"approved"}`)), token))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthenticatedAlgorithmProposalDecisionBindsOmittedTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &fakeQueryRepository{algorithmSignalProposals: []storage.AlgorithmSignalProposalRecord{{ProposalID: "proposal-local", TenantID: "tenant-local", Status: storage.AlgorithmSignalProposalStatusProposed}}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/algorithms/signal-proposals/proposal-local/decision", bytes.NewBufferString(`{"status":"rejected","actor":"untrusted-body"}`)), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastAlgorithmProposalMutation.TenantID != "tenant-local" || repo.lastAlgorithmProposalMutation.ReviewedBy != "operator-auth" {
		t.Fatalf("mutation = %+v", repo.lastAlgorithmProposalMutation)
	}
}

func TestAuthenticatedAlertLifecycleRejectsForeignRecord(t *testing.T) {
	fixture := newTestAuthFixture(t)
	alert := validAlertRecord()
	alert.TenantID = "tenant-other"
	repo := &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{alert}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/alerts/alert-1/acknowledge", bytes.NewBufferString(`{"note":"triaged"}`)), token))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.alerts[0].AcknowledgedAt != nil {
		t.Fatalf("foreign alert was mutated: %+v", repo.alerts[0])
	}
}

func TestAuthenticatedRawIngestInjectsPrincipalTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	publisher := &fakePublisher{}
	persistence := &fakePublishRepository{}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, Publisher: publisher, RawTopic: "signalops.test.raw.v1", PublishRepository: persistence})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})

	body := bytes.NewBufferString(`{"event_id":"evt-auth","source_id":"source-test","source_adapter":"manual-test","dataset":"equity-test","observation_time":"2026-07-08T06:00:00Z"}`)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/events/raw", body), token))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if persistence.ledger.TenantID != "tenant-local" {
		t.Fatalf("persisted tenant = %q", persistence.ledger.TenantID)
	}
	if !bytes.Contains(publisher.msg.Value, []byte(`"tenant_id":"tenant-local"`)) {
		t.Fatalf("published payload = %s", publisher.msg.Value)
	}
}

func TestAuthenticatedGraphProposalRoutesBindTenantAndHideForeignProposal(t *testing.T) {
	fixture := newTestAuthFixture(t)
	proposal := validMarketOpsDSMGraphProposalRecord()
	proposal.TenantID = "tenant-other"
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{proposal}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/graph-proposals", nil), token))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", list.Code, list.Body.String())
	}
	if repo.lastGraphProposalFilter.TenantID != "tenant-local" {
		t.Fatalf("list tenant = %q", repo.lastGraphProposalFilter.TenantID)
	}

	get := httptest.NewRecorder()
	router.ServeHTTP(get, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/graph-proposals/"+proposal.ProposalID, nil), token))
	if get.Code != http.StatusNotFound {
		t.Fatalf("foreign get status = %d body = %s", get.Code, get.Body.String())
	}

	decision := httptest.NewRecorder()
	router.ServeHTTP(decision, withBearer(httptest.NewRequest(http.MethodPost, "/v1/marketops/graph-proposals/"+proposal.ProposalID+"/decision", bytes.NewBufferString(`{"status":"accepted","actor":"untrusted-body"}`)), token))
	if decision.Code != http.StatusNotFound {
		t.Fatalf("foreign decision status = %d body = %s", decision.Code, decision.Body.String())
	}
	if repo.dsmGraphProposals[0].Status != storage.MarketOpsDSMGraphProposalStatusProposed || repo.lastGraphProposalMutation.ProposalID != "" {
		t.Fatalf("foreign proposal was mutated: proposal=%+v mutation=%+v", repo.dsmGraphProposals[0], repo.lastGraphProposalMutation)
	}
}

func TestAuthenticatedGraphProposalDecisionBindsPrincipalTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	proposal := validMarketOpsDSMGraphProposalRecord()
	proposal.ProposalSource = storage.MarketOpsGraphProposalSourceMarketState
	proposal.TenantID = "tenant-local"
	repo := &fakeQueryRepository{dsmGraphProposals: []storage.MarketOpsDSMGraphProposalRecord{proposal}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodPost, "/v1/marketops/graph-proposals/"+proposal.ProposalID+"/decision", bytes.NewBufferString(`{"status":"accepted","actor":"untrusted-body"}`)), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastGraphProposalMutation.TenantID != "tenant-local" || repo.lastGraphProposalMutation.ReviewedBy != "operator-auth" {
		t.Fatalf("mutation = %+v", repo.lastGraphProposalMutation)
	}
}

func TestAuthenticatedDSMArtifactRoutesBindTenantAndHideForeignArtifact(t *testing.T) {
	fixture := newTestAuthFixture(t)
	artifact := validMarketOpsDSMArtifactRecord()
	artifact.TenantID = "tenant-other"
	repo := &fakeQueryRepository{dsmArtifacts: []storage.MarketOpsDSMArtifactRecord{artifact}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/artifacts", nil), token))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", list.Code, list.Body.String())
	}
	if repo.lastDSMFilter.TenantID != "tenant-local" {
		t.Fatalf("list tenant = %q", repo.lastDSMFilter.TenantID)
	}

	get := httptest.NewRecorder()
	router.ServeHTTP(get, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/dsm/artifacts/"+artifact.ArtifactID, nil), token))
	if get.Code != http.StatusNotFound {
		t.Fatalf("foreign get status = %d body = %s", get.Code, get.Body.String())
	}
}

type signalAssuranceTenantBindingRepository struct {
	*fakeQueryRepository
	assertion                storage.SignalAssertionRecord
	lastAssertionFilter      storage.SignalAssuranceAssertionFilter
	lastEffectivenessFilter  storage.SignalAssuranceEffectivenessFilter
	lastObservationFilter    storage.SignalAssuranceEffectivenessFilter
	lastRecommendationFilter storage.SignalAssuranceEffectivenessFilter
}

func (r *signalAssuranceTenantBindingRepository) ListSignalAssuranceAssertions(_ context.Context, filter storage.SignalAssuranceAssertionFilter) ([]storage.SignalAssertionRecord, error) {
	r.lastAssertionFilter = filter
	if filter.TenantID != r.assertion.TenantID {
		return []storage.SignalAssertionRecord{}, nil
	}
	return []storage.SignalAssertionRecord{r.assertion}, nil
}

func (r *signalAssuranceTenantBindingRepository) GetSignalAssuranceAssertion(_ context.Context, tenantID, assertionID string) (storage.SignalAssertionRecord, error) {
	if tenantID != r.assertion.TenantID || assertionID != r.assertion.AssertionID {
		return storage.SignalAssertionRecord{}, storage.ErrNotFound
	}
	return r.assertion, nil
}

func (r *signalAssuranceTenantBindingRepository) ListSignalAssuranceEvaluations(context.Context, storage.SignalAssuranceEvaluationFilter) ([]storage.SignalAssertionEvaluationRecord, error) {
	return []storage.SignalAssertionEvaluationRecord{}, nil
}

func (r *signalAssuranceTenantBindingRepository) GetSignalValidationContract(context.Context, string) (storage.SignalValidationContractRecord, error) {
	return storage.SignalValidationContractRecord{}, storage.ErrNotFound
}

func (r *signalAssuranceTenantBindingRepository) ListSignalAssuranceEffectiveness(_ context.Context, filter storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessRecord, error) {
	r.lastEffectivenessFilter = filter
	return []storage.SignalAssuranceEffectivenessRecord{}, nil
}

func (r *signalAssuranceTenantBindingRepository) ListSignalAssuranceEffectivenessObservations(_ context.Context, filter storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceEffectivenessObservationRecord, error) {
	r.lastObservationFilter = filter
	return []storage.SignalAssuranceEffectivenessObservationRecord{}, nil
}

func (r *signalAssuranceTenantBindingRepository) ListSignalAssuranceRecommendations(_ context.Context, filter storage.SignalAssuranceEffectivenessFilter) ([]storage.SignalAssuranceRecommendationRecord, error) {
	r.lastRecommendationFilter = filter
	return []storage.SignalAssuranceRecommendationRecord{}, nil
}

func TestAuthenticatedSignalAssuranceRoutesBindTenantAndHideForeignData(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &signalAssuranceTenantBindingRepository{
		fakeQueryRepository: &fakeQueryRepository{},
		assertion:           storage.SignalAssertionRecord{AssertionID: "assertion-foreign", TenantID: "tenant-other"},
	}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	assertions := httptest.NewRecorder()
	router.ServeHTTP(assertions, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/signal-assurance/assertions", nil), token))
	if assertions.Code != http.StatusOK {
		t.Fatalf("assertions status = %d body = %s", assertions.Code, assertions.Body.String())
	}
	if repo.lastAssertionFilter.TenantID != "tenant-local" {
		t.Fatalf("assertion list tenant = %q", repo.lastAssertionFilter.TenantID)
	}

	foreign := httptest.NewRecorder()
	router.ServeHTTP(foreign, withBearer(httptest.NewRequest(http.MethodGet, "/v1/marketops/signal-assurance/assertions/assertion-foreign", nil), token))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign assertion status = %d body = %s", foreign.Code, foreign.Body.String())
	}

	requests := []struct {
		path   string
		filter *storage.SignalAssuranceEffectivenessFilter
	}{
		{path: "/v1/marketops/signal-assurance/effectiveness", filter: &repo.lastEffectivenessFilter},
		{path: "/v1/marketops/signal-assurance/effectiveness/observations?dimension_value=signal_type", filter: &repo.lastObservationFilter},
		{path: "/v1/marketops/signal-assurance/recommendations", filter: &repo.lastRecommendationFilter},
	}
	for _, request := range requests {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, request.path, nil), token))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d body = %s", request.path, recorder.Code, recorder.Body.String())
		}
		if request.filter.TenantID != "tenant-local" {
			t.Fatalf("%s tenant = %q", request.path, request.filter.TenantID)
		}
	}
}

func TestAuthenticatedCoreLedgersBindTenantAndHideForeignDetails(t *testing.T) {
	fixture := newTestAuthFixture(t)
	raw := validRawEventLedgerRecord()
	raw.EventID, raw.TenantID = "raw-foreign", "tenant-other"
	repo := &fakeQueryRepository{
		rawEvents:        []storage.RawEventLedgerRecord{raw},
		normalizedEvents: []storage.NormalizedEventLedgerRecord{{EventID: "normalized-foreign", TenantID: "tenant-other"}},
		signals:          []storage.SignalLedgerRecord{{SignalID: "signal-foreign", TenantID: "tenant-other"}},
	}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	for _, request := range []struct {
		path  string
		check func() string
	}{
		{path: "/v1/raw-events", check: func() string { return repo.lastFilter.TenantID }},
		{path: "/v1/normalized-events", check: func() string { return repo.lastFilter.TenantID }},
		{path: "/v1/signals", check: func() string { return repo.lastSignalLedgerFilter.TenantID }},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, request.path, nil), token))
		if recorder.Code != http.StatusOK || request.check() != "tenant-local" {
			t.Fatalf("%s status=%d tenant=%q body=%s", request.path, recorder.Code, request.check(), recorder.Body.String())
		}
	}

	for _, path := range []string{"/v1/raw-events/raw-foreign", "/v1/normalized-events/normalized-foreign", "/v1/signals/signal-foreign"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, path, nil), token))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestAuthenticatedBacktestReadRoutesBindTenantAndHideForeignDetails(t *testing.T) {
	fixture := newTestAuthFixture(t)
	const foreignTenant = "tenant-other"
	repo := &fakeQueryRepository{
		backtestCoverage:               []storage.MarketOpsBacktestCoverageRecord{{TenantID: foreignTenant}},
		backtestCampaigns:              []storage.MarketOpsBacktestCampaignRecord{{CampaignID: "campaign-foreign", TenantID: foreignTenant}},
		backtestRuns:                   []storage.MarketOpsBacktestRunRecord{{RunID: "run-local", TenantID: "tenant-local"}, {RunID: "run-foreign", TenantID: foreignTenant}},
		backtestCalibrationSummaries:   []storage.MarketOpsBacktestCalibrationSummaryRecord{{SummaryID: "summary-foreign", TenantID: foreignTenant}},
		backtestCalibrationBaselines:   []storage.MarketOpsBacktestCalibrationBaselineRecord{{BaselineID: "baseline-foreign", TenantID: foreignTenant}},
		backtestCalibrationComparisons: []storage.MarketOpsBacktestCalibrationComparisonRecord{{ComparisonID: "comparison-foreign", TenantID: foreignTenant}},
		backtestPromotionCandidates:    []storage.MarketOpsBacktestPromotionCandidateRecord{{CandidateID: "candidate-foreign", TenantID: foreignTenant}},
		backtestCalibrationReadiness:   []storage.MarketOpsBacktestCalibrationReadinessRecord{{ReadinessID: "readiness-foreign", TenantID: foreignTenant}},
		backtestSignals:                []storage.MarketOpsBacktestSignalRecord{{RunID: "run-local", SignalLedgerRecord: storage.SignalLedgerRecord{TenantID: foreignTenant}}},
		backtestGraphProposals:         []storage.MarketOpsBacktestGraphProposalRecord{{RunID: "run-local", MarketOpsDSMGraphProposalRecord: storage.MarketOpsDSMGraphProposalRecord{TenantID: foreignTenant}}},
		backtestPolicyResults:          []storage.MarketOpsBacktestPolicyResultRecord{{RunID: "run-local", TenantID: foreignTenant}},
		backtestEvaluations:            []storage.MarketOpsBacktestEvaluationRecord{{EvaluationID: "evaluation-foreign", TenantID: foreignTenant}},
		backtestEvaluationLabels:       []storage.MarketOpsBacktestEvaluationLabelRecord{{LabelID: "label-foreign", TenantID: foreignTenant}},
	}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	for _, request := range []struct {
		path   string
		tenant func() string
	}{
		{path: "/v1/marketops/backtest-coverage", tenant: func() string { return repo.lastBacktestCoverageFilter.TenantID }},
		{path: "/v1/marketops/backtest-campaigns", tenant: func() string { return repo.lastBacktestCampaignFilter.TenantID }},
		{path: "/v1/marketops/backtests", tenant: func() string { return repo.lastBacktestRunFilter.TenantID }},
		{path: "/v1/marketops/backtest-calibration-summaries", tenant: func() string { return repo.lastBacktestCalibrationFilter.TenantID }},
		{path: "/v1/marketops/backtest-calibration-baselines", tenant: func() string { return repo.lastBacktestBaselineFilter.TenantID }},
		{path: "/v1/marketops/backtest-calibration-comparisons", tenant: func() string { return repo.lastBacktestComparisonFilter.TenantID }},
		{path: "/v1/marketops/backtest-promotion-candidates", tenant: func() string { return repo.lastBacktestPromotionFilter.TenantID }},
		{path: "/v1/marketops/backtest-calibration-readiness", tenant: func() string { return repo.lastBacktestReadinessFilter.TenantID }},
		{path: "/v1/marketops/backtests/run-local/signals", tenant: func() string { return repo.lastBacktestSignalFilter.TenantID }},
		{path: "/v1/marketops/backtests/run-local/graph-proposals", tenant: func() string { return repo.lastBacktestGraphFilter.TenantID }},
		{path: "/v1/marketops/backtest-evaluations", tenant: func() string { return repo.lastBacktestEvaluationFilter.TenantID }},
		{path: "/v1/marketops/backtest-evaluation-labels", tenant: func() string { return repo.lastEvaluationLabelFilter.TenantID }},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, request.path, nil), token))
		if recorder.Code != http.StatusOK || request.tenant() != "tenant-local" {
			t.Fatalf("%s status=%d tenant=%q body=%s", request.path, recorder.Code, request.tenant(), recorder.Body.String())
		}
	}

	for _, path := range []string{
		"/v1/marketops/backtest-campaigns/campaign-foreign",
		"/v1/marketops/backtests/run-foreign",
		"/v1/marketops/backtest-calibration-summaries/summary-foreign",
		"/v1/marketops/backtest-calibration-baselines/baseline-foreign",
		"/v1/marketops/backtest-calibration-comparisons/comparison-foreign",
		"/v1/marketops/backtest-promotion-candidates/candidate-foreign",
		"/v1/marketops/backtest-calibration-readiness/readiness-foreign",
		"/v1/marketops/backtest-evaluations/evaluation-foreign",
		"/v1/marketops/backtest-evaluation-labels/label-foreign",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, path, nil), token))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestAuthenticatedIntelligenceSyncraticAndAlgorithmReadsBindTenant(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &fakeQueryRepository{
		cohortRuns:              []storage.MarketOpsIntelligenceCohortRunRecord{{RunID: "cohort-foreign", TenantID: "tenant-other"}},
		cohortResults:           []storage.MarketOpsIntelligenceCohortSymbolResultRecord{{ResultID: "readiness-foreign", TenantID: "tenant-other"}},
		syncraticContextWindows: []storage.SyncraticContextWindowRecord{{ContextWindowID: "context-foreign", TenantID: "tenant-other"}},
		syncraticInsights:       []storage.SyncraticInsightRecord{{SyncraticInsightID: "insight-foreign", TenantID: "tenant-other"}},
	}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleOperator}}})

	for _, request := range []struct {
		path   string
		tenant func() string
	}{
		{path: "/v1/marketops/intelligence/cohort-runs", tenant: func() string { return repo.lastCohortRunFilter.TenantID }},
		{path: "/v1/marketops/intelligence/readiness", tenant: func() string { return repo.lastCohortReadinessFilter.TenantID }},
		{path: "/v1/syncratic/context-windows", tenant: func() string { return repo.lastSyncraticContextWindowFilter.TenantID }},
		{path: "/v1/syncratic/insights", tenant: func() string { return repo.lastSyncraticInsightFilter.TenantID }},
		{path: "/v1/algorithms/definitions", tenant: func() string { return repo.lastAlgorithmDefinitionFilter.TenantID }},
		{path: "/v1/algorithms/execution-requests", tenant: func() string { return repo.lastAlgorithmExecutionFilter.TenantID }},
		{path: "/v1/algorithms/results", tenant: func() string { return repo.lastAlgorithmResultFilter.TenantID }},
		{path: "/v1/algorithms/signal-materializations", tenant: func() string { return repo.lastAlgorithmMaterializationFilter.TenantID }},
		{path: "/v1/algorithms/signal-proposals", tenant: func() string { return repo.lastAlgorithmProposalFilter.TenantID }},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, request.path, nil), token))
		if recorder.Code != http.StatusOK || request.tenant() != "tenant-local" {
			t.Fatalf("%s status=%d tenant=%q body=%s", request.path, recorder.Code, request.tenant(), recorder.Body.String())
		}
	}

	for _, path := range []string{
		"/v1/marketops/intelligence/cohort-runs/cohort-foreign",
		"/v1/syncratic/context-windows/context-foreign",
		"/v1/syncratic/insights/insight-foreign",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, withBearer(httptest.NewRequest(http.MethodGet, path, nil), token))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}
