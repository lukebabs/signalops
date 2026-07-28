package connect

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	AppID         = "cyberops"
	EventType     = "cyberops.syslog.raw"
	Dataset       = "cyberops.syslog.raw"
	SchemaID      = "signalops.raw_signal_event.v1"
	SchemaVersion = "1.0.0"
)

type InvalidEventError struct{ Err error }

func (e InvalidEventError) Error() string { return e.Err.Error() }
func (e InvalidEventError) Unwrap() error { return e.Err }

type Event struct {
	TenantID       string
	IngressEventID string
	SourceID       string
	PayloadHash    string
	Raw            []byte
	Metadata       map[string]any
	Lineage        []byte
	OccurredAt     time.Time
	IngestedAt     time.Time
	Hostname       string
	Application    string
	Severity       *int
	Facility       *int
	Message        string
	CorrelationID  string
	CausationID    string
	TraceID        string
}

func IsCandidate(value []byte, headers map[string]string) bool {
	if strings.TrimSpace(headers["connect_ingress_event_id"]) != "" || strings.TrimSpace(headers["connect_delivery_id"]) != "" {
		return true
	}
	var envelope map[string]any
	if json.Unmarshal(value, &envelope) != nil {
		return false
	}
	if stringValue(envelope["app_id"]) == AppID || strings.HasPrefix(stringValue(envelope["source_adapter"]), "connect:") {
		return true
	}
	metadata, _ := envelope["metadata"].(map[string]any)
	_, ok := metadata["connect"]
	return ok
}

