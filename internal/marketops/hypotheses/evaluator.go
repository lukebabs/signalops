package hypotheses

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	marketopsstate "github.com/lukebabs/signalops/internal/marketops/state"
	"github.com/lukebabs/signalops/internal/storage"
)

type inputSet struct {
	state        storage.MarketOpsMarketStateRecord
	observations []storage.MarketOpsFeatureObservationRecord
	transitions  []storage.MarketOpsStateTransitionRecord
	evidence     []storage.MarketOpsEvidenceRecord
}

type evaluation struct {
	eligible, triggered, invalidated                       bool
	reasons, featureIDs, transitionIDs                     []string
	checks                                                 map[string]any
	magnitude, rarity, persistence, corroboration, quality float64
}

func Evaluate(runID string, definition storage.MarketOpsHypothesisDefinitionRecord, state storage.MarketOpsMarketStateRecord, observations []storage.MarketOpsFeatureObservationRecord, transitions []storage.MarketOpsStateTransitionRecord, evidence []storage.MarketOpsEvidenceRecord) (storage.MarketOpsHypothesisEvaluationRecord, error) {
	if strings.TrimSpace(runID) == "" || definition.TenantID != state.TenantID || definition.HypothesisVersion == "" {
		return storage.MarketOpsHypothesisEvaluationRecord{}, fmt.Errorf("hypothesis evaluation run, tenant, and version must match")
	}
	input := inputSet{state: state, observations: observations, transitions: transitions, evidence: evidence}
	var result evaluation
	switch definition.HypothesisKey {
	case "H001":
		result = evaluateH001(input)
	case "H004":
		result = evaluateH004(input)
	case "H006":
		result = evaluateH006(input)
	case "H007":
		result = evaluateH007(input)
	default:
		return storage.MarketOpsHypothesisEvaluationRecord{}, fmt.Errorf("unsupported research hypothesis %s", definition.HypothesisKey)
	}
	result.reasons = unique(result.reasons)
	if result.eligible && !result.triggered && len(result.reasons) == 0 {
		result.reasons = []string{"eligible_not_triggered"}
	}
	if result.triggered {
		result.reasons = append(result.reasons, "triggered_research_only")
	}
	identity, err := marketopsstate.NewIdentity(marketopsstate.IdentityHypothesisEvaluation, state.TenantID, state.AssetID, state.SessionDate.Format("2006-01-02"), definition.HypothesisKey, definition.HypothesisVersion, state.MarketStateID)
	if err != nil {
		return storage.MarketOpsHypothesisEvaluationRecord{}, err
	}
	triggerScore := clamp(.25*result.magnitude + .2*result.rarity + .15*result.persistence + .2*result.corroboration + .2*result.quality)
	var triggerScorePtr, confidencePtr, magnitudePtr, rarityPtr, persistencePtr, corroborationPtr, qualityPtr *float64
	qualityPtr = pointer(clamp(result.quality))
	if result.eligible {
		triggerScorePtr, confidencePtr = pointer(triggerScore), pointer(triggerScore)
		magnitudePtr, rarityPtr, persistencePtr, corroborationPtr = pointer(clamp(result.magnitude)), pointer(clamp(result.rarity)), pointer(clamp(result.persistence)), pointer(clamp(result.corroboration))
	}
	evidenceIDs := linkedEvidenceIDs(evidence, result.featureIDs, result.transitionIDs)
	direction, horizon, family := evaluationOpportunityProfile(definition, result.checks)
	payload, _ := json.Marshal(map[string]any{"research_only": true, "checks": result.checks, "source_feature_ids": unique(result.featureIDs), "source_transition_ids": unique(result.transitionIDs), "state_quality": state.QualityState, "resolved_direction": direction, "horizon": horizon, "hypothesis_family": family})
	return storage.MarketOpsHypothesisEvaluationRecord{EvaluationID: identity.ID, TenantID: state.TenantID, AppID: "marketops", HypothesisKey: definition.HypothesisKey, HypothesisVersion: definition.HypothesisVersion, MarketStateID: state.MarketStateID, AssetID: state.AssetID, Symbol: state.Symbol, SessionDate: state.SessionDate, AsOfTime: state.AsOfTime, Eligible: result.eligible, Triggered: result.triggered, TriggerScore: triggerScorePtr, ConfidenceScore: confidencePtr, MagnitudeScore: magnitudePtr, RarityScore: rarityPtr, PersistenceScore: persistencePtr, CorroborationScore: corroborationPtr, QualityScore: qualityPtr, Invalidated: result.invalidated, EvidenceIDs: evidenceIDs, ReasonCodes: result.reasons, EvaluationPayloadJSON: payload, EvaluationRunID: runID, DeterministicKey: identity.DeterministicKey}, nil
}

