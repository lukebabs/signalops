package options

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const RequiredSurfaceCellCount = 5

type AnalyticsReadiness struct {
	Ready                bool     `json:"ready"`
	ContractCount        int      `json:"contract_count"`
	UsableIVCount        int      `json:"usable_iv_count"`
	UsableGreeksCount    int      `json:"usable_greeks_count"`
	OpenInterestCount    int      `json:"open_interest_count"`
	UnderlyingPriceCount int      `json:"underlying_price_count"`
	RequiredSurfaceCells int      `json:"required_surface_cells"`
	MissingCells         []string `json:"missing_cells"`
	QualityReasons       []string `json:"quality_reasons"`
}

type surfaceCell struct {
	name       string
	dte        int
	delta      float64
	optionType string
}

var requiredSurfaceCells = []surfaceCell{
	{name: "atm_30d", dte: 30, delta: .50},
	{name: "atm_60d", dte: 60, delta: .50},
	{name: "atm_90d", dte: 90, delta: .50},
	{name: "put_25d_30d", dte: 30, delta: .25, optionType: "put"},
	{name: "call_25d_30d", dte: 30, delta: .25, optionType: "call"},
}

// AssessAnalyticsReadiness applies the same point-in-time surface policy used by G141.
func AssessAnalyticsReadiness(session time.Time, records []storage.MarketOpsOptionsChainRecord) AnalyticsReadiness {
	report := AnalyticsReadiness{ContractCount: len(records)}
	for _, record := range records {
		if record.ImpliedVolatility != nil && *record.ImpliedVolatility > 0 {
			report.UsableIVCount++
		}
		if record.Delta != nil && record.Gamma != nil && record.Theta != nil && record.Vega != nil {
			report.UsableGreeksCount++
		}
		if record.OpenInterest != nil {
			report.OpenInterestCount++
		}
		if record.UnderlyingClose != nil && *record.UnderlyingClose > 0 {
			report.UnderlyingPriceCount++
		}
	}
	for _, cell := range requiredSurfaceCells {
		if surfaceCellPresent(session, records, cell) {
			report.RequiredSurfaceCells++
		} else {
			report.MissingCells = append(report.MissingCells, cell.name)
		}
	}
	if len(records) == 0 {
		report.QualityReasons = append(report.QualityReasons, "no_contracts")
	}
	if report.UsableIVCount == 0 {
		report.QualityReasons = append(report.QualityReasons, "missing_usable_implied_volatility")
	}
	if report.UsableGreeksCount == 0 {
		report.QualityReasons = append(report.QualityReasons, "missing_complete_greeks")
	}
	if report.OpenInterestCount == 0 {
		report.QualityReasons = append(report.QualityReasons, "missing_open_interest")
	}
	if report.UnderlyingPriceCount == 0 {
		report.QualityReasons = append(report.QualityReasons, "missing_underlying_price")
	}
	if len(report.MissingCells) > 0 {
		report.QualityReasons = append(report.QualityReasons, fmt.Sprintf("missing_required_surface_cells:%s", strings.Join(report.MissingCells, ",")))
	}
	sort.Strings(report.QualityReasons)
	report.Ready = report.RequiredSurfaceCells == RequiredSurfaceCellCount
	return report
}

func surfaceCellPresent(session time.Time, records []storage.MarketOpsOptionsChainRecord, cell surfaceCell) bool {
	for _, record := range records {
		if cell.optionType != "" && strings.ToLower(record.ContractType) != cell.optionType {
			continue
		}
		if record.ImpliedVolatility == nil || *record.ImpliedVolatility <= 0 || record.Delta == nil || record.UnderlyingClose == nil || *record.UnderlyingClose <= 0 {
			continue
		}
		dte := int(dayOnly(record.ExpirationDate).Sub(dayOnly(session)).Hours() / 24)
		tolerance := 15
		if cell.dte >= 60 {
			tolerance = 20
		}
		if cell.dte >= 90 {
			tolerance = 30
		}
		if dte >= 7 && dte <= 180 && absInt(dte-cell.dte) <= tolerance && math.Abs(math.Abs(*record.Delta)-cell.delta) <= .15 {
			return true
		}
	}
	return false
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
