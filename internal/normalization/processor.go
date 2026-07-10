package normalization

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/appmeta"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

const (
	SchemaID      = "signalops.normalized_signal_event.v1"
	SchemaVersion = "1"
)

type InvalidEventError struct{ Err error }

func (e InvalidEventError) Error() string { return e.Err.Error() }
func (e InvalidEventError) Unwrap() error { return e.Err }

type Processor struct {
	Publisher   broker.Publisher
	Repository  storage.NormalizedEventLedgerRepository
	OutputTopic string
}

func (p Processor) Process(ctx context.Context, message broker.ConsumedMessage) (storage.NormalizedEventLedgerRecord, error) {
	if p.Publisher == nil || p.Repository == nil || strings.TrimSpace(p.OutputTopic) == "" {
		return storage.NormalizedEventLedgerRecord{}, errors.New("normalizer is not fully configured")
	}
	event, err := BuildEvent(message, time.Now().UTC())
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, InvalidEventError{Err: err}
	}
	value, err := json.Marshal(event)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, fmt.Errorf("marshal normalized event: %w", err)
	}
	result, err := p.Publisher.Publish(ctx, broker.Message{
		Topic: p.OutputTopic, Key: event.IdempotencyKey, Value: value,
		Headers: map[string]string{
			"content_type": "application/json", "signalops_schema_id": SchemaID,
			"signalops_event_id": event.EventID, "signalops_idempotency": event.IdempotencyKey,
			"signalops_source_topic":     message.Topic,
			"signalops_source_partition": fmt.Sprint(message.Partition),
			"signalops_source_offset":    fmt.Sprint(message.Offset),
		},
		CorrelationID: event.CorrelationID, CausationID: event.CausationID,
		TraceID: event.TraceID, PublishedAt: time.Now().UTC(),
	})
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, fmt.Errorf("publish normalized event: %w", err)
	}
	record, err := ledgerRecord(event, value, message, result)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	if err := p.Repository.UpsertNormalizedEventLedger(ctx, record); err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	return record, nil
}

type Event struct {
	TenantID          string           `json:"tenant_id"`
	SourceID          string           `json:"source_id"`
	AppID             string           `json:"app_id,omitempty"`
	Domain            string           `json:"domain,omitempty"`
	UseCase           string           `json:"use_case,omitempty"`
	SourceDomain      string           `json:"source_domain"`
	SourceAdapter     string           `json:"source_adapter"`
	IngestionMode     string           `json:"ingestion_mode"`
	Dataset           string           `json:"dataset"`
	EventID           string           `json:"event_id"`
	EventType         string           `json:"event_type"`
	SchemaID          string           `json:"schema_id"`
	SchemaVersion     string           `json:"schema_version"`
	ObservationTime   time.Time        `json:"observation_time"`
	EffectiveTime     time.Time        `json:"effective_time"`
	ProcessingTime    time.Time        `json:"processing_time"`
	OccurredAt        time.Time        `json:"occurred_at"`
	ObservedAt        time.Time        `json:"observed_at"`
	NormalizedPayload map[string]any   `json:"normalized_payload"`
	Entities          []map[string]any `json:"entities"`
	Confidence        float64          `json:"confidence"`
	Metadata          map[string]any   `json:"metadata"`
	Evidence          []map[string]any `json:"evidence"`
	CorrelationID     string           `json:"correlation_id"`
	IdempotencyKey    string           `json:"idempotency_key"`
	TraceID           string           `json:"trace_id,omitempty"`
	CausationID       string           `json:"causation_id,omitempty"`
	ReplayJobID       string           `json:"replay_job_id,omitempty"`
}

