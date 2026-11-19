package valuation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
)

func TestAnnualInputFromFMPLocallyDerivesMarketValues(t *testing.T) {
	periods := []fmp.AnnualFinancialPeriod{
		{PeriodEnd: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), AcceptedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			Income:   json.RawMessage(`{"revenue":100,"netIncome":20,"ebitda":30,"operatingIncome":25,"weightedAverageShsOutDil":10,"incomeBeforeTax":25,"incomeTaxExpense":5,"interestExpense":2}`),
			Balance:  json.RawMessage(`{"totalDebt":40,"cashAndCashEquivalents":10,"totalStockholdersEquity":80,"totalAssets":150,"totalCurrentAssets":70,"totalCurrentLiabilities":35,"inventory":5}`),
			CashFlow: json.RawMessage(`{"netCashProvidedByOperatingActivities":28,"capitalExpenditure":-6}`)},
		{}, {}, {Income: json.RawMessage(`{"revenue":50}`)},
	}
	in, available, ok := AnnualInputFromFMP(" acme ", periods, 12, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC))
	if !ok || in.Ticker != "ACME" || in.MarketCap != 120 || in.EnterpriseValue != 150 {
		t.Fatalf("unexpected annual input: %#v ok=%v", in, ok)
	}
	if in.Revenue3YAgo != 50 || in.EffectiveTaxRate == nil || *in.EffectiveTaxRate != .2 || available.IsZero() {
		t.Fatalf("expected raw-statement derivation: %#v available=%s", in, available)
	}
}

func TestAnnualInputFromFMPDecodesPersistedSnapshotShape(t *testing.T) {
	var payload struct {
		Periods []fmp.AnnualFinancialPeriod `json:"periods"`
	}
	raw := []byte(`{"periods":[{"FiscalYear":"2025","PeriodEnd":"2025-12-31T00:00:00Z","AcceptedAt":"2026-02-01T00:00:00Z","Income":{"revenue":216,"netIncome":24,"ebitda":36,"operatingIncome":30,"weightedAverageShsOutDil":10},"Balance":{"totalDebt":40,"cashAndCashEquivalents":8,"totalStockholdersEquity":90,"totalAssets":170,"totalCurrentAssets":80,"totalCurrentLiabilities":40},"CashFlow":{"netCashProvidedByOperatingActivities":32,"capitalExpenditure":-8}},{"FiscalYear":"2024","PeriodEnd":"2024-12-31T00:00:00Z","AcceptedAt":"2025-02-01T00:00:00Z","Income":{"revenue":180},"Balance":{"totalAssets":160},"CashFlow":{"netCashProvidedByOperatingActivities":28}},{"FiscalYear":"2023","PeriodEnd":"2023-12-31T00:00:00Z","AcceptedAt":"2024-02-01T00:00:00Z","Income":{"revenue":150},"Balance":{"totalAssets":150},"CashFlow":{"netCashProvidedByOperatingActivities":26}},{"FiscalYear":"2022","PeriodEnd":"2022-12-31T00:00:00Z","AcceptedAt":"2023-02-01T00:00:00Z","Income":{"revenue":100},"Balance":{"totalAssets":140},"CashFlow":{"netCashProvidedByOperatingActivities":24}}]}`)
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode persisted snapshot: %v", err)
	}
	in, _, ok := AnnualInputFromFMP("ACME", payload.Periods, 12, time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	if !ok || in.Revenue != 216 || in.Revenue3YAgo != 100 || in.MarketCap != 120 {
		t.Fatalf("expected persisted snapshot to derive annual input: %#v ok=%v", in, ok)
	}
	result := EvaluateAnnual(in)
	if !result.Eligible || result.Raw["revenue_cagr_3y_annual"] <= 0 || result.Components["revenue_growth_score"] <= 0 {
		t.Fatalf("expected CAGR-backed annual result: %#v", result)
	}
}
