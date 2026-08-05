package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestCurateRiskRewardSnapshotSummariesKeepsPriorSessionForFullUniverse(t *testing.T) {
	active := map[string]struct{}{}
	snapshots := make([]storage.MarketOpsRiskRewardSnapshotRecord, 0, 230)
	latest := time.Date(2026, 8, 4, 20, 0, 0, 0, time.UTC)
	prior := time.Date(2026, 8, 3, 20, 0, 0, 0, time.UTC)
	for index := 0; index < 115; index++ {
		symbol := fmt.Sprintf("T%03d", index)
		active[symbol] = struct{}{}
		snapshots = append(snapshots,
			storage.MarketOpsRiskRewardSnapshotRecord{Symbol: symbol, SessionDate: latest, TechnicalScore: 20, TechnicalDirection: "bullish", RiskLevel: "medium", Confidence: 0.8, UsableInputCount: 5, Eligible: true, CreatedAt: latest.Add(time.Hour)},
			storage.MarketOpsRiskRewardSnapshotRecord{Symbol: symbol, SessionDate: prior, TechnicalScore: 5, TechnicalDirection: "bullish", RiskLevel: "medium", Confidence: 0.8, UsableInputCount: 5, Eligible: true, CreatedAt: prior.Add(time.Hour)},
		)
	}

	summaries := curateRiskRewardSnapshotSummaries(snapshots, active)
	if len(summaries) != len(active) {
		t.Fatalf("summary count = %d, want %d", len(summaries), len(active))
	}
	for _, summary := range summaries {
		if got, ok := summary["score_change"].(float64); !ok || got != 15 {
			t.Fatalf("summary %#v missing expected prior-session score change", summary)
		}
		if got := summary["previous_trade_date"]; got != "2026-08-03" {
			t.Fatalf("previous trade date = %v, want 2026-08-03", got)
		}
	}
}
