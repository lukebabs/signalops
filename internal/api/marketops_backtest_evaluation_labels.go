package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	marketOpsEvaluationLabelVersion = "marketops.eval_label.v1"
	marketOpsEvaluationLabelSource  = "g080_graph_proposal_decision"
)

type marketOpsBacktestEvaluationLabelSyncRequest struct {
	TenantID          string `json:"tenant_id"`
	AppID             string `json:"app_id"`
	Domain            string `json:"domain"`
	UseCase           string `json:"use_case"`
	Status            string `json:"status"`
	IncludeUnresolved bool   `json:"include_unresolved"`
	Limit             int    `json:"limit"`
	RequestedBy       string `json:"requested_by"`
}

type marketOpsBacktestEvaluationLabelSyncResponse struct {
	Synced int                                   `json:"synced"`
	Labels []marketOpsBacktestEvaluationLabelDTO `json:"labels"`
}

type marketOpsBacktestEvaluationLabelDTO struct {
	LabelID          string          `json:"label_id"`
	TenantID         string          `json:"tenant_id"`
	AppID            string          `json:"app_id"`
	Domain           string          `json:"domain"`
	UseCase          string          `json:"use_case"`
	SourceProposalID string          `json:"source_proposal_id"`
	ArtifactID       string          `json:"artifact_id"`
	SignalID         string          `json:"signal_id"`
	SubjectSymbol    string          `json:"subject_symbol"`
	CandidateType    string          `json:"candidate_type"`
	GraphFactKey     string          `json:"graph_fact_key"`
	DecisionStatus   string          `json:"decision_status"`
	Label            string          `json:"label"`
	LabelSource      string          `json:"label_source"`
	LabeledBy        string          `json:"labeled_by"`
	LabeledAt        time.Time       `json:"labeled_at"`
	LabelVersion     string          `json:"label_version"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func marketOpsBacktestEvaluationLabelResponse(record storage.MarketOpsBacktestEvaluationLabelRecord) marketOpsBacktestEvaluationLabelDTO {
	return marketOpsBacktestEvaluationLabelDTO{LabelID: record.LabelID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain, UseCase: record.UseCase, SourceProposalID: record.SourceProposalID, ArtifactID: record.ArtifactID, SignalID: record.SignalID, SubjectSymbol: record.SubjectSymbol, CandidateType: record.CandidateType, GraphFactKey: record.GraphFactKey, DecisionStatus: record.DecisionStatus, Label: record.Label, LabelSource: record.LabelSource, LabeledBy: record.LabeledBy, LabeledAt: record.LabeledAt, LabelVersion: record.LabelVersion, Metadata: json.RawMessage(jsonOrDefault(record.MetadataJSON, `{}`)), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsBacktestEvaluationLabelResponses(records []storage.MarketOpsBacktestEvaluationLabelRecord) []marketOpsBacktestEvaluationLabelDTO {
	responses := make([]marketOpsBacktestEvaluationLabelDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsBacktestEvaluationLabelResponse(record))
	}
	return responses
}

func marketOpsBacktestEvaluationLabelsFromProposals(proposals []storage.MarketOpsDSMGraphProposalRecord, actor string) ([]storage.MarketOpsBacktestEvaluationLabelRecord, error) {
	labels := make([]storage.MarketOpsBacktestEvaluationLabelRecord, 0, len(proposals))
	for _, proposal := range proposals {
		label, ok := marketOpsBacktestEvaluationLabelForStatus(proposal.Status)
		if !ok {
			continue
		}
		labeledAt := proposal.UpdatedAt
		if proposal.DecidedAt != nil && !proposal.DecidedAt.IsZero() {
			labeledAt = proposal.DecidedAt.UTC()
		}
		metadata, err := json.Marshal(map[string]any{"decision_note": proposal.DecisionNote, "proposal_updated_at": proposal.UpdatedAt, "proposal_confidence": proposal.Confidence})
		if err != nil {
			return nil, err
		}
		labels = append(labels, storage.MarketOpsBacktestEvaluationLabelRecord{LabelID: stableMarketOpsBacktestEvaluationLabelID(proposal.ProposalID, marketOpsEvaluationLabelVersion), TenantID: proposal.TenantID, AppID: proposal.AppID, Domain: proposal.Domain, UseCase: proposal.UseCase, SourceProposalID: proposal.ProposalID, ArtifactID: proposal.ArtifactID, SignalID: proposal.SignalID, SubjectSymbol: proposal.SubjectSymbol, CandidateType: proposal.CandidateType, GraphFactKey: marketOpsGraphFactKey(proposal), DecisionStatus: proposal.Status, Label: label, LabelSource: marketOpsEvaluationLabelSource, LabeledBy: firstNonEmptyBacktestValue(actor, proposal.ReviewedBy, "operator-local"), LabeledAt: labeledAt, LabelVersion: marketOpsEvaluationLabelVersion, MetadataJSON: metadata})
	}
	return labels, nil
}

func marketOpsBacktestEvaluationLabelForStatus(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case storage.MarketOpsDSMGraphProposalStatusAccepted:
		return "positive", true
	case storage.MarketOpsDSMGraphProposalStatusRejected:
		return "negative", true
	case storage.MarketOpsDSMGraphProposalStatusSuperseded:
		return "superseded", true
	case storage.MarketOpsDSMGraphProposalStatusProposed:
		return "unresolved", true
	default:
		return "", false
	}
}

func marketOpsGraphFactKey(proposal storage.MarketOpsDSMGraphProposalRecord) string {
	if strings.TrimSpace(proposal.NodeID) != "" {
		return "node:" + strings.TrimSpace(proposal.NodeID)
	}
	parts := []string{strings.TrimSpace(proposal.FromNode), strings.TrimSpace(proposal.Relationship), strings.TrimSpace(proposal.ToNode)}
	return "relationship:" + strings.Join(parts, "|")
}

func stableMarketOpsBacktestEvaluationLabelID(proposalID string, version string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(proposalID) + "\x00" + strings.TrimSpace(version)))
	return "btlabel_marketops_" + hex.EncodeToString(h[:])[:24]
}

func marketOpsBacktestEvaluationLabelSyncStatuses(req marketOpsBacktestEvaluationLabelSyncRequest) []string {
	status := strings.TrimSpace(req.Status)
	if status != "" {
		return []string{status}
	}
	statuses := []string{storage.MarketOpsDSMGraphProposalStatusAccepted, storage.MarketOpsDSMGraphProposalStatusRejected, storage.MarketOpsDSMGraphProposalStatusSuperseded}
	if req.IncludeUnresolved {
		statuses = append(statuses, storage.MarketOpsDSMGraphProposalStatusProposed)
	}
	return statuses
}