func BuildEvent(message broker.ConsumedMessage, now time.Time) (Event, error) {
	var raw map[string]any
	if err := json.Unmarshal(message.Value, &raw); err != nil {
		return Event{}, fmt.Errorf("decode raw event: %w", err)
	}
	required := func(name string) (string, error) {
		value, _ := raw[name].(string)
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("raw event %s is required", name)
		}
		return value, nil
	}
	tenantID, err := required("tenant_id")
	if err != nil {
		return Event{}, err
	}
	sourceID, err := required("source_id")
	if err != nil {
		return Event{}, err
	}
	sourceDomain, err := required("source_domain")
	if err != nil {
		return Event{}, err
	}
	if !allowedValue(sourceDomain, "market_data", "crm", "security", "operations", "iot", "procurement", "custom") {
		return Event{}, fmt.Errorf("raw event source_domain %q is unsupported", sourceDomain)
	}
	sourceAdapter, err := required("source_adapter")
	if err != nil {
		return Event{}, err
	}
	dataset, err := required("dataset")
	if err != nil {
		return Event{}, err
	}
	eventID, err := required("event_id")
	if err != nil {
		return Event{}, err
	}
	idempotencyKey := firstString(raw, "idempotency_key", eventID)
	correlationID := firstString(raw, "correlation_id", message.CorrelationID, eventID)
	observation, err := eventTime(raw, "observation_time", time.Time{})
	if err != nil {
		return Event{}, err
	}
	processing, err := eventTime(raw, "processing_time", now)
	if err != nil {
		return Event{}, err
	}
	effective, err := eventTime(raw, "effective_time", observation)
	if err != nil {
		return Event{}, err
	}
	occurred, err := eventTime(raw, "occurred_at", observation)
	if err != nil {
		return Event{}, err
	}
	observed, err := eventTime(raw, "observed_at", observation)
	if err != nil {
		return Event{}, err
	}
	payload := objectField(raw, "normalized_payload")
	if payload == nil {
		payload = objectField(raw, "payload")
	}
	if payload == nil {
		return Event{}, errors.New("raw event payload is required")
	}
	normalizedPayload, normalizationStrategy, err := normalizePayload(raw, payload)
	if err != nil {
		return Event{}, err
	}
	entities := entityRefs(raw)
	evidence := objectArrayField(raw, "evidence")
	if evidence == nil {
		evidence = []map[string]any{{"type": "raw_event", "ref": eventID, "metadata": map[string]any{
			"topic": message.Topic, "partition": message.Partition, "offset": message.Offset,
		}}}
	}
	metadata := objectField(raw, "metadata")
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["normalization"] = map[string]any{"strategy": normalizationStrategy, "normalized_at": now.Format(time.RFC3339Nano)}
	confidence := 1.0
	if value, ok := raw["confidence"].(float64); ok {
		confidence = value
	}
	if confidence < 0 || confidence > 1 {
		return Event{}, errors.New("confidence must be between 0 and 1")
	}
	meta := appmeta.Normalize(firstString(raw, "app_id"), firstString(raw, "domain"), firstString(raw, "use_case"), sourceDomain)
	ingestionMode := normalizeIngestionMode(firstString(raw, "ingestion_mode", "push_event"))
	if !allowedValue(ingestionMode, "push_event", "scheduled_pull", "bulk_file", "replay", "websocket_stream_future") {
		return Event{}, fmt.Errorf("raw event ingestion_mode %q is unsupported", ingestionMode)
	}
	return Event{
		TenantID: tenantID, SourceID: sourceID, AppID: meta.AppID, Domain: meta.Domain, UseCase: meta.UseCase, SourceDomain: sourceDomain, SourceAdapter: sourceAdapter,
		IngestionMode: ingestionMode, Dataset: dataset,
		EventID: eventID, EventType: firstString(raw, "event_type", dataset+".normalized"),
		SchemaID: SchemaID, SchemaVersion: SchemaVersion, ObservationTime: observation,
		EffectiveTime: effective, ProcessingTime: processing, OccurredAt: occurred, ObservedAt: observed,
		NormalizedPayload: normalizedPayload, Entities: entities, Confidence: confidence, Metadata: metadata,
		Evidence: evidence, CorrelationID: correlationID, IdempotencyKey: idempotencyKey,
		TraceID:     firstString(raw, "trace_id", message.TraceID),
		CausationID: firstString(raw, "causation_id", message.CausationID, eventID),
		ReplayJobID: firstString(raw, "replay_job_id"),
	}, nil
}

