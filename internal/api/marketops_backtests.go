package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type marketOpsBacktestCreateRequest struct {
	RunID                string   `json:"run_id"`
	TenantID             string   `json:"tenant_id"`
	SourceID             string   `json:"source_id"`
	SourceAdapter        string   `json:"source_adapter"`
	Dataset              string   `json:"dataset"`
	DetectorID           string   `json:"detector_id"`
	DetectorVersion      string   `json:"detector_version"`
	RequestedBy          string   `json:"requested_by"`
	WindowStart          string   `json:"window_start"`
	WindowEnd            string   `json:"window_end"`
	Symbols              []string `json:"symbols"`
	MaxRecords           int      `json:"max_records"`
	BatchSize            int      `json:"batch_size"`
	AutoAcceptConfidence float64  `json:"auto_accept_confidence"`
}

type marketOpsBacktestCreateResponse struct {
	BacktestRun marketOpsBacktestRunDTO `json:"backtest_run"`
	Metrics     json.RawMessage         `json:"metrics"`
}

type marketOpsBacktestRunDTO struct {
	RunID           string          `json:"run_id"`
	TenantID        string          `json:"tenant_id"`
	AppID           string          `json:"app_id"`
	Domain          string          `json:"domain"`
	UseCase         string          `json:"use_case"`
	SourceID        string          `json:"source_id"`
	SourceAdapter   string          `json:"source_adapter"`
	Dataset         string          `json:"dataset"`
	DetectorID      string          `json:"detector_id"`
	DetectorVersion string          `json:"detector_version"`
	Status          string          `json:"status"`
	RequestedBy     string          `json:"requested_by"`
	WindowStart     time.Time       `json:"window_start"`
	WindowEnd       time.Time       `json:"window_end"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Filters         json.RawMessage `json:"filters"`
	Parameters      json.RawMessage `json:"parameters"`
	Metrics         json.RawMessage `json:"metrics"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type marketOpsBacktestSignalDTO struct {
	RunID  string    `json:"run_id"`
	Signal signalDTO `json:"signal"`
}

type marketOpsBacktestGraphProposalDTO struct {
	RunID         string                       `json:"run_id"`
	GraphProposal marketOpsDSMGraphProposalDTO `json:"graph_proposal"`
}

type marketOpsBacktestPolicyResultDTO struct {
	RunID          string          `json:"run_id"`
	PolicyResultID string          `json:"policy_result_id"`
	ProposalID     string          `json:"proposal_id"`
	ArtifactID     string          `json:"artifact_id"`
	SignalID       string          `json:"signal_id"`
	TenantID       string          `json:"tenant_id"`
	SubjectSymbol  string          `json:"subject_symbol"`
	CandidateType  string          `json:"candidate_type"`
	Recommendation string          `json:"recommendation"`
	Reason         string          `json:"reason"`
	PolicyVersion  string          `json:"policy_version"`
	Confidence     float64         `json:"confidence"`
	DecisionInputs json.RawMessage `json:"decision_inputs"`
	CreatedAt      time.Time       `json:"created_at"`
}

func marketOpsBacktestRunResponse(record storage.MarketOpsBacktestRunRecord) marketOpsBacktestRunDTO {
	return marketOpsBacktestRunDTO{RunID: record.RunID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset, DetectorID: record.DetectorID, DetectorVersion: record.DetectorVersion, Status: record.Status, RequestedBy: record.RequestedBy, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd, StartedAt: record.StartedAt, CompletedAt: record.CompletedAt, Filters: json.RawMessage(jsonOrDefault(record.FiltersJSON, `{}`)), Parameters: json.RawMessage(jsonOrDefault(record.ParametersJSON, `{}`)), Metrics: json.RawMessage(jsonOrDefault(record.MetricsJSON, `{}`)), ErrorMessage: record.ErrorMessage, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsBacktestRunResponses(records []storage.MarketOpsBacktestRunRecord) []marketOpsBacktestRunDTO {
	responses := make([]marketOpsBacktestRunDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestRunResponse(record))
	}
	return responses
}

func marketOpsBacktestSignalResponses(records []storage.MarketOpsBacktestSignalRecord) []marketOpsBacktestSignalDTO {
	responses := make([]marketOpsBacktestSignalDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestSignalDTO{RunID: record.RunID, Signal: signalResponse(record.SignalLedgerRecord)})
	}
	return responses
}

func marketOpsBacktestGraphProposalResponses(records []storage.MarketOpsBacktestGraphProposalRecord) []marketOpsBacktestGraphProposalDTO {
	responses := make([]marketOpsBacktestGraphProposalDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestGraphProposalDTO{RunID: record.RunID, GraphProposal: marketOpsDSMGraphProposalResponse(record.MarketOpsDSMGraphProposalRecord)})
	}
	return responses
}

func marketOpsBacktestPolicyResultResponses(records []storage.MarketOpsBacktestPolicyResultRecord) []marketOpsBacktestPolicyResultDTO {
	responses := make([]marketOpsBacktestPolicyResultDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestPolicyResultDTO{RunID: record.RunID, PolicyResultID: record.PolicyResultID, ProposalID: record.ProposalID, ArtifactID: record.ArtifactID, SignalID: record.SignalID, TenantID: record.TenantID, SubjectSymbol: record.SubjectSymbol, CandidateType: record.CandidateType, Recommendation: record.Recommendation, Reason: record.Reason, PolicyVersion: record.PolicyVersion, Confidence: record.Confidence, DecisionInputs: json.RawMessage(jsonOrDefault(record.DecisionInputsJSON, `{}`)), CreatedAt: record.CreatedAt})
	}
	return responses
}

