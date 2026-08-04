package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
	"time"
)

func registerMarketOpsEEOMRoutes(mux *http.ServeMux, cfg RouterConfig) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/marketops/earnings-opportunities", func(w http.ResponseWriter, r *http.Request) {
		repo, ok := any(cfg.QueryRepository).(storage.MarketOpsEEOMRepository)
		if !ok {
			writeError(w, http.StatusServiceUnavailable, "eeom_unavailable", "EEOM results are unavailable")
			return
		}
		start, end := time.Now().UTC(), time.Now().UTC().AddDate(0, 0, 30)
		if v := r.URL.Query().Get("start_date"); v != "" {
			start, _ = time.Parse("2006-01-02", v)
		}
		if v := r.URL.Query().Get("end_date"); v != "" {
			end, _ = time.Parse("2006-01-02", v)
		}
		rows, err := repo.ListMarketOpsEEOMResults(r.Context(), storage.MarketOpsEEOMFilter{TenantID: strings.TrimSpace(r.PathValue("tenant_id")), Symbol: r.URL.Query().Get("symbol"), StartDate: start, EndDate: end, EligibleOnly: strings.EqualFold(r.URL.Query().Get("eligible_only"), "true"), Limit: queryLimit(r, 200)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to list EEOM results")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, x := range rows {
			trace := map[string]any{}
			_ = json.Unmarshal(x.ResultJSON, &trace)
			out = append(out, map[string]any{"result_id": x.ResultID, "ticker": x.Symbol, "earnings_event_id": x.EarningsEventID, "earnings_date": x.EarningsDate.Format("2006-01-02"), "trade_date": x.SessionDate.Format("2006-01-02"), "model_version": x.ModelVersion, "score": x.Score, "posture": x.Posture, "classification": x.Classification, "evidence_quality": x.EvidenceQuality, "eligible": x.Eligible, "trace": trace})
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": out, "research_only": true, "description": "Pre-earnings setup quality, not an earnings outcome or direction forecast."})
	})
}
