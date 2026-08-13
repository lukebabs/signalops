package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

const valuationCompositeAlgorithmID = "signalops.algorithms.valuation_composite_v3"
const dosmAlgorithmID = "signalops.algorithms.distressed_opportunity_scoring_v3"
const tacticalValuationCompositeAlgorithmID = "signalops.algorithms.tactical_valuation_composite_v1"
const tacticalDOSMAlgorithmID = "signalops.algorithms.tactical_distressed_opportunity_v1"
const tacticalPostureAlgorithmID = "signalops.algorithms.tactical_market_posture_v1"

func registerMarketOpsValuationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/valuation", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := any(cfg.QueryRepository).(storage.MarketOpsValuationRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "valuation_unavailable", "valuation results are unavailable")
			return
		}
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		watchlistContext, ok := requireSubscriberWatchlistContext(w, r, cfg, tenant)
		if !ok {
			return
		}
		results, err := repo.ListMarketOpsValuationResults(r.Context(), storage.MarketOpsValuationFilter{TenantID: tenant, Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), EligibleOnly: strings.EqualFold(r.URL.Query().Get("eligible_only"), "true"), Limit: 200})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list valuation results")
			return
		}
		if subscriberWatchlistContextEnabled(cfg, tenant) {
			visible := results[:0]
			for _, result := range results {
				if _, allowed := watchlistContext.Tickers[strings.ToUpper(result.Symbol)]; allowed {
					visible = append(visible, result)
				}
			}
			results = visible
		}
		response := map[string]any{"results": valuationRows(results), "research_only": true}
		if subscriberWatchlistContextEnabled(cfg, tenant) {
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
		}
		writeJSON(w, http.StatusOK, response)
	})
}

func valuationRows(results []storage.MarketOpsValuationResultRecord) []map[string]any {
	type pair struct {
		vc, dosm, tactical, tacticalVC, tacticalDOSM *storage.MarketOpsValuationResultRecord
	}
	byKey := map[string]*pair{}
	for i := range results {
		item := &results[i]
		key := item.Symbol
		if byKey[key] == nil {
			byKey[key] = &pair{}
		}
		if item.AlgorithmID == valuationCompositeAlgorithmID {
			byKey[key].vc = item
		}
		if item.AlgorithmID == dosmAlgorithmID {
			byKey[key].dosm = item
		}
		if item.AlgorithmID == tacticalPostureAlgorithmID {
			byKey[key].tactical = item
		}
		if item.AlgorithmID == tacticalValuationCompositeAlgorithmID {
			byKey[key].tacticalVC = item
		}
		if item.AlgorithmID == tacticalDOSMAlgorithmID {
			byKey[key].tacticalDOSM = item
		}
	}
	rows := []map[string]any{}
	for _, item := range byKey {
		primary := item.dosm
		if primary == nil {
			primary = item.vc
		}
		if primary == nil {
			continue
		}
		primaryOutput := valuationOutput(*primary)
		trace, _ := primaryOutput["trace"].(map[string]any)
		row := map[string]any{"ticker": primary.Symbol, "trade_date": primary.SessionDate.Format("2006-01-02"), "eligible": primary.Eligible, "evaluation_status": primary.EvaluationStatus, "confidence": primary.Confidence, "confidence_label": primary.ConfidenceLabel, "model_version": primary.ModelVersion, "data_profile": trace["data_profile"], "growth_status": trace["growth_status"]}
		if item.vc != nil {
			row["vc"] = valuationOutput(*item.vc)
		}
		if item.dosm != nil {
			row["dosm"] = valuationOutput(*item.dosm)
		}
		if item.tactical != nil {
			row["tactical"] = tacticalPostureOutput(*item.tactical)
		}
		if item.tacticalVC != nil {
			row["tactical_vc"] = valuationOutput(*item.tacticalVC)
		}
		if item.tacticalDOSM != nil {
			row["tactical_dosm"] = valuationOutput(*item.tacticalDOSM)
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		left, _ := rows[i]["dosm"].(map[string]any)
		right, _ := rows[j]["dosm"].(map[string]any)
		ls, _ := left["score"].(float64)
		rs, _ := right["score"].(float64)
		return ls > rs
	})
	return rows
}

func valuationOutput(item storage.MarketOpsValuationResultRecord) map[string]any {
	trace := map[string]any{}
	_ = json.Unmarshal(item.ResultJSON, &trace)
	return map[string]any{"score": item.Score, "fair_value": item.FairValue, "classification": item.Classification, "algorithm_id": item.AlgorithmID, "trace": trace}
}

func tacticalPostureOutput(item storage.MarketOpsValuationResultRecord) map[string]any {
	trace := map[string]any{}
	_ = json.Unmarshal(item.ResultJSON, &trace)
	return map[string]any{"posture": item.Classification, "overlay": item.Score, "algorithm_id": item.AlgorithmID, "session_date": item.SessionDate.Format("2006-01-02"), "technical_components": trace["technical_components"], "feature_observation_ids": trace["feature_observation_ids"]}
}
