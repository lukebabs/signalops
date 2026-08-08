// Package signalassurance owns the deterministic Signal Assurance Framework
// domain rules. It deliberately contains no broker, HTTP, or UI logic.
package signalassurance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const EvaluationEngineVersion = "signal_assurance.v1.1"

type eligiblePayload struct {
	EligibleEventID         string          `json:"eligible_event_id"`
	TenantID                string          `json:"tenant_id"`
	SignalID                string          `json:"signal_id"`
	SignalLedgerID          string          `json:"signal_ledger_id"`
	AssetID                 string          `json:"asset_id"`
	Symbol                  string          `json:"symbol"`
	SignalType              string          `json:"signal_type"`
	Direction               string          `json:"direction"`
	Score                   *float64        `json:"score,omitempty"`
	Confidence              *float64        `json:"confidence,omitempty"`
	Status                  string          `json:"status"`
	Algorithm               string          `json:"algorithm"`
	AlgorithmVersion        string          `json:"algorithm_version"`
	ConfirmedAt             time.Time       `json:"confirmed_at"`
	EventAvailableAt        time.Time       `json:"event_available_at"`
	ConfirmationRuleVersion string          `json:"confirmation_rule_version"`
	ValidationContractRef   string          `json:"validation_contract_ref"`
	BaselineSnapshot        json.RawMessage `json:"baseline_snapshot"`
	BaselineProvenance      json.RawMessage `json:"baseline_provenance"`
	EvaluationMode          string          `json:"evaluation_mode"`
	EvaluationRunID         string          `json:"evaluation_run_id,omitempty"`
}

// DecodeEligibleEvent strictly validates the versioned internal event. The
// source signal.v1 remains untouched and cannot carry SAF-only fields.
func DecodeEligibleEvent(value []byte) (storage.SignalAssuranceEligibleEvent, error) {
	var raw eligiblePayload
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return storage.SignalAssuranceEligibleEvent{}, fmt.Errorf("decode signal assurance eligible event: %w", err)
	}
	if decoder.More() {
		return storage.SignalAssuranceEligibleEvent{}, errors.New("eligible event has trailing JSON values")
	}
	event := storage.SignalAssuranceEligibleEvent{
		EligibleEventID: raw.EligibleEventID, TenantID: raw.TenantID, SignalID: raw.SignalID,
		SignalLedgerID: raw.SignalLedgerID, AssetID: raw.AssetID, Symbol: strings.ToUpper(strings.TrimSpace(raw.Symbol)),
		SignalType: raw.SignalType, Direction: strings.ToLower(strings.TrimSpace(raw.Direction)), Score: raw.Score, Confidence: raw.Confidence,
		Status: strings.ToLower(strings.TrimSpace(raw.Status)), Algorithm: raw.Algorithm,
		AlgorithmVersion: raw.AlgorithmVersion, ConfirmedAt: raw.ConfirmedAt.UTC(),
		EventAvailableAt: raw.EventAvailableAt.UTC(), ConfirmationRuleVersion: raw.ConfirmationRuleVersion,
		ValidationContractRef: raw.ValidationContractRef, BaselineSnapshot: append([]byte(nil), raw.BaselineSnapshot...),
		BaselineProvenance: append([]byte(nil), raw.BaselineProvenance...), EvaluationMode: strings.ToUpper(strings.TrimSpace(raw.EvaluationMode)), EvaluationRunID: strings.TrimSpace(raw.EvaluationRunID),
	}
	if event.EvaluationMode == "" {
		event.EvaluationMode = storage.SignalAssuranceModeLive
	}
	if err := ValidateEligibleEvent(event); err != nil {
		return storage.SignalAssuranceEligibleEvent{}, err
	}
	return event, nil
}

func ValidateEligibleEvent(event storage.SignalAssuranceEligibleEvent) error {
	for field, value := range map[string]string{
		"eligible_event_id": event.EligibleEventID, "tenant_id": event.TenantID, "signal_id": event.SignalID,
		"signal_ledger_id": event.SignalLedgerID, "asset_id": event.AssetID, "symbol": event.Symbol,
		"signal_type": event.SignalType, "direction": event.Direction, "status": event.Status,
		"algorithm": event.Algorithm, "algorithm_version": event.AlgorithmVersion,
		"confirmation_rule_version": event.ConfirmationRuleVersion, "validation_contract_ref": event.ValidationContractRef,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("eligible event %s is required", field)
		}
	}
	if event.SignalID != event.SignalLedgerID {
		return errors.New("eligible event signal_id must equal signal_ledger_id")
	}
	if event.Status != "confirmed" || (event.Direction != "bullish" && event.Direction != "bearish") {
		return errors.New("eligible event must be a confirmed bullish or bearish signal")
	}
	if event.Confidence != nil && (*event.Confidence < 0 || *event.Confidence > 1) {
		return errors.New("eligible event confidence must be between zero and one")
	}
	if event.ConfirmedAt.IsZero() || event.EventAvailableAt.IsZero() || event.EventAvailableAt.Before(event.ConfirmedAt) {
		return errors.New("eligible event confirmation and availability times are invalid")
	}
	if !validMode(event.EvaluationMode) || (event.EvaluationMode == storage.SignalAssuranceModeLive && event.EvaluationRunID != "") || (event.EvaluationMode != storage.SignalAssuranceModeLive && event.EvaluationRunID == "") {
		return errors.New("eligible event evaluation mode/run namespace is invalid")
	}
	if !jsonObject(event.BaselineSnapshot) || !jsonObject(event.BaselineProvenance) {
		return errors.New("eligible event baseline snapshot and provenance must be JSON objects")
	}
	var baseline map[string]any
	_ = json.Unmarshal(event.BaselineSnapshot, &baseline)
	if price, ok := baseline["asset_price"].(float64); !ok || price <= 0 {
		return errors.New("eligible event baseline_snapshot.asset_price must be positive")
	}
	return nil
}

