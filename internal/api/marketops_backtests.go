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

func firstNonEmptyBacktestValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
