package state

import (
	"encoding/json"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	FeatureVersion     = "v1"
	StateSchemaVersion = "marketops.market_state.v2"
	EvidenceVersion    = "v1"
)

type featureSpec struct {
	Key            string
	Domain         string
	Title          string
	Description    string
	Unit           string
	RequiredInputs []string
	Calculation    map[string]any
	QualityPolicy  map[string]any
}

var g137FeatureSpecs = []featureSpec{
	{Key: "return_1d", Domain: "underlying_momentum", Title: "1-session return", Unit: "percent", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "close_return", "lookback_sessions": 1}},
	{Key: "return_5d", Domain: "underlying_momentum", Title: "5-session return", Unit: "percent", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "close_return", "lookback_sessions": 5}},
	{Key: "return_10d", Domain: "underlying_momentum", Title: "10-session return", Unit: "percent", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "close_return", "lookback_sessions": 10}},
	{Key: "return_20d", Domain: "underlying_momentum", Title: "20-session return", Unit: "percent", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "close_return", "lookback_sessions": 20}},
	{Key: "rsi_14", Domain: "underlying_momentum", Title: "14-session RSI", Unit: "index", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "wilder_rsi", "lookback_sessions": 14}},
	{Key: "distance_sma_20_pct", Domain: "underlying_momentum", Title: "Distance from 20-session SMA", Unit: "percent", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "distance_from_sma", "lookback_sessions": 20}},
	{Key: "volume_ratio_20d", Domain: "underlying_momentum", Title: "Volume ratio to 20-session mean", Unit: "ratio", RequiredInputs: []string{"equity.volume"}, Calculation: map[string]any{"operator": "ratio_to_mean", "lookback_sessions": 20}},
	{Key: "gap_pct", Domain: "underlying_momentum", Title: "Opening gap", Unit: "percent", RequiredInputs: []string{"equity.open", "prior_equity.close"}, Calculation: map[string]any{"operator": "open_gap"}},
	{Key: "atr_14_pct", Domain: "underlying_momentum", Title: "14-session ATR as percent of close", Unit: "percent", RequiredInputs: []string{"equity.high", "equity.low", "equity.close"}, Calculation: map[string]any{"operator": "average_true_range", "lookback_sessions": 14}},
	{Key: "atm_iv_30d", Domain: "implied_volatility", Title: "30-DTE ATM implied volatility", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "options.delta", "options.expiration_date"}, Calculation: map[string]any{"operator": "nearest_surface_cell", "target_dte": 30, "target_abs_delta": 0.5}},
	{Key: "atm_iv_60d", Domain: "implied_volatility", Title: "60-DTE ATM implied volatility", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "options.delta", "options.expiration_date"}, Calculation: map[string]any{"operator": "nearest_surface_cell", "target_dte": 60, "target_abs_delta": 0.5}},
	{Key: "atm_iv_90d", Domain: "implied_volatility", Title: "90-DTE ATM implied volatility", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "options.delta", "options.expiration_date"}, Calculation: map[string]any{"operator": "nearest_surface_cell", "target_dte": 90, "target_abs_delta": 0.5}},
	{Key: "iv", Domain: "implied_volatility", Title: "Delta-target implied volatility", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "options.delta", "options.expiration_date"}, Calculation: map[string]any{"operator": "nearest_surface_cell", "dimensions": []string{"option_type", "target_dte", "target_delta"}}},
	{Key: "iv_term_slope", Domain: "volatility_surface", Title: "Implied-volatility term slope", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility"}, Calculation: map[string]any{"operator": "far_iv_minus_near_iv", "dimensions": []string{"near_dte", "far_dte"}}},
	{Key: "risk_reversal", Domain: "volatility_surface", Title: "25-delta risk reversal", Unit: "decimal", RequiredInputs: []string{"options.call_implied_volatility", "options.put_implied_volatility"}, Calculation: map[string]any{"operator": "call_iv_minus_put_iv", "dimensions": []string{"target_dte", "target_delta"}}},
	{Key: "surface_selection_confidence", Domain: "volatility_surface", Title: "Surface selection confidence", Unit: "ratio", RequiredInputs: []string{"options.selection_score"}, Calculation: map[string]any{"operator": "mean_one_minus_bounded_selection_score", "required_cells": 7}},
	{Key: "mid_premium", Domain: "option_premium", Title: "Selected option midpoint premium", Unit: "currency", RequiredInputs: []string{"options.bid", "options.ask"}, Calculation: map[string]any{"operator": "bid_ask_midpoint"}, QualityPolicy: map[string]any{"requires_positive_bid_ask": true}},
	{Key: "extrinsic_premium", Domain: "option_premium", Title: "Selected option extrinsic premium", Unit: "currency", RequiredInputs: []string{"options.bid", "options.ask", "equity.close", "options.strike_price"}, Calculation: map[string]any{"operator": "midpoint_less_intrinsic"}, QualityPolicy: map[string]any{"requires_positive_bid_ask": true}},
	{Key: "premium_pct_spot", Domain: "option_premium", Title: "Selected premium as percent of spot", Unit: "percent", RequiredInputs: []string{"options.bid", "options.ask", "equity.close"}, Calculation: map[string]any{"operator": "midpoint_percent_spot"}, QualityPolicy: map[string]any{"requires_positive_bid_ask": true}},
	{Key: "spread_pct", Domain: "option_liquidity", Title: "Selected option bid-ask spread", Unit: "percent", RequiredInputs: []string{"options.bid", "options.ask"}, Calculation: map[string]any{"operator": "bid_ask_spread_percent_midpoint"}, QualityPolicy: map[string]any{"requires_positive_bid_ask": true}},
	{Key: "oi_change_1d", Domain: "option_positioning", Title: "Delta-target open-interest change", Unit: "contracts", RequiredInputs: []string{"options.open_interest", "prior_options.open_interest"}, Calculation: map[string]any{"operator": "surface_bucket_absolute_difference", "dimensions": []string{"option_type", "target_dte", "target_delta"}}, QualityPolicy: map[string]any{"missing_is_not_zero": true}},
	{Key: "put_call_oi_ratio", Domain: "option_positioning", Title: "Put/call open-interest ratio", Unit: "ratio", RequiredInputs: []string{"options.total_put_open_interest", "options.total_call_open_interest"}, Calculation: map[string]any{"operator": "put_divided_by_call_open_interest"}, QualityPolicy: map[string]any{"requires_usable_open_interest": true}},
	{Key: "put_call_oi_change_1d", Domain: "option_positioning", Title: "1-session put/call OI ratio change", Unit: "ratio", RequiredInputs: []string{"put_call_oi_ratio"}, Calculation: map[string]any{"operator": "absolute_difference", "lookback_sessions": 1}, QualityPolicy: map[string]any{"requires_usable_open_interest": true}},
	{Key: "put_call_volume_ratio", Domain: "option_positioning", Title: "Put/call volume ratio", Unit: "ratio", RequiredInputs: []string{"options.total_put_volume", "options.total_call_volume"}, Calculation: map[string]any{"operator": "put_divided_by_call_volume"}},
	{Key: "usable_contract_ratio", Domain: "option_liquidity", Title: "Usable option contract ratio", Unit: "ratio", RequiredInputs: []string{"options.chain"}, Calculation: map[string]any{"operator": "eligible_contracts_divided_by_contracts"}},
	{Key: "missing_iv_ratio", Domain: "option_liquidity", Title: "Missing implied-volatility ratio", Unit: "ratio", RequiredInputs: []string{"options.chain"}, Calculation: map[string]any{"operator": "missing_field_ratio", "field": "implied_volatility"}},
	{Key: "missing_greeks_ratio", Domain: "option_liquidity", Title: "Missing delta ratio", Unit: "ratio", RequiredInputs: []string{"options.chain"}, Calculation: map[string]any{"operator": "missing_field_ratio", "field": "delta"}},
	{Key: "surface_coverage_ratio", Domain: "volatility_surface", Title: "Required IV surface-cell coverage", Unit: "ratio", RequiredInputs: []string{"options.implied_volatility", "options.delta", "options.expiration_date"}, Calculation: map[string]any{"operator": "selected_cells_divided_by_required_cells", "required_cells": 7}},
	{Key: "oi_quality_state", Domain: "option_liquidity", Title: "Open-interest quality state", RequiredInputs: []string{"options.distribution.metrics.open_interest_quality"}, Calculation: map[string]any{"operator": "source_quality_passthrough"}},
}

