package detection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lukebabs/signalops/internal/normalization"
	"github.com/lukebabs/signalops/internal/signals"
	"github.com/lukebabs/signalops/pkg/broker"
)

const AllowedServiceExposureDetector = "cyberops.firewall.allowed_service_exposure.v1"

type exposureState struct {
	FirstObserved time.Time `json:"first_observed"`
	SignalEmitted bool      `json:"signal_emitted"`
}

type Processor struct {
	Publisher   broker.Publisher
	SignalTopic string
	StateTopic  string
	IoTCIDRs    func(context.Context, string) ([]string, error)
	mu          sync.Mutex
	states      map[string]exposureState
}

func (p *Processor) Restore(key string, value []byte) error {
	var state exposureState
	if err := json.Unmarshal(value, &state); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.states == nil {
		p.states = map[string]exposureState{}
	}
	if existing, ok := p.states[key]; !ok || state.FirstObserved.Before(existing.FirstObserved) || (state.FirstObserved.Equal(existing.FirstObserved) && state.SignalEmitted && !existing.SignalEmitted) {
		p.states[key] = state
	}
	return nil
}

func (p *Processor) Process(ctx context.Context, message broker.ConsumedMessage) error {
	var event normalization.Event
	if json.Unmarshal(message.Value, &event) != nil || event.AppID != "cyberops" || event.Dataset != "cyberops.syslog.raw" {
		return nil
	}
	firewall, ok := ParseOPNsenseFilterlog(stringValue(event.NormalizedPayload["message"]))
	if !ok {
		return nil
	}
	if p.IoTCIDRs != nil {
		if err := p.processIoT(ctx, event, firewall); err != nil {
			return err
		}
	}
	if !IsPublicRoutable(firewall.SourceIP) {
		return nil
	}
	key := exposureKey(event.TenantID, firewall)
	p.mu.Lock()
	if p.states == nil {
		p.states = map[string]exposureState{}
	}
	state, seen := p.states[key]
	p.mu.Unlock()
	if seen && state.SignalEmitted {
		return nil
	}

	firstObserved := event.OccurredAt.UTC()
	if seen && !state.FirstObserved.IsZero() {
		firstObserved = state.FirstObserved
	}
	recommendation := map[string]any{
		"action": "review_public_service_exposure", "destination_ip": firewall.DestinationIP,
		"protocol": firewall.Protocol, "destination_port": firewall.DestinationPort,
		"first_observed_at": firstObserved.Format(time.RFC3339),
	}
	if err := p.publish(ctx, signal(event, firewall, AllowedServiceExposureDetector, "cyberops.firewall.new_public_service_exposure", "low", []string{event.EventID}, firstObserved, firstObserved, recommendation)); err != nil {
		return err
	}
	state = exposureState{FirstObserved: firstObserved, SignalEmitted: true}
	if p.StateTopic != "" {
		value, _ := json.Marshal(state)
		_, err := p.Publisher.Publish(ctx, broker.Message{Topic: p.StateTopic, Key: key, Value: value, CorrelationID: event.CorrelationID, CausationID: event.EventID, TraceID: event.TraceID, PublishedAt: time.Now().UTC()})
		if err != nil {
			return err
		}
	}
	p.mu.Lock()
	p.states[key] = state
	p.mu.Unlock()
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
	id := stable(detector + "|" + exposureKey(event.TenantID, firewall))
	title := fmt.Sprintf("New public service exposure: %s/%d", strings.ToUpper(firewall.Protocol), firewall.DestinationPort)
	summary := fmt.Sprintf("First observed allowed connection from public source %s to %s:%d over %s.", firewall.SourceIP, firewall.DestinationIP, firewall.DestinationPort, strings.ToUpper(firewall.Protocol))
	return signals.Event{SignalID: id, TenantID: event.TenantID, SourceID: event.SourceID, AppID: "cyberops", Domain: "security", UseCase: "cyberops", SourceDomain: "security", SourceAdapter: event.SourceAdapter, IngestionMode: event.IngestionMode, Dataset: event.Dataset, EventIDs: eventIDs, ArtifactIDs: []string{}, SignalType: kind, DetectorID: detector, DetectorVersion: "1.0.0", ModelVersion: "deterministic-v1", Timestamp: end, ObservationTime: event.ObservationTime, EffectiveTime: event.EffectiveTime, ProcessingTime: time.Now().UTC(), WindowStart: start, WindowEnd: end, Confidence: 0.9, Severity: severity, InsightTitle: title, InsightSummary: summary, Entities: []map[string]any{{"type": "ip", "value": firewall.SourceIP, "role": "source"}, {"type": "ip", "value": firewall.DestinationIP, "role": "destination"}}, SupportingMetrics: recommendation, GraphTargets: []map[string]any{}, SemanticEvidence: []map[string]any{{"parser": "opnsense.filterlog.v1", "action": firewall.Action, "protocol": firewall.Protocol, "destination_port": firewall.DestinationPort}}, Evidence: []map[string]any{{"type": "normalized_event", "ref": event.EventID}}, Recommendation: mustJSON(recommendation), CorrelationID: event.CorrelationID, TraceID: event.TraceID, CausationID: event.EventID}
}

