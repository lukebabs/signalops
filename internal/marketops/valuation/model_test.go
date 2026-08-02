package valuation

import "testing"

func floatPtr(v float64) *float64 { return &v }

func TestEvaluateWorkedPath(t *testing.T) {
	rsi, sma50, sma200 := 28.0, 104.0, 116.0
	growth := -0.01
	result := Evaluate(Input{Ticker: "TGT", Price: 100, MarketCap: 80, EnterpriseValue: 70, RevenueTTM: 100, NetIncomeGAAPTTM: 80.0 / 11, EBITDAProviderTTM: 10, OperatingIncomeTTM: 5.2, OperatingCashFlowTTM: 7, CapitalExpendituresTTM: 3.5, TotalDebt: 18.5, CashAndEquivalents: 3.5, ShareholdersEquity: 13.5, InvestedCapital: 28.5, RevenueCAGR3Y: &growth, RSI14: &rsi, SMA50: &sma50, SMA200: &sma200, PeerCount: 5, PeerPSMedian: floatPtr(.8), PeerPEMedian: floatPtr(11), PeerEVEBITDAMedian: floatPtr(7)})
	if !result.Eligible {
		t.Fatalf("expected eligible result: %#v", result)
	}
	if result.VCScore < 8 || result.DOSMScore < 6 {
		t.Fatalf("unexpected scores: %#v", result)
	}
	if result.VCFairValue <= 100 || result.DOSMFairValue <= 100 {
		t.Fatalf("expected positive fair-value anchors: %#v", result)
	}
}

func TestEvaluateTTMOnlyWithholdsGrowthAndPenalty(t *testing.T) {
	result := Evaluate(Input{Ticker: "TTM", Price: 100, MarketCap: 80, EnterpriseValue: 70, RevenueTTM: 100, NetIncomeGAAPTTM: 8, EBITDAProviderTTM: 10, OperatingIncomeTTM: 8, OperatingCashFlowTTM: 12, CapitalExpendituresTTM: 2, TotalDebt: 18, CashAndEquivalents: 3, ShareholdersEquity: 30, InvestedCapital: 45})
	if !result.Eligible || result.DataProfile != "ttm_only" || result.GrowthStatus == "" || result.Status != "complete_ttm_only" {
		t.Fatalf("expected eligible TTM-only result: %#v", result)
	}
	if result.Components["growth_valuation_penalty"] != 0 || result.Components["revenue_growth_score"] != 0 {
		t.Fatalf("growth must be withheld, got %#v", result.Components)
	}
}

func TestEvaluateRequiresFundamentals(t *testing.T) {
	result := Evaluate(Input{Price: 100, MarketCap: 100, RevenueTTM: 100})
	if result.Eligible || result.Status != "insufficient_data" || result.Confidence != 0 {
		t.Fatalf("expected insufficient input state: %#v", result)
	}
}
