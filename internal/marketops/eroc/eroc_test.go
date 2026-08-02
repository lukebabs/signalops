package eroc

import "testing"

func descending(start float64, count int) []float64 { out := make([]float64, count); for i := range out { out[i] = start-float64(i) }; return out }
func ascending(start float64, count int) []float64 { out := make([]float64, count); for i := range out { out[i] = start+float64(i) }; return out }
func volumes(base float64, count int) []float64 { out := make([]float64, count); for i := range out { out[i] = base }; return out }

func TestFadingDriftCompleteCandidate(t *testing.T) {
	v := volumes(100, 21)
	for i := 16; i < len(v); i++ { v[i] = 60 }
	got := Evaluate(Input{Closes: descending(130, 21), Volumes: v, CallVolume: 150, PutVolume: 100})
	if got.Regime != RegimeFadingDrift || !got.Candidate || !got.EvidenceComplete || got.State != StateConfirmed || got.Direction != DirectionBullish { t.Fatalf("unexpected result: %+v", got) }
}

func TestTrendSupportedIsNotReversalCandidate(t *testing.T) {
	v := volumes(100, 21)
	for i := 16; i < len(v); i++ { v[i] = 110 }
	got := Evaluate(Input{Closes: ascending(100, 21), Volumes: v, CallVolume: 150, PutVolume: 100})
	if got.Regime != RegimeTrendSupported || got.Candidate || got.Score != 0 || got.Outcome != "trend_supported_not_ranked" { t.Fatalf("unexpected result: %+v", got) }
}

func TestClimacticCandidateWithoutReversalFlowIsIncomplete(t *testing.T) {
	v := volumes(100, 21); v[len(v)-1] = 200
	got := Evaluate(Input{Closes: ascending(100, 21), Volumes: v, CallVolume: 100, PutVolume: 100})
	if got.Regime != RegimeClimacticExtension || !got.Candidate || got.EvidenceComplete || got.State != StateObserved { t.Fatalf("unexpected result: %+v", got) }
}

func TestInsufficientHistoryUnavailable(t *testing.T) {
	got := Evaluate(Input{Closes: descending(120, 20), Volumes: volumes(100, 20), CallVolume: 150, PutVolume: 100})
	if got.State != StateUnavailable || got.Ready { t.Fatalf("unexpected result: %+v", got) }
}
