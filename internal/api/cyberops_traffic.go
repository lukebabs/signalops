package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/cyberops/detection"
	"github.com/lukebabs/signalops/internal/storage"
)

type cyberOpsTrafficFlow struct {
	SourceIP        string    `json:"source_ip"`
	DestinationIP   string    `json:"destination_ip"`
	Protocol        string    `json:"protocol"`
	DestinationPort int       `json:"destination_port"`
	Count           int       `json:"count"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
}
type cyberOpsTrafficCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}
type cyberOpsTrafficPoint struct {
	Time  time.Time `json:"time"`
	Count int       `json:"count"`
}

const (
	cyberOpsLiveTrafficWindow   = 30 * time.Minute
	cyberOpsLiveTrafficBucket   = time.Minute
	cyberOpsLiveTrafficInterval = 5 * time.Second
)

type cyberOpsLiveTrafficPoint struct {
	Time         time.Time `json:"time"`
	ReceivedLogs int       `json:"received_logs"`
	AllowedLogs  int       `json:"allowed_events"`
	UnparsedLogs int       `json:"unparsed_logs"`
}

type cyberOpsLiveTrafficSnapshot struct {
	GeneratedAt    time.Time                  `json:"generated_at"`
	LastObservedAt *time.Time                 `json:"last_observed_at,omitempty"`
	Points         []cyberOpsLiveTrafficPoint `json:"points"`
}

func registerCyberOpsTrafficRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/traffic-overview", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(storage.CyberOpsTrafficRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "cyberops_traffic_unavailable", "cyberops traffic overview is unavailable")
			return
		}
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		principal, authenticated := principalFromContext(r.Context())
		if authenticated && principal.TenantID != tenant {
			writeError(w, http.StatusForbidden, "tenant_scope_mismatch", "tenant scope mismatch")
			return
		}
		if tenant == "" {
			writeError(w, http.StatusBadRequest, "missing_tenant", "tenant is required")
			return
		}
		window, duration := cyberOpsTrafficWindow(r.URL.Query().Get("window"))
		if duration == 0 {
			writeError(w, http.StatusBadRequest, "invalid_window", "window must be 1h, 24h, or 7d")
			return
		}
		now := time.Now().UTC()
		inputs, err := reader.ListCyberOpsTrafficInputs(r.Context(), tenant, now.Add(-duration), now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query_failed", "failed to build cyberops traffic overview")
			return
		}
		writeJSON(w, http.StatusOK, buildCyberOpsTrafficOverview(inputs, window, now, duration))
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/live-traffic", func(w http.ResponseWriter, r *http.Request) {
		reader, ok := any(repo).(storage.CyberOpsTrafficRepository)
		if !ok {
			writeError(w, http.StatusNotImplemented, "cyberops_traffic_unavailable", "cyberops traffic overview is unavailable")
			return
		}
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		principal, authenticated := principalFromContext(r.Context())
		if authenticated && principal.TenantID != tenant {
			writeError(w, http.StatusForbidden, "tenant_scope_mismatch", "tenant scope mismatch")
			return
		}
		if tenant == "" {
			writeError(w, http.StatusBadRequest, "missing_tenant", "tenant is required")
			return
		}
		streamCyberOpsLiveTraffic(w, r, reader, tenant, cyberOpsLiveTrafficInterval)
	})
}

func streamCyberOpsLiveTraffic(w http.ResponseWriter, r *http.Request, reader storage.CyberOpsTrafficRepository, tenant string, interval time.Duration) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_unsupported", "response streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	emit := func(event sseEvent) bool {
		if err := writeSSE(w, event); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	emitSnapshot := func(event string) bool {
		now := time.Now().UTC()
		inputs, err := reader.ListCyberOpsTrafficInputs(r.Context(), tenant, now.Add(-cyberOpsLiveTrafficWindow), now)
		if err != nil {
			_ = emit(sseEvent{Event: "error", Data: map[string]string{"error": "query_failed", "message": "failed to load live CyberOps traffic"}})
			return false
		}
		return emit(sseEvent{Event: event, Data: buildCyberOpsLiveTrafficSnapshot(inputs, now)})
	}
	if !emitSnapshot("snapshot") {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !emitSnapshot("traffic") {
				return
			}
		}
	}
}

func buildCyberOpsLiveTrafficSnapshot(inputs []storage.CyberOpsTrafficInput, generated time.Time) cyberOpsLiveTrafficSnapshot {
	end := generated.UTC().Truncate(cyberOpsLiveTrafficBucket)
	start := end.Add(-cyberOpsLiveTrafficWindow).Add(cyberOpsLiveTrafficBucket)
	points := make(map[time.Time]*cyberOpsLiveTrafficPoint, int(cyberOpsLiveTrafficWindow/cyberOpsLiveTrafficBucket))
	ordered := make([]cyberOpsLiveTrafficPoint, 0, int(cyberOpsLiveTrafficWindow/cyberOpsLiveTrafficBucket))
	for at := start; !at.After(end); at = at.Add(cyberOpsLiveTrafficBucket) {
		point := &cyberOpsLiveTrafficPoint{Time: at}
		points[at] = point
		ordered = append(ordered, *point)
	}
	var lastObservedAt *time.Time
	for _, input := range inputs {
		observed := input.ObservedAt.UTC()
		bucket := observed.Truncate(cyberOpsLiveTrafficBucket)
		point := points[bucket]
		if point == nil {
			continue
		}
		point.ReceivedLogs++
		if _, ok := detection.ParseOPNsenseFilterlog(input.Message); ok {
			point.AllowedLogs++
		} else {
			point.UnparsedLogs++
		}
		if lastObservedAt == nil || observed.After(*lastObservedAt) {
			value := observed
			lastObservedAt = &value
		}
	}
	for index, point := range ordered {
		ordered[index] = *points[point.Time]
	}
	return cyberOpsLiveTrafficSnapshot{GeneratedAt: generated.UTC(), LastObservedAt: lastObservedAt, Points: ordered}
}

func cyberOpsTrafficWindow(value string) (string, time.Duration) {
	switch value {
	case "", "24h":
		return "24h", 24 * time.Hour
	case "1h":
		return value, time.Hour
	case "7d":
		return value, 7 * 24 * time.Hour
	default:
		return "", 0
	}
}
func buildCyberOpsTrafficOverview(inputs []storage.CyberOpsTrafficInput, window string, generated time.Time, duration time.Duration) map[string]any {
	sources, destinations, protocols, ports, flows, points := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]*cyberOpsTrafficFlow{}, map[time.Time]int{}
	parsed := 0
	bucket := time.Hour
	if duration == time.Hour {
		bucket = 5 * time.Minute
	}
	for _, input := range inputs {
		firewall, ok := detection.ParseOPNsenseFilterlog(input.Message)
		if !ok {
			continue
		}
		parsed++
		sources[firewall.SourceIP]++
		destinations[firewall.DestinationIP]++
		protocols[firewall.Protocol]++
		ports[firewall.Protocol+"/"+strconv.Itoa(firewall.DestinationPort)]++
		key := firewall.SourceIP + "|" + firewall.DestinationIP + "|" + firewall.Protocol + "|" + strconv.Itoa(firewall.DestinationPort)
		flow := flows[key]
		if flow == nil {
			flow = &cyberOpsTrafficFlow{SourceIP: firewall.SourceIP, DestinationIP: firewall.DestinationIP, Protocol: firewall.Protocol, DestinationPort: firewall.DestinationPort, FirstSeen: input.ObservedAt}
			flows[key] = flow
		}
		flow.Count++
		if input.ObservedAt.Before(flow.FirstSeen) {
			flow.FirstSeen = input.ObservedAt
		}
		if input.ObservedAt.After(flow.LastSeen) {
			flow.LastSeen = input.ObservedAt
		}
		points[input.ObservedAt.UTC().Truncate(bucket)]++
	}
	return map[string]any{"generated_at": generated, "window": window, "total_logs": len(inputs), "allowed_events": parsed, "unparsed_logs": len(inputs) - parsed, "unique_sources": len(sources), "unique_destinations": len(destinations), "unique_services": len(ports), "timeline": trafficPoints(points), "top_sources": trafficCounts(sources, 10), "top_destinations": trafficCounts(destinations, 10), "protocols": trafficCounts(protocols, 10), "destination_ports": trafficCounts(ports, 10), "flows": trafficFlows(flows, 50)}
}
func trafficCounts(values map[string]int, limit int) []cyberOpsTrafficCount {
	out := make([]cyberOpsTrafficCount, 0, len(values))
	for key, count := range values {
		out = append(out, cyberOpsTrafficCount{key, count})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || out[i].Count == out[j].Count && out[i].Key < out[j].Key
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
func trafficFlows(values map[string]*cyberOpsTrafficFlow, limit int) []cyberOpsTrafficFlow {
	out := make([]cyberOpsTrafficFlow, 0, len(values))
	for _, flow := range values {
		out = append(out, *flow)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
func trafficPoints(values map[time.Time]int) []cyberOpsTrafficPoint {
	out := make([]cyberOpsTrafficPoint, 0, len(values))
	for at, count := range values {
		out = append(out, cyberOpsTrafficPoint{at, count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}
