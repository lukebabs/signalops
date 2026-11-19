package postgres

import (
	"github.com/lukebabs/signalops/internal/storage"
	"testing"
)

func TestAggregateEffectivenessSeparatesCoverageAndAccuracy(t *testing.T) {
	values := make([]effectivenessObservation, 0, 32)
	for i := 0; i < 30; i++ {
		value := -.01
		if i < 15 {
			value = .01
		}
		values = append(values, effectivenessObservation{source: "SAF", algorithm: "algo", version: "v1", signalType: "directional", direction: "bullish", state: storage.SignalAssertionMaterialized, absoluteReturn: &value, complete: true, horizon: 5})
	}
	values = append(values, effectivenessObservation{source: "SAF", algorithm: "algo", version: "v1", signalType: "directional", direction: "bullish", state: storage.SignalAssertionActive})
	values = append(values, effectivenessObservation{source: "SAF", algorithm: "algo", version: "v1", signalType: "directional", direction: "bullish", state: storage.SignalAssertionExpired, complete: false})
	rows := aggregateEffectiveness(values, "overall")
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	row := rows[0]
	if row.SampleSize != 30 || row.DirectionalHits != 15 || row.CensoredCount != 1 || row.ExcludedCount != 1 {
		t.Fatalf("unexpected aggregate: %#v", row)
	}
	if row.Exploratory {
		t.Fatal("30 matured observations must be rankable")
	}
	if row.DirectionalAccuracy == nil || *row.DirectionalAccuracy != .5 {
		t.Fatalf("accuracy=%v", row.DirectionalAccuracy)
	}
	if row.AccuracyLowerBound == nil || row.AccuracyUpperBound == nil {
		t.Fatal("expected Wilson interval")
	}
}

func TestBenchmarkCoverageDimensionSeparatesUnmappedSectorEvidence(t *testing.T) {
	matched := effectivenessObservation{source: "LEGACY", broadMarketBenchmarkState: "matched", sectorBenchmarkState: "matched"}
	unmapped := effectivenessObservation{source: "LEGACY", broadMarketBenchmarkState: "matched", sectorBenchmarkState: "sector_unmapped"}
	if got := dimensionValue(matched, "benchmark_coverage"); got != "broad=matched; sector=matched" {
		t.Fatalf("matched coverage=%q", got)
	}
	if got := dimensionValue(unmapped, "benchmark_coverage"); got != "broad=matched; sector=sector_unmapped" {
		t.Fatalf("unmapped coverage=%q", got)
	}
}
