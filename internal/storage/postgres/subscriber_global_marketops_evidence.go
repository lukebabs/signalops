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

const subscriberGlobalMarketOpsEvidenceWorker = "subscriber-global-eod-reconciler"

var subscriberGlobalMarketOpsEvidenceKinds = map[string]struct{}{
	"eod_bar": {}, "feature_vector": {}, "market_state": {}, "eroc": {}, "valuation": {}, "eeom": {},
	"material_event": {}, "signal_assertion": {}, "outcome": {}, "sri_snapshot": {}, "options_snapshot": {},
}

var subscriberGlobalMarketOpsEvidenceQualityStates = map[string]struct{}{
	"usable": {}, "partial": {}, "invalid": {},
}

var subscriberGlobalMarketOpsEvidenceSourceSystems = map[string]struct{}{
	"massive": {}, "fmp": {}, "state_street": {}, "marketops": {}, "legacy_parity_review": {},
}

func (r *Repository) RecordSubscriberGlobalMarketOpsEvidenceRun(ctx context.Context, run storage.SubscriberGlobalMarketOpsEvidenceRun) (storage.SubscriberGlobalMarketOpsEvidenceRun, error) {
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRun(&run); err != nil {
		return run, err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_runs
  (evidence_run_id,evidence_kind,algorithm_id,algorithm_version,execution_mode,source_scope,session_start_date,session_end_date,input_manifest_fingerprint,validation_contract_ref,immutable_baseline_ref,provenance,recorded_by,correlation_id,recorded_at)
VALUES ($1,$2,$3,$4,'shadow_read_only',$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14)`,
		run.EvidenceRunID, run.EvidenceKind, run.AlgorithmID, run.AlgorithmVersion, run.SourceScope,
		nullTime(run.SessionStartDate), nullTime(run.SessionEndDate), run.InputManifestFingerprint,
		run.ValidationContractRef, run.ImmutableBaselineRef, string(run.ProvenanceJSON), run.RecordedBy,
		run.CorrelationID, run.RecordedAt.UTC()); err != nil {
		return run, fmt.Errorf("record subscriber global MarketOps evidence run: %w", err)
	}
	return run, nil
}

func (r *Repository) AppendSubscriberGlobalMarketOpsEvidence(ctx context.Context, record storage.SubscriberGlobalMarketOpsEvidenceRecord) (storage.SubscriberGlobalMarketOpsEvidenceRecord, error) {
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRecord(&record); err != nil {
		return record, err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO subscriber_global_marketops_evidence_records
  (global_evidence_id,evidence_run_id,global_asset_id,session_date,evidence_kind,algorithm_id,algorithm_version,quality_state,source_system,source_event_id,source_run_id,evidence_fingerprint,validation_contract_ref,immutable_baseline_ref,payload,provenance,observed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17)`,
		record.GlobalEvidenceID, record.EvidenceRunID, record.GlobalAssetID, record.SessionDate.UTC(), record.EvidenceKind,
		record.AlgorithmID, record.AlgorithmVersion, record.QualityState, record.SourceSystem, record.SourceEventID,
		record.SourceRunID, record.EvidenceFingerprint, record.ValidationContractRef, record.ImmutableBaselineRef,
		string(record.PayloadJSON), string(record.ProvenanceJSON), record.ObservedAt.UTC()); err != nil {
		return record, fmt.Errorf("append subscriber global MarketOps evidence: %w", err)
	}
	return record, nil
}

