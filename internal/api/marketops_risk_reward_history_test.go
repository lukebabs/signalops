package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestRiskRewardOverviewPrefersBestEvidenceOverLaterNeutralRevision(t *testing.T) {
	observed := "2026-07-28T20:00:00Z"
	payload := func(score float64, direction string, usable int) []byte {
		value, err := json.Marshal(map[string]any{
			"symbol": "AMAT", "observation_time": observed, "technical_score": score, "technical_direction": direction,
			"confidence_basis": map[string]any{"usable_technical_inputs": usable, "required_technical_inputs": 8},
		})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	results := []storage.AlgorithmResultRecord{
		{AlgorithmResultID: "valid", AlgorithmID: riskRewardAlgorithmID, ResultPayloadJSON: payload(50, "bullish", 5), CreatedAt: time.Date(2026, 7, 28, 22, 0, 0, 0, time.UTC)},
		{AlgorithmResultID: "degraded", AlgorithmID: riskRewardAlgorithmID, ResultPayloadJSON: payload(0, "neutral", 2), CreatedAt: time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)},
	}
	points := signalOverviewRiskRewardBest(results, map[string]struct{}{"AMAT": {}}, 10)
	if len(points) != 1 {
		t.Fatalf("point count = %d, want 1", len(points))
	}
	coverage := points[0]["coverage"].(map[string]int)
	if coverage["eligible"] != 1 || coverage["insufficient_inputs"] != 0 {
		t.Fatalf("coverage = %#v, want one eligible selected candidate", coverage)
	}
	categories := points[0]["categories"].([]map[string]any)
	if got := categories[0]["count"]; got != 1 {
		t.Fatalf("bullish count = %v, want 1", got)
	}
}

func TestRiskRewardOverviewExposesInsufficientInputsInsteadOfNeutral(t *testing.T) {
	payload := []byte(`{"symbol":"AMAT","observation_time":"2026-07-28T20:00:00Z","technical_score":0,"technical_direction":"neutral","confidence_basis":{"usable_technical_inputs":2,"required_technical_inputs":8}}`)
	points := signalOverviewRiskRewardBest([]storage.AlgorithmResultRecord{{AlgorithmID: riskRewardAlgorithmID, ResultPayloadJSON: payload}}, map[string]struct{}{"AMAT": {}}, 10)
	if len(points) != 1 {
		t.Fatalf("point count = %d, want 1", len(points))
	}
	coverage := points[0]["coverage"].(map[string]int)
	if coverage["eligible"] != 0 || coverage["insufficient_inputs"] != 1 {
		t.Fatalf("coverage = %#v, want insufficient input coverage", coverage)
	}
	for _, category := range points[0]["categories"].([]map[string]any) {
		if category["count"].(int) != 0 {
			t.Fatalf("insufficient input revision appeared in category %#v", category)
		}
	}
}
