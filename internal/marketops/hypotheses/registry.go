package hypotheses

import (
	"encoding/json"

	"github.com/lukebabs/signalops/internal/storage"
)

const Version = "v1"

func ResearchDefinitions(tenantID string) []storage.MarketOpsHypothesisDefinitionRecord {
	return []storage.MarketOpsHypothesisDefinitionRecord{
		definition(tenantID, "H001", "Overbought downside-hedging expansion", "momentum_exhaustion", "downside",
			[]any{"rsi_14", cell("iv", "put", 30, .25), cell("iv", "put", 60, .25), cell("extrinsic_premium", "put", 30, .25), cell("oi_change_1d", "put", 30, .25), "surface_coverage_ratio"},
			[]any{cell("iv", "put", 30, .25), cell("iv", "put", 60, .25), cell("extrinsic_premium", "put", 30, .25), cell("oi_change_1d", "put", 30, .25)},
			map[string]any{"rsi_min": 70, "iv_change_min": .02, "premium_change_min": 0, "oi_change_min": 0, "surface_coverage_min": .8},
			[]any{"forward_drawdown_5d", "forward_drawdown_10d", "forward_drawdown_20d", "realized_volatility_change"}),
		definition(tenantID, "H004", "Volatility term-structure regime shift", "volatility_surface", "non_directional",
			[]any{"atm_iv_30d", "atm_iv_60d", "atm_iv_90d", "surface_coverage_ratio"},
			[]any{"atm_iv_30d", "atm_iv_60d", "atm_iv_90d"},
			map[string]any{"surface_coverage_min": .8, "term_spread_min": .02, "minimum_persistence_sessions": 2},
			[]any{"realized_volatility_change", "term_structure_state"}),
		definition(tenantID, "H006", "Premium-price divergence", "divergence", "conditional",
			[]any{"return_1d", cell("mid_premium", "put_or_call", 30, .25), cell("iv", "put_or_call", 30, .25)},
			[]any{cell("mid_premium", "put_or_call", 30, .25), cell("iv", "put_or_call", 30, .25)},
			map[string]any{"underlying_move_min_pct": 1, "premium_or_iv_change_min": .01, "requires_bid_ask": true},
			[]any{"forward_return_5d", "forward_return_10d", "realized_volatility_change"}),
		definition(tenantID, "H007", "Delta-bucket unusual OI accumulation", "option_positioning", "conditional",
			[]any{cell("oi_change_1d", "put_or_call", 30, .25), "surface_coverage_ratio"},
			[]any{cell("oi_change_1d", "put_or_call", 30, .25)},
			map[string]any{"minimum_oi_change": 100, "minimum_zscore": 2, "minimum_percentile": .95, "surface_coverage_min": .6},
			[]any{"forward_return_5d", "forward_return_10d", "realized_volatility_change"}),
	}
}

func definition(tenantID, key, title, domain, direction string, features, transitions []any, thresholds map[string]any, outcomes []any) storage.MarketOpsHypothesisDefinitionRecord {
	return storage.MarketOpsHypothesisDefinitionRecord{
		TenantID: tenantID, HypothesisKey: key, HypothesisVersion: Version, Title: title, Domain: domain, Direction: direction,
		Description:          "Research-only deterministic evaluation of " + title + ".",
		Rationale:            "Preserve eligible, triggered, and rejected state evaluations before calibration or production promotion.",
		RequiredFeaturesJSON: j(features), RequiredTransitionsJSON: j(transitions),
		QualityPolicyJSON:         j(map[string]any{"accepted_quality_states": []string{storage.MarketOpsQualityUsable, storage.MarketOpsQualityUsableWithWarning}, "missing_is_not_zero": true}),
		EligibilityExpressionJSON: j(map[string]any{"operator": "all_required_inputs_usable"}),
		TriggerExpressionJSON:     j(map[string]any{"operator": "hypothesis_specific_thresholds", "thresholds": thresholds}),
		PersistenceRuleJSON:       j(map[string]any{"minimum_sessions": thresholds["minimum_persistence_sessions"]}),
		CorroborationRuleJSON:     j(map[string]any{"requires_independent_evidence": true}),
		InvalidationRuleJSON:      j(map[string]any{"quality_failure_invalidates": true, "earnings_window": "record_when_available"}),
		ExpectedOutcomesJSON:      j(outcomes),
		ScoringConfigJSON:         j(map[string]any{"magnitude_weight": .25, "rarity_weight": .2, "persistence_weight": .15, "corroboration_weight": .2, "quality_weight": .2}),
		CalibrationPolicyJSON:     j(map[string]any{"minimum_sample_size": 100, "requires_walk_forward": true, "production_materialization_allowed": false}),
		LifecycleStatus:           storage.MarketOpsHypothesisLifecycleResearch, Owner: "marketops-research",
	}
}

func cell(key, optionType string, dte int, delta float64) map[string]any {
	return map[string]any{"feature_key": key, "dimensions": map[string]any{"option_type": optionType, "target_dte": dte, "target_delta": delta}}
}

func j(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
