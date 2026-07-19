package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsMarketStateRoutes(mux *http.ServeMux, queryRepository storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/features/definitions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsFeatureDefinitions(r.Context(), storage.MarketOpsFeatureDefinitionFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), FeatureKey: strings.TrimSpace(r.URL.Query().Get("feature_key")),
			FeatureVersion: strings.TrimSpace(r.URL.Query().Get("feature_version")), Domain: strings.TrimSpace(r.URL.Query().Get("domain")),
			Status: strings.TrimSpace(r.URL.Query().Get("status")), Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps feature definitions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"feature_definitions": marketOpsFeatureDefinitionResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/features/observations", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		dimensions, ok := marketOpsDimensionsQuery(w, r)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsFeatureObservations(r.Context(), storage.MarketOpsFeatureObservationFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")),
			FeatureKey: strings.TrimSpace(r.URL.Query().Get("feature_key")), FeatureVersion: strings.TrimSpace(r.URL.Query().Get("feature_version")),
			Domain: strings.TrimSpace(r.URL.Query().Get("domain")), QualityState: strings.TrimSpace(r.URL.Query().Get("quality_state")),
			DimensionsJSON: dimensions,
			SessionStart:   start, SessionEnd: end, Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps feature observations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"feature_observations": marketOpsFeatureObservationResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/states", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsMarketStates(r.Context(), storage.MarketOpsMarketStateFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")),
			StateSchemaVersion: strings.TrimSpace(r.URL.Query().Get("state_schema_version")),
			QualityState:       strings.TrimSpace(r.URL.Query().Get("quality_state")), SessionStart: start, SessionEnd: end,
			Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps market states")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"market_states": marketOpsMarketStateResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/states/{market_state_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsMarketState(r.Context(), r.PathValue("market_state_id"))
		if err != nil {
			writeQueryError(w, err, "market_state_not_found", "MarketOps market state not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"market_state": marketOpsMarketStateResponse(record)})
	})

	mux.HandleFunc("GET /v1/marketops/states/{market_state_id}/lineage", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		state, err := repo.GetMarketOpsMarketState(r.Context(), r.PathValue("market_state_id"))
		if err != nil {
			writeQueryError(w, err, "market_state_not_found", "MarketOps market state not found")
			return
		}
		lineage, err := resolveMarketOpsStateLineage(r, repo, state)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to resolve MarketOps market state lineage")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lineage": lineage})
	})

	mux.HandleFunc("GET /v1/marketops/transitions", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsStateTransitions(r.Context(), storage.MarketOpsStateTransitionFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")),
			CurrentStateID: strings.TrimSpace(r.URL.Query().Get("current_state_id")), FeatureKey: strings.TrimSpace(r.URL.Query().Get("feature_key")),
			FeatureVersion: strings.TrimSpace(r.URL.Query().Get("feature_version")), TransitionType: strings.TrimSpace(r.URL.Query().Get("transition_type")),
			QualityState: strings.TrimSpace(r.URL.Query().Get("quality_state")), SessionStart: start, SessionEnd: end,
			Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps state transitions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"transitions": marketOpsStateTransitionResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/evidence", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		records, err := repo.ListMarketOpsEvidence(r.Context(), storage.MarketOpsEvidenceFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			AssetID: strings.TrimSpace(r.URL.Query().Get("asset_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")),
			EvidenceType: strings.TrimSpace(r.URL.Query().Get("evidence_type")), EvidenceVersion: strings.TrimSpace(r.URL.Query().Get("evidence_version")),
			Domain: strings.TrimSpace(r.URL.Query().Get("domain")), Direction: strings.TrimSpace(r.URL.Query().Get("direction")),
			SessionStart: start, SessionEnd: end, Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps evidence")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"evidence": marketOpsEvidenceResponses(records)})
	})

	mux.HandleFunc("GET /v1/marketops/evidence/{evidence_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		record, err := repo.GetMarketOpsEvidence(r.Context(), r.PathValue("evidence_id"))
		if err != nil {
			writeQueryError(w, err, "evidence_not_found", "MarketOps evidence not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"evidence": marketOpsEvidenceResponse(record)})
	})
}