type marketOpsBacktestCalibrationSummaryCreateRequest struct {
	SummaryID   string `json:"summary_id"`
	TenantID    string `json:"tenant_id"`
	AppID       string `json:"app_id"`
	Domain      string `json:"domain"`
	UseCase     string `json:"use_case"`
	SourceID    string `json:"source_id"`
	Dataset     string `json:"dataset"`
	DetectorID  string `json:"detector_id"`
	Status      string `json:"status"`
	Limit       int    `json:"limit"`
	RequestedBy string `json:"requested_by"`
}

type marketOpsBacktestCalibrationSummaryDTO struct {
	SummaryID              string          `json:"summary_id"`
	TenantID               string          `json:"tenant_id"`
	AppID                  string          `json:"app_id"`
	Domain                 string          `json:"domain"`
	UseCase                string          `json:"use_case"`
	SourceID               string          `json:"source_id"`
	Dataset                string          `json:"dataset"`
	DetectorID             string          `json:"detector_id"`
	StatusFilter           string          `json:"status_filter"`
	RequestedBy            string          `json:"requested_by"`
	RunIDs                 []string        `json:"run_ids"`
	RunCount               int             `json:"run_count"`
	SucceededCount         int             `json:"succeeded_count"`
	FailedCount            int             `json:"failed_count"`
	ZeroInputCount         int             `json:"zero_input_count"`
	Scanned                int             `json:"scanned"`
	Signals                int             `json:"signals"`
	Artifacts              int             `json:"artifacts"`
	GraphProposals         int             `json:"graph_proposals"`
	PolicyResults          int             `json:"policy_results"`
	SignalYield            float64         `json:"signal_yield"`
	PolicyResultsPerSignal float64         `json:"policy_results_per_signal"`
	RecommendationCounts   json.RawMessage `json:"recommendation_counts"`
	RecommendationShares   json.RawMessage `json:"recommendation_shares"`
	DominantRecommendation json.RawMessage `json:"dominant_recommendation"`
	Filters                json.RawMessage `json:"filters"`
	Parameters             json.RawMessage `json:"parameters"`
	CreatedAt              time.Time       `json:"created_at"`
}

func marketOpsBacktestCalibrationSummaryResponse(record storage.MarketOpsBacktestCalibrationSummaryRecord) marketOpsBacktestCalibrationSummaryDTO {
	return marketOpsBacktestCalibrationSummaryDTO{SummaryID: record.SummaryID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceID: record.SourceID, Dataset: record.Dataset, DetectorID: record.DetectorID, StatusFilter: record.StatusFilter, RequestedBy: record.RequestedBy, RunIDs: record.RunIDs, RunCount: record.RunCount, SucceededCount: record.SucceededCount, FailedCount: record.FailedCount, ZeroInputCount: record.ZeroInputCount, Scanned: record.Scanned, Signals: record.Signals, Artifacts: record.Artifacts, GraphProposals: record.GraphProposals, PolicyResults: record.PolicyResults, SignalYield: record.SignalYield, PolicyResultsPerSignal: record.PolicyResultsPerSignal, RecommendationCounts: json.RawMessage(jsonOrDefault(record.RecommendationCounts, `{}`)), RecommendationShares: json.RawMessage(jsonOrDefault(record.RecommendationShares, `{}`)), DominantRecommendation: json.RawMessage(jsonOrDefault(record.DominantRecommendation, `{}`)), Filters: json.RawMessage(jsonOrDefault(record.FiltersJSON, `{}`)), Parameters: json.RawMessage(jsonOrDefault(record.ParametersJSON, `{}`)), CreatedAt: record.CreatedAt}
}

func marketOpsBacktestCalibrationSummaryResponses(records []storage.MarketOpsBacktestCalibrationSummaryRecord) []marketOpsBacktestCalibrationSummaryDTO {
	responses := make([]marketOpsBacktestCalibrationSummaryDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestCalibrationSummaryResponse(record))
	}
	return responses
}

type backtestMetricSnapshot struct {
	Scanned              int            `json:"scanned"`
	Signals              int            `json:"signals"`
	Artifacts            int            `json:"artifacts"`
	GraphProposals       int            `json:"graph_proposals"`
	PolicyResults        int            `json:"policy_results"`
	RecommendationCounts map[string]int `json:"recommendation_counts"`
}

func marketOpsBacktestRunFilterFromCalibrationRequest(req marketOpsBacktestCalibrationSummaryCreateRequest) storage.MarketOpsBacktestRunFilter {
	return storage.MarketOpsBacktestRunFilter{TenantID: strings.TrimSpace(req.TenantID), AppID: strings.TrimSpace(req.AppID), Domain: strings.TrimSpace(req.Domain), UseCase: strings.TrimSpace(req.UseCase), SourceID: strings.TrimSpace(req.SourceID), Dataset: strings.TrimSpace(req.Dataset), DetectorID: strings.TrimSpace(req.DetectorID), Status: strings.TrimSpace(req.Status), Limit: req.Limit}
}

