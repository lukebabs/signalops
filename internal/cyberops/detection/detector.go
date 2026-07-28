package detection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/lukebabs/signalops/internal/normalization"
	"github.com/lukebabs/signalops/internal/signals"
	"github.com/lukebabs/signalops/pkg/broker"
)

const (
	ExternalDenyDetector = "cyberops.firewall.external_deny.v1"
	PortScanDetector     = "cyberops.network.port_scan.v1"
)

type scanState struct {
	Latest        time.Time         `json:"latest"`
	Ports         map[int]time.Time `json:"ports"`
	EmittedWindow string            `json:"emitted_window"`
}

type Processor struct {
	Publisher   broker.Publisher
	SignalTopic string
	StateTopic  string
	mu          sync.Mutex
	states      map[string]scanState
}

func (p *Processor) Restore(key string, value []byte) error {
	var state scanState
	if err := json.Unmarshal(value, &state); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.states == nil {
		p.states = map[string]scanState{}
	}
	p.states[key] = state
	return nil
}

func (p *Processor) Process(ctx context.Context, message broker.ConsumedMessage) error {
	var event normalization.Event
	if json.Unmarshal(message.Value, &event) != nil || event.AppID != "cyberops" || event.Dataset != "cyberops.syslog.raw" {
		return nil
	}
	firewall, ok := ParseOPNsenseFilterlog(stringValue(event.NormalizedPayload["message"]))
	if !ok || !IsPublicRoutable(firewall.SourceIP) {
		return nil
	}
	if err := p.publish(ctx, signal(event, firewall, ExternalDenyDetector, "cyberops.firewall.external_deny", "medium", []string{event.EventID}, event.OccurredAt, event.OccurredAt, map[string]any{"action": "review_blocked_external_connection", "protocol": firewall.Protocol, "destination_port": firewall.DestinationPort})); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.states == nil {
		p.states = map[string]scanState{}
	}
	key := event.TenantID + "|" + firewall.SourceIP + "|" + firewall.DestinationIP
	state := p.states[key]
	if state.Ports == nil {
		state.Ports = map[int]time.Time{}
	}
	if event.OccurredAt.After(state.Latest) {
		state.Latest = event.OccurredAt
	}
	cutoff := state.Latest.Add(-5 * time.Minute)
	for port, seen := range state.Ports {
		if seen.Before(cutoff) {
			delete(state.Ports, port)
		}
	}
	state.Ports[firewall.DestinationPort] = event.OccurredAt
	window := state.Latest.Truncate(5 * time.Minute).Format(time.RFC3339)
	if len(state.Ports) >= 10 && state.EmittedWindow != window {
		ids := []string{event.EventID}
		if err := p.publish(ctx, signal(event, firewall, PortScanDetector, "cyberops.network.port_scan", "high", ids, state.Latest.Add(-5*time.Minute), state.Latest, map[string]any{"action": "investigate_port_scan", "distinct_destination_ports": len(state.Ports), "window_minutes": 5})); err != nil {
			return err
		}
		state.EmittedWindow = window
	}
	p.states[key] = state
	if p.StateTopic != "" {
		value, _ := json.Marshal(state)
		_, err := p.Publisher.Publish(ctx, broker.Message{Topic: p.StateTopic, Key: key, Value: value, CorrelationID: event.CorrelationID, CausationID: event.EventID, TraceID: event.TraceID, PublishedAt: time.Now().UTC()})
		if err != nil {
			return err
		}
	}
	return nil
}
func (p *Processor) publish(ctx context.Context, event signals.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = p.Publisher.Publish(ctx, broker.Message{Topic: p.SignalTopic, Key: event.SignalID, Value: value, CorrelationID: event.CorrelationID, CausationID: event.CausationID, TraceID: event.TraceID, PublishedAt: time.Now().UTC()})
	return err
}
func signal(event normalization.Event, firewall FirewallEvent, detector, kind, severity string, eventIDs []string, start, end time.Time, recommendation map[string]any) signals.Event {
	id := stable(detector + "|" + event.TenantID + "|" + firewall.SourceIP + "|" + firewall.DestinationIP + "|" + end.Truncate(5*time.Minute).Format(time.RFC3339))
	return signals.Event{SignalID: id, TenantID: event.TenantID, SourceID: event.SourceID, AppID: "cyberops", Domain: "security", UseCase: "cyberops", SourceDomain: "security", SourceAdapter: event.SourceAdapter, IngestionMode: event.IngestionMode, Dataset: event.Dataset, EventIDs: eventIDs, ArtifactIDs: []string{}, SignalType: kind, DetectorID: detector, DetectorVersion: "1.0.0", ModelVersion: "deterministic-v1", Timestamp: end, ObservationTime: event.ObservationTime, EffectiveTime: event.EffectiveTime, ProcessingTime: time.Now().UTC(), WindowStart: start, WindowEnd: end, Confidence: 0.9, Severity: severity, Entities: []map[string]any{{"type": "ip", "value": firewall.SourceIP, "role": "source"}, {"type": "ip", "value": firewall.DestinationIP, "role": "destination"}}, SupportingMetrics: recommendation, GraphTargets: []map[string]any{}, SemanticEvidence: []map[string]any{{"parser": "opnsense.filterlog.v1", "action": firewall.Action, "protocol": firewall.Protocol, "destination_port": firewall.DestinationPort}}, Evidence: []map[string]any{{"type": "normalized_event", "ref": event.EventID}}, Recommendation: mustJSON(recommendation), CorrelationID: event.CorrelationID, TraceID: event.TraceID, CausationID: event.EventID}
}
func stable(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "cybsig_" + hex.EncodeToString(sum[:16])
}
func mustJSON(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
func stringValue(value any) string       { text, _ := value.(string); return strings.TrimSpace(text) }
