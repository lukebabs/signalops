// Package eeom implements deterministic pre-earnings setup prioritization.
package eeom

import "math"

const ModelVersion = "eeom-v1"

type Component struct {
	Score           float64 `json:"score"`
	Weight          float64 `json:"weight"`
	EffectiveWeight float64 `json:"effective_weight"`
	Available       bool    `json:"available"`
	Direction       string  `json:"direction,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}
type Input struct {
	DaysToEarnings                                                int
	Technical, Options, RiskReward, VC, DOSM, Materiality, Sector Component
}
type Result struct {
	Score           float64              `json:"score"`
	Posture         string               `json:"posture"`
	Classification  string               `json:"classification"`
	EvidenceQuality string               `json:"evidence_quality"`
	Eligible        bool                 `json:"eligible"`
	DaysToEarnings  int                  `json:"days_to_earnings"`
	Components      map[string]Component `json:"components"`
	Withheld        []string             `json:"withheld_inputs"`
}

func Evaluate(in Input) Result {
	components := map[string]Component{"technical": in.Technical, "options": in.Options, "risk_reward": in.RiskReward, "vc": in.VC, "dosm": in.DOSM, "event_materiality": in.Materiality, "sector": in.Sector}
	weights := weightsFor(in.DaysToEarnings)
	withheld := []string{}
	total := 0.0
	for k, c := range components {
		c.Weight = weights[k]
		c.Score = clamp(c.Score)
		components[k] = c
		if c.Available {
			total += c.Weight
		} else {
			withheld = append(withheld, k)
		}
	}
	core := components["technical"].Available && components["risk_reward"].Available
	r := Result{DaysToEarnings: in.DaysToEarnings, Components: components, Withheld: withheld, Eligible: core}
	if !core {
		r.EvidenceQuality = "withheld"
		r.Classification = "informational_only"
		r.Posture = "neutral"
		return r
	}
	score := 0.0
	for k, c := range components {
		if c.Available {
			c.EffectiveWeight = c.Weight * 100 / total
			components[k] = c
			score += c.Score * c.EffectiveWeight / 100
		}
	}
	r.Score = round(score)
	r.EvidenceQuality = "complete"
	if len(withheld) > 0 {
		r.EvidenceQuality = "partial"
	}
	r.Posture = posture(components)
	r.Classification = classification(r.Score, r.Posture, in.DaysToEarnings, r.EvidenceQuality)
	return r
}
func weightsFor(days int) map[string]float64 {
	if days >= 21 {
		return map[string]float64{"technical": 25, "options": 20, "risk_reward": 15, "vc": 15, "dosm": 10, "event_materiality": 10, "sector": 5}
	}
	if days >= 10 {
		return map[string]float64{"technical": 26, "options": 21, "risk_reward": 15, "vc": 13, "dosm": 10, "event_materiality": 10, "sector": 5}
	}
	if days >= 5 {
		return map[string]float64{"technical": 28, "options": 22, "risk_reward": 15, "vc": 10, "dosm": 10, "event_materiality": 10, "sector": 5}
	}
	return map[string]float64{"technical": 30, "options": 25, "risk_reward": 15, "vc": 10, "dosm": 10, "event_materiality": 5, "sector": 5}
}
func posture(c map[string]Component) string {
	up, down := 0.0, 0.0
	for _, k := range []string{"technical", "options", "risk_reward"} {
		x := c[k]
		if !x.Available {
			continue
		}
		if x.Direction == "bullish" {
			up += x.EffectiveWeight
		}
		if x.Direction == "bearish" {
			down += x.EffectiveWeight
		}
	}
	if up >= 20 && down >= 20 {
		return "mixed"
	}
	if up > down {
		return "bullish"
	}
	if down > up {
		return "bearish"
	}
	return "neutral"
}
func classification(score float64, posture string, days int, quality string) string {
	if quality == "withheld" {
		return "informational_only"
	}
	if score >= 80 && posture != "mixed" {
		return "priority_a"
	}
	if score >= 65 {
		return "priority_b"
	}
	if score < 35 {
		return "avoid"
	}
	if days <= 5 && posture == "mixed" {
		return "await_validation"
	}
	if score >= 55 {
		return "distressed_inflection"
	}
	return "informational_only"
}
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
