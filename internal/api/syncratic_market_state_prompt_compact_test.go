package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMarketStateAskV2CompactsPromptWithTruncationMetadata(t *testing.T) {
	repo := marketStateContextFixture()
	record, err := buildMarketStateSyncraticContext(context.Background(), repo, "tenant-1", "mstate-aapl-1", "")
	if err != nil {
		t.Fatal(err)
	}
	var lineage map[string]any
	if err := json.Unmarshal(record.LineageRefsJSON, &lineage); err != nil {
		t.Fatal(err)
	}
	template := lineage["features"].([]any)[0]
	features := make([]any, 30)
	for i := range features {
		features[i] = template
	}
	lineage["features"] = features
	record.LineageRefsJSON = mustJSON(lineage)
	prompt, meta, err := buildMarketStateAskPrompt(record, syncraticAskRequest{MaxPromptBytes: 12000})
	if err != nil {
		t.Fatal(err)
	}
	if len(prompt) > 12000 || meta.PromptBytes != len(prompt) || !strings.Contains(prompt, `"truncated":true`) {
		t.Fatalf("bytes=%d prompt=%s", len(prompt), prompt)
	}
}
