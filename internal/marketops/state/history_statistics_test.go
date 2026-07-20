package state

import (
	"math"
	"testing"
)

func TestTrailingStatisticsRequirePriorHistory(t *testing.T) {
	prior := make([]float64, 20)
	for index := range prior {
		prior[index] = float64(index % 5)
	}
	if got := trailingZScore(10, prior[:19]); got != nil {
		t.Fatalf("z-score before minimum history = %v", *got)
	}
	if got := trailingPercentile(10, prior[:19]); got != nil {
		t.Fatalf("percentile before minimum history = %v", *got)
	}
	zscore := trailingZScore(10, prior)
	percentile := trailingPercentile(10, prior)
	if zscore == nil || *zscore <= 2 {
		t.Fatalf("z-score = %v", zscore)
	}
	if percentile == nil || *percentile != 1 {
		t.Fatalf("percentile = %v", percentile)
	}
}

func TestTrailingStatisticsUseOnlyLastSixtyPriorValues(t *testing.T) {
	prior := make([]float64, 80)
	for index := range prior {
		if index < 20 {
			prior[index] = 1000
		} else {
			prior[index] = float64(index % 3)
		}
	}
	got := trailingZScore(10, prior)
	if got == nil || math.Abs(*got) > 20 {
		t.Fatalf("expected old outliers excluded, z-score = %v", got)
	}
}

func TestTrailingPersistence(t *testing.T) {
	if got := trailingPersistence("up", []string{"down", "up", "up"}); got != 3 {
		t.Fatalf("up persistence = %d", got)
	}
	if got := trailingPersistence("down", []string{"up", "up"}); got != 1 {
		t.Fatalf("direction change persistence = %d", got)
	}
	if got := trailingPersistence("flat", []string{"flat", "flat"}); got != 1 {
		t.Fatalf("flat persistence = %d", got)
	}
}
