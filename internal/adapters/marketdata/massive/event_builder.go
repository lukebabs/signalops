package massive

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/pkg/contracts"
)

const (
	AdapterID = "market_data.massive"

	DatasetOptionsContractsDaily = "options_contracts_daily"
	DatasetEquityEODPrices       = "equity_eod_prices"

	EventTypeOptionContractDaily = "market_data.massive.options_contract_daily"
	EventTypeEquityEODPrice      = "market_data.massive.equity_eod_price"

	RawSignalSchemaID      = "signalops.raw_signal_event.v1"
	RawSignalSchemaVersion = "1.0.0"
)

type AdapterConfig struct {
	TenantID      string
	SourceID      string
	CorrelationID string
	TraceID       string
	ProcessingAt  time.Time
}

type OptionContractDailyRecord struct {
	ProviderContractID string
	OptionTicker       string
	UnderlyingSymbol   string
	ContractType       string
	ExpirationDate     time.Time
	StrikePrice        float64
	ObservationDate    time.Time
	Open               *float64
	High               *float64
	Low                *float64
	Close              *float64
	Volume             *int64
	OpenInterest       *int64
	VWAP               *float64
	Raw                map[string]any
}

type EquityEODPriceRecord struct {
	ProviderEventID string
	Symbol          string
	ObservationDate time.Time
	Open            *float64
	High            *float64
	Low             *float64
	Close           *float64
	Volume          *int64
	VWAP            *float64
	Raw             map[string]any
}

func BuildOptionContractDailyEvent(cfg AdapterConfig, record OptionContractDailyRecord) (contracts.RawSignalEvent, error) {
	if err := validateConfig(cfg); err != nil {
		return contracts.RawSignalEvent{}, err
	}
	optionTicker := strings.TrimSpace(record.OptionTicker)
	underlying := normalizeSymbol(record.UnderlyingSymbol)
	contractType := strings.ToLower(strings.TrimSpace(record.ContractType))
	observationAt, err := dayUTC(record.ObservationDate, "observation_date")
	if err != nil {
		return contracts.RawSignalEvent{}, err
	}
	expirationAt, err := dayUTC(record.ExpirationDate, "expiration_date")
	if err != nil {
		return contracts.RawSignalEvent{}, err
	}
	if optionTicker == "" {
		return contracts.RawSignalEvent{}, errors.New("option ticker is required")
	}
	if underlying == "" {
		return contracts.RawSignalEvent{}, errors.New("underlying symbol is required")
	}
	if contractType == "" {
		return contracts.RawSignalEvent{}, errors.New("contract type is required")
	}
	if record.StrikePrice <= 0 {
		return contracts.RawSignalEvent{}, errors.New("strike price must be greater than zero")
	}

	processingAt := processingTime(cfg)
	eventID := stableID("evt", cfg.TenantID, cfg.SourceID, DatasetOptionsContractsDaily, optionTicker, dateKey(observationAt))
	idempotencyKey := stableID("idem", cfg.TenantID, cfg.SourceID, DatasetOptionsContractsDaily, optionTicker, dateKey(observationAt))

	payload := map[string]any{
		"provider":             "massive",
		"dataset":              DatasetOptionsContractsDaily,
		"provider_contract_id": strings.TrimSpace(record.ProviderContractID),
		"option_ticker":        optionTicker,
		"underlying_symbol":    underlying,
		"contract_type":        contractType,
		"expiration_date":      dateKey(expirationAt),
		"strike_price":         record.StrikePrice,
		"observation_date":     dateKey(observationAt),
	}
	addOptionalFloat(payload, "open", record.Open)
	addOptionalFloat(payload, "high", record.High)
	addOptionalFloat(payload, "low", record.Low)
	addOptionalFloat(payload, "close", record.Close)
	addOptionalInt(payload, "volume", record.Volume)
	addOptionalInt(payload, "open_interest", record.OpenInterest)
	addOptionalFloat(payload, "vwap", record.VWAP)
	if record.Raw != nil {
		payload["raw"] = record.Raw
	}

	return contracts.RawSignalEvent{
		EventEnvelope: contracts.EventEnvelope{
			TenantID:       strings.TrimSpace(cfg.TenantID),
			SourceID:       strings.TrimSpace(cfg.SourceID),
			SourceDomain:   contracts.SourceDomainMarketData,
			SourceAdapter:  AdapterID,
			IngestionMode:  contracts.IngestionModeScheduledPull,
			Dataset:        DatasetOptionsContractsDaily,
			EventID:        eventID,
			EventType:      EventTypeOptionContractDaily,
			SchemaID:       RawSignalSchemaID,
			SchemaVersion:  RawSignalSchemaVersion,
			ObservationAt:  observationAt,
			EffectiveAt:    observationAt,
			ProcessingAt:   processingAt,
			OccurredAt:     observationAt,
			ObservedAt:     observationAt,
			Metadata:       metadata(DatasetOptionsContractsDaily),
			CorrelationID:  correlationID(cfg, eventID),
			IdempotencyKey: idempotencyKey,
			TraceID:        strings.TrimSpace(cfg.TraceID),
		},
		Payload: payload,
		EntityHints: []contracts.EntityHint{
			{Type: "option_contract", ExternalID: optionTicker},
			{Type: "ticker", ExternalID: underlying},
		},
	}, nil
}

