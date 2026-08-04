package eeom

import "testing"

func TestEvaluateReweightsAndSeparatesPosture(t *testing.T) {
	r := Evaluate(Input{DaysToEarnings: 4, Technical: Component{Score: 80, Available: true, Direction: "bullish"}, RiskReward: Component{Score: 70, Available: true, Direction: "bullish"}, VC: Component{Score: 60, Available: true}, DOSM: Component{Score: 60, Available: true}, Materiality: Component{Score: 100, Available: true}, Sector: Component{Score: 50, Available: true}})
	if !r.Eligible || r.Posture != "bullish" || r.EvidenceQuality != "partial" || r.Score <= 0 {
		t.Fatalf("unexpected %+v", r)
	}
	total := 0.0
	for _, x := range r.Components {
		total += x.EffectiveWeight
	}
	if total != 100 {
		t.Fatalf("effective weights %v", total)
	}
}
func TestEvaluateWithholdsWithoutCoreEvidence(t *testing.T) {
	r := Evaluate(Input{DaysToEarnings: 8, Technical: Component{Score: 80, Available: true}})
	if r.Eligible || r.EvidenceQuality != "withheld" {
		t.Fatalf("unexpected %+v", r)
	}
}
