package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

// RecordSubscriberGlobalAssetEligibilityDecision retains immutable admission
// evidence. It may promote an asset only after explicit US-common-stock
// evidence; it never starts coverage or a provider request.
func (r *Repository) RecordSubscriberGlobalAssetEligibilityDecision(ctx context.Context, record storage.SubscriberGlobalAssetEligibilityDecision) (storage.SubscriberGlobalAssetEligibilityDecision, error) {
	record.GlobalAssetID = strings.TrimSpace(record.GlobalAssetID)
	record.Decision = strings.TrimSpace(record.Decision)
	record.PolicyVersion = strings.TrimSpace(record.PolicyVersion)
	record.ReasonCode = strings.TrimSpace(record.ReasonCode)
	record.DecidedBy = strings.TrimSpace(record.DecidedBy)
	if record.DecisionID == "" {
		record.DecisionID = newSubscriberID("subelig")
	}
	if record.PolicyVersion == "" {
		record.PolicyVersion = storage.SubscriberGlobalEligibilityPolicyVersion
	}
	if record.DecidedAt.IsZero() {
		record.DecidedAt = time.Now().UTC()
	}
	if record.GlobalAssetID == "" || record.ReasonCode == "" || record.DecidedBy == "" {
		return record, errors.New("invalid global asset eligibility decision")
	}
	switch record.Decision {
	case "eligible":
		if !validUSCommonStockEvidence(record.EvidenceJSON) {
			return record, errors.New("eligible decision requires US exchange-listed common-stock provider evidence")
		}
	case "ineligible", "deferred":
		if !json.Valid(record.EvidenceJSON) {
			return record, errors.New("eligibility evidence must be valid JSON")
		}
	default:
		return record, errors.New("unknown global asset eligibility decision")
	}
	if len(record.ProvenanceJSON) == 0 {
		record.ProvenanceJSON = []byte("{}")
	}
	if !json.Valid(record.ProvenanceJSON) {
		return record, errors.New("eligibility provenance must be valid JSON")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return record, fmt.Errorf("begin global asset eligibility decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT eligibility_status FROM subscriber_global_assets WHERE global_asset_id=$1 FOR UPDATE`, record.GlobalAssetID).Scan(&status); err != nil {
		return record, mapScanError("lock global asset eligibility", err)
	}
	nextStatus := status
	if record.Decision == "eligible" {
		nextStatus = "eligible"
	}
	if record.Decision == "ineligible" {
		nextStatus = "ineligible"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscriber_global_asset_eligibility_decisions
  (decision_id,global_asset_id,decision,policy_version,reason_code,provider_reference_at,evidence,provenance,decided_by,decided_at)
VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10)`,
		record.DecisionID, record.GlobalAssetID, record.Decision, record.PolicyVersion, record.ReasonCode, record.ProviderReferenceAt, string(record.EvidenceJSON), string(record.ProvenanceJSON), record.DecidedBy, record.DecidedAt); err != nil {
		return record, fmt.Errorf("insert global asset eligibility decision: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE subscriber_global_assets SET eligibility_status=$2,updated_at=now() WHERE global_asset_id=$1`, record.GlobalAssetID, nextStatus); err != nil {
		return record, fmt.Errorf("update global asset eligibility status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return record, fmt.Errorf("commit global asset eligibility decision: %w", err)
	}
	return record, nil
}

func validUSCommonStockEvidence(raw []byte) bool {
	var evidence map[string]any
	if !json.Valid(raw) || json.Unmarshal(raw, &evidence) != nil {
		return false
	}
	country, _ := evidence["country_code"].(string)
	securityType, _ := evidence["security_type"].(string)
	exchangeListed, _ := evidence["exchange_listed"].(bool)
	providerEligible, _ := evidence["provider_eligible"].(bool)
	active, _ := evidence["is_active"].(bool)
	return strings.EqualFold(strings.TrimSpace(country), "US") &&
		strings.EqualFold(strings.TrimSpace(securityType), "common_stock") &&
		exchangeListed && providerEligible && active
}
