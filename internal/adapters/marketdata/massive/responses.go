package massive

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type optionContractsResponse struct {
	Results []optionContractResult `json:"results"`
}

type optionContractResult struct {
	Ticker           string          `json:"ticker"`
	UnderlyingTicker string          `json:"underlying_ticker"`
	ContractType     string          `json:"contract_type"`
	ExpirationDate   string          `json:"expiration_date"`
	StrikePrice      json.RawMessage `json:"strike_price"`
	Raw              map[string]any  `json:"-"`
}

func (r *optionContractResult) UnmarshalJSON(value []byte) error {
	type alias optionContractResult
	var decoded alias
	if err := json.Unmarshal(value, &decoded); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(value, &raw); err != nil {
		return err
	}
	*r = optionContractResult(decoded)
	r.Raw = raw
	return nil
}

func (r optionContractsResponse) records(asOf time.Time) []OptionContractDailyRecord {
	records := make([]OptionContractDailyRecord, 0, len(r.Results))
	for _, result := range r.Results {
		expiration, err := parseDate(result.ExpirationDate)
		if err != nil {
			continue
		}
		records = append(records, OptionContractDailyRecord{
			ProviderContractID: stringValue(result.Raw, "id", "ticker"),
			OptionTicker:       result.Ticker,
			UnderlyingSymbol:   result.UnderlyingTicker,
			ContractType:       result.ContractType,
			ExpirationDate:     expiration,
			StrikePrice:        numberFromRaw(result.StrikePrice),
			ObservationDate:    asOf,
			Raw:                result.Raw,
		})
	}
	return records
}

type aggregateBarsResponse struct {
	Ticker  string               `json:"ticker"`
	Results []aggregateBarResult `json:"results"`
}

type aggregateBarResult struct {
	Open      *float64       `json:"o"`
	High      *float64       `json:"h"`
	Low       *float64       `json:"l"`
	Close     *float64       `json:"c"`
	Volume    *float64       `json:"v"`
	VWAP      *float64       `json:"vw"`
	Timestamp *int64         `json:"t"`
	Raw       map[string]any `json:"-"`
}

func (r *aggregateBarResult) UnmarshalJSON(value []byte) error {
	type alias aggregateBarResult
	var decoded alias
	if err := json.Unmarshal(value, &decoded); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(value, &raw); err != nil {
		return err
	}
	*r = aggregateBarResult(decoded)
	r.Raw = raw
	return nil
}

func (r aggregateBarsResponse) equityRecords(symbol string, fallbackDate time.Time) []EquityEODPriceRecord {
	records := make([]EquityEODPriceRecord, 0, len(r.Results))
	for _, result := range r.Results {
		observation := aggregateObservationDate(result.Timestamp, fallbackDate)
		records = append(records, EquityEODPriceRecord{
			ProviderEventID: aggregateProviderID(symbol, observation),
			Symbol:          symbol,
			ObservationDate: observation,
			Open:            result.Open,
			High:            result.High,
			Low:             result.Low,
			Close:           result.Close,
			Volume:          intFromFloatPtr(result.Volume),
			VWAP:            result.VWAP,
			Raw:             result.Raw,
		})
	}
	return records
}

func (r aggregateBarsResponse) optionRecords(optionTicker string, underlying string, fallbackDate time.Time) []OptionContractDailyRecord {
	records := make([]OptionContractDailyRecord, 0, len(r.Results))
	for _, result := range r.Results {
		observation := aggregateObservationDate(result.Timestamp, fallbackDate)
		records = append(records, OptionContractDailyRecord{
			ProviderContractID: optionTicker,
			OptionTicker:       optionTicker,
			UnderlyingSymbol:   underlying,
			ObservationDate:    observation,
			Open:               result.Open,
			High:               result.High,
			Low:                result.Low,
			Close:              result.Close,
			Volume:             intFromFloatPtr(result.Volume),
			VWAP:               result.VWAP,
			Raw:                result.Raw,
		})
	}
	return records
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func aggregateObservationDate(timestamp *int64, fallback time.Time) time.Time {
	if timestamp == nil || *timestamp <= 0 {
		day, _ := dayUTC(fallback, "fallback_date")
		return day
	}
	return time.UnixMilli(*timestamp).UTC()
}

func aggregateProviderID(symbol string, observation time.Time) string {
	return strings.ToLower(strings.TrimSpace(symbol)) + ":" + dateKey(observation)
}

func intFromFloatPtr(value *float64) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func numberFromRaw(value json.RawMessage) float64 {
	if len(value) == 0 {
		return 0
	}
	var number float64
	if err := json.Unmarshal(value, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(text), 64)
		return parsed
	}
	return 0
}

func stringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
