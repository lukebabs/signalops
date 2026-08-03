package api

import (
	"encoding/json"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
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
			if x, ok := latest[id]; ok {
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
		d := 24 * time.Hour
		if r.URL.Query().Get("window") == "7d" {
			d = 7 * 24 * time.Hour
		}
		if r.URL.Query().Get("window") == "30d" {
			d = 30 * 24 * time.Hour
		}
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
