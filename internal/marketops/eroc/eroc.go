package eroc

import "math"

type Direction string
type Regime string

const (
	DirectionNone    Direction = "NONE"
	DirectionBullish Direction = "BULLISH"
	DirectionBearish Direction = "BEARISH"

	RegimeTrendSupported     Regime = "trend_supported"
	RegimeFadingDrift        Regime = "fading_drift"
	RegimeClimacticExtension Regime = "climactic_extension"
	RegimeUnresolved         Regime = "unresolved"
)

type State string

const (
	StateUnavailable State = "UNAVAILABLE"
	StateObserved    State = "OBSERVED"
	StateConfirmed   State = "CONFIRMED"
)

const ModelVersion = "eroc-v6.1"

type Input struct {
	Closes     []float64
	Volumes    []float64
	CallVolume int64
	PutVolume  int64
	IVRegime   string
}

type Result struct {
	Direction            Direction `json:"direction"`
	DriftDirection       Direction `json:"drift_direction"`
	Regime               Regime    `json:"regime"`
	Candidate            bool      `json:"reversal_candidate"`
	EvidenceComplete     bool      `json:"evidence_complete"`
	State                State     `json:"state"`
	Score                float64   `json:"score"`
	StanceScore          float64   `json:"stance_score"`
	Outcome              string    `json:"outcome"`
	Tier                 string    `json:"proximity_tier"`
	Ready                bool      `json:"ready"`
	PriceScore           float64   `json:"price_score"`
	FlowScore            float64   `json:"flow_score"`
	VolumeScore          float64   `json:"volume_score"`
	PersistenceScore     float64   `json:"persistence_score"`
	ExtensionScore       float64   `json:"extension_score"`
	PriceExtensionPct    float64   `json:"price_extension_pct"`
	PriceExtensionUnits  float64   `json:"price_extension_units"`
	RelativeVolume20d    float64   `json:"relative_volume_20d"`
	Consecutive          int       `json:"consecutive_directional_closes"`
	DirectionalDays      int       `json:"directional_days"`
	DirectionalWindow    int       `json:"directional_window"`
	CurrentAverageVolume float64   `json:"current_5d_average_volume"`
	PriorAverageVolume   float64   `json:"prior_5d_average_volume"`
	VolumeRatio          float64   `json:"volume_ratio"`
	DirectionalFlowRatio *float64  `json:"directional_flow_ratio,omitempty"`
	ReversalFlowRatio    *float64  `json:"reversal_flow_ratio,omitempty"`
	TrendFlowRatio       *float64  `json:"trend_flow_ratio,omitempty"`
	PutCallVolumeRatio   *float64  `json:"put_call_volume_ratio,omitempty"`
	TotalOptionVolume    int64     `json:"total_option_volume"`
	OptionsFlowExtreme   string    `json:"options_flow_extreme,omitempty"`
	IVRegime             string    `json:"iv_regime,omitempty"`
	IVAdjustment         float64   `json:"iv_adjustment,omitempty"`
	PriceEligible        bool      `json:"price_eligible"`
	FlowEligible         bool      `json:"flow_eligible"`
	VolumeEligible       bool      `json:"volume_eligible"`
	Reasons              []string  `json:"reasons"`
}