func buildMarketOpsBacktestCalibrationSummary(summaryID string, requestedBy string, filter storage.MarketOpsBacktestRunFilter, runs []storage.MarketOpsBacktestRunRecord) (storage.MarketOpsBacktestCalibrationSummaryRecord, error) {
	recCounts := map[string]int{}
	runIDs := make([]string, 0, len(runs))
	var summary storage.MarketOpsBacktestCalibrationSummaryRecord
	summary.SummaryID = summaryID
	summary.TenantID = strings.TrimSpace(filter.TenantID)
	summary.AppID = strings.TrimSpace(filter.AppID)
	summary.Domain = strings.TrimSpace(filter.Domain)
	summary.UseCase = strings.TrimSpace(filter.UseCase)
	summary.SourceID = strings.TrimSpace(filter.SourceID)
	summary.Dataset = strings.TrimSpace(filter.Dataset)
	summary.DetectorID = strings.TrimSpace(filter.DetectorID)
	summary.StatusFilter = strings.TrimSpace(filter.Status)
	summary.RequestedBy = firstNonEmptyBacktestValue(requestedBy, "operator-local")
	for _, run := range runs {
		runIDs = append(runIDs, run.RunID)
		if summary.TenantID == "" {
			summary.TenantID = run.TenantID
		}
		if summary.AppID == "" {
			summary.AppID = run.AppID
		}
		if summary.Domain == "" {
			summary.Domain = run.Domain
		}
		if summary.UseCase == "" {
			summary.UseCase = run.UseCase
		}
		if summary.SourceID == "" {
			summary.SourceID = run.SourceID
		}
		if summary.Dataset == "" {
			summary.Dataset = run.Dataset
		}
		if summary.DetectorID == "" {
			summary.DetectorID = run.DetectorID
		}
		if run.Status == storage.RunStatusSucceeded {
			summary.SucceededCount++
		}
		if run.Status == storage.RunStatusFailed {
			summary.FailedCount++
		}
		var metrics backtestMetricSnapshot
		if len(run.MetricsJSON) > 0 {
			if err := json.Unmarshal(run.MetricsJSON, &metrics); err != nil {
				return storage.MarketOpsBacktestCalibrationSummaryRecord{}, err
			}
		}
		if run.Status == storage.RunStatusSucceeded && metrics.Scanned == 0 {
			summary.ZeroInputCount++
		}
		summary.Scanned += metrics.Scanned
		summary.Signals += metrics.Signals
		summary.Artifacts += metrics.Artifacts
		summary.GraphProposals += metrics.GraphProposals
		summary.PolicyResults += metrics.PolicyResults
		for key, count := range metrics.RecommendationCounts {
			recCounts[key] += count
		}
	}
	summary.RunIDs = runIDs
	summary.RunCount = len(runIDs)
	if summary.Scanned > 0 {
		summary.SignalYield = float64(summary.Signals) / float64(summary.Scanned)
	}
	if summary.Signals > 0 {
		summary.PolicyResultsPerSignal = float64(summary.PolicyResults) / float64(summary.Signals)
	}
	shares := map[string]float64{}
	dominantKey := ""
	dominantCount := 0
	for key, count := range recCounts {
		if summary.PolicyResults > 0 {
			shares[key] = float64(count) / float64(summary.PolicyResults)
		}
		if count > dominantCount || (count == dominantCount && (dominantKey == "" || key < dominantKey)) {
			dominantKey = key
			dominantCount = count
		}
	}
	dominant := map[string]any{}
	if dominantKey != "" {
		dominant = map[string]any{"key": dominantKey, "count": dominantCount, "share": shares[dominantKey]}
	}
	filters, _ := json.Marshal(map[string]any{"tenant_id": filter.TenantID, "app_id": filter.AppID, "domain": filter.Domain, "use_case": filter.UseCase, "source_id": filter.SourceID, "dataset": filter.Dataset, "detector_id": filter.DetectorID, "status": filter.Status, "limit": filter.Limit})
	params, _ := json.Marshal(map[string]any{"summary_version": "marketops.backtest.calibration_summary.v1"})
	summary.RecommendationCounts, _ = json.Marshal(recCounts)
	summary.RecommendationShares, _ = json.Marshal(shares)
	summary.DominantRecommendation, _ = json.Marshal(dominant)
	summary.FiltersJSON = filters
	summary.ParametersJSON = params
	return summary, nil
}

type marketOpsBacktestCalibrationReadinessCreateRequest struct {
	ReadinessID   string          `json:"readiness_id"`
	TenantID      string          `json:"tenant_id"`
	BaselineID    string          `json:"baseline_id"`
	ComparisonID  string          `json:"comparison_id"`
	EvaluationID  string          `json:"evaluation_id"`
	CandidateID   string          `json:"candidate_id"`
	DatasetScope  []string        `json:"dataset_scope"`
	UniverseGroup string          `json:"universe_group"`
	WindowStart   string          `json:"window_start"`
	WindowEnd     string          `json:"window_end"`
	Thresholds    json.RawMessage `json:"thresholds"`
	RequestedBy   string          `json:"requested_by"`
}

