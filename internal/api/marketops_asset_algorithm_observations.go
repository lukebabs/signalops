package api

import (
	"context"
	"encoding/json"
	"errors"
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

func registerMarketOpsAssetAlgorithmObservationRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repo := cfg.QueryRepository
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/risk-reward", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(marketOpsAssetAlgorithmObservationReader)
		if !ok {
			writeError(w, http.StatusNotImplemented, "risk_reward_unavailable", "risk/reward summaries are unavailable")
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
		universeGroup := strings.TrimSpace(r.URL.Query().Get("universe_group"))
		if universeGroup == "" {
			universeGroup = "top50_megacap"
		}
		activeSymbols := map[string]struct{}{}
		if subscriberWatchlistContextEnabled(cfg, tenant) {
			for ticker := range watchlistContext.Tickers {
				activeSymbols[ticker] = struct{}{}
			}
		} else {
			assets, err := reader.ListMarketOpsAssets(r.Context(), tenant, universeGroup, true, 500)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list marketops assets")
				return
			}
			for _, asset := range assets {
				if asset.IsActive {
					activeSymbols[strings.ToUpper(asset.Ticker)] = struct{}{}
				}
			}
		}
		// Risk/Reward evolution requires at least two persisted sessions per asset.
		// Do not derive it from a globally capped raw-result list: once the active
		// universe exceeds half that cap, valid prior-session results are omitted.
		if snapshots, ok := any(repo).(storage.MarketOpsRiskRewardSnapshotRepository); ok {
			limit := len(activeSymbols) * 6
			if limit < 1000 {
				limit = 1000
			}
			items, err := snapshots.ListMarketOpsRiskRewardSnapshots(r.Context(), storage.MarketOpsRiskRewardSnapshotFilter{
				TenantID: tenant, Symbols: mapKeys(activeSymbols), SessionStart: time.Now().UTC().AddDate(0, 0, -21), EligibleOnly: true, Limit: limit,
			})
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list risk/reward snapshots")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"summaries": curateRiskRewardSnapshotSummaries(items, activeSymbols)})
			return
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
		tenant, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		symbol := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		if symbol == "" {
			writeError(w, http.StatusBadRequest, "missing_path", "symbol is required")
			return
		}
		watchlistContext, ok := requireSubscriberWatchlistContext(w, r, cfg, tenant)
		if !ok {
			return
		}
		if subscriberWatchlistContextEnabled(cfg, tenant) {
			if _, allowed := watchlistContext.Tickers[symbol]; !allowed {
				writeError(w, http.StatusNotFound, "marketops_asset_not_found", "MarketOps asset was not found in the selected watchlist")
				return
			}
		}
		results, err := reader.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenant, Limit: 2000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list algorithm results")
			return
		}
		eod, other := curateAssetAlgorithmObservations(results, symbol)
		riskReward := curateRiskRewardObservations(results, symbol)
		var currentEOD any
		revisionReview := map[string]any{"available": false, "usage_context": "revision_review", "initial_observation_role": "initial_tenant_local_capture", "revised_observation_role": "global_reobservation", "deltas": []map[string]any{}}
		if currentReader, ok := any(repo).(storage.SubscriberCurrentEODContextRepository); ok {
			current, currentErr := currentReader.GetSubscriberCurrentEODContext(r.Context(), tenant, symbol)
			if currentErr != nil && !errors.Is(currentErr, storage.ErrNotFound) {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to load current EOD context")
				return
			}
			if currentErr == nil {
				currentEOD = currentEODContextResponse(current)
			}
		}
		if reviewReader, ok := any(repo).(storage.SubscriberEODRevisionReviewRepository); ok {
			deltas, reviewErr := reviewReader.ListSubscriberEODRevisionDeltas(r.Context(), tenant, symbol, 24)
			if reviewErr != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to load EOD revision review")
				return
			}
			revisionReview = revisionReviewResponse(deltas)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"symbol":              symbol,
			"eod_zscores":         eod,
			"other_outputs":       algorithmResultResponses(other),
			"risk_reward":         riskReward,
			"current_eod_context": currentEOD,
			"eod_revision_review": revisionReview,
		})
	})
}

