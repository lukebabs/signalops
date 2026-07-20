package options

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const (
	RequiredSurfaceCellCount      = 7
	SurfaceSelectionPolicyVersion = "marketops.options.surface_selection.v1"
)

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
	{name: "put_25d_60d", dte: 60, delta: .25, optionType: "put"},
	{name: "call_25d_60d", dte: 60, delta: .25, optionType: "call"},
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

// SelectRequiredSurfaceEvidence retains at most one deterministic source contract
// per currently implemented analytical surface cell.
func SelectRequiredSurfaceEvidence(session time.Time, records []storage.MarketOpsOptionsChainRecord) []storage.MarketOpsOptionsChainRecord {
	selected := make([]storage.MarketOpsOptionsChainRecord, 0, RequiredSurfaceCellCount)
	used := map[string]struct{}{}
	for _, cell := range requiredSurfaceCells {
		record, ok := bestSurfaceCell(session, records, cell, used)
		if !ok {
			continue
		}
		score := surfaceSelectionScore(session, record, cell)
		record.SelectionCell, record.SelectionPolicyVersion = cell.name, SurfaceSelectionPolicyVersion
		record.SelectionScore = &score
		selected = append(selected, record)
		used[record.OptionTicker] = struct{}{}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].OptionTicker < selected[j].OptionTicker })
	return selected
}

func surfaceCellPresent(session time.Time, records []storage.MarketOpsOptionsChainRecord, cell surfaceCell) bool {
	_, ok := bestSurfaceCell(session, records, cell, nil)
	return ok
}

func bestSurfaceCell(session time.Time, records []storage.MarketOpsOptionsChainRecord, cell surfaceCell, excluded map[string]struct{}) (storage.MarketOpsOptionsChainRecord, bool) {
	bestScore := math.MaxFloat64
	bestOpenInterest := int64(-1)
	bestVolume := int64(-1)
	var best storage.MarketOpsOptionsChainRecord
	found := false
	for _, record := range records {
		if _, skip := excluded[record.OptionTicker]; skip {
			continue
		}
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
		dteDistance := absInt(dte - cell.dte)
		deltaDistance := math.Abs(math.Abs(*record.Delta) - cell.delta)
		if dte < 7 || dte > 180 || dteDistance > tolerance || deltaDistance > .15 {
			continue
		}
		score := float64(dteDistance)/float64(tolerance) + deltaDistance/.15
		oi := int64(-1)
		if record.OpenInterest != nil {
			oi = *record.OpenInterest
		}
		volume := int64(-1)
		if record.Volume != nil {
			volume = *record.Volume
		}
		if !found || score < bestScore || (score == bestScore && (oi > bestOpenInterest || (oi == bestOpenInterest && (volume > bestVolume || (volume == bestVolume && record.OptionTicker < best.OptionTicker))))) {
			best, found = record, true
			bestScore, bestOpenInterest, bestVolume = score, oi, volume
		}
	}
	return best, found
}

func surfaceSelectionScore(session time.Time, record storage.MarketOpsOptionsChainRecord, cell surfaceCell) float64 {
	dte := int(dayOnly(record.ExpirationDate).Sub(dayOnly(session)).Hours() / 24)
	tolerance := 15
	if cell.dte >= 60 {
		tolerance = 20
	}
	if cell.dte >= 90 {
		tolerance = 30
	}
	deltaDistance := 0.0
	if record.Delta != nil {
		deltaDistance = math.Abs(math.Abs(*record.Delta) - cell.delta)
	}
	return float64(absInt(dte-cell.dte))/float64(tolerance) + deltaDistance/.15
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
