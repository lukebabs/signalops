package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	marketOpsPromotionMinimumLabelCoverage = 0.8
	marketOpsPromotionMinimumPrecision     = 0.9
	marketOpsPromotionMinimumRecall        = 0.8
)

type marketOpsBacktestPromotionCandidateCreateRequest struct {
	CandidateID      string `json:"candidate_id"`
	TenantID         string `json:"tenant_id"`
	BaselineID       string `json:"baseline_id"`
	ComparisonID     string `json:"comparison_id"`
	EvaluationID     string `json:"evaluation_id"`
	CandidateVersion string `json:"candidate_version"`
	RequestedBy      string `json:"requested_by"`
}

type marketOpsBacktestPromotionCandidateDecisionRequest struct {
	Status       string `json:"status"`
	ReviewedBy   string `json:"reviewed_by"`
	DecisionNote string `json:"decision_note"`
}

type marketOpsBacktestPromotionCandidateDTO struct {
	CandidateID      string          `json:"candidate_id"`
	TenantID         string          `json:"tenant_id"`
	AppID            string          `json:"app_id"`
	Domain           string          `json:"domain"`
	UseCase          string          `json:"use_case"`
	BaselineID       string          `json:"baseline_id"`
	ComparisonID     string          `json:"comparison_id"`
	EvaluationID     string          `json:"evaluation_id"`
	RunID            string          `json:"run_id"`
	DetectorID       string          `json:"detector_id"`
	DetectorVersion  string          `json:"detector_version"`
	Dataset          string          `json:"dataset"`
	PolicyVersion    string          `json:"policy_version"`
	CandidateVersion string          `json:"candidate_version"`
	ReadinessStatus  string          `json:"readiness_status"`
	ReadinessReasons []string        `json:"readiness_reasons"`
	Evidence         json.RawMessage `json:"evidence"`
	Status           string          `json:"status"`
	RequestedBy      string          `json:"requested_by"`
	ReviewedBy       string          `json:"reviewed_by"`
	ReviewedAt       *time.Time      `json:"reviewed_at,omitempty"`
	DecisionNote     string          `json:"decision_note"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func marketOpsBacktestPromotionCandidateResponse(record storage.MarketOpsBacktestPromotionCandidateRecord) marketOpsBacktestPromotionCandidateDTO {
	return marketOpsBacktestPromotionCandidateDTO{CandidateID: record.CandidateID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, BaselineID: record.BaselineID, ComparisonID: record.ComparisonID, EvaluationID: record.EvaluationID, RunID: record.RunID, DetectorID: record.DetectorID, DetectorVersion: record.DetectorVersion, Dataset: record.Dataset, PolicyVersion: record.PolicyVersion, CandidateVersion: record.CandidateVersion, ReadinessStatus: record.ReadinessStatus, ReadinessReasons: record.ReadinessReasons, Evidence: json.RawMessage(jsonOrDefault(record.EvidenceJSON, `{}`)), Status: record.Status, RequestedBy: record.RequestedBy, ReviewedBy: record.ReviewedBy, ReviewedAt: record.ReviewedAt, DecisionNote: record.DecisionNote, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsBacktestPromotionCandidateResponses(records []storage.MarketOpsBacktestPromotionCandidateRecord) []marketOpsBacktestPromotionCandidateDTO {
	responses := make([]marketOpsBacktestPromotionCandidateDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestPromotionCandidateResponse(record))
	}
	return responses
}

func buildMarketOpsBacktestPromotionCandidate(candidateID string, requestedBy string, candidateVersion string, baseline storage.MarketOpsBacktestCalibrationBaselineRecord, comparison storage.MarketOpsBacktestCalibrationComparisonRecord, evaluation *storage.MarketOpsBacktestEvaluationRecord, run *storage.MarketOpsBacktestRunRecord, policyVersion string) (storage.MarketOpsBacktestPromotionCandidateRecord, error) {
	readiness, reasons := marketOpsBacktestPromotionReadiness(comparison, evaluation)
	runID := ""
	detectorVersion := ""
	if run != nil {
		runID = run.RunID
		detectorVersion = run.DetectorVersion
	}
	evaluationID := ""
	if evaluation != nil {
		evaluationID = evaluation.EvaluationID
		if runID == "" {
			runID = evaluation.RunID
		}
	}
	evidence, err := json.Marshal(map[string]any{
		"baseline":       map[string]any{"baseline_id": baseline.BaselineID, "summary_id": baseline.SummaryID, "name": baseline.Name},
		"comparison":     map[string]any{"comparison_id": comparison.ComparisonID, "recommendation": comparison.Recommendation, "recommendation_reason": comparison.RecommendationReason, "metrics": json.RawMessage(jsonOrDefault(comparison.ComparisonMetricsJSON, `{}`))},
		"evaluation":     marketOpsBacktestPromotionEvaluationEvidence(evaluation),
		"detector":       map[string]any{"detector_id": firstNonEmptyBacktestValue(comparison.DetectorID, baseline.DetectorID), "detector_version": detectorVersion},
		"run":            map[string]any{"run_id": runID},
		"policy_version": policyVersion,
		"readiness":      map[string]any{"status": readiness, "reasons": reasons, "minimum_label_coverage": marketOpsPromotionMinimumLabelCoverage, "minimum_precision": marketOpsPromotionMinimumPrecision, "minimum_recall": marketOpsPromotionMinimumRecall},
	})
	if err != nil {
		return storage.MarketOpsBacktestPromotionCandidateRecord{}, err
	}
	return storage.MarketOpsBacktestPromotionCandidateRecord{CandidateID: strings.TrimSpace(candidateID), TenantID: baseline.TenantID, AppID: baseline.AppID, Domain: baseline.Domain, UseCase: baseline.UseCase, BaselineID: baseline.BaselineID, ComparisonID: comparison.ComparisonID, EvaluationID: evaluationID, RunID: runID, DetectorID: firstNonEmptyBacktestValue(comparison.DetectorID, baseline.DetectorID), DetectorVersion: detectorVersion, Dataset: firstNonEmptyBacktestValue(comparison.Dataset, baseline.Dataset), PolicyVersion: strings.TrimSpace(policyVersion), CandidateVersion: strings.TrimSpace(candidateVersion), ReadinessStatus: readiness, ReadinessReasons: reasons, EvidenceJSON: evidence, Status: storage.MarketOpsBacktestPromotionCandidateStatusProposed, RequestedBy: firstNonEmptyBacktestValue(requestedBy, "operator-local")}, nil
}

func marketOpsBacktestPromotionReadiness(comparison storage.MarketOpsBacktestCalibrationComparisonRecord, evaluation *storage.MarketOpsBacktestEvaluationRecord) (string, []string) {
	reasons := []string{}
	if strings.TrimSpace(comparison.ComparisonID) == "" {
		return storage.MarketOpsBacktestPromotionReadinessBlocked, []string{"comparison evidence is required"}
	}
	switch comparison.Recommendation {
	case storage.MarketOpsBacktestCalibrationRecommendationRegression:
		return storage.MarketOpsBacktestPromotionReadinessRegressionDetected, []string{"baseline comparison is a regression_candidate"}
	case storage.MarketOpsBacktestCalibrationRecommendationNeedsMoreData:
		return storage.MarketOpsBacktestPromotionReadinessNeedsMoreData, []string{"baseline comparison needs more data"}
	case storage.MarketOpsBacktestCalibrationRecommendationManualReview:
		reasons = append(reasons, "baseline comparison requires manual review")
	}
	if evaluation == nil || strings.TrimSpace(evaluation.EvaluationID) == "" {
		reasons = append(reasons, "label-aware evaluation is not attached")
		return storage.MarketOpsBacktestPromotionReadinessManualReviewRequired, reasons
	}
	if evaluation.Recommendation == storage.MarketOpsBacktestCalibrationRecommendationNeedsMoreData {
		return storage.MarketOpsBacktestPromotionReadinessNeedsMoreData, []string{"label-aware evaluation needs more data"}
	}
	if evaluation.LabelCoverage < marketOpsPromotionMinimumLabelCoverage {
		return storage.MarketOpsBacktestPromotionReadinessNeedsMoreData, []string{"label coverage is below review threshold"}
	}
	if evaluation.FalsePositive > 0 && evaluation.Precision < marketOpsPromotionMinimumPrecision {
		return storage.MarketOpsBacktestPromotionReadinessManualReviewRequired, []string{"false positives exist and precision is below review threshold"}
	}
	if evaluation.Recall < marketOpsPromotionMinimumRecall {
		reasons = append(reasons, "recall is below review threshold")
	}
	if len(reasons) > 0 {
		return storage.MarketOpsBacktestPromotionReadinessManualReviewRequired, reasons
	}
	return storage.MarketOpsBacktestPromotionReadinessReadyForReview, []string{"comparison and evaluation evidence meet review thresholds"}
}

func marketOpsBacktestPromotionEvaluationEvidence(evaluation *storage.MarketOpsBacktestEvaluationRecord) any {
	if evaluation == nil {
		return map[string]any{}
	}
	return map[string]any{"evaluation_id": evaluation.EvaluationID, "recommendation": evaluation.Recommendation, "recommendation_note": evaluation.RecommendationNote, "precision": evaluation.Precision, "recall": evaluation.Recall, "accuracy": evaluation.Accuracy, "label_coverage": evaluation.LabelCoverage, "candidate_count": evaluation.CandidateCount, "labeled_count": evaluation.LabeledCount, "true_positive": evaluation.TruePositive, "false_positive": evaluation.FalsePositive, "true_negative": evaluation.TrueNegative, "false_negative": evaluation.FalseNegative}
}

func marketOpsBacktestPromotionCandidateDecisionStatusAllowed(status string) bool {
	switch strings.TrimSpace(status) {
	case storage.MarketOpsBacktestPromotionCandidateStatusApprovedForPromotion, storage.MarketOpsBacktestPromotionCandidateStatusRejected, storage.MarketOpsBacktestPromotionCandidateStatusDeferred, storage.MarketOpsBacktestPromotionCandidateStatusSuperseded:
		return true
	default:
		return false
	}
}
