package options

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/storage"
)

func ChainRecordFromMassiveSnapshot(tenantID string, sourceID string, ingestionRunID string, record massive.OptionContractDailyRecord) (storage.MarketOpsOptionsChainRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	symbol := strings.ToUpper(strings.TrimSpace(record.UnderlyingSymbol))
	optionTicker := strings.TrimSpace(record.OptionTicker)
	contractType := strings.ToLower(strings.TrimSpace(record.ContractType))
	if tenantID == "" {
		return storage.MarketOpsOptionsChainRecord{}, errors.New("tenant id is required")
	}
	if symbol == "" || optionTicker == "" || record.ObservationDate.IsZero() {
		return storage.MarketOpsOptionsChainRecord{}, errors.New("snapshot record symbol, option ticker, and observation date are required")
	}
	if contractType != "call" && contractType != "put" {
		return storage.MarketOpsOptionsChainRecord{}, errors.New("snapshot record contract type must be call or put")
	}
	if record.ExpirationDate.IsZero() || record.StrikePrice <= 0 {
		return storage.MarketOpsOptionsChainRecord{}, errors.New("snapshot record expiration date and strike price are required")
	}
	rawPayload, _ := json.Marshal(record.Raw)
	if len(rawPayload) == 0 || string(rawPayload) == "null" {
		rawPayload = []byte(`{}`)
	}
	var moneyness *float64
	if record.UnderlyingClose != nil && *record.UnderlyingClose > 0 {
		value := record.StrikePrice / *record.UnderlyingClose
		moneyness = &value
	}
	return storage.MarketOpsOptionsChainRecord{
		TenantID:          tenantID,
		Symbol:            symbol,
		TradeDate:         dayOnly(record.ObservationDate),
		OptionTicker:      optionTicker,
		Provider:          "massive",
		SourceID:          firstNonEmptyString(sourceID, "src-massive"),
		IngestionRunID:    strings.TrimSpace(ingestionRunID),
		ContractType:      contractType,
		ExpirationDate:    dayOnly(record.ExpirationDate),
		StrikePrice:       record.StrikePrice,
		UnderlyingClose:   record.UnderlyingClose,
		Moneyness:         moneyness,
		Open:              record.Open,
		High:              record.High,
		Low:               record.Low,
		Close:             record.Close,
		VWAP:              record.VWAP,
		Bid:               record.Bid,
		Ask:               record.Ask,
		QuoteTimestamp:    record.QuoteTimestamp,
		Volume:            record.Volume,
		OpenInterest:      record.OpenInterest,
		ImpliedVolatility: record.ImpliedVolatility,
		Delta:             record.Delta,
		Gamma:             record.Gamma,
		Theta:             record.Theta,
		Vega:              record.Vega,
		ExerciseStyle:     strings.ToLower(strings.TrimSpace(record.ExerciseStyle)),
		SharesPerContract: record.SharesPerContract,
		ProviderRequestID: strings.TrimSpace(record.ProviderRequestID),
		PayloadHash:       payloadHash(rawPayload),
		RawPayloadJSON:    rawPayload,
	}, nil
}

func LatestTradeDate(records []storage.MarketOpsOptionsChainRecord) time.Time {
	var latest time.Time
	for _, record := range records {
		date := dayOnly(record.TradeDate)
		if date.After(latest) {
			latest = date
		}
	}
	return latest
}

func payloadHash(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
