package normalization

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakePublisher struct {
	message broker.Message
}

func (p *fakePublisher) Publish(_ context.Context, message broker.Message) (broker.PublishResult, error) {
	p.message = message
	return broker.PublishResult{Topic: message.Topic, Partition: 3, Offset: 19}, nil
}
func (*fakePublisher) Close(context.Context) error { return nil }

type fakeRepository struct {
	record storage.NormalizedEventLedgerRecord
}

func (r *fakeRepository) UpsertNormalizedEventLedger(_ context.Context, record storage.NormalizedEventLedgerRecord) error {
	r.record = record
	return nil
}

func TestBuildEventNormalizesGenericRawEnvelope(t *testing.T) {
	message := validMessage()
	event, err := BuildEvent(message, time.Date(2026, 7, 8, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if event.SchemaID != SchemaID || event.NormalizedPayload["temperature_c"] != 21.7 {
		t.Fatalf("event = %+v", event)
	}
	if len(event.Entities) != 1 || event.Entities[0]["id"] != "sensor:sensor-42" {
		t.Fatalf("entities = %+v", event.Entities)
	}
	if len(event.Evidence) != 1 || event.Evidence[0]["ref"] != "evt-42" {
		t.Fatalf("evidence = %+v", event.Evidence)
	}
}

func TestBuildEventPreservesAlreadyNormalizedReplay(t *testing.T) {
	message := validMessage()
	var payload map[string]any
	if err := json.Unmarshal(message.Value, &payload); err != nil {
		t.Fatal(err)
	}
	payload["normalized_payload"] = map[string]any{"canonical": true}
	payload["entities"] = []any{map[string]any{"type": "sensor", "id": "sensor:42"}}
	payload["evidence"] = []any{map[string]any{"type": "raw_event", "ref": "evt-original"}}
	delete(payload, "payload")
	message.Value, _ = json.Marshal(payload)
	event, err := BuildEvent(message, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if event.NormalizedPayload["canonical"] != true || event.Evidence[0]["ref"] != "evt-original" {
		t.Fatalf("event = %+v", event)
	}
}

func TestProcessorPublishesAndPersistsBrokerLineage(t *testing.T) {
	publisher := &fakePublisher{}
	repository := &fakeRepository{}
	processor := Processor{
		Publisher: publisher, Repository: repository, OutputTopic: "signalops.test.normalized.v1",
	}
	record, err := processor.Process(context.Background(), validMessage())
	if err != nil {
		t.Fatal(err)
	}
	if publisher.message.Topic != "signalops.test.normalized.v1" {
		t.Fatalf("published topic = %q", publisher.message.Topic)
	}
	if record.RawPartition != 2 || record.RawOffset != 7 ||
		record.NormalizedPartition != 3 || record.NormalizedOffset != 19 {
		t.Fatalf("record lineage = %+v", record)
	}
	if repository.record.EventID != "evt-42" {
		t.Fatalf("persisted record = %+v", repository.record)
	}
}

func validMessage() broker.ConsumedMessage {
	return broker.ConsumedMessage{
		Message: broker.Message{
			Topic: "signalops.test.raw.v1", Key: "idem-42",
			CorrelationID: "corr-42", TraceID: "trace-42",
			Value: []byte(`{
				"tenant_id":"tenant-test","source_id":"source-test","source_domain":"iot",
				"source_adapter":"test.sensor","ingestion_mode":"push_event","dataset":"sensor_observations",
				"event_id":"evt-42","event_type":"sensor.observed","idempotency_key":"idem-42",
				"observation_time":"2026-07-08T19:00:00Z","processing_time":"2026-07-08T19:00:01Z",
				"payload":{"temperature_c":21.7},
				"entity_hints":[{"type":"sensor","external_id":"sensor-42"}],
				"metadata":{},"correlation_id":"corr-42"
			}`),
		},
		Partition: 2, Offset: 7,
	}
}