func Evaluate(in Input) Result {
	r := Result{Direction: DirectionNone, DriftDirection: DirectionNone, Regime: RegimeUnresolved, State: StateUnavailable}
	if len(in.Closes) < 21 || len(in.Volumes) < 21 || len(in.Closes) != len(in.Volumes) {
		r.Reasons = []string{"insufficient_completed_eod_history"}
		return r
	}
	r.Ready = true
	r.IVRegime = in.IVRegime
	recentCloses := in.Closes[len(in.Closes)-10:]
	recentVolumes := in.Volumes[len(in.Volumes)-10:]
	r.DriftDirection = driftDirection(recentCloses)
	r.Direction = r.DriftDirection
	if r.Direction == DirectionNone {
		r.State = StateObserved
		r.Tier = "UNRESOLVED"
		r.Outcome = "unresolved_not_ranked"
		r.Reasons = []string{"ten_day_drift_flat"}
		return r
	}
	down := r.Direction == DirectionBullish
	r.Consecutive = consecutive(recentCloses, down)
	r.DirectionalDays, r.DirectionalWindow = directionalWindow(recentCloses, down)
	r.PersistenceScore = persistenceScore(r.Consecutive, directionalRate(recentCloses, down))
	persistenceEligible := r.Consecutive >= 4 || (r.DirectionalWindow > 0 && r.DirectionalDays*5 >= r.DirectionalWindow*4)
	r.PriceExtensionPct = extensionPercent(recentCloses, down)
	averageMove := meanAbsoluteDailyMove(in.Closes[len(in.Closes)-21:])
	if averageMove > 0 {
		r.PriceExtensionUnits = r.PriceExtensionPct / averageMove
	}
	r.ExtensionScore = extensionScore(r.PriceExtensionUnits)
	extensionEligible := r.PriceExtensionUnits >= 3
	r.PriceEligible = persistenceEligible && extensionEligible
	r.PriceScore = round(.4*r.PersistenceScore + .6*r.ExtensionScore)
	if !persistenceEligible {
		r.Reasons = append(r.Reasons, "price_persistence_not_met")
	}
	if !extensionEligible {
		r.Reasons = append(r.Reasons, "price_extension_not_met")
	}

	r.CurrentAverageVolume = mean(recentVolumes[5:])
	r.PriorAverageVolume = mean(recentVolumes[:5])
	if r.PriorAverageVolume > 0 {
		r.VolumeRatio = r.CurrentAverageVolume / r.PriorAverageVolume
	}
	priorTwentyVolume := mean(in.Volumes[len(in.Volumes)-21 : len(in.Volumes)-1])
	if priorTwentyVolume > 0 {
		r.RelativeVolume20d = in.Volumes[len(in.Volumes)-1] / priorTwentyVolume
	}

	if in.CallVolume > 0 {
		putCall := float64(in.PutVolume) / float64(in.CallVolume)
		r.TotalOptionVolume = in.CallVolume + in.PutVolume
		r.PutCallVolumeRatio = &putCall
		if r.TotalOptionVolume >= 1000 {
			if putCall < .30 {
				r.OptionsFlowExtreme = "call_volume_extreme"
			} else if putCall > 1.20 {
				r.OptionsFlowExtreme = "put_volume_extreme"
			}
		}
	}

	if in.CallVolume > 0 && in.PutVolume > 0 {
		callPut := float64(in.CallVolume) / float64(in.PutVolume)
		reversal, trend := callPut, 1/callPut
		if r.Direction == DirectionBearish {
			reversal, trend = 1/callPut, callPut
		}
		r.ReversalFlowRatio = &reversal
		r.DirectionalFlowRatio = &reversal
		r.TrendFlowRatio = &trend
		r.FlowEligible = reversal >= 1.2
		r.FlowScore = flowProximity(reversal)
	} else {
		r.Reasons = append(r.Reasons, "reversal_options_flow_unavailable")
	}

	switch {
	case r.PriceEligible && r.RelativeVolume20d >= 1.75:
		r.Regime = RegimeClimacticExtension
		r.Candidate = true
		r.VolumeEligible = true
		r.VolumeScore = climacticVolumeScore(r.RelativeVolume20d)
	case r.PriceEligible && r.VolumeRatio > 0 && r.VolumeRatio <= .85:
		r.Regime = RegimeFadingDrift
		r.Candidate = true
		r.VolumeEligible = true
		r.VolumeScore = fadingVolumeScore(r.VolumeRatio)
	case r.PriceEligible && r.VolumeRatio >= .95 && r.TrendFlowRatio != nil && *r.TrendFlowRatio >= 1.2:
		r.Regime = RegimeTrendSupported
		r.Reasons = append(r.Reasons, "trend_supported_not_ranked")
	default:
		r.Regime = RegimeUnresolved
		r.Reasons = append(r.Reasons, "reversal_regime_not_established")
	}
	if r.Candidate {
		r.EvidenceComplete = r.FlowEligible
		r.Score = round(.25*r.PersistenceScore + .30*r.ExtensionScore + .25*r.VolumeScore + .20*r.FlowScore)
		if r.IVRegime == "elevated_premium" {
			r.IVAdjustment = 10
			r.Score = clamp(r.Score + r.IVAdjustment)
			r.Reasons = append(r.Reasons, "medium_term_iv_premium_corroborates_reversal_review")
		}
		r.StanceScore = stance(r.Direction, r.Score)
		if r.EvidenceComplete {
			r.State = StateConfirmed
		} else {
			r.State = StateObserved
			r.Reasons = append(r.Reasons, "reversal_options_flow_not_met")
		}
		r.Tier = tier(r.Score, r.EvidenceComplete)
		r.Outcome = outcome(r.Direction, r.EvidenceComplete)
		return r
	}
	r.State = StateObserved
	if r.Regime == RegimeTrendSupported {
		r.Tier = "TREND"
		r.Outcome = "trend_supported_not_ranked"
	} else {
		r.Tier = "UNRESOLVED"
		r.Outcome = "unresolved_not_ranked"
	}
	return r
}

