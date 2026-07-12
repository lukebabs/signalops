package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const marketOpsEvaluationScoringVersion = "marketops.eval_scoring.v1"

type marketOpsBacktestEvaluationCreateRequest struct {
	EvaluationID string `json:"evaluation_id"`
	TenantID     string `json:"tenant_id"`
	RunID        string `json:"run_id"`
	LabelSource  string `json:"label_source"`
	RequestedBy  string `json:"requested_by"`
}

type marketOpsBacktestEvaluationDTO struct {
	EvaluationID       string          `json:"evaluation_id"`
	TenantID           string          `json:"tenant_id"`
	AppID              string          `json:"app_id"`
	Domain             string          `json:"domain"`
	UseCase            string          `json:"use_case"`
	RunID              string          `json:"run_id"`
	DetectorID         string          `json:"detector_id"`
	Dataset            string          `json:"dataset"`
	LabelSource        string          `json:"label_source"`
	LabelVersion       string          `json:"label_version"`
	ScoringVersion     string          `json:"scoring_version"`
	RequestedBy        string          `json:"requested_by"`
	CandidateCount     int             `json:"candidate_count"`
	LabeledCount       int             `json:"labeled_count"`
	PositiveCount      int             `json:"positive_count"`
	NegativeCount      int             `json:"negative_count"`
	SupersededCount    int             `json:"superseded_count"`
	UnresolvedCount    int             `json:"unresolved_count"`
	TruePositive       int             `json:"true_positive"`
	FalsePositive      int             `json:"false_positive"`
	TrueNegative       int             `json:"true_negative"`
	FalseNegative      int             `json:"false_negative"`
	ManualReviewCount  int             `json:"manual_review_count"`
	UnscoredCount      int             `json:"unscored_count"`
	Precision          float64         `json:"precision"`
	Recall             float64         `json:"recall"`
	Specificity        float64         `json:"specificity"`
	Accuracy           float64         `json:"accuracy"`
	LabelCoverage      float64         `json:"label_coverage"`
	Recommendation     string          `json:"recommendation"`
	RecommendationNote string          `json:"recommendation_note"`
	Metrics            json.RawMessage `json:"metrics"`
	CreatedAt          time.Time       `json:"created_at"`
}

func marketOpsBacktestEvaluationResponse(record storage.MarketOpsBacktestEvaluationRecord) marketOpsBacktestEvaluationDTO {
	return marketOpsBacktestEvaluationDTO{EvaluationID: record.EvaluationID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, RunID: record.RunID, DetectorID: record.DetectorID, Dataset: record.Dataset, LabelSource: record.LabelSource, LabelVersion: record.LabelVersion, ScoringVersion: record.ScoringVersion, RequestedBy: record.RequestedBy, CandidateCount: record.CandidateCount, LabeledCount: record.LabeledCount, PositiveCount: record.PositiveCount, NegativeCount: record.NegativeCount, SupersededCount: record.SupersededCount, UnresolvedCount: record.UnresolvedCount, TruePositive: record.TruePositive, FalsePositive: record.FalsePositive, TrueNegative: record.TrueNegative, FalseNegative: record.FalseNegative, ManualReviewCount: record.ManualReviewCount, UnscoredCount: record.UnscoredCount, Precision: record.Precision, Recall: record.Recall, Specificity: record.Specificity, Accuracy: record.Accuracy, LabelCoverage: record.LabelCoverage, Recommendation: record.Recommendation, RecommendationNote: record.RecommendationNote, Metrics: json.RawMessage(jsonOrDefault(record.MetricsJSON, `{}`)), CreatedAt: record.CreatedAt}
}

func marketOpsBacktestEvaluationResponses(records []storage.MarketOpsBacktestEvaluationRecord) []marketOpsBacktestEvaluationDTO {
	responses := make([]marketOpsBacktestEvaluationDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestEvaluationResponse(record))
	}
	return responses
}