type marketOpsFeatureDefinitionDTO struct {
	TenantID        string          `json:"tenant_id"`
	FeatureKey      string          `json:"feature_key"`
	FeatureVersion  string          `json:"feature_version"`
	Domain          string          `json:"domain"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	ValueType       string          `json:"value_type"`
	Unit            string          `json:"unit,omitempty"`
	CalculationSpec json.RawMessage `json:"calculation_spec"`
	RequiredInputs  json.RawMessage `json:"required_inputs"`
	QualityPolicy   json.RawMessage `json:"quality_policy"`
	Status          string          `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type marketOpsFeatureObservationDTO struct {
	FeatureObservationID string          `json:"feature_observation_id"`
	TenantID             string          `json:"tenant_id"`
	AppID                string          `json:"app_id"`
	AssetID              string          `json:"asset_id"`
	Symbol               string          `json:"symbol"`
	SessionDate          time.Time       `json:"session_date"`
	AsOfTime             time.Time       `json:"as_of_time"`
	FeatureKey           string          `json:"feature_key"`
	FeatureVersion       string          `json:"feature_version"`
	Dimensions           json.RawMessage `json:"dimensions"`
	NumericValue         *float64        `json:"numeric_value,omitempty"`
	TextValue            *string         `json:"text_value,omitempty"`
	BooleanValue         *bool           `json:"boolean_value,omitempty"`
	QualityState         string          `json:"quality_state"`
	QualityScore         *float64        `json:"quality_score,omitempty"`
	QualityDetails       json.RawMessage `json:"quality_details"`
	SourceEventIDs       []string        `json:"source_event_ids"`
	SourceArtifactIDs    []string        `json:"source_artifact_ids"`
	CalculationRunID     string          `json:"calculation_run_id"`
	DeterministicKey     string          `json:"deterministic_key"`
	CreatedAt            time.Time       `json:"created_at"`
}

type marketOpsMarketStateDTO struct {
	MarketStateID         string          `json:"market_state_id"`
	TenantID              string          `json:"tenant_id"`
	AppID                 string          `json:"app_id"`
	AssetID               string          `json:"asset_id"`
	Symbol                string          `json:"symbol"`
	SessionDate           time.Time       `json:"session_date"`
	AsOfTime              time.Time       `json:"as_of_time"`
	StateSchemaVersion    string          `json:"state_schema_version"`
	StatePayload          json.RawMessage `json:"state_payload"`
	FeatureObservationIDs []string        `json:"feature_observation_ids"`
	FeatureCount          int             `json:"feature_count"`
	RequiredFeatureCount  int             `json:"required_feature_count"`
	CompletenessRatio     float64         `json:"completeness_ratio"`
	QualityState          string          `json:"quality_state"`
	QualityScore          *float64        `json:"quality_score,omitempty"`
	QualitySummary        json.RawMessage `json:"quality_summary"`
	EligibleHypotheses    []string        `json:"eligible_hypotheses"`
	BuildRunID            string          `json:"build_run_id"`
	DeterministicKey      string          `json:"deterministic_key"`
	CreatedAt             time.Time       `json:"created_at"`
}

type marketOpsStateTransitionDTO struct {
	TransitionID        string          `json:"transition_id"`
	TenantID            string          `json:"tenant_id"`
	AppID               string          `json:"app_id"`
	AssetID             string          `json:"asset_id"`
	Symbol              string          `json:"symbol"`
	SessionDate         time.Time       `json:"session_date"`
	AsOfTime            time.Time       `json:"as_of_time"`
	CurrentStateID      string          `json:"current_state_id"`
	BaselineStateID     string          `json:"baseline_state_id,omitempty"`
	FeatureKey          string          `json:"feature_key"`
	FeatureVersion      string          `json:"feature_version"`
	Dimensions          json.RawMessage `json:"dimensions"`
	TransitionType      string          `json:"transition_type"`
	LookbackSessions    *int            `json:"lookback_sessions,omitempty"`
	CurrentValue        *float64        `json:"current_value,omitempty"`
	BaselineValue       *float64        `json:"baseline_value,omitempty"`
	TransitionValue     *float64        `json:"transition_value,omitempty"`
	ZScore              *float64        `json:"zscore,omitempty"`
	Percentile          *float64        `json:"percentile,omitempty"`
	PersistenceSessions *int            `json:"persistence_sessions,omitempty"`
	Direction           string          `json:"direction,omitempty"`
	QualityState        string          `json:"quality_state"`
	TransitionPayload   json.RawMessage `json:"transition_payload"`
	CalculationRunID    string          `json:"calculation_run_id"`
	DeterministicKey    string          `json:"deterministic_key"`
	CreatedAt           time.Time       `json:"created_at"`
}

type marketOpsEvidenceDTO struct {
	EvidenceID          string          `json:"evidence_id"`
	TenantID            string          `json:"tenant_id"`
	AppID               string          `json:"app_id"`
	AssetID             string          `json:"asset_id"`
	Symbol              string          `json:"symbol"`
	SessionDate         time.Time       `json:"session_date"`
	AsOfTime            time.Time       `json:"as_of_time"`
	EvidenceType        string          `json:"evidence_type"`
	EvidenceVersion     string          `json:"evidence_version"`
	Domain              string          `json:"domain"`
	Direction           string          `json:"direction,omitempty"`
	Magnitude           *float64        `json:"magnitude,omitempty"`
	RarityScore         *float64        `json:"rarity_score,omitempty"`
	PersistenceScore    *float64        `json:"persistence_score,omitempty"`
	QualityScore        *float64        `json:"quality_score,omitempty"`
	Statement           string          `json:"statement"`
	EvidencePayload     json.RawMessage `json:"evidence_payload"`
	SourceFeatureIDs    []string        `json:"source_feature_ids"`
	SourceTransitionIDs []string        `json:"source_transition_ids"`
	DeterministicKey    string          `json:"deterministic_key"`
	CreatedAt           time.Time       `json:"created_at"`
}

type marketOpsMarketStateLineageDTO struct {
	MarketState                  marketOpsMarketStateDTO          `json:"market_state"`
	FeatureObservations          []marketOpsFeatureObservationDTO `json:"feature_observations"`
	SourceEventIDs               []string                         `json:"source_event_ids"`
	SourceArtifactIDs            []string                         `json:"source_artifact_ids"`
	MissingFeatureObservationIDs []string                         `json:"missing_feature_observation_ids"`
}

func marketOpsFeatureDefinitionResponse(record storage.MarketOpsFeatureDefinitionRecord) marketOpsFeatureDefinitionDTO {
	return marketOpsFeatureDefinitionDTO{TenantID: record.TenantID, FeatureKey: record.FeatureKey, FeatureVersion: record.FeatureVersion,
		Domain: record.Domain, Title: record.Title, Description: record.Description, ValueType: record.ValueType, Unit: record.Unit,
		CalculationSpec: json.RawMessage(jsonOrDefault(record.CalculationSpec, `{}`)), RequiredInputs: json.RawMessage(jsonOrDefault(record.RequiredInputs, `[]`)),
		QualityPolicy: json.RawMessage(jsonOrDefault(record.QualityPolicy, `{}`)), Status: record.Status, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}

func marketOpsFeatureDefinitionResponses(records []storage.MarketOpsFeatureDefinitionRecord) []marketOpsFeatureDefinitionDTO {
	responses := make([]marketOpsFeatureDefinitionDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsFeatureDefinitionResponse(record))
	}
	return responses
}

func marketOpsFeatureObservationResponse(record storage.MarketOpsFeatureObservationRecord) marketOpsFeatureObservationDTO {
	return marketOpsFeatureObservationDTO{FeatureObservationID: record.FeatureObservationID, TenantID: record.TenantID, AppID: record.AppID,
		AssetID: record.AssetID, Symbol: record.Symbol, SessionDate: record.SessionDate, AsOfTime: record.AsOfTime,
		FeatureKey: record.FeatureKey, FeatureVersion: record.FeatureVersion, Dimensions: json.RawMessage(jsonOrDefault(record.DimensionsJSON, `{}`)),
		NumericValue: record.NumericValue, TextValue: record.TextValue, BooleanValue: record.BooleanValue,
		QualityState: record.QualityState, QualityScore: record.QualityScore, QualityDetails: json.RawMessage(jsonOrDefault(record.QualityDetailsJSON, `{}`)),
		SourceEventIDs: nonNilStrings(record.SourceEventIDs), SourceArtifactIDs: nonNilStrings(record.SourceArtifactIDs),
		CalculationRunID: record.CalculationRunID, DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt}
}

func marketOpsFeatureObservationResponses(records []storage.MarketOpsFeatureObservationRecord) []marketOpsFeatureObservationDTO {
	responses := make([]marketOpsFeatureObservationDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsFeatureObservationResponse(record))
	}
	return responses
}

