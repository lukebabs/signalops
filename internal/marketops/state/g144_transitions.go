package state

import (
	"encoding/json"

	"github.com/lukebabs/signalops/internal/storage"
)

var g144TransitionLookbacks = []int{3, 5, 10, 20}

func buildG144Transitions(config BuildConfig, states []storage.MarketOpsMarketStateRecord, observations map[string][]storage.MarketOpsFeatureObservationRecord) ([]storage.MarketOpsStateTransitionRecord, error) {
	out := []storage.MarketOpsStateTransitionRecord{}
	for index, current := range states {
		currentObservations := observations[dateKey(current.SessionDate)]
		for _, lookback := range g144TransitionLookbacks {
			if index < lookback {
				continue
			}
			baseline := states[index-lookback]
			baselineByKey := observationMap(observations[dateKey(baseline.SessionDate)])
			for _, currentObservation := range currentObservations {
				canonical, _ := CanonicalDimensions(currentObservation.DimensionsJSON)
				prior, ok := baselineByKey[observationKey(currentObservation.FeatureKey, canonical)]
				if currentObservation.NumericValue == nil || !isUsable(currentObservation.QualityState) || !ok || prior.NumericValue == nil || !isUsable(prior.QualityState) {
					continue
				}
				identity, err := NewIdentity(IdentityStateTransition, config.TenantID, config.AssetID, dateKey(current.SessionDate), currentObservation.FeatureKey, FeatureVersion, canonical, "absolute_difference", dateKey(baseline.SessionDate))
				if err != nil {
					return nil, err
				}
				change := round(*currentObservation.NumericValue-*prior.NumericValue, 8)
				payload, _ := json.Marshal(map[string]any{
					"current_feature_observation_id":  currentObservation.FeatureObservationID,
					"baseline_feature_observation_id": prior.FeatureObservationID,
					"baseline_session_date":           dateKey(baseline.SessionDate),
					"operator":                        "absolute_difference",
					"point_in_time_statistics":        true,
					"configured_lookback_sessions":    lookback,
				})
				lookbackValue := lookback
				out = append(out, storage.MarketOpsStateTransitionRecord{
					TransitionID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
					SessionDate: current.SessionDate, AsOfTime: current.AsOfTime, CurrentStateID: current.MarketStateID, BaselineStateID: baseline.MarketStateID,
					FeatureKey: currentObservation.FeatureKey, FeatureVersion: FeatureVersion, DimensionsJSON: currentObservation.DimensionsJSON,
					TransitionType: "absolute_difference", LookbackSessions: &lookbackValue, CurrentValue: currentObservation.NumericValue, BaselineValue: prior.NumericValue,
					TransitionValue: &change, Direction: directionFor(change), QualityState: storage.MarketOpsQualityUsable,
					TransitionPayloadJSON: payload, CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
				})
			}
		}
		if index >= 2 {
			accelerations, err := g144AccelerationTransitions(config, states, observations, index)
			if err != nil {
				return nil, err
			}
			out = append(out, accelerations...)
		}
		if index >= 1 {
			regime, err := g144TermRegimeTransition(config, states[index-1], current, observations)
			if err != nil {
				return nil, err
			}
			if regime.TransitionID != "" {
				out = append(out, regime)
			}
		}
	}
	return out, nil
}

