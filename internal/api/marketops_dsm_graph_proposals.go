package api

import (
	"encoding/json"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type marketOpsDSMGraphProposalDTO struct {
	ProposalID    string          `json:"proposal_id"`
	TenantID      string          `json:"tenant_id"`
	AppID         string          `json:"app_id"`
	Domain        string          `json:"domain"`
	UseCase       string          `json:"use_case"`
	SourceID      string          `json:"source_id"`
	SourceAdapter string          `json:"source_adapter"`
	Dataset       string          `json:"dataset"`
	ArtifactID    string          `json:"artifact_id"`
	SignalID      string          `json:"signal_id"`
	SignalType    string          `json:"signal_type"`
	DetectorID    string          `json:"detector_id"`
	Severity      string          `json:"severity"`
	Confidence    float64         `json:"confidence"`
	EventIDs      []string        `json:"event_ids"`
	SubjectSymbol string          `json:"subject_symbol"`
	CandidateType string          `json:"candidate_type"`
	NodeID        string          `json:"node_id"`
	FromNode      string          `json:"from_node"`
	Relationship  string          `json:"relationship"`
	ToNode        string          `json:"to_node"`
	Labels        []string        `json:"labels"`
	Properties    json.RawMessage `json:"properties"`
	RawCandidate  json.RawMessage `json:"raw_candidate"`
	Status        string          `json:"status"`
	ReviewedBy    string          `json:"reviewed_by"`
	DecisionNote  string          `json:"decision_note"`
	DecidedAt     *time.Time      `json:"decided_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func marketOpsDSMGraphProposalResponse(record storage.MarketOpsDSMGraphProposalRecord) marketOpsDSMGraphProposalDTO {
	return marketOpsDSMGraphProposalDTO{
		ProposalID: record.ProposalID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase,
		SourceID: record.SourceID, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset, ArtifactID: record.ArtifactID, SignalID: record.SignalID,
		SignalType: record.SignalType, DetectorID: record.DetectorID, Severity: record.Severity, Confidence: record.Confidence, EventIDs: record.EventIDs,
		SubjectSymbol: record.SubjectSymbol, CandidateType: record.CandidateType, NodeID: record.NodeID, FromNode: record.FromNode, Relationship: record.Relationship,
		ToNode: record.ToNode, Labels: record.Labels, Properties: json.RawMessage(jsonOrDefault(record.PropertiesJSON, `{}`)), RawCandidate: json.RawMessage(jsonOrDefault(record.RawCandidate, `{}`)),
		Status: record.Status, ReviewedBy: record.ReviewedBy, DecisionNote: record.DecisionNote, DecidedAt: record.DecidedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func marketOpsDSMGraphProposalResponses(records []storage.MarketOpsDSMGraphProposalRecord) []marketOpsDSMGraphProposalDTO {
	responses := make([]marketOpsDSMGraphProposalDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsDSMGraphProposalResponse(record))
	}
	return responses
}

type graphProposalDecisionRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
	Actor  string `json:"actor"`
}
