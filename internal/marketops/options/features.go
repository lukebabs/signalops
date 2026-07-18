package options

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	DistributionFeatureDataset       = "options_distribution_daily"
	DistributionFeatureSchemaID      = "signalops.normalized_signal_event.v1"
	DistributionFeatureSchemaVersion = "1"
)

func NormalizedEventFromDistribution(record storage.MarketOpsOptionsDistributionRecord, processingTime time.Time) (storage.NormalizedEventLedgerRecord, error) {
	if strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Symbol) == "" || record.TradeDate.IsZero() {
		return storage.NormalizedEventLedgerRecord{}, fmt.Errorf("distribution tenant_id, symbol, and trade_date are required")
	}
	if processingTime.IsZero() {
		processingTime = time.Now().UTC()
	}
	symbol := strings.ToUpper(strings.TrimSpace(record.Symbol))
	windowName := firstNonEmptyString(record.WindowName, DefaultWindowName)
	sourceID := firstNonEmptyString(record.SourceID, "src-massive")
	eventID := stableFeatureID("evt_opt_dist", record.TenantID, symbol, dayOnly(record.TradeDate).Format("2006-01-02"), windowName)
	idempotencyKey := stableFeatureID("idem_opt_dist", record.TenantID, symbol, dayOnly(record.TradeDate).Format("2006-01-02"), windowName)

	metrics := jsonObject(record.MetricsJSON)
	openInterestZeroCount := intFromMetrics(metrics, "open_interest_zero_count")
	openInterestPositiveCount := intFromMetrics(metrics, "open_interest_positive_count")
	openInterestZeroRate := floatFromMetrics(metrics, "open_interest_zero_rate")
	openInterestQuality := stringFromMetrics(metrics, "open_interest_quality")
	callPutOIDenominatorZero := boolFromMetrics(metrics, "call_put_oi_denominator_is_zero")
	callPutOIRatioQuality := stringFromMetrics(metrics, "call_put_oi_ratio_quality")

	payload := map[string]any{
		"provider":                        firstNonEmptyString(record.Provider, "massive"),
		"dataset":                         DistributionFeatureDataset,
		"symbol":                          symbol,
		"asset_type":                      "equity",
		"observation_date":                dayOnly(record.TradeDate).Format("2006-01-02"),
		"window_name":                     windowName,
		"trade_days":                      record.TradeDays,
		"contract_count":                  record.ContractCount,
		"call_contract_count":             record.CallContractCount,
		"put_contract_count":              record.PutContractCount,
		"total_call_open_interest":        record.TotalCallOpenInterest,
		"total_put_open_interest":         record.TotalPutOpenInterest,
		"total_call_volume":               record.TotalCallVolume,
		"total_put_volume":                record.TotalPutVolume,
		"missing_open_interest_count":     record.MissingOpenInterestCount,
		"open_interest_zero_count":        openInterestZeroCount,
		"open_interest_positive_count":    openInterestPositiveCount,
		"open_interest_zero_rate":         openInterestZeroRate,
		"open_interest_quality":           openInterestQuality,
		"call_put_oi_denominator_is_zero": callPutOIDenominatorZero,
		"call_put_oi_ratio_quality":       callPutOIRatioQuality,
		"call_put_open_interest_ratio":    record.CallPutOpenInterestRatio,
		"call_put_volume_ratio":           record.CallPutVolumeRatio,
		"call_put_oi_ratio_delta":         record.RatioDelta,
		"call_put_oi_ratio_change_pct":    record.RatioChangePct,
		"call_put_oi_zscore":              record.RatioZScore,
		"call_put_oi_change_point_score":  record.ChangePointScore,
		"distribution_confidence":         record.Confidence,
		"moneyness_distribution":          jsonObject(record.MoneynessDistributionJSON),
		"expiration_distribution":         jsonObject(record.ExpirationDistributionJSON),
		"source_trade_dates":              dateStrings(record.SourceTradeDates),
		"features": map[string]any{
			"call_put_open_interest_ratio":   record.CallPutOpenInterestRatio,
			"call_put_volume_ratio":          record.CallPutVolumeRatio,
			"call_put_oi_ratio_delta":        record.RatioDelta,
			"call_put_oi_ratio_change_pct":   record.RatioChangePct,
			"call_put_oi_zscore":             record.RatioZScore,
			"call_put_oi_change_point_score": record.ChangePointScore,
			"total_call_open_interest":       record.TotalCallOpenInterest,
			"total_put_open_interest":        record.TotalPutOpenInterest,
			"total_call_volume":              record.TotalCallVolume,
			"total_put_volume":               record.TotalPutVolume,
			"open_interest_zero_rate":        openInterestZeroRate,
		},
	}
	entities := []map[string]any{{"type": "ticker", "external_id": symbol}}
	evidence := []map[string]any{{"type": "marketops_options_distribution", "ref": strings.Join([]string{record.TenantID, symbol, dayOnly(record.TradeDate).Format("2006-01-02"), windowName}, ":"), "metadata": map[string]any{"source": "marketops_options_distribution_daily"}}}
	metadata := map[string]any{"feature_builder": "marketops.options_distribution_feature_v1", "primary_metric": "open_interest", "window_name": windowName, "source_distribution_metrics": metrics}
	event := map[string]any{
		"tenant_id":          record.TenantID,
		"source_id":          sourceID,
		"app_id":             "marketops",
		"domain":             "market_data",
		"use_case":           "daily_market_surveillance",
		"source_domain":      "market_data",
		"source_adapter":     "market_data.massive",
		"ingestion_mode":     "derived_feature",
		"dataset":            DistributionFeatureDataset,
		"event_id":           eventID,
		"event_type":         "marketops.options_distribution_daily",
		"schema_id":          DistributionFeatureSchemaID,
		"schema_version":     DistributionFeatureSchemaVersion,
		"observation_time":   dayOnly(record.TradeDate).Format(time.RFC3339Nano),
		"effective_time":     dayOnly(record.TradeDate).Format(time.RFC3339Nano),
		"processing_time":    processingTime.UTC().Format(time.RFC3339Nano),
		"occurred_at":        dayOnly(record.TradeDate).Format(time.RFC3339Nano),
		"observed_at":        dayOnly(record.TradeDate).Format(time.RFC3339Nano),
		"normalized_payload": payload,
		"entities":           entities,
		"confidence":         1.0,
		"metadata":           metadata,
		"evidence":           evidence,
		"correlation_id":     eventID,
		"idempotency_key":    idempotencyKey,
		"causation_id":       strings.Join([]string{record.TenantID, symbol, dayOnly(record.TradeDate).Format("2006-01-02"), windowName}, ":"),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	entitiesJSON, err := json.Marshal(entities)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return storage.NormalizedEventLedgerRecord{}, err
	}
	return storage.NormalizedEventLedgerRecord{
		EventID: eventID, TenantID: record.TenantID, SourceID: sourceID, AppID: "marketops", Domain: "market_data", UseCase: "daily_market_surveillance",
		SourceAdapter: "market_data.massive", Dataset: DistributionFeatureDataset, IdempotencyKey: idempotencyKey,
		SchemaID: DistributionFeatureSchemaID, SchemaVersion: DistributionFeatureSchemaVersion, ObservationTime: dayOnly(record.TradeDate),
		ProcessingTime: processingTime.UTC(), Confidence: 1.0, RawTopic: "signalops.internal.marketops.options_distribution.v1",
		RawPartition: -1, RawOffset: stableFeatureOffset(eventID), NormalizedTopic: "signalops.internal.normalized.v1", NormalizedPartition: -1, NormalizedOffset: -1,
		NormalizedPayload: payloadJSON, EntitiesJSON: entitiesJSON, EvidenceJSON: evidenceJSON, MetadataJSON: metadataJSON, EventJSON: eventJSON,
	}, nil
}

func intFromMetrics(metrics map[string]any, key string) int {
	value, ok := metrics[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func floatFromMetrics(metrics map[string]any, key string) float64 {
	value, ok := metrics[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		return 0
	}
}

func stringFromMetrics(metrics map[string]any, key string) string {
	value, ok := metrics[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func boolFromMetrics(metrics map[string]any, key string) bool {
	value, ok := metrics[key].(bool)
	if !ok {
		return false
	}
	return value
}

func jsonObject(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func dateStrings(values []time.Time) []string {
	out := []string{}
	for _, value := range values {
		if !value.IsZero() {
			out = append(out, dayOnly(value).Format("2006-01-02"))
		}
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func stableFeatureID(prefix string, values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return prefix + "_" + hex.EncodeToString(sum[:])[:32]
}

func stableFeatureOffset(value string) int64 {
	sum := sha256.Sum256([]byte(value))
	return int64(binary.BigEndian.Uint64(sum[:8]) & ((1 << 63) - 1))
}
