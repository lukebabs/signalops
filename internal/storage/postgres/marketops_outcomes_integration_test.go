package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestMarketOpsSignalOutcomeRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("SIGNALOPS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SIGNALOPS_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	repo, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	matured := validMarketOpsSignalOutcome()
	if err := repo.UpsertMarketOpsSignalOutcome(ctx, matured); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListMarketOpsSignalOutcomes(ctx, storageOutcomeFilter(matured))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].OutcomeID != matured.OutcomeID || list[0].ForwardReturn == nil {
		t.Fatalf("unexpected outcome list: %+v", list)
	}
	got, err := repo.GetMarketOpsSignalOutcome(ctx, matured.TenantID, matured.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeterministicKey != matured.DeterministicKey || len(got.OutcomeEventIDs) != 1 {
		t.Fatalf("unexpected outcome detail: %+v", got)
	}

	olderPending := pendingOutcome(matured, "run-older")
	if err := repo.UpsertMarketOpsSignalOutcome(ctx, olderPending); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetMarketOpsSignalOutcome(ctx, matured.TenantID, matured.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OutcomeStatus != storage.MarketOpsOutcomeMatured || got.CalculationRunID != matured.CalculationRunID {
		t.Fatalf("matured outcome regressed: %+v", got)
	}

	secondMatured := matured
	secondMatured.OutcomeID = "moutcome-2"
	secondMatured.SourceID = "eval-2"
	secondMatured.DeterministicKey = "key-2"
	pending := pendingOutcome(secondMatured, "run-pending")
	if err := repo.UpsertMarketOpsSignalOutcome(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertMarketOpsSignalOutcome(ctx, secondMatured); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetMarketOpsSignalOutcome(ctx, secondMatured.TenantID, secondMatured.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OutcomeStatus != storage.MarketOpsOutcomeMatured || got.CalculationRunID != secondMatured.CalculationRunID {
		t.Fatalf("pending outcome did not advance: %+v", got)
	}
}

func pendingOutcome(record storage.MarketOpsSignalOutcomeRecord, runID string) storage.MarketOpsSignalOutcomeRecord {
	record.OutcomeStatus = storage.MarketOpsOutcomePending
	record.MaturedSessionDate = nil
	record.ForwardReturn = nil
	record.MaxFavorableExcursion = nil
	record.MaxAdverseExcursion = nil
	record.MaximumDrawdown = nil
	record.RealizedVolChange = nil
	record.DirectionalHit = nil
	record.ThresholdHit = nil
	record.DaysToThreshold = nil
	record.OriginEventID = ""
	record.OutcomeEventIDs = nil
	record.CalculationRunID = runID
	return record
}

func storageOutcomeFilter(record storage.MarketOpsSignalOutcomeRecord) storage.MarketOpsSignalOutcomeFilter {
	return storage.MarketOpsSignalOutcomeFilter{TenantID: record.TenantID, SourceType: record.SourceType, SourceID: record.SourceID, Limit: 10}
}