type marketOpsBacktestCalibrationReadinessDTO struct {
	ReadinessID       string          `json:"readiness_id"`
	TenantID          string          `json:"tenant_id"`
	AppID             string          `json:"app_id"`
	Domain            string          `json:"domain"`
	UseCase           string          `json:"use_case"`
	BaselineID        string          `json:"baseline_id"`
	ComparisonID      string          `json:"comparison_id"`
	EvaluationID      string          `json:"evaluation_id"`
	CandidateID       string          `json:"candidate_id"`
	DetectorID        string          `json:"detector_id"`
	DatasetScope      []string        `json:"dataset_scope"`
	UniverseGroup     string          `json:"universe_group"`
	WindowStart       *time.Time      `json:"window_start,omitempty"`
	WindowEnd         *time.Time      `json:"window_end,omitempty"`
	ReadinessStatus   string          `json:"readiness_status"`
	ReadinessReasons  []string        `json:"readiness_reasons"`
	CoverageMetrics   json.RawMessage `json:"coverage_metrics"`
	LabelMetrics      json.RawMessage `json:"label_metrics"`
	EvaluationMetrics json.RawMessage `json:"evaluation_metrics"`
	Thresholds        json.RawMessage `json:"thresholds"`
	Evidence          json.RawMessage `json:"evidence"`
	RequestedBy       string          `json:"requested_by"`
	CreatedAt         time.Time       `json:"created_at"`
}

type marketOpsBacktestReadinessThresholds struct {
	MinSymbolCoverageRatio   float64 `json:"min_symbol_coverage_ratio"`
	MinHistoricalWindows     int     `json:"min_historical_windows"`
	MinOptionsWindows        int     `json:"min_options_windows"`
	MinReviewedLabels        int     `json:"min_reviewed_labels"`
	MinLabelCoverageRatio    float64 `json:"min_label_coverage_ratio"`
	MaxConflictingLabelRatio float64 `json:"max_conflicting_label_ratio"`
}

func defaultMarketOpsBacktestReadinessThresholds() marketOpsBacktestReadinessThresholds {
	return marketOpsBacktestReadinessThresholds{MinSymbolCoverageRatio: 0.8, MinHistoricalWindows: 20, MinOptionsWindows: 10, MinReviewedLabels: 100, MinLabelCoverageRatio: 0.8, MaxConflictingLabelRatio: 0.05}
}

func marketOpsBacktestCalibrationReadinessResponse(record storage.MarketOpsBacktestCalibrationReadinessRecord) marketOpsBacktestCalibrationReadinessDTO {
	return marketOpsBacktestCalibrationReadinessDTO{ReadinessID: record.ReadinessID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, BaselineID: record.BaselineID, ComparisonID: record.ComparisonID, EvaluationID: record.EvaluationID, CandidateID: record.CandidateID, DetectorID: record.DetectorID, DatasetScope: record.DatasetScope, UniverseGroup: record.UniverseGroup, WindowStart: record.WindowStart, WindowEnd: record.WindowEnd, ReadinessStatus: record.ReadinessStatus, ReadinessReasons: record.ReadinessReasons, CoverageMetrics: json.RawMessage(jsonOrDefault(record.CoverageMetricsJSON, `{}`)), LabelMetrics: json.RawMessage(jsonOrDefault(record.LabelMetricsJSON, `{}`)), EvaluationMetrics: json.RawMessage(jsonOrDefault(record.EvaluationMetricsJSON, `{}`)), Thresholds: json.RawMessage(jsonOrDefault(record.ThresholdsJSON, `{}`)), Evidence: json.RawMessage(jsonOrDefault(record.EvidenceJSON, `{}`)), RequestedBy: record.RequestedBy, CreatedAt: record.CreatedAt}
}

func marketOpsBacktestCalibrationReadinessResponses(records []storage.MarketOpsBacktestCalibrationReadinessRecord) []marketOpsBacktestCalibrationReadinessDTO {
	responses := make([]marketOpsBacktestCalibrationReadinessDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestCalibrationReadinessResponse(record))
	}
	return responses
}

