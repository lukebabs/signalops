package contracts

import "time"

type SourceDomain string
type IngestionMode string
type Severity string

const (
	SourceDomainMarketData  SourceDomain = "market_data"
	SourceDomainCRM         SourceDomain = "crm"
	SourceDomainSecurity    SourceDomain = "security"
	SourceDomainOperations  SourceDomain = "operations"
	SourceDomainIoT         SourceDomain = "iot"
	SourceDomainProcurement SourceDomain = "procurement"
	SourceDomainCustom      SourceDomain = "custom"
)

const (
	IngestionModePushEvent       IngestionMode = "push_event"
	IngestionModeScheduledPull   IngestionMode = "scheduled_pull"
	IngestionModeBulkFile        IngestionMode = "bulk_file"
	IngestionModeReplay          IngestionMode = "replay"
	IngestionModeWebsocketFuture IngestionMode = "websocket_stream_future"
)

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type EntityHint struct {
	Type       string         `json:"type"`
	ExternalID string         `json:"external_id"`
	Confidence *float64       `json:"confidence,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type EntityRef struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	ExternalID string         `json:"external_id,omitempty"`
	Confidence *float64       `json:"confidence,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type EvidenceRef struct {
	Type     string         `json:"type"`
	Ref      string         `json:"ref"`
	Summary  string         `json:"summary,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type EventEnvelope struct {
	TenantID       string         `json:"tenant_id"`
	SourceID       string         `json:"source_id"`
	SourceDomain   SourceDomain   `json:"source_domain"`
	SourceAdapter  string         `json:"source_adapter"`
	IngestionMode  IngestionMode  `json:"ingestion_mode"`
	Dataset        string         `json:"dataset"`
	EventID        string         `json:"event_id"`
	EventType      string         `json:"event_type"`
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	ObservationAt  time.Time      `json:"observation_time"`
	EffectiveAt    time.Time      `json:"effective_time"`
	ProcessingAt   time.Time      `json:"processing_time"`
	OccurredAt     time.Time      `json:"occurred_at"`
	ObservedAt     time.Time      `json:"observed_at"`
	Metadata       map[string]any `json:"metadata"`
	CorrelationID  string         `json:"correlation_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	TraceID        string         `json:"trace_id,omitempty"`
	CausationID    string         `json:"causation_id,omitempty"`
	ReplayJobID    string         `json:"replay_job_id,omitempty"`
}

type RawSignalEvent struct {
	EventEnvelope
	Payload     map[string]any `json:"payload"`
	EntityHints []EntityHint   `json:"entity_hints"`
}

type NormalizedSignalEvent struct {
	EventEnvelope
	NormalizedPayload map[string]any `json:"normalized_payload"`
	Entities          []EntityRef    `json:"entities"`
	Confidence        float64        `json:"confidence"`
	Evidence          []EvidenceRef  `json:"evidence"`
}

type Signal struct {
	SignalID          string           `json:"signal_id"`
	TenantID          string           `json:"tenant_id"`
	SourceID          string           `json:"source_id"`
	SourceDomain      SourceDomain     `json:"source_domain"`
	SourceAdapter     string           `json:"source_adapter"`
	IngestionMode     IngestionMode    `json:"ingestion_mode"`
	Dataset           string           `json:"dataset"`
	EventIDs          []string         `json:"event_ids"`
	ArtifactIDs       []string         `json:"artifact_ids"`
	SignalType        string           `json:"signal_type"`
	DetectorID        string           `json:"detector_id"`
	DetectorVersion   string           `json:"detector_version"`
	ModelVersion      string           `json:"model_version"`
	Timestamp         time.Time        `json:"timestamp"`
	ObservationAt     time.Time        `json:"observation_time"`
	EffectiveAt       time.Time        `json:"effective_time"`
	ProcessingAt      time.Time        `json:"processing_time"`
	WindowStart       time.Time        `json:"window_start"`
	WindowEnd         time.Time        `json:"window_end"`
	Confidence        float64          `json:"confidence"`
	Severity          Severity         `json:"severity"`
	Entities          []EntityRef      `json:"entities"`
	SupportingMetrics map[string]any   `json:"supporting_metrics"`
	GraphTargets      []map[string]any `json:"graph_targets"`
	SemanticEvidence  []map[string]any `json:"semantic_evidence"`
	Evidence          []EvidenceRef    `json:"evidence"`
	Recommendation    map[string]any   `json:"recommendation"`
	CorrelationID     string           `json:"correlation_id"`
	TraceID           string           `json:"trace_id,omitempty"`
	CausationID       string           `json:"causation_id,omitempty"`
	ReplayJobID       string           `json:"replay_job_id,omitempty"`
}
