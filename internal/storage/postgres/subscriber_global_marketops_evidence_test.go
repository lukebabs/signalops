package postgres

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestSubscriberGlobalMarketOpsEvidenceValidation(t *testing.T) {
	now := time.Now().UTC()
	run := storage.SubscriberGlobalMarketOpsEvidenceRun{
		EvidenceKind: "market_state", AlgorithmID: "market-state", AlgorithmVersion: "v1", SourceScope: "legacy_parity_review",
		InputManifestFingerprint: "manifest-1", ValidationContractRef: "contract-1", ImmutableBaselineRef: "baseline-1",
	}
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRun(&run); err != nil {
		t.Fatalf("valid global evidence run rejected: %v", err)
	}
	if run.EvidenceRunID == "" || run.RecordedBy != subscriberGlobalMarketOpsEvidenceWorker || len(run.ProvenanceJSON) == 0 {
		t.Fatalf("run normalization missing defaults: %+v", run)
	}

	record := storage.SubscriberGlobalMarketOpsEvidenceRecord{
		EvidenceRunID: run.EvidenceRunID, GlobalAssetID: "subglobal-aapl", SessionDate: now, EvidenceKind: "market_state",
		AlgorithmID: "market-state", AlgorithmVersion: "v1", QualityState: "usable", SourceSystem: "marketops",
		EvidenceFingerprint: "evidence-1", ValidationContractRef: "contract-1", ImmutableBaselineRef: "baseline-1", ObservedAt: now,
	}
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRecord(&record); err != nil {
		t.Fatalf("valid global evidence record rejected: %v", err)
	}
	if record.GlobalEvidenceID == "" || string(record.PayloadJSON) != "{}" || string(record.ProvenanceJSON) != "{}" {
		t.Fatalf("record normalization missing defaults: %+v", record)
	}

	record.QualityState = "invented"
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRecord(&record); err == nil {
		t.Fatal("invalid quality state accepted")
	}

	badRun := run
	badRun.SourceScope = "live"
	if err := normalizeSubscriberGlobalMarketOpsEvidenceRun(&badRun); err == nil {
		t.Fatal("non-shadow run accepted")
	}
}
