package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const erocAlgorithmID = "signalops.algorithms.eroc_v6"

type erocOptionsDistributionReader interface {
	ListMarketOpsOptionsDistributions(context.Context, storage.MarketOpsOptionsDistributionFilter) ([]storage.MarketOpsOptionsDistributionRecord, error)
}

type erocRow struct {
	Ticker       string         `json:"ticker"`
	TradeDate    string         `json:"trade_date"`
	Score        float64        `json:"score"`
	State        string         `json:"state"`
	Confirmed    bool           `json:"confirmed"`
	ModelVersion string         `json:"model_version"`
	Trace        map[string]any `json:"trace"`
}

func erocRows(rows []storage.MarketOpsValuationResultRecord) []erocRow {
	out := make([]erocRow, 0, len(rows))
	for _, row := range rows {
		trace := map[string]any{}
		_ = json.Unmarshal(row.ResultJSON, &trace)
		out = append(out, erocRow{Ticker: row.Symbol, TradeDate: row.SessionDate.Format("2006-01-02"), Score: row.Score, State: row.Classification, Confirmed: row.Eligible, ModelVersion: row.ModelVersion, Trace: trace})
	}
	return out
}
func enrichEROCOptionsFlow(ctx context.Context, repo storage.QueryRepository, tenant string, rows []erocRow) error {
	reader, ok := any(repo).(erocOptionsDistributionReader)
	if !ok { return nil }
	for index := range rows {
		distributions, err := reader.ListMarketOpsOptionsDistributions(ctx, storage.MarketOpsOptionsDistributionFilter{TenantID: tenant, Symbol: rows[index].Ticker, WindowName: "10_trade_days", Limit: 10})
		if err != nil { return err }
		for _, distribution := range distributions {
			if distribution.TradeDate.UTC().Format("2006-01-02") != rows[index].TradeDate { continue }
			trace := rows[index].Trace
			trace["total_option_volume"] = distribution.TotalCallVolume + distribution.TotalPutVolume
			if distribution.TotalCallVolume > 0 {
				ratio := float64(distribution.TotalPutVolume) / float64(distribution.TotalCallVolume)
				trace["put_call_volume_ratio"] = ratio
				if distribution.TotalCallVolume + distribution.TotalPutVolume >= 1000 {
					if ratio < .30 { trace["options_flow_extreme"] = "call_volume_extreme" } else if ratio > 1.20 { trace["options_flow_extreme"] = "put_volume_extreme" }
				}
			}
			break
		}
	}
	return nil
}

func registerMarketOpsEROCRoutes(mux *http.ServeMux, cfg RouterConfig) {
	read := func(r *http.Request) ([]storage.MarketOpsValuationResultRecord, error) {
		repo, ok := any(cfg.QueryRepository).(storage.MarketOpsValuationRepository)
		if !ok {
			return nil, errEROCAvailable{}
		}
		return repo.ListMarketOpsValuationResults(r.Context(), storage.MarketOpsValuationFilter{TenantID: strings.TrimSpace(r.PathValue("tenant_id")), Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")), AlgorithmID: erocAlgorithmID, Limit: 5000})
	}
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/eroc", func(w http.ResponseWriter, r *http.Request) {
		rows, err := read(r)
		if err != nil {
			writeEROCErr(w, err)
			return
		}
		all := erocRows(rows)
		latest := map[string]erocRow{}
		for _, row := range all {
			if current, ok := latest[row.Ticker]; !ok || row.TradeDate > current.TradeDate {
				latest[row.Ticker] = row
			}
		}
		out := make([]erocRow, 0, len(latest))
		for _, row := range latest {
			out = append(out, row)
		}
		if err := enrichEROCOptionsFlow(r.Context(), cfg.QueryRepository, strings.TrimSpace(r.PathValue("tenant_id")), out); err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to load EROC options-flow context")
			return
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
		writeJSON(w, http.StatusOK, map[string]any{"results": out, "research_only": true})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/eroc/overview", func(w http.ResponseWriter, r *http.Request) {
		rows, err := read(r)
		if err != nil {
			writeEROCErr(w, err)
			return
		}
		days := 10
		if r.URL.Query().Get("window") == "30_trade_days" {
			days = 30
		} else if r.URL.Query().Get("window") == "60_trade_days" {
			days = 60
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -days*3)
		groups := map[string]map[string][]erocRow{}
		for _, row := range erocRows(rows) {
			d, _ := time.Parse("2006-01-02", row.TradeDate)
			if d.Before(cutoff) {
				continue
			}
			direction, _ := row.Trace["direction"].(string)
			if direction != "BULLISH" && direction != "BEARISH" {
				continue
			}
			if groups[row.TradeDate] == nil {
				groups[row.TradeDate] = map[string][]erocRow{}
			}
			groups[row.TradeDate][strings.ToLower(direction)] = append(groups[row.TradeDate][strings.ToLower(direction)], row)
		}
		dates := make([]string, 0, len(groups))
		for d := range groups {
			dates = append(dates, d)
		}
		sort.Strings(dates)
		if len(dates) > days {
			dates = dates[len(dates)-days:]
		}
		points := make([]map[string]any, 0, len(dates))
		for _, date := range dates {
			series := map[string]any{}
			for _, direction := range []string{"bullish", "bearish"} {
				members := groups[date][direction]
				scores := make([]float64, 0, len(members))
				for _, member := range members {
					scores = append(scores, member.Score)
				}
				sort.Float64s(scores)
				avg := 0.0
				for _, score := range scores {
					avg += score
				}
				if len(scores) > 0 {
					avg /= float64(len(scores))
				}
				p75 := 0.0
				if len(scores) > 0 {
					p75 = scores[int(float64(len(scores)-1)*.75)]
				}
				series[direction] = map[string]any{"average": avg, "p75": p75, "members": members}
			}
			points = append(points, map[string]any{"trade_date": date, "series": series})
		}
		writeJSON(w, http.StatusOK, map[string]any{"points": points, "research_only": true})
	})
}

type errEROCAvailable struct{}

func (errEROCAvailable) Error() string { return "eroc unavailable" }
func writeEROCErr(w http.ResponseWriter, err error) {
	if _, ok := err.(errEROCAvailable); ok {
		writeError(w, http.StatusServiceUnavailable, "eroc_unavailable", "EROC results are unavailable")
		return
	}
	writeError(w, http.StatusInternalServerError, "query_failed", "failed to list EROC results")
}
