package main

import (
	"context"
	"math"
	"strings"
	"testing"
)

func TestSectorBenchmark(t *testing.T) {
	cases := []struct{ raw, segment, symbol string }{
		{"Services-Prepackaged Software", "sector_technology", "XLK"},
		{"Financial Services", "sector_financials", "XLF"},
		{"Aerospace & Defense", "sector_industrials", "XLI"},
		{"", "", ""},
	}
	for _, item := range cases {
		segment, symbol := sectorBenchmark(item.raw)
		if segment != item.segment || symbol != item.symbol {
			t.Fatalf("sectorBenchmark(%q) = (%q,%q), want (%q,%q)", item.raw, segment, symbol, item.segment, item.symbol)
		}
	}
}

func TestBenchmarkChoicesLeavesUnknownSectorExplicit(t *testing.T) {
	items := benchmarkChoices(observation{sector: "", symbol: "ABC"})
	if len(items) != 2 || items[0].symbol != "SPY" || items[0].state != "matched" || items[1].state != "sector_unmapped" {
		t.Fatalf("unexpected benchmark choices: %#v", items)
	}
}

func TestDirectionalRelativeReturnAlignsBearishAndBullishEvidence(t *testing.T) {
	if got := directionalRelativeReturn("upside", .05, .02); math.Abs(got-.03) > 1e-12 {
		t.Fatalf("upside relative=%v, want .03", got)
	}
	if got := directionalRelativeReturn("downside", -.05, -.02); math.Abs(got-.03) > 1e-12 {
		t.Fatalf("downside relative=%v, want .03", got)
	}
}

func TestRunRejectsUnsafeCalculationVersionBeforeDatabaseAccess(t *testing.T) {
	err := run(context.Background(), []string{
		"--dry-run",
		"--database-url", "postgres://example.invalid/marketops",
		"--temporal-database-url", "postgres://example.invalid/marketops_temporal",
		"--calculation-version", "saf_benchmark.v2;drop",
	})
	if err == nil || !strings.Contains(err.Error(), "calculation-version") {
		t.Fatalf("unsafe calculation version error=%v", err)
	}
}
