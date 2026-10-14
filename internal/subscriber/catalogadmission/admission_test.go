package catalogadmission

import (
	"testing"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
)

func TestEvaluateAdmitsOnlyActiveUSCommonStock(t *testing.T) {
	decision := Evaluate(massive.TickerDetails{Ticker: "AAPL", Name: "Apple Inc.", Locale: "us", Market: "stocks", Type: "cs", Exchange: "XNAS", Active: true})
	if decision.Decision != "eligible" {
		t.Fatalf("decision = %+v", decision)
	}
	for _, details := range []massive.TickerDetails{
		{Ticker: "SPY", Name: "ETF", Locale: "us", Market: "stocks", Type: "etf", Exchange: "ARCX", Active: true},
		{Ticker: "SHOP", Name: "Shopify", Locale: "ca", Market: "stocks", Type: "cs", Exchange: "XTSE", Active: true},
		{Ticker: "AAPL", Name: "Apple", Locale: "us", Market: "stocks", Type: "cs", Exchange: "XNAS", Active: false},
	} {
		if got := Evaluate(details); got.Decision != "ineligible" {
			t.Fatalf("accepted %+v", details)
		}
	}
}
