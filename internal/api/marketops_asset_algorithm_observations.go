package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const zscoreAlgorithmID = "signalops.algorithms.zscore_anomaly_v1"
const riskRewardAlgorithmID = "signalops.algorithms.risk_reward_temporal_v1"

var marketOpsPlatformAlgorithmIDs = map[string]struct{}{
	zscoreAlgorithmID:                               {},
	"signalops.algorithms.river_anomaly_v1":         {},
	"signalops.algorithms.ruptures_change_point_v1": {},
	"signalops.algorithms.statsmodels_forecast_v1":  {},
	riskRewardAlgorithmID:                           {},
}

type marketOpsAssetAlgorithmObservationReader interface {
	ListAlgorithmResults(context.Context, storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error)
	ListMarketOpsAssets(context.Context, string, string, bool, int) ([]storage.MarketOpsAssetRecord, error)
}

type marketOpsEODZScoreDTO struct {
	TradeDate       string              `json:"trade_date"`
	AlgorithmResult *algorithmResultDTO `json:"algorithm_result"`
	Status          string              `json:"status"`
	Reason          string              `json:"reason,omitempty"`
}

func registerMarketOpsAssetAlgorithmObservationRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/risk-reward", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(marketOpsAssetAlgorithmObservationReader)
		if !ok {
			writeError(w, http.StatusNotImplemented, "risk_reward_unavailable", "risk/reward summaries are unavailable")
			return
		}
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		if tenant == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id is required")
			return
		}
		universeGroup := strings.TrimSpace(r.URL.Query().Get("universe_group"))
		if universeGroup == "" {
			universeGroup = "top50_megacap"
		}
		assets, err := reader.ListMarketOpsAssets(r.Context(), tenant, universeGroup, true, 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list marketops assets")
			return
		}
		activeSymbols := make(map[string]struct{}, len(assets))
		for _, asset := range assets {
			if asset.IsActive {
				activeSymbols[strings.ToUpper(asset.Ticker)] = struct{}{}
			}
		}
		results, err := reader.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenant, AlgorithmID: riskRewardAlgorithmID, Limit: 200})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list risk/reward results")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"summaries": curateRiskRewardSummaries(results, activeSymbols)})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(marketOpsAssetAlgorithmObservationReader)
		if !ok {
			writeError(w, http.StatusNotImplemented, "algorithm_observations_unavailable", "asset algorithm observations are unavailable")
			return
		}
		tenant, symbol := strings.TrimSpace(r.PathValue("tenant_id")), strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		if tenant == "" || symbol == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "tenant_id and symbol are required")
			return
		}
		results, err := reader.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenant, Limit: 2000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm results")
			return
		}
		eod, other := curateAssetAlgorithmObservations(results, symbol)
		riskReward := curateRiskRewardObservations(results, symbol)
		writeJSON(w, http.StatusOK, map[string]any{
			"symbol":        symbol,
			"eod_zscores":   eod,
			"other_outputs": algorithmResultResponses(other),
			"risk_reward":   riskReward,
		})
	})
}