func ledgerRecord(event Event, value []byte, source broker.ConsumedMessage, result broker.PublishResult) (storage.NormalizedEventLedgerRecord, error) {
	payload, err := json.Marshal(event.NormalizedPayload)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	entities, err := json.Marshal(event.Entities)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	evidence, err := json.Marshal(event.Evidence)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	return storage.NormalizedEventLedgerRecord{
		EventID: event.EventID, TenantID: event.TenantID, SourceID: event.SourceID,
		AppID: event.AppID, Domain: event.Domain, UseCase: event.UseCase, SourceAdapter: event.SourceAdapter, Dataset: event.Dataset, IdempotencyKey: event.IdempotencyKey,
		SchemaID: event.SchemaID, SchemaVersion: event.SchemaVersion, ObservationTime: event.ObservationTime,
		ProcessingTime: event.ProcessingTime, Confidence: event.Confidence, RawTopic: source.Topic,
		RawPartition: source.Partition, RawOffset: source.Offset, NormalizedTopic: result.Topic,
		NormalizedPartition: result.Partition, NormalizedOffset: result.Offset, NormalizedPayload: payload,
		EntitiesJSON: entities, EvidenceJSON: evidence, MetadataJSON: metadata, EventJSON: value,
	}, nil
}

func PublishInvalidEvent(ctx context.Context, publisher broker.Publisher, topic string, message broker.ConsumedMessage, cause error) error {
	value, err := json.Marshal(map[string]any{
		"schema_id": "signalops.dlq.normalization.v1", "failed_at": time.Now().UTC().Format(time.RFC3339Nano),
		"error_type": "InvalidNormalizationEvent", "error_message": cause.Error(),
		"source": map[string]any{"topic": message.Topic, "partition": message.Partition, "offset": message.Offset,
			"key": message.Key, "headers": message.Headers, "value_base64": base64.StdEncoding.EncodeToString(message.Value)},
	})
	if err != nil {
		return err
	}
	_, err = publisher.Publish(ctx, broker.Message{Topic: topic, Key: message.Key, Value: value, CorrelationID: message.CorrelationID,
		Headers: map[string]string{"content_type": "application/json", "signalops_dlq_reason": "InvalidNormalizationEvent",
			"signalops_source_topic": message.Topic, "signalops_source_partition": fmt.Sprint(message.Partition), "signalops_source_offset": fmt.Sprint(message.Offset)}})
	if err != nil {
		return fmt.Errorf("publish normalization DLQ event: %w", err)
	}
	return nil
}

func normalizePayload(raw map[string]any, payload map[string]any) (map[string]any, string, error) {
	if firstString(raw, "app_id") == "marketops" && firstString(raw, "source_adapter") == "market_data.massive" && firstString(raw, "dataset") == "options_contracts_daily" {
		normalized, err := normalizeMassiveOptionContractDaily(payload)
		if err != nil {
			return nil, "", err
		}
		return normalized, "marketops_massive_option_contract_daily_v1", nil
	}
	return payload, "identity_v1", nil
}

func normalizeMassiveOptionContractDaily(payload map[string]any) (map[string]any, error) {
	optionTicker := requiredPayloadString(payload, "option_ticker")
	underlying := strings.ToUpper(requiredPayloadString(payload, "underlying_symbol"))
	contractType := strings.ToLower(requiredPayloadString(payload, "contract_type"))
	expirationDate := requiredPayloadString(payload, "expiration_date")
	observationDate := requiredPayloadString(payload, "observation_date")
	strikePrice, strikeOK := positivePayloadFloat(payload, "strike_price")

	if optionTicker == "" {
		return nil, errors.New("Massive option contract daily payload option_ticker is required")
	}
	if underlying == "" {
		return nil, errors.New("Massive option contract daily payload underlying_symbol is required")
	}
	if !allowedValue(contractType, "call", "put") {
		return nil, fmt.Errorf("Massive option contract daily payload contract_type %q is unsupported", contractType)
	}
	if _, err := parsePayloadDate(expirationDate, "expiration_date"); err != nil {
		return nil, err
	}
	if _, err := parsePayloadDate(observationDate, "observation_date"); err != nil {
		return nil, err
	}
	if !strikeOK {
		return nil, errors.New("Massive option contract daily payload strike_price must be greater than zero")
	}

	normalized := map[string]any{
		"provider":             payloadString(payload, "provider", "massive"),
		"dataset":              "options_contracts_daily",
		"provider_contract_id": payloadString(payload, "provider_contract_id"),
		"option_ticker":        optionTicker,
		"underlying_symbol":    underlying,
		"contract_type":        contractType,
		"expiration_date":      expirationDate,
		"strike_price":         strikePrice,
		"observation_date":     observationDate,
		"asset_type":           "option_contract",
	}
	for _, field := range []string{"open", "high", "low", "close", "vwap"} {
		value, present, err := optionalNonNegativePayloadFloat(payload, field)
		if err != nil {
			return nil, err
		}
		if present {
			normalized[field] = value
		}
	}
	for _, field := range []string{"volume", "open_interest"} {
		value, present, err := optionalNonNegativePayloadInt(payload, field)
		if err != nil {
			return nil, err
		}
		if present {
			normalized[field] = value
		}
	}
	if raw, ok := payload["raw"].(map[string]any); ok {
		normalized["raw"] = raw
	}
	return normalized, nil
}