func BuildEquityEODPriceEvent(cfg AdapterConfig, record EquityEODPriceRecord) (contracts.RawSignalEvent, error) {
	if err := validateConfig(cfg); err != nil {
		return contracts.RawSignalEvent{}, err
	}
	symbol := normalizeSymbol(record.Symbol)
	observationAt, err := dayUTC(record.ObservationDate, "observation_date")
	if err != nil {
		return contracts.RawSignalEvent{}, err
	}
	if symbol == "" {
		return contracts.RawSignalEvent{}, errors.New("symbol is required")
	}

	processingAt := processingTime(cfg)
	eventID := stableID("evt", cfg.TenantID, cfg.SourceID, DatasetEquityEODPrices, symbol, dateKey(observationAt))
	idempotencyKey := stableID("idem", cfg.TenantID, cfg.SourceID, DatasetEquityEODPrices, symbol, dateKey(observationAt))

	payload := map[string]any{
		"provider":          "massive",
		"dataset":           DatasetEquityEODPrices,
		"provider_event_id": strings.TrimSpace(record.ProviderEventID),
		"symbol":            symbol,
		"observation_date":  dateKey(observationAt),
	}
	addOptionalFloat(payload, "open", record.Open)
	addOptionalFloat(payload, "high", record.High)
	addOptionalFloat(payload, "low", record.Low)
	addOptionalFloat(payload, "close", record.Close)
	addOptionalInt(payload, "volume", record.Volume)
	addOptionalFloat(payload, "vwap", record.VWAP)
	if record.Raw != nil {
		payload["raw"] = record.Raw
	}

	return contracts.RawSignalEvent{
		EventEnvelope: contracts.EventEnvelope{
			TenantID:       strings.TrimSpace(cfg.TenantID),
			SourceID:       strings.TrimSpace(cfg.SourceID),
			SourceDomain:   contracts.SourceDomainMarketData,
			SourceAdapter:  AdapterID,
			IngestionMode:  contracts.IngestionModeScheduledPull,
			Dataset:        DatasetEquityEODPrices,
			EventID:        eventID,
			EventType:      EventTypeEquityEODPrice,
			SchemaID:       RawSignalSchemaID,
			SchemaVersion:  RawSignalSchemaVersion,
			ObservationAt:  observationAt,
			EffectiveAt:    observationAt,
			ProcessingAt:   processingAt,
			OccurredAt:     observationAt,
			ObservedAt:     observationAt,
			Metadata:       metadata(DatasetEquityEODPrices),
			CorrelationID:  correlationID(cfg, eventID),
			IdempotencyKey: idempotencyKey,
			TraceID:        strings.TrimSpace(cfg.TraceID),
		},
		Payload:     payload,
		EntityHints: []contracts.EntityHint{{Type: "ticker", ExternalID: symbol}},
	}, nil
}

func validateConfig(cfg AdapterConfig) error {
	if strings.TrimSpace(cfg.TenantID) == "" {
		return errors.New("tenant id is required")
	}
	if strings.TrimSpace(cfg.SourceID) == "" {
		return errors.New("source id is required")
	}
	return nil
}

func processingTime(cfg AdapterConfig) time.Time {
	if cfg.ProcessingAt.IsZero() {
		return time.Now().UTC()
	}
	return cfg.ProcessingAt.UTC()
}

func correlationID(cfg AdapterConfig, fallback string) string {
	if value := strings.TrimSpace(cfg.CorrelationID); value != "" {
		return value
	}
	return fallback
}

func metadata(dataset string) map[string]any {
	return map[string]any{
		"provider":       "massive",
		"source_adapter": AdapterID,
		"dataset":        dataset,
		"streaming":      false,
	}
}

func dayUTC(value time.Time, name string) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC), nil
}

func normalizeSymbol(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func dateKey(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}

func stableID(prefix string, parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(strings.TrimSpace(part)))
		h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func addOptionalFloat(payload map[string]any, key string, value *float64) {
	if value != nil {
		payload[key] = *value
	}
}

func addOptionalInt(payload map[string]any, key string, value *int64) {
	if value != nil {
		payload[key] = *value
	}
}