func Validate(value []byte) (Event, error) {
	var raw map[string]any
	if err := json.Unmarshal(value, &raw); err != nil {
		return Event{}, InvalidEventError{fmt.Errorf("decode raw event: %w", err)}
	}
	required := func(key string) (string, error) {
		value := strings.TrimSpace(stringValue(raw[key]))
		if value == "" {
			return "", InvalidEventError{fmt.Errorf("%s is required", key)}
		}
		return value, nil
	}
	for _, key := range []string{"tenant_id", "source_id", "source_domain", "source_adapter", "ingestion_mode", "dataset", "event_id", "event_type", "schema_id", "schema_version", "correlation_id", "idempotency_key"} {
		if _, err := required(key); err != nil {
			return Event{}, err
		}
	}
	if stringValue(raw["app_id"]) != AppID || stringValue(raw["domain"]) != "security" || stringValue(raw["use_case"]) != AppID ||
		stringValue(raw["source_domain"]) != "security" || !strings.HasPrefix(stringValue(raw["source_adapter"]), "connect:") ||
		stringValue(raw["ingestion_mode"]) != "push_event" || stringValue(raw["dataset"]) != Dataset ||
		stringValue(raw["event_type"]) != EventType || stringValue(raw["schema_id"]) != SchemaID || stringValue(raw["schema_version"]) != SchemaVersion {
		return Event{}, InvalidEventError{errors.New("event does not match the CyberOps Connect envelope")}
	}
	metadata, _ := raw["metadata"].(map[string]any)
	connect, ok := metadata["connect"].(map[string]any)
	if !ok {
		return Event{}, InvalidEventError{errors.New("metadata.connect is required")}
	}
	requiredConnect := []string{"contract_version", "tenant_id", "ingress_event_id", "connector_id", "connector_version", "channel_id", "producer_id", "protocol_key", "protocol_version", "mapping_key", "mapping_version", "dataset_binding_id", "dataset_key", "dataset_version", "destination", "payload_hash_algorithm", "payload_hash", "processing_run_id", "delivery_id"}
	for _, key := range requiredConnect {
		if strings.TrimSpace(stringValue(connect[key])) == "" {
			return Event{}, InvalidEventError{fmt.Errorf("metadata.connect.%s is required", key)}
		}
	}
	ingress := stringValue(connect["ingress_event_id"])
	if stringValue(raw["source_adapter"]) != "connect:"+stringValue(connect["connector_id"]) {
		return Event{}, InvalidEventError{errors.New("source_adapter must match metadata.connect.connector_id")}
	}
	if stringValue(connect["contract_version"]) != "1.0.0" || stringValue(connect["tenant_id"]) != stringValue(raw["tenant_id"]) ||
		stringValue(connect["producer_id"]) != stringValue(raw["source_id"]) || stringValue(connect["protocol_key"]) != "syslog-rfc5424" ||
		stringValue(connect["protocol_version"]) != "1.0.0" || stringValue(connect["payload_hash_algorithm"]) != "sha256" ||
		stringValue(raw["event_id"]) != ingress || stringValue(raw["correlation_id"]) != ingress ||
		stringValue(raw["causation_id"]) != ingress || stringValue(raw["idempotency_key"]) != ingress {
		return Event{}, InvalidEventError{errors.New("metadata.connect identity or protocol mismatch")}
	}
	hash := stringValue(connect["payload_hash"])
	if len(hash) != 64 {
		return Event{}, InvalidEventError{errors.New("metadata.connect.payload_hash must be sha256 hex")}
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return Event{}, InvalidEventError{errors.New("metadata.connect.payload_hash must be sha256 hex")}
	}
	payload, ok := raw["payload"].(map[string]any)
	if !ok {
		return Event{}, InvalidEventError{errors.New("payload is required")}
	}
	message := stringValue(payload["message"])
	if message == "" {
		return Event{}, InvalidEventError{errors.New("payload.message is required")}
	}
	occurred, err := parseTime(raw, "occurred_at")
	if err != nil {
		return Event{}, InvalidEventError{err}
	}
	effective, err := parseTime(raw, "effective_time")
	if err != nil || !effective.Equal(occurred) {
		return Event{}, InvalidEventError{errors.New("effective_time must equal occurred_at")}
	}
	payloadOccurred, err := parseTime(payload, "occurred_at")
	if err != nil || !payloadOccurred.Equal(occurred) {
		return Event{}, InvalidEventError{errors.New("payload.occurred_at must equal occurred_at")}
	}
	observed, err := parseTime(raw, "observed_at")
	if err != nil {
		return Event{}, InvalidEventError{err}
	}
	if observation, err := parseTime(raw, "observation_time"); err != nil || !observation.Equal(observed) {
		return Event{}, InvalidEventError{errors.New("observation_time must equal observed_at")}
	}
	processing, err := parseTime(raw, "processing_time")
	if err != nil {
		return Event{}, InvalidEventError{err}
	}
	source, _ := payload["source"].(map[string]any)
	syslog, _ := payload["syslog"].(map[string]any)
	lineage := cloneLineage(connect)
	lineageJSON, _ := json.Marshal(lineage)
	return Event{TenantID: stringValue(raw["tenant_id"]), IngressEventID: ingress, SourceID: stringValue(raw["source_id"]), PayloadHash: hash, Raw: value, Metadata: connect, Lineage: lineageJSON, OccurredAt: occurred, IngestedAt: processing, Hostname: stringValue(source["hostname"]), Application: stringValue(source["application"]), Severity: intValue(syslog["severity"]), Facility: intValue(syslog["facility"]), Message: message, CorrelationID: ingress, CausationID: ingress, TraceID: first(stringValue(raw["trace_id"]), "trc_"+ingress)}, nil
}

func cloneLineage(connect map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range connect {
		if key != "processing_run_id" && key != "delivery_id" {
			out[key] = value
		}
	}
	return out
}

func parseTime(values map[string]any, key string) (time.Time, error) {
	value := stringValue(values[key])
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339Nano", key)
	}
	return parsed.UTC(), nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func intValue(value any) *int {
	number, ok := value.(float64)
	if !ok || number != float64(int(number)) {
		return nil
	}
	result := int(number)
	return &result
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
