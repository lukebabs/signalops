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

func TestBuildEventNormalizesMassiveOptionContractDaily(t *testing.T) {
	message := massiveOptionMessage()
	event, err := BuildEvent(message, time.Date(2026, 7, 10, 18, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	payload := event.NormalizedPayload
	if payload["provider"] != "massive" || payload["dataset"] != "options_contracts_daily" {
		t.Fatalf("provider/dataset = %v/%v", payload["provider"], payload["dataset"])
	}
	if payload["option_ticker"] != "O:SPY260116C00600000" || payload["underlying_symbol"] != "SPY" {
		t.Fatalf("option refs = %v/%v", payload["option_ticker"], payload["underlying_symbol"])
	}
	if payload["contract_type"] != "call" || payload["asset_type"] != "option_contract" {
		t.Fatalf("contract fields = %v/%v", payload["contract_type"], payload["asset_type"])
	}
	if payload["strike_price"] != 600.0 || payload["volume"] != int64(1200) || payload["open_interest"] != int64(4000) {
		t.Fatalf("market fields = %+v", payload)
	}
	if len(event.Entities) != 2 || event.Entities[0]["id"] != "option_contract:O:SPY260116C00600000" || event.Entities[1]["id"] != "ticker:SPY" {
		t.Fatalf("entities = %+v", event.Entities)
	}
	if event.AppID != "marketops" || event.Domain != "market_data" || event.UseCase != "daily_market_surveillance" {
		t.Fatalf("app metadata = %q/%q/%q", event.AppID, event.Domain, event.UseCase)
	}
	normalization, _ := event.Metadata["normalization"].(map[string]any)
	if normalization["strategy"] != "marketops_massive_option_contract_daily_v1" {
		t.Fatalf("normalization metadata = %+v", normalization)
	}
}

func TestBuildEventRejectsInvalidMassiveOptionContractDaily(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing option ticker", mutate: func(payload map[string]any) { delete(payload, "option_ticker") }},
		{name: "bad contract type", mutate: func(payload map[string]any) { payload["contract_type"] = "straddle" }},
		{name: "bad expiration date", mutate: func(payload map[string]any) { payload["expiration_date"] = "2026/01/16" }},
		{name: "zero strike", mutate: func(payload map[string]any) { payload["strike_price"] = 0.0 }},
		{name: "negative close", mutate: func(payload map[string]any) { payload["close"] = -1.0 }},
		{name: "fractional volume", mutate: func(payload map[string]any) { payload["volume"] = 12.5 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			message := massiveOptionMessage()
			var raw map[string]any
			if err := json.Unmarshal(message.Value, &raw); err != nil {
				t.Fatal(err)
			}
			payload := raw["payload"].(map[string]any)
			tc.mutate(payload)
			message.Value, _ = json.Marshal(raw)
			if _, err := BuildEvent(message, time.Now().UTC()); err == nil {
				t.Fatal("expected invalid option normalization error")
			}
		})
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

func massiveOptionMessage() broker.ConsumedMessage {
	return broker.ConsumedMessage{
		Message: broker.Message{
			Topic: "signalops.test.raw.v1", Key: "idem-option-1",
			CorrelationID: "corr-option-1", TraceID: "trace-option-1",
			Value: []byte(`{
				"tenant_id":"tenant-local","source_id":"src-massive","app_id":"marketops","domain":"market_data",
				"use_case":"daily_market_surveillance","source_domain":"market_data","source_adapter":"market_data.massive",
				"ingestion_mode":"scheduled_pull","dataset":"options_contracts_daily",
				"event_id":"evt-option-1","event_type":"market_data.massive.options_contract_daily","idempotency_key":"idem-option-1",
				"observation_time":"2026-07-08T00:00:00Z","processing_time":"2026-07-08T20:00:01Z",
				"payload":{
					"provider":"massive","dataset":"options_contracts_daily","provider_contract_id":"contract-123",
					"option_ticker":"O:SPY260116C00600000","underlying_symbol":"spy","contract_type":"CALL",
					"expiration_date":"2026-01-16","strike_price":600,"observation_date":"2026-07-08",
					"open":11.2,"high":12.8,"low":10.9,"close":12.34,"volume":1200,"open_interest":4000,"vwap":12.01,
					"raw":{"source_field":"value"}
				},
				"entity_hints":[{"type":"option_contract","external_id":"O:SPY260116C00600000"},{"type":"ticker","external_id":"SPY"}],
				"metadata":{},"correlation_id":"corr-option-1"
			}`),
		},
		Partition: 2, Offset: 9,
	}
}
