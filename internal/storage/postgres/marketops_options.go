package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const marketOpsOptionsChainSelect = `
SELECT tenant_id, symbol, trade_date, option_ticker, provider, source_id, ingestion_run_id,
 contract_type, expiration_date, strike_price, underlying_close, moneyness, open, high, low, close, vwap,
 volume, open_interest, implied_volatility, delta, gamma, theta, vega, provider_request_id, payload_hash,
 raw_payload, created_at, updated_at FROM marketops_options_chain_daily`

const marketOpsOptionsDistributionSelect = `
SELECT tenant_id, symbol, trade_date, window_name, source_id, provider, trade_days, contract_count,
 call_contract_count, put_contract_count, total_call_open_interest, total_put_open_interest,
 total_call_volume, total_put_volume, missing_open_interest_count, call_put_open_interest_ratio,
 call_put_volume_ratio, ratio_delta, ratio_change_pct, ratio_zscore, change_point_score, confidence,
 moneyness_distribution, expiration_distribution, metrics, COALESCE(array_to_json(source_trade_dates), '[]'::json)::text,
 created_at, updated_at FROM marketops_options_distribution_daily`

func (r *Repository) UpsertMarketOpsOptionsChain(ctx context.Context, record storage.MarketOpsOptionsChainRecord) error {
	if strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Symbol) == "" || strings.TrimSpace(record.OptionTicker) == "" || record.TradeDate.IsZero() {
		return fmt.Errorf("marketops options chain tenant_id, symbol, trade_date, and option_ticker are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_options_chain_daily (
 tenant_id, symbol, trade_date, option_ticker, provider, source_id, ingestion_run_id, contract_type,
 expiration_date, strike_price, underlying_close, moneyness, open, high, low, close, vwap, volume,
 open_interest, implied_volatility, delta, gamma, theta, vega, provider_request_id, payload_hash, raw_payload, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,now())
ON CONFLICT (tenant_id, symbol, trade_date, option_ticker) DO UPDATE SET
 provider=EXCLUDED.provider, source_id=EXCLUDED.source_id, ingestion_run_id=EXCLUDED.ingestion_run_id,
 contract_type=EXCLUDED.contract_type, expiration_date=EXCLUDED.expiration_date, strike_price=EXCLUDED.strike_price,
 underlying_close=EXCLUDED.underlying_close, moneyness=EXCLUDED.moneyness, open=EXCLUDED.open, high=EXCLUDED.high,
 low=EXCLUDED.low, close=EXCLUDED.close, vwap=EXCLUDED.vwap, volume=EXCLUDED.volume, open_interest=EXCLUDED.open_interest,
 implied_volatility=EXCLUDED.implied_volatility, delta=EXCLUDED.delta, gamma=EXCLUDED.gamma, theta=EXCLUDED.theta,
 vega=EXCLUDED.vega, provider_request_id=EXCLUDED.provider_request_id, payload_hash=EXCLUDED.payload_hash,
 raw_payload=EXCLUDED.raw_payload, updated_at=now()`,
		record.TenantID, strings.ToUpper(strings.TrimSpace(record.Symbol)), dayOnly(record.TradeDate), strings.TrimSpace(record.OptionTicker),
		firstNonEmptyString(record.Provider, "massive"), record.SourceID, record.IngestionRunID, strings.ToLower(strings.TrimSpace(record.ContractType)),
		dayOnly(record.ExpirationDate), record.StrikePrice, record.UnderlyingClose, record.Moneyness, record.Open, record.High, record.Low,
		record.Close, record.VWAP, record.Volume, record.OpenInterest, record.ImpliedVolatility, record.Delta, record.Gamma, record.Theta,
		record.Vega, record.ProviderRequestID, record.PayloadHash, jsonOrEmpty(record.RawPayloadJSON))
	if err != nil {
		return fmt.Errorf("upsert marketops options chain: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsOptionsChain(ctx context.Context, filter storage.MarketOpsOptionsChainFilter) ([]storage.MarketOpsOptionsChainRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsOptionsChainSelect+`
WHERE tenant_id=$1 AND upper(symbol)=upper($2)
 AND ($3::date IS NULL OR trade_date=$3)
 AND ($4::timestamptz IS NULL OR trade_date >= $4::date)
 AND ($5::timestamptz IS NULL OR trade_date < $5::date)
 AND ($6='' OR contract_type=$6)
ORDER BY trade_date DESC, expiration_date ASC, strike_price ASC, contract_type ASC
LIMIT $7`, strings.TrimSpace(filter.TenantID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), nullTime(filter.TradeDate), nullTime(filter.WindowStart), nullTime(filter.WindowEnd), strings.ToLower(strings.TrimSpace(filter.ContractType)), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops options chain: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsOptionsChainRecord{}
	for rows.Next() {
		record, err := scanMarketOpsOptionsChain(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops options chain rows: %w", err)
	}
	return records, nil
}

func (r *Repository) GetMarketOpsOptionsCoverage(ctx context.Context, tenantID string, symbol string) (storage.MarketOpsOptionsCoverageRecord, error) {
	var record storage.MarketOpsOptionsCoverageRecord
	err := r.db.QueryRowContext(ctx, `
SELECT tenant_id, symbol, count(DISTINCT trade_date), count(*), min(trade_date), max(trade_date), max(updated_at)
FROM marketops_options_chain_daily
WHERE tenant_id=$1 AND upper(symbol)=upper($2)
GROUP BY tenant_id, symbol`, strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(symbol))).Scan(&record.TenantID, &record.Symbol, &record.TradeDayCount, &record.ContractCount, &record.FirstTradeDate, &record.LastTradeDate, &record.LastUpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.MarketOpsOptionsCoverageRecord{}, storage.ErrNotFound
		}
		return storage.MarketOpsOptionsCoverageRecord{}, fmt.Errorf("get marketops options coverage: %w", err)
	}
	return record, nil
}

func (r *Repository) UpsertMarketOpsOptionsDistribution(ctx context.Context, record storage.MarketOpsOptionsDistributionRecord) error {
	if strings.TrimSpace(record.TenantID) == "" || strings.TrimSpace(record.Symbol) == "" || record.TradeDate.IsZero() {
		return fmt.Errorf("marketops options distribution tenant_id, symbol, and trade_date are required")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO marketops_options_distribution_daily (
 tenant_id, symbol, trade_date, window_name, source_id, provider, trade_days, contract_count,
 call_contract_count, put_contract_count, total_call_open_interest, total_put_open_interest,
 total_call_volume, total_put_volume, missing_open_interest_count, call_put_open_interest_ratio,
 call_put_volume_ratio, ratio_delta, ratio_change_pct, ratio_zscore, change_point_score, confidence,
 moneyness_distribution, expiration_distribution, metrics, source_trade_dates, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26::date[],now())
ON CONFLICT (tenant_id, symbol, trade_date, window_name) DO UPDATE SET
 source_id=EXCLUDED.source_id, provider=EXCLUDED.provider, trade_days=EXCLUDED.trade_days, contract_count=EXCLUDED.contract_count,
 call_contract_count=EXCLUDED.call_contract_count, put_contract_count=EXCLUDED.put_contract_count,
 total_call_open_interest=EXCLUDED.total_call_open_interest, total_put_open_interest=EXCLUDED.total_put_open_interest,
 total_call_volume=EXCLUDED.total_call_volume, total_put_volume=EXCLUDED.total_put_volume,
 missing_open_interest_count=EXCLUDED.missing_open_interest_count, call_put_open_interest_ratio=EXCLUDED.call_put_open_interest_ratio,
 call_put_volume_ratio=EXCLUDED.call_put_volume_ratio, ratio_delta=EXCLUDED.ratio_delta, ratio_change_pct=EXCLUDED.ratio_change_pct,
 ratio_zscore=EXCLUDED.ratio_zscore, change_point_score=EXCLUDED.change_point_score, confidence=EXCLUDED.confidence,
 moneyness_distribution=EXCLUDED.moneyness_distribution, expiration_distribution=EXCLUDED.expiration_distribution,
 metrics=EXCLUDED.metrics, source_trade_dates=EXCLUDED.source_trade_dates, updated_at=now()`,
		record.TenantID, strings.ToUpper(strings.TrimSpace(record.Symbol)), dayOnly(record.TradeDate), firstNonEmptyString(record.WindowName, "10_trade_days"),
		record.SourceID, firstNonEmptyString(record.Provider, "massive"), record.TradeDays, record.ContractCount, record.CallContractCount,
		record.PutContractCount, record.TotalCallOpenInterest, record.TotalPutOpenInterest, record.TotalCallVolume, record.TotalPutVolume,
		record.MissingOpenInterestCount, record.CallPutOpenInterestRatio, record.CallPutVolumeRatio, record.RatioDelta, record.RatioChangePct,
		record.RatioZScore, record.ChangePointScore, record.Confidence, jsonOrEmpty(record.MoneynessDistributionJSON),
		jsonOrEmpty(record.ExpirationDistributionJSON), jsonOrEmpty(record.MetricsJSON), dateArray(record.SourceTradeDates))
	if err != nil {
		return fmt.Errorf("upsert marketops options distribution: %w", err)
	}
	return nil
}