var g144FeatureSpecs = []featureSpec{
	{Key: "rv_10d", Domain: "realized_volatility", Title: "10-session realized volatility", Unit: "decimal", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "annualized_log_return_standard_deviation", "lookback_sessions": 10, "annualization_sessions": 252}},
	{Key: "rv_20d", Domain: "realized_volatility", Title: "20-session realized volatility", Unit: "decimal", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "annualized_log_return_standard_deviation", "lookback_sessions": 20, "annualization_sessions": 252}},
	{Key: "rv_60d", Domain: "realized_volatility", Title: "60-session realized volatility", Unit: "decimal", RequiredInputs: []string{"equity.close"}, Calculation: map[string]any{"operator": "annualized_log_return_standard_deviation", "lookback_sessions": 60, "annualization_sessions": 252}},
	{Key: "rv_acceleration_5d", Domain: "realized_volatility", Title: "5-session realized-volatility acceleration", Unit: "decimal", RequiredInputs: []string{"rv_20d"}, Calculation: map[string]any{"operator": "absolute_difference", "lookback_sessions": 5}},
	{Key: "iv_change_1d", Domain: "implied_volatility", Title: "1-session normalized-cell IV change", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "prior_options.implied_volatility"}, Calculation: map[string]any{"operator": "surface_cell_absolute_difference", "lookback_sessions": 1}},
	{Key: "iv_change_5d", Domain: "implied_volatility", Title: "5-session normalized-cell IV change", Unit: "decimal", RequiredInputs: []string{"options.implied_volatility", "prior_options.implied_volatility"}, Calculation: map[string]any{"operator": "surface_cell_absolute_difference", "lookback_sessions": 5}},
	{Key: "iv_minus_rv_20d", Domain: "implied_volatility", Title: "30-DTE ATM IV less 20-session realized volatility", Unit: "decimal", RequiredInputs: []string{"atm_iv_30d", "rv_20d"}, Calculation: map[string]any{"operator": "absolute_difference"}},
	{Key: "iv_rv_ratio_20d", Domain: "implied_volatility", Title: "30-DTE ATM IV to 20-session realized-volatility ratio", Unit: "ratio", RequiredInputs: []string{"atm_iv_30d", "rv_20d"}, Calculation: map[string]any{"operator": "ratio"}},
	{Key: "term_structure_state", Domain: "volatility_surface", Title: "30/60/90-DTE term-structure state", RequiredInputs: []string{"atm_iv_30d", "atm_iv_60d", "atm_iv_90d"}, Calculation: map[string]any{"operator": "classified_curve_state", "flat_tolerance": 0.0025}},
	{Key: "premium_change_1d", Domain: "option_premium", Title: "1-session normalized-cell premium change", Unit: "currency", RequiredInputs: []string{"options.bid", "options.ask", "prior_options.bid", "prior_options.ask"}, Calculation: map[string]any{"operator": "surface_cell_midpoint_absolute_difference", "lookback_sessions": 1}},
	{Key: "premium_change_5d", Domain: "option_premium", Title: "5-session normalized-cell premium change", Unit: "currency", RequiredInputs: []string{"options.bid", "options.ask", "prior_options.bid", "prior_options.ask"}, Calculation: map[string]any{"operator": "surface_cell_midpoint_absolute_difference", "lookback_sessions": 5}},
	{Key: "oi_change_5d", Domain: "option_positioning", Title: "5-session normalized-cell open-interest change", Unit: "contracts", RequiredInputs: []string{"options.open_interest", "prior_options.open_interest"}, Calculation: map[string]any{"operator": "surface_cell_absolute_difference", "lookback_sessions": 5}, QualityPolicy: map[string]any{"missing_is_not_zero": true}},
	{Key: "days_to_earnings", Domain: "market_event_context", Title: "Days to next known earnings event", Unit: "calendar_days", RequiredInputs: []string{"market_event_calendar.earnings_date", "market_event_calendar.known_at"}, Calculation: map[string]any{"operator": "calendar_days_to_next_point_in_time_known_event"}},
	{Key: "days_since_earnings", Domain: "market_event_context", Title: "Days since latest known earnings event", Unit: "calendar_days", RequiredInputs: []string{"market_event_calendar.earnings_date", "market_event_calendar.known_at"}, Calculation: map[string]any{"operator": "calendar_days_since_latest_point_in_time_known_event"}},
	{Key: "earnings_window_state", Domain: "market_event_context", Title: "Known earnings-window state", RequiredInputs: []string{"market_event_calendar.earnings_date", "market_event_calendar.known_at"}, Calculation: map[string]any{"operator": "classified_earnings_window", "pre_event_days": 5, "post_event_days": 2}},
}