func normalizeSubscriberGlobalMarketOpsEvidenceRun(run *storage.SubscriberGlobalMarketOpsEvidenceRun) error {
	run.EvidenceRunID = strings.TrimSpace(run.EvidenceRunID)
	run.EvidenceKind = strings.TrimSpace(run.EvidenceKind)
	run.AlgorithmID = strings.TrimSpace(run.AlgorithmID)
	run.AlgorithmVersion = strings.TrimSpace(run.AlgorithmVersion)
	run.SourceScope = strings.TrimSpace(run.SourceScope)
	run.InputManifestFingerprint = strings.TrimSpace(run.InputManifestFingerprint)
	run.ValidationContractRef = strings.TrimSpace(run.ValidationContractRef)
	run.ImmutableBaselineRef = strings.TrimSpace(run.ImmutableBaselineRef)
	run.RecordedBy = strings.TrimSpace(run.RecordedBy)
	run.CorrelationID = strings.TrimSpace(run.CorrelationID)
	if run.EvidenceRunID == "" {
		run.EvidenceRunID = newSubscriberID("subglobalevrun")
	}
	if run.RecordedBy == "" {
		run.RecordedBy = subscriberGlobalMarketOpsEvidenceWorker
	}
	if run.RecordedAt.IsZero() {
		run.RecordedAt = time.Now().UTC()
	}
	if _, ok := subscriberGlobalMarketOpsEvidenceKinds[run.EvidenceKind]; !ok || run.AlgorithmID == "" || run.AlgorithmVersion == "" ||
		(run.SourceScope != "global_provider_capture" && run.SourceScope != "legacy_parity_review") || run.InputManifestFingerprint == "" ||
		run.ValidationContractRef == "" || run.ImmutableBaselineRef == "" || run.RecordedBy != subscriberGlobalMarketOpsEvidenceWorker ||
		(!run.SessionStartDate.IsZero() && !run.SessionEndDate.IsZero() && run.SessionStartDate.After(run.SessionEndDate)) {
		return errors.New("invalid subscriber global MarketOps evidence run")
	}
	return normalizeSubscriberGlobalEvidenceJSON(&run.ProvenanceJSON)
}

func normalizeSubscriberGlobalMarketOpsEvidenceRecord(record *storage.SubscriberGlobalMarketOpsEvidenceRecord) error {
	record.GlobalEvidenceID = strings.TrimSpace(record.GlobalEvidenceID)
	record.EvidenceRunID = strings.TrimSpace(record.EvidenceRunID)
	record.GlobalAssetID = strings.TrimSpace(record.GlobalAssetID)
	record.EvidenceKind = strings.TrimSpace(record.EvidenceKind)
	record.AlgorithmID = strings.TrimSpace(record.AlgorithmID)
	record.AlgorithmVersion = strings.TrimSpace(record.AlgorithmVersion)
	record.QualityState = strings.TrimSpace(record.QualityState)
	record.SourceSystem = strings.TrimSpace(record.SourceSystem)
	record.SourceEventID = strings.TrimSpace(record.SourceEventID)
	record.SourceRunID = strings.TrimSpace(record.SourceRunID)
	record.EvidenceFingerprint = strings.TrimSpace(record.EvidenceFingerprint)
	record.ValidationContractRef = strings.TrimSpace(record.ValidationContractRef)
	record.ImmutableBaselineRef = strings.TrimSpace(record.ImmutableBaselineRef)
	if record.GlobalEvidenceID == "" {
		record.GlobalEvidenceID = newSubscriberID("subglobalev")
	}
	if record.EvidenceRunID == "" || record.GlobalAssetID == "" || record.SessionDate.IsZero() || record.ObservedAt.IsZero() ||
		record.AlgorithmID == "" || record.AlgorithmVersion == "" || record.EvidenceFingerprint == "" ||
		record.ValidationContractRef == "" || record.ImmutableBaselineRef == "" {
		return errors.New("invalid subscriber global MarketOps evidence record")
	}
	if _, ok := subscriberGlobalMarketOpsEvidenceKinds[record.EvidenceKind]; !ok {
		return errors.New("invalid subscriber global MarketOps evidence kind")
	}
	if _, ok := subscriberGlobalMarketOpsEvidenceQualityStates[record.QualityState]; !ok {
		return errors.New("invalid subscriber global MarketOps evidence quality state")
	}
	if _, ok := subscriberGlobalMarketOpsEvidenceSourceSystems[record.SourceSystem]; !ok {
		return errors.New("invalid subscriber global MarketOps evidence source system")
	}
	if err := normalizeSubscriberGlobalEvidenceJSON(&record.PayloadJSON); err != nil {
		return err
	}
	return normalizeSubscriberGlobalEvidenceJSON(&record.ProvenanceJSON)
}

func normalizeSubscriberGlobalEvidenceJSON(value *[]byte) error {
	if len(*value) == 0 {
		*value = []byte(`{}`)
	}
	if !json.Valid(*value) {
		return errors.New("subscriber global MarketOps evidence JSON is invalid")
	}
	return nil
}