func marketOpsMarketStateResponse(record storage.MarketOpsMarketStateRecord) marketOpsMarketStateDTO {
	return marketOpsMarketStateDTO{MarketStateID: record.MarketStateID, TenantID: record.TenantID, AppID: record.AppID,
		AssetID: record.AssetID, Symbol: record.Symbol, SessionDate: record.SessionDate, AsOfTime: record.AsOfTime,
		StateSchemaVersion: record.StateSchemaVersion, StatePayload: json.RawMessage(jsonOrDefault(record.StatePayloadJSON, `{}`)),
		FeatureObservationIDs: nonNilStrings(record.FeatureObservationIDs), FeatureCount: record.FeatureCount,
		RequiredFeatureCount: record.RequiredFeatureCount, CompletenessRatio: record.CompletenessRatio,
		QualityState: record.QualityState, QualityScore: record.QualityScore, QualitySummary: json.RawMessage(jsonOrDefault(record.QualitySummaryJSON, `{}`)),
		EligibleHypotheses: nonNilStrings(record.EligibleHypotheses), BuildRunID: record.BuildRunID,
		DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt}
}

func marketOpsMarketStateResponses(records []storage.MarketOpsMarketStateRecord) []marketOpsMarketStateDTO {
	responses := make([]marketOpsMarketStateDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsMarketStateResponse(record))
	}
	return responses
}

