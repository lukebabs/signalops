package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"sort"
	"strings"
	"time"
)

func registerStorageMonitorRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	reader, ok := any(repo).(storage.StorageMonitorRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/administration/storage/overview", func(w http.ResponseWriter, r *http.Request) {
		rows, err := reader.ListStorageMonitorSnapshots(r.Context(), "", time.Now().UTC().Add(-24*time.Hour), 3000)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load storage monitoring")
			return
		}
		latest := map[string]storage.StorageMonitorSnapshot{}
		for _, x := range rows {
			latest[x.StoreID] = x
		}
		out := []any{}
		for _, id := range []string{"postgres", "timescaledb", "redpanda"} {
			if x, found := latest[id]; found {
				out = append(out, storageSnapshotResponse(x))
			} else {
				out = append(out, map[string]any{"store_id": id, "status": "unavailable", "message": "no storage snapshot recorded"})
			}
		}
		writeJSON(w, 200, map[string]any{"stores": out})
	})
	mux.HandleFunc("GET /v1/administration/storage/history", func(w http.ResponseWriter, r *http.Request) {
		store := strings.TrimSpace(r.URL.Query().Get("store_id"))
		if store != "postgres" && store != "timescaledb" && store != "redpanda" {
			writeError(w, 400, "invalid_store_id", "invalid store_id")
			return
		}
		d := storageWindow(r.URL.Query().Get("window"))
		rows, err := reader.ListStorageMonitorSnapshots(r.Context(), store, time.Now().UTC().Add(-d), 3000)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load storage history")
			return
		}
		out := []any{}
		for _, x := range rows {
			out = append(out, storageSnapshotResponse(x))
		}
		writeJSON(w, 200, map[string]any{"store_id": store, "snapshots": out})
	})
	analysis, ok := any(repo).(storage.StorageAnalysisRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/administration/storage/analysis", func(w http.ResponseWriter, r *http.Request) {
		window := storageWindow(r.URL.Query().Get("window"))
		rows, err := analysis.ListStorageComponentSnapshots(r.Context(), time.Now().UTC().Add(-window), 20000)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load storage analysis")
			return
		}
		writeJSON(w, 200, storageAnalysisResponse(rows))
	})
}
func storageWindow(value string) time.Duration {
	switch value {
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	}
	return 7 * 24 * time.Hour
}
func storageSnapshotResponse(x storage.StorageMonitorSnapshot) map[string]any {
	var d any = map[string]any{}
	_ = json.Unmarshal(x.DetailJSON, &d)
	pct := 0.0
	if x.CapacityBytes > 0 {
		pct = float64(x.UsedBytes) * 100 / float64(x.CapacityBytes)
	}
	return map[string]any{"store_id": x.StoreID, "observed_at": x.ObservedAt, "used_bytes": x.UsedBytes, "capacity_bytes": x.CapacityBytes, "free_bytes": x.FreeBytes, "usage_percent": pct, "status": x.Status, "detail": d}
}
func storageAnalysisResponse(rows []storage.StorageComponentSnapshot) map[string]any {
	latest := map[string]time.Time{}
	for _, row := range rows {
		if row.ObservedAt.After(latest[row.StoreID]) {
			latest[row.StoreID] = row.ObservedAt
		}
	}
	current := []any{}
	totals := map[string]int64{}
	history := map[string]map[string]int64{}
	for _, row := range rows {
		key := row.AppID + "|" + row.Domain
		date := row.ObservedAt.UTC().Format("2006-01-02")
		if history[date] == nil {
			history[date] = map[string]int64{}
		}
		history[date][key] += row.AttributedBytes
		if row.ObservedAt.Equal(latest[row.StoreID]) {
			current = append(current, storageComponentResponse(row))
			totals[key] += row.AttributedBytes
		}
	}
	totalRows := []any{}
	for key, bytes := range totals {
		parts := strings.SplitN(key, "|", 2)
		totalRows = append(totalRows, map[string]any{"app_id": parts[0], "domain": parts[1], "attributed_bytes": bytes})
	}
	sort.Slice(totalRows, func(i, j int) bool {
		return totalRows[i].(map[string]any)["attributed_bytes"].(int64) > totalRows[j].(map[string]any)["attributed_bytes"].(int64)
	})
	dates := make([]string, 0, len(history))
	for date := range history {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	series := []any{}
	for _, date := range dates {
		for key, bytes := range history[date] {
			parts := strings.SplitN(key, "|", 2)
			series = append(series, map[string]any{"date": date, "app_id": parts[0], "domain": parts[1], "attributed_bytes": bytes})
		}
	}
	return map[string]any{"components": current, "ownership_totals": totalRows, "history": series, "attribution_note": "Exact values are physical table/topic sizes. Shared ledgers are allocated by current row proportion and labelled estimated."}
}
func storageComponentResponse(x storage.StorageComponentSnapshot) map[string]any {
	var metadata any = map[string]any{}
	_ = json.Unmarshal(x.MetadataJSON, &metadata)
	return map[string]any{"store_id": x.StoreID, "component_kind": x.ComponentKind, "component_name": x.ComponentName, "app_id": x.AppID, "domain": x.Domain, "attribution_method": x.AttributionMethod, "physical_bytes": x.PhysicalBytes, "attributed_bytes": x.AttributedBytes, "observed_at": x.ObservedAt, "metadata": metadata}
}
