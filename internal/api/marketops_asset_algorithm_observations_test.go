package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestCurateAssetAlgorithmObservationsSelectsOneUsableZScorePerLatestThreeDates(t *testing.T) {
	at := time.Date(2026, 7, 22, 4, 0, 0, 0, time.UTC)
	result := func(id, algorithm, day, feature, quality string, confidence, score float64, severity string) storage.AlgorithmResultRecord {
		payload, _ := json.Marshal(map[string]any{"symbol": "NVDA", "observation_time": day + "T00:00:00Z", "feature": feature, "call_put_oi_ratio_quality": quality})
		return storage.AlgorithmResultRecord{AlgorithmResultID: id, AlgorithmID: algorithm, ResultType: "anomaly", Confidence: confidence, Score: score, Severity: severity, ResultPayloadJSON: payload, CreatedAt: at}
	}
	results := []storage.AlgorithmResultRecord{
		result("bad-new", zscoreAlgorithmID, "2026-07-21", "call_put_open_interest_ratio", "all_zero", 0.99, 9, "critical"),
		result("good-new", zscoreAlgorithmID, "2026-07-21", "open_close_move_pct", "", 0.51, 0.77, "medium"),
		result("middle", zscoreAlgorithmID, "2026-07-20", "open_close_move_pct", "", 0.45, 0.65, "low"),
		result("old", zscoreAlgorithmID, "2026-07-19", "open_close_move_pct", "", 0.35, 0.50, "low"),
		result("outside-window", zscoreAlgorithmID, "2026-07-18", "open_close_move_pct", "", 0.90, 1.2, "high"),
		result("river", "signalops.algorithms.river_anomaly_v1", "2026-07-21", "open_close_move_pct", "", 0.10, 0.1, "info"),
	}
	eod, other := curateAssetAlgorithmObservations(results, "NVDA")
	if len(eod) != 3 {
		t.Fatalf("expected three EOD dates, got %d", len(eod))
	}
	if eod[0].TradeDate != "2026-07-21" || eod[0].AlgorithmResult == nil || eod[0].AlgorithmResult.AlgorithmResultID != "good-new" {
		t.Fatalf("expected usable selected newest result, got %#v", eod[0])
	}
	if eod[1].TradeDate != "2026-07-20" || eod[2].TradeDate != "2026-07-19" {
		t.Fatalf("unexpected EOD date order: %#v", eod)
	}
	ids := map[string]bool{}
	for _, item := range other {
		ids[item.AlgorithmResultID] = true
	}
	if !ids["bad-new"] || !ids["river"] || ids["good-new"] {
		t.Fatalf("raw evidence preservation mismatch: %#v", ids)
	}
}
