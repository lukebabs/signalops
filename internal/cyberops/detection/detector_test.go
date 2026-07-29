package detection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/normalization"
	"github.com/lukebabs/signalops/internal/signals"
	"github.com/lukebabs/signalops/pkg/broker"
)

type fakePublisher struct{ messages []broker.Message }

func (p *fakePublisher) Publish(_ context.Context, message broker.Message) (broker.PublishResult, error) {
	p.messages = append(p.messages, message)
	return broker.PublishResult{Topic: message.Topic}, nil
}
func (p *fakePublisher) Close(context.Context) error { return nil }

func TestParseOPNsenseFilterlogAcceptsExplicitAllowActions(t *testing.T) {
	for _, action := range []string{"pass", "allow"} {
		event, ok := ParseOPNsenseFilterlog(filterlogMessage(action, "8.8.8.8", "443"))
		if !ok {
			t.Fatalf("%s was not parsed", action)
		}
		if event.Action != action || event.Protocol != "tcp" || event.SourceIP != "8.8.8.8" || event.DestinationIP != "10.0.0.1" || event.DestinationPort != 443 {
			t.Fatalf("event = %+v", event)
		}
	}
}

func TestParseOPNsenseFilterlogAcceptsDeniedTraffic(t *testing.T) {
	for _, action := range []string{"block", "deny"} {
		event, ok := ParseOPNsenseFilterlog(filterlogMessage(action, "8.8.8.8", "443"))
		if !ok || event.Action != action {
			t.Fatalf("%s was not parsed: %+v", action, event)
		}
	}
}

func TestProcessorEmitsOneInsightSignalPerAllowedService(t *testing.T) {
	publisher := &fakePublisher{}
	processor := &Processor{Publisher: publisher, SignalTopic: "signals", StateTopic: "state"}
	if err := processor.Process(context.Background(), normalizedMessage(t, "event-1", filterlogMessage("pass", "8.8.8.8", "443"))); err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), normalizedMessage(t, "event-2", filterlogMessage("allow", "1.1.1.1", "443"))); err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 2 {
		t.Fatalf("published messages = %d, want one durable signal per allowed event", len(publisher.messages))
	}
	var event signals.Event
	if err := json.Unmarshal(publisher.messages[0].Value, &event); err != nil {
		t.Fatal(err)
	}
	if event.DetectorID != AllowedServiceExposureDetector || event.SignalType != "cyberops.firewall.new_public_service_exposure" || event.Severity != "low" {
		t.Fatalf("signal = %+v", event)
	}
	if event.InsightTitle != "New public service exposure: TCP/443" || event.InsightSummary == "" {
		t.Fatalf("presentation = %+v", event)
	}
	var second signals.Event
	if err := json.Unmarshal(publisher.messages[1].Value, &second); err != nil || second.SignalID == event.SignalID {
		t.Fatalf("second durable signal = %+v, err=%v", second, err)
	}
}

func TestProcessorRestoredExposureStillEmitsDurableEvidence(t *testing.T) {
	publisher := &fakePublisher{}
	processor := &Processor{Publisher: publisher, SignalTopic: "signals", StateTopic: "state"}
	firewall, ok := ParseOPNsenseFilterlog(filterlogMessage("pass", "8.8.8.8", "443"))
	if !ok {
		t.Fatal("test filterlog did not parse")
	}
	state, err := json.Marshal(exposureState{FirstObserved: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), SignalEmitted: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Restore(exposureKey("tenant-local", firewall), state); err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), normalizedMessage(t, "event-1", filterlogMessage("pass", "8.8.8.8", "443"))); err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("published messages = %+v", publisher.messages)
	}
}

func TestProcessorIgnoresPrivateSourcesAndRetainsExternalDenies(t *testing.T) {
	publisher := &fakePublisher{}
	processor := &Processor{Publisher: publisher, SignalTopic: "signals", StateTopic: "state"}
	for _, value := range []string{filterlogMessage("pass", "10.1.2.3", "443"), filterlogMessage("block", "8.8.8.8", "443")} {
		if err := processor.Process(context.Background(), normalizedMessage(t, "event", value)); err != nil {
			t.Fatal(err)
		}
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("published messages = %+v", publisher.messages)
	}
}

func normalizedMessage(t *testing.T, eventID, message string) broker.ConsumedMessage {
	t.Helper()
	occurredAt := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	value, err := json.Marshal(normalization.Event{
		TenantID: "tenant-local", SourceID: "firewall-1", AppID: "cyberops", Domain: "security", UseCase: "cyberops",
		SourceDomain: "security", SourceAdapter: "connect:opnsense", IngestionMode: "push_event", Dataset: "cyberops.syslog.raw",
		EventID: eventID, EventType: "cyberops.syslog.raw", SchemaID: "signalops.normalized_event.v1", SchemaVersion: "1.0.0",
		ObservationTime: occurredAt, EffectiveTime: occurredAt, ProcessingTime: occurredAt, OccurredAt: occurredAt, ObservedAt: occurredAt,
		NormalizedPayload: map[string]any{"message": message}, CorrelationID: eventID, CausationID: eventID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return broker.ConsumedMessage{Message: broker.Message{Topic: "normalized", Key: eventID, Value: value}}
}

func filterlogMessage(action, source, port string) string {
	return "x,x," + action + ",x,x,x,x,x,x,tcp,x," + source + ",10.0.0.1,40000," + port
}