func AssertionRegistration(event storage.SignalAssuranceEligibleEvent, contract storage.SignalValidationContractRecord) (storage.SignalAssuranceRegistration, error) {
	if err := ValidateEligibleEvent(event); err != nil {
		return storage.SignalAssuranceRegistration{}, err
	}
	if err := ValidateContract(contract); err != nil {
		return storage.SignalAssuranceRegistration{}, err
	}
	if contract.SignalType != event.SignalType || contract.Direction != event.Direction || !contract.Active {
		return storage.SignalAssuranceRegistration{}, errors.New("eligible event does not resolve to an active validation contract")
	}
	if strings.TrimSpace(event.ValidationContractRef) != contract.ContractID && strings.TrimSpace(event.ValidationContractRef) != contract.SignalType+":"+contract.ContractVersion {
		return storage.SignalAssuranceRegistration{}, errors.New("eligible event validation contract reference does not match resolved contract")
	}
	contractJSON, _ := json.Marshal(contract)
	payload, _ := json.Marshal(event)
	key := registrationKey(event)
	assertionID := deterministicID("saf_assertion", event.TenantID, key)
	assertion := storage.SignalAssertionRecord{
		AssertionID: assertionID, TenantID: event.TenantID, AssetID: event.AssetID, Symbol: event.Symbol,
		SignalID: event.SignalID, SourceLedgerSignalID: event.SignalLedgerID, SignalType: event.SignalType,
		SignalDirection: event.Direction, SignalScore: event.Score, Confidence: event.Confidence, Algorithm: event.Algorithm,
		AlgorithmVersion: event.AlgorithmVersion, ConfirmedAt: event.ConfirmedAt, State: storage.SignalAssertionActive,
		EvaluationMode: event.EvaluationMode, EvaluationRunID: event.EvaluationRunID, RegistrationIdempotencyKey: key,
		ValidationContractID: contract.ContractID, ValidationContractVersion: contract.ContractVersion,
		ValidationContractJSON: contractJSON, EvaluationEngineVersion: EvaluationEngineVersion,
		BaselineSnapshotJSON: append([]byte(nil), event.BaselineSnapshot...), BaselineProvenanceJSON: append([]byte(nil), event.BaselineProvenance...), TransitionSequence: 1,
	}
	return storage.SignalAssuranceRegistration{Event: event, Contract: contract, Assertion: assertion, PayloadJSON: payload}, nil
}

func ValidateContract(contract storage.SignalValidationContractRecord) error {
	for field, value := range map[string]string{"contract_id": contract.ContractID, "signal_type": contract.SignalType, "contract_version": contract.ContractVersion, "direction": contract.Direction, "primary_metric": contract.PrimaryMetric, "comparison_operator": contract.ComparisonOperator, "materialization_policy": contract.MaterializationPolicy} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("validation contract %s is required", field)
		}
	}
	if (contract.Direction != "bullish" && contract.Direction != "bearish") || (contract.PrimaryMetric != "absolute_return" && contract.PrimaryMetric != "benchmark_relative_return") || !validOperator(contract.ComparisonOperator) || contract.MaxHorizonTradingDays <= 0 || !jsonArray(contract.EvaluationWindowsJSON) || !jsonObject(contract.ConfigJSON) {
		return errors.New("validation contract contains unsupported values")
	}
	return nil
}

func registrationKey(event storage.SignalAssuranceEligibleEvent) string {
	return strings.Join([]string{event.TenantID, event.AssetID, event.SignalID, event.AlgorithmVersion, event.ConfirmedAt.UTC().Format(time.RFC3339Nano)}, "|")
}

func deterministicID(prefix string, parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func validMode(mode string) bool {
	return mode == storage.SignalAssuranceModeLive || mode == storage.SignalAssuranceModeBacktest || mode == storage.SignalAssuranceModeResearch
}
func validOperator(op string) bool { return op == ">=" || op == ">" || op == "<=" || op == "<" }
func jsonObject(value []byte) bool {
	var v map[string]any
	return len(value) > 0 && json.Unmarshal(value, &v) == nil
}
func jsonArray(value []byte) bool {
	var v []any
	return len(value) > 0 && json.Unmarshal(value, &v) == nil
}
