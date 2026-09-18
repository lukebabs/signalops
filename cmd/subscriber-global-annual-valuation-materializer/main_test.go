package main

import (
	"testing"

	"github.com/lukebabs/signalops/internal/marketops/valuation"
)

func TestAppendableAnnualResultsSkipsIneligibleOutputs(t *testing.T) {
	results := []resultCandidate{
		{source: sourceCandidate{symbol: "AAPL"}, result: valuation.AnnualResult{Eligible: true}},
		{source: sourceCandidate{symbol: "NOPE"}, result: valuation.AnnualResult{Eligible: false}},
		{source: sourceCandidate{symbol: "NVDA"}, result: valuation.AnnualResult{Eligible: true}},
	}
	got := appendableAnnualResults(results)
	if len(got) != 2 {
		t.Fatalf("appendable results = %d, want 2", len(got))
	}
	if got[0].source.symbol != "AAPL" || got[1].source.symbol != "NVDA" {
		t.Fatalf("appendable result order/symbols = %#v", got)
	}
}
