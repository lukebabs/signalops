package signals

import (
	"context"
	"strings"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakeRepository struct {
	record   storage.SignalLedgerRecord
	alerts   []storage.AlertLedgerRecord
	insights []storage.InsightLedgerRecord
	err      error
}

func (r *fakeRepository) UpsertSignalLedger(_ context.Context, record storage.SignalLedgerRecord) error {
	r.record = record
	return r.err
}

func (r *fakeRepository) UpsertAlertLedger(_ context.Context, record storage.AlertLedgerRecord) error {
	r.alerts = append(r.alerts, record)
	return r.err
}

func (r *fakeRepository) UpsertInsightLedger(_ context.Context, record storage.InsightLedgerRecord) error {
	r.insights = append(r.insights, record)
	return r.err
}

func (r *fakeRepository) PersistSignalLifecycle(_ context.Context, record storage.SignalLedgerRecord, alerts []storage.AlertLedgerRecord, insights []storage.InsightLedgerRecord) error {
	r.record = record
	r.alerts = append([]storage.AlertLedgerRecord(nil), alerts...)
	r.insights = append([]storage.InsightLedgerRecord(nil), insights...)
	return r.err
}

func TestDecodeEventValidatesSignalContract(t *testing.T) {
	event, err := DecodeEvent(validSignalJSON())
	if err != nil {
		t.Fatal(err)
	}
	if event.SignalID != "signal-g045" || event.DetectorID != "signalops.static_test" ||
		event.Severity != "high" || len(event.EventIDs) != 1 {
		t.Fatalf("event = %+v", event)
	}
}

func TestDecodeEventRejectsUnknownFields(t *testing.T) {
	value := strings.Replace(string(validSignalJSON()), `"signal_id":"signal-g045"`, `"signal_id":"signal-g045","unknown":true`, 1)
	if _, err := DecodeEvent([]byte(value)); err == nil {
		t.Fatal("DecodeEvent() expected unknown field error")
	}
}

func TestProcessorPersistsSignalBrokerLineage(t *testing.T) {
	repository := &fakeRepository{}
	message := broker.ConsumedMessage{
		Message:   broker.Message{Topic: "signalops.test.signal.v1", Key: "signal-g045", Value: validSignalJSON()},
		Partition: 2, Offset: 11,
	}
	record, err := (Processor{Repository: repository}).Process(context.Background(), message)
	if err != nil {
		t.Fatal(err)
	}
	if record.BrokerTopic != message.Topic || record.BrokerPartition != 2 || record.BrokerOffset != 11 {
		t.Fatalf("broker lineage = %+v", record)
	}
	if repository.record.SignalID != "signal-g045" || repository.record.EventIDs[0] != "event-g045" {
		t.Fatalf("persisted record = %+v", repository.record)
	}
	if len(repository.alerts) != 1 || repository.alerts[0].AlertID != "alert:signal-g045" || repository.alerts[0].Status != storage.AlertStatusOpen {
		t.Fatalf("alerts = %+v", repository.alerts)
	}
	if len(repository.insights) != 1 || repository.insights[0].InsightID != "insight:signal-g045" || repository.insights[0].Status != storage.InsightStatusActive {
		t.Fatalf("insights = %+v", repository.insights)
	}
}

func TestProcessorDoesNotCreateAlertForLowSeverity(t *testing.T) {
	repository := &fakeRepository{}
	value := strings.Replace(string(validSignalJSON()), `"severity":"high"`, `"severity":"low"`, 1)
	message := broker.ConsumedMessage{
		Message: broker.Message{Topic: "signalops.test.signal.v1", Key: "signal-g045", Value: []byte(value)},
	}
	if _, err := (Processor{Repository: repository}).Process(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(repository.alerts) != 0 {
		t.Fatalf("alerts = %+v", repository.alerts)
	}
	if len(repository.insights) != 1 {
		t.Fatalf("insights = %+v", repository.insights)
	}
}

func validSignalJSON() []byte {
	return []byte(`{
		"signal_id":"signal-g045","tenant_id":"tenant-local","source_id":"src-g045",
		"source_domain":"iot","source_adapter":"iot.generic.sensor","ingestion_mode":"push_event",
		"dataset":"sensor_observations","event_ids":["event-g045"],"artifact_ids":[],
		"signal_type":"temperature.anomaly","detector_id":"signalops.static_test",
		"detector_version":"1.0.0","model_version":"static-1",
		"timestamp":"2026-07-08T22:00:00Z","observation_time":"2026-07-08T21:59:00Z",
		"effective_time":"2026-07-08T21:59:00Z","processing_time":"2026-07-08T22:00:00Z",
		"window_start":"2026-07-08T21:58:00Z","window_end":"2026-07-08T21:59:00Z",
		"confidence":0.92,"severity":"high",
		"entities":[{"type":"sensor","id":"sensor:45"}],
		"supporting_metrics":{"temperature_c":45.2},"graph_targets":[],
		"semantic_evidence":[],"evidence":[{"type":"normalized_event","ref":"event-g045"}],
		"recommendation":{"action":"inspect_sensor"},"correlation_id":"corr-g045",
		"trace_id":"trace-g045","causation_id":"event-g045","replay_job_id":""
	}`)
}
