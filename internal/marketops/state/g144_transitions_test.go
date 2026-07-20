package state

import (
	"testing"
	"time"
)

func TestG144TransitionsIncludeConfiguredWindowsAndAcceleration(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-transitions", MaxSessions: 30}, BuildInput{EquityEvents: equityFixtures(start, 30)})
	if err != nil {
		t.Fatal(err)
	}
	foundFiveDay, foundAcceleration := false, false
	for _, transition := range result.Transitions {
		if transition.FeatureKey == "return_1d" && transition.TransitionType == "absolute_difference" && transition.LookbackSessions != nil && *transition.LookbackSessions == 5 {
			foundFiveDay = true
		}
		if transition.FeatureKey == "rv_20d" && transition.TransitionType == "acceleration" && transition.LookbackSessions != nil && *transition.LookbackSessions == 2 {
			foundAcceleration = true
		}
	}
	if !foundFiveDay || !foundAcceleration {
		t.Fatalf("missing G144 transitions: five_day=%v acceleration=%v", foundFiveDay, foundAcceleration)
	}
}

func TestG144TransitionsClassifyTermStructureRegimeChanges(t *testing.T) {
	firstSession := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	first := usableSurfaceFixtures(firstSession)
	second := usableSurfaceFixtures(firstSession.AddDate(0, 0, 1))
	for index := range second {
		switch second[index].OptionTicker {
		case "O:AAPL-ATM30C":
			*second[index].ImpliedVolatility = .40
		case "O:AAPL-ATM60C":
			*second[index].ImpliedVolatility = .35
		case "O:AAPL-ATM90P":
			*second[index].ImpliedVolatility = .30
		}
	}
	result, err := Build(BuildConfig{TenantID: "tenant-local", Symbol: "AAPL", RunID: "g144-regime"}, BuildInput{OptionChain: append(first, second...)})
	if err != nil {
		t.Fatal(err)
	}
	for _, transition := range result.Transitions {
		if transition.FeatureKey == "term_structure_state" && transition.TransitionType == "regime_transition" && transition.Direction == "backwardation" {
			return
		}
	}
	t.Fatal("expected contango-to-backwardation regime transition")
}