func g144AccelerationTransitions(config BuildConfig, states []storage.MarketOpsMarketStateRecord, observations map[string][]storage.MarketOpsFeatureObservationRecord, index int) ([]storage.MarketOpsStateTransitionRecord, error) {
	current, prior, baseline := states[index], states[index-1], states[index-2]
	currentByKey := observationMap(observations[dateKey(current.SessionDate)])
	priorByKey := observationMap(observations[dateKey(prior.SessionDate)])
	baselineByKey := observationMap(observations[dateKey(baseline.SessionDate)])
	out := []storage.MarketOpsStateTransitionRecord{}
	for key, currentObservation := range currentByKey {
		if !supportsG144Acceleration(currentObservation.FeatureKey) || currentObservation.NumericValue == nil || !isUsable(currentObservation.QualityState) {
			continue
		}
		priorObservation, priorOK := priorByKey[key]
		baselineObservation, baselineOK := baselineByKey[key]
		if !priorOK || !baselineOK || priorObservation.NumericValue == nil || baselineObservation.NumericValue == nil || !isUsable(priorObservation.QualityState) || !isUsable(baselineObservation.QualityState) {
			continue
		}
		canonical, _ := CanonicalDimensions(currentObservation.DimensionsJSON)
		identity, err := NewIdentity(IdentityStateTransition, config.TenantID, config.AssetID, dateKey(current.SessionDate), currentObservation.FeatureKey, FeatureVersion, canonical, "acceleration", dateKey(baseline.SessionDate))
		if err != nil {
			return nil, err
		}
		currentChange := *currentObservation.NumericValue - *priorObservation.NumericValue
		priorChange := *priorObservation.NumericValue - *baselineObservation.NumericValue
		acceleration := round(currentChange-priorChange, 8)
		lookback := 2
		payload, _ := json.Marshal(map[string]any{
			"operator":                            "second_difference",
			"intermediate_state_id":               prior.MarketStateID,
			"current_feature_observation_id":      currentObservation.FeatureObservationID,
			"intermediate_feature_observation_id": priorObservation.FeatureObservationID,
			"baseline_feature_observation_id":     baselineObservation.FeatureObservationID,
			"current_first_difference":            round(currentChange, 8),
			"prior_first_difference":              round(priorChange, 8),
			"point_in_time_statistics":            true,
		})
		out = append(out, storage.MarketOpsStateTransitionRecord{
			TransitionID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
			SessionDate: current.SessionDate, AsOfTime: current.AsOfTime, CurrentStateID: current.MarketStateID, BaselineStateID: baseline.MarketStateID,
			FeatureKey: currentObservation.FeatureKey, FeatureVersion: FeatureVersion, DimensionsJSON: currentObservation.DimensionsJSON,
			TransitionType: "acceleration", LookbackSessions: &lookback, CurrentValue: currentObservation.NumericValue, BaselineValue: baselineObservation.NumericValue,
			TransitionValue: &acceleration, Direction: directionFor(acceleration), QualityState: storage.MarketOpsQualityUsable,
			TransitionPayloadJSON: payload, CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
		})
	}
	return out, nil
}

func supportsG144Acceleration(featureKey string) bool {
	switch featureKey {
	case "rv_20d", "atm_iv_30d", "atm_iv_60d", "atm_iv_90d", "iv", "iv_term_slope", "mid_premium", "extrinsic_premium", "oi_change_1d":
		return true
	default:
		return false
	}
}

func g144TermRegimeTransition(config BuildConfig, baseline, current storage.MarketOpsMarketStateRecord, observations map[string][]storage.MarketOpsFeatureObservationRecord) (storage.MarketOpsStateTransitionRecord, error) {
	currentObservation, currentOK := observationMap(observations[dateKey(current.SessionDate)])[observationKey("term_structure_state", "{}")]
	baselineObservation, baselineOK := observationMap(observations[dateKey(baseline.SessionDate)])[observationKey("term_structure_state", "{}")]
	if !currentOK || !baselineOK || currentObservation.TextValue == nil || baselineObservation.TextValue == nil || !isUsable(currentObservation.QualityState) || !isUsable(baselineObservation.QualityState) || *currentObservation.TextValue == *baselineObservation.TextValue {
		return storage.MarketOpsStateTransitionRecord{}, nil
	}
	identity, err := NewIdentity(IdentityStateTransition, config.TenantID, config.AssetID, dateKey(current.SessionDate), "term_structure_state", FeatureVersion, "{}", "regime_transition", dateKey(baseline.SessionDate))
	if err != nil {
		return storage.MarketOpsStateTransitionRecord{}, err
	}
	lookback := 1
	payload, _ := json.Marshal(map[string]any{
		"operator":                        "classified_regime_transition",
		"baseline_state":                  *baselineObservation.TextValue,
		"current_state":                   *currentObservation.TextValue,
		"baseline_feature_observation_id": baselineObservation.FeatureObservationID,
		"current_feature_observation_id":  currentObservation.FeatureObservationID,
		"point_in_time_statistics":        true,
	})
	return storage.MarketOpsStateTransitionRecord{
		TransitionID: identity.ID, TenantID: config.TenantID, AppID: "marketops", AssetID: config.AssetID, Symbol: config.Symbol,
		SessionDate: current.SessionDate, AsOfTime: current.AsOfTime, CurrentStateID: current.MarketStateID, BaselineStateID: baseline.MarketStateID,
		FeatureKey: "term_structure_state", FeatureVersion: FeatureVersion, DimensionsJSON: []byte("{}"),
		TransitionType: "regime_transition", LookbackSessions: &lookback, Direction: *currentObservation.TextValue,
		QualityState: storage.MarketOpsQualityUsable, TransitionPayloadJSON: payload, CalculationRunID: config.RunID, DeterministicKey: identity.DeterministicKey,
	}, nil
}
