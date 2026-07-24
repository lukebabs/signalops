package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const signalOverviewAllActive = "all_active"

type signalOverviewMember struct {
	Ticker string   `json:"ticker"`
	Label  string   `json:"label"`
	Score  *float64 `json:"score,omitempty"`
	AsOf   string   `json:"as_of"`
}

func registerMarketOpsSignalOverviewRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/signal-overview", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(storage.MarketOpsSignalOverviewRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "signal_overview_unavailable", "signal overview is unavailable")
			return
		}
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		group := strings.TrimSpace(r.URL.Query().Get("universe_group"))
		if group == "" || group == signalOverviewAllActive {
			group = ""
		}
		if tenant == "" || (group != "" && group != "top50_megacap" && group != analystWatchlistGroup) {
			writeError(w, http.StatusBadRequest, "invalid_signal_overview_filter", "a valid tenant and universe group are required")
			return
		}
		window, days := signalOverviewWindow(r.URL.Query().Get("window"))
		if days == 0 {
			writeError(w, http.StatusBadRequest, "invalid_window", "window must be 10_trade_days, 30_trade_days, or 60_trade_days")
			return
		}
		inputs, err := reader.ListMarketOpsSignalOverviewInputs(r.Context(), storage.MarketOpsSignalOverviewFilter{TenantID: tenant, UniverseGroup: group, SessionStart: time.Now().UTC().AddDate(0, 0, -days*3)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to build signal overview")
			return
		}
		writeJSON(w, http.StatusOK, buildMarketOpsSignalOverview(inputs, group, window, days))
	})
}

func signalOverviewWindow(value string) (string, int) {
	switch value {
	case "", "60_trade_days":
		return "60_trade_days", 60
	case "30_trade_days":
		return value, 30
	case "10_trade_days":
		return value, 10
	default:
		return "", 0
	}
}

func buildMarketOpsSignalOverview(inputs storage.MarketOpsSignalOverviewInputs, group, window string, days int) map[string]any {
	active := map[string]struct{}{}
	for _, asset := range inputs.Assets {
		if asset.IsActive {
			active[strings.ToUpper(asset.Ticker)] = struct{}{}
		}
	}
	risk := signalOverviewRiskReward(inputs.AlgorithmResults, active, days)
	hypotheses := signalOverviewHypotheses(inputs.HypothesisEvaluations, inputs.HypothesisDefinitions, active, days)
	intraday := signalOverviewIntraday(inputs.IntradayConditionSnaps, active)
	if group == "" {
		group = signalOverviewAllActive
	}
	return map[string]any{"generated_at": time.Now().UTC(), "universe_group": group, "window": window, "asset_count": len(active), "risk_reward": map[string]any{"points": risk}, "hypotheses": map[string]any{"points": hypotheses}, "intraday": intraday}
}

func signalOverviewRiskReward(results []storage.AlgorithmResultRecord, active map[string]struct{}, days int) []map[string]any {
	byDate := map[string]map[string]signalOverviewMember{}
	for _, result := range results {
		payload := map[string]any{}
		if json.Unmarshal(result.ResultPayloadJSON, &payload) != nil {
			continue
		}
		symbol := strings.ToUpper(stringAny(payload["symbol"]))
		if _, ok := active[symbol]; !ok {
			continue
		}
		date := observationDate(payload)
		score, ok := numberAny(payload["technical_score"])
		if date == "" || !ok {
			continue
		}
		category := strings.ToLower(stringAny(payload["technical_direction"]))
		if category != "bullish" && category != "bearish" {
			category = "neutral"
		}
		if byDate[date] == nil {
			byDate[date] = map[string]signalOverviewMember{}
		}
		key := symbol
		candidate := signalOverviewMember{Ticker: symbol, Label: category, Score: &score, AsOf: date}
		if _, exists := byDate[date][key]; !exists {
			byDate[date][key] = candidate
		}
	}
	return signalOverviewPoints(byDate, days)
}

func signalOverviewHypotheses(evaluations []storage.MarketOpsHypothesisEvaluationRecord, definitions []storage.MarketOpsHypothesisDefinitionRecord, active map[string]struct{}, days int) []map[string]any {
	directions := map[string]string{}
	titles := map[string]string{}
	for _, definition := range definitions {
		key := definition.HypothesisKey + ":" + definition.HypothesisVersion
		directions[key] = strings.ToLower(definition.Direction)
		titles[key] = definition.Title
	}
	byDate := map[string]map[string]signalOverviewMember{}
	for _, evaluation := range evaluations {
		symbol := strings.ToUpper(evaluation.Symbol)
		if !evaluation.Triggered || evaluation.Invalidated {
			continue
		}
		if _, ok := active[symbol]; !ok {
			continue
		}
		key := evaluation.HypothesisKey + ":" + evaluation.HypothesisVersion
		category := directions[key]
		if category != "bullish" && category != "bearish" {
			category = "neutral"
		}
		date := evaluation.SessionDate.UTC().Format("2006-01-02")
		if byDate[date] == nil {
			byDate[date] = map[string]signalOverviewMember{}
		}
		memberKey := symbol + ":" + evaluation.HypothesisKey + ":" + evaluation.HypothesisVersion
		label := titles[key]
		if label == "" {
			label = evaluation.HypothesisKey
		}
		byDate[date][memberKey] = signalOverviewMember{Ticker: symbol, Label: label, Score: evaluation.TriggerScore, AsOf: date + " · " + category}
	}
	return signalOverviewPoints(byDate, days)
}

