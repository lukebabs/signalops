package anomaly

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/lukebabs/signalops/internal/cyberops/detection"
	"github.com/lukebabs/signalops/internal/storage"
	"math"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"
)

const AlgorithmID = "signalops.algorithms.zscore_anomaly_v1"

type Repository interface {
	storage.AlgorithmRepository
	storage.CyberOpsTrafficRepository
	storage.CyberOpsIoTRepository
}
type hourValue struct {
	count  int
	peers  map[string]bool
	events []string
}

func Run(ctx context.Context, repo Repository, now time.Time) (int, error) {
	configs, err := repo.ListCyberOpsIoTNetworkConfigs(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	end := now.UTC().Truncate(time.Hour)
	for _, config := range configs {
		n, err := runTenant(ctx, repo, config, end)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
func runTenant(ctx context.Context, repo Repository, config storage.CyberOpsIoTNetworkConfig, end time.Time) (int, error) {
	prefixes := []netip.Prefix{}
	for _, text := range config.InternalCIDRs {
		p, err := netip.ParsePrefix(text)
		if err != nil {
			return 0, err
		}
		prefixes = append(prefixes, p.Masked())
	}
	if len(prefixes) == 0 {
		return 0, nil
	}
	inputs, err := repo.ListCyberOpsTrafficInputs(ctx, config.TenantID, end.Add(-8*24*time.Hour), end)
	if err != nil {
		return 0, err
	}
	values := map[string]map[time.Time]*hourValue{}
	inside := func(a netip.Addr) bool {
		for _, p := range prefixes {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	for _, in := range inputs {
		f, ok := detection.ParseOPNsenseFilterlog(in.Message)
		if !ok {
			continue
		}
		s, _ := netip.ParseAddr(f.SourceIP)
		d, _ := netip.ParseAddr(f.DestinationIP)
		devices := [][2]string{}
		if inside(s) {
			devices = append(devices, [2]string{f.SourceIP, f.DestinationIP})
		}
		if inside(d) && f.SourceIP != f.DestinationIP {
			devices = append(devices, [2]string{f.DestinationIP, f.SourceIP})
		}
		for _, pair := range devices {
			bucket := in.ObservedAt.UTC().Truncate(time.Hour)
			if values[pair[0]] == nil {
				values[pair[0]] = map[time.Time]*hourValue{}
			}
			if values[pair[0]][bucket] == nil {
				values[pair[0]][bucket] = &hourValue{peers: map[string]bool{}}
			}
			v := values[pair[0]][bucket]
			v.count++
			v.peers[pair[1]] = true
			if in.EventID != "" {
				v.events = append(v.events, in.EventID)
			}
		}
	}
	written := 0
	target := end.Add(-time.Hour)
	for device, series := range values {
		nonzero := 0
		counts := []float64{}
		fans := []float64{}
		for hour := end.Add(-8 * 24 * time.Hour); hour.Before(target); hour = hour.Add(time.Hour) {
			v := series[hour]
			if v == nil {
				counts = append(counts, 0)
				fans = append(fans, 0)
			} else {
				counts = append(counts, float64(v.count))
				fans = append(fans, float64(len(v.peers)))
				nonzero++
			}
		}
		if nonzero < 24 || series[target] == nil {
			continue
		}
		for _, metric := range []struct {
			name     string
			baseline []float64
			observed float64
		}{{"allowed_log_count", counts, float64(series[target].count)}, {"distinct_peers", fans, float64(len(series[target].peers))}} {
			mean, std := stats(metric.baseline)
			if std == 0 {
				continue
			}
			score := math.Abs((metric.observed - mean) / std)
			if score < 3 {
				continue
			}
			severity := "medium"
			if score >= 5 {
				severity = "high"
			}
			payload, _ := json.Marshal(map[string]any{"device_ip": device, "metric": metric.name, "observed": metric.observed, "baseline_mean": mean, "baseline_stddev": std, "z_score": score, "baseline_hours": len(metric.baseline), "hour": target})
			execution := "cybiot_" + short(config.TenantID+"|"+target.Format(time.RFC3339)+"|"+metric.name)
			_ = repo.UpsertAlgorithmExecutionRequest(ctx, storage.AlgorithmExecutionRequestRecord{ExecutionRequestID: execution, TenantID: config.TenantID, AlgorithmID: AlgorithmID, AlgorithmVersion: "v1", EntityRefs: []string{"device:" + device}, FeatureRefs: []string{"iot:" + metric.name}, WindowRef: target.Format(time.RFC3339), ConfigJSON: []byte(`{"z_threshold":3,"baseline_days":7}`), CorrelationID: execution, Status: storage.AlgorithmExecutionStatusSucceeded, RequestedBy: "cyberops-iot-anomaly-worker", ResultJSON: []byte(`{"mode":"hourly_iot_anomaly"}`)})
			sort.Strings(series[target].events)
			id := "cybiotalg_" + short(config.TenantID+"|"+device+"|"+metric.name+"|"+target.Format(time.RFC3339))
			if err := repo.InsertAlgorithmResult(ctx, storage.AlgorithmResultRecord{AlgorithmResultID: id, TenantID: config.TenantID, AlgorithmID: AlgorithmID, AlgorithmVersion: "v1", ExecutionRequestID: execution, ResultType: "z_score", Score: score, Confidence: math.Min(0.99, score/5), Severity: severity, ResultPayloadJSON: payload, SourceEventIDs: series[target].events, EvidenceRefs: []string{"cyberops_iot_device:" + device, "hour:" + target.Format(time.RFC3339)}, CorrelationID: execution}); err != nil {
				return written, err
			}
			written++
		}
	}
	return written, nil
}
func stats(v []float64) (float64, float64) {
	var sum float64
	for _, x := range v {
		sum += x
	}
	mean := sum / float64(len(v))
	var variance float64
	for _, x := range v {
		variance += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(variance / float64(len(v)))
}
func short(v string) string { s := sha256.Sum256([]byte(v)); return hex.EncodeToString(s[:12]) }

var _ = strconv.Itoa
var _ = strings.TrimSpace
