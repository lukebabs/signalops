package api

import (
	"context"
	"github.com/lukebabs/signalops/internal/storage"
	"net/http"
	"strings"
	"time"
)

func registerMarketOpsSRIRoutes(mux *http.ServeMux, repository storage.QueryRepository) {
	q, ok := repository.(storage.MarketOpsSRIRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/marketops/sectors", func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		items, err := q.ListMarketOpsSRISegments(r.Context(), tenant, true, queryLimit(r, 100))
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list SRI segments")
			return
		}
		writeJSON(w, 200, map[string]any{"segments": sriSegmentResponses(items), "research_only": true})
	})
	mux.HandleFunc("GET /v1/marketops/sectors/rankings", func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		items, err := q.ListMarketOpsSRISnapshots(r.Context(), storage.MarketOpsSRISnapshotFilter{TenantID: tenant, SegmentType: strings.TrimSpace(r.URL.Query().Get("segment_type")), State: strings.TrimSpace(r.URL.Query().Get("state")), Limit: 200})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list SRI rankings")
			return
		}
		writeJSON(w, 200, map[string]any{"snapshots": sriLatestResponses(items), "research_only": true, "evidence_note": "Price-led foundation context only. It does not assert sector rotation, breadth, diffusion, flows, or a trade recommendation."})
	})
	mux.HandleFunc("GET /v1/marketops/sectors/{segment_id}", func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		id := r.PathValue("segment_id")
		items, err := q.ListMarketOpsSRISnapshots(r.Context(), storage.MarketOpsSRISnapshotFilter{TenantID: tenant, SegmentID: id, Limit: 1})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to read SRI segment")
			return
		}
		if len(items) == 0 {
			writeError(w, 404, "not_found", "SRI segment snapshot not found")
			return
		}
		writeJSON(w, 200, map[string]any{"snapshot": sriSnapshotResponse(items[0]), "research_only": true})
	})
	mux.HandleFunc("GET /v1/marketops/sectors/{segment_id}/history", func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		items, err := q.ListMarketOpsSRISnapshots(r.Context(), storage.MarketOpsSRISnapshotFilter{TenantID: tenant, SegmentID: r.PathValue("segment_id"), Limit: queryLimit(r, 100)})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to list SRI history")
			return
		}
		writeJSON(w, 200, map[string]any{"snapshots": sriSnapshotResponses(items), "research_only": true})
	})
}
func registerMarketOpsSRIAssetContextRoute(mux *http.ServeMux, repository storage.QueryRepository) {
	q, ok := repository.(interface {
		storage.MarketOpsSRIRepository
		ListMarketOpsAssets(context.Context, string, string, bool, int) ([]storage.MarketOpsAssetRecord, error)
	})
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/marketops/assets/{symbol}/sector-context", func(w http.ResponseWriter, r *http.Request) {
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		symbol := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
		assets, err := q.ListMarketOpsAssets(r.Context(), tenant, "all_active", true, 500)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to resolve asset context")
			return
		}
		var asset *storage.MarketOpsAssetRecord
		for i := range assets {
			if strings.EqualFold(assets[i].Ticker, symbol) {
				asset = &assets[i]
				break
			}
		}
		if asset == nil {
			writeJSON(w, 200, map[string]any{"symbol": symbol, "context": "unmapped", "research_only": true})
			return
		}
		segments, err := q.ListMarketOpsSRISegments(r.Context(), tenant, true, 100)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to resolve SRI segment")
			return
		}
		key := sriKey(asset.Industry)
		if key == "" {
			key = sriKey(asset.Sector)
		}
		var id string
		for _, segment := range segments {
			if segment.SegmentKey == "industry_"+key || segment.SegmentKey == "sector_"+key {
				id = segment.SegmentID
				break
			}
		}
		if id == "" {
			writeJSON(w, 200, map[string]any{"symbol": symbol, "sector": asset.Sector, "industry": asset.Industry, "context": "unmapped", "research_only": true})
			return
		}
		snapshots, err := q.ListMarketOpsSRISnapshots(r.Context(), storage.MarketOpsSRISnapshotFilter{TenantID: tenant, SegmentID: id, Limit: 1})
		if err != nil {
			writeError(w, 500, "query_failed", "failed to read SRI context")
			return
		}
		if len(snapshots) == 0 {
			writeJSON(w, 200, map[string]any{"symbol": symbol, "segment_id": id, "context": "not_ready", "research_only": true})
			return
		}
		writeJSON(w, 200, map[string]any{"symbol": symbol, "segment_id": id, "context": "informational", "snapshot": sriSnapshotResponse(snapshots[0]), "research_only": true})
	})
}
func sriKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " & ", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return strings.ReplaceAll(value, "-", "_")
}

func sriSegmentResponses(items []storage.MarketOpsSRISegmentRecord) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		out = append(out, map[string]any{"segment_id": x.SegmentID, "segment_key": x.SegmentKey, "name": x.Name, "segment_type": x.SegmentType, "parent_segment_key": x.ParentSegmentKey, "active": x.Active, "registry_version": x.RegistryVersion})
	}
	return out
}
func sriSnapshotResponse(x storage.MarketOpsSRISnapshotRecord) map[string]any {
	return map[string]any{"snapshot_id": x.SnapshotID, "segment_id": x.SegmentID, "session_date": x.SessionDate.Format("2006-01-02"), "as_of": x.AsOfTime.UTC().Format(time.RFC3339), "state": x.State, "composite_score": x.CompositeScore, "relative_strength_score": x.RelativeStrengthScore, "momentum_score": x.MomentumScore, "momentum_acceleration": x.MomentumAcceleration, "rank": x.Rank, "rank_change_5d": x.RankChange5D, "evidence_quality": x.EvidenceQuality, "quality_state": x.QualityState, "quality_flags": jsonRawOrEmptyArray(x.QualityFlagsJSON), "components": jsonRawOrEmptyObject(x.ComponentsJSON), "input_provenance": jsonRawOrEmptyObject(x.InputProvenanceJSON), "algorithm_version": x.AlgorithmVersion, "configuration_version": x.ConfigurationVersion}
}
func sriSnapshotResponses(items []storage.MarketOpsSRISnapshotRecord) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		out = append(out, sriSnapshotResponse(x))
	}
	return out
}
func sriLatestResponses(items []storage.MarketOpsSRISnapshotRecord) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	day := items[0].SessionDate
	out := []map[string]any{}
	for _, x := range items {
		if x.SessionDate.Equal(day) {
			out = append(out, sriSnapshotResponse(x))
		}
	}
	return out
}