func buildMarketOpsBacktestCalibrationReadiness(readinessID string, actor string, req marketOpsBacktestCalibrationReadinessCreateRequest, baseline storage.MarketOpsBacktestCalibrationBaselineRecord, comparison storage.MarketOpsBacktestCalibrationComparisonRecord, evaluation *storage.MarketOpsBacktestEvaluationRecord, candidate *storage.MarketOpsBacktestPromotionCandidateRecord, runs []storage.MarketOpsBacktestRunRecord, labels []storage.MarketOpsBacktestEvaluationLabelRecord, universeAssets []storage.MarketOpsAssetRecord) (storage.MarketOpsBacktestCalibrationReadinessRecord, error) {
	thresholds := defaultMarketOpsBacktestReadinessThresholds()
	if len(req.Thresholds) > 0 {
		if err := json.Unmarshal(req.Thresholds, &thresholds); err != nil {
			return storage.MarketOpsBacktestCalibrationReadinessRecord{}, err
		}
	}
	if thresholds.MinSymbolCoverageRatio <= 0 {
		thresholds.MinSymbolCoverageRatio = 0.8
	}
	if thresholds.MinHistoricalWindows <= 0 {
		thresholds.MinHistoricalWindows = 20
	}
	if thresholds.MinOptionsWindows <= 0 {
		thresholds.MinOptionsWindows = 10
	}
	if thresholds.MinReviewedLabels <= 0 {
		thresholds.MinReviewedLabels = 100
	}
	if thresholds.MinLabelCoverageRatio <= 0 {
		thresholds.MinLabelCoverageRatio = 0.8
	}
	if thresholds.MaxConflictingLabelRatio <= 0 {
		thresholds.MaxConflictingLabelRatio = 0.05
	}

	status := storage.MarketOpsBacktestCalibrationReadinessReady
	reasons := []string{}
	datasets := cleanDatasetScope(req.DatasetScope)
	if len(datasets) == 0 {
		datasets = cleanDatasetScope([]string{comparison.Dataset, baseline.Dataset})
	}
	coverage := marketOpsBacktestCoverageMetrics(runs, datasets, universeAssets)
	labelMetrics := marketOpsBacktestLabelMetrics(labels)
	evalMetrics := map[string]any{"present": evaluation != nil}
	if evaluation != nil {
		evalMetrics = map[string]any{"present": true, "evaluation_id": evaluation.EvaluationID, "run_id": evaluation.RunID, "candidate_count": evaluation.CandidateCount, "labeled_count": evaluation.LabeledCount, "precision": evaluation.Precision, "recall": evaluation.Recall, "accuracy": evaluation.Accuracy, "label_coverage": evaluation.LabelCoverage, "false_positive": evaluation.FalsePositive, "false_negative": evaluation.FalseNegative, "recommendation": evaluation.Recommendation, "recommendation_note": evaluation.RecommendationNote}
	}

	coveredRatio := floatFromMap(coverage, "symbol_coverage_ratio")
	distinctWindows := intFromMap(coverage, "distinct_window_count")
	optionsWindows := intFromMap(coverage, "options_window_count")
	matchedLabels := intFromMap(labelMetrics, "matched_label_count")
	conflictRatio := floatFromMap(labelMetrics, "conflicting_label_ratio")

	if comparison.Recommendation == storage.MarketOpsBacktestCalibrationRecommendationRegression || (evaluation != nil && evaluation.Recommendation == storage.MarketOpsBacktestCalibrationRecommendationRegression) {
		status = storage.MarketOpsBacktestCalibrationReadinessRegressionDetected
		reasons = append(reasons, "comparison or evaluation recommendation indicates regression")
	}
	if comparison.ComparisonID == "" || baseline.BaselineID == "" {
		status = storage.MarketOpsBacktestCalibrationReadinessBlocked
		reasons = append(reasons, "baseline and comparison evidence are required")
	}
	if len(runs) == 0 || coveredRatio < thresholds.MinSymbolCoverageRatio || distinctWindows < thresholds.MinHistoricalWindows {
		if status == storage.MarketOpsBacktestCalibrationReadinessReady {
			status = storage.MarketOpsBacktestCalibrationReadinessNeedsMoreHistoricalData
		}
		reasons = append(reasons, "historical Top 50/window coverage is below readiness thresholds")
	}
	if containsString(datasets, "options_contracts_daily") && optionsWindows < thresholds.MinOptionsWindows {
		if status == storage.MarketOpsBacktestCalibrationReadinessReady {
			status = storage.MarketOpsBacktestCalibrationReadinessNeedsMoreHistoricalData
		}
		reasons = append(reasons, "options daily window coverage is below readiness threshold")
	}
	if evaluation == nil || matchedLabels < thresholds.MinReviewedLabels || (evaluation != nil && evaluation.LabelCoverage < thresholds.MinLabelCoverageRatio) || (evaluation != nil && evaluation.Recommendation == storage.MarketOpsBacktestCalibrationRecommendationNeedsMoreData) {
		if status == storage.MarketOpsBacktestCalibrationReadinessReady {
			status = storage.MarketOpsBacktestCalibrationReadinessNeedsMoreLabels
		}
		reasons = append(reasons, "reviewed label volume or label coverage is below readiness thresholds")
	}
	if conflictRatio > thresholds.MaxConflictingLabelRatio {
		if status == storage.MarketOpsBacktestCalibrationReadinessReady || status == storage.MarketOpsBacktestCalibrationReadinessNeedsMoreLabels {
			status = storage.MarketOpsBacktestCalibrationReadinessLabelQualityBlocked
		}
		reasons = append(reasons, "conflicting labels exceed readiness threshold")
	}
	if status == storage.MarketOpsBacktestCalibrationReadinessReady {
		reasons = append(reasons, "historical coverage, label volume, and evaluation evidence meet calibration readiness thresholds")
	}

	thresholdJSON, _ := json.Marshal(thresholds)
	coverageJSON, _ := json.Marshal(coverage)
	labelJSON, _ := json.Marshal(labelMetrics)
	evalJSON, _ := json.Marshal(evalMetrics)
	evidenceJSON, _ := json.Marshal(map[string]any{"baseline_id": baseline.BaselineID, "comparison_id": comparison.ComparisonID, "evaluation_id": strings.TrimSpace(req.EvaluationID), "candidate_id": strings.TrimSpace(req.CandidateID), "deployment_block": "calibration readiness is advisory and does not deploy runtime policy"})
	var windowStart, windowEnd *time.Time
	if parsed, ok := parseOptionalRFC3339(req.WindowStart); ok {
		windowStart = &parsed
	}
	if parsed, ok := parseOptionalRFC3339(req.WindowEnd); ok {
		windowEnd = &parsed
	}
	return storage.MarketOpsBacktestCalibrationReadinessRecord{ReadinessID: strings.TrimSpace(readinessID), TenantID: strings.TrimSpace(req.TenantID), AppID: baseline.AppID, Domain: baseline.Domain, UseCase: baseline.UseCase, BaselineID: baseline.BaselineID, ComparisonID: comparison.ComparisonID, EvaluationID: strings.TrimSpace(req.EvaluationID), CandidateID: strings.TrimSpace(req.CandidateID), DetectorID: firstNonEmptyBacktestValue(comparison.DetectorID, baseline.DetectorID), DatasetScope: datasets, UniverseGroup: strings.TrimSpace(req.UniverseGroup), WindowStart: windowStart, WindowEnd: windowEnd, ReadinessStatus: status, ReadinessReasons: reasons, CoverageMetricsJSON: coverageJSON, LabelMetricsJSON: labelJSON, EvaluationMetricsJSON: evalJSON, ThresholdsJSON: thresholdJSON, EvidenceJSON: evidenceJSON, RequestedBy: firstNonEmptyBacktestValue(actor, "operator-local")}, nil
}