func buildMarketOpsBacktestEvaluation(evaluationID string, requestedBy string, run storage.MarketOpsBacktestRunRecord, proposals []storage.MarketOpsBacktestGraphProposalRecord, policyResults []storage.MarketOpsBacktestPolicyResultRecord, labels []storage.MarketOpsBacktestEvaluationLabelRecord) (storage.MarketOpsBacktestEvaluationRecord, error) {
	labelByFact := map[string]storage.MarketOpsBacktestEvaluationLabelRecord{}
	for _, label := range labels {
		key := strings.TrimSpace(label.GraphFactKey)
		if key != "" {
			labelByFact[key] = label
		}
	}
	policyByProposal := map[string]storage.MarketOpsBacktestPolicyResultRecord{}
	for _, policy := range policyResults {
		policyByProposal[policy.ProposalID] = policy
	}
	samples := []map[string]any{}
	record := storage.MarketOpsBacktestEvaluationRecord{EvaluationID: strings.TrimSpace(evaluationID), TenantID: run.TenantID, AppID: run.AppID, Domain: run.Domain, UseCase: run.UseCase, RunID: run.RunID, DetectorID: run.DetectorID, Dataset: run.Dataset, LabelSource: marketOpsEvaluationLabelSource, LabelVersion: marketOpsEvaluationLabelVersion, ScoringVersion: marketOpsEvaluationScoringVersion, RequestedBy: firstNonEmptyBacktestValue(requestedBy, run.RequestedBy, "operator-local")}
	record.CandidateCount = len(proposals)
	for _, proposalRecord := range proposals {
		proposal := proposalRecord.MarketOpsDSMGraphProposalRecord
		factKey := marketOpsGraphFactKey(proposal)
		label, ok := labelByFact[factKey]
		policy := policyByProposal[proposal.ProposalID]
		if !ok {
			record.UnscoredCount++
			continue
		}
		record.LabeledCount++
		switch label.Label {
		case "positive":
			record.PositiveCount++
			if policy.Recommendation == storage.MarketOpsBacktestPolicyAutoAcceptCandidate {
				record.TruePositive++
			} else if policy.Recommendation == storage.MarketOpsBacktestPolicyAutoRejectCandidate {
				record.FalseNegative++
			} else {
				record.ManualReviewCount++
			}
		case "negative":
			record.NegativeCount++
			if policy.Recommendation == storage.MarketOpsBacktestPolicyAutoAcceptCandidate {
				record.FalsePositive++
			} else if policy.Recommendation == storage.MarketOpsBacktestPolicyAutoRejectCandidate {
				record.TrueNegative++
			} else {
				record.ManualReviewCount++
			}
		case "superseded":
			record.SupersededCount++
		case "unresolved":
			record.UnresolvedCount++
		}
		if len(samples) < 20 {
			samples = append(samples, map[string]any{"proposal_id": proposal.ProposalID, "graph_fact_key": factKey, "label": label.Label, "recommendation": policy.Recommendation})
		}
	}
	record.Precision = ratio(record.TruePositive, record.TruePositive+record.FalsePositive)
	record.Recall = ratio(record.TruePositive, record.TruePositive+record.FalseNegative)
	record.Specificity = ratio(record.TrueNegative, record.TrueNegative+record.FalsePositive)
	record.Accuracy = ratio(record.TruePositive+record.TrueNegative, record.TruePositive+record.TrueNegative+record.FalsePositive+record.FalseNegative)
	record.LabelCoverage = ratio(record.LabeledCount, record.CandidateCount)
	record.Recommendation, record.RecommendationNote = marketOpsBacktestEvaluationRecommendation(record)
	metrics, err := json.Marshal(map[string]any{"matched_samples": samples, "scoring_notes": []string{"manual_review_required and supersede_candidate recommendations are not counted as automatic true/false outcomes"}})
	if err != nil {
		return storage.MarketOpsBacktestEvaluationRecord{}, err
	}
	record.MetricsJSON = metrics
	return record, nil
}

func marketOpsBacktestEvaluationRecommendation(record storage.MarketOpsBacktestEvaluationRecord) (string, string) {
	if record.LabeledCount == 0 {
		return storage.MarketOpsBacktestCalibrationRecommendationNeedsMoreData, "no matching evaluation labels were found for this run"
	}
	if record.FalsePositive > 0 || record.Precision < 0.8 {
		return storage.MarketOpsBacktestCalibrationRecommendationManualReview, "false positives or low precision require operator review"
	}
	if record.Recall >= 0.8 && record.Precision >= 0.9 && record.ManualReviewCount == 0 {
		return storage.MarketOpsBacktestCalibrationRecommendationImprovement, "automatic recommendations align with available labels"
	}
	return storage.MarketOpsBacktestCalibrationRecommendationNeutral, "label-aware scores are within review tolerance"
}

func ratio(num int, den int) float64 {
	if den <= 0 {
		return 0
	}
	return float64(num) / float64(den)
}
