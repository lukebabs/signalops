package api

import (
	"encoding/json"
	"fmt"
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
	return map[string]any{"generated_at": now, "internal_cidrs": cidrs, "baseline_days": 7, "learning": len(cidrs) > 0, "device_count": len(devices), "flows": out}
}
func inCIDRs(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