func revisionReviewResponse(records []storage.SubscriberEODRevisionDeltaRecord) map[string]any {
	deltas := make([]map[string]any, 0, len(records))
	reviewRequired := 0
	for _, record := range records {
		if record.Materiality == "review_required" {
			reviewRequired++
		}
		deltas = append(deltas, map[string]any{
			"session_date": record.SessionDate.UTC().Format("2006-01-02"), "field_name": record.FieldName,
			"initial_value": record.InitialValue, "revised_value": record.RevisedValue, "delta_class": record.DeltaClass, "materiality": record.Materiality,
			"initial_observed_at": record.InitialObservedAt.UTC().Format(time.RFC3339), "revised_observed_at": record.RevisedObservedAt.UTC().Format(time.RFC3339),
			"initial_source_event_id": record.InitialSourceEventID, "revised_source_event_id": record.RevisedSourceEventID,
			"initial_source_run_id": record.InitialSourceRunID, "revised_source_run_id": record.RevisedSourceRunID,
			"initial_payload_fingerprint": record.InitialPayloadFingerprint, "revised_payload_fingerprint": record.RevisedPayloadFingerprint,
			"initial_algorithm_version": record.InitialAlgorithmVersion, "revised_algorithm_version": record.RevisedAlgorithmVersion,
		})
	}
	return map[string]any{"available": len(deltas) > 0, "usage_context": "revision_review", "initial_observation_role": "initial_tenant_local_capture", "revised_observation_role": "global_reobservation", "review_required_count": reviewRequired, "deltas": deltas}
}

func currentEODContextResponse(record storage.SubscriberCurrentEODContextRecord) map[string]any {
	return map[string]any{
		"symbol": record.Symbol, "session_date": record.SessionDate.UTC().Format("2006-01-02"),
		"open": record.Open, "high": record.High, "low": record.Low, "close": record.Close, "volume": record.Volume, "vwap": record.VWAP,
		"provider": record.Provider, "usage_context": "current_market_context", "selected_observation_role": record.SelectedObservationRole,
		"policy_version": record.SelectionPolicyVersion, "payload_fingerprint": record.PayloadFingerprint,
		"source_event_id": record.SourceEventID, "source_run_id": record.SourceRunID, "algorithm_version": record.AlgorithmVersion,
		"quality_state": record.QualityState, "as_of_time": record.AsOfTime.UTC().Format(time.RFC3339),
	}
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
	// A scheduled runner can be retried, so select one immutable result for each
	// symbol/session before deriving the day-over-day evolution.  The comparison is
	// intentionally between adjacent persisted trading sessions, not calendar days.
	bySymbol := map[string]map[string]map[string]any{}
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
		tradeDate := observationDate(payload)
		if tradeDate == "" {
			continue
		}
		candidate := map[string]any{
			"ticker":        symbol,
			"trade_date":    tradeDate,
			"_observed_at":  stringAny(payload["observation_time"]),
			"_created_at":   result.CreatedAt.UTC().Format(time.RFC3339Nano),
			"direction":     payload["technical_direction"],
			"score":         payload["technical_score"],
			"confidence":    result.Confidence,
			"risk_level":    payload["risk_level"],
			"research_only": true,
		}
		if bySymbol[symbol] == nil {
			bySymbol[symbol] = map[string]map[string]any{}
		}
		if current, exists := bySymbol[symbol][tradeDate]; !exists || betterRiskRewardPayload(candidate, current) {
			bySymbol[symbol][tradeDate] = candidate
		}
	}
	items := make([]map[string]any, 0, len(bySymbol))
	for _, byDate := range bySymbol {
		dates := make([]string, 0, len(byDate))
		for date := range byDate {
			dates = append(dates, date)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		latest := byDate[dates[0]]
		delete(latest, "_observed_at")
		delete(latest, "_created_at")
		if len(dates) > 1 {
			previous := byDate[dates[1]]
			latest["previous_trade_date"] = dates[1]
			latest["previous_score"] = previous["score"]
			if currentScore, ok := numberAny(latest["score"]); ok {
				if previousScore, ok := numberAny(previous["score"]); ok {
					latest["score_change"] = currentScore - previousScore
				}
			}
		}
		items = append(items, latest)
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["ticker"]) < fmt.Sprint(items[j]["ticker"])
	})
	return items
}

