package api

import (
	"context"
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"sort"
	"strings"
	"time"
)

type marketOpsQuantitativeSeriesReader interface {
	ListMarketOpsBacktestNormalizedEvents(context.Context, storage.MarketOpsBacktestEventFilter) ([]storage.NormalizedEventLedgerRecord, error)
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
	ListAlgorithmResults(context.Context, storage.AlgorithmResultFilter) ([]storage.AlgorithmResultRecord, error)
	ListMarketOpsAlgorithmAdjudications(context.Context, storage.MarketOpsAlgorithmAdjudicationFilter) ([]storage.MarketOpsAlgorithmAdjudicationRecord, error)
}

func registerMarketOpsQuantitativeSeriesRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/quantitative-series", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(marketOpsQuantitativeSeriesReader)
		if !ok {
			writeError(w, 501, "quantitative_series_unavailable", "quantitative series is unavailable")
			return
		}
		tenant, symbol := strings.TrimSpace(r.PathValue("tenant_id")), strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		if tenant == "" || symbol == "" {
			writeError(w, 400, "missing_path", "tenant_id and symbol are required")
			return
		}
		days := 10
		switch r.URL.Query().Get("window") {
		case "30_trade_days":
			days = 30
		case "60_trade_days":
			days = 60
		case "10_trade_days", "":
		default:
			writeError(w, 400, "invalid_window", "window must be 10_trade_days, 30_trade_days, or 60_trade_days")
			return
		}
		end := time.Now().UTC().AddDate(0, 0, 1)
		start := end.AddDate(0, 0, -days*3)
		events, err := reader.ListMarketOpsBacktestNormalizedEvents(r.Context(), storage.MarketOpsBacktestEventFilter{TenantID: tenant, Dataset: "equity_eod_prices", Symbols: []string{symbol}, WindowStart: start, WindowEnd: end, Limit: days * 4})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list EOD prices")
			return
		}
		distributions, err := reader.ListMarketOpsOptionsDistributions(r.Context(), storage.MarketOpsOptionsDistributionFilter{TenantID: tenant, Symbol: symbol, WindowName: "10_trade_days", Limit: days})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list options distributions")
			return
		}
		results, err := reader.ListAlgorithmResults(r.Context(), storage.AlgorithmResultFilter{TenantID: tenant, Limit: 1000})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list algorithm results")
			return
		}
		adjudications, err := reader.ListMarketOpsAlgorithmAdjudications(r.Context(), storage.MarketOpsAlgorithmAdjudicationFilter{TenantID: tenant, Symbol: symbol, Limit: 1000})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list algorithm adjudications")
			return
		}
		points := map[string]map[string]any{}
		ensure := func(day string) map[string]any {
			if points[day] == nil {
				points[day] = map[string]any{"trade_date": day, "markers": []any{}}
			}
			return points[day]
		}
		for _, e := range events {
			p := map[string]any{}
			if json.Unmarshal(e.NormalizedPayload, &p) != nil {
				continue
			}
			day := e.ObservationTime.UTC().Format("2006-01-02")
			close, ok := number(p["close"])
			if !ok {
				continue
			}
			point := ensure(day)
			point["eod_close"] = close
			if open, ok := number(p["open"]); ok && open != 0 {
				point["daily_move_pct"] = roundFloat((close - open) / open * 100)
			}
		}
		for _, d := range distributions {
			point := ensure(d.TradeDate.UTC().Format("2006-01-02"))
			point["call_put_open_interest_ratio"] = d.CallPutOpenInterestRatio
			point["call_put_volume_ratio"] = d.CallPutVolumeRatio
			point["ratio_quality"] = ratioQuality(d.MetricsJSON)
		}
		byResult := map[string][]storage.MarketOpsAlgorithmAdjudicationRecord{}
		for _, a := range adjudications {
			byResult[a.AlgorithmResultID] = append(byResult[a.AlgorithmResultID], a)
		}
		for _, ar := range results {
			if ar.Severity != "medium" && ar.Severity != "high" && ar.Severity != "critical" {
				continue
			}
			payload := map[string]any{}
			_ = json.Unmarshal(ar.ResultPayloadJSON, &payload)
			if strings.ToUpper(stringAny(payload["symbol"])) != symbol {
				continue
			}
			when := stringAny(payload["observation_time"])
			t, err := time.Parse(time.RFC3339Nano, when)
			if err != nil {
				continue
			}
			marker := map[string]any{"algorithm_result_id": ar.AlgorithmResultID, "algorithm_id": ar.AlgorithmID, "severity": ar.Severity, "score": ar.Score, "confidence": ar.Confidence, "direction": stringAny(payload["direction"])}
			if matches := byResult[ar.AlgorithmResultID]; len(matches) > 0 {
				marker["adjudications"] = matches
			}
			point := ensure(t.UTC().Format("2006-01-02"))
			point["markers"] = append(point["markers"].([]any), marker)
		}
		dates := make([]string, 0, len(points))
		for day := range points {
			dates = append(dates, day)
		}
		sort.Strings(dates)
		if len(dates) > days {
			dates = dates[len(dates)-days:]
		}
		out := make([]any, 0, len(dates))
		for _, day := range dates {
			out = append(out, points[day])
		}
		writeJSON(w, 200, map[string]any{"symbol": symbol, "window": r.URL.Query().Get("window"), "points": out})
	})
}
func number(v any) (float64, bool) { n, ok := v.(float64); return n, ok }
func stringAny(v any) string       { s, _ := v.(string); return s }
func roundFloat(v float64) float64 { return float64(int(v*10000+0.5)) / 10000 }
func ratioQuality(raw []byte) string {
	p := map[string]any{}
	_ = json.Unmarshal(raw, &p)
	return stringAny(p["call_put_oi_ratio_quality"])
}