func marketOpsBacktestCoverageMetrics(runs []storage.MarketOpsBacktestRunRecord, datasets []string, universeAssets []storage.MarketOpsAssetRecord) map[string]any {
	coveredSymbols := map[string]struct{}{}
	windowKeys := map[string]struct{}{}
	optionsWindowKeys := map[string]struct{}{}
	datasetCounts := map[string]int{}
	scanned := 0
	for _, run := range runs {
		if run.Status != storage.RunStatusSucceeded {
			continue
		}
		datasetCounts[run.Dataset]++
		windowKeys[run.WindowStart.UTC().Format("2006-01-02")] = struct{}{}
		if run.Dataset == "options_contracts_daily" {
			optionsWindowKeys[run.WindowStart.UTC().Format("2006-01-02")] = struct{}{}
		}
		for _, symbol := range symbolsFromBacktestRun(run) {
			coveredSymbols[symbol] = struct{}{}
		}
		var metrics backtestMetricSnapshot
		_ = json.Unmarshal(run.MetricsJSON, &metrics)
		scanned += metrics.Scanned
	}
	universeCount := len(universeAssets)
	if universeCount == 0 {
		universeCount = len(coveredSymbols)
	}
	ratio := 0.0
	if universeCount > 0 {
		ratio = float64(len(coveredSymbols)) / float64(universeCount)
	}
	return map[string]any{"run_count": len(runs), "succeeded_run_count": len(runs), "scanned": scanned, "covered_symbol_count": len(coveredSymbols), "universe_symbol_count": universeCount, "symbol_coverage_ratio": ratio, "distinct_window_count": len(windowKeys), "options_window_count": len(optionsWindowKeys), "dataset_counts": datasetCounts, "dataset_scope": datasets}
}

func marketOpsBacktestLabelMetrics(labels []storage.MarketOpsBacktestEvaluationLabelRecord) map[string]any {
	byFact := map[string]map[string]struct{}{}
	labelCounts := map[string]int{}
	for _, label := range labels {
		labelCounts[label.Label]++
		key := label.GraphFactKey
		if key == "" {
			key = label.SourceProposalID
		}
		if byFact[key] == nil {
			byFact[key] = map[string]struct{}{}
		}
		byFact[key][label.Label] = struct{}{}
	}
	conflicts := 0
	for _, labelsForFact := range byFact {
		if len(labelsForFact) > 1 {
			conflicts++
		}
	}
	ratio := 0.0
	if len(byFact) > 0 {
		ratio = float64(conflicts) / float64(len(byFact))
	}
	return map[string]any{"matched_label_count": len(labels), "distinct_graph_fact_count": len(byFact), "conflicting_graph_fact_count": conflicts, "conflicting_label_ratio": ratio, "label_counts": labelCounts}
}

func symbolsFromBacktestRun(run storage.MarketOpsBacktestRunRecord) []string {
	payload := struct {
		Symbols []string `json:"symbols"`
	}{}
	_ = json.Unmarshal(run.FiltersJSON, &payload)
	return cleanSymbols(payload.Symbols)
}

