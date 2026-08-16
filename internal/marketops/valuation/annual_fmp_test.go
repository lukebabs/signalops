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
