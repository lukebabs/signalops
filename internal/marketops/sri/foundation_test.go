package sri

import (
	"fmt"
	"github.com/lukebabs/signalops/internal/storage"
	"testing"
	"time"
)

func series(symbol string, drift float64) []PricePoint {
	out := make([]PricePoint, 0, 61)
	for i := 0; i < 61; i++ {
		out = append(out, PricePoint{Session: time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC), Close: 100 + float64(i)*drift, EventID: fmt.Sprintf("%s-%d", symbol, i)})
	}
	return out
}
func TestScoreRanksUsablePriceLedSegments(t *testing.T) {
	segments := []storage.MarketOpsSRISegmentRecord{{SegmentID: "a", SegmentType: "sector"}, {SegmentID: "b", SegmentType: "sector"}}
	registry := []storage.MarketOpsSRIETFRecord{{SegmentID: "a", ETFSymbol: "AAA", Role: "primary"}, {SegmentID: "b", ETFSymbol: "BBB", Role: "primary"}}
	prices := map[string][]PricePoint{"SPY": series("SPY", .1), "QQQ": series("QQQ", .1), "RSP": series("RSP", .1), "AAA": series("AAA", 1), "BBB": series("BBB", .01)}
	out := Score(segments, registry, prices)
	if len(out) != 2 || out[0].Rank == nil || out[1].Rank == nil {
		t.Fatalf("missing ranks: %#v", out)
	}
	if *out[0].Rank != 1 || out[0].State != storage.MarketOpsSRIStateLeading {
		t.Fatalf("unexpected leading result: %#v", out[0])
	}
}
func TestScoreRequiresSixtyOneSessions(t *testing.T) {
	segments := []storage.MarketOpsSRISegmentRecord{{SegmentID: "a", SegmentType: "sector"}}
	registry := []storage.MarketOpsSRIETFRecord{{SegmentID: "a", ETFSymbol: "AAA", Role: "primary"}}
	prices := map[string][]PricePoint{"SPY": series("SPY", 1), "QQQ": series("QQQ", 1), "RSP": series("RSP", 1), "AAA": series("AAA", 1)[:60]}
	out := Score(segments, registry, prices)
	if out[0].QualityState != "partial" || out[0].Composite != nil {
		t.Fatalf("expected partial: %#v", out[0])
	}
}
