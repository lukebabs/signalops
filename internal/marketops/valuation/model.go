// Package valuation implements the deterministic VC and DOSM v3 models.
package valuation

import (
	"math"
)

const ModelVersion = "vc-dosm-3.0"

type Input struct {
	Ticker                 string
	Price                  float64
	MarketCap              float64
	EnterpriseValue        float64
	RevenueTTM             float64
	NetIncomeGAAPTTM       float64
	EBITDAProviderTTM      float64
	OperatingIncomeTTM     float64
	OperatingCashFlowTTM   float64
	CapitalExpendituresTTM float64
	TotalDebt              float64
	CashAndEquivalents     float64
	ShareholdersEquity     float64
	InvestedCapital        float64
	Revenue3YAgo           float64
	RevenueCAGR3Y          *float64
	RSI14                  *float64
	SMA50                  *float64
	SMA200                 *float64
	PeerCount              int
	PeerPSMedian           *float64
	PeerPEMedian           *float64
	PeerEVEBITDAMedian     *float64
	FinancialAgeDays       int
	EffectiveTaxRate       *float64
}

type Result struct {
	VCScore            float64            `json:"vc_score"`
	DOSMScore          float64            `json:"dosm_score"`
	VCFairValue        float64            `json:"vc_fair_value"`
	DOSMFairValue      float64            `json:"dosm_fair_value"`
	VCClassification   string             `json:"vc_classification"`
	DOSMClassification string             `json:"dosm_classification"`
	Confidence         int                `json:"confidence"`
	ConfidenceLabel    string             `json:"confidence_label"`
	Status             string             `json:"evaluation_status"`
	Eligible           bool               `json:"eligible"`
	DataProfile        string             `json:"data_profile"`
	GrowthStatus       string             `json:"growth_status"`
	Raw                map[string]float64 `json:"raw_metrics"`
	Components         map[string]float64 `json:"component_scores"`
	Reasons            []string           `json:"confidence_reasons"`
}

var psPoints = [][2]float64{{.5, 10}, {1, 9}, {2, 7.5}, {3, 6}, {5, 4}, {8, 2}, {12, 1}, {15, .5}, {20, 0}}
var pePoints = [][2]float64{{5, 10}, {8, 9}, {12, 8}, {16, 7}, {20, 6}, {25, 5}, {30, 4}, {40, 2.5}, {60, 1}, {80, 0}}
var evPoints = [][2]float64{{3, 10}, {5, 9}, {7, 8}, {9, 7}, {12, 5.5}, {15, 4}, {20, 2.5}, {30, 1}, {40, 0}}
var growthPoints = [][2]float64{{-.15, 0}, {-.1, 1.5}, {-.05, 3}, {0, 5}, {.05, 6}, {.1, 7}, {.15, 8}, {.25, 9}, {.4, 10}}
var operatingMarginPoints = [][2]float64{{-.15, 0}, {-.05, 1.5}, {0, 3}, {.05, 5}, {.1, 6.5}, {.15, 7.5}, {.2, 8.5}, {.3, 9.5}, {.4, 10}}
var positiveMarginPoints = [][2]float64{{0, 5}, {.03, 6}, {.05, 6.5}, {.08, 7.5}, {.12, 8.5}, {.2, 9.5}, {.3, 10}}
var negativeMarginPoints = [][2]float64{{-.3, 0}, {-.2, .5}, {-.1, 1.5}, {-.05, 2.5}, {0, 3.5}}
var debtEquityPoints = [][2]float64{{0, 10}, {.25, 9}, {.5, 8}, {1, 6.5}, {1.5, 5}, {2, 3.5}, {3, 2}, {5, 0}}
var netDebtEBITDAPoints = [][2]float64{{0, 10}, {1, 8.5}, {2, 7}, {3, 5.5}, {4, 4}, {5, 2.5}, {7, 0}}
var roicPoints = [][2]float64{{-.1, 0}, {0, 3}, {.05, 5}, {.08, 6.5}, {.1, 7}, {.15, 8}, {.2, 9}, {.3, 10}}

