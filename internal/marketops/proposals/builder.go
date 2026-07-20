package proposals

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
)

type Result struct {
	Scanned        int
	Proposals      []storage.AlgorithmSignalProposalRecord
	SkippedReasons map[string]int
}

func Build(runID, createdBy string, definitions []storage.MarketOpsHypothesisDefinitionRecord, evaluations []storage.MarketOpsHypothesisEvaluationRecord) (Result, error) {
	runID, createdBy = strings.TrimSpace(runID), strings.TrimSpace(createdBy)
	if runID == "" || createdBy == "" {
		return Result{}, fmt.Errorf("proposal run and creator are required")
	}
	result := Result{Scanned: len(evaluations), SkippedReasons: map[string]int{}}
	definitionsByVersion := map[string]storage.MarketOpsHypothesisDefinitionRecord{}
	for _, definition := range definitions {
		definitionsByVersion[definitionKey(definition.TenantID, definition.HypothesisKey, definition.HypothesisVersion)] = definition
	}
	sort.Slice(evaluations, func(i, j int) bool { return evaluations[i].EvaluationID < evaluations[j].EvaluationID })
	for _, evaluation := range evaluations {
		definition, ok := definitionsByVersion[definitionKey(evaluation.TenantID, evaluation.HypothesisKey, evaluation.HypothesisVersion)]
		if !ok {
			result.SkippedReasons["missing_exact_definition"]++
			continue
		}
		if !evaluation.Eligible || !evaluation.Triggered || evaluation.Invalidated {
			result.SkippedReasons["evaluation_not_triggered_eligible"]++
			continue
		}
		if evaluation.TriggerScore == nil || evaluation.ConfidenceScore == nil {
			result.SkippedReasons["evaluation_missing_scores"]++
			continue
		}
		if definition.LifecycleStatus != storage.MarketOpsHypothesisLifecycleCandidate && definition.LifecycleStatus != storage.MarketOpsHypothesisLifecycleApproved {
			result.SkippedReasons["hypothesis_lifecycle_ineligible"]++
			continue
		}
		materializationEligible := false
		if definition.LifecycleStatus == storage.MarketOpsHypothesisLifecycleApproved {
			if definition.ApprovedAt == nil || strings.TrimSpace(definition.ApprovedBy) == "" {
				result.SkippedReasons["approved_hypothesis_missing_audit"]++
				continue
			}
			materializationEligible = productionMaterializationAllowed(definition.CalibrationPolicyJSON)
		}
		proposal, err := proposalRecord(runID, createdBy, definition, evaluation, materializationEligible)
		if err != nil {
			return result, err
		}
		result.Proposals = append(result.Proposals, proposal)
	}
	return result, nil
}

func proposalRecord(runID, createdBy string, definition storage.MarketOpsHypothesisDefinitionRecord, evaluation storage.MarketOpsHypothesisEvaluationRecord, materializationEligible bool) (storage.AlgorithmSignalProposalRecord, error) {
	signalType := "marketops.hypothesis." + strings.ToLower(strings.TrimSpace(definition.HypothesisKey)) + ".candidate"
	identity, err := marketopsstate.NewIdentity(marketopsstate.IdentitySignalProposal, evaluation.TenantID, evaluation.EvaluationID, signalType)
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, err
	}
	score, confidence := *evaluation.TriggerScore, *evaluation.ConfidenceScore
	researchOnly := !materializationEligible
	payload, err := json.Marshal(map[string]any{
		"schema_version": "marketops_hypothesis_signal_proposal.v1",
		"asset_id":       evaluation.AssetID, "symbol": evaluation.Symbol,
		"session_date":       evaluation.SessionDate.Format("2006-01-02"),
		"market_state_id":    evaluation.MarketStateID,
		"hypothesis_key":     definition.HypothesisKey,
		"hypothesis_version": definition.HypothesisVersion,
		"direction":          definition.Direction,
		"research_only":      researchOnly,
	})
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, err
	}
	rationale, err := json.Marshal(map[string]any{
		"definition_title":    definition.Title,
		"definition_domain":   definition.Domain,
		"reason_codes":        evaluation.ReasonCodes,
		"review_required":     true,
		"direct_signal_write": false,
	})
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, err
	}
	eligibility, err := json.Marshal(map[string]any{
		"schema_version":                     "marketops_hypothesis_proposal_eligibility.v1",
		"run_id":                             runID,
		"evaluation_eligible":                evaluation.Eligible,
		"evaluation_triggered":               evaluation.Triggered,
		"evaluation_invalidated":             evaluation.Invalidated,
		"hypothesis_lifecycle_status":        definition.LifecycleStatus,
		"approved_by":                        definition.ApprovedBy,
		"approved_at":                        definition.ApprovedAt,
		"production_materialization_allowed": productionMaterializationAllowed(definition.CalibrationPolicyJSON),
		"materialization_eligible":           materializationEligible,
	})
	if err != nil {
		return storage.AlgorithmSignalProposalRecord{}, err
	}
	evidenceRefs := []string{"marketops_hypothesis_evaluation:" + evaluation.EvaluationID, "marketops_market_state:" + evaluation.MarketStateID}
	for _, evidenceID := range evaluation.EvidenceIDs {
		evidenceRefs = append(evidenceRefs, "marketops_evidence:"+evidenceID)
	}
	return storage.AlgorithmSignalProposalRecord{
		ProposalID: identity.ID, TenantID: evaluation.TenantID,
		ProposalSource:         storage.SignalProposalSourceHypothesisEvaluation,
		HypothesisEvaluationID: evaluation.EvaluationID, HypothesisKey: definition.HypothesisKey,
		HypothesisVersion: definition.HypothesisVersion, HypothesisLifecycle: definition.LifecycleStatus,
		ProposedSignalType: signalType, Status: storage.AlgorithmSignalProposalStatusProposed,
		Score: score, Confidence: confidence, Severity: severity(score),
		ProposalPayloadJSON: payload, RationaleJSON: rationale, EvidenceRefs: evidenceRefs,
		CorrelationID: firstNonEmpty(evaluation.EvaluationRunID, runID),
		ResearchOnly:  researchOnly, MaterializationEligible: materializationEligible,
		EligibilitySnapshotJSON: eligibility, CreatedBy: createdBy,
	}, nil
}

func productionMaterializationAllowed(raw []byte) bool {
	var policy struct {
		Allowed bool `json:"production_materialization_allowed"`
	}
	return json.Unmarshal(raw, &policy) == nil && policy.Allowed
}

func definitionKey(tenantID, key, version string) string {
	return strings.TrimSpace(tenantID) + "\x00" + strings.TrimSpace(key) + "\x00" + strings.TrimSpace(version)
}

func severity(score float64) string {
	switch {
	case score >= .9:
		return "critical"
	case score >= .75:
		return "high"
	case score >= .5:
		return "medium"
	case score >= .25:
		return "low"
	default:
		return "info"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
