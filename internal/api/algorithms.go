package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
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

type algorithmSignalProposalDTO struct {
	ProposalID         string          `json:"proposal_id"`
	TenantID           string          `json:"tenant_id"`
	AlgorithmResultID  string          `json:"algorithm_result_id"`
	AlgorithmID        string          `json:"algorithm_id"`
	AlgorithmVersion   string          `json:"algorithm_version"`
	ExecutionRequestID string          `json:"execution_request_id"`
	ProposedSignalType string          `json:"proposed_signal_type"`
	Status             string          `json:"status"`
	Score              float64         `json:"score"`
	Confidence         float64         `json:"confidence"`
	Severity           string          `json:"severity"`
	ProposalPayload    json.RawMessage `json:"proposal_payload"`
	Rationale          json.RawMessage `json:"rationale"`
	SourceEventIDs     []string        `json:"source_event_ids"`
	EvidenceRefs       []string        `json:"evidence_refs"`
	CorrelationID      string          `json:"correlation_id"`
	CreatedBy          string          `json:"created_by"`
	ReviewedBy         string          `json:"reviewed_by"`
	DecisionNote       string          `json:"decision_note"`
	DecidedAt          *time.Time      `json:"decided_at,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type algorithmSignalProposalDecisionRequest struct {
	TenantID string          `json:"tenant_id"`
	Status   string          `json:"status"`
	Note     string          `json:"note"`
	Actor    string          `json:"actor"`
	Metadata json.RawMessage `json:"metadata"`
}

type algorithmSignalMaterializationRequest struct {
	TenantID       string          `json:"tenant_id"`
	PolicyVersion  string          `json:"policy_version"`
	RequestedBy    string          `json:"requested_by"`
	IdempotencyKey string          `json:"idempotency_key"`
	Metadata       json.RawMessage `json:"metadata"`
}

type algorithmSignalProposalSummaryDTO struct {
	TenantID                    string         `json:"tenant_id"`
	TotalProposals              int            `json:"total_proposals"`
	ProposedCount               int            `json:"proposed_count"`
	ReviewedCount               int            `json:"reviewed_count"`
	RejectedCount               int            `json:"rejected_count"`
	SupersededCount             int            `json:"superseded_count"`
	ReviewedRatio               float64        `json:"reviewed_ratio"`
	HighCriticalUnreviewedCount int            `json:"high_critical_unreviewed_count"`
	StatusCounts                map[string]int `json:"status_counts"`
	SeverityCounts              map[string]int `json:"severity_counts"`
	ProposedSignalTypeCounts    map[string]int `json:"proposed_signal_type_counts"`
	AlgorithmIDCounts           map[string]int `json:"algorithm_id_counts"`
	ReviewerCounts              map[string]int `json:"reviewer_counts"`
}

type algorithmSignalProposalMaterializationPreflightDTO struct {
	TenantID                    string                                                   `json:"tenant_id"`
	PolicyVersion               string                                                   `json:"policy_version"`
	TotalProposals              int                                                      `json:"total_proposals"`
	EligibleCount               int                                                      `json:"eligible_count"`
	DuplicateRiskCount          int                                                      `json:"duplicate_risk_count"`
	BlockedCount                int                                                      `json:"blocked_count"`
	InvalidCount                int                                                      `json:"invalid_count"`
	WouldWriteCount             int                                                      `json:"would_write_count"`
	ReviewedRatio               float64                                                  `json:"reviewed_ratio"`
	MinReviewedRatio            float64                                                  `json:"min_reviewed_ratio"`
	ReviewCoverageSatisfied     bool                                                     `json:"review_coverage_satisfied"`
	HighCriticalUnreviewedCount int                                                      `json:"high_critical_unreviewed_count"`
	GlobalBlockingReasons       map[string]int                                           `json:"global_blocking_reasons"`
	ItemReasonCounts            map[string]int                                           `json:"item_reason_counts"`
	Items                       []algorithmSignalProposalMaterializationPreflightItemDTO `json:"items"`
}

type algorithmSignalProposalMaterializationPreflightItemDTO struct {
	ProposalID            string   `json:"proposal_id"`
	AlgorithmResultID     string   `json:"algorithm_result_id"`
	AlgorithmID           string   `json:"algorithm_id"`
	ExecutionRequestID    string   `json:"execution_request_id"`
	ProposedSignalType    string   `json:"proposed_signal_type"`
	Status                string   `json:"status"`
	Severity              string   `json:"severity"`
	Confidence            float64  `json:"confidence"`
	PreflightStatus       string   `json:"preflight_status"`
	Reasons               []string `json:"reasons"`
	DuplicateSignalIDs    []string `json:"duplicate_signal_ids"`
	SourceEventIDs        []string `json:"source_event_ids"`
	WouldWrite            bool     `json:"would_write"`
	MaterializationPolicy string   `json:"materialization_policy"`
}

type algorithmSignalMaterializationDTO struct {
	MaterializationID            string          `json:"materialization_id"`
	TenantID                     string          `json:"tenant_id"`
	ProposalID                   string          `json:"proposal_id"`
	AlgorithmResultID            string          `json:"algorithm_result_id"`
	ExecutionRequestID           string          `json:"execution_request_id"`
	AlgorithmID                  string          `json:"algorithm_id"`
	AlgorithmVersion             string          `json:"algorithm_version"`
	ProposedSignalType           string          `json:"proposed_signal_type"`
	SignalID                     string          `json:"signal_id"`
	MaterializationStatus        string          `json:"materialization_status"`
	MaterializationPolicyVersion string          `json:"materialization_policy_version"`
	IdempotencyKey               string          `json:"idempotency_key"`
	DuplicateOfSignalID          string          `json:"duplicate_of_signal_id"`
	RequestedBy                  string          `json:"requested_by"`
	RequestedAt                  time.Time       `json:"requested_at"`
	StartedAt                    *time.Time      `json:"started_at,omitempty"`
	CompletedAt                  *time.Time      `json:"completed_at,omitempty"`
	FailedAt                     *time.Time      `json:"failed_at,omitempty"`
	ErrorCode                    string          `json:"error_code"`
	ErrorMessage                 string          `json:"error_message"`
	RequestMetadata              json.RawMessage `json:"request_metadata"`
	PreflightSnapshot            json.RawMessage `json:"preflight_snapshot"`
	SignalPayloadPreview         json.RawMessage `json:"signal_payload_preview"`
	CreatedAt                    time.Time       `json:"created_at"`
	UpdatedAt                    time.Time       `json:"updated_at"`
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

func algorithmSignalProposalResponse(record storage.AlgorithmSignalProposalRecord) algorithmSignalProposalDTO {
	return algorithmSignalProposalDTO{ProposalID: record.ProposalID, TenantID: record.TenantID, AlgorithmResultID: record.AlgorithmResultID, AlgorithmID: record.AlgorithmID, AlgorithmVersion: record.AlgorithmVersion, ExecutionRequestID: record.ExecutionRequestID, ProposedSignalType: record.ProposedSignalType, Status: record.Status, Score: record.Score, Confidence: record.Confidence, Severity: record.Severity, ProposalPayload: json.RawMessage(jsonOrDefault(record.ProposalPayloadJSON, `{}`)), Rationale: json.RawMessage(jsonOrDefault(record.RationaleJSON, `{}`)), SourceEventIDs: record.SourceEventIDs, EvidenceRefs: record.EvidenceRefs, CorrelationID: record.CorrelationID, CreatedBy: record.CreatedBy, ReviewedBy: record.ReviewedBy, DecisionNote: record.DecisionNote, DecidedAt: record.DecidedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func algorithmSignalProposalResponses(records []storage.AlgorithmSignalProposalRecord) []algorithmSignalProposalDTO {
	out := make([]algorithmSignalProposalDTO, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmSignalProposalResponse(record))
	}
	return out
}

func algorithmSignalProposalSummaryResponse(record storage.AlgorithmSignalProposalSummaryRecord) algorithmSignalProposalSummaryDTO {
	return algorithmSignalProposalSummaryDTO{TenantID: record.TenantID, TotalProposals: record.TotalProposals, ProposedCount: record.ProposedCount, ReviewedCount: record.ReviewedCount, RejectedCount: record.RejectedCount, SupersededCount: record.SupersededCount, ReviewedRatio: record.ReviewedRatio, HighCriticalUnreviewedCount: record.HighCriticalUnreviewedCount, StatusCounts: mapOrEmpty(record.StatusCounts), SeverityCounts: mapOrEmpty(record.SeverityCounts), ProposedSignalTypeCounts: mapOrEmpty(record.ProposedSignalTypeCounts), AlgorithmIDCounts: mapOrEmpty(record.AlgorithmIDCounts), ReviewerCounts: mapOrEmpty(record.ReviewerCounts)}
}

func algorithmSignalMaterializationResponse(record storage.AlgorithmSignalMaterializationRecord) algorithmSignalMaterializationDTO {
	return algorithmSignalMaterializationDTO{MaterializationID: record.MaterializationID, TenantID: record.TenantID, ProposalID: record.ProposalID, AlgorithmResultID: record.AlgorithmResultID, ExecutionRequestID: record.ExecutionRequestID, AlgorithmID: record.AlgorithmID, AlgorithmVersion: record.AlgorithmVersion, ProposedSignalType: record.ProposedSignalType, SignalID: record.SignalID, MaterializationStatus: record.MaterializationStatus, MaterializationPolicyVersion: record.MaterializationPolicyVersion, IdempotencyKey: record.IdempotencyKey, DuplicateOfSignalID: record.DuplicateOfSignalID, RequestedBy: record.RequestedBy, RequestedAt: record.RequestedAt, StartedAt: record.StartedAt, CompletedAt: record.CompletedAt, FailedAt: record.FailedAt, ErrorCode: record.ErrorCode, ErrorMessage: record.ErrorMessage, RequestMetadata: json.RawMessage(jsonOrDefault(record.RequestMetadataJSON, `{}`)), PreflightSnapshot: json.RawMessage(jsonOrDefault(record.PreflightSnapshotJSON, `{}`)), SignalPayloadPreview: json.RawMessage(jsonOrDefault(record.SignalPayloadPreviewJSON, `{}`)), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func algorithmSignalMaterializationResponses(records []storage.AlgorithmSignalMaterializationRecord) []algorithmSignalMaterializationDTO {
	out := make([]algorithmSignalMaterializationDTO, 0, len(records))
	for _, record := range records {
		out = append(out, algorithmSignalMaterializationResponse(record))
	}
	return out
}

func algorithmMaterializationPolicyVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "algorithm_materialization.v1"
	}
	return value
}

func algorithmMaterializationDigest(tenantID string, proposal storage.AlgorithmSignalProposalRecord, policyVersion string) string {
	events := append([]string{}, proposal.SourceEventIDs...)
	sort.Strings(events)
	raw := strings.Join([]string{strings.TrimSpace(tenantID), strings.TrimSpace(proposal.ProposalID), strings.TrimSpace(policyVersion), strings.Join(events, ","), strings.TrimSpace(proposal.ProposedSignalType)}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func algorithmMaterializationID(digest string) string {
	if len(digest) > 24 {
		digest = digest[:24]
	}
	return "algmat_" + digest
}

func algorithmMaterializationSignalID(digest string) string {
	if len(digest) > 24 {
		digest = digest[:24]
	}
	return "sig_alg_" + digest
}

func algorithmMaterializationIdempotencyKey(digest string, clientKey string) string {
	clientKey = strings.TrimSpace(clientKey)
	if clientKey == "" {
		return "algmat_idem_" + digest
	}
	return "algmat_client_" + clientKey
}

func algorithmSignalMaterializationBase(tenantID string, proposal storage.AlgorithmSignalProposalRecord, policyVersion string, digest string, requestedBy string, requestedAt time.Time, req algorithmSignalMaterializationRequest) storage.AlgorithmSignalMaterializationRecord {
	return storage.AlgorithmSignalMaterializationRecord{MaterializationID: algorithmMaterializationID(digest), TenantID: tenantID, ProposalID: proposal.ProposalID, AlgorithmResultID: proposal.AlgorithmResultID, ExecutionRequestID: proposal.ExecutionRequestID, AlgorithmID: proposal.AlgorithmID, AlgorithmVersion: proposal.AlgorithmVersion, ProposedSignalType: proposal.ProposedSignalType, MaterializationStatus: storage.AlgorithmSignalMaterializationStatusRequested, MaterializationPolicyVersion: policyVersion, IdempotencyKey: algorithmMaterializationIdempotencyKey(digest, req.IdempotencyKey), RequestedBy: firstNonEmptyBacktestValue(requestedBy, req.RequestedBy, "operator-local"), RequestedAt: requestedAt, RequestMetadataJSON: algorithmJSONOrDefaultObject(req.Metadata), PreflightSnapshotJSON: []byte(`{}`), SignalPayloadPreviewJSON: []byte(`{}`)}
}

func algorithmSignalRecordFromMaterialization(proposal storage.AlgorithmSignalProposalRecord, result storage.AlgorithmResultRecord, materialization storage.AlgorithmSignalMaterializationRecord, signalID string, now time.Time) storage.SignalLedgerRecord {
	signalTime := proposal.CreatedAt
	if signalTime.IsZero() {
		signalTime = result.CreatedAt
	}
	if signalTime.IsZero() {
		signalTime = now
	}
	metrics, _ := json.Marshal(map[string]any{"algorithm_result_id": result.AlgorithmResultID, "algorithm_result_type": result.ResultType, "score": proposal.Score, "result_score": result.Score, "proposal_payload": json.RawMessage(jsonOrDefault(proposal.ProposalPayloadJSON, `{}`)), "result_payload": json.RawMessage(jsonOrDefault(result.ResultPayloadJSON, `{}`))})
	semantic, _ := json.Marshal([]map[string]any{{"kind": "algorithm_signal_materialization", "materialization_id": materialization.MaterializationID, "proposal_id": proposal.ProposalID, "algorithm_result_id": result.AlgorithmResultID, "algorithm_id": proposal.AlgorithmID, "algorithm_version": proposal.AlgorithmVersion, "score": proposal.Score, "confidence": proposal.Confidence, "severity": proposal.Severity, "reviewed_by": proposal.ReviewedBy, "decision_note": proposal.DecisionNote, "policy_version": materialization.MaterializationPolicyVersion}})
	evidence, _ := json.Marshal([]map[string]any{{"kind": "algorithm_signal_proposal", "proposal_id": proposal.ProposalID}, {"kind": "algorithm_result", "algorithm_result_id": result.AlgorithmResultID}, {"kind": "algorithm_signal_materialization", "materialization_id": materialization.MaterializationID}, {"kind": "source_event_ids", "event_ids": proposal.SourceEventIDs}})
	recommendation, _ := json.Marshal(map[string]any{"action": "review", "source": "algorithm_materialization", "policy_version": materialization.MaterializationPolicyVersion})
	eventJSON, _ := json.Marshal(map[string]any{"schema_version": "signal.v1", "signal_id": signalID, "signal_type": proposal.ProposedSignalType, "source": "algorithm_signal_materialization", "materialization_id": materialization.MaterializationID, "proposal_id": proposal.ProposalID, "algorithm_result_id": result.AlgorithmResultID, "event_ids": proposal.SourceEventIDs})
	return storage.SignalLedgerRecord{SignalID: signalID, TenantID: proposal.TenantID, SourceID: "algorithm_signal_materialization", AppID: "signalops", Domain: "algorithms", UseCase: "algorithm_signal_materialization", SourceDomain: "algorithms", SourceAdapter: "signalops.algorithms", IngestionMode: "materialization", Dataset: "algorithm_signal_proposals", EventIDs: proposal.SourceEventIDs, ArtifactIDs: []string{}, SignalType: proposal.ProposedSignalType, DetectorID: proposal.AlgorithmID, DetectorVersion: proposal.AlgorithmVersion, ModelVersion: materialization.MaterializationPolicyVersion, SignalTime: signalTime, ObservationTime: signalTime, EffectiveTime: signalTime, ProcessingTime: now, WindowStart: signalTime, WindowEnd: signalTime, Confidence: proposal.Confidence, Severity: proposal.Severity, EntitiesJSON: []byte(`[]`), SupportingMetrics: metrics, GraphTargetsJSON: []byte(`[]`), SemanticEvidenceJSON: semantic, EvidenceJSON: evidence, RecommendationJSON: recommendation, CorrelationID: proposal.CorrelationID, TraceID: proposal.ExecutionRequestID, CausationID: proposal.ProposalID, BrokerTopic: "signalops.internal.algorithm_materializations.v1", EventJSON: eventJSON}
}

func algorithmMaterializationPreview(signal storage.SignalLedgerRecord) []byte {
	preview, _ := json.Marshal(map[string]any{"signal_id": signal.SignalID, "signal_type": signal.SignalType, "detector_id": signal.DetectorID, "detector_version": signal.DetectorVersion, "model_version": signal.ModelVersion, "event_ids": signal.EventIDs, "confidence": signal.Confidence, "severity": signal.Severity})
	return preview
}

func algorithmPreflightSnapshot(item algorithmSignalProposalMaterializationPreflightItemDTO) []byte {
	snapshot, _ := json.Marshal(item)
	return snapshot
}

func algorithmSignalProposalMaterializationPreflightResponse(tenantID string, policyVersion string, minReviewedRatio float64, proposals []storage.AlgorithmSignalProposalRecord, results map[string]storage.AlgorithmResultRecord, signals []storage.SignalLedgerRecord, summary storage.AlgorithmSignalProposalSummaryRecord) algorithmSignalProposalMaterializationPreflightDTO {
	policyVersion = strings.TrimSpace(policyVersion)
	if policyVersion == "" {
		policyVersion = "materialization_preflight.v1"
	}
	if minReviewedRatio <= 0 || minReviewedRatio > 1 {
		minReviewedRatio = 1
	}
	globalReasons := map[string]int{}
	if summary.TotalProposals > 0 && summary.ReviewedRatio < minReviewedRatio {
		globalReasons["review_coverage_below_threshold"] = 1
	}
	if summary.HighCriticalUnreviewedCount > 0 {
		globalReasons["high_critical_unreviewed_proposals"] = summary.HighCriticalUnreviewedCount
	}
	globalBlocked := len(globalReasons) > 0
	out := algorithmSignalProposalMaterializationPreflightDTO{TenantID: tenantID, PolicyVersion: policyVersion, ReviewedRatio: summary.ReviewedRatio, MinReviewedRatio: minReviewedRatio, ReviewCoverageSatisfied: !globalBlocked, HighCriticalUnreviewedCount: summary.HighCriticalUnreviewedCount, GlobalBlockingReasons: globalReasons, ItemReasonCounts: map[string]int{}, Items: []algorithmSignalProposalMaterializationPreflightItemDTO{}}
	for _, proposal := range proposals {
		item := algorithmSignalProposalMaterializationPreflightItemDTO{ProposalID: proposal.ProposalID, AlgorithmResultID: proposal.AlgorithmResultID, AlgorithmID: proposal.AlgorithmID, ExecutionRequestID: proposal.ExecutionRequestID, ProposedSignalType: proposal.ProposedSignalType, Status: proposal.Status, Severity: proposal.Severity, Confidence: proposal.Confidence, SourceEventIDs: proposal.SourceEventIDs, DuplicateSignalIDs: []string{}, MaterializationPolicy: policyVersion}
		reasons := []string{}
		if proposal.Status != storage.AlgorithmSignalProposalStatusReviewed {
			reasons = append(reasons, proposalStatusPreflightReason(proposal.Status))
		}
		if len(proposal.SourceEventIDs) == 0 {
			reasons = append(reasons, "missing_source_events")
		}
		if !json.Valid([]byte(jsonOrDefault(proposal.ProposalPayloadJSON, `{}`))) {
			reasons = append(reasons, "invalid_proposal_payload")
		}
		if !json.Valid([]byte(jsonOrDefault(proposal.RationaleJSON, `{}`))) {
			reasons = append(reasons, "invalid_rationale")
		}
		result, ok := results[proposal.AlgorithmResultID]
		if !ok {
			reasons = append(reasons, "missing_algorithm_result")
		} else {
			if strings.TrimSpace(result.TenantID) != strings.TrimSpace(proposal.TenantID) {
				reasons = append(reasons, "algorithm_result_tenant_mismatch")
			}
			if strings.TrimSpace(result.AlgorithmID) != strings.TrimSpace(proposal.AlgorithmID) {
				reasons = append(reasons, "algorithm_result_algorithm_mismatch")
			}
			if strings.TrimSpace(result.ExecutionRequestID) != strings.TrimSpace(proposal.ExecutionRequestID) {
				reasons = append(reasons, "algorithm_result_execution_mismatch")
			}
		}
		item.DuplicateSignalIDs = duplicateSignalIDsForProposal(proposal, signals)
		if len(item.DuplicateSignalIDs) > 0 {
			reasons = append(reasons, "duplicate_signal_event_overlap")
		}
		item.Reasons = uniqueStrings(reasons)
		for _, reason := range item.Reasons {
			out.ItemReasonCounts[reason]++
		}
		item.PreflightStatus = "eligible"
		if hasAnyString(item.Reasons, "missing_source_events", "invalid_proposal_payload", "invalid_rationale", "missing_algorithm_result", "algorithm_result_tenant_mismatch", "algorithm_result_algorithm_mismatch", "algorithm_result_execution_mismatch") {
			item.PreflightStatus = "invalid"
			out.InvalidCount++
		} else if len(item.DuplicateSignalIDs) > 0 {
			item.PreflightStatus = "duplicate_risk"
			out.DuplicateRiskCount++
		} else if len(item.Reasons) > 0 || globalBlocked {
			item.PreflightStatus = "blocked"
			out.BlockedCount++
		} else {
			out.EligibleCount++
		}
		item.WouldWrite = item.PreflightStatus == "eligible" && !globalBlocked
		if item.WouldWrite {
			out.WouldWriteCount++
		}
		out.Items = append(out.Items, item)
	}
	out.TotalProposals = len(out.Items)
	return out
}

func proposalStatusPreflightReason(status string) string {
	switch strings.TrimSpace(status) {
	case storage.AlgorithmSignalProposalStatusProposed:
		return "unreviewed_proposal"
	case storage.AlgorithmSignalProposalStatusRejected:
		return "rejected_proposal"
	case storage.AlgorithmSignalProposalStatusSuperseded:
		return "superseded_proposal"
	default:
		return "unsupported_proposal_status"
	}
}

func duplicateSignalIDsForProposal(proposal storage.AlgorithmSignalProposalRecord, signals []storage.SignalLedgerRecord) []string {
	sourceEvents := stringSet(proposal.SourceEventIDs)
	out := []string{}
	for _, signal := range signals {
		if strings.TrimSpace(signal.TenantID) != strings.TrimSpace(proposal.TenantID) {
			continue
		}
		if strings.TrimSpace(signal.SignalType) != strings.TrimSpace(proposal.ProposedSignalType) {
			continue
		}
		for _, eventID := range signal.EventIDs {
			if _, ok := sourceEvents[strings.TrimSpace(eventID)]; ok {
				out = append(out, signal.SignalID)
				break
			}
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	out := []string{}
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

func hasAnyString(values []string, needles ...string) bool {
	set := stringSet(values)
	for _, needle := range needles {
		if _, ok := set[needle]; ok {
			return true
		}
	}
	return false
}

func mapOrEmpty(values map[string]int) map[string]int {
	if values == nil {
		return map[string]int{}
	}
	return values
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

type algorithmExecutionSummaryDTO struct {
	ExecutionRequest algorithmExecutionRequestDTO `json:"execution_request"`
	ResultCount      int                          `json:"result_count"`
	SeverityCounts   map[string]int               `json:"severity_counts"`
	MaxScore         float64                      `json:"max_score"`
	MaxConfidence    float64                      `json:"max_confidence"`
	TopResults       []algorithmResultDTO         `json:"top_results"`
}

func algorithmExecutionSummaryResponse(request storage.AlgorithmExecutionRequestRecord, results []storage.AlgorithmResultRecord, topLimit int) algorithmExecutionSummaryDTO {
	severityCounts := map[string]int{}
	maxScore := 0.0
	maxConfidence := 0.0
	ordered := append([]storage.AlgorithmResultRecord(nil), results...)
	for _, result := range ordered {
		severityCounts[result.Severity]++
		if result.Score > maxScore {
			maxScore = result.Score
		}
		if result.Confidence > maxConfidence {
			maxConfidence = result.Confidence
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Score == ordered[j].Score {
			return ordered[i].CreatedAt.After(ordered[j].CreatedAt)
		}
		return ordered[i].Score > ordered[j].Score
	})
	if topLimit <= 0 || topLimit > len(ordered) {
		topLimit = len(ordered)
	}
	return algorithmExecutionSummaryDTO{ExecutionRequest: algorithmExecutionRequestResponse(request), ResultCount: len(results), SeverityCounts: severityCounts, MaxScore: maxScore, MaxConfidence: maxConfidence, TopResults: algorithmResultResponses(ordered[:topLimit])}
}
