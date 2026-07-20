package postgres

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateHypothesisSignalProposalGovernance(t *testing.T) {
	valid := storage.AlgorithmSignalProposalRecord{
		ProposalID: "msigprop-1", TenantID: "tenant-1",
		ProposalSource:         storage.SignalProposalSourceHypothesisEvaluation,
		HypothesisEvaluationID: "mhypeval-1", HypothesisKey: "H001", HypothesisVersion: "v1",
		HypothesisLifecycle: storage.MarketOpsHypothesisLifecycleCandidate,
		ProposedSignalType:  "marketops.hypothesis.h001.candidate",
		Status:              storage.AlgorithmSignalProposalStatusProposed, Score: .8, Confidence: .8, Severity: "high",
		CorrelationID: "run-1", ResearchOnly: true, ProposalPayloadJSON: []byte(`{}`),
		RationaleJSON: []byte(`{}`), EligibilitySnapshotJSON: []byte(`{}`),
	}
	if err := validateAlgorithmSignalProposal(valid); err != nil {
		t.Fatalf("valid candidate rejected: %v", err)
	}
	escaped := valid
	escaped.ResearchOnly, escaped.MaterializationEligible = false, true
	if validateAlgorithmSignalProposal(escaped) == nil {
		t.Fatal("candidate materialization escape was accepted")
	}
	ineligible := valid
	ineligible.HypothesisLifecycle = storage.MarketOpsHypothesisLifecycleResearch
	if validateAlgorithmSignalProposal(ineligible) == nil {
		t.Fatal("research lifecycle proposal was accepted")
	}
	approved := valid
	approved.HypothesisLifecycle = storage.MarketOpsHypothesisLifecycleApproved
	approved.ResearchOnly, approved.MaterializationEligible = false, true
	if err := validateAlgorithmSignalProposal(approved); err != nil {
		t.Fatalf("eligible approved proposal rejected: %v", err)
	}
}