func Evaluate(in Input) Result {
	r := Result{Raw: map[string]float64{}, Components: map[string]float64{}, Status: "complete", Eligible: true, Confidence: 100, DataProfile: "full_growth"}
	if !positive(in.Price) || !positive(in.MarketCap) || !positive(in.RevenueTTM) {
		return incomplete(r, "canonical price, market capitalization, and revenue TTM are required")
	}
	ps := in.MarketCap / in.RevenueTTM
	r.Raw["ps_ttm"] = ps
	psScore := descending(ps, psPoints)
	r.Components["ps_score"] = psScore
	peScore := 0.0
	pe := 0.0
	if positive(in.NetIncomeGAAPTTM) {
		pe = in.MarketCap / in.NetIncomeGAAPTTM
		peScore = descending(pe, pePoints)
		r.Raw["pe_gaap_ttm"] = pe
	}
	r.Components["pe_score"] = peScore
	evScore := 0.0
	ev := 0.0
	if positive(in.EnterpriseValue) && positive(in.EBITDAProviderTTM) {
		ev = in.EnterpriseValue / in.EBITDAProviderTTM
		evScore = descending(ev, evPoints)
		r.Raw["ev_ebitda_ttm"] = ev
	}
	r.Components["ev_ebitda_score"] = evScore
	vcBase := .4*psScore + .3*peScore + .3*evScore
	r.Components["vc_base"] = vcBase
	peer := peerAdjustment(in, ps, pe, ev)
	r.Components["peer_adjustment"] = peer
	growth, growthOK := revenueGrowth(in)
	penalty := 0.0
	if growthOK {
		r.Raw["revenue_cagr_3y"] = growth
		penalty = growthPenalty(ps, growth, in.NetIncomeGAAPTTM)
	} else {
		r.DataProfile = "ttm_only"
		r.GrowthStatus = "unavailable_requires_16_quarters"
		r.Status = "complete_ttm_only"
		deduct(&r, 15, "three-year revenue CAGR unavailable; growth score and valuation penalty withheld")
	}
	r.Components["growth_valuation_penalty"] = penalty
	r.VCScore = round4(clamp(vcBase+peer-penalty, 0, 10))
	operatingMargin := in.OperatingIncomeTTM / in.RevenueTTM
	netMargin := in.NetIncomeGAAPTTM / in.RevenueTTM
	fcf := in.OperatingCashFlowTTM - in.CapitalExpendituresTTM
	fcfMargin := fcf / in.RevenueTTM
	r.Raw["operating_margin"], r.Raw["net_margin"], r.Raw["fcf_margin"] = operatingMargin, netMargin, fcfMargin
	revenueScore := 0.0
	if growthOK {
		revenueScore = ascending(growth, growthPoints)
	}
	operatingScore := ascending(operatingMargin, operatingMarginPoints)
	profitScore := marginScore(netMargin, in.NetIncomeGAAPTTM)
	fcfScore := marginScore(fcfMargin, fcf)
	debtScore, debtOK := debtProfile(in)
	roic, roicOK := roicScore(in)
	if !debtOK || !roicOK {
		return incomplete(r, "debt profile and ROIC inputs are required")
	}
	r.Components["revenue_growth_score"], r.Components["operating_margin_score"], r.Components["profitability_score"], r.Components["fcf_score"], r.Components["debt_profile_score"], r.Components["capital_efficiency_score"] = revenueScore, operatingScore, profitScore, fcfScore, debtScore, roic
	fundamental := (revenueScore + operatingScore + profitScore + fcfScore + debtScore + roic) / 6
	if !growthOK {
		// A missing CAGR is not a zero-growth assertion. Reweight the five TTM-derived dimensions.
		fundamental = (operatingScore + profitScore + fcfScore + debtScore + roic) / 5
	}
	r.Components["fundamental_score"] = fundamental
	rsiAdj, trendAdj, technicalMissing := technical(in)
	r.Components["rsi_adjustment"], r.Components["trend_adjustment"] = rsiAdj, trendAdj
	technical := clamp(rsiAdj+trendAdj, -1, 1)
	r.Components["technical_adjustment"] = technical
	r.DOSMScore = round4(clamp(.5*r.VCScore+.5*fundamental+technical, 0, 10))
	r.VCFairValue = round2(in.Price * math.Exp(.1*(r.VCScore-5)))
	r.DOSMFairValue = round2(in.Price * math.Exp(.1*(r.DOSMScore-5)))
	if in.PeerCount < 3 {
		deduct(&r, 10, "peer adjustment unavailable")
	}
	if technicalMissing {
		deduct(&r, 5, "technical input unavailable")
	}
	if in.FinancialAgeDays > 365 {
		deduct(&r, 30, "financial snapshot older than 365 days")
	} else if in.FinancialAgeDays > 180 {
		deduct(&r, 10, "financial snapshot older than 180 days")
	}
	r.VCClassification = classify(r.VCScore)
	r.DOSMClassification = classify(r.DOSMScore)
	r.ConfidenceLabel = confidenceLabel(r.Confidence)
	if r.Confidence < 50 {
		r.Status = "insufficient_data"
		r.Eligible = false
	}
	return r
}