func (r *Repository) ListMarketOpsOptionsDistributions(ctx context.Context, filter storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error) {
	rows, err := r.db.QueryContext(ctx, marketOpsOptionsDistributionSelect+`
WHERE tenant_id=$1 AND upper(symbol)=upper($2) AND ($3='' OR window_name=$3)
ORDER BY trade_date DESC LIMIT $4`, strings.TrimSpace(filter.TenantID), strings.ToUpper(strings.TrimSpace(filter.Symbol)), strings.TrimSpace(filter.WindowName), clampLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list marketops options distributions: %w", err)
	}
	defer rows.Close()
	records := []storage.MarketOpsOptionsDistributionRecord{}
	for rows.Next() {
		record, err := scanMarketOpsOptionsDistribution(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list marketops options distributions rows: %w", err)
	}
	return records, nil
}

func scanMarketOpsOptionsChain(scanner interface{ Scan(...any) error }) (storage.MarketOpsOptionsChainRecord, error) {
	var record storage.MarketOpsOptionsChainRecord
	var underlyingClose, moneyness, open, high, low, closeValue, vwap, impliedVolatility, delta, gamma, theta, vega sql.NullFloat64
	var volume, openInterest sql.NullInt64
	if err := scanner.Scan(&record.TenantID, &record.Symbol, &record.TradeDate, &record.OptionTicker, &record.Provider, &record.SourceID, &record.IngestionRunID, &record.ContractType, &record.ExpirationDate, &record.StrikePrice, &underlyingClose, &moneyness, &open, &high, &low, &closeValue, &vwap, &volume, &openInterest, &impliedVolatility, &delta, &gamma, &theta, &vega, &record.ProviderRequestID, &record.PayloadHash, &record.RawPayloadJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsOptionsChainRecord{}, mapScanError("scan marketops options chain", err)
	}
	record.UnderlyingClose = nullableFloatPtr(underlyingClose)
	record.Moneyness = nullableFloatPtr(moneyness)
	record.Open = nullableFloatPtr(open)
	record.High = nullableFloatPtr(high)
	record.Low = nullableFloatPtr(low)
	record.Close = nullableFloatPtr(closeValue)
	record.VWAP = nullableFloatPtr(vwap)
	record.Volume = nullableIntPtr(volume)
	record.OpenInterest = nullableIntPtr(openInterest)
	record.ImpliedVolatility = nullableFloatPtr(impliedVolatility)
	record.Delta = nullableFloatPtr(delta)
	record.Gamma = nullableFloatPtr(gamma)
	record.Theta = nullableFloatPtr(theta)
	record.Vega = nullableFloatPtr(vega)
	return record, nil
}

func scanMarketOpsOptionsDistribution(scanner interface{ Scan(...any) error }) (storage.MarketOpsOptionsDistributionRecord, error) {
	var record storage.MarketOpsOptionsDistributionRecord
	var tradeDatesJSON string
	if err := scanner.Scan(&record.TenantID, &record.Symbol, &record.TradeDate, &record.WindowName, &record.SourceID, &record.Provider, &record.TradeDays, &record.ContractCount, &record.CallContractCount, &record.PutContractCount, &record.TotalCallOpenInterest, &record.TotalPutOpenInterest, &record.TotalCallVolume, &record.TotalPutVolume, &record.MissingOpenInterestCount, &record.CallPutOpenInterestRatio, &record.CallPutVolumeRatio, &record.RatioDelta, &record.RatioChangePct, &record.RatioZScore, &record.ChangePointScore, &record.Confidence, &record.MoneynessDistributionJSON, &record.ExpirationDistributionJSON, &record.MetricsJSON, &tradeDatesJSON, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return storage.MarketOpsOptionsDistributionRecord{}, mapScanError("scan marketops options distribution", err)
	}
	record.SourceTradeDates = parseDateJSONList(tradeDatesJSON)
	return record, nil
}

func nullableFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func nullableIntPtr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func dayOnly(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func dateArray(values []time.Time) stringArray {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !value.IsZero() {
			out = append(out, dayOnly(value).Format("2006-01-02"))
		}
	}
	return stringArray(out)
}

func parseDateJSONList(value string) []time.Time {
	raw := []string{}
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil
	}
	out := []time.Time{}
	for _, item := range raw {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(item))
		if err == nil {
			out = append(out, parsed.UTC())
		}
	}
	return out
}
