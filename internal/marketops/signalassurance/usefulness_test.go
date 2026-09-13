package signalassurance

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestOneDayAdverseBullishObservationIsDevelopingNotMiss(t *testing.T) {
	down := -0.01
	mfe := 0.0
	mae := -0.01
	hit := false
	result := AssessUsefulnessObservation(UsefulnessObservationInput{Direction: "upside", HorizonSessions: 1, DirectionalHit: &hit, DirectionalReturn: &down, MFE: &mfe, MAE: &mae, RelativeReturn: floatPtr(-0.005), SectorRelativeReturn: floatPtr(-0.004)})
	if result.LifecycleState != LifecycleDeveloping {
		t.Fatalf("lifecycle = %q, want %q", result.LifecycleState, LifecycleDeveloping)
	}
	if result.Score == nil {
		t.Fatalf("expected usefulness score")
	}
}

func TestOneDayLargeAdverseMoveIsWarningNotAutomaticInvalidation(t *testing.T) {
	down := -0.04
	mfe := 0.0
	mae := -0.04
	hit := false
	result := AssessUsefulnessObservation(UsefulnessObservationInput{Direction: "upside", HorizonSessions: 1, DirectionalHit: &hit, DirectionalReturn: &down, MFE: &mfe, MAE: &mae, RelativeReturn: floatPtr(-0.03), SectorRelativeReturn: floatPtr(-0.02)})
	if result.LifecycleState != LifecycleAdverseWarning {
		t.Fatalf("lifecycle = %q, want %q", result.LifecycleState, LifecycleAdverseWarning)
	}
}

func TestMatchedObservationCanOutperform(t *testing.T) {
	up := 0.06
	mfe := 0.08
	mae := -0.01
	relative := 0.03
	sector := 0.02
	hit := true
	result := AssessUsefulnessObservation(UsefulnessObservationInput{Direction: "upside", HorizonSessions: 5, DirectionalHit: &hit, DirectionalReturn: &up, MFE: &mfe, MAE: &mae, RelativeReturn: &relative, SectorRelativeReturn: &sector})
	if result.LifecycleState != LifecycleOutperformed {
		t.Fatalf("lifecycle = %q, want %q", result.LifecycleState, LifecycleOutperformed)
	}
	if result.TimeToMaterializationSessions == nil || *result.TimeToMaterializationSessions != 5 {
		t.Fatalf("time to materialization = %#v", result.TimeToMaterializationSessions)
	}
	if result.Score == nil || *result.Score < 8 {
		t.Fatalf("score = %#v, want strong usefulness", result.Score)
	}
}

func TestEffectivenessUsefulnessRequiresBenchmarkCoverage(t *testing.T) {
	result := AssessUsefulnessEffectiveness(storage.SignalAssuranceEffectivenessRecord{SampleSize: 30, DirectionalHits: 20, DirectionalAccuracy: floatPtr(.66), AverageMFE: floatPtr(.04), AverageMAE: floatPtr(-.01)})
	if result.Score != nil {
		t.Fatalf("score = %#v, want nil until benchmark evidence exists", result.Score)
	}
}