func incomplete(r Result, reason string) Result {
	r.Status = "insufficient_data"
	r.Eligible = false
	r.Confidence = 0
	r.ConfidenceLabel = "insufficient"
	r.Reasons = append(r.Reasons, reason)
	return r
}
func positive(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0 }
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func ascending(v float64, p [][2]float64) float64 {
	if v <= p[0][0] {
		return p[0][1]
	}
	if v >= p[len(p)-1][0] {
		return p[len(p)-1][1]
	}
	for i := 0; i < len(p)-1; i++ {
		if v >= p[i][0] && v <= p[i+1][0] {
			return p[i][1] + (v-p[i][0])/(p[i+1][0]-p[i][0])*(p[i+1][1]-p[i][1])
		}
	}
	return 0
}
func descending(v float64, p [][2]float64) float64 { return ascending(v, p) }
func revenueGrowth(in Input) (float64, bool) {
	if in.RevenueCAGR3Y != nil {
		return *in.RevenueCAGR3Y, true
	}
	if positive(in.Revenue3YAgo) {
		return math.Pow(in.RevenueTTM/in.Revenue3YAgo, 1.0/3) - 1, true
	}
	return 0, false
}
func marginScore(m, absolute float64) float64 {
	if absolute < 0 {
		return ascending(m, negativeMarginPoints)
	}
	return ascending(m, positiveMarginPoints)
}
func debtProfile(in Input) (float64, bool) {
	scores := []float64{}
	if in.ShareholdersEquity > 0 {
		scores = append(scores, descending(in.TotalDebt/in.ShareholdersEquity, debtEquityPoints))
	}
	netDebt := in.TotalDebt - in.CashAndEquivalents
	if positive(in.EBITDAProviderTTM) {
		scores = append(scores, descending(netDebt/in.EBITDAProviderTTM, netDebtEBITDAPoints))
	} else if netDebt > 0 {
		scores = append(scores, 0)
	}
	if len(scores) == 0 {
		return 0, false
	}
	if len(scores) == 1 {
		return scores[0], true
	}
	return (scores[0] + scores[1]) / 2, true
}
func effectiveTaxRate(in Input) float64 {
	if in.EffectiveTaxRate != nil {
		return *in.EffectiveTaxRate
	}
	return .25
}
func roicScore(in Input) (float64, bool) {
	if !positive(in.InvestedCapital) {
		return 0, false
	}
	return ascending(in.OperatingIncomeTTM*(1-effectiveTaxRate(in))/in.InvestedCapital, roicPoints), true
}
func technical(in Input) (float64, float64, bool) {
	missing := false
	rsi := 0.0
	if in.RSI14 == nil {
		missing = true
	} else if *in.RSI14 < 30 {
		rsi = .5
	} else if *in.RSI14 > 70 {
		rsi = -.5
	}
	trend := 0.0
	if in.SMA50 == nil || in.SMA200 == nil {
		missing = true
	} else if in.Price > *in.SMA50 && in.Price > *in.SMA200 {
		trend = .5
	} else if in.Price < *in.SMA50 && in.Price < *in.SMA200 {
		trend = -.5
	}
	return rsi, trend, missing
}
func peerAdjustment(in Input, ps, pe, ev float64) float64 {
	if in.PeerCount < 3 {
		return 0
	}
	values := []float64{}
	if in.PeerPSMedian != nil && positive(*in.PeerPSMedian) {
		values = append(values, relativeAdjustment(ps / *in.PeerPSMedian))
	}
	if pe > 0 && in.PeerPEMedian != nil && positive(*in.PeerPEMedian) {
		values = append(values, relativeAdjustment(pe / *in.PeerPEMedian))
	}
	if ev > 0 && in.PeerEVEBITDAMedian != nil && positive(*in.PeerEVEBITDAMedian) {
		values = append(values, relativeAdjustment(ev / *in.PeerEVEBITDAMedian))
	}
	if len(values) < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return clamp(sum/float64(len(values)), -.5, .5)
}
func relativeAdjustment(v float64) float64 {
	p := [][2]float64{{.6, .2}, {.8, .1}, {.95, 0}, {1.05, 0}, {1.2, -.1}, {1.5, -.2}}
	if v <= .6 {
		return .2
	}
	if v >= 1.5 {
		return -.2
	}
	return ascending(v, p)
}
func growthPenalty(ps, growth, netIncome float64) float64 {
	if ps <= 15 || growth >= .3 {
		return 0
	}
	p := .5
	if ps > 30 {
		p = 1
	} else if ps > 20 {
		p = .75
	}
	if netIncome <= 0 {
		p += .5
	}
	return clamp(p, 0, 1.5)
}
func classify(v float64) string {
	if v < 2 {
		return "avoid"
	}
	if v < 4 {
		return "weak"
	}
	if v < 6 {
		return "neutral"
	}
	if v < 8 {
		return "opportunity"
	}
	return "exceptional"
}
func deduct(r *Result, n int, reason string) {
	r.Confidence -= n
	r.Reasons = append(r.Reasons, reason)
}
func confidenceLabel(v int) string {
	if v >= 90 {
		return "high"
	}
	if v >= 70 {
		return "moderate"
	}
	if v >= 50 {
		return "low"
	}
	return "insufficient"
}