func exposureKey(tenantID string, firewall FirewallEvent) string {
	return tenantID + "|" + firewall.DestinationIP + "|" + firewall.Protocol + "|" + strconv.Itoa(firewall.DestinationPort)
}

func stable(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "cybsig_" + hex.EncodeToString(sum[:16])
}
func mustJSON(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
func stringValue(value any) string       { text, _ := value.(string); return strings.TrimSpace(text) }

func (p *Processor) processIoT(ctx context.Context, event normalization.Event, firewall FirewallEvent) error {
	cidrs, err := p.IoTCIDRs(ctx, event.TenantID)
	if err != nil || len(cidrs) == 0 {
		return err
	}
	prefixes := []netip.Prefix{}
	for _, value := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	source, _ := netip.ParseAddr(firewall.SourceIP)
	destination, _ := netip.ParseAddr(firewall.DestinationIP)
	inside := func(a netip.Addr) bool {
		for _, p := range prefixes {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	si, di := inside(source), inside(destination)
	pairs := [][3]string{}
	if si {
		direction := "egress"
		if di {
			direction = "lateral"
		}
		pairs = append(pairs, [3]string{firewall.SourceIP, firewall.DestinationIP, direction})
	}
	if di {
		direction := "ingress"
		if si {
			direction = "lateral"
		}
		pairs = append(pairs, [3]string{firewall.DestinationIP, firewall.SourceIP, direction})
	}
	for _, pair := range pairs {
		device, peer, direction := pair[0], pair[1], pair[2]
		deviceKey := "iot:device:" + event.TenantID + "|" + device
		peerKey := "iot:peer:" + event.TenantID + "|" + device + "|" + peer + "|" + direction
		pathKey := peerKey + "|" + firewall.Protocol + "|" + strconv.Itoa(firewall.DestinationPort)
		p.mu.Lock()
		deviceState, knownDevice := p.states[deviceKey]
		_, knownPeer := p.states[peerKey]
		_, knownPath := p.states[pathKey]
		p.mu.Unlock()
		first := event.OccurredAt.UTC()
		if !knownDevice {
			deviceState = exposureState{FirstObserved: first}
			if err := p.persistState(ctx, deviceKey, deviceState, event); err != nil {
				return err
			}
		}
		mature := knownDevice && first.Sub(deviceState.FirstObserved) >= 7*24*time.Hour
		if mature && !knownPath {
			kind, title := "cyberops.iot.new_service", "New device service"
			if !knownPeer {
				kind = "cyberops.iot.new_peer"
				title = "New device communication peer"
			}
			s := signal(event, firewall, "cyberops.iot.behaviour.v1", kind, "low", []string{event.EventID}, first, first, map[string]any{"action": "review_device_behaviour_change", "device_ip": device, "peer_ip": peer, "direction": direction, "baseline_days": 7})
			s.SignalID = stable("iot|" + event.TenantID + "|" + device + "|" + peer + "|" + direction + "|" + firewall.Protocol + "|" + strconv.Itoa(firewall.DestinationPort))
			s.InsightTitle = title
			s.InsightSummary = fmt.Sprintf("Device %s established a newly observed %s relationship with %s over %s/%d after its seven-day learning period.", device, direction, peer, strings.ToUpper(firewall.Protocol), firewall.DestinationPort)
			if err := p.publish(ctx, s); err != nil {
				return err
			}
		}
		state := exposureState{FirstObserved: first, SignalEmitted: true}
		for _, item := range []struct {
			k string
			s exposureState
		}{{peerKey, state}, {pathKey, state}} {
			if err := p.persistState(ctx, item.k, item.s, event); err != nil {
				return err
			}
		}
	}
	return nil
}
func (p *Processor) persistState(ctx context.Context, key string, state exposureState, event normalization.Event) error {
	if p.StateTopic != "" {
		value, _ := json.Marshal(state)
		if _, err := p.Publisher.Publish(ctx, broker.Message{Topic: p.StateTopic, Key: key, Value: value, CorrelationID: event.CorrelationID, CausationID: event.EventID, TraceID: event.TraceID, PublishedAt: time.Now().UTC()}); err != nil {
			return err
		}
	}
	p.mu.Lock()
	if p.states == nil {
		p.states = map[string]exposureState{}
	}
	if _, ok := p.states[key]; !ok {
		p.states[key] = state
	}
	p.mu.Unlock()
	return nil
}
