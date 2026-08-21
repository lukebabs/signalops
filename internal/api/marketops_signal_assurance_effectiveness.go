package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/lukebabs/signalops/internal/marketops/signalassurance"
	"github.com/lukebabs/signalops/internal/storage"
	"github.com/lukebabs/signalops/internal/subscriber/eodrevisionpolicy"
)

func signalAssuranceObservationLimit(r *http.Request) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return 200
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 200
	}
	if parsed > 2000 {
		return 2000
	}
	return parsed
}

func registerMarketOpsSignalAssuranceEffectivenessRoutes(mux *http.ServeMux, cfg RouterConfig) {
	query, ok := cfg.QueryRepository.(storage.SignalAssuranceEffectivenessRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/marketops/signal-assurance/effectiveness", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		filter := storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode"))), Dimension: strings.TrimSpace(r.URL.Query().Get("dimension")), Limit: queryLimit(r, 100)}
		watchlistContext, global, allowed := subscriberGlobalSignalAssuranceContext(w, r, cfg, tenantID)
		if !allowed {
			return
		}
		var rows []storage.SignalAssuranceEffectivenessRecord
		var err error
		if global {
			reader, supported := cfg.QueryRepository.(storage.SubscriberGlobalSignalAssuranceEffectivenessRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "global_signal_assurance_unavailable", "global Signal Assurance projection is unavailable")
				return
			}
			rows, err = reader.ListSubscriberGlobalSignalAssuranceEffectiveness(r.Context(), authorizedEROCTickers(watchlistContext, ""), filter)
		} else {
			rows, err = query.ListSignalAssuranceEffectiveness(r.Context(), filter)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to calculate signal assurance effectiveness")
			return
		}
		response := map[string]any{"effectiveness": effectivenessResponses(rows), "minimum_ranked_sample": 30, "evidence_source_note": "LEGACY records are historical outcome evidence and are not SAF-validated assertions.", "data_selection": historicalAssuranceDataSelection()}
		if global {
			response["data_scope"] = "platform-global"
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
			response["evidence_source_note"] = "SAF has no confirmed global assertions yet. Historical outcome evidence is platform-global and filtered to the selected watchlist."
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/effectiveness/observations", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		dimensionValue := strings.TrimSpace(r.URL.Query().Get("dimension_value"))
		if tenantID == "" || dimensionValue == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id and dimension_value are required")
			return
		}
		filter := storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode"))), Dimension: strings.TrimSpace(r.URL.Query().Get("dimension")), DimensionValue: dimensionValue, Limit: signalAssuranceObservationLimit(r)}
		watchlistContext, global, allowed := subscriberGlobalSignalAssuranceContext(w, r, cfg, tenantID)
		if !allowed {
			return
		}
		var rows []storage.SignalAssuranceEffectivenessObservationRecord
		var err error
		if global {
			reader, supported := cfg.QueryRepository.(storage.SubscriberGlobalSignalAssuranceEffectivenessRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "global_signal_assurance_unavailable", "global Signal Assurance projection is unavailable")
				return
			}
			rows, err = reader.ListSubscriberGlobalSignalAssuranceEffectivenessObservations(r.Context(), authorizedEROCTickers(watchlistContext, ""), filter)
		} else {
			rows, err = query.ListSignalAssuranceEffectivenessObservations(r.Context(), filter)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signal assurance effectiveness observations")
			return
		}
		response := map[string]any{"observations": effectivenessObservationResponses(rows), "evidence_source_note": "LEGACY observations are historical outcome evidence and are not SAF-validated assertions.", "data_selection": historicalAssuranceDataSelection()}
		if global {
			response["data_scope"] = "platform-global"
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
			response["evidence_source_note"] = "SAF has no confirmed global assertions yet. Historical outcome evidence is platform-global and filtered to the selected watchlist."
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/recommendations", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.URL.Query().Get("tenant_id"))
		if !ok {
			return
		}
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		filter := storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode")))}
		watchlistContext, global, allowed := subscriberGlobalSignalAssuranceContext(w, r, cfg, tenantID)
		if !allowed {
			return
		}
		var rows []storage.SignalAssuranceRecommendationRecord
		var err error
		if global {
			reader, supported := cfg.QueryRepository.(storage.SubscriberGlobalSignalAssuranceEffectivenessRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "global_signal_assurance_unavailable", "global Signal Assurance projection is unavailable")
				return
			}
			rows, err = reader.ListSubscriberGlobalSignalAssuranceRecommendations(r.Context(), authorizedEROCTickers(watchlistContext, ""), filter)
		} else {
			rows, err = query.ListSignalAssuranceRecommendations(r.Context(), filter)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to calculate signal assurance recommendations")
			return
		}
		response := map[string]any{"recommendations": recommendationResponses(rows), "minimum_ranked_sample": 30, "data_selection": historicalAssuranceDataSelection()}
		if global {
			response["data_scope"] = "platform-global"
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
		}
		writeJSON(w, http.StatusOK, response)
	})
}