func marketOpsStateTransitionResponse(record storage.MarketOpsStateTransitionRecord) marketOpsStateTransitionDTO {
	return marketOpsStateTransitionDTO{TransitionID: record.TransitionID, TenantID: record.TenantID, AppID: record.AppID,
		AssetID: record.AssetID, Symbol: record.Symbol, SessionDate: record.SessionDate, AsOfTime: record.AsOfTime,
		CurrentStateID: record.CurrentStateID, BaselineStateID: record.BaselineStateID, FeatureKey: record.FeatureKey,
		FeatureVersion: record.FeatureVersion, Dimensions: json.RawMessage(jsonOrDefault(record.DimensionsJSON, `{}`)),
		TransitionType: record.TransitionType, LookbackSessions: record.LookbackSessions, CurrentValue: record.CurrentValue,
		BaselineValue: record.BaselineValue, TransitionValue: record.TransitionValue, ZScore: record.ZScore, Percentile: record.Percentile,
		PersistenceSessions: record.PersistenceSessions, Direction: record.Direction, QualityState: record.QualityState,
		TransitionPayload: json.RawMessage(jsonOrDefault(record.TransitionPayloadJSON, `{}`)), CalculationRunID: record.CalculationRunID,
		DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt}
}

func marketOpsStateTransitionResponses(records []storage.MarketOpsStateTransitionRecord) []marketOpsStateTransitionDTO {
	responses := make([]marketOpsStateTransitionDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsStateTransitionResponse(record))
	}
	return responses
}

func marketOpsEvidenceResponse(record storage.MarketOpsEvidenceRecord) marketOpsEvidenceDTO {
	return marketOpsEvidenceDTO{EvidenceID: record.EvidenceID, TenantID: record.TenantID, AppID: record.AppID,
		AssetID: record.AssetID, Symbol: record.Symbol, SessionDate: record.SessionDate, AsOfTime: record.AsOfTime,
		EvidenceType: record.EvidenceType, EvidenceVersion: record.EvidenceVersion, Domain: record.Domain, Direction: record.Direction,
		Magnitude: record.Magnitude, RarityScore: record.RarityScore, PersistenceScore: record.PersistenceScore, QualityScore: record.QualityScore,
		Statement: record.Statement, EvidencePayload: json.RawMessage(jsonOrDefault(record.EvidencePayloadJSON, `{}`)),
		SourceFeatureIDs: nonNilStrings(record.SourceFeatureIDs), SourceTransitionIDs: nonNilStrings(record.SourceTransitionIDs),
		DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt}
}

