package postgres

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestValidateMarketOpsSignalOutcome(t *testing.T) {
	record := validMarketOpsSignalOutcome()
	if err := validateMarketOpsSignalOutcome(record); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*storage.MarketOpsSignalOutcomeRecord){
		"source":     func(record *storage.MarketOpsSignalOutcomeRecord) { record.SourceType = "alert" },
		"direction":  func(record *storage.MarketOpsSignalOutcomeRecord) { record.Direction = "sideways" },
		"horizon":    func(record *storage.MarketOpsSignalOutcomeRecord) { record.HorizonSessions = 3 },
		"maturity":   func(record *storage.MarketOpsSignalOutcomeRecord) { record.MaturedSessionDate = nil },
		"hypothesis": func(record *storage.MarketOpsSignalOutcomeRecord) { record.HypothesisKey = "" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := validMarketOpsSignalOutcome()
			mutate(&invalid)
			if validateMarketOpsSignalOutcome(invalid) == nil {
				t.Fatalf("expected validation failure: %+v", invalid)
			}
		})
	}
}

func TestValidatePendingMarketOpsSignalOutcome(t *testing.T) {
	record := validMarketOpsSignalOutcome()
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
	if err := validateMarketOpsSignalOutcome(record); err != nil {
		t.Fatal(err)
	}
}

func validMarketOpsSignalOutcome() storage.MarketOpsSignalOutcomeRecord {
	origin := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	matured := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
	forward, favorable, adverse, drawdown := .05, .07, -.02, -.03
	hit := true
	days := 3
	return storage.MarketOpsSignalOutcomeRecord{
		OutcomeID: "moutcome-1", TenantID: "tenant-local", AppID: "marketops",
		SourceType: storage.MarketOpsOutcomeSourceHypothesisEvaluation, SourceID: "eval-1",
		HypothesisKey: "H001", HypothesisVersion: "v1", AssetID: "ticker:AAPL", Symbol: "AAPL",
		Direction: "upside", OriginSessionDate: origin, HorizonSessions: 5, MaturedSessionDate: &matured,
		OutcomeStatus: storage.MarketOpsOutcomeMatured, ForwardReturn: &forward,
		MaxFavorableExcursion: &favorable, MaxAdverseExcursion: &adverse, MaximumDrawdown: &drawdown,
		DirectionalHit: &hit, ThresholdHit: &hit, DaysToThreshold: &days, OriginEventID: "evt-0",
		OutcomeEventIDs: []string{"evt-1"}, OutcomePayloadJSON: []byte(`{"threshold":0.02}`),
		CalculationVersion: "marketops.forward_outcome.v1", CalculationRunID: "run-1", DeterministicKey: "key-1",
	}
}
