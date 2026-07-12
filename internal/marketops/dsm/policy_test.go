package dsm

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestPolicyEvaluatorAutoAcceptsAllowedHighConfidenceNode(t *testing.T) {
	evaluator := NewPolicyEvaluator(0.75)
	result, err := evaluator.Evaluate("bt-1", storage.MarketOpsDSMGraphProposalRecord{ProposalID: "prop-1", ArtifactID: "artifact-1", SignalID: "signal-1", TenantID: "tenant-1", SubjectSymbol: "AAPL", CandidateType: "node_candidate", NodeID: "ticker:AAPL", Labels: []string{"Ticker"}, Confidence: 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != storage.MarketOpsBacktestPolicyAutoAcceptCandidate || result.PolicyVersion != PolicyVersion {
		t.Fatalf("result = %+v", result)
	}
}

func TestPolicyEvaluatorRejectsMalformedRelationship(t *testing.T) {
	evaluator := NewPolicyEvaluator(0.75)
	result, err := evaluator.Evaluate("bt-1", storage.MarketOpsDSMGraphProposalRecord{ProposalID: "prop-1", ArtifactID: "artifact-1", SignalID: "signal-1", TenantID: "tenant-1", CandidateType: "relationship_candidate", FromNode: "ticker:AAPL", Relationship: "EXHIBITS_SIGNAL", Confidence: 0.9})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != storage.MarketOpsBacktestPolicyAutoRejectCandidate {
		t.Fatalf("result = %+v", result)
	}
}

func TestPolicyEvaluatorRequiresManualReviewBelowThreshold(t *testing.T) {
	evaluator := NewPolicyEvaluator(0.9)
	result, err := evaluator.Evaluate("bt-1", storage.MarketOpsDSMGraphProposalRecord{ProposalID: "prop-1", ArtifactID: "artifact-1", SignalID: "signal-1", TenantID: "tenant-1", CandidateType: "relationship_candidate", FromNode: "ticker:AAPL", Relationship: "EXHIBITS_SIGNAL", ToNode: "signal_type:marketops.dsm.volatility_expansion", Confidence: 0.8})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != storage.MarketOpsBacktestPolicyManualReviewRequired {
		t.Fatalf("result = %+v", result)
	}
}

func TestPolicyEvaluatorSupersedesDuplicateIdentity(t *testing.T) {
	evaluator := NewPolicyEvaluator(0.75)
	_, err := evaluator.Evaluate("bt-1", storage.MarketOpsDSMGraphProposalRecord{ProposalID: "prop-1", ArtifactID: "artifact-1", SignalID: "signal-1", TenantID: "tenant-1", CandidateType: "node_candidate", NodeID: "ticker:AAPL", Confidence: 1.0})
	if err != nil {
		t.Fatal(err)
	}
	result, err := evaluator.Evaluate("bt-1", storage.MarketOpsDSMGraphProposalRecord{ProposalID: "prop-2", ArtifactID: "artifact-2", SignalID: "signal-2", TenantID: "tenant-1", CandidateType: "node_candidate", NodeID: "ticker:AAPL", Confidence: 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != storage.MarketOpsBacktestPolicySupersedeCandidate {
		t.Fatalf("result = %+v", result)
	}
}
