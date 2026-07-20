package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	marketStateContextStrategy       = "market_state_session_v1"
	marketStateContextBuilderVersion = "syncratic.context_builder.v2"
	marketStateContextPayloadVersion = "marketops.syncratic.context_payload.v2"
	marketStateAskPromptVersion      = "marketops.syncratic.ask_prompt.v2"
)

type marketStateContextWarning struct {
	Code     string `json:"code"`
	RecordID string `json:"record_id,omitempty"`
	Message  string `json:"message"`
	Blocking bool   `json:"blocking"`
}

func buildMarketStateSyncraticContext(ctx context.Context, repo storage.QueryRepository, tenantID, marketStateID, builderVersion string) (storage.SyncraticContextWindowRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	marketStateID = strings.TrimSpace(marketStateID)
	if tenantID == "" || marketStateID == "" {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("tenant_id and market_state_id are required")
	}
	state, err := repo.GetMarketOpsMarketState(ctx, marketStateID)
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	if state.TenantID != tenantID {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("market_state_id does not belong to tenant_id")
	}
	if strings.TrimSpace(state.AssetID) == "" || strings.TrimSpace(state.Symbol) == "" || state.SessionDate.IsZero() || strings.TrimSpace(state.StateSchemaVersion) == "" {
		return storage.SyncraticContextWindowRecord{}, fmt.Errorf("market_state_id has incomplete persisted identity")
	}
	builderVersion = firstNonEmpty(strings.TrimSpace(builderVersion), marketStateContextBuilderVersion)
	symbol := strings.ToUpper(strings.TrimSpace(state.Symbol))
	session := dateUTC(state.SessionDate)
	windowEnd := session.AddDate(0, 0, 1)

	features, err := repo.ListMarketOpsFeatureObservations(ctx, storage.MarketOpsFeatureObservationFilter{TenantID: tenantID, AppID: state.AppID, AssetID: state.AssetID, Symbol: symbol, SessionStart: session, SessionEnd: session, FeatureObservationIDs: state.FeatureObservationIDs, Limit: 500})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	transitions, err := repo.ListMarketOpsStateTransitions(ctx, storage.MarketOpsStateTransitionFilter{TenantID: tenantID, AppID: state.AppID, AssetID: state.AssetID, Symbol: symbol, CurrentStateID: state.MarketStateID, Limit: 200})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	evaluations, err := repo.ListMarketOpsHypothesisEvaluations(ctx, storage.MarketOpsHypothesisEvaluationFilter{TenantID: tenantID, AppID: state.AppID, MarketStateID: state.MarketStateID, AssetID: state.AssetID, Symbol: symbol, Limit: 100})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	sessionEvidence, err := repo.ListMarketOpsEvidence(ctx, storage.MarketOpsEvidenceFilter{TenantID: tenantID, AppID: state.AppID, AssetID: state.AssetID, Symbol: symbol, SessionStart: session, SessionEnd: session, Limit: 500})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	opportunities, err := repo.ListMarketOpsOpportunities(ctx, storage.MarketOpsOpportunityFilter{TenantID: tenantID, AppID: state.AppID, AssetID: state.AssetID, Symbol: symbol, SessionStart: session, SessionEnd: session, Limit: 200})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	outcomes, err := repo.ListMarketOpsSignalOutcomes(ctx, storage.MarketOpsSignalOutcomeFilter{TenantID: tenantID, AppID: state.AppID, Symbol: symbol, OriginStart: session, OriginEnd: session, Limit: 500})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}
	calibrations, err := repo.ListMarketOpsBacktestCalibrationSummaries(ctx, storage.MarketOpsBacktestCalibrationSummaryFilter{TenantID: tenantID, AppID: state.AppID, Domain: "market_data", UseCase: "daily_market_surveillance", Limit: 200})
	if err != nil {
		return storage.SyncraticContextWindowRecord{}, err
	}

	warnings := []marketStateContextWarning{}
	block := func(code, id, message string) {
		if len(warnings) < 50 {
			warnings = append(warnings, marketStateContextWarning{Code: code, RecordID: id, Message: message, Blocking: true})
		}
	}
	warn := func(code, id, message string) {
		if len(warnings) < 50 {
			warnings = append(warnings, marketStateContextWarning{Code: code, RecordID: id, Message: message})
		}
	}

	featureByID := map[string]storage.MarketOpsFeatureObservationRecord{}
	for _, item := range features {
		if !marketStateIdentityMatches(state, item.TenantID, item.AssetID, item.Symbol, item.SessionDate) {
			block("feature_identity_mismatch", item.FeatureObservationID, "feature observation does not match selected state identity")
			continue
		}
		featureByID[item.FeatureObservationID] = item
	}
	for _, id := range state.FeatureObservationIDs {
		if _, ok := featureByID[id]; !ok {
			block("unresolved_feature_lineage", id, "state feature lineage ID was not resolved")
		}
	}
	sort.Slice(features, func(i, j int) bool {
		pi, pj := featurePriority(features[i]), featurePriority(features[j])
		if pi != pj {
			return pi < pj
		}
		if features[i].FeatureKey != features[j].FeatureKey {
			return features[i].FeatureKey < features[j].FeatureKey
		}
		return features[i].FeatureObservationID < features[j].FeatureObservationID
	})
	features = pureFeatures(features, featureByID)
	featureAvailable := len(features)
	features = truncateFeatures(features, 25, warn)

	pureTransitions := make([]storage.MarketOpsStateTransitionRecord, 0, len(transitions))
	for _, item := range transitions {
		if item.CurrentStateID != state.MarketStateID || !marketStateIdentityMatches(state, item.TenantID, item.AssetID, item.Symbol, item.SessionDate) {
			block("transition_identity_mismatch", item.TransitionID, "transition does not match selected state and session")
			continue
		}
		pureTransitions = append(pureTransitions, item)
	}
	transitions = pureTransitions
	transitionAvailable := len(transitions)
	if len(transitions) > 50 {
		warn("truncated_state_transitions", state.MarketStateID, fmt.Sprintf("returned 50 of %d transitions", len(transitions)))
		transitions = transitions[:50]
	}

	pureEvaluations := make([]storage.MarketOpsHypothesisEvaluationRecord, 0, len(evaluations))
	evalSet := map[string]bool{}
	evalVersion := map[string]string{}
	for _, item := range evaluations {
		if item.MarketStateID != state.MarketStateID || !marketStateIdentityMatches(state, item.TenantID, item.AssetID, item.Symbol, item.SessionDate) {
			block("evaluation_identity_mismatch", item.EvaluationID, "hypothesis evaluation does not match selected state identity")
			continue
		}
		def, defErr := repo.GetMarketOpsHypothesisDefinition(ctx, tenantID, item.HypothesisKey, item.HypothesisVersion)
		if defErr != nil || def.HypothesisVersion != item.HypothesisVersion {
			block("evaluation_definition_unresolved", item.EvaluationID, "exact hypothesis definition version was not resolved")
			continue
		}
		pureEvaluations = append(pureEvaluations, item)
	}
	sort.Slice(pureEvaluations, func(i, j int) bool {
		if pureEvaluations[i].HypothesisKey != pureEvaluations[j].HypothesisKey {
			return pureEvaluations[i].HypothesisKey < pureEvaluations[j].HypothesisKey
		}
		return pureEvaluations[i].EvaluationID < pureEvaluations[j].EvaluationID
	})
	evaluationAvailable := len(pureEvaluations)
	if len(pureEvaluations) > 8 {
		warn("truncated_hypothesis_evaluations", state.MarketStateID, fmt.Sprintf("returned 8 of %d evaluations", len(pureEvaluations)))
		pureEvaluations = pureEvaluations[:8]
	}
	for _, item := range pureEvaluations {
		evalSet[item.EvaluationID] = true
		evalVersion[item.HypothesisKey+"|"+item.HypothesisVersion] = item.EvaluationID
	}

	wantedEvidence := map[string]bool{}
	for _, item := range pureEvaluations {
		for _, id := range item.EvidenceIDs {
			wantedEvidence[id] = true
		}
	}
	for _, item := range transitions {
		for _, evidence := range sessionEvidence {
			for _, id := range evidence.SourceTransitionIDs {
				if id == item.TransitionID {
					wantedEvidence[evidence.EvidenceID] = true
				}
			}
		}
	}
	pureEvidence := []storage.MarketOpsEvidenceRecord{}
	for _, item := range sessionEvidence {
		if !wantedEvidence[item.EvidenceID] {
			continue
		}
		if !marketStateIdentityMatches(state, item.TenantID, item.AssetID, item.Symbol, item.SessionDate) {
			block("evidence_identity_mismatch", item.EvidenceID, "evidence does not match selected state identity")
			continue
		}
		conflicts := extractKnownSymbols(item.EvidencePayloadJSON)
		for candidate := range extractKnownSymbols([]byte(item.Statement)) {
			conflicts[candidate] = struct{}{}
		}
		conflictingTicker := false
		for candidate := range conflicts {
			if candidate != symbol {
				block("evidence_symbol_conflict", item.EvidenceID, "evidence contains conflicting ticker "+candidate)
				conflictingTicker = true
			}
		}
		if conflictingTicker {
			continue
		}
		pureEvidence = append(pureEvidence, item)
	}
	sort.Slice(pureEvidence, func(i, j int) bool { return pureEvidence[i].EvidenceID < pureEvidence[j].EvidenceID })
	evidenceAvailable := len(pureEvidence)
	if len(pureEvidence) > 20 {
		warn("truncated_marketops_evidence", state.MarketStateID, fmt.Sprintf("returned 20 of %d evidence rows", len(pureEvidence)))
		pureEvidence = pureEvidence[:20]
	}
	resolvedEvidence := map[string]bool{}
	for _, item := range pureEvidence {
		resolvedEvidence[item.EvidenceID] = true
	}
	for id := range wantedEvidence {
		if !resolvedEvidence[id] {
			block("unresolved_evidence_lineage", id, "linked evidence ID was not resolved in the bounded session")
		}
	}

	pureOpportunities := []storage.MarketOpsOpportunityRecord{}
	for _, item := range opportunities {
		if item.TenantID != tenantID || item.AssetID != state.AssetID || strings.ToUpper(item.Symbol) != symbol {
			block("opportunity_identity_mismatch", item.OpportunityID, "opportunity does not match selected state identity")
			continue
		}
		contributes := false
		outside := false
		for _, id := range append(append([]string{}, item.HypothesisEvaluationIDs...), item.ConflictingEvaluationIDs...) {
			if evalSet[id] {
				contributes = true
			} else {
				outside = true
			}
		}
		if !contributes || outside {
			block("opportunity_contribution_mismatch", item.OpportunityID, "opportunity contributions are outside the selected context")
			continue
		}
		pureOpportunities = append(pureOpportunities, item)
	}
	opportunityAvailable := len(pureOpportunities)
	if len(pureOpportunities) > 10 {
		warn("truncated_opportunities", state.MarketStateID, fmt.Sprintf("returned 10 of %d opportunities", len(pureOpportunities)))
		pureOpportunities = pureOpportunities[:10]
	}
	opportunitySet := map[string]bool{}
	for _, item := range pureOpportunities {
		opportunitySet[item.OpportunityID] = true
	}

	pureOutcomes := []storage.MarketOpsSignalOutcomeRecord{}
	for _, item := range outcomes {
		sourcePresent := item.SourceType == storage.MarketOpsOutcomeSourceHypothesisEvaluation && evalSet[item.SourceID] ||
			item.SourceType == storage.MarketOpsOutcomeSourceOpportunity && opportunitySet[item.SourceID]
		if !sourcePresent {
			if item.SourceType == storage.MarketOpsOutcomeSourceHypothesisEvaluation || item.SourceType == storage.MarketOpsOutcomeSourceOpportunity {
				block("outcome_source_absent", item.OutcomeID, "outcome source is absent from selected context")
			}
			continue
		}
		if item.TenantID != tenantID || item.AssetID != state.AssetID || strings.ToUpper(item.Symbol) != symbol || !sameUTCDate(item.OriginSessionDate, session) {
			block("outcome_identity_mismatch", item.OutcomeID, "outcome does not match selected state identity")
			continue
		}
		pureOutcomes = append(pureOutcomes, item)
	}
	outcomeAvailable := len(pureOutcomes)
	if len(pureOutcomes) > 20 {
		warn("truncated_outcomes", state.MarketStateID, fmt.Sprintf("returned 20 of %d outcomes", len(pureOutcomes)))
		pureOutcomes = pureOutcomes[:20]
	}

	calibrationPayloads := []map[string]any{}
	calibrationIDs := []string{}
	seenCalibration := map[string]bool{}
	for _, item := range calibrations {
		var report map[string]any
		if json.Unmarshal(item.ParametersJSON, &report) != nil || asString(report["summary_version"]) != "marketops.hypothesis_calibration.v1" {
			continue
		}
		key := asString(report["hypothesis_key"])
		versions := stringSliceAny(report["hypothesis_versions"])
		for _, version := range versions {
			pair := key + "|" + version
			if _, needed := evalVersion[pair]; !needed || seenCalibration[pair] {
				continue
			}
			versionMap, ok := report["versions"].(map[string]any)
			if !ok {
				block("calibration_version_missing", item.SummaryID, "calibration report lacks exact version map")
				continue
			}
			selected, ok := versionMap[version]
			if !ok {
				block("calibration_version_missing", item.SummaryID, "calibration report lacks exact hypothesis version")
				continue
			}
			seenCalibration[pair] = true
			calibrationIDs = append(calibrationIDs, item.SummaryID)
			calibrationPayloads = append(calibrationPayloads, map[string]any{"summary_id": item.SummaryID, "hypothesis_key": key, "hypothesis_version": version, "selected_version": selected, "warnings": report["warnings"], "minimum_sample_size": report["minimum_sample_size"], "promotion_allowed": report["promotion_allowed"]})
			if len(calibrationIDs) == 8 {
				break
			}
		}
		if len(calibrationIDs) == 8 {
			break
		}
	}
	for pair, id := range evalVersion {
		if !seenCalibration[pair] {
			warn("calibration_unavailable", id, "no exact-version calibration summary is available")
		}
	}

	featureSummaries := make([]map[string]any, 0, len(features))
	for _, item := range features {
		featureSummaries = append(featureSummaries, map[string]any{"feature_observation_id": item.FeatureObservationID, "feature_key": item.FeatureKey, "feature_version": item.FeatureVersion, "dimensions": json.RawMessage(jsonOrDefault(item.DimensionsJSON, `{}`)), "numeric_value": item.NumericValue, "text_value": item.TextValue, "boolean_value": item.BooleanValue, "quality_state": item.QualityState, "quality_score": item.QualityScore, "source_event_ids": limitStrings(item.SourceEventIDs, 8), "source_artifact_ids": limitStrings(item.SourceArtifactIDs, 8)})
	}
	transitionSummaries := make([]map[string]any, 0, len(transitions))
	for _, item := range transitions {
		transitionSummaries = append(transitionSummaries, map[string]any{"transition_id": item.TransitionID, "feature_key": item.FeatureKey, "feature_version": item.FeatureVersion, "transition_type": item.TransitionType, "direction": item.Direction, "transition_value": item.TransitionValue, "zscore": item.ZScore, "percentile": item.Percentile, "quality_state": item.QualityState})
	}
	evaluationSummaries := make([]map[string]any, 0, len(pureEvaluations))
	for _, item := range pureEvaluations {
		evaluationSummaries = append(evaluationSummaries, map[string]any{"evaluation_id": item.EvaluationID, "hypothesis_key": item.HypothesisKey, "hypothesis_version": item.HypothesisVersion, "eligible": item.Eligible, "triggered": item.Triggered, "invalidated": item.Invalidated, "trigger_score": item.TriggerScore, "confidence_score": item.ConfidenceScore, "evidence_ids": item.EvidenceIDs, "reason_codes": item.ReasonCodes})
	}
	evidenceSummaries := make([]map[string]any, 0, len(pureEvidence))
	for _, item := range pureEvidence {
		evidenceSummaries = append(evidenceSummaries, map[string]any{"evidence_id": item.EvidenceID, "evidence_type": item.EvidenceType, "evidence_version": item.EvidenceVersion, "domain": item.Domain, "direction": item.Direction, "magnitude": item.Magnitude, "rarity_score": item.RarityScore, "persistence_score": item.PersistenceScore, "quality_score": item.QualityScore, "statement": truncateText(item.Statement, 240), "source_feature_ids": item.SourceFeatureIDs, "source_transition_ids": item.SourceTransitionIDs})
	}
	opportunitySummaries := make([]map[string]any, 0, len(pureOpportunities))
	for _, item := range pureOpportunities {
		opportunitySummaries = append(opportunitySummaries, map[string]any{"opportunity_id": item.OpportunityID, "direction": item.Direction, "horizon": item.Horizon, "lifecycle_status": item.LifecycleStatus, "opportunity_score": item.OpportunityScore, "confidence_score": item.ConfidenceScore, "hypothesis_evaluation_ids": item.HypothesisEvaluationIDs, "conflicting_evaluation_ids": item.ConflictingEvaluationIDs, "research_only": item.ResearchOnly, "version": item.Version})
	}
	outcomeSummaries := make([]map[string]any, 0, len(pureOutcomes))
	for _, item := range pureOutcomes {
		outcomeSummaries = append(outcomeSummaries, map[string]any{"outcome_id": item.OutcomeID, "source_type": item.SourceType, "source_id": item.SourceID, "hypothesis_key": item.HypothesisKey, "hypothesis_version": item.HypothesisVersion, "direction": item.Direction, "horizon_sessions": item.HorizonSessions, "outcome_status": item.OutcomeStatus, "forward_return": item.ForwardReturn, "directional_hit": item.DirectionalHit, "calculation_version": item.CalculationVersion})
	}
	counts := map[string]any{
		"features": countMeta(featureAvailable, len(features)), "state_transitions": countMeta(transitionAvailable, len(transitions)),
		"hypothesis_evaluations": countMeta(evaluationAvailable, len(pureEvaluations)), "marketops_evidence": countMeta(evidenceAvailable, len(pureEvidence)),
		"opportunities": countMeta(opportunityAvailable, len(pureOpportunities)), "outcomes": countMeta(outcomeAvailable, len(pureOutcomes)),
		"calibration_summaries": countMeta(len(seenCalibration), len(calibrationIDs)),
	}
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Code != warnings[j].Code {
			return warnings[i].Code < warnings[j].Code
		}
		if warnings[i].RecordID != warnings[j].RecordID {
			return warnings[i].RecordID < warnings[j].RecordID
		}
		return warnings[i].Message < warnings[j].Message
	})
	analysisMode := "market_interpretation_allowed"
	for _, item := range warnings {
		if item.Blocking {
			analysisMode = "data_quality_blocked"
			break
		}
	}
	lineage := map[string]any{
		"market_state":  map[string]any{"market_state_id": state.MarketStateID, "asset_id": state.AssetID, "symbol": symbol, "session_date": session.Format("2006-01-02"), "as_of_time": state.AsOfTime.UTC().Format(time.RFC3339), "state_schema_version": state.StateSchemaVersion, "quality_state": state.QualityState, "quality_score": state.QualityScore, "feature_count": state.FeatureCount, "required_feature_count": state.RequiredFeatureCount, "completeness_ratio": state.CompletenessRatio, "eligible_hypotheses": state.EligibleHypotheses},
		"surface_cells": marketStateSurfaceCells(features), "features": featureSummaries, "state_transitions": transitionSummaries,
		"hypothesis_evaluations": evaluationSummaries, "marketops_evidence": evidenceSummaries, "opportunities": opportunitySummaries,
		"outcomes": outcomeSummaries, "calibration_summaries": calibrationPayloads,
	}
	record := storage.SyncraticContextWindowRecord{
		TenantID: tenantID, AppID: firstNonEmpty(state.AppID, "marketops"), Domain: "market_data", UseCase: "daily_market_surveillance",
		SubjectType: "asset", SubjectID: state.AssetID, SubjectSymbol: symbol, WindowStart: session, WindowEnd: windowEnd,
		ContextStrategy: marketStateContextStrategy, ContextBuilderVersion: builderVersion, ContextPayloadVersion: marketStateContextPayloadVersion,
		MarketStateIDs: []string{state.MarketStateID}, StateTransitionIDs: transitionIDs(transitions), MarketOpsEvidenceIDs: evidenceIDs(pureEvidence),
		HypothesisEvaluationIDs: evaluationIDs(pureEvaluations), OpportunityIDs: opportunityIDs(pureOpportunities), OutcomeIDs: outcomeIDs(pureOutcomes),
		CalibrationSummaryIDs: uniqueSorted(calibrationIDs), BaselineRefsJSON: []byte(`[]`), EvaluationRefsJSON: []byte(`[]`),
		PromotionCandidateRefsJSON: []byte(`[]`), QualityWarningsJSON: mustJSON(warnings), LineageRefsJSON: mustJSON(lineage),
		SummaryMetricsJSON: mustJSON(map[string]any{"analysis_mode": analysisMode, "counts": counts, "subject_symbol": symbol, "session_date": session.Format("2006-01-02"), "state_schema_version": state.StateSchemaVersion, "context_strategy": marketStateContextStrategy}),
		Status:             "active",
	}
	record.IdempotencyKey = strings.Join([]string{tenantID, record.UseCase, marketStateContextStrategy, state.MarketStateID, state.StateSchemaVersion, builderVersion}, "|")
	record.EvidenceDigest = marketStateContextDigest(record)
	record.ContextWindowID = stableSyncraticID("synctx", record.IdempotencyKey)
	return record, nil
}

