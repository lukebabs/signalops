package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"sort"
	"strings"
	"time"
)

func registerMarketOpsEEOMRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/earnings-opportunities", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		watchlistContext, ok := requireSubscriberWatchlistContext(w, r, cfg, tenantID)
		if !ok {
			return
		}
		start, end := time.Now().UTC(), time.Now().UTC().AddDate(0, 0, 30)
		if v := r.URL.Query().Get("start_date"); v != "" {
			start, _ = time.Parse("2006-01-02", v)
		}
		if v := r.URL.Query().Get("end_date"); v != "" {
			end, _ = time.Parse("2006-01-02", v)
		}
		filter := storage.MarketOpsEEOMFilter{TenantID: tenantID, Symbol: r.URL.Query().Get("symbol"), StartDate: start, EndDate: end, EligibleOnly: strings.EqualFold(r.URL.Query().Get("eligible_only"), "true"), Limit: queryLimit(r, 200)}
		var rows []storage.MarketOpsEEOMResultRecord
		var err error
		if subscriberWatchlistContextEnabled(cfg, tenantID) {
			globalReader, supported := any(cfg.QueryRepository).(storage.SubscriberGlobalEEOMRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "global_eeom_unavailable", "global EEOM projection is unavailable")
				return
			}
			rows, err = globalReader.ListSubscriberGlobalEEOMResults(r.Context(), authorizedEROCTickers(watchlistContext, filter.Symbol), filter)
		} else {
			repo, supported := any(cfg.QueryRepository).(storage.MarketOpsEEOMRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "eeom_unavailable", "EEOM results are unavailable")
				return
			}
			rows, err = repo.ListMarketOpsEEOMResults(r.Context(), filter)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list EEOM results")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, x := range rows {
			if subscriberWatchlistContextEnabled(cfg, tenantID) {
				if _, allowed := watchlistContext.Tickers[strings.ToUpper(x.Symbol)]; !allowed {
					continue
				}
			}
			trace := map[string]any{}
			_ = json.Unmarshal(x.ResultJSON, &trace)
			out = append(out, map[string]any{"result_id": x.ResultID, "ticker": x.Symbol, "earnings_event_id": x.EarningsEventID, "earnings_date": x.EarningsDate.Format("2006-01-02"), "trade_date": x.SessionDate.Format("2006-01-02"), "model_version": x.ModelVersion, "score": x.Score, "posture": x.Posture, "classification": x.Classification, "evidence_quality": x.EvidenceQuality, "eligible": x.Eligible, "data_scope": x.TenantID, "event": trace["event"], "trace": trace})
		}
		response := map[string]any{"results": out, "research_only": true, "description": "Pre-earnings setup quality, not an earnings outcome or direction forecast."}
		if subscriberWatchlistContextEnabled(cfg, tenantID) {
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/material-events", func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := requireRequestTenant(w, r, r.PathValue("tenant_id"))
		if !ok {
			return
		}
		watchlistContext, ok := requireSubscriberWatchlistContext(w, r, cfg, tenantID)
		if !ok {
			return
		}
		if subscriberWatchlistContextEnabled(cfg, tenantID) {
			globalReader, supported := any(cfg.QueryRepository).(storage.SubscriberGlobalMaterialEventRepository)
			if !supported {
				writeError(w, http.StatusServiceUnavailable, "global_material_events_unavailable", "global Material Events projection is unavailable")
				return
			}
			today := time.Now().UTC().Truncate(24 * time.Hour)
			symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
			globalEvents, err := globalReader.ListSubscriberGlobalMaterialEvents(r.Context(), authorizedEROCTickers(watchlistContext, symbol), today.AddDate(0, 0, -2), queryLimit(r, 500))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "query_failed", "failed to list global MarketOps material events")
				return
			}
			events := make([]map[string]any, 0, len(globalEvents))
			for _, record := range globalEvents {
				payload := map[string]any{}
				if json.Unmarshal(record.PayloadJSON, &payload) != nil || strings.ToLower(eeomString(payload["event_type"])) != "earnings" {
					continue
				}
				payload["event_id"] = record.EventID
				payload["data_scope"] = "platform-global"
				payload["days_to_event"] = int(recordedEventDate(payload).Sub(today).Hours() / 24)
				events = append(events, payload)
			}
			sort.Slice(events, func(i, j int) bool {
				left, right := eeomString(events[i]["event_date"]), eeomString(events[j]["event_date"])
				if left == right {
					return eeomString(events[i]["symbol"]) < eeomString(events[j]["symbol"])
				}
				return left < right
			})
			writeJSON(w, http.StatusOK, map[string]any{"events": events, "research_only": true, "description": "Point-in-time-known earnings dates from Financial Modeling Prep; timing and confirmation are unavailable from this source.", "watchlist_context": subscriberWatchlistContextResponse(watchlistContext)})
			return
		}
		records, err := cfg.QueryRepository.ListNormalizedEventLedger(r.Context(), storage.RawEventLedgerFilter{TenantID: tenantID, AppID: "marketops", Dataset: "market_event_calendar", Limit: queryLimit(r, 500)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list MarketOps material events")
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
		events := make([]map[string]any, 0, len(records))
		for _, record := range records {
			if record.SourceAdapter != "market_data.fmp" {
				continue
			}
			payload := map[string]any{}
			if json.Unmarshal(record.NormalizedPayload, &payload) != nil || strings.ToLower(eeomString(payload["event_type"])) != "earnings" {
				continue
			}
			ticker := strings.ToUpper(eeomString(payload["symbol"]))
			if symbol != "" && ticker != symbol {
				continue
			}
			if subscriberWatchlistContextEnabled(cfg, tenantID) {
				if _, allowed := watchlistContext.Tickers[ticker]; !allowed {
					continue
				}
			}
			date, parseErr := time.Parse("2006-01-02", eeomString(payload["event_date"]))
			if parseErr != nil || date.Before(today.AddDate(0, 0, -2)) {
				continue
			}
			payload["days_to_event"] = int(date.Sub(today).Hours() / 24)
			payload["event_id"] = record.EventID
			events = append(events, payload)
		}
		sort.Slice(events, func(i, j int) bool {
			left, right := eeomString(events[i]["event_date"]), eeomString(events[j]["event_date"])
			if left == right {
				return eeomString(events[i]["symbol"]) < eeomString(events[j]["symbol"])
			}
			return left < right
		})
		response := map[string]any{"events": events, "research_only": true, "description": "Point-in-time-known earnings dates from Financial Modeling Prep; timing and confirmation are unavailable from this source."}
		if subscriberWatchlistContextEnabled(cfg, tenantID) {
			response["watchlist_context"] = subscriberWatchlistContextResponse(watchlistContext)
		}
		writeJSON(w, http.StatusOK, response)
	})
}

func recordedEventDate(payload map[string]any) time.Time {
	date, _ := time.Parse("2006-01-02", eeomString(payload["event_date"]))
	return date.UTC()
}

func eeomString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