func signalOverviewPoints(byDate map[string]map[string]signalOverviewMember, days int) []map[string]any {
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if len(dates) > days {
		dates = dates[:days]
	}
	sort.Strings(dates)
	points := make([]map[string]any, 0, len(dates))
	for _, date := range dates {
		categories := map[string][]signalOverviewMember{"bullish": {}, "neutral": {}, "bearish": {}}
		for _, member := range byDate[date] {
			category := member.Label
			if member.Score != nil && (member.Label == "bullish" || member.Label == "neutral" || member.Label == "bearish") {
				category = member.Label
			} else if strings.HasSuffix(member.AsOf, " · bullish") {
				category = "bullish"
			} else if strings.HasSuffix(member.AsOf, " · bearish") {
				category = "bearish"
			} else {
				category = "neutral"
			}
			categories[category] = append(categories[category], member)
		}
		entries := make([]map[string]any, 0, 3)
		for _, category := range []string{"bullish", "neutral", "bearish"} {
			members := categories[category]
			sort.Slice(members, func(i, j int) bool { return members[i].Ticker < members[j].Ticker })
			entries = append(entries, map[string]any{"key": category, "count": len(members), "members": members})
		}
		points = append(points, map[string]any{"trade_date": date, "categories": entries})
	}
	return points
}

func signalOverviewIntraday(records []storage.MarketOpsIntradayConditionSnapshotRecord, active map[string]struct{}) map[string]any {
	type condition struct {
		Title string  `json:"title"`
		Tone  string  `json:"tone"`
		Score float64 `json:"score"`
	}
	latest := map[string]storage.MarketOpsIntradayConditionSnapshotRecord{}
	for _, record := range records {
		symbol := strings.ToUpper(record.Symbol)
		if _, ok := active[symbol]; ok {
			latest[symbol] = record
		}
	}
	categories := map[string][]signalOverviewMember{"positive": {}, "negative": {}, "neutral": {}, "no_active_condition": {}, "unavailable": {}}
	asOf := time.Time{}
	for symbol := range active {
		record, exists := latest[symbol]
		if !exists {
			categories["unavailable"] = append(categories["unavailable"], signalOverviewMember{Ticker: symbol, Label: "No persisted intraday monitor snapshot", AsOf: ""})
			continue
		}
		if record.AsOfTime.After(asOf) {
			asOf = record.AsOfTime
		}
		if record.Stale {
			categories["unavailable"] = append(categories["unavailable"], signalOverviewMember{Ticker: symbol, Label: "Stale intraday monitor snapshot", AsOf: record.AsOfTime.UTC().Format(time.RFC3339)})
			continue
		}
		var conditions []condition
		_ = json.Unmarshal(record.ConditionsJSON, &conditions)
		if len(conditions) == 0 {
			categories["no_active_condition"] = append(categories["no_active_condition"], signalOverviewMember{Ticker: symbol, Label: "No active condition", AsOf: record.AsOfTime.UTC().Format(time.RFC3339)})
			continue
		}
		top := conditions[0]
		for _, candidate := range conditions[1:] {
			if candidate.Score > top.Score {
				top = candidate
			}
		}
		category := "neutral"
		if top.Tone == "positive" {
			category = "positive"
		} else if top.Tone == "negative" {
			category = "negative"
		}
		score := top.Score
		categories[category] = append(categories[category], signalOverviewMember{Ticker: symbol, Label: top.Title, Score: &score, AsOf: record.AsOfTime.UTC().Format(time.RFC3339)})
	}
	entries := make([]map[string]any, 0, len(categories))
	for _, category := range []string{"positive", "negative", "neutral", "no_active_condition", "unavailable"} {
		members := categories[category]
		sort.Slice(members, func(i, j int) bool { return members[i].Ticker < members[j].Ticker })
		entries = append(entries, map[string]any{"key": category, "count": len(members), "members": members})
	}
	return map[string]any{"as_of_time": asOf, "categories": entries}
}