func marketOpsEvidenceResponses(records []storage.MarketOpsEvidenceRecord) []marketOpsEvidenceDTO {
	responses := make([]marketOpsEvidenceDTO, 0, len(records))
	for _, record := range records {
		responses = append(responses, marketOpsEvidenceResponse(record))
	}
	return responses
}

func resolveMarketOpsStateLineage(r *http.Request, repo storage.QueryRepository, state storage.MarketOpsMarketStateRecord) (marketOpsMarketStateLineageDTO, error) {
	byID := make(map[string]storage.MarketOpsFeatureObservationRecord, len(state.FeatureObservationIDs))
	for start := 0; start < len(state.FeatureObservationIDs); start += 200 {
		end := start + 200
		if end > len(state.FeatureObservationIDs) {
			end = len(state.FeatureObservationIDs)
		}
		records, err := repo.ListMarketOpsFeatureObservations(r.Context(), storage.MarketOpsFeatureObservationFilter{
			TenantID: state.TenantID, AppID: state.AppID, FeatureObservationIDs: state.FeatureObservationIDs[start:end], Limit: end - start,
		})
		if err != nil {
			return marketOpsMarketStateLineageDTO{}, err
		}
		for _, record := range records {
			byID[record.FeatureObservationID] = record
		}
	}

	ordered := make([]storage.MarketOpsFeatureObservationRecord, 0, len(byID))
	missing := []string{}
	events, artifacts := []string{}, []string{}
	seenEvents, seenArtifacts := map[string]struct{}{}, map[string]struct{}{}
	for _, id := range state.FeatureObservationIDs {
		record, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		ordered = append(ordered, record)
		for _, eventID := range record.SourceEventIDs {
			if _, seen := seenEvents[eventID]; !seen {
				seenEvents[eventID] = struct{}{}
				events = append(events, eventID)
			}
		}
		for _, artifactID := range record.SourceArtifactIDs {
			if _, seen := seenArtifacts[artifactID]; !seen {
				seenArtifacts[artifactID] = struct{}{}
				artifacts = append(artifacts, artifactID)
			}
		}
	}
	return marketOpsMarketStateLineageDTO{MarketState: marketOpsMarketStateResponse(state),
		FeatureObservations: marketOpsFeatureObservationResponses(ordered), SourceEventIDs: events,
		SourceArtifactIDs: artifacts, MissingFeatureObservationIDs: missing}, nil
}

func marketOpsDimensionsQuery(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	raw := []byte(strings.TrimSpace(r.URL.Query().Get("dimensions")))
	if len(raw) == 0 {
		return []byte(`{}`), true
	}
	var dimensions map[string]any
	if err := json.Unmarshal(raw, &dimensions); err != nil || dimensions == nil {
		writeError(w, http.StatusBadRequest, "invalid_dimensions", "dimensions must be a JSON object")
		return nil, false
	}
	return raw, true
}

func marketOpsSessionRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	start, err := parseMarketOpsSessionDate(r.URL.Query().Get("session_start"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_session_start", err.Error())
		return time.Time{}, time.Time{}, false
	}
	end, err := parseMarketOpsSessionDate(r.URL.Query().Get("session_end"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_session_end", err.Error())
		return time.Time{}, time.Time{}, false
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		writeError(w, http.StatusBadRequest, "invalid_session_range", "session_end must be on or after session_start")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func parseMarketOpsSessionDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, &marketOpsDateError{value: value}
	}
	return parsed.UTC(), nil
}

type marketOpsDateError struct{ value string }

func (e *marketOpsDateError) Error() string { return "session date must use YYYY-MM-DD: " + e.value }

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