func evaluateH001(in inputSet) evaluation {
	r := base(in)
	rsi := requireFeature(&r, in, "rsi_14", nil)
	coverage := requireFeature(&r, in, "surface_coverage_ratio", nil)
	put30 := requireFeature(&r, in, "iv", dims("put", 30, .25))
	put60 := requireFeature(&r, in, "iv", dims("put", 60, .25))
	premium := requireFeature(&r, in, "extrinsic_premium", dims("put", 30, .25))
	oi := requireFeature(&r, in, "oi_change_1d", dims("put", 30, .25))
	t30 := requireTransition(&r, in, "iv", dims("put", 30, .25))
	t60 := requireTransition(&r, in, "iv", dims("put", 60, .25))
	tp := requireTransition(&r, in, "extrinsic_premium", dims("put", 30, .25))
	to := requireTransition(&r, in, "oi_change_1d", dims("put", 30, .25))
	earningsWindow := optionalTextFeature(in, "earnings_window_state")
	if earningsWindow.FeatureObservationID != "" && usable(earningsWindow.QualityState) && earningsWindow.TextValue != nil {
		r.featureIDs = append(r.featureIDs, earningsWindow.FeatureObservationID)
		switch *earningsWindow.TextValue {
		case "pre_earnings", "earnings_day", "post_earnings":
			r.invalidated = true
			r.reasons = append(r.reasons, "known_earnings_window")
		}
	}
	r.eligible = len(r.reasons) == 0 && !r.invalidated
	if r.eligible {
		r.triggered = value(rsi) >= 70 && value(coverage) >= .8 && change(t30) > .02 && change(t60) > .02 && change(tp) > 0 && value(oi) > 0 && change(to) > 0 && value(premium) >= 0 && value(put30) > 0 && value(put60) > 0
		r.checks = map[string]any{"rsi_14": value(rsi), "surface_coverage_ratio": value(coverage), "put_iv_change_30d": change(t30), "put_iv_change_60d": change(t60), "premium_change": change(tp), "put_oi_change": value(oi)}
		r.magnitude = clamp((value(rsi)-70)/20 + math.Max(change(t30), 0)*5 + math.Max(change(t60), 0)*5)
		r.corroboration, r.persistence, r.rarity = 1, .5, .5
		if !r.triggered {
			r.reasons = thresholdReasons(map[string]bool{"rsi_below_threshold": value(rsi) < 70, "surface_coverage_below_minimum": value(coverage) < .8, "put_iv_not_expanding": change(t30) <= .02 || change(t60) <= .02, "premium_not_expanding": change(tp) <= 0, "put_oi_not_increasing": value(oi) <= 0 || change(to) <= 0})
		}
	}
	return r
}

func evaluateH004(in inputSet) evaluation {
	r := base(in)
	iv30 := requireFeature(&r, in, "atm_iv_30d", nil)
	iv60 := requireFeature(&r, in, "atm_iv_60d", nil)
	iv90 := requireFeature(&r, in, "atm_iv_90d", nil)
	coverage := requireFeature(&r, in, "surface_coverage_ratio", nil)
	t30 := requireTransition(&r, in, "atm_iv_30d", nil)
	t60 := requireTransition(&r, in, "atm_iv_60d", nil)
	t90 := requireTransition(&r, in, "atm_iv_90d", nil)
	if transitionPersistence(t30) < 2 && transitionPersistence(t60) < 2 && transitionPersistence(t90) < 2 {
		r.reasons = append(r.reasons, "insufficient_transition_persistence")
	}
	r.eligible = len(r.reasons) == 0
	if r.eligible {
		spread := value(iv30) - value(iv90)
		aligned := sameNonZeroDirection(change(t30), change(t60), change(t90))
		r.triggered = value(coverage) >= .8 && math.Abs(spread) >= .02 && aligned
		r.checks = map[string]any{"atm_iv_30d": value(iv30), "atm_iv_60d": value(iv60), "atm_iv_90d": value(iv90), "term_spread_30_90": spread, "aligned_transition": aligned, "surface_coverage_ratio": value(coverage)}
		r.magnitude, r.persistence, r.corroboration, r.rarity = clamp(math.Abs(spread)*10), 1, boolScore(aligned), .5
		if !r.triggered {
			r.reasons = thresholdReasons(map[string]bool{"surface_coverage_below_minimum": value(coverage) < .8, "term_spread_below_threshold": math.Abs(spread) < .02, "curve_transition_not_corroborated": !aligned})
		}
	}
	return r
}

