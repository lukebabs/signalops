package hypotheses

import (
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestG144KnownEarningsWindowInvalidatesH001(t *testing.T) {
	observations := []storage.MarketOpsFeatureObservationRecord{
		feature("rsi_14", 78, nil),
		feature("surface_coverage_ratio", .95, nil),
		feature("iv", .36, dims("put", 30, .25)),
		feature("iv", .39, dims("put", 60, .25)),
		feature("extrinsic_premium", 4.2, dims("put", 30, .25)),
		feature("oi_change_1d", 250, dims("put", 30, .25)),
	}
	window := "pre_earnings"
	observations = append(observations, storage.MarketOpsFeatureObservationRecord{
		FeatureObservationID: "feature-earnings-window", FeatureKey: "earnings_window_state", FeatureVersion: "v1",
		DimensionsJSON: []byte("{}"), TextValue: &window, QualityState: storage.MarketOpsQualityUsable,
	})
	transitions := []storage.MarketOpsStateTransitionRecord{
		transition("iv", .03, dims("put", 30, .25), 2, nil, nil),
		transition("iv", .04, dims("put", 60, .25), 2, nil, nil),
		transition("extrinsic_premium", .4, dims("put", 30, .25), 2, nil, nil),
		transition("oi_change_1d", 250, dims("put", 30, .25), 2, nil, nil),
	}
	result, err := Evaluate("g144-event-invalidation", definitionByKey(ResearchDefinitions("tenant-local"), "H001"), testState(), observations, transitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Eligible || result.Triggered || !result.Invalidated || !contains(result.ReasonCodes, "known_earnings_window") {
		t.Fatalf("known event did not invalidate H001: %+v", result)
	}
}

func TestG144MissingEarningsContextDoesNotBlockH001(t *testing.T) {
	observations := []storage.MarketOpsFeatureObservationRecord{
		feature("rsi_14", 78, nil),
		feature("surface_coverage_ratio", .95, nil),
		feature("iv", .36, dims("put", 30, .25)),
		feature("iv", .39, dims("put", 60, .25)),
		feature("extrinsic_premium", 4.2, dims("put", 30, .25)),
		feature("oi_change_1d", 250, dims("put", 30, .25)),
		{FeatureObservationID: "feature-earnings-missing", FeatureKey: "earnings_window_state", FeatureVersion: "v1", DimensionsJSON: []byte("{}"), QualityState: storage.MarketOpsQualityMissing},
	}
	transitions := []storage.MarketOpsStateTransitionRecord{
		transition("iv", .03, dims("put", 30, .25), 2, nil, nil),
		transition("iv", .04, dims("put", 60, .25), 2, nil, nil),
		transition("extrinsic_premium", .4, dims("put", 30, .25), 2, nil, nil),
		transition("oi_change_1d", 250, dims("put", 30, .25), 2, nil, nil),
	}
	result, err := Evaluate("g144-event-optional", definitionByKey(ResearchDefinitions("tenant-local"), "H001"), testState(), observations, transitions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Eligible || result.Invalidated {
		t.Fatalf("missing optional event context blocked H001: %+v", result)
	}
}
