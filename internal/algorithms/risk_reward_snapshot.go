package algorithms

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	riskRewardRequiredInputs = 8
	riskRewardMinimumUsable  = 5
)

func riskRewardSnapshotRecord(cfg Config, result storage.AlgorithmResultRecord, item struct {
	SourceEventID string         `json:"source_event_id"`
	ResultType    string         `json:"result_type"`
	Score         float64        `json:"score"`
	Confidence    float64        `json:"confidence"`
	Severity      string         `json:"severity"`
	Payload       map[string]any `json:"payload"`
}, observedAt time.Time, inputSnapshot map[string]any) (storage.MarketOpsRiskRewardSnapshotRecord, error) {
	if observedAt.IsZero() {
		return storage.MarketOpsRiskRewardSnapshotRecord{}, fmt.Errorf("risk/reward snapshot observation time is required")
	}
	payload := item.Payload
	symbol := strings.ToUpper(strings.TrimSpace(stringValue(payload["symbol"])))
	if symbol == "" {
		return storage.MarketOpsRiskRewardSnapshotRecord{}, fmt.Errorf("risk/reward snapshot symbol is required")
	}
	score, ok := numberValue(payload["technical_score"])
	if !ok {
		return storage.MarketOpsRiskRewardSnapshotRecord{}, fmt.Errorf("risk/reward snapshot technical score is required")
	}
	usable, required := 0, riskRewardRequiredInputs
	if basis, ok := payload["confidence_basis"].(map[string]any); ok {
		if value, ok := numberValue(basis["usable_technical_inputs"]); ok {
			usable = int(value)
		}
		if value, ok := numberValue(basis["required_technical_inputs"]); ok && value > 0 {
			required = int(value)
		}
	}
	resultPayload, err := json.Marshal(payload)
	if err != nil {
		return storage.MarketOpsRiskRewardSnapshotRecord{}, fmt.Errorf("encode risk/reward result payload: %w", err)
	}
	if inputSnapshot == nil {
		inputSnapshot = map[string]any{}
	}
	inputPayload, err := json.Marshal(inputSnapshot)
	if err != nil {
		return storage.MarketOpsRiskRewardSnapshotRecord{}, fmt.Errorf("encode risk/reward input snapshot: %w", err)
	}
	direction := stringValue(payload["technical_direction"])
	if direction != "bullish" && direction != "bearish" && direction != "neutral" {
		direction = "neutral"
	}
	riskLevel := stringValue(payload["risk_level"])
	if riskLevel != "low" && riskLevel != "medium" && riskLevel != "high" && riskLevel != "unavailable" {
		riskLevel = "unavailable"
	}
	return storage.MarketOpsRiskRewardSnapshotRecord{
		SnapshotID:         "rrsnap_" + stableHash(cfg.TenantID + "|" + result.AlgorithmResultID)[:32],
		TenantID:           cfg.TenantID,
		AlgorithmResultID:  result.AlgorithmResultID,
		ExecutionRequestID: result.ExecutionRequestID,
		Symbol:             symbol,
		SessionDate:        observedAt.UTC().Truncate(24 * time.Hour),
		ObservedAt:         observedAt.UTC(),
		TechnicalScore:     score,
		TechnicalDirection: direction,
		RiskLevel:          riskLevel,
		Confidence:         result.Confidence,
		UsableInputCount:   usable,
		RequiredInputCount: required,
		Eligible:           usable >= riskRewardMinimumUsable,
		ResultPayloadJSON:  resultPayload,
		InputSnapshotJSON:  inputPayload,
	}, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func numberValue(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		result, err := value.Float64()
		return result, err == nil
	default:
		return 0, false
	}
}
