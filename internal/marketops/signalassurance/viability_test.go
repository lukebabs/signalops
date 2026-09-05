package signalassurance

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestAssessViabilityRequiresMinimumMaturedSample(t *testing.T) {
	result := AssessViability(storage.SignalAssuranceEffectivenessRecord{SampleSize: 29})
	if result.State != ViabilityInsufficientEvidence {
		t.Fatalf("state = %q", result.State)
	}
}

func TestAssessViabilityRequiresBenchmarkEvidence(t *testing.T) {
	result := AssessViability(storage.SignalAssuranceEffectivenessRecord{
		SampleSize: 30, DirectionalAccuracy: floatPtr(.7), AccuracyLowerBound: floatPtr(.52),
		AverageMFE: floatPtr(.08), AverageMAE: floatPtr(-.03),
	})
	if result.State != ViabilityBenchmarkPending {
		t.Fatalf("state = %q", result.State)
	}
}

func TestAssessViabilityRequiresCompleteSectorBenchmarkCoverage(t *testing.T) {
	result := AssessViability(storage.SignalAssuranceEffectivenessRecord{
		SampleSize: 30, DirectionalAccuracy: floatPtr(.7), AccuracyLowerBound: floatPtr(.52),
		AverageRelativeReturn: floatPtr(.02), BroadMarketBenchmarkSampleSize: 30, SectorBenchmarkSampleSize: 29,
		AverageMFE: floatPtr(.08), AverageMAE: floatPtr(-.03),
	})
	if result.State != ViabilityBenchmarkPending {
		t.Fatalf("state = %q", result.State)
	}
}

func TestAssessViabilityDoesNotOverstateInSampleEvidence(t *testing.T) {
	result := AssessViability(storage.SignalAssuranceEffectivenessRecord{
		SampleSize: 30, DirectionalAccuracy: floatPtr(.7), AccuracyLowerBound: floatPtr(.52),
		AverageRelativeReturn: floatPtr(.02), BroadMarketBenchmarkSampleSize: 30, SectorBenchmarkSampleSize: 30, AverageMFE: floatPtr(.08), AverageMAE: floatPtr(-.03),
	})
	if result.State != ViabilityResearchSupportedInSample {
		t.Fatalf("state = %q", result.State)
	}
	if len(result.Reasons) != 2 {
		t.Fatalf("reasons = %#v", result.Reasons)
	}
}