func requiredPayloadString(payload map[string]any, name string) string {
	return strings.TrimSpace(payloadString(payload, name))
}

func payloadString(payload map[string]any, name string, fallbacks ...string) string {
	if value, ok := payload[name].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	for _, fallback := range fallbacks {
		if strings.TrimSpace(fallback) != "" {
			return strings.TrimSpace(fallback)
		}
	}
	return ""
}

func parsePayloadDate(value string, name string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("Massive option contract daily payload %s must be YYYY-MM-DD: %w", name, err)
	}
	return parsed, nil
}

func positivePayloadFloat(payload map[string]any, name string) (float64, bool) {
	value, ok := payloadFloat(payload, name)
	return value, ok && value > 0
}

func optionalNonNegativePayloadFloat(payload map[string]any, name string) (float64, bool, error) {
	if _, present := payload[name]; !present {
		return 0, false, nil
	}
	value, ok := payloadFloat(payload, name)
	if !ok || value < 0 {
		return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative number", name)
	}
	return value, true, nil
}

func payloadFloat(payload map[string]any, name string) (float64, bool) {
	switch value := payload[name].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func optionalNonNegativePayloadInt(payload map[string]any, name string) (int64, bool, error) {
	if _, present := payload[name]; !present {
		return 0, false, nil
	}
	switch value := payload[name].(type) {
	case float64:
		if value < 0 || value != float64(int64(value)) {
			return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative integer", name)
		}
		return int64(value), true, nil
	case int:
		if value < 0 {
			return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative integer", name)
		}
		return int64(value), true, nil
	case int64:
		if value < 0 {
			return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative integer", name)
		}
		return value, true, nil
	case json.Number:
		parsed, err := value.Int64()
		if err != nil || parsed < 0 {
			return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative integer", name)
		}
		return parsed, true, nil
	default:
		return 0, false, fmt.Errorf("Massive option contract daily payload %s must be a non-negative integer", name)
	}
}

func normalizeIngestionMode(value string) string {
	switch strings.TrimSpace(value) {
	case "", "push", "event":
		return "push_event"
	case "stream", "websocket":
		return "websocket_stream_future"
	default:
		return strings.TrimSpace(value)
	}
}

func allowedValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func firstString(raw map[string]any, name string, fallbacks ...string) string {
	if value, ok := raw[name].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	for _, value := range fallbacks {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func eventTime(raw map[string]any, name string, fallback time.Time) (time.Time, error) {
	value, _ := raw[name].(string)
	if strings.TrimSpace(value) == "" {
		if fallback.IsZero() {
			return time.Time{}, fmt.Errorf("raw event %s is required", name)
		}
		return fallback.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("raw event %s must be RFC3339: %w", name, err)
	}
	return parsed.UTC(), nil
}

func objectField(raw map[string]any, name string) map[string]any {
	value, _ := raw[name].(map[string]any)
	return value
}

func objectArrayField(raw map[string]any, name string) []map[string]any {
	values, _ := raw[name].([]any)
	if values == nil {
		return nil
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			result = append(result, item)
		}
	}
	return result
}

func entityRefs(raw map[string]any) []map[string]any {
	if existing := objectArrayField(raw, "entities"); existing != nil {
		return existing
	}
	hints := objectArrayField(raw, "entity_hints")
	result := make([]map[string]any, 0, len(hints))
	for _, hint := range hints {
		kind := firstString(hint, "type", firstString(hint, "entity_type"))
		externalID := firstString(hint, "external_id", firstString(hint, "entity_id"))
		if kind == "" || externalID == "" {
			continue
		}
		ref := map[string]any{"type": kind, "id": kind + ":" + externalID, "external_id": externalID}
		if confidence, ok := hint["confidence"].(float64); ok {
			ref["confidence"] = confidence
		}
		if metadata, ok := hint["metadata"].(map[string]any); ok {
			ref["metadata"] = metadata
		}
		result = append(result, ref)
	}
	return result
}
