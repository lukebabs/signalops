package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

type algorithmDefinitionRequest struct {
	AlgorithmID     string          `json:"algorithm_id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	AlgorithmType   string          `json:"algorithm_type"`
	RuntimeType     string          `json:"runtime_type"`
	InputFeatures   []string        `json:"input_features"`
	InputEventTypes []string        `json:"input_event_types"`
	OutputSchema    json.RawMessage `json:"output_schema"`
	ConfigSchema    json.RawMessage `json:"config_schema"`
	DefaultConfig   json.RawMessage `json:"default_config"`
	Version         string          `json:"version"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
}

type algorithmDefinitionDTO struct {
	AlgorithmID     string          `json:"algorithm_id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	AlgorithmType   string          `json:"algorithm_type"`
	RuntimeType     string          `json:"runtime_type"`
	InputFeatures   []string        `json:"input_features"`
	InputEventTypes []string        `json:"input_event_types"`
	OutputSchema    json.RawMessage `json:"output_schema"`
	ConfigSchema    json.RawMessage `json:"config_schema"`
	DefaultConfig   json.RawMessage `json:"default_config"`
	Version         string          `json:"version"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type algorithmExecutionRequestCreate struct {
	ExecutionRequestID string          `json:"execution_request_id"`
	TenantID           string          `json:"tenant_id"`
	AlgorithmID        string          `json:"algorithm_id"`
	AlgorithmVersion   string          `json:"algorithm_version"`
	EventIDs           []string        `json:"event_ids"`
	FeatureRefs        []string        `json:"feature_refs"`
	EntityRefs         []string        `json:"entity_refs"`
	WindowRef          string          `json:"window_ref"`
	Config             json.RawMessage `json:"config"`
	CorrelationID      string          `json:"correlation_id"`
	RequestedBy        string          `json:"requested_by"`
}

type algorithmExecutionRequestDTO struct {
	ExecutionRequestID string          `json:"execution_request_id"`
	TenantID           string          `json:"tenant_id"`
	AlgorithmID        string          `json:"algorithm_id"`
	AlgorithmVersion   string          `json:"algorithm_version"`
	EventIDs           []string        `json:"event_ids"`
	FeatureRefs        []string        `json:"feature_refs"`
	EntityRefs         []string        `json:"entity_refs"`
	WindowRef          string          `json:"window_ref"`
	Config             json.RawMessage `json:"config"`
	CorrelationID      string          `json:"correlation_id"`
	Status             string          `json:"status"`
	RequestedBy        string          `json:"requested_by"`
	Result             json.RawMessage `json:"result"`
	ErrorMessage       string          `json:"error_message"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type algorithmResultDTO struct {
	AlgorithmResultID  string          `json:"algorithm_result_id"`
	TenantID           string          `json:"tenant_id"`
	AlgorithmID        string          `json:"algorithm_id"`
	AlgorithmVersion   string          `json:"algorithm_version"`
	ExecutionRequestID string          `json:"execution_request_id"`
	ResultType         string          `json:"result_type"`
	Score              float64         `json:"score"`
	Confidence         float64         `json:"confidence"`
	Severity           string          `json:"severity"`
	ResultPayload      json.RawMessage `json:"result_payload"`
	SourceEventIDs     []string        `json:"source_event_ids"`
	FeatureValueIDs    []string        `json:"feature_value_ids"`
	EvidenceRefs       []string        `json:"evidence_refs"`
	CorrelationID      string          `json:"correlation_id"`
	CreatedAt          time.Time       `json:"created_at"`
}

func algorithmDefinitionRecord(req algorithmDefinitionRequest) storage.AlgorithmDefinitionRecord {
	return storage.AlgorithmDefinitionRecord{AlgorithmID: strings.TrimSpace(req.AlgorithmID), TenantID: strings.TrimSpace(req.TenantID), Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), AlgorithmType: strings.TrimSpace(req.AlgorithmType), RuntimeType: firstNonEmptyBacktestValue(req.RuntimeType, storage.AlgorithmRuntimePythonPlugin), InputFeatures: cleanStrings(req.InputFeatures), InputEventTypes: cleanStrings(req.InputEventTypes), OutputSchema: algorithmJSONOrDefaultObject(req.OutputSchema), ConfigSchema: algorithmJSONOrDefaultObject(req.ConfigSchema), DefaultConfig: algorithmJSONOrDefaultObject(req.DefaultConfig), Version: strings.TrimSpace(req.Version), Status: firstNonEmptyBacktestValue(req.Status, storage.AlgorithmDefinitionStatusDraft), MetadataJSON: algorithmJSONOrDefaultObject(req.Metadata)}
}

func algorithmExecutionRequestRecord(req algorithmExecutionRequestCreate, actor string) storage.AlgorithmExecutionRequestRecord {
	id := strings.TrimSpace(req.ExecutionRequestID)
	if id == "" {
		id = newID("algexec")
	}
	correlationID := strings.TrimSpace(req.CorrelationID)
	if correlationID == "" {
		correlationID = id
	}
	return storage.AlgorithmExecutionRequestRecord{ExecutionRequestID: id, TenantID: strings.TrimSpace(req.TenantID), AlgorithmID: strings.TrimSpace(req.AlgorithmID), AlgorithmVersion: strings.TrimSpace(req.AlgorithmVersion), EventIDs: cleanStrings(req.EventIDs), FeatureRefs: cleanStrings(req.FeatureRefs), EntityRefs: cleanStrings(req.EntityRefs), WindowRef: strings.TrimSpace(req.WindowRef), ConfigJSON: algorithmJSONOrDefaultObject(req.Config), CorrelationID: correlationID, Status: storage.AlgorithmExecutionStatusQueued, RequestedBy: firstNonEmptyBacktestValue(actor, "operator-local"), ResultJSON: []byte(`{}`)}
}

func algorithmDefinitionResponse(record storage.AlgorithmDefinitionRecord) algorithmDefinitionDTO {
	return algorithmDefinitionDTO{AlgorithmID: record.AlgorithmID, TenantID: record.TenantID, Name: record.Name, Description: record.Description, AlgorithmType: record.AlgorithmType, RuntimeType: record.RuntimeType, InputFeatures: record.InputFeatures, InputEventTypes: record.InputEventTypes, OutputSchema: json.RawMessage(jsonOrDefault(record.OutputSchema, `{}`)), ConfigSchema: json.RawMessage(jsonOrDefault(record.ConfigSchema, `{}`)), DefaultConfig: json.RawMessage(jsonOrDefault(record.DefaultConfig, `{}`)), Version: record.Version, Status: record.Status, Metadata: json.RawMessage(jsonOrDefault(record.MetadataJSON, `{}`)), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func algorithmDefinitionResponses(records []storage.AlgorithmDefinitionRecord) []algorithmDefinitionDTO {
	out := make([]algorithmDefinitionDTO, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmDefinitionResponse(record))
	}
	return out
}

func algorithmExecutionRequestResponse(record storage.AlgorithmExecutionRequestRecord) algorithmExecutionRequestDTO {
	return algorithmExecutionRequestDTO{ExecutionRequestID: record.ExecutionRequestID, TenantID: record.TenantID, AlgorithmID: record.AlgorithmID, AlgorithmVersion: record.AlgorithmVersion, EventIDs: record.EventIDs, FeatureRefs: record.FeatureRefs, EntityRefs: record.EntityRefs, WindowRef: record.WindowRef, Config: json.RawMessage(jsonOrDefault(record.ConfigJSON, `{}`)), CorrelationID: record.CorrelationID, Status: record.Status, RequestedBy: record.RequestedBy, Result: json.RawMessage(jsonOrDefault(record.ResultJSON, `{}`)), ErrorMessage: record.ErrorMessage, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func algorithmExecutionRequestResponses(records []storage.AlgorithmExecutionRequestRecord) []algorithmExecutionRequestDTO {
	out := make([]algorithmExecutionRequestDTO, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmExecutionRequestResponse(record))
	}
	return out
}

func algorithmResultResponse(record storage.AlgorithmResultRecord) algorithmResultDTO {
	return algorithmResultDTO{AlgorithmResultID: record.AlgorithmResultID, TenantID: record.TenantID, AlgorithmID: record.AlgorithmID, AlgorithmVersion: record.AlgorithmVersion, ExecutionRequestID: record.ExecutionRequestID, ResultType: record.ResultType, Score: record.Score, Confidence: record.Confidence, Severity: record.Severity, ResultPayload: json.RawMessage(jsonOrDefault(record.ResultPayloadJSON, `{}`)), SourceEventIDs: record.SourceEventIDs, FeatureValueIDs: record.FeatureValueIDs, EvidenceRefs: record.EvidenceRefs, CorrelationID: record.CorrelationID, CreatedAt: record.CreatedAt}
}

func algorithmResultResponses(records []storage.AlgorithmResultRecord) []algorithmResultDTO {
	out := make([]algorithmResultDTO, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmResultResponse(record))
	}
	return out
}

func algorithmJSONOrDefaultObject(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
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
