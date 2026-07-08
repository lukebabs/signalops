package signals

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type InvalidEventError struct{ Err error }

func (e InvalidEventError) Error() string { return e.Err.Error() }
func (e InvalidEventError) Unwrap() error { return e.Err }

type Processor struct {
	Repository storage.SignalLedgerRepository
}

func (p Processor) Process(ctx context.Context, message broker.ConsumedMessage) (storage.SignalLedgerRecord, error) {
	if p.Repository == nil {
		return storage.SignalLedgerRecord{}, errors.New("signal processor repository is required")
	}
	event, err := DecodeEvent(message.Value)
	if err != nil {
		return storage.SignalLedgerRecord{}, InvalidEventError{Err: err}
	}
	record, err := ledgerRecord(event, message)
	if err != nil {
		return storage.SignalLedgerRecord{}, InvalidEventError{Err: err}
	}
	if err := p.Repository.UpsertSignalLedger(ctx, record); err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	return record, nil
}

type Event struct {
	SignalID          string           `json:"signal_id"`
	TenantID          string           `json:"tenant_id"`
	SourceID          string           `json:"source_id"`
	SourceDomain      string           `json:"source_domain"`
	SourceAdapter     string           `json:"source_adapter"`
	IngestionMode     string           `json:"ingestion_mode"`
	Dataset           string           `json:"dataset"`
	EventIDs          []string         `json:"event_ids"`
	ArtifactIDs       []string         `json:"artifact_ids"`
	SignalType        string           `json:"signal_type"`
	DetectorID        string           `json:"detector_id"`
	DetectorVersion   string           `json:"detector_version"`
	ModelVersion      string           `json:"model_version"`
	Timestamp         time.Time        `json:"timestamp"`
	ObservationTime   time.Time        `json:"observation_time"`
	EffectiveTime     time.Time        `json:"effective_time"`
	ProcessingTime    time.Time        `json:"processing_time"`
	WindowStart       time.Time        `json:"window_start"`
	WindowEnd         time.Time        `json:"window_end"`
	Confidence        float64          `json:"confidence"`
	Severity          string           `json:"severity"`
	Entities          []map[string]any `json:"entities"`
	SupportingMetrics map[string]any   `json:"supporting_metrics"`
	GraphTargets      []map[string]any `json:"graph_targets"`
	SemanticEvidence  []map[string]any `json:"semantic_evidence"`
	Evidence          []map[string]any `json:"evidence"`
	Recommendation    json.RawMessage  `json:"recommendation"`
	CorrelationID     string           `json:"correlation_id"`
	TraceID           string           `json:"trace_id,omitempty"`
	CausationID       string           `json:"causation_id,omitempty"`
	ReplayJobID       string           `json:"replay_job_id,omitempty"`
}

func DecodeEvent(value []byte) (Event, error) {
	var event Event
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return Event{}, fmt.Errorf("decode signal event: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Event{}, err
	}
	for name, field := range map[string]string{
		"signal_id": event.SignalID, "tenant_id": event.TenantID, "source_id": event.SourceID,
		"source_domain": event.SourceDomain, "source_adapter": event.SourceAdapter,
		"ingestion_mode": event.IngestionMode, "dataset": event.Dataset, "signal_type": event.SignalType,
		"detector_id": event.DetectorID, "detector_version": event.DetectorVersion,
		"model_version": event.ModelVersion, "correlation_id": event.CorrelationID,
	} {
		if strings.TrimSpace(field) == "" {
			return Event{}, fmt.Errorf("signal event %s is required", name)
		}
	}
	if len(event.EventIDs) == 0 || hasEmpty(event.EventIDs) {
		return Event{}, errors.New("signal event event_ids must contain non-empty values")
	}
	if hasEmpty(event.ArtifactIDs) {
		return Event{}, errors.New("signal event artifact_ids must contain non-empty values")
	}
	if !allowed(event.SourceDomain, "market_data", "crm", "security", "operations", "iot", "procurement", "custom") {
		return Event{}, fmt.Errorf("signal event source_domain %q is unsupported", event.SourceDomain)
	}
	if !allowed(event.IngestionMode, "push_event", "scheduled_pull", "bulk_file", "replay", "websocket_stream_future") {
		return Event{}, fmt.Errorf("signal event ingestion_mode %q is unsupported", event.IngestionMode)
	}
	if !allowed(event.Severity, "info", "low", "medium", "high", "critical") {
		return Event{}, fmt.Errorf("signal event severity %q is unsupported", event.Severity)
	}
	if event.Confidence < 0 || event.Confidence > 1 {
		return Event{}, errors.New("signal event confidence must be between 0 and 1")
	}
	for name, value := range map[string]time.Time{
		"timestamp": event.Timestamp, "observation_time": event.ObservationTime,
		"effective_time": event.EffectiveTime, "processing_time": event.ProcessingTime,
		"window_start": event.WindowStart, "window_end": event.WindowEnd,
	} {
		if value.IsZero() {
			return Event{}, fmt.Errorf("signal event %s is required", name)
		}
	}
	if len(event.Recommendation) == 0 {
		return Event{}, errors.New("signal event recommendation is required")
	}
	var recommendation any
	if err := json.Unmarshal(event.Recommendation, &recommendation); err != nil {
		return Event{}, errors.New("signal event recommendation must be an object or null")
	}
	if recommendation != nil {
		if _, ok := recommendation.(map[string]any); !ok {
			return Event{}, errors.New("signal event recommendation must be an object or null")
		}
	}
	return event, nil
}