func cleanDatasetScope(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intFromMap(values map[string]any, key string) int {
	switch v := values[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func floatFromMap(values map[string]any, key string) float64 {
	switch v := values[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

func parseOptionalRFC3339(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func firstNonEmptyBacktestValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type marketOpsBacktestCalibrationBaselineCreateRequest struct {
	BaselineID  string          `json:"baseline_id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SummaryID   string          `json:"summary_id"`
	Scope       json.RawMessage `json:"scope"`
	Status      string          `json:"status"`
	CreatedBy   string          `json:"created_by"`
}

type marketOpsBacktestCalibrationBaselineDTO struct {
	BaselineID  string          `json:"baseline_id"`
	TenantID    string          `json:"tenant_id"`
	AppID       string          `json:"app_id"`
	Domain      string          `json:"domain"`
	UseCase     string          `json:"use_case"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SummaryID   string          `json:"summary_id"`
	DetectorID  string          `json:"detector_id"`
	Dataset     string          `json:"dataset"`
	Scope       json.RawMessage `json:"scope"`
	Status      string          `json:"status"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type marketOpsBacktestCalibrationComparisonCreateRequest struct {
	ComparisonID       string `json:"comparison_id"`
	TenantID           string `json:"tenant_id"`
	BaselineID         string `json:"baseline_id"`
	CandidateSummaryID string `json:"candidate_summary_id"`
	CreatedBy          string `json:"created_by"`
}

type marketOpsBacktestCalibrationComparisonDTO struct {
	ComparisonID         string          `json:"comparison_id"`
	TenantID             string          `json:"tenant_id"`
	BaselineID           string          `json:"baseline_id"`
	BaselineSummaryID    string          `json:"baseline_summary_id"`
	CandidateSummaryID   string          `json:"candidate_summary_id"`
	DetectorID           string          `json:"detector_id"`
	Dataset              string          `json:"dataset"`
	ComparisonMetrics    json.RawMessage `json:"comparison_metrics"`
	Recommendation       string          `json:"recommendation"`
	RecommendationReason string          `json:"recommendation_reason"`
	CreatedBy            string          `json:"created_by"`
	CreatedAt            time.Time       `json:"created_at"`
}

func marketOpsBacktestCalibrationBaselineResponse(record storage.MarketOpsBacktestCalibrationBaselineRecord) marketOpsBacktestCalibrationBaselineDTO {
	return marketOpsBacktestCalibrationBaselineDTO{BaselineID: record.BaselineID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, Name: record.Name, Description: record.Description, SummaryID: record.SummaryID, DetectorID: record.DetectorID, Dataset: record.Dataset, Scope: json.RawMessage(jsonOrDefault(record.ScopeJSON, `{}`)), Status: record.Status, CreatedBy: record.CreatedBy, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsBacktestCalibrationBaselineResponses(records []storage.MarketOpsBacktestCalibrationBaselineRecord) []marketOpsBacktestCalibrationBaselineDTO {
	responses := make([]marketOpsBacktestCalibrationBaselineDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestCalibrationBaselineResponse(record))
	}
	return responses
}

func marketOpsBacktestCalibrationComparisonResponse(record storage.MarketOpsBacktestCalibrationComparisonRecord) marketOpsBacktestCalibrationComparisonDTO {
	return marketOpsBacktestCalibrationComparisonDTO{ComparisonID: record.ComparisonID, TenantID: record.TenantID, BaselineID: record.BaselineID, BaselineSummaryID: record.BaselineSummaryID, CandidateSummaryID: record.CandidateSummaryID, DetectorID: record.DetectorID, Dataset: record.Dataset, ComparisonMetrics: json.RawMessage(jsonOrDefault(record.ComparisonMetricsJSON, `{}`)), Recommendation: record.Recommendation, RecommendationReason: record.RecommendationReason, CreatedBy: record.CreatedBy, CreatedAt: record.CreatedAt}
}

func marketOpsBacktestCalibrationComparisonResponses(records []storage.MarketOpsBacktestCalibrationComparisonRecord) []marketOpsBacktestCalibrationComparisonDTO {
	responses := make([]marketOpsBacktestCalibrationComparisonDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestCalibrationComparisonResponse(record))
	}
	return responses
}

func marketOpsBacktestCalibrationBaselineFromRequest(req marketOpsBacktestCalibrationBaselineCreateRequest, actor string, summary storage.MarketOpsBacktestCalibrationSummaryRecord) storage.MarketOpsBacktestCalibrationBaselineRecord {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = storage.MarketOpsBacktestCalibrationBaselineStatusActive
	}
	return storage.MarketOpsBacktestCalibrationBaselineRecord{BaselineID: strings.TrimSpace(req.BaselineID), TenantID: strings.TrimSpace(req.TenantID), AppID: summary.AppID, Domain: summary.Domain, UseCase: summary.UseCase, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), SummaryID: summary.SummaryID, DetectorID: summary.DetectorID, Dataset: summary.Dataset, ScopeJSON: []byte(jsonOrDefault(req.Scope, `{}`)), Status: status, CreatedBy: firstNonEmptyBacktestValue(actor, "operator-local")}
}

type marketOpsBacktestCalibrationComparisonSnapshot struct {
	SummaryID              string             `json:"summary_id"`
	RunCount               int                `json:"run_count"`
	ZeroInputCount         int                `json:"zero_input_count"`
	ZeroInputRate          float64            `json:"zero_input_rate"`
	Scanned                int                `json:"scanned"`
	Signals                int                `json:"signals"`
	PolicyResults          int                `json:"policy_results"`
	SignalYield            float64            `json:"signal_yield"`
	PolicyResultsPerSignal float64            `json:"policy_results_per_signal"`
	RecommendationShares   map[string]float64 `json:"recommendation_shares"`
	DominantRecommendation string             `json:"dominant_recommendation"`
}

type marketOpsBacktestCalibrationComparisonMetrics struct {
	Baseline  marketOpsBacktestCalibrationComparisonSnapshot `json:"baseline"`
	Candidate marketOpsBacktestCalibrationComparisonSnapshot `json:"candidate"`
	Deltas    map[string]any                                 `json:"deltas"`
}

func buildMarketOpsBacktestCalibrationComparison(comparisonID string, actor string, baseline storage.MarketOpsBacktestCalibrationBaselineRecord, baselineSummary storage.MarketOpsBacktestCalibrationSummaryRecord, candidate storage.MarketOpsBacktestCalibrationSummaryRecord) (storage.MarketOpsBacktestCalibrationComparisonRecord, error) {
	base := marketOpsBacktestCalibrationComparisonSnapshotFromSummary(baselineSummary)
	cand := marketOpsBacktestCalibrationComparisonSnapshotFromSummary(candidate)
	deltas := map[string]any{
		"run_count_delta":                          cand.RunCount - base.RunCount,
		"zero_input_rate_delta":                    cand.ZeroInputRate - base.ZeroInputRate,
		"scanned_delta":                            cand.Scanned - base.Scanned,
		"signal_yield_delta":                       cand.SignalYield - base.SignalYield,
		"policy_results_per_signal_delta":          cand.PolicyResultsPerSignal - base.PolicyResultsPerSignal,
		"auto_accept_candidate_share_delta":        recommendationShare(cand, "auto_accept_candidate") - recommendationShare(base, "auto_accept_candidate"),
		"auto_reject_candidate_share_delta":        recommendationShare(cand, "auto_reject_candidate") - recommendationShare(base, "auto_reject_candidate"),
		"manual_review_required_share_delta":       recommendationShare(cand, "manual_review_required") - recommendationShare(base, "manual_review_required"),
		"supersede_existing_candidate_share_delta": recommendationShare(cand, "supersede_existing_candidate") - recommendationShare(base, "supersede_existing_candidate"),
		"dominant_recommendation_changed":          cand.DominantRecommendation != base.DominantRecommendation,
	}
	metrics, err := json.Marshal(marketOpsBacktestCalibrationComparisonMetrics{Baseline: base, Candidate: cand, Deltas: deltas})
	if err != nil {
		return storage.MarketOpsBacktestCalibrationComparisonRecord{}, err
	}
	recommendation, reason := marketOpsBacktestCalibrationComparisonRecommendation(base, cand)
	return storage.MarketOpsBacktestCalibrationComparisonRecord{ComparisonID: strings.TrimSpace(comparisonID), TenantID: baseline.TenantID, BaselineID: baseline.BaselineID, BaselineSummaryID: baselineSummary.SummaryID, CandidateSummaryID: candidate.SummaryID, DetectorID: firstNonEmptyBacktestValue(candidate.DetectorID, baseline.DetectorID), Dataset: firstNonEmptyBacktestValue(candidate.Dataset, baseline.Dataset), ComparisonMetricsJSON: metrics, Recommendation: recommendation, RecommendationReason: reason, CreatedBy: firstNonEmptyBacktestValue(actor, "operator-local")}, nil
}

func marketOpsBacktestCalibrationComparisonSnapshotFromSummary(summary storage.MarketOpsBacktestCalibrationSummaryRecord) marketOpsBacktestCalibrationComparisonSnapshot {
	shares := map[string]float64{}
	_ = json.Unmarshal(summary.RecommendationShares, &shares)
	dominant := struct {
		Key string `json:"key"`
	}{}
	_ = json.Unmarshal(summary.DominantRecommendation, &dominant)
	zeroInputRate := 0.0
	if summary.RunCount > 0 {
		zeroInputRate = float64(summary.ZeroInputCount) / float64(summary.RunCount)
	}
	return marketOpsBacktestCalibrationComparisonSnapshot{SummaryID: summary.SummaryID, RunCount: summary.RunCount, ZeroInputCount: summary.ZeroInputCount, ZeroInputRate: zeroInputRate, Scanned: summary.Scanned, Signals: summary.Signals, PolicyResults: summary.PolicyResults, SignalYield: summary.SignalYield, PolicyResultsPerSignal: summary.PolicyResultsPerSignal, RecommendationShares: shares, DominantRecommendation: dominant.Key}
}

func marketOpsBacktestCalibrationComparisonRecommendation(base marketOpsBacktestCalibrationComparisonSnapshot, cand marketOpsBacktestCalibrationComparisonSnapshot) (string, string) {
	manualReviewDelta := recommendationShare(cand, "manual_review_required") - recommendationShare(base, "manual_review_required")
	autoAcceptDelta := recommendationShare(cand, "auto_accept_candidate") - recommendationShare(base, "auto_accept_candidate")
	if cand.RunCount < 1 || cand.Scanned == 0 {
		return storage.MarketOpsBacktestCalibrationRecommendationNeedsMoreData, "candidate summary has no usable input coverage"
	}
	if cand.ZeroInputRate > base.ZeroInputRate+0.2 {
		return storage.MarketOpsBacktestCalibrationRecommendationRegression, "candidate zero-input rate increased materially"
	}
	if cand.SignalYield < base.SignalYield*0.8 || manualReviewDelta > 0.1 || autoAcceptDelta > 0.1 {
		return storage.MarketOpsBacktestCalibrationRecommendationManualReview, "candidate drift requires operator review before promotion"
	}
	if manualReviewDelta < -0.05 && cand.SignalYield >= base.SignalYield {
		return storage.MarketOpsBacktestCalibrationRecommendationImprovement, "candidate reduces manual review share without lowering signal yield"
	}
	if absFloat(cand.SignalYield-base.SignalYield) < 0.02 && absFloat(manualReviewDelta) < 0.05 && absFloat(autoAcceptDelta) < 0.05 {
		return storage.MarketOpsBacktestCalibrationRecommendationNeutral, "candidate is within baseline tolerance bands"
	}
	return storage.MarketOpsBacktestCalibrationRecommendationNeutral, "candidate deltas do not cross promotion or regression thresholds"
}

func recommendationShare(snapshot marketOpsBacktestCalibrationComparisonSnapshot, key string) float64 {
	if snapshot.RecommendationShares == nil {
		return 0
	}
	return snapshot.RecommendationShares[key]
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
