package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerMarketOpsOutcomeRoutes(mux *http.ServeMux, queryRepository storage.QueryRepository) {
	mux.HandleFunc("GET /v1/marketops/outcomes", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		start, end, ok := marketOpsSessionRange(w, r)
		if !ok {
			return
		}
		horizon := 0
		if raw := strings.TrimSpace(r.URL.Query().Get("horizon_sessions")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || (value != 1 && value != 5 && value != 10 && value != 20) {
				writeError(w, http.StatusBadRequest, "invalid_horizon", "horizon_sessions must be one of 1, 5, 10, or 20")
				return
			}
			horizon = value
		}
		records, err := repo.ListMarketOpsSignalOutcomes(r.Context(), storage.MarketOpsSignalOutcomeFilter{
			TenantID: strings.TrimSpace(r.URL.Query().Get("tenant_id")), AppID: strings.TrimSpace(r.URL.Query().Get("app_id")),
			SourceType: strings.TrimSpace(r.URL.Query().Get("source_type")), SourceID: strings.TrimSpace(r.URL.Query().Get("source_id")),
			HypothesisKey: strings.TrimSpace(r.URL.Query().Get("hypothesis_key")), HypothesisVersion: strings.TrimSpace(r.URL.Query().Get("hypothesis_version")),
			Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), Direction: strings.TrimSpace(r.URL.Query().Get("direction")),
			OutcomeStatus: strings.TrimSpace(r.URL.Query().Get("outcome_status")), HorizonSessions: horizon,
			OriginStart: start, OriginEnd: end, Limit: queryLimit(r, 50),
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps outcomes")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"outcomes": marketOpsOutcomeResponses(records)})
	})
	mux.HandleFunc("GET /v1/marketops/outcomes/{outcome_id}", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := requireQueryRepository(w, queryRepository)
		if !ok {
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		record, err := repo.GetMarketOpsSignalOutcome(r.Context(), tenantID, r.PathValue("outcome_id"))
		if err != nil {
			writeQueryError(w, err, "outcome_not_found", "MarketOps outcome not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"outcome": marketOpsOutcomeResponse(record)})
	})
}

type marketOpsOutcomeDTO struct {
	OutcomeID             string          `json:"outcome_id"`
	TenantID              string          `json:"tenant_id"`
	AppID                 string          `json:"app_id"`
	SourceType            string          `json:"source_type"`
	SourceID              string          `json:"source_id"`
	HypothesisKey         string          `json:"hypothesis_key,omitempty"`
	HypothesisVersion     string          `json:"hypothesis_version,omitempty"`
	AssetID               string          `json:"asset_id"`
	Symbol                string          `json:"symbol"`
	Direction             string          `json:"direction"`
	OriginSessionDate     time.Time       `json:"origin_session_date"`
	HorizonSessions       int             `json:"horizon_sessions"`
	MaturedSessionDate    *time.Time      `json:"matured_session_date,omitempty"`
	OutcomeStatus         string          `json:"outcome_status"`
	ForwardReturn         *float64        `json:"forward_return,omitempty"`
	MaxFavorableExcursion *float64        `json:"max_favorable_excursion,omitempty"`
	MaxAdverseExcursion   *float64        `json:"max_adverse_excursion,omitempty"`
	MaximumDrawdown       *float64        `json:"maximum_drawdown,omitempty"`
	RealizedVolChange     *float64        `json:"realized_vol_change,omitempty"`
	DirectionalHit        *bool           `json:"directional_hit,omitempty"`
	ThresholdHit          *bool           `json:"threshold_hit,omitempty"`
	DaysToThreshold       *int            `json:"days_to_threshold,omitempty"`
	OriginEventID         string          `json:"origin_event_id,omitempty"`
	OutcomeEventIDs       []string        `json:"outcome_event_ids"`
	OutcomePayload        json.RawMessage `json:"outcome_payload"`
	CalculationVersion    string          `json:"calculation_version"`
	CalculationRunID      string          `json:"calculation_run_id"`
	DeterministicKey      string          `json:"deterministic_key"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

func marketOpsOutcomeResponse(record storage.MarketOpsSignalOutcomeRecord) marketOpsOutcomeDTO {
	return marketOpsOutcomeDTO{
		OutcomeID: record.OutcomeID, TenantID: record.TenantID, AppID: record.AppID,
		SourceType: record.SourceType, SourceID: record.SourceID, HypothesisKey: record.HypothesisKey,
		HypothesisVersion: record.HypothesisVersion, AssetID: record.AssetID, Symbol: record.Symbol,
		Direction: record.Direction, OriginSessionDate: record.OriginSessionDate,
		HorizonSessions: record.HorizonSessions, MaturedSessionDate: record.MaturedSessionDate,
		OutcomeStatus: record.OutcomeStatus, ForwardReturn: record.ForwardReturn,
		MaxFavorableExcursion: record.MaxFavorableExcursion, MaxAdverseExcursion: record.MaxAdverseExcursion,
		MaximumDrawdown: record.MaximumDrawdown, RealizedVolChange: record.RealizedVolChange,
		DirectionalHit: record.DirectionalHit, ThresholdHit: record.ThresholdHit,
		DaysToThreshold: record.DaysToThreshold, OriginEventID: record.OriginEventID,
		OutcomeEventIDs:    nonNilStrings(record.OutcomeEventIDs),
		OutcomePayload:     json.RawMessage(jsonOrDefault(record.OutcomePayloadJSON, `{}`)),
		CalculationVersion: record.CalculationVersion, CalculationRunID: record.CalculationRunID,
		DeterministicKey: record.DeterministicKey, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func marketOpsOutcomeResponses(records []storage.MarketOpsSignalOutcomeRecord) []marketOpsOutcomeDTO {
	out := make([]marketOpsOutcomeDTO, 0, len(records))
	for _, record := range records {
		out = append(out, marketOpsOutcomeResponse(record))
	}
	return out
}
