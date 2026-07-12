package api

import (
	"encoding/json"
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
