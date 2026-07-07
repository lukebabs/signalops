package massive

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/pkg/contracts"
)

func TestBuildOptionContractDailyEvent(t *testing.T) {
	closePrice := 12.34
	volume := int64(1200)
	openInterest := int64(4000)
	processingAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	event, err := BuildOptionContractDailyEvent(AdapterConfig{
		TenantID:      "tenant-1",
		SourceID:      "src-massive",
		CorrelationID: "corr-1",
		TraceID:       "trace-1",
		ProcessingAt:  processingAt,
	}, OptionContractDailyRecord{
		ProviderContractID: "contract-123",
		OptionTicker:       "O:SPY260116C00600000",
		UnderlyingSymbol:   "spy",
		ContractType:       "CALL",
		ExpirationDate:     time.Date(2026, 1, 16, 15, 30, 0, 0, time.UTC),
		StrikePrice:        600,
		ObservationDate:    time.Date(2026, 7, 6, 20, 0, 0, 0, time.UTC),
		Close:              &closePrice,
		Volume:             &volume,
		OpenInterest:       &openInterest,
		Raw:                map[string]any{"source_field": "value"},
	})
	if err != nil {
		t.Fatalf("build option contract event: %v", err)
	}

	if event.SourceDomain != contracts.SourceDomainMarketData {
		t.Fatalf("source domain = %q", event.SourceDomain)
	}
	if event.SourceAdapter != AdapterID {
		t.Fatalf("source adapter = %q", event.SourceAdapter)
	}
	if event.IngestionMode != contracts.IngestionModeScheduledPull {
		t.Fatalf("ingestion mode = %q", event.IngestionMode)
	}
	if event.Dataset != DatasetOptionsContractsDaily {
		t.Fatalf("dataset = %q", event.Dataset)
	}
	if event.EventType != EventTypeOptionContractDaily {
		t.Fatalf("event type = %q", event.EventType)
	}
	if event.SchemaID != RawSignalSchemaID {
		t.Fatalf("schema id = %q", event.SchemaID)
	}
	if event.CorrelationID != "corr-1" || event.TraceID != "trace-1" {
		t.Fatalf("correlation/trace = %q/%q", event.CorrelationID, event.TraceID)
	}
	if got := event.EntityHints[0].ExternalID; got != "O:SPY260116C00600000" {
		t.Fatalf("option entity = %q", got)
	}
	if got := event.EntityHints[1].ExternalID; got != "SPY" {
		t.Fatalf("underlying entity = %q", got)
	}
	if event.Payload["contract_type"] != "call" {
		t.Fatalf("contract type = %v", event.Payload["contract_type"])
	}
	if event.Payload["close"] != closePrice {
		t.Fatalf("close = %v", event.Payload["close"])
	}
	if event.Payload["volume"] != volume {
		t.Fatalf("volume = %v", event.Payload["volume"])
	}
	if event.Payload["open_interest"] != openInterest {
		t.Fatalf("open interest = %v", event.Payload["open_interest"])
	}
	if event.Payload["expiration_date"] != "2026-01-16" {
		t.Fatalf("expiration date = %v", event.Payload["expiration_date"])
	}
	if event.ObservationAt.Format(time.RFC3339) != "2026-07-06T00:00:00Z" {
		t.Fatalf("observation time = %s", event.ObservationAt.Format(time.RFC3339))
	}

	assertJSONField(t, event, "source_adapter", AdapterID)
	assertJSONField(t, event, "schema_id", RawSignalSchemaID)
}

func TestBuildEquityEODPriceEvent(t *testing.T) {
	open := 500.25
	closePrice := 501.75
	volume := int64(980000)
	processingAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	event, err := BuildEquityEODPriceEvent(AdapterConfig{
		TenantID:     "tenant-1",
		SourceID:     "src-massive",
		ProcessingAt: processingAt,
	}, EquityEODPriceRecord{
		ProviderEventID: "agg-123",
		Symbol:          " qqq ",
		ObservationDate: time.Date(2026, 7, 6, 22, 0, 0, 0, time.UTC),
		Open:            &open,
		Close:           &closePrice,
		Volume:          &volume,
	})
	if err != nil {
		t.Fatalf("build eod price event: %v", err)
	}

	if event.Dataset != DatasetEquityEODPrices {
		t.Fatalf("dataset = %q", event.Dataset)
	}
	if event.EventType != EventTypeEquityEODPrice {
		t.Fatalf("event type = %q", event.EventType)
	}
	if event.CorrelationID != event.EventID {
		t.Fatalf("correlation id = %q, event id = %q", event.CorrelationID, event.EventID)
	}
	if got := event.EntityHints[0].ExternalID; got != "QQQ" {
		t.Fatalf("symbol entity = %q", got)
	}
	if event.Payload["symbol"] != "QQQ" {
		t.Fatalf("payload symbol = %v", event.Payload["symbol"])
	}
	if event.Payload["close"] != closePrice {
		t.Fatalf("close = %v", event.Payload["close"])
	}
	if event.Payload["volume"] != volume {
		t.Fatalf("volume = %v", event.Payload["volume"])
	}
}

func TestBuildEventsUseStableIDs(t *testing.T) {
	cfg := AdapterConfig{TenantID: "tenant-1", SourceID: "src-massive", ProcessingAt: time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)}
	record := EquityEODPriceRecord{Symbol: "SPY", ObservationDate: time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)}

	first, err := BuildEquityEODPriceEvent(cfg, record)
	if err != nil {
		t.Fatalf("build first event: %v", err)
	}
	second, err := BuildEquityEODPriceEvent(cfg, record)
	if err != nil {
		t.Fatalf("build second event: %v", err)
	}

	if first.EventID != second.EventID {
		t.Fatalf("event ids differ: %q != %q", first.EventID, second.EventID)
	}
	if first.IdempotencyKey != second.IdempotencyKey {
		t.Fatalf("idempotency keys differ: %q != %q", first.IdempotencyKey, second.IdempotencyKey)
	}
}

func TestBuildEventsValidateRequiredFields(t *testing.T) {
	_, err := BuildOptionContractDailyEvent(AdapterConfig{TenantID: "tenant-1", SourceID: "src-1"}, OptionContractDailyRecord{})
	if err == nil {
		t.Fatal("expected option validation error")
	}
	_, err = BuildEquityEODPriceEvent(AdapterConfig{TenantID: "tenant-1", SourceID: "src-1"}, EquityEODPriceRecord{})
	if err == nil {
		t.Fatal("expected eod validation error")
	}
	_, err = BuildEquityEODPriceEvent(AdapterConfig{SourceID: "src-1"}, EquityEODPriceRecord{Symbol: "SPY", ObservationDate: time.Now()})
	if err == nil {
		t.Fatal("expected config validation error")
	}
}

func assertJSONField(t *testing.T, event contracts.RawSignalEvent, key string, want string) {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	got, ok := decoded[key].(string)
	if !ok {
		t.Fatalf("%s is %T", key, decoded[key])
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