func marketStateIdentityMatches(state storage.MarketOpsMarketStateRecord, tenantID, assetID, symbol string, session time.Time) bool {
	return tenantID == state.TenantID && assetID == state.AssetID && strings.ToUpper(strings.TrimSpace(symbol)) == strings.ToUpper(strings.TrimSpace(state.Symbol)) && sameUTCDate(session, state.SessionDate)
}

func dateUTC(value time.Time) time.Time {
	y, m, d := value.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func sameUTCDate(a, b time.Time) bool { return dateUTC(a).Equal(dateUTC(b)) }

func featurePriority(item storage.MarketOpsFeatureObservationRecord) int {
	if canonicalSurfaceCell(item) != "" {
		return 0
	}
	if item.QualityState == storage.MarketOpsQualityUsable || item.QualityState == storage.MarketOpsQualityUsableWithWarning {
		return 1
	}
	return 2
}

func pureFeatures(items []storage.MarketOpsFeatureObservationRecord, allowed map[string]storage.MarketOpsFeatureObservationRecord) []storage.MarketOpsFeatureObservationRecord {
	out := make([]storage.MarketOpsFeatureObservationRecord, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if _, ok := allowed[item.FeatureObservationID]; ok && !seen[item.FeatureObservationID] {
			seen[item.FeatureObservationID] = true
			out = append(out, item)
		}
	}
	return out
}

func truncateFeatures(items []storage.MarketOpsFeatureObservationRecord, cap int, warn func(string, string, string)) []storage.MarketOpsFeatureObservationRecord {
	if len(items) <= cap {
		return items
	}
	warn("truncated_features", "", fmt.Sprintf("returned %d of %d feature summaries", cap, len(items)))
	return items[:cap]
}

func canonicalSurfaceCell(item storage.MarketOpsFeatureObservationRecord) string {
	switch item.FeatureKey {
	case "atm_iv_30d":
		return "atm_30d"
	case "atm_iv_60d":
		return "atm_60d"
	case "atm_iv_90d":
		return "atm_90d"
	}
	if item.FeatureKey != "iv" {
		return ""
	}
	var dims map[string]any
	if json.Unmarshal(item.DimensionsJSON, &dims) != nil {
		return ""
	}
	optionType := strings.ToLower(asString(dims["option_type"]))
	dte := int(asFloat(dims["target_dte"]))
	delta := asFloat(dims["target_delta"])
	if delta < 0 {
		delta = -delta
	}
	if (optionType == "call" || optionType == "put") && (dte == 30 || dte == 60) && delta >= .24 && delta <= .26 {
		return fmt.Sprintf("%s_%dd_25delta", optionType, dte)
	}
	return ""
}

func marketStateSurfaceCells(features []storage.MarketOpsFeatureObservationRecord) []map[string]any {
	names := []string{"atm_30d", "atm_60d", "atm_90d", "call_30d_25delta", "put_30d_25delta", "call_60d_25delta", "put_60d_25delta"}
	byName := map[string]storage.MarketOpsFeatureObservationRecord{}
	for _, item := range features {
		if name := canonicalSurfaceCell(item); name != "" {
			byName[name] = item
		}
	}
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		if item, ok := byName[name]; ok {
			out = append(out, map[string]any{"cell": name, "status": item.QualityState, "feature_observation_id": item.FeatureObservationID, "numeric_value": item.NumericValue})
		} else {
			out = append(out, map[string]any{"cell": name, "status": "missing", "feature_observation_id": nil, "numeric_value": nil})
		}
	}
	return out
}