func FeatureDefinitions(tenantID string) []storage.MarketOpsFeatureDefinitionRecord {
	specs := append(append([]featureSpec{}, g137FeatureSpecs...), g144FeatureSpecs...)
	definitions := make([]storage.MarketOpsFeatureDefinitionRecord, 0, len(specs))
	for _, spec := range specs {
		calculation, _ := json.Marshal(spec.Calculation)
		required, _ := json.Marshal(spec.RequiredInputs)
		policy := spec.QualityPolicy
		if policy == nil {
			policy = map[string]any{"missing_is_not_zero": true}
		}
		quality, _ := json.Marshal(policy)
		valueType := "numeric"
		if spec.Key == "oi_quality_state" || spec.Key == "term_structure_state" || spec.Key == "earnings_window_state" {
			valueType = "text"
		}
		definitions = append(definitions, storage.MarketOpsFeatureDefinitionRecord{
			TenantID: tenantID, FeatureKey: spec.Key, FeatureVersion: FeatureVersion,
			Domain: spec.Domain, Title: spec.Title, Description: spec.Description,
			ValueType: valueType, Unit: spec.Unit, CalculationSpec: calculation,
			RequiredInputs: required, QualityPolicy: quality,
			Status: storage.MarketOpsFeatureDefinitionStatusActive,
		})
	}
	return definitions
}
