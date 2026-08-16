package valuation

import (
	"math"
	"testing"
)

func TestEvaluateAnnualBalancedSixDimensionProfile(t *testing.T) {
	result := EvaluateAnnual(AnnualInput{
		Ticker: "ACME", Price: 100, MarketCap: 800, EnterpriseValue: 900,
		Revenue: 400, Revenue3YAgo: 250, NetIncome: 64, EBITDA: 100,
		OperatingIncome: 80, OperatingCashFlow: 92, CapitalExpenditures: 20,
		TotalDebt: 100, CashAndEquivalents: 40, ShareholdersEquity: 300,
		InvestedCapital: 400, CurrentAssets: 260, CurrentLiabilities: 140,
		Inventory: 40, InterestExpense: 10,
	})
	if !result.Eligible || result.Status != "complete" {
		t.Fatalf("expected complete annual profile: %#v", result)
	}
	if result.VCScore <= 0 || result.DOSMScore <= 0 || result.Components["available_dimensions"] != 6 {
		t.Fatalf("unexpected annual result: %#v", result)
	}
	if result.Components["valuation_weight"] != .40 || result.Components["growth_weight"] != .12 {
		t.Fatalf("incorrect approved weights: %#v", result.Components)
	}
	if result.Raw["revenue_cagr_3y_annual"] <= 0 || result.Raw["current_ratio_annual"] <= 1 {
		t.Fatalf("expected locally derived annual ratios: %#v", result.Raw)
	}
}

func TestEvaluateAnnualMissingDimensionReweightsRatherThanScoringZero(t *testing.T) {
	result := EvaluateAnnual(AnnualInput{
		MarketCap: 100, Revenue: 100, Revenue3YAgo: 80, NetIncome: 10,
		EBITDA: 15, OperatingIncome: 12, OperatingCashFlow: 18, CapitalExpenditures: 3,
		TotalDebt: 20, CashAndEquivalents: 5, ShareholdersEquity: 60,
		InvestedCapital: 80, // current balance fields intentionally absent
	})
	if !result.Eligible || result.Status != "complete_partial_annual" {
		t.Fatalf("expected eligible partial annual result: %#v", result)
	}
	if result.Components["available_dimension_weight"] >= 1 || result.DOSMScore == 0 {
		t.Fatalf("expected reweighted score: %#v", result.Components)
	}
}

func TestEvaluateAnnualRequiresAnnualValuationInputs(t *testing.T) {
	result := EvaluateAnnual(AnnualInput{Revenue: 100})
	if result.Eligible || result.Confidence != 0 || result.Status != "insufficient_data" {
		t.Fatalf("expected insufficient annual result: %#v", result)
	}
}

func TestEvaluateAnnualGrowthScoreCalibration(t *testing.T) {
	cases := []struct{ cagr, want float64 }{{0, 2}, {.05, 3}, {.10, 4}, {.15, 5}, {.25, 6}, {.35, 8}, {.50, 10}}
	for _, test := range cases {
		input := AnnualInput{Revenue: 100, Revenue3YAgo: 100 / math.Pow(1+test.cagr, 3)}
		score, ok := annualGrowth(input, &AnnualResult{Raw: map[string]float64{}, Components: map[string]float64{}})
		if !ok || math.Abs(score-test.want) > 0.000001 {
			t.Fatalf("cagr %.2f: got %.6f (ok=%v), want %.2f", test.cagr, score, ok, test.want)
		}
	}
}