func subscriberGlobalSignalAssuranceContext(w http.ResponseWriter, r *http.Request, cfg RouterConfig, tenantID string) (subscriberWatchlistContext, bool, bool) {
	if !subscriberWatchlistContextEnabled(cfg, tenantID) {
		return subscriberWatchlistContext{}, false, true
	}
	context, ok := requireSubscriberWatchlistContext(w, r, cfg, tenantID)
	return context, true, ok
}

// historicalAssuranceDataSelection exposes the immutable point-in-time EOD contract used by SAF outcomes.
func historicalAssuranceDataSelection() map[string]any {
	selection, err := eodrevisionpolicy.SelectionFor(eodrevisionpolicy.HistoricalAssurance)
	if err != nil {
		return map[string]any{"usage_context": string(eodrevisionpolicy.HistoricalAssurance)}
	}
	return map[string]any{
		"usage_context":             string(selection.UsageContext),
		"selected_observation_role": string(selection.SelectedObservationRole),
		"policy_version":            selection.PolicyVersion,
		"as_of_policy":              "initial_capture",
		"restatement":               "disabled",
	}
}

func effectivenessObservationResponses(values []storage.SignalAssuranceEffectivenessObservationRecord) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, x := range values {
		item := map[string]any{"evidence_source": x.EvidenceSource, "observation_id": x.ObservationID, "reference_id": x.ReferenceID, "symbol": x.Symbol, "signal_type": x.SignalType, "direction": x.Direction, "algorithm": x.Algorithm, "algorithm_version": x.AlgorithmVersion, "state": x.State, "evaluation_mode": x.EvaluationMode, "horizon_sessions": x.HorizonSessions, "signal_score": x.SignalScore, "confidence": x.Confidence, "directional_hit": x.DirectionalHit, "absolute_return": x.AbsoluteReturn, "directional_return": x.DirectionalReturn, "relative_return": x.RelativeReturn, "sector_relative_return": x.SectorRelativeReturn, "mfe": x.MFE, "mae": x.MAE, "calculation_version": x.CalculationVersion, "calculation_run_id": x.CalculationRunID, "broad_market_benchmark_state": x.BroadMarketBenchmarkState, "sector_benchmark_state": x.SectorBenchmarkState}
		if x.OriginAt != nil {
			item["origin_at"] = x.OriginAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		if x.OutcomeAt != nil {
			item["outcome_at"] = x.OutcomeAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, item)
	}
	return out
}

type signalAssuranceEffectivenessDTO struct {
	EvidenceSource                 string   `json:"evidence_source"`
	Dimension                      string   `json:"dimension"`
	DimensionValue                 string   `json:"dimension_value"`
	SampleSize                     int      `json:"sample_size"`
	DirectionalHits                int      `json:"directional_hits"`
	MaterializedCount              int      `json:"materialized_count"`
	InvalidatedCount               int      `json:"invalidated_count"`
	ExpiredCount                   int      `json:"expired_count"`
	CensoredCount                  int      `json:"censored_count"`
	ExcludedCount                  int      `json:"excluded_count"`
	DirectionalAccuracy            *float64 `json:"directional_accuracy,omitempty"`
	AccuracyLowerBound             *float64 `json:"accuracy_lower_bound,omitempty"`
	AccuracyUpperBound             *float64 `json:"accuracy_upper_bound,omitempty"`
	MaterializationRate            *float64 `json:"materialization_rate,omitempty"`
	AverageReturn                  *float64 `json:"average_return,omitempty"`
	AverageRelativeReturn          *float64 `json:"average_relative_return,omitempty"`
	AverageSectorRelativeReturn    *float64 `json:"average_sector_relative_return,omitempty"`
	BroadMarketBenchmarkSampleSize int      `json:"broad_market_benchmark_sample_size"`
	SectorBenchmarkSampleSize      int      `json:"sector_benchmark_sample_size"`
	AverageMFE                     *float64 `json:"average_mfe,omitempty"`
	AverageMAE                     *float64 `json:"average_mae,omitempty"`
	Exploratory                    bool     `json:"exploratory"`
	AsOf                           string   `json:"as_of"`
	ViabilityState                 string   `json:"viability_state"`
	ViabilityReasons               []string `json:"viability_reasons"`
	ViabilityPolicyVersion         string   `json:"viability_policy_version"`
	MetricDefinitionVersion        string   `json:"metric_definition_version"`
}

