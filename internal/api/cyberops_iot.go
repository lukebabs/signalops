package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/cyberops/detection"
	"github.com/lukebabs/signalops/internal/storage"
)

type cyberOpsIoTConfigRequest struct {
	InternalCIDRs []string `json:"internal_cidrs"`
}

func registerCyberOpsIoTRoutes(mux *http.ServeMux, repo storage.QueryRepository) {
	configRepo, configOK := any(repo).(storage.CyberOpsIoTRepository)
	trafficRepo, trafficOK := any(repo).(storage.CyberOpsTrafficRepository)
	authorize := func(w http.ResponseWriter, r *http.Request) (string, bool) {
		tenant := strings.TrimSpace(r.PathValue("tenant_id"))
		p, ok := principalFromContext(r.Context())
		if !ok || p.TenantID != tenant {
			writeError(w, http.StatusForbidden, "tenant_scope_mismatch", "tenant scope mismatch")
			return "", false
		}
		return tenant, true
	}
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/iot/network-config", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := authorize(w, r)
		if !ok {
			return
		}
		if !configOK {
			writeError(w, 501, "iot_config_unavailable", "iot configuration is unavailable")
			return
		}
		item, err := configRepo.GetCyberOpsIoTNetworkConfig(r.Context(), tenant)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load iot configuration")
			return
		}
		writeJSON(w, 200, map[string]any{"network_config": map[string]any{"tenant_id": item.TenantID, "internal_cidrs": item.InternalCIDRs, "updated_at": item.UpdatedAt}})
	})
	mux.HandleFunc("PUT /v1/tenants/{tenant_id}/cyberops/iot/network-config", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := authorize(w, r)
		if !ok {
			return
		}
		if !configOK {
			writeError(w, 501, "iot_config_unavailable", "iot configuration is unavailable")
			return
		}
		var req cyberOpsIoTConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid_json", "invalid network configuration")
			return
		}
		cidrs, err := validIoTCIDRs(req.InternalCIDRs)
		if err != nil {
			writeError(w, 400, "invalid_cidrs", err.Error())
			return
		}
		item := storage.CyberOpsIoTNetworkConfig{TenantID: tenant, InternalCIDRs: cidrs}
		if err := configRepo.UpsertCyberOpsIoTNetworkConfig(r.Context(), item); err != nil {
			writeError(w, 500, "persistence_failed", "failed to save iot configuration")
			return
		}
		writeJSON(w, 200, map[string]any{"network_config": map[string]any{"tenant_id": tenant, "internal_cidrs": cidrs}})
	})
	mux.HandleFunc("GET /v1/tenants/{tenant_id}/cyberops/iot/behaviour", func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := authorize(w, r)
		if !ok {
			return
		}
		if !configOK || !trafficOK {
			writeError(w, 501, "iot_behaviour_unavailable", "iot behaviour is unavailable")
			return
		}
		config, err := configRepo.GetCyberOpsIoTNetworkConfig(r.Context(), tenant)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load iot configuration")
			return
		}
		cidrs, err := validIoTCIDRs(config.InternalCIDRs)
		if err != nil {
			writeError(w, 400, "invalid_cidrs", err.Error())
			return
		}
		now := time.Now().UTC()
		inputs, err := trafficRepo.ListCyberOpsTrafficInputs(r.Context(), tenant, now.Add(-14*24*time.Hour), now)
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load firewall traffic")
			return
		}
		writeJSON(w, 200, buildIoTBehaviour(inputs, cidrs, now))
	})
}
func validIoTCIDRs(values []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range values {
		p, err := netip.ParsePrefix(strings.TrimSpace(v))
		if err != nil {
			return nil, err
		}
		p = p.Masked()
		if seen[p.String()] {
			return nil, fmt.Errorf("duplicate CIDR %s", p)
		}
		seen[p.String()] = true
		out = append(out, p.String())
	}
	return out, nil
}
func buildIoTBehaviour(inputs []storage.CyberOpsTrafficInput, cidrs []string, now time.Time) map[string]any {
	prefixes := []netip.Prefix{}
	for _, v := range cidrs {
		p, _ := netip.ParsePrefix(v)
		prefixes = append(prefixes, p)
	}
	type flow struct {
		DeviceIP        string    `json:"device_ip"`
		PeerIP          string    `json:"peer_ip"`
		Direction       string    `json:"direction"`
		Protocol        string    `json:"protocol"`
		DestinationPort int       `json:"destination_port"`
		Count           int       `json:"count"`
		FirstSeen       time.Time `json:"first_seen"`
		LastSeen        time.Time `json:"last_seen"`
	}
	flows := map[string]*flow{}
	devices := map[string]int{}
	cutoff := now.Add(-24 * time.Hour)
	for _, in := range inputs {
		f, ok := detection.ParseOPNsenseFilterlog(in.Message)
		if !ok {
			continue
		}
		s, _ := netip.ParseAddr(f.SourceIP)
		d, _ := netip.ParseAddr(f.DestinationIP)
		si, di := inCIDRs(s, prefixes), inCIDRs(d, prefixes)
		add := func(device, peer, dir string) {
			key := device + "|" + peer + "|" + dir + "|" + f.Protocol + "|" + strconv.Itoa(f.DestinationPort)
			x := flows[key]
			if x == nil {
				x = &flow{DeviceIP: device, PeerIP: peer, Direction: dir, Protocol: f.Protocol, DestinationPort: f.DestinationPort, FirstSeen: in.ObservedAt}
				flows[key] = x
			}
			x.Count++
			if in.ObservedAt.After(x.LastSeen) {
				x.LastSeen = in.ObservedAt
			}
			if in.ObservedAt.Before(x.FirstSeen) {
				x.FirstSeen = in.ObservedAt
			}
			if in.ObservedAt.After(cutoff) {
				devices[device]++
			}
		}
		if si && di {
			add(f.SourceIP, f.DestinationIP, "lateral")
			add(f.DestinationIP, f.SourceIP, "lateral")
		} else if si {
			add(f.SourceIP, f.DestinationIP, "egress")
		} else if di {
			add(f.DestinationIP, f.SourceIP, "ingress")
		}
	}
	out := []flow{}
	for _, x := range flows {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 100 {
		out = out[:100]
	}
	return map[string]any{"generated_at": now, "internal_cidrs": cidrs, "baseline_days": 7, "learning": len(cidrs) > 0, "device_count": len(devices), "flows": out, "learning_devices": buildIoTLearningDevices(inputs, prefixes, now), "novel_services": buildIoTNovelServices(inputs, prefixes, now)}
}

type iotLearningDevice struct {
	DeviceIP            string   `json:"device_ip"`
	BaselineActiveHours int      `json:"baseline_active_hours"`
	RequiredActiveHours int      `json:"required_active_hours"`
	BaselineProgress    float64  `json:"baseline_progress"`
	CurrentHour         string   `json:"current_hour"`
	Observed            int      `json:"observed"`
	Metric              string   `json:"metric"`
	BaselineMean        float64  `json:"baseline_mean"`
	BaselineStddev      float64  `json:"baseline_stddev"`
	ZScore              *float64 `json:"z_score,omitempty"`
	Threshold           float64  `json:"threshold"`
	Status              string   `json:"status"`
	WithheldReason      string   `json:"withheld_reason,omitempty"`
}

type iotHourValue struct {
	count int
	peers map[string]bool
}

// buildIoTLearningDevices mirrors the hourly anomaly worker's seven-day,
// 24-active-hour, three-sigma rules without creating a result or signal.
func buildIoTLearningDevices(inputs []storage.CyberOpsTrafficInput, prefixes []netip.Prefix, now time.Time) []iotLearningDevice {
	values := map[string]map[time.Time]*iotHourValue{}
	for _, input := range inputs {
		f, ok := detection.ParseOPNsenseFilterlog(input.Message)
		if !ok {
			continue
		}
		source, _ := netip.ParseAddr(f.SourceIP)
		destination, _ := netip.ParseAddr(f.DestinationIP)
		pairs := [][2]string{}
		if inCIDRs(source, prefixes) {
			pairs = append(pairs, [2]string{f.SourceIP, f.DestinationIP})
		}
		if inCIDRs(destination, prefixes) && f.SourceIP != f.DestinationIP {
			pairs = append(pairs, [2]string{f.DestinationIP, f.SourceIP})
		}
		for _, pair := range pairs {
			hour := input.ObservedAt.UTC().Truncate(time.Hour)
			if values[pair[0]] == nil {
				values[pair[0]] = map[time.Time]*iotHourValue{}
			}
			if values[pair[0]][hour] == nil {
				values[pair[0]][hour] = &iotHourValue{peers: map[string]bool{}}
			}
			value := values[pair[0]][hour]
			value.count++
			value.peers[pair[1]] = true
		}
	}
	end := now.UTC().Truncate(time.Hour)
	target := end.Add(-time.Hour)
	start := end.Add(-8 * 24 * time.Hour)
	out := make([]iotLearningDevice, 0, len(values))
	for device, series := range values {
		activeHours := 0
		counts := make([]float64, 0, 7*24)
		peers := make([]float64, 0, 7*24)
		for hour := start; hour.Before(target); hour = hour.Add(time.Hour) {
			value := series[hour]
			if value == nil {
				counts = append(counts, 0)
				peers = append(peers, 0)
				continue
			}
			activeHours++
			counts = append(counts, float64(value.count))
			peers = append(peers, float64(len(value.peers)))
		}
		current := series[target]
		observedCount, observedPeers := 0, 0
		if current != nil {
			observedCount, observedPeers = current.count, len(current.peers)
		}
		metric, observed, baseline := "allowed_log_count", observedCount, counts
		if _, countStddev := iotStats(counts); countStddev == 0 {
			metric, observed, baseline = "distinct_peers", observedPeers, peers
		}
		mean, stddev := iotStats(baseline)
		item := iotLearningDevice{DeviceIP: device, BaselineActiveHours: activeHours, RequiredActiveHours: 24, BaselineProgress: math.Min(1, float64(activeHours)/24), CurrentHour: target.Format(time.RFC3339), Observed: observed, Metric: metric, BaselineMean: mean, BaselineStddev: stddev, Threshold: 3, Status: "learning"}
		if activeHours < 24 {
			item.WithheldReason = fmt.Sprintf("Learning: %d of 24 required active baseline hours", activeHours)
		} else if current == nil {
			item.Status = "waiting_for_completed_hour"
			item.WithheldReason = "No activity in the latest completed hour"
		} else if stddev == 0 {
			item.Status = "baseline_ready"
			item.WithheldReason = "Baseline has no variation yet"
		} else {
			score := math.Abs((float64(observed) - mean) / stddev)
			item.ZScore = &score
			item.Status = "baseline_ready"
			if score >= item.Threshold {
				item.Status = "threshold_met"
			} else {
				item.WithheldReason = fmt.Sprintf("Below %.0fσ threshold", item.Threshold)
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status == "learning" && out[j].Status != "learning" {
			return true
		}
		if out[j].Status == "learning" && out[i].Status != "learning" {
			return false
		}
		if out[i].BaselineProgress != out[j].BaselineProgress {
			return out[i].BaselineProgress < out[j].BaselineProgress
		}
		return out[i].DeviceIP < out[j].DeviceIP
	})
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

func iotStats(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}

type iotNovelService struct {
	DeviceIP        string    `json:"device_ip"`
	PeerIP          string    `json:"peer_ip"`
	Protocol        string    `json:"protocol"`
	DestinationPort int       `json:"destination_port"`
	Count           int       `json:"count"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	Status          string    `json:"status"`
}

// buildIoTNovelServices identifies recurring services in the latest completed hour that were absent from the prior seven completed days.
func buildIoTNovelServices(inputs []storage.CyberOpsTrafficInput, prefixes []netip.Prefix, now time.Time) []iotNovelService {
	end := now.UTC().Truncate(time.Hour)
	target := end.Add(-time.Hour)
	start := target.Add(-7 * 24 * time.Hour)
	history := map[string]bool{}
	current := map[string]*iotNovelService{}
	for _, input := range inputs {
		hour := input.ObservedAt.UTC().Truncate(time.Hour)
		if hour.Before(start) || !hour.Equal(target) && !hour.Before(target) {
			continue
		}
		f, ok := detection.ParseOPNsenseFilterlog(input.Message)
		if !ok {
			continue
		}
		source, _ := netip.ParseAddr(f.SourceIP)
		destination, _ := netip.ParseAddr(f.DestinationIP)
		pairs := [][2]string{}
		if inCIDRs(source, prefixes) {
			pairs = append(pairs, [2]string{f.SourceIP, f.DestinationIP})
		}
		if inCIDRs(destination, prefixes) && f.SourceIP != f.DestinationIP {
			pairs = append(pairs, [2]string{f.DestinationIP, f.SourceIP})
		}
		for _, pair := range pairs {
			key := pair[0] + "|" + pair[1] + "|" + f.Protocol + "|" + strconv.Itoa(f.DestinationPort)
			if hour.Before(target) {
				history[key] = true
				continue
			}
			item := current[key]
			if item == nil {
				item = &iotNovelService{DeviceIP: pair[0], PeerIP: pair[1], Protocol: f.Protocol, DestinationPort: f.DestinationPort, FirstSeen: input.ObservedAt, LastSeen: input.ObservedAt, Status: "new_peer_service"}
				current[key] = item
			}
			item.Count++
			if input.ObservedAt.Before(item.FirstSeen) {
				item.FirstSeen = input.ObservedAt
			}
			if input.ObservedAt.After(item.LastSeen) {
				item.LastSeen = input.ObservedAt
			}
		}
	}
	out := []iotNovelService{}
	for key, item := range current {
		if !history[key] && item.Count >= 3 {
			out = append(out, *item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 25 {
		out = out[:25]
	}
	return out
}

func inCIDRs(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
