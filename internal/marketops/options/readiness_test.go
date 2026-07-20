package options

import (
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

func TestAssessAnalyticsReadinessRequiresFiveActualCells(t *testing.T) {
	session := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	records := []storage.MarketOpsOptionsChainRecord{
		readinessRecord(session, 30, .50, "call"),
		readinessRecord(session, 60, -.50, "put"),
		readinessRecord(session, 90, .50, "call"),
		readinessRecord(session, 30, -.25, "put"),
		readinessRecord(session, 30, .25, "call"),
	}
	report := AssessAnalyticsReadiness(session, records)
	if !report.Ready || report.RequiredSurfaceCells != 5 || report.UsableIVCount != 5 || report.UsableGreeksCount != 5 || report.OpenInterestCount != 5 || report.UnderlyingPriceCount != 5 {
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
