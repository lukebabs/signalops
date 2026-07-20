package proposals

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildEnforcesLifecycleAndMaterializationPolicy(t *testing.T) {
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	score, confidence := .82, .77
	baseEvaluation := storage.MarketOpsHypothesisEvaluationRecord{
		EvaluationID: "mhypeval_1", TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1",
		MarketStateID: "mstate_1", AssetID: "asset-1", Symbol: "AAPL", SessionDate: now,
		Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &confidence,
		EvidenceIDs: []string{"mevidence_1"}, EvaluationRunID: "evalrun-1",
	}
	definitions := []storage.MarketOpsHypothesisDefinitionRecord{
		{TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1", Title: "Candidate", Domain: "momentum", Direction: "downside", LifecycleStatus: storage.MarketOpsHypothesisLifecycleCandidate, CalibrationPolicyJSON: []byte(`{"production_materialization_allowed":true}`)},
		{TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v1", Title: "Approved", Domain: "volatility", Direction: "non_directional", LifecycleStatus: storage.MarketOpsHypothesisLifecycleApproved, ApprovedBy: "reviewer", ApprovedAt: &now, CalibrationPolicyJSON: []byte(`{"production_materialization_allowed":true}`)},
		{TenantID: "tenant-1", HypothesisKey: "H006", HypothesisVersion: "v1", Title: "Research", Domain: "divergence", Direction: "conditional", LifecycleStatus: storage.MarketOpsHypothesisLifecycleResearch},
	}
	approved := baseEvaluation
	approved.EvaluationID, approved.HypothesisKey = "mhypeval_2", "H004"
	research := baseEvaluation
	research.EvaluationID, research.HypothesisKey = "mhypeval_3", "H006"

	result, err := Build("proposal-run-1", "analyst-1", definitions, []storage.MarketOpsHypothesisEvaluationRecord{research, approved, baseEvaluation})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Proposals) != 2 || result.SkippedReasons["hypothesis_lifecycle_ineligible"] != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	byKey := map[string]storage.AlgorithmSignalProposalRecord{}
	for _, proposal := range result.Proposals {
		byKey[proposal.HypothesisKey] = proposal
		if proposal.ProposalSource != storage.SignalProposalSourceHypothesisEvaluation || proposal.Status != storage.AlgorithmSignalProposalStatusProposed {
			t.Fatalf("proposal did not enter governed workflow: %+v", proposal)
		}
		if proposal.AlgorithmResultID != "" || len(proposal.EvidenceRefs) < 2 {
			t.Fatalf("proposal has false algorithm lineage or missing evidence: %+v", proposal)
		}
	}
	if !byKey["H001"].ResearchOnly || byKey["H001"].MaterializationEligible {
		t.Fatalf("candidate escaped research-only control: %+v", byKey["H001"])
	}
	if byKey["H004"].ResearchOnly || !byKey["H004"].MaterializationEligible {
		t.Fatalf("approved proposal did not honor explicit production policy: %+v", byKey["H004"])
	}
}

func TestBuildRejectsUnusableEvaluationsAndUnauditedApproval(t *testing.T) {
	score, confidence := .8, .8
	definition := storage.MarketOpsHypothesisDefinitionRecord{
		TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1",
		LifecycleStatus:       storage.MarketOpsHypothesisLifecycleApproved,
		CalibrationPolicyJSON: []byte(`{"production_materialization_allowed":true}`),
	}
	evaluation := storage.MarketOpsHypothesisEvaluationRecord{
		EvaluationID: "evaluation-1", TenantID: "tenant-1", HypothesisKey: "H001", HypothesisVersion: "v1",
		Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &confidence,
	}
	result, err := Build("run-1", "analyst-1", []storage.MarketOpsHypothesisDefinitionRecord{definition}, []storage.MarketOpsHypothesisEvaluationRecord{evaluation})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Proposals) != 0 || result.SkippedReasons["approved_hypothesis_missing_audit"] != 1 {
		t.Fatalf("unaudited approval was not blocked: %+v", result)
	}
	evaluation.Invalidated = true
	definition.LifecycleStatus = storage.MarketOpsHypothesisLifecycleCandidate
	result, err = Build("run-1", "analyst-1", []storage.MarketOpsHypothesisDefinitionRecord{definition}, []storage.MarketOpsHypothesisEvaluationRecord{evaluation})
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedReasons["evaluation_not_triggered_eligible"] != 1 {
		t.Fatalf("invalidated evaluation was not blocked: %+v", result)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	score := .6
	definition := storage.MarketOpsHypothesisDefinitionRecord{TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v2", LifecycleStatus: storage.MarketOpsHypothesisLifecycleCandidate}
	evaluation := storage.MarketOpsHypothesisEvaluationRecord{EvaluationID: "eval-1", TenantID: "tenant-1", HypothesisKey: "H004", HypothesisVersion: "v2", MarketStateID: "state-1", Eligible: true, Triggered: true, TriggerScore: &score, ConfidenceScore: &score}
	first, err := Build("run-1", "analyst", []storage.MarketOpsHypothesisDefinitionRecord{definition}, []storage.MarketOpsHypothesisEvaluationRecord{evaluation})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build("run-2", "analyst", []storage.MarketOpsHypothesisDefinitionRecord{definition}, []storage.MarketOpsHypothesisEvaluationRecord{evaluation})
	if err != nil {
		t.Fatal(err)
	}
	if first.Proposals[0].ProposalID != second.Proposals[0].ProposalID {
		t.Fatalf("proposal identity changed across runs: %s != %s", first.Proposals[0].ProposalID, second.Proposals[0].ProposalID)
	}
}
