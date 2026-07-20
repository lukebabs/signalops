package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsHypothesisRoutes(mux *http.ServeMux, queryRepository storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/hypotheses", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsHypothesisDefinitions(r.Context(), storage.MarketOpsHypothesisDefinitionFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), HypothesisKey: strings.TrimSpace(r.URL.Query().Get("hypothesis_key")), HypothesisVersion: strings.TrimSpace(r.URL.Query().Get("hypothesis_version")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")), LifecycleStatus: strings.TrimSpace(r.URL.Query().Get("lifecycle_status")), Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps hypothesis definitions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hypotheses": hypothesisDefinitionResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/hypotheses/{hypothesis_key}/{hypothesis_version}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := repo.GetMarketOpsHypothesisDefinition(r.Context(), tenantID, r.PathValue("hypothesis_key"), r.PathValue("hypothesis_version"))
		if err != nil {
			writeQueryError(w, err, "hypothesis_not_found", "MarketOps hypothesis not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hypothesis": hypothesisDefinitionResponse(record)})
	})
	mux.HandleFunc("GET /v1/marketops/hypothesis-evaluations", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		eligible, ok := optionalBoolQuery(w, r, "eligible")
		if !ok {
			return
		}
		triggered, ok := optionalBoolQuery(w, r, "triggered")
		if !ok {
			return
		}
		invalidated, ok := optionalBoolQuery(w, r, "invalidated")
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsHypothesisEvaluations(r.Context(), storage.MarketOpsHypothesisEvaluationFilter{TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")), HypothesisKey: strings.TrimSpace(r.URL.Query().Get("hypothesis_key")), HypothesisVersion: strings.TrimSpace(r.URL.Query().Get("hypothesis_version")), MarketStateID: strings.TrimSpace(r.URL.Query().Get("market_state_id")), AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Eligible: eligible, Triggered: triggered, Invalidated: invalidated, SessionStart: start, SessionEnd: end, Limit: queryLimit(r, 50)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps hypothesis evaluations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hypothesis_evaluations": hypothesisEvaluationResponses(records)})
	})
}

