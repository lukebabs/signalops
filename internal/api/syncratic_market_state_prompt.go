package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func buildMarketStateAskPrompt(contextWindow storage.SyncraticContextWindowRecord, req syncraticAskRequest) (string, syncraticAskPromptMeta, error) {
	if contextWindow.ContextStrategy != marketStateContextStrategy || len(contextWindow.MarketStateIDs) != 1 {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("market-state Ask requires exactly one persisted market_state_id")
	}
	version := firstNonEmpty(strings.TrimSpace(req.PromptBuilderVersion), marketStateAskPromptVersion)
	if version != marketStateAskPromptVersion {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("market-state context requires prompt_builder_version %s", marketStateAskPromptVersion)
	}
	maxPromptBytes := req.MaxPromptBytes
	if maxPromptBytes <= 0 {
		maxPromptBytes = 12000
	}
	if maxPromptBytes < 1000 {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("max_prompt_bytes must be at least 1000")
	}
	if maxPromptBytes > 12000 {
		maxPromptBytes = 12000
	}
	promptContext, promptTruncation := compactMarketStatePromptContext(contextWindow.LineageRefsJSON)
	caps := map[string]int{"max_market_states": 1, "max_features": 25, "max_state_transitions": 50, "max_hypothesis_evaluations": 8, "max_marketops_evidence": 20, "max_opportunities": 10, "max_outcomes": 20, "max_calibration_summaries": 8, "max_prompt_bytes": maxPromptBytes, "prompt_features": 8, "prompt_transitions": 12, "prompt_evidence": 8, "prompt_opportunities": 5, "prompt_outcomes": 8}
	payload := map[string]any{
		"prompt_builder_version": version,
		"role":                   "MarketOps research-surveillance reasoning over one bounded deterministic market-state session.",
		"instructions": []string{
			"Use only supplied persisted records. Do not retrieve, search, infer from external knowledge, or treat missing evidence as zero or neutral.",
			"Prefix every substantive claim with exactly one category: OBSERVED_FACT, CALCULATED_FEATURE, STATISTICAL_RARITY, HYPOTHESIS_INFERENCE, HISTORICAL_ASSOCIATION, GOVERNANCE_STATE, or UNKNOWN_FUTURE_OUTCOME.",
			"Cite persisted IDs for each claim. Keep calibration sample-size and compatibility warnings visible.",
			"Any historical association is not a guaranteed future outcome.",
			"Do not provide trade, order, allocation, position, or portfolio instructions.",
			"Do not claim hypothesis promotion, graph-proposal acceptance, lifecycle advancement, production signal materialization, or any mutation.",
			"If analysis_mode is data_quality_blocked, discuss only the quality defects and remediation; make no market-direction inference.",
		},
		"context_metadata":  map[string]any{"tenant_id": contextWindow.TenantID, "app_id": contextWindow.AppID, "domain": contextWindow.Domain, "use_case": contextWindow.UseCase, "context_window_id": contextWindow.ContextWindowID, "context_payload_version": contextWindow.ContextPayloadVersion, "context_strategy": contextWindow.ContextStrategy, "context_builder_version": contextWindow.ContextBuilderVersion, "subject_type": contextWindow.SubjectType, "subject_id": contextWindow.SubjectID, "subject_symbol": contextWindow.SubjectSymbol, "window_start": contextWindow.WindowStart, "window_end": contextWindow.WindowEnd, "evidence_digest": contextWindow.EvidenceDigest},
		"summary_metrics":   json.RawMessage(jsonOrDefault(contextWindow.SummaryMetricsJSON, `{}`)),
		"quality_warnings":  json.RawMessage(jsonOrDefault(contextWindow.QualityWarningsJSON, `[]`)),
		"bounded_context":   promptContext,
		"prompt_truncation": promptTruncation,
		"persisted_ids":     map[string]any{"market_states": contextWindow.MarketStateIDs, "state_transitions": contextWindow.StateTransitionIDs, "marketops_evidence": contextWindow.MarketOpsEvidenceIDs, "hypothesis_evaluations": contextWindow.HypothesisEvaluationIDs, "opportunities": contextWindow.OpportunityIDs, "outcomes": contextWindow.OutcomeIDs, "calibration_summaries": contextWindow.CalibrationSummaryIDs},
		"output_contract":   []string{"title and one-sentence summary", "categorized claims with cited persisted IDs", "quality_warnings and calibration_warnings", "uncertainty and unknown_future_outcomes", "operator next checks limited to research review or data remediation"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", syncraticAskPromptMeta{}, err
	}
	prompt := "Produce a bounded, evidence-pure Market State explanation. Generated synthesis is not deterministic evidence.\nCONTEXT_JSON:\n" + string(raw)
	if len(prompt) > maxPromptBytes {
		return "", syncraticAskPromptMeta{}, fmt.Errorf("prompt exceeds max_prompt_bytes")
	}
	sum := sha256.Sum256([]byte(prompt))
	return prompt, syncraticAskPromptMeta{PromptBuilderVersion: version, PromptDigest: "sha256:" + hex.EncodeToString(sum[:]), ContextEvidenceDigest: contextWindow.EvidenceDigest, MaxPromptBytes: maxPromptBytes, IncludedRecordDetails: true, Caps: caps, PromptBytes: len(prompt)}, nil
}

func marketStateAskQuestion(contextWindow storage.SyncraticContextWindowRecord) string {
	mode := asString(jsonObjectOrEmpty(contextWindow.SummaryMetricsJSON)["analysis_mode"])
	if mode == "data_quality_blocked" {
		return "Explain only the persisted data-quality blockers in this bounded Market State context, cite affected record IDs, and recommend deterministic evidence remediation. Do not infer market direction."
	}
	return "Explain this bounded Market State session using the required claim categories and persisted citations. Preserve missingness, calibration warnings, governance boundaries, and unknown future outcomes; provide no trading or lifecycle action."
}