func evaluateH006(in inputSet) evaluation {
	r := base(in)
	underlying := requireFeature(&r, in, "return_1d", nil)
	premium := requireAnyOptionFeature(&r, in, "mid_premium")
	iv := requireAnyOptionFeature(&r, in, "iv")
	premiumTransition := requireAnyOptionTransition(&r, in, "mid_premium")
	ivTransition := requireAnyOptionTransition(&r, in, "iv")
	r.eligible = len(r.reasons) == 0
	if r.eligible {
		move, premiumMove, ivMove := value(underlying), change(premiumTransition), change(ivTransition)
		diverges := math.Abs(move) >= 1 && ((move > 0 && (premiumMove < -.01 || ivMove < -.01)) || (move < 0 && (premiumMove > .01 || ivMove > .01)))
		r.triggered = diverges && value(premium) > 0 && value(iv) > 0
		r.checks = map[string]any{"return_1d": move, "premium_change": premiumMove, "iv_change": ivMove, "divergence": diverges}
		r.magnitude, r.corroboration, r.persistence, r.rarity = clamp(math.Abs(move)/5+math.Max(math.Abs(premiumMove), math.Abs(ivMove))*5), 1, .5, .5
		if !r.triggered {
			r.reasons = thresholdReasons(map[string]bool{"underlying_move_below_threshold": math.Abs(move) < 1, "premium_iv_divergence_not_detected": !diverges})
		}
	}
	return r
}

func evaluateH007(in inputSet) evaluation {
	r := base(in)
	oi := requireAnyOptionFeature(&r, in, "oi_change_1d")
	coverage := requireFeature(&r, in, "surface_coverage_ratio", nil)
	transition := requireAnyOptionTransition(&r, in, "oi_change_1d")
	r.eligible = len(r.reasons) == 0
	if r.eligible {
		rare := (transition.ZScore != nil && *transition.ZScore >= 2) || (transition.Percentile != nil && *transition.Percentile >= .95)
		r.triggered = value(oi) >= 100 && value(coverage) >= .6 && rare
		r.checks = map[string]any{"oi_change_1d": value(oi), "surface_coverage_ratio": value(coverage), "zscore": transition.ZScore, "percentile": transition.Percentile, "rarity_threshold_met": rare, "option_type": dimensionString(oi.DimensionsJSON, "option_type")}
		r.magnitude, r.rarity, r.persistence, r.corroboration = clamp(value(oi)/1000), boolScore(rare), .5, .25
		if !r.triggered {
			r.reasons = thresholdReasons(map[string]bool{"oi_change_below_minimum": value(oi) < 100, "surface_coverage_below_minimum": value(coverage) < .6, "oi_change_not_statistically_unusual": !rare})
		}
	}
	return r
}

func base(in inputSet) evaluation {
	q := 0.0
	if in.state.QualityScore != nil {
		q = *in.state.QualityScore
	}
	return evaluation{checks: map[string]any{}, quality: clamp(q)}
}

func requireFeature(r *evaluation, in inputSet, key string, dimensions map[string]any) storage.MarketOpsFeatureObservationRecord {
	for _, item := range in.observations {
		if item.FeatureKey == key && dimensionsMatch(item.DimensionsJSON, dimensions) {
			if !usable(item.QualityState) || item.NumericValue == nil {
				r.reasons = append(r.reasons, "unusable_feature:"+key+":"+item.QualityState)
				return item
			}
			r.featureIDs = append(r.featureIDs, item.FeatureObservationID)
			return item
		}
	}
	r.reasons = append(r.reasons, "missing_feature:"+key+dimensionToken(dimensions))
	return storage.MarketOpsFeatureObservationRecord{}
}

func requireAnyOptionFeature(r *evaluation, in inputSet, key string) storage.MarketOpsFeatureObservationRecord {
	for _, optionType := range []string{"put", "call"} {
		item := findFeature(in, key, dims(optionType, 30, .25))
		if item.FeatureObservationID != "" && usable(item.QualityState) && item.NumericValue != nil {
			r.featureIDs = append(r.featureIDs, item.FeatureObservationID)
			return item
		}
	}
	r.reasons = append(r.reasons, "missing_or_unusable_option_feature:"+key)
	return storage.MarketOpsFeatureObservationRecord{}
}

func requireTransition(r *evaluation, in inputSet, key string, dimensions map[string]any) storage.MarketOpsStateTransitionRecord {
	for _, item := range in.transitions {
		if item.FeatureKey == key && dimensionsMatch(item.DimensionsJSON, dimensions) && usable(item.QualityState) && item.TransitionValue != nil {
			r.transitionIDs = append(r.transitionIDs, item.TransitionID)
			return item
		}
	}
	r.reasons = append(r.reasons, "missing_usable_transition:"+key+dimensionToken(dimensions))
	return storage.MarketOpsStateTransitionRecord{}
}