func curateRiskRewardObservations(results []storage.AlgorithmResultRecord, symbol string) map[string]any {
	byTradeDate := map[string]map[string]any{}
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
		tradeDate := observationDate(payload)
		if tradeDate == "" {
			continue
		}
		candidate := map[string]any{"algorithm_result_id": result.AlgorithmResultID, "trade_date": tradeDate, "score": payload["technical_score"], "direction": payload["technical_direction"], "risk_level": payload["risk_level"], "confidence": result.Confidence, "severity": result.Severity, "technical_factors": payload["technical_factors"], "speculative_corroboration": payload["speculative_corroboration"], "research_only": true, "_observed_at": stringAny(payload["observation_time"]), "_created_at": result.CreatedAt.UTC().Format(time.RFC3339Nano)}
		if current, exists := byTradeDate[tradeDate]; !exists || betterRiskRewardPayload(candidate, current) {
			byTradeDate[tradeDate] = candidate
		}
	}
	history := make([]map[string]any, 0, len(byTradeDate))
	for _, point := range byTradeDate {
		delete(point, "_observed_at")
		delete(point, "_created_at")
		history = append(history, point)
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

func numberAny(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func mapKeys(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	return items
}

func curateRiskRewardSnapshotSummaries(snapshots []storage.MarketOpsRiskRewardSnapshotRecord, activeSymbols map[string]struct{}) []map[string]any {
	bySymbol := map[string]map[string]storage.MarketOpsRiskRewardSnapshotRecord{}
	for _, snapshot := range snapshots {
		symbol := strings.ToUpper(strings.TrimSpace(snapshot.Symbol))
		if _, active := activeSymbols[symbol]; !active || snapshot.SessionDate.IsZero() || !snapshot.Eligible {
			continue
		}
		tradeDate := snapshot.SessionDate.UTC().Format("2006-01-02")
		if bySymbol[symbol] == nil {
			bySymbol[symbol] = map[string]storage.MarketOpsRiskRewardSnapshotRecord{}
		}
		if current, exists := bySymbol[symbol][tradeDate]; !exists || betterRiskRewardSnapshot(snapshot, current) {
			bySymbol[symbol][tradeDate] = snapshot
		}
	}
	items := make([]map[string]any, 0, len(bySymbol))
	for symbol, byDate := range bySymbol {
		dates := make([]string, 0, len(byDate))
		for date := range byDate {
			dates = append(dates, date)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		latest := byDate[dates[0]]
		item := map[string]any{"ticker": symbol, "trade_date": dates[0], "direction": latest.TechnicalDirection, "score": latest.TechnicalScore, "confidence": latest.Confidence, "risk_level": latest.RiskLevel, "research_only": true}
		if len(dates) > 1 {
			previous := byDate[dates[1]]
			item["previous_trade_date"] = dates[1]
			item["previous_score"] = previous.TechnicalScore
			item["score_change"] = latest.TechnicalScore - previous.TechnicalScore
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return fmt.Sprint(items[i]["ticker"]) < fmt.Sprint(items[j]["ticker"]) })
	return items
}

func betterRiskRewardSnapshot(candidate, current storage.MarketOpsRiskRewardSnapshotRecord) bool {
	if candidate.UsableInputCount != current.UsableInputCount {
		return candidate.UsableInputCount > current.UsableInputCount
	}
	if candidate.Confidence != current.Confidence {
		return candidate.Confidence > current.Confidence
	}
	return candidate.CreatedAt.After(current.CreatedAt)
}
