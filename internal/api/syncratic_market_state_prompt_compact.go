package api

import (
	"encoding/json"
)

func compactMarketStatePromptContext(raw []byte) (json.RawMessage, map[string]any) {
	var source map[string]any
	if json.Unmarshal(raw, &source) != nil {
		return json.RawMessage(`{}`), map[string]any{"invalid_lineage_json": true}
	}
	limits := map[string]int{"features": 8, "state_transitions": 12, "hypothesis_evaluations": 8, "marketops_evidence": 8, "opportunities": 5, "outcomes": 8, "calibration_summaries": 8}
	meta := map[string]any{}
	for key, limit := range limits {
		items, ok := source[key].([]any)
		if !ok {
			continue
		}
		available := len(items)
		if available > limit {
			items = items[:limit]
		}
		for _, item := range items {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if key == "features" {
				delete(record, "source_event_ids")
				delete(record, "source_artifact_ids")
			}
			if key == "hypothesis_evaluations" {
				reasons, ok := record["reason_codes"].([]any)
				if ok && len(reasons) > 5 {
					record["reason_codes"] = reasons[:5]
					record["omitted_reason_code_count"] = len(reasons) - 5
				}
			}
		}
		source[key] = items
		meta[key] = map[string]any{"available": available, "included": len(items), "truncated": available > len(items), "omitted": max(0, available-len(items))}
	}
	return json.RawMessage(mustJSON(source)), meta
}