func effectivenessResponses(values []storage.SignalAssuranceEffectivenessRecord) []signalAssuranceEffectivenessDTO {
	out := make([]signalAssuranceEffectivenessDTO, 0, len(values))
	for _, x := range values {
		assessment := signalassurance.AssessViability(x)
		out = append(out, signalAssuranceEffectivenessDTO{EvidenceSource: x.EvidenceSource, Dimension: x.Dimension, DimensionValue: x.DimensionValue, SampleSize: x.SampleSize, DirectionalHits: x.DirectionalHits, MaterializedCount: x.MaterializedCount, InvalidatedCount: x.InvalidatedCount, ExpiredCount: x.ExpiredCount, CensoredCount: x.CensoredCount, ExcludedCount: x.ExcludedCount, DirectionalAccuracy: x.DirectionalAccuracy, AccuracyLowerBound: x.AccuracyLowerBound, AccuracyUpperBound: x.AccuracyUpperBound, MaterializationRate: x.MaterializationRate, AverageReturn: x.AverageReturn, AverageRelativeReturn: x.AverageRelativeReturn, AverageSectorRelativeReturn: x.AverageSectorRelativeReturn, BroadMarketBenchmarkSampleSize: x.BroadMarketBenchmarkSampleSize, SectorBenchmarkSampleSize: x.SectorBenchmarkSampleSize, AverageMFE: x.AverageMFE, AverageMAE: x.AverageMAE, Exploratory: x.Exploratory, ViabilityState: assessment.State, ViabilityReasons: assessment.Reasons, ViabilityPolicyVersion: signalassurance.ViabilityPolicyVersion, AsOf: x.AsOf.UTC().Format("2006-01-02T15:04:05Z"), MetricDefinitionVersion: x.MetricDefinitionVersion})
	}
	return out
}

type signalAssuranceRecommendationDTO struct {
	RecommendationID        string   `json:"recommendation_id"`
	EvidenceSource          string   `json:"evidence_source"`
	Dimension               string   `json:"dimension"`
	DimensionValue          string   `json:"dimension_value"`
	Priority                string   `json:"priority"`
	Kind                    string   `json:"kind"`
	Summary                 string   `json:"summary"`
	SampleSize              int      `json:"sample_size"`
	DirectionalAccuracy     *float64 `json:"directional_accuracy,omitempty"`
	AccuracyUpperBound      *float64 `json:"accuracy_upper_bound,omitempty"`
	MetricDefinitionVersion string   `json:"metric_definition_version"`
	AsOf                    string   `json:"as_of"`
}

func recommendationResponses(values []storage.SignalAssuranceRecommendationRecord) []signalAssuranceRecommendationDTO {
	out := make([]signalAssuranceRecommendationDTO, 0, len(values))
	for _, x := range values {
		out = append(out, signalAssuranceRecommendationDTO{RecommendationID: x.RecommendationID, EvidenceSource: x.EvidenceSource, Dimension: x.Dimension, DimensionValue: x.DimensionValue, Priority: x.Priority, Kind: x.Kind, Summary: x.Summary, SampleSize: x.SampleSize, DirectionalAccuracy: x.DirectionalAccuracy, AccuracyUpperBound: x.AccuracyUpperBound, MetricDefinitionVersion: x.MetricDefinitionVersion, AsOf: x.AsOf.UTC().Format("2006-01-02T15:04:05Z")})
	}
	return out
}