func requireAnyOptionTransition(r *evaluation, in inputSet, key string) storage.MarketOpsStateTransitionRecord {
	for _, optionType := range []string{"put", "call"} {
		for _, item := range in.transitions {
			if item.FeatureKey == key && dimensionsMatch(item.DimensionsJSON, dims(optionType, 30, .25)) && usable(item.QualityState) && item.TransitionValue != nil {
				r.transitionIDs = append(r.transitionIDs, item.TransitionID)
				return item
			}
		}
	}
	r.reasons = append(r.reasons, "missing_usable_option_transition:"+key)
	return storage.MarketOpsStateTransitionRecord{}
}

func findFeature(in inputSet, key string, dimensions map[string]any) storage.MarketOpsFeatureObservationRecord {
	for _, item := range in.observations {
		if item.FeatureKey == key && dimensionsMatch(item.DimensionsJSON, dimensions) {
			return item
		}
	}
	return storage.MarketOpsFeatureObservationRecord{}
}
func dims(optionType string, dte int, delta float64) map[string]any {
	return map[string]any{"option_type": optionType, "target_dte": float64(dte), "target_delta": delta}
}
func dimensionsMatch(raw []byte, expected map[string]any) bool {
	if expected == nil {
		expected = map[string]any{}
	}
	actual := map[string]any{}
	_ = json.Unmarshal(raw, &actual)
	for key, value := range expected {
		if fmt.Sprint(actual[key]) != fmt.Sprint(value) {
			return false
		}
	}
	return true
}
func dimensionToken(value map[string]any) string {
	if len(value) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(value)
	return ":" + string(encoded)
}
func usable(q string) bool {
	return q == storage.MarketOpsQualityUsable || q == storage.MarketOpsQualityUsableWithWarning
}
func value(v storage.MarketOpsFeatureObservationRecord) float64 {
	if v.NumericValue == nil {
		return 0
	}
	return *v.NumericValue
}
func change(v storage.MarketOpsStateTransitionRecord) float64 {
	if v.TransitionValue == nil {
		return 0
	}
	return *v.TransitionValue
}
func transitionPersistence(v storage.MarketOpsStateTransitionRecord) int {
	if v.PersistenceSessions == nil {
		return 0
	}
	return *v.PersistenceSessions
}
func sameNonZeroDirection(values ...float64) bool {
	sign := 0
	for _, v := range values {
		current := 0
		if v > 0 {
			current = 1
		} else if v < 0 {
			current = -1
		}
		if current == 0 {
			return false
		}
		if sign == 0 {
			sign = current
		} else if sign != current {
			return false
		}
	}
	return true
}
func optionalTextFeature(in inputSet, key string) storage.MarketOpsFeatureObservationRecord {
	for _, item := range in.observations {
		if item.FeatureKey == key && dimensionsMatch(item.DimensionsJSON, nil) {
			return item
		}
	}
	return storage.MarketOpsFeatureObservationRecord{}
}

func thresholdReasons(checks map[string]bool) []string {
	out := []string{}
	for key, failed := range checks {
		if failed {
			out = append(out, "threshold_not_met:"+key)
		}
	}
	sort.Strings(out)
	return out
}
func linkedEvidenceIDs(records []storage.MarketOpsEvidenceRecord, featureIDs, transitionIDs []string) []string {
	refs := map[string]struct{}{}
	for _, v := range append(append([]string{}, featureIDs...), transitionIDs...) {
		refs[v] = struct{}{}
	}
	out := []string{}
	for _, record := range records {
		linked := false
		for _, v := range append(append([]string{}, record.SourceFeatureIDs...), record.SourceTransitionIDs...) {
			if _, ok := refs[v]; ok {
				linked = true
			}
		}
		if linked {
			out = append(out, record.EvidenceID)
		}
	}
	return unique(out)
}
func unique(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
func evaluationOpportunityProfile(definition storage.MarketOpsHypothesisDefinitionRecord, checks map[string]any) (string, string, string) {
	direction := definition.Direction
	switch definition.HypothesisKey {
	case "H006":
		move, _ := checks["return_1d"].(float64)
		if move > 0 {
			direction = "downside"
		} else if move < 0 {
			direction = "upside"
		}
	case "H007":
		optionType, _ := checks["option_type"].(string)
		if optionType == "put" {
			direction = "downside"
		} else if optionType == "call" {
			direction = "upside"
		}
	}
	return direction, "5_to_20_sessions", definition.Domain
}
func dimensionString(raw []byte, key string) string {
	values := map[string]any{}
	_ = json.Unmarshal(raw, &values)
	value, _ := values[key].(string)
	return value
}
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func pointer(v float64) *float64 { return &v }
func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
