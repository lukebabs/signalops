package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRawSignalEventJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 7, 6, 20, 31, 1, 0, time.UTC)
	event := RawSignalEvent{
		EventEnvelope: EventEnvelope{
			TenantID:       "tenant_123",
			SourceID:       "src_123",
			SourceDomain:   SourceDomainMarketData,
			SourceAdapter:  "market_data.massive",
			IngestionMode:  IngestionModeScheduledPull,
			Dataset:        "options_daily_prices",
			EventID:        "evt_123",
			EventType:      "option.daily_ohlc",
			SchemaID:       "raw_signal_event",
			SchemaVersion:  "1.0.0",
			ObservationAt:  ts,
			EffectiveAt:    ts,
			ProcessingAt:   ts,
			OccurredAt:     ts,
			ObservedAt:     ts,
			Metadata:       map[string]any{"provider": "massive"},
			CorrelationID:  "corr_123",
			IdempotencyKey: "tenant_123:src_123:evt_123",
		},
		Payload: map[string]any{"ticker": "O:SPY260116C00600000"},
		EntityHints: []EntityHint{{
			Type:       "option_contract",
			ExternalID: "O:SPY260116C00600000",
		}},
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal raw signal event: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal raw signal event: %v", err)
	}

	assertString(t, decoded, "source_domain", string(SourceDomainMarketData))
	assertString(t, decoded, "ingestion_mode", string(IngestionModeScheduledPull))
	assertString(t, decoded, "observation_time", "2026-07-06T20:31:01Z")
	assertString(t, decoded, "idempotency_key", "tenant_123:src_123:evt_123")
}

func TestSignalJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 7, 6, 20, 31, 1, 0, time.UTC)
	signal := Signal{
		SignalID:        "sig_123",
		TenantID:        "tenant_123",
		SourceID:        "src_123",
		SourceDomain:    SourceDomainMarketData,
		SourceAdapter:   "market_data.massive",
		IngestionMode:   IngestionModeScheduledPull,
		Dataset:         "options_daily_prices",
		EventIDs:        []string{"evt_123"},
		ArtifactIDs:     []string{},
		SignalType:      "options_price_volume_anomaly",
		DetectorID:      "detector_options_volume_v1",
		DetectorVersion: "1.0.0",
		ModelVersion:    "none",
		Timestamp:       ts,
		ObservationAt:   ts,
		EffectiveAt:     ts,
		ProcessingAt:    ts,
		WindowStart:     ts.Add(-24 * time.Hour),
		WindowEnd:       ts,
		Confidence:      0.88,
		Severity:        SeverityHigh,
		Entities: []EntityRef{{
			Type: "option_contract",
			ID:   "contract_123",
		}},
		SupportingMetrics: map[string]any{"volume_zscore": 4.2},
		GraphTargets:      []map[string]any{},
		SemanticEvidence:  []map[string]any{},
		Evidence: []EvidenceRef{{
			Type: "event",
			Ref:  "evt_123",
		}},
		Recommendation: map[string]any{"action": "review"},
		CorrelationID:  "corr_123",
	}

	payload, err := json.Marshal(signal)
	if err != nil {
		t.Fatalf("marshal signal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal signal: %v", err)
	}

	assertString(t, decoded, "signal_id", "sig_123")
	assertString(t, decoded, "source_domain", string(SourceDomainMarketData))
	assertString(t, decoded, "severity", string(SeverityHigh))
	if decoded["confidence"] != 0.88 {
		t.Fatalf("confidence = %v, want 0.88", decoded["confidence"])
	}
}

func assertString(t *testing.T, values map[string]any, key string, want string) {
	t.Helper()
	got, ok := values[key].(string)
	if !ok {
		t.Fatalf("%s is %T, want string", key, values[key])
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