func countMeta(available, returned int) map[string]any {
	return map[string]any{"available": available, "returned": returned, "truncated": available > returned, "omitted": max(0, available-returned)}
}

func stringSliceAny(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, item := range raw {
		if text := strings.TrimSpace(asString(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func asFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case json.Number:
		number, _ := typed.Float64()
		return number
	default:
		return 0
	}
}

func transitionIDs(items []storage.MarketOpsStateTransitionRecord) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.TransitionID)
	}
	return uniqueSorted(out)
}
func evidenceIDs(items []storage.MarketOpsEvidenceRecord) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.EvidenceID)
	}
	return uniqueSorted(out)
}
func evaluationIDs(items []storage.MarketOpsHypothesisEvaluationRecord) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.EvaluationID)
	}
	return uniqueSorted(out)
}
func opportunityIDs(items []storage.MarketOpsOpportunityRecord) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.OpportunityID)
	}
	return uniqueSorted(out)
}
func outcomeIDs(items []storage.MarketOpsSignalOutcomeRecord) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.OutcomeID)
	}
	return uniqueSorted(out)
}

func marketStateContextDigest(record storage.SyncraticContextWindowRecord) string {
	raw := mustJSON(map[string]any{"market_states": record.MarketStateIDs, "transitions": record.StateTransitionIDs, "evidence": record.MarketOpsEvidenceIDs, "evaluations": record.HypothesisEvaluationIDs, "opportunities": record.OpportunityIDs, "outcomes": record.OutcomeIDs, "calibration": record.CalibrationSummaryIDs, "quality_warnings": json.RawMessage(jsonOrDefault(record.QualityWarningsJSON, `[]`)), "lineage": json.RawMessage(jsonOrDefault(record.LineageRefsJSON, `{}`)), "metrics": json.RawMessage(jsonOrDefault(record.SummaryMetricsJSON, `{}`))})
	sum := sha256Bytes(raw)
	return sum
}

func sha256Bytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