func ledgerRecord(event Event, message broker.ConsumedMessage) (storage.SignalLedgerRecord, error) {
	entities, err := json.Marshal(event.Entities)
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	metrics, err := json.Marshal(event.SupportingMetrics)
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	graphTargets, err := json.Marshal(event.GraphTargets)
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	semanticEvidence, err := json.Marshal(event.SemanticEvidence)
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	evidence, err := json.Marshal(event.Evidence)
	if err != nil {
		return storage.SignalLedgerRecord{}, err
	}
	return storage.SignalLedgerRecord{
		SignalID: event.SignalID, TenantID: event.TenantID, SourceID: event.SourceID,
		SourceDomain: event.SourceDomain, SourceAdapter: event.SourceAdapter,
		IngestionMode: event.IngestionMode, Dataset: event.Dataset, EventIDs: event.EventIDs,
		ArtifactIDs: event.ArtifactIDs, SignalType: event.SignalType, DetectorID: event.DetectorID,
		DetectorVersion: event.DetectorVersion, ModelVersion: event.ModelVersion,
		SignalTime: event.Timestamp, ObservationTime: event.ObservationTime,
		EffectiveTime: event.EffectiveTime, ProcessingTime: event.ProcessingTime,
		WindowStart: event.WindowStart, WindowEnd: event.WindowEnd, Confidence: event.Confidence,
		Severity: event.Severity, EntitiesJSON: entities, SupportingMetrics: metrics,
		GraphTargetsJSON: graphTargets, SemanticEvidenceJSON: semanticEvidence, EvidenceJSON: evidence,
		RecommendationJSON: append([]byte(nil), event.Recommendation...), CorrelationID: event.CorrelationID,
		TraceID: event.TraceID, CausationID: event.CausationID, ReplayJobID: event.ReplayJobID,
		BrokerTopic: message.Topic, BrokerPartition: message.Partition, BrokerOffset: message.Offset,
		EventJSON: append([]byte(nil), message.Value...),
	}, nil
}

func PublishInvalidEvent(ctx context.Context, publisher broker.Publisher, topic string, message broker.ConsumedMessage, cause error) error {
	value, err := json.Marshal(map[string]any{
		"schema_id":  "signalops.dlq.signal_persistence.v1",
		"failed_at":  time.Now().UTC().Format(time.RFC3339Nano),
		"error_type": "InvalidSignalEvent", "error_message": cause.Error(),
		"source": map[string]any{
			"topic": message.Topic, "partition": message.Partition, "offset": message.Offset,
			"key": message.Key, "headers": message.Headers,
			"value_base64": base64.StdEncoding.EncodeToString(message.Value),
		},
	})
	if err != nil {
		return err
	}
	_, err = publisher.Publish(ctx, broker.Message{
		Topic: topic, Key: message.Key, Value: value, CorrelationID: message.CorrelationID,
		Headers: map[string]string{
			"content_type": "application/json", "signalops_dlq_reason": "InvalidSignalEvent",
			"signalops_source_topic":     message.Topic,
			"signalops_source_partition": fmt.Sprint(message.Partition),
			"signalops_source_offset":    fmt.Sprint(message.Offset),
		},
	})
	if err != nil {
		return fmt.Errorf("publish signal persistence DLQ event: %w", err)
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("signal event contains multiple JSON values")
		}
		return fmt.Errorf("decode trailing signal event data: %w", err)
	}
	return nil
}

func hasEmpty(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func allowed(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
