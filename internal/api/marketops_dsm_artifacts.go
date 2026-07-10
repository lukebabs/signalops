package api

import (
	"encoding/json"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type marketOpsDSMArtifactDTO struct {
	ArtifactID        string          `json:"artifact_id"`
	TenantID          string          `json:"tenant_id"`
	AppID             string          `json:"app_id"`
	Domain            string          `json:"domain"`
	UseCase           string          `json:"use_case"`
	SourceID          string          `json:"source_id"`
	SourceAdapter     string          `json:"source_adapter"`
	Dataset           string          `json:"dataset"`
	SignalID          string          `json:"signal_id"`
	SignalType        string          `json:"signal_type"`
	DetectorID        string          `json:"detector_id"`
	Severity          string          `json:"severity"`
	Confidence        float64         `json:"confidence"`
	EventIDs          []string        `json:"event_ids"`
	SubjectSymbol     string          `json:"subject_symbol"`
	ArtifactType      string          `json:"artifact_type"`
	Artifact          json.RawMessage `json:"artifact"`
	SemanticEvidence  json.RawMessage `json:"semantic_evidence"`
	GraphTargets      json.RawMessage `json:"graph_targets"`
	SupportingMetrics json.RawMessage `json:"supporting_metrics"`
	QualityIssues     []string        `json:"quality_issues"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func marketOpsDSMArtifactResponse(record storage.MarketOpsDSMArtifactRecord) marketOpsDSMArtifactDTO {
	return marketOpsDSMArtifactDTO{
		ArtifactID: record.ArtifactID, TenantID: record.TenantID, AppID: record.AppID, Domain: record.Domain,
		UseCase: record.UseCase, SourceID: record.SourceID, SourceAdapter: record.SourceAdapter, Dataset: record.Dataset,
		SignalID: record.SignalID, SignalType: record.SignalType, DetectorID: record.DetectorID, Severity: record.Severity,
		Confidence: record.Confidence, EventIDs: record.EventIDs, SubjectSymbol: record.SubjectSymbol, ArtifactType: record.ArtifactType,
		Artifact:          json.RawMessage(jsonOrDefault(record.ArtifactJSON, `{}`)),
		SemanticEvidence:  json.RawMessage(jsonOrDefault(record.SemanticEvidenceJSON, `{}`)),
		GraphTargets:      json.RawMessage(jsonOrDefault(record.GraphTargetsJSON, `[]`)),
		SupportingMetrics: json.RawMessage(jsonOrDefault(record.SupportingMetrics, `{}`)),
		QualityIssues:     record.QualityIssues, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func marketOpsDSMArtifactResponses(records []storage.MarketOpsDSMArtifactRecord) []marketOpsDSMArtifactDTO {
	responses := make([]marketOpsDSMArtifactDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsDSMArtifactResponse(record))
	}
	return responses
}

func jsonOrDefault(raw []byte, fallback string) []byte {
	if len(raw) == 0 {
		return []byte(fallback)
	}
	return raw
}
