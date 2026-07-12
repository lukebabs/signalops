package dsm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const PolicyVersion = "marketops.backtest.policy_v1"

type PolicyEvaluator struct {
	AutoAcceptConfidence float64
	seen                 map[string]string
}

func NewPolicyEvaluator(autoAcceptConfidence float64) *PolicyEvaluator {
	if autoAcceptConfidence <= 0 || autoAcceptConfidence > 1 {
		autoAcceptConfidence = 0.75
	}
	return &PolicyEvaluator{AutoAcceptConfidence: autoAcceptConfidence, seen: map[string]string{}}
}

func (e *PolicyEvaluator) Evaluate(runID string, proposal storage.MarketOpsDSMGraphProposalRecord) (storage.MarketOpsBacktestPolicyResultRecord, error) {
	if e.seen == nil {
		e.seen = map[string]string{}
	}
	recommendation, reason := e.classify(proposal)
	identity := proposalIdentity(proposal)
	if recommendation != storage.MarketOpsBacktestPolicyAutoRejectCandidate && identity != "" {
		if existing := e.seen[identity]; existing != "" && existing != proposal.ProposalID {
			recommendation = storage.MarketOpsBacktestPolicySupersedeCandidate
			reason = "duplicate candidate identity superseded within the run"
		} else {
			e.seen[identity] = proposal.ProposalID
		}
	}
	inputs, err := json.Marshal(map[string]any{
		"proposal_id":            proposal.ProposalID,
		"candidate_type":         proposal.CandidateType,
		"node_id":                proposal.NodeID,
		"from_node":              proposal.FromNode,
		"relationship":           proposal.Relationship,
		"to_node":                proposal.ToNode,
		"labels":                 proposal.Labels,
		"confidence":             proposal.Confidence,
		"auto_accept_confidence": e.AutoAcceptConfidence,
	})
	if err != nil {
		return storage.MarketOpsBacktestPolicyResultRecord{}, err
	}
	return storage.MarketOpsBacktestPolicyResultRecord{
		RunID: strings.TrimSpace(runID), PolicyResultID: stablePolicyResultID(runID, proposal.ProposalID), ProposalID: proposal.ProposalID,
		ArtifactID: proposal.ArtifactID, SignalID: proposal.SignalID, TenantID: proposal.TenantID, SubjectSymbol: proposal.SubjectSymbol,
		CandidateType: proposal.CandidateType, Recommendation: recommendation, Reason: reason, PolicyVersion: PolicyVersion,
		Confidence: proposal.Confidence, DecisionInputsJSON: inputs,
	}, nil
}

func (e *PolicyEvaluator) classify(proposal storage.MarketOpsDSMGraphProposalRecord) (string, string) {
	if strings.TrimSpace(proposal.ProposalID) == "" || strings.TrimSpace(proposal.ArtifactID) == "" || strings.TrimSpace(proposal.SignalID) == "" {
		return storage.MarketOpsBacktestPolicyAutoRejectCandidate, "missing required proposal identity"
	}
	switch proposal.CandidateType {
	case "node_candidate":
		if strings.TrimSpace(proposal.NodeID) == "" {
			return storage.MarketOpsBacktestPolicyAutoRejectCandidate, "node candidate is missing node_id"
		}
		if !allowedNode(proposal.NodeID, proposal.Labels) {
			return storage.MarketOpsBacktestPolicyManualReviewRequired, "node candidate is outside the current auto-accept allowlist"
		}
	case "relationship_candidate":
		if strings.TrimSpace(proposal.FromNode) == "" || strings.TrimSpace(proposal.Relationship) == "" || strings.TrimSpace(proposal.ToNode) == "" {
			return storage.MarketOpsBacktestPolicyAutoRejectCandidate, "relationship candidate is missing identity fields"
		}
		if !allowedRelationship(proposal.Relationship) {
			return storage.MarketOpsBacktestPolicyManualReviewRequired, fmt.Sprintf("relationship %q is outside the current auto-accept allowlist", proposal.Relationship)
		}
	default:
		return storage.MarketOpsBacktestPolicyAutoRejectCandidate, "candidate_type is unsupported"
	}
	if proposal.Confidence < e.AutoAcceptConfidence {
		return storage.MarketOpsBacktestPolicyManualReviewRequired, "candidate confidence is below auto-accept threshold"
	}
	return storage.MarketOpsBacktestPolicyAutoAcceptCandidate, "candidate matches deterministic auto-accept policy"
}

func allowedNode(nodeID string, labels []string) bool {
	if strings.HasPrefix(nodeID, "ticker:") || strings.HasPrefix(nodeID, "signal_type:") || strings.HasPrefix(nodeID, "artifact:") {
		return true
	}
	for _, label := range labels {
		switch strings.TrimSpace(label) {
		case "MarketAsset", "Ticker", "DSMSignalType", "DSMArtifact":
			return true
		}
	}
	return false
}

func allowedRelationship(relationship string) bool {
	switch strings.TrimSpace(relationship) {
	case "EXHIBITS_SIGNAL", "SUPPORTED_BY_ARTIFACT":
		return true
	default:
		return false
	}
}

func proposalIdentity(proposal storage.MarketOpsDSMGraphProposalRecord) string {
	if proposal.CandidateType == "node_candidate" {
		return "node|" + strings.TrimSpace(proposal.NodeID)
	}
	if proposal.CandidateType == "relationship_candidate" {
		return "rel|" + strings.TrimSpace(proposal.FromNode) + "|" + strings.TrimSpace(proposal.Relationship) + "|" + strings.TrimSpace(proposal.ToNode)
	}
	return ""
}

func stablePolicyResultID(runID string, proposalID string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(runID) + "\x00" + strings.TrimSpace(proposalID)))
	return "btpolicy_marketops_v1_" + hex.EncodeToString(h[:])[:24]
}
