package signalassurance

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const ResearchMaterializationRunID = "saf-research-materializations-v1"

// ResearchContractFor creates the sole provisional contract permitted by the
// research-first rollout. It is never selected for LIVE assertions.
func ResearchContractFor(event storage.SignalAssuranceEligibleEvent) storage.SignalValidationContractRecord {
	scope := strings.Join([]string{event.Algorithm, event.AlgorithmVersion}, "|")
	return storage.SignalValidationContractRecord{ContractID: deterministicID("saf_contract", "research-default-v1", event.SignalType, event.Direction, scope), SignalType: event.SignalType, ContractVersion: "research-default-v1", Algorithm: event.Algorithm, AlgorithmVersion: event.AlgorithmVersion, Direction: event.Direction, PrimaryMetric: "absolute_return", ComparisonOperator: ">=", Threshold: .05, EvaluationWindowsJSON: []byte(`[5,10,20]`), MaxHorizonTradingDays: 20, MaterializationPolicy: "threshold", InvalidationPolicy: "normalized_return_floor", ConfigJSON: []byte(`{"invalidation_threshold":-0.10,"provisional":true,"research_only":true}`), Active: true, ContractScopeKey: scope}
}

// EligibleFromMaterialization maps a completed materialization and only
// point-in-time available canonical bars into the internal SAF event.
func EligibleFromMaterialization(materialization storage.AlgorithmSignalMaterializationRecord, proposal storage.AlgorithmSignalProposalRecord, result storage.AlgorithmResultRecord, signal storage.SignalLedgerRecord, events []storage.NormalizedEventLedgerRecord, benchmark string) (storage.SignalAssuranceEligibleEvent, error) {
	if materialization.MaterializationStatus != storage.AlgorithmSignalMaterializationStatusSucceeded || strings.TrimSpace(signal.SignalID) == "" {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("materialization is not a newly succeeded ledger signal")
	}
	var payload struct {
		AssetID   string `json:"asset_id"`
		Symbol    string `json:"symbol"`
		Direction string `json:"direction"`
	}
	if err := json.Unmarshal(proposal.ProposalPayloadJSON, &payload); err != nil {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("decode proposal payload: %w", err)
	}
	direction := normalizeDirection(payload.Direction)
	if direction == "" {
		var resultPayload struct {
			Direction          string `json:"direction"`
			TechnicalDirection string `json:"technical_direction"`
		}
		_ = json.Unmarshal(result.ResultPayloadJSON, &resultPayload)
		direction = normalizeDirection(firstNonEmpty(resultPayload.Direction, resultPayload.TechnicalDirection))
	}
	if payload.AssetID == "" || payload.Symbol == "" || direction == "" {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("materialization lacks directional asset identity")
	}
	confirmed := signal.SignalTime.UTC()
	if confirmed.IsZero() && materialization.CompletedAt != nil {
		confirmed = materialization.CompletedAt.UTC()
	}
	if confirmed.IsZero() {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("materialization confirmation time is missing")
	}
	asset, assetEvent := canonicalClose(events, payload.Symbol, confirmed)
	bench, benchEvent := canonicalClose(events, benchmark, confirmed)
	if asset <= 0 || bench <= 0 {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("canonical baseline price unavailable")
	}
	baseline, _ := json.Marshal(map[string]any{"asset_price": asset, "benchmark_price": bench, "benchmark_symbol": strings.ToUpper(benchmark), "price_basis": "raw_regular_session_close"})
	provenance, _ := json.Marshal(map[string]any{"asset_event_id": assetEvent.EventID, "asset_observation_time": assetEvent.ObservationTime.UTC().Format(time.RFC3339Nano), "asset_available_at": assetEvent.ProcessingTime.UTC().Format(time.RFC3339Nano), "benchmark_event_id": benchEvent.EventID, "benchmark_observation_time": benchEvent.ObservationTime.UTC().Format(time.RFC3339Nano), "benchmark_available_at": benchEvent.ProcessingTime.UTC().Format(time.RFC3339Nano), "market_calendar_version": "nyse:v1", "corporate_action_policy_version": "raw_regular_session_close:v1"})
	event := storage.SignalAssuranceEligibleEvent{EligibleEventID: deterministicID("safelig", materialization.MaterializationID), TenantID: signal.TenantID, SignalID: signal.SignalID, SignalLedgerID: signal.SignalID, AssetID: payload.AssetID, Symbol: strings.ToUpper(payload.Symbol), SignalType: signal.SignalType, Direction: direction, Score: &proposal.Score, Confidence: &proposal.Confidence, Status: "confirmed", Algorithm: materialization.AlgorithmID, AlgorithmVersion: materialization.AlgorithmVersion, ConfirmedAt: confirmed, EventAvailableAt: time.Now().UTC(), ConfirmationRuleVersion: materialization.MaterializationPolicyVersion, BaselineSnapshot: baseline, BaselineProvenance: provenance, EvaluationMode: storage.SignalAssuranceModeResearch, EvaluationRunID: ResearchMaterializationRunID}
	contract := ResearchContractFor(event)
	event.ValidationContractRef = contract.ContractID
	return event, nil
}

func canonicalClose(events []storage.NormalizedEventLedgerRecord, symbol string, asOf time.Time) (float64, storage.NormalizedEventLedgerRecord) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	var best storage.NormalizedEventLedgerRecord
	var price float64
	for _, event := range events {
		if event.ObservationTime.After(asOf) || event.ProcessingTime.After(asOf) {
			continue
		}
		var payload struct {
			Symbol string  `json:"symbol"`
			Ticker string  `json:"ticker"`
			Close  float64 `json:"close"`
		}
		if json.Unmarshal(event.NormalizedPayload, &payload) != nil || payload.Close <= 0 {
			continue
		}
		ticker := strings.ToUpper(firstNonEmpty(payload.Symbol, payload.Ticker))
		if ticker != symbol {
			continue
		}
		if best.EventID == "" || event.ObservationTime.After(best.ObservationTime) || (event.ObservationTime.Equal(best.ObservationTime) && event.ProcessingTime.After(best.ProcessingTime)) {
			best, event = event, event
			price = payload.Close
		}
	}
	return price, best
}
func normalizeDirection(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bullish", "upside":
		return "bullish"
	case "bearish", "downside":
		return "bearish"
	}
	return ""
}
