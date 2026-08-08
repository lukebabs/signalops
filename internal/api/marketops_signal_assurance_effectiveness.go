package api

import (
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
)

func registerMarketOpsSignalAssuranceEffectivenessRoutes(mux *http.ServeMux, repository storage.QueryRepository) {
	query, ok := repository.(storage.SignalAssuranceEffectivenessRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/marketops/signal-assurance/effectiveness", func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		rows, err := query.ListSignalAssuranceEffectiveness(r.Context(), storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode"))), Dimension: strings.TrimSpace(r.URL.Query().Get("dimension")), Limit: queryLimit(r, 100)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to calculate signal assurance effectiveness")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"effectiveness": effectivenessResponses(rows), "minimum_ranked_sample": 30, "evidence_source_note": "LEGACY records are historical outcome evidence and are not SAF-validated assertions."})
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/effectiveness/observations", func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		dimensionValue := strings.TrimSpace(r.URL.Query().Get("dimension_value"))
		if tenantID == "" || dimensionValue == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id and dimension_value are required")
			return
		}
		rows, err := query.ListSignalAssuranceEffectivenessObservations(r.Context(), storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode"))), Dimension: strings.TrimSpace(r.URL.Query().Get("dimension")), DimensionValue: dimensionValue, Limit: queryLimit(r, 200)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list signal assurance effectiveness observations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"observations": effectivenessObservationResponses(rows), "evidence_source_note": "LEGACY observations are historical outcome evidence and are not SAF-validated assertions."})
	})
	mux.HandleFunc("GET /v1/marketops/signal-assurance/recommendations", func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "missing_query", "tenant_id is required")
			return
		}
		rows, err := query.ListSignalAssuranceRecommendations(r.Context(), storage.SignalAssuranceEffectivenessFilter{TenantID: tenantID, EvidenceSource: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evidence_source"))), EvaluationMode: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("evaluation_mode")))})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to calculate signal assurance recommendations")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"recommendations": recommendationResponses(rows), "minimum_ranked_sample": 30})
	})
}

func effectivenessObservationResponses(values []storage.SignalAssuranceEffectivenessObservationRecord) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, x := range values {
		item := map[string]any{"evidence_source": x.EvidenceSource, "observation_id": x.ObservationID, "reference_id": x.ReferenceID, "symbol": x.Symbol, "signal_type": x.SignalType, "direction": x.Direction, "algorithm": x.Algorithm, "algorithm_version": x.AlgorithmVersion, "state": x.State, "evaluation_mode": x.EvaluationMode, "horizon_sessions": x.HorizonSessions, "signal_score": x.SignalScore, "confidence": x.Confidence, "directional_hit": x.DirectionalHit, "absolute_return": x.AbsoluteReturn, "directional_return": x.DirectionalReturn, "relative_return": x.RelativeReturn, "mfe": x.MFE, "mae": x.MAE, "calculation_version": x.CalculationVersion, "calculation_run_id": x.CalculationRunID}
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
	EvidenceSource          string   `json:"evidence_source"`
	Dimension               string   `json:"dimension"`
	DimensionValue          string   `json:"dimension_value"`
	SampleSize              int      `json:"sample_size"`
	DirectionalHits         int      `json:"directional_hits"`
	MaterializedCount       int      `json:"materialized_count"`
	InvalidatedCount        int      `json:"invalidated_count"`
	ExpiredCount            int      `json:"expired_count"`
	CensoredCount           int      `json:"censored_count"`
	ExcludedCount           int      `json:"excluded_count"`
	DirectionalAccuracy     *float64 `json:"directional_accuracy,omitempty"`
	AccuracyLowerBound      *float64 `json:"accuracy_lower_bound,omitempty"`
	AccuracyUpperBound      *float64 `json:"accuracy_upper_bound,omitempty"`
	MaterializationRate     *float64 `json:"materialization_rate,omitempty"`
	AverageReturn           *float64 `json:"average_return,omitempty"`
	AverageRelativeReturn   *float64 `json:"average_relative_return,omitempty"`
	AverageMFE              *float64 `json:"average_mfe,omitempty"`
	AverageMAE              *float64 `json:"average_mae,omitempty"`
	Exploratory             bool     `json:"exploratory"`
	AsOf                    string   `json:"as_of"`
	MetricDefinitionVersion string   `json:"metric_definition_version"`
}

func effectivenessResponses(values []storage.SignalAssuranceEffectivenessRecord) []signalAssuranceEffectivenessDTO {
	out := make([]signalAssuranceEffectivenessDTO, 0, len(values))
	for _, x := range values {
		out = append(out, signalAssuranceEffectivenessDTO{EvidenceSource: x.EvidenceSource, Dimension: x.Dimension, DimensionValue: x.DimensionValue, SampleSize: x.SampleSize, DirectionalHits: x.DirectionalHits, MaterializedCount: x.MaterializedCount, InvalidatedCount: x.InvalidatedCount, ExpiredCount: x.ExpiredCount, CensoredCount: x.CensoredCount, ExcludedCount: x.ExcludedCount, DirectionalAccuracy: x.DirectionalAccuracy, AccuracyLowerBound: x.AccuracyLowerBound, AccuracyUpperBound: x.AccuracyUpperBound, MaterializationRate: x.MaterializationRate, AverageReturn: x.AverageReturn, AverageRelativeReturn: x.AverageRelativeReturn, AverageMFE: x.AverageMFE, AverageMAE: x.AverageMAE, Exploratory: x.Exploratory, AsOf: x.AsOf.UTC().Format("2006-01-02T15:04:05Z"), MetricDefinitionVersion: x.MetricDefinitionVersion})
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
