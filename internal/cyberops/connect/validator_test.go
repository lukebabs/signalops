package connect

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

func TestValidateAcceptsCyberOpsConnectEvent(t *testing.T) {
	value := validEvent(t)
	event, err := Validate(value)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if event.IngressEventID != "ing-1" || event.Message != "blocked packet" || event.Severity == nil || *event.Severity != 4 {
		t.Fatalf("unexpected event: %+v", event)
	}
}
func TestValidateRejectsIdempotencyMismatch(t *testing.T) {
	var value map[string]any
	if err := json.Unmarshal(validEvent(t), &value); err != nil {
		t.Fatal(err)
	}
	value["idempotency_key"] = "wrong"
	raw, _ := json.Marshal(value)
	if _, err := Validate(raw); err == nil {
		t.Fatal("expected semantic validation failure")
	}
}
func TestProcessorIgnoresNonCandidate(t *testing.T) {
	result, err := Processor{Repository: &fakeRepository{}, AcceptedTopic: "signalops.local.connect-accepted-raw.v1"}.Process(context.Background(), broker.ConsumedMessage{Message: broker.Message{Value: []byte(`{"app_id":"marketops"}`)}})
	if err != nil || !result.Ignored {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

type fakeRepository struct {
	raw    storage.CyberOpsConnectRawRecord
	outbox storage.CyberOpsOutboxRecord
}

func (f *fakeRepository) PersistCyberOpsConnectRaw(_ context.Context, raw storage.CyberOpsConnectRawRecord, outbox storage.CyberOpsOutboxRecord, _ []byte) (storage.CyberOpsPersistResult, error) {
	f.raw = raw
	f.outbox = outbox
	return storage.CyberOpsPersistResult{}, nil
}
func validEvent(t *testing.T) []byte {
	t.Helper()
	occurred := time.Date(2026, 7, 28, 1, 2, 3, 400, time.UTC).Format(time.RFC3339Nano)
	received := time.Date(2026, 7, 28, 1, 2, 4, 0, time.UTC).Format(time.RFC3339Nano)
	processed := time.Date(2026, 7, 28, 1, 2, 5, 0, time.UTC).Format(time.RFC3339Nano)
	connect := map[string]any{"contract_version": "1.0.0", "tenant_id": "tenant-local", "ingress_event_id": "ing-1", "connector_id": "opnsense", "connector_version": "1.0.0", "channel_id": "channel-1", "producer_id": "firewall-1", "protocol_key": "syslog-rfc5424", "protocol_version": "1.0.0", "mapping_key": "rfc5424", "mapping_version": "1.0.0", "dataset_binding_id": "binding-1", "dataset_key": "cyberops.syslog.raw", "dataset_version": "1.0.0", "destination": "signalops", "payload_hash_algorithm": "sha256", "payload_hash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "processing_run_id": "run-1", "delivery_id": "delivery-1"}
	value := map[string]any{"tenant_id": "tenant-local", "app_id": "cyberops", "domain": "security", "use_case": "cyberops", "source_id": "firewall-1", "source_domain": "security", "source_adapter": "connect:opnsense", "ingestion_mode": "push_event", "dataset": "cyberops.syslog.raw", "event_id": "ing-1", "event_type": "cyberops.syslog.raw", "schema_id": "signalops.raw_signal_event.v1", "schema_version": "1.0.0", "observation_time": received, "effective_time": occurred, "processing_time": processed, "occurred_at": occurred, "observed_at": received, "payload": map[string]any{"source": map[string]any{"hostname": "fw-1", "application": "filterlog"}, "message": "blocked packet", "occurred_at": occurred, "syslog": map[string]any{"severity": 4, "facility": 20}}, "entity_hints": []any{}, "metadata": map[string]any{"connect": connect}, "correlation_id": "ing-1", "causation_id": "ing-1", "idempotency_key": "ing-1"}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