func driftDirection(closes []float64) Direction {
	if closes[len(closes)-1] < closes[0] {
		return DirectionBullish
	}
	if closes[len(closes)-1] > closes[0] {
		return DirectionBearish
	}
	return DirectionNone
}
func consecutive(c []float64, down bool) int {
	n := 0
	for i := len(c) - 1; i > 0; i-- {
		if (down && c[i] < c[i-1]) || (!down && c[i] > c[i-1]) {
			n++
		} else {
			break
		}
	}
	return n
}
func directionalWindow(c []float64, down bool) (int, int) {
	bestN, bestW := 0, 0
	for _, w := range []int{5, 6, 7} {
		n := 0
		for i := len(c) - w; i < len(c); i++ {
			if (down && c[i] < c[i-1]) || (!down && c[i] > c[i-1]) {
				n++
			}
		}
		if n*5 >= w*4 && (n > bestN || n == bestN && w > bestW) {
			bestN, bestW = n, w
		}
	}
	return bestN, bestW
}
func directionalRate(c []float64, down bool) float64 {
	best := 0.0
	for _, w := range []int{5, 6, 7} {
		n := 0
		for i := len(c) - w; i < len(c); i++ {
			if (down && c[i] < c[i-1]) || (!down && c[i] > c[i-1]) {
				n++
			}
		}
		best = math.Max(best, float64(n)/float64(w))
	}
	return best
}
func persistenceScore(streak int, rate float64) float64 {
	return clamp(math.Max(math.Min(float64(streak)/4, 1), rate) * 100)
}
func extensionPercent(c []float64, down bool) float64 {
	current := c[len(c)-1]
	extreme := current
	for _, close := range c {
		if down && close > extreme {
			extreme = close
		}
		if !down && close < extreme {
			extreme = close
		}
	}
	if extreme <= 0 {
		return 0
	}
	return math.Abs(current-extreme) / extreme * 100
}
func meanAbsoluteDailyMove(c []float64) float64 {
	values := make([]float64, 0, len(c)-1)
	for i := 1; i < len(c); i++ {
		if c[i-1] > 0 {
			values = append(values, math.Abs(c[i]/c[i-1]-1)*100)
		}
	}
	return mean(values)
}
func extensionScore(units float64) float64          { return clamp((units - 1.5) / 1.5 * 100) }
func fadingVolumeScore(ratio float64) float64       { return clamp((1.1 - ratio) / .4 * 100) }
func climacticVolumeScore(relative float64) float64 { return clamp((relative - 1) / .75 * 100) }
func flowProximity(ratio float64) float64           { return clamp((ratio - .8) / .4 * 100) }
func stance(direction Direction, score float64) float64 {
	if direction == DirectionBullish {
		return score
	}
	if direction == DirectionBearish {
		return -score
	}
	return 0
}
func outcome(direction Direction, complete bool) string {
	prefix := "incomplete"
	if complete {
		prefix = "complete"
	}
	if direction == DirectionBullish {
		return prefix + "_bullish_reversal_review"
	}
	if direction == DirectionBearish {
		return prefix + "_bearish_reversal_review"
	}
	return "unresolved_not_ranked"
}
func tier(score float64, complete bool) string {
	if complete {
		return "CONFIRMED"
	}
	if score >= 75 {
		return "STRONG"
	}
	if score >= 50 {
		return "DEVELOPING"
	}
	if score >= 25 {
		return "EMERGING"
	}
	return "LOW"
}
func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
func round(value float64) float64 { return math.Round(value*100) / 100 }