func curateAssetAlgorithmObservations(results []storage.AlgorithmResultRecord, symbol string) ([]marketOpsEODZScoreDTO, []storage.AlgorithmResultRecord) {
	byDate := map[string][]storage.AlgorithmResultRecord{}
	parsed := map[string]map[string]any{}
	other := make([]storage.AlgorithmResultRecord, 0)
	for _, result := range results {
		if _, platform := marketOpsPlatformAlgorithmIDs[result.AlgorithmID]; !platform {
			continue
		}
		payload := map[string]any{}
		if json.Unmarshal(result.ResultPayloadJSON, &payload) != nil || strings.ToUpper(stringAny(payload["symbol"])) != symbol {
			continue
		}
		parsed[result.AlgorithmResultID] = payload
		if result.AlgorithmID != zscoreAlgorithmID {
			other = append(other, result)
			continue
		}
		observationTime, err := time.Parse(time.RFC3339Nano, stringAny(payload["observation_time"]))
		if err != nil {
			other = append(other, result)
			continue
		}
		day := observationTime.UTC().Format("2006-01-02")
		byDate[day] = append(byDate[day], result)
	}
	dates := make([]string, 0, len(byDate))
	for day := range byDate {
		dates = append(dates, day)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if len(dates) > 3 {
		dates = dates[:3]
	}

	selected := map[string]struct{}{}
	eod := make([]marketOpsEODZScoreDTO, 0, len(dates))
	for _, day := range dates {
		candidates := make([]storage.AlgorithmResultRecord, 0, len(byDate[day]))
		for _, result := range byDate[day] {
			if usableEODZScore(parsed[result.AlgorithmResultID]) {
				candidates = append(candidates, result)
			}
		}
		if len(candidates) == 0 {
			eod = append(eod, marketOpsEODZScoreDTO{TradeDate: day, Status: "no_usable_zscore", Reason: "All z-score candidates for this date use unusable options-ratio inputs."})
			other = append(other, byDate[day]...)
			continue
		}
		sort.SliceStable(candidates, func(i, j int) bool { return preferredEODZScore(candidates[i], candidates[j]) })
		chosen := candidates[0]
		selected[chosen.AlgorithmResultID] = struct{}{}
		dto := algorithmResultResponse(chosen)
		eod = append(eod, marketOpsEODZScoreDTO{TradeDate: day, AlgorithmResult: &dto, Status: "selected"})
		for _, result := range byDate[day] {
			if _, isSelected := selected[result.AlgorithmResultID]; !isSelected {
				other = append(other, result)
			}
		}
	}
	sort.SliceStable(other, func(i, j int) bool {
		left, right := observationDate(parsed[other[i].AlgorithmResultID]), observationDate(parsed[other[j].AlgorithmResultID])
		if left != right {
			return left > right
		}
		return other[i].CreatedAt.After(other[j].CreatedAt)
	})
	return eod, other
}

func curateRiskRewardSummaries(results []storage.AlgorithmResultRecord, activeSymbols map[string]struct{}) []map[string]any {
	bySymbol := map[string]map[string]any{}
	for _, result := range results {
		if result.AlgorithmID != riskRewardAlgorithmID {
			continue
		}
		payload := map[string]any{}
		if json.Unmarshal(result.ResultPayloadJSON, &payload) != nil {
			continue
		}
		symbol := strings.ToUpper(stringAny(payload["symbol"]))
		if _, active := activeSymbols[symbol]; !active || stringAny(payload["observation_time"]) == "" {
			continue
		}
		candidate := map[string]any{
			"ticker":        symbol,
			"trade_date":    observationDate(payload),
			"_observed_at":  stringAny(payload["observation_time"]),
			"direction":     payload["technical_direction"],
			"score":         payload["technical_score"],
			"confidence":    result.Confidence,
			"risk_level":    payload["risk_level"],
			"research_only": true,
		}
		if current, exists := bySymbol[symbol]; !exists || fmt.Sprint(candidate["_observed_at"]) > fmt.Sprint(current["_observed_at"]) {
			bySymbol[symbol] = candidate
		}
	}
	items := make([]map[string]any, 0, len(bySymbol))
	for _, item := range bySymbol {
		delete(item, "_observed_at")
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["ticker"]) < fmt.Sprint(items[j]["ticker"])
	})
	return items
}

func curateRiskRewardObservations(results []storage.AlgorithmResultRecord, symbol string) map[string]any {
	history := make([]map[string]any, 0, 60)
	for _, result := range results {
		if result.AlgorithmID != riskRewardAlgorithmID {
			continue
		}
		payload := map[string]any{}
		if json.Unmarshal(result.ResultPayloadJSON, &payload) != nil || strings.ToUpper(stringAny(payload["symbol"])) != symbol {
			continue
		}
		if stringAny(payload["observation_time"]) == "" {
			continue
		}
		history = append(history, map[string]any{"algorithm_result_id": result.AlgorithmResultID, "trade_date": observationDate(payload), "score": payload["technical_score"], "direction": payload["technical_direction"], "risk_level": payload["risk_level"], "confidence": result.Confidence, "severity": result.Severity, "technical_factors": payload["technical_factors"], "speculative_corroboration": payload["speculative_corroboration"], "research_only": true})
	}
	sort.SliceStable(history, func(i, j int) bool {
		return fmt.Sprint(history[i]["trade_date"]) > fmt.Sprint(history[j]["trade_date"])
	})
	if len(history) > 60 {
		history = history[:60]
	}
	out := map[string]any{"history": history}
	if len(history) > 0 {
		out["latest"] = history[0]
	}
	return out
}

func usableEODZScore(payload map[string]any) bool {
	feature := stringAny(payload["feature"])
	if feature != "call_put_open_interest_ratio" && feature != "call_put_volume_ratio" {
		return true
	}
	switch stringAny(payload["call_put_oi_ratio_quality"]) {
	case "partial_zero", "all_zero", "denominator_zero":
		return false
	default:
		return true
	}
}

func preferredEODZScore(left, right storage.AlgorithmResultRecord) bool {
	if left.Confidence != right.Confidence {
		return left.Confidence > right.Confidence
	}
	if severityRank(left.Severity) != severityRank(right.Severity) {
		return severityRank(left.Severity) > severityRank(right.Severity)
	}
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func severityRank(value string) int {
	switch value {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}

func observationDate(payload map[string]any) string {
	at, err := time.Parse(time.RFC3339Nano, stringAny(payload["observation_time"]))
	if err != nil {
		return ""
	}
	return at.UTC().Format("2006-01-02")
}