type hypothesisDefinitionDTO struct {
	TenantID              string          `json:"tenant_id"`
	HypothesisKey         string          `json:"hypothesis_key"`
	HypothesisVersion     string          `json:"hypothesis_version"`
	Title                 string          `json:"title"`
	Domain                string          `json:"domain"`
	Direction             string          `json:"direction"`
	Description           string          `json:"description"`
	Rationale             string          `json:"rationale"`
	RequiredFeatures      json.RawMessage `json:"required_features"`
	RequiredTransitions   json.RawMessage `json:"required_transitions"`
	QualityPolicy         json.RawMessage `json:"quality_policy"`
	EligibilityExpression json.RawMessage `json:"eligibility_expression"`
	TriggerExpression     json.RawMessage `json:"trigger_expression"`
	PersistenceRule       json.RawMessage `json:"persistence_rule"`
	CorroborationRule     json.RawMessage `json:"corroboration_rule"`
	InvalidationRule      json.RawMessage `json:"invalidation_rule"`
	ExpectedOutcomes      json.RawMessage `json:"expected_outcomes"`
	ScoringConfig         json.RawMessage `json:"scoring_config"`
	CalibrationPolicy     json.RawMessage `json:"calibration_policy"`
	LifecycleStatus       string          `json:"lifecycle_status"`
	Owner                 string          `json:"owner,omitempty"`
	ApprovedBy            string          `json:"approved_by,omitempty"`
	ApprovedAt            *time.Time      `json:"approved_at,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}
type hypothesisEvaluationDTO struct {
	EvaluationID       string          `json:"evaluation_id"`
	TenantID           string          `json:"tenant_id"`
	AppID              string          `json:"app_id"`
	HypothesisKey      string          `json:"hypothesis_key"`
	HypothesisVersion  string          `json:"hypothesis_version"`
	MarketStateID      string          `json:"market_state_id"`
	AssetID            string          `json:"asset_id"`
	Symbol             string          `json:"symbol"`
	SessionDate        time.Time       `json:"session_date"`
	AsOfTime           time.Time       `json:"as_of_time"`
	Eligible           bool            `json:"eligible"`
	Triggered          bool            `json:"triggered"`
	TriggerScore       *float64        `json:"trigger_score,omitempty"`
	ConfidenceScore    *float64        `json:"confidence_score,omitempty"`
	MagnitudeScore     *float64        `json:"magnitude_score,omitempty"`
	RarityScore        *float64        `json:"rarity_score,omitempty"`
	PersistenceScore   *float64        `json:"persistence_score,omitempty"`
	CorroborationScore *float64        `json:"corroboration_score,omitempty"`
	QualityScore       *float64        `json:"quality_score,omitempty"`
	Invalidated        bool            `json:"invalidated"`
	EvidenceIDs        []string        `json:"evidence_ids"`
	ReasonCodes        []string        `json:"reason_codes"`
	EvaluationPayload  json.RawMessage `json:"evaluation_payload"`
	EvaluationRunID    string          `json:"evaluation_run_id"`
	DeterministicKey   string          `json:"deterministic_key"`
	CreatedAt          time.Time       `json:"created_at"`
}

func hypothesisDefinitionResponse(r storage.MarketOpsHypothesisDefinitionRecord) hypothesisDefinitionDTO {
	return hypothesisDefinitionDTO{TenantID: r.TenantID, HypothesisKey: r.HypothesisKey, HypothesisVersion: r.HypothesisVersion, Title: r.Title, Domain: r.Domain, Direction: r.Direction, Description: r.Description, Rationale: r.Rationale, RequiredFeatures: json.RawMessage(jsonOrDefault(r.RequiredFeaturesJSON, `[]`)), RequiredTransitions: json.RawMessage(jsonOrDefault(r.RequiredTransitionsJSON, `[]`)), QualityPolicy: json.RawMessage(jsonOrDefault(r.QualityPolicyJSON, `{}`)), EligibilityExpression: json.RawMessage(jsonOrDefault(r.EligibilityExpressionJSON, `{}`)), TriggerExpression: json.RawMessage(jsonOrDefault(r.TriggerExpressionJSON, `{}`)), PersistenceRule: json.RawMessage(jsonOrDefault(r.PersistenceRuleJSON, `{}`)), CorroborationRule: json.RawMessage(jsonOrDefault(r.CorroborationRuleJSON, `{}`)), InvalidationRule: json.RawMessage(jsonOrDefault(r.InvalidationRuleJSON, `{}`)), ExpectedOutcomes: json.RawMessage(jsonOrDefault(r.ExpectedOutcomesJSON, `[]`)), ScoringConfig: json.RawMessage(jsonOrDefault(r.ScoringConfigJSON, `{}`)), CalibrationPolicy: json.RawMessage(jsonOrDefault(r.CalibrationPolicyJSON, `{}`)), LifecycleStatus: r.LifecycleStatus, Owner: r.Owner, ApprovedBy: r.ApprovedBy, ApprovedAt: r.ApprovedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func hypothesisDefinitionResponses(records []storage.MarketOpsHypothesisDefinitionRecord) []hypothesisDefinitionDTO {
	out := make([]hypothesisDefinitionDTO, 0, len(records))
	for _, r := range records {
		out = append(out, hypothesisDefinitionResponse(r))
	}
	return out
}
func hypothesisEvaluationResponse(r storage.MarketOpsHypothesisEvaluationRecord) hypothesisEvaluationDTO {
	return hypothesisEvaluationDTO{EvaluationID: r.EvaluationID, TenantID: r.TenantID, AppID: r.AppID, HypothesisKey: r.HypothesisKey, HypothesisVersion: r.HypothesisVersion, MarketStateID: r.MarketStateID, AssetID: r.AssetID, Symbol: r.Symbol, SessionDate: r.SessionDate, AsOfTime: r.AsOfTime, Eligible: r.Eligible, Triggered: r.Triggered, TriggerScore: r.TriggerScore, ConfidenceScore: r.ConfidenceScore, MagnitudeScore: r.MagnitudeScore, RarityScore: r.RarityScore, PersistenceScore: r.PersistenceScore, CorroborationScore: r.CorroborationScore, QualityScore: r.QualityScore, Invalidated: r.Invalidated, EvidenceIDs: nonNilStrings(r.EvidenceIDs), ReasonCodes: nonNilStrings(r.ReasonCodes), EvaluationPayload: json.RawMessage(jsonOrDefault(r.EvaluationPayloadJSON, `{}`)), EvaluationRunID: r.EvaluationRunID, DeterministicKey: r.DeterministicKey, CreatedAt: r.CreatedAt}
}
func hypothesisEvaluationResponses(records []storage.MarketOpsHypothesisEvaluationRecord) []hypothesisEvaluationDTO {
	out := make([]hypothesisEvaluationDTO, 0, len(records))
	for _, r := range records {
		out = append(out, hypothesisEvaluationResponse(r))
	}
	return out
}
func optionalBoolQuery(w http.ResponseWriter, r *http.Request, key string) (*bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", key+" must be true or false")
		return nil, false
	}
	return &value, true
}
