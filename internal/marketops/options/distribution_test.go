package options

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestBuildDistributionUsesLatestDayOpenInterestAndBuckets(t *testing.T) {
	tradeDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	underlying := 100.0
	callOI := int64(300)
	putOI := int64(100)
	callVolume := int64(30)
	putVolume := int64(10)
	callIV, putIV, callDelta, putDelta := .30, .40, .50, -.25
	rows := []storage.MarketOpsOptionsChainRecord{
		row("call", tradeDate.AddDate(0, 0, -1), 105, underlying, 200, 100, 20, 10),
		row("put", tradeDate.AddDate(0, 0, -1), 95, underlying, 100, 100, 10, 10),
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "call", StrikePrice: 105, ExpirationDate: tradeDate.AddDate(0, 0, 14), UnderlyingClose: &underlying, OpenInterest: &callOI, Volume: &callVolume, ImpliedVolatility: &callIV, Delta: &callDelta},
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "put", StrikePrice: 95, ExpirationDate: tradeDate.AddDate(0, 0, 3), UnderlyingClose: &underlying, OpenInterest: &putOI, Volume: &putVolume, ImpliedVolatility: &putIV, Delta: &putDelta},
	}

	got := BuildDistribution("tenant-local", "nvda", tradeDate, rows)

	if got.Symbol != "NVDA" || got.TradeDays != 2 || got.ContractCount != 2 {
		t.Fatalf("distribution identity = %+v", got)
	}
	if got.TotalCallOpenInterest != 300 || got.TotalPutOpenInterest != 100 || got.CallPutOpenInterestRatio != 3 {
		t.Fatalf("open interest metrics = %+v", got)
	}
	if got.TotalCallVolume != 30 || got.TotalPutVolume != 10 || got.CallPutVolumeRatio != 3 {
		t.Fatalf("volume metrics = %+v", got)
	}
	var moneyness map[string]bucketTotals
	if err := json.Unmarshal(got.MoneynessDistributionJSON, &moneyness); err != nil {
		t.Fatalf("moneyness JSON error = %v", err)
	}
	if moneyness["100-105%"].CallOpenInterest != 300 || moneyness["95-100%"].PutOpenInterest != 100 {
		t.Fatalf("moneyness = %+v", moneyness)
	}
	var expiration map[string]bucketTotals
	if err := json.Unmarshal(got.ExpirationDistributionJSON, &expiration); err != nil {
		t.Fatalf("expiration JSON error = %v", err)
	}
	if expiration["8-30d"].CallOpenInterest != 300 || expiration["0-7d"].PutOpenInterest != 100 {
		t.Fatalf("expiration = %+v", expiration)
	}
	var metrics map[string]any
	if err := json.Unmarshal(got.MetricsJSON, &metrics); err != nil {
		t.Fatalf("metrics JSON error = %v", err)
	}
	if metrics["open_interest_quality"] != "usable" || metrics["open_interest_zero_rate"].(float64) != 0 || metrics["eligible_candidate_count"].(float64) != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
	deltaBuckets := metrics["delta_distribution"].(map[string]any)
	dteBuckets := metrics["dte_distribution"].(map[string]any)
	rejections := metrics["candidate_rejection_reasons"].(map[string]any)
	if deltaBuckets["0.40-0.60"] == nil || deltaBuckets["0.20-0.30"] == nil || dteBuckets["7-21d"] == nil || rejections["outside_surface_dte"].(float64) != 1 {
		t.Fatalf("bounded bucket metrics = %+v", metrics)
	}
}

func TestBuildDistributionFlagsAllZeroOpenInterestQuality(t *testing.T) {
	tradeDate := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	underlying := 100.0
	zero := int64(0)
	rows := []storage.MarketOpsOptionsChainRecord{
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "call", StrikePrice: 105, ExpirationDate: tradeDate.AddDate(0, 0, 14), UnderlyingClose: &underlying, OpenInterest: &zero},
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "put", StrikePrice: 95, ExpirationDate: tradeDate.AddDate(0, 0, 14), UnderlyingClose: &underlying, OpenInterest: &zero},
	}

	got := BuildDistribution("tenant-local", "NVDA", tradeDate, rows)

	var metrics map[string]any
	if err := json.Unmarshal(got.MetricsJSON, &metrics); err != nil {
		t.Fatalf("metrics JSON error = %v", err)
	}
	if metrics["open_interest_quality"] != "all_zero" || metrics["open_interest_zero_count"].(float64) != 2 || metrics["open_interest_zero_rate"].(float64) != 1 {
		t.Fatalf("metrics = %+v", metrics)
	}
	if metrics["call_put_oi_denominator_is_zero"] != true || metrics["call_put_oi_ratio_quality"] != "all_zero" {
		t.Fatalf("expected denominator zero/all-zero ratio flags, metrics = %+v", metrics)
	}
}

func TestBuildDistributionFlagsDenominatorZeroRatioQuality(t *testing.T) {
	tradeDate := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	callOI := int64(4437)
	putOI := int64(0)
	rows := []storage.MarketOpsOptionsChainRecord{
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "call", StrikePrice: 100, ExpirationDate: tradeDate.AddDate(0, 0, 14), OpenInterest: &callOI},
		{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: "put", StrikePrice: 100, ExpirationDate: tradeDate.AddDate(0, 0, 14), OpenInterest: &putOI},
	}

	got := BuildDistribution("tenant-local", "NVDA", tradeDate, rows)

	var metrics map[string]any
	if err := json.Unmarshal(got.MetricsJSON, &metrics); err != nil {
		t.Fatalf("metrics JSON error = %v", err)
	}
	if metrics["open_interest_quality"] != "partial_zero" || metrics["call_put_oi_ratio_quality"] != "denominator_zero" {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestBuildDistributionComputesDivergenceAgainstPriorWindow(t *testing.T) {
	tradeDate := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	rows := []storage.MarketOpsOptionsChainRecord{}
	for i, ratio := range []float64{1, 1, 1, 1, 4} {
		date := tradeDate.AddDate(0, 0, i-4)
		callOI := int64(ratio * 100)
		putOI := int64(100)
		rows = append(rows, storage.MarketOpsOptionsChainRecord{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: date, ContractType: "call", ExpirationDate: date.AddDate(0, 0, 30), StrikePrice: 100, OpenInterest: &callOI})
		rows = append(rows, storage.MarketOpsOptionsChainRecord{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: date, ContractType: "put", ExpirationDate: date.AddDate(0, 0, 30), StrikePrice: 100, OpenInterest: &putOI})
	}

	got := BuildDistribution("tenant-local", "NVDA", tradeDate, rows)

	if got.CallPutOpenInterestRatio != 4 || got.RatioDelta != 3 || got.RatioChangePct != 300 {
		t.Fatalf("divergence metrics = %+v", got)
	}
	if got.ChangePointScore != 0 || got.Confidence != 0 {
		t.Fatalf("flat prior window should not invent z-score, got %+v", got)
	}
}

func row(contractType string, tradeDate time.Time, strike float64, underlying float64, oi int64, _ int64, volume int64, _ int64) storage.MarketOpsOptionsChainRecord {
	return storage.MarketOpsOptionsChainRecord{TenantID: "tenant-local", Symbol: "NVDA", TradeDate: tradeDate, ContractType: contractType, StrikePrice: strike, ExpirationDate: tradeDate.AddDate(0, 0, 14), UnderlyingClose: &underlying, OpenInterest: &oi, Volume: &volume}
}
