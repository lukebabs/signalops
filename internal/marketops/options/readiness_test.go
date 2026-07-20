package options

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestAssessAnalyticsReadinessRequiresSevenActualCells(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	records := []storage.MarketOpsOptionsChainRecord{
		readinessRecord(session, 30, .50, "call"),
		readinessRecord(session, 60, -.50, "put"),
		readinessRecord(session, 90, .50, "call"),
		readinessRecord(session, 30, -.25, "put"),
		readinessRecord(session, 30, .25, "call"),
		readinessRecord(session, 60, -.25, "put"),
		readinessRecord(session, 60, .25, "call"),
	}
	report := AssessAnalyticsReadiness(session, records)
	if !report.Ready || report.RequiredSurfaceCells != 7 || report.UsableIVCount != 7 || report.UsableGreeksCount != 7 || report.OpenInterestCount != 7 || report.UnderlyingPriceCount != 7 {
		t.Fatalf("report = %+v", report)
	}
}

func TestAssessAnalyticsReadinessDoesNotUseContractCountAsProxy(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	records := make([]storage.MarketOpsOptionsChainRecord, 20)
	for index := range records {
		records[index] = storage.MarketOpsOptionsChainRecord{TradeDate: session, ExpirationDate: session.AddDate(0, 0, 30), ContractType: "call"}
	}
	report := AssessAnalyticsReadiness(session, records)
	if report.Ready || report.RequiredSurfaceCells != 0 || len(report.QualityReasons) == 0 {
		t.Fatalf("report = %+v", report)
	}
}

func readinessRecord(session time.Time, dte int, delta float64, contractType string) storage.MarketOpsOptionsChainRecord {
	iv, underlying, gamma, theta, vega := .30, 100.0, .02, -.01, .10
	oi := int64(100)
	return storage.MarketOpsOptionsChainRecord{TradeDate: session, ExpirationDate: session.AddDate(0, 0, dte), ContractType: contractType, ImpliedVolatility: &iv, Delta: &delta, Gamma: &gamma, Theta: &theta, Vega: &vega, UnderlyingClose: &underlying, OpenInterest: &oi}
}

func TestSelectRequiredSurfaceEvidenceRetainsOnlyDeterministicCells(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	records := []storage.MarketOpsOptionsChainRecord{
		readinessRecord(session, 31, .49, "call"),
		readinessRecord(session, 60, -.50, "put"),
		readinessRecord(session, 90, .50, "call"),
		readinessRecord(session, 30, -.25, "put"),
		readinessRecord(session, 30, .25, "call"),
		readinessRecord(session, 60, -.25, "put"),
		readinessRecord(session, 60, .25, "call"),
		readinessRecord(session, 30, .90, "call"),
	}
	for index := range records {
		records[index].OptionTicker = string(rune('A' + index))
	}
	selected := SelectRequiredSurfaceEvidence(session, records)
	if len(selected) != RequiredSurfaceCellCount {
		t.Fatalf("selected = %+v", selected)
	}
	if !AssessAnalyticsReadiness(session, selected).Ready {
		t.Fatalf("selected surface is not ready: %+v", selected)
	}
	for _, record := range selected {
		if record.SelectionCell == "" || record.SelectionPolicyVersion != SurfaceSelectionPolicyVersion || record.SelectionScore == nil {
			t.Fatalf("selection lineage missing: %+v", record)
		}
		if record.OptionTicker == "H" {
			t.Fatalf("irrelevant deep-delta contract was retained: %+v", selected)
		}
	}
}
