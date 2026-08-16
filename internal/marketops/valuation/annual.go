package valuation

import "math"

// AnnualModelVersion is deliberately separate from the live TTM model. It is
// an annual-only, evidence-backed research profile and must not replace v3
// until its own calibration and rollout gates are approved.
const AnnualModelVersion = "vc-dosm-4.1-annual"

// AnnualInput contains values derived locally from raw, provider-captured
// annual statements. Provider ratios are useful references but are never used
// as scoring inputs.
type AnnualInput struct {
	Ticker              string
	Price               float64
	MarketCap           float64
	EnterpriseValue     float64
	Revenue             float64
	Revenue3YAgo        float64
	NetIncome           float64
	EBITDA              float64
	OperatingIncome     float64
	OperatingCashFlow   float64
	CapitalExpenditures float64
	TotalDebt           float64
	CashAndEquivalents  float64
	ShareholdersEquity  float64
	InvestedCapital     float64
	CurrentAssets       float64
	CurrentLiabilities  float64
	Inventory           float64
	InterestExpense     float64
	EffectiveTaxRate    *float64
	FinancialAgeDays    int
}

// AnnualResult reports each independently auditable annual dimension. Scores
// remain 0..10. Missing dimensions are reweighted rather than treated as an
// assertion of poor fundamentals, while confidence describes the reduction in
// evidence coverage.
type AnnualResult struct {
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
	Raw                map[string]float64 `json:"raw_metrics"`
	Components         map[string]float64 `json:"component_scores"`
	Reasons            []string           `json:"confidence_reasons"`
}

// annualGrowthPoints makes zero growth a low (2/10) result and reserves top scores for exceptional growth.
var annualGrowthPoints = [][2]float64{{-.15, 0}, {0, 2}, {.05, 3}, {.10, 4}, {.15, 5}, {.25, 6}, {.35, 8}, {.50, 10}}

var currentRatioPoints = [][2]float64{{0.5, 0}, {0.75, 2}, {1, 5}, {1.25, 7}, {1.5, 8.5}, {2, 10}}
var quickRatioPoints = [][2]float64{{0.4, 0}, {0.6, 2}, {0.8, 5}, {1, 7.5}, {1.25, 9}, {1.5, 10}}
var interestCoveragePoints = [][2]float64{{0, 0}, {1, 1}, {2, 3}, {3, 5}, {5, 7}, {8, 8.5}, {12, 10}}
var roePoints = [][2]float64{{-.1, 0}, {0, 3}, {.05, 5}, {.1, 6.5}, {.15, 7.5}, {.2, 8.5}, {.3, 10}}

// EvaluateAnnual calculates the six approved annual dimensions: valuation
// (40%), profitability, growth, liquidity, leverage, and capital efficiency
// (12% each). It is intentionally free of technical inputs and recommendations.
func EvaluateAnnual(in AnnualInput) AnnualResult {
	r := AnnualResult{
		Raw: map[string]float64{}, Components: map[string]float64{}, Status: "complete", Eligible: true,
		Confidence: 100, DataProfile: "annual_5y",
	}
	if !positive(in.MarketCap) || !positive(in.Revenue) {
		return incompleteAnnual(r, "market capitalization and annual revenue are required")
	}

	valuation, valuationOK := annualValuation(in, &r)
	if !valuationOK {
		return incompleteAnnual(r, "no usable annual valuation multiple is available")
	}
	profitability, profitabilityOK := annualProfitability(in, &r)
	growth, growthOK := annualGrowth(in, &r)
	liquidity, liquidityOK := annualLiquidity(in, &r)
	leverage, leverageOK := annualLeverage(in, &r)
	capital, capitalOK := annualCapitalEfficiency(in, &r)

	dimensions := []struct {
		name          string
		weight, score float64
		ok            bool
	}{
		{"valuation", .40, valuation, valuationOK},
		{"profitability", .12, profitability, profitabilityOK},
		{"growth", .12, growth, growthOK},
		{"liquidity", .12, liquidity, liquidityOK},
		{"leverage", .12, leverage, leverageOK},
		{"capital_efficiency", .12, capital, capitalOK},
	}
	availableWeight, weighted := 0.0, 0.0
	available := 0
	for _, dimension := range dimensions {
		r.Components[dimension.name+"_weight"] = dimension.weight
		if !dimension.ok {
			deductAnnual(&r, 10, dimension.name+" inputs unavailable; annual score reweighted")
			continue
		}
		available++
		availableWeight += dimension.weight
		weighted += dimension.weight * dimension.score
	}
	if availableWeight == 0 || available < 3 {
		return incompleteAnnual(r, "fewer than three annual scoring dimensions are available")
	}
	r.Components["available_dimension_weight"] = availableWeight
	r.Components["available_dimensions"] = float64(available)
	r.VCScore = round4(valuation)
	r.DOSMScore = round4(weighted / availableWeight)
	if positive(in.Price) {
		r.VCFairValue = round2(in.Price * math.Exp(.1*(r.VCScore-5)))
		r.DOSMFairValue = round2(in.Price * math.Exp(.1*(r.DOSMScore-5)))
	} else {
		deductAnnual(&r, 5, "canonical price unavailable; fair-value anchors withheld")
	}
	if in.FinancialAgeDays > 730 {
		deductAnnual(&r, 35, "annual filing older than 730 days")
	} else if in.FinancialAgeDays > 500 {
		deductAnnual(&r, 20, "annual filing older than 500 days")
	} else if in.FinancialAgeDays > 365 {
		deductAnnual(&r, 10, "annual filing older than 365 days")
	}
	r.Confidence = max(0, r.Confidence)
	r.ConfidenceLabel = confidenceLabel(r.Confidence)
	r.VCClassification = classify(r.VCScore)
	r.DOSMClassification = classify(r.DOSMScore)
	if r.Confidence < 50 {
		r.Status = "insufficient_data"
		r.Eligible = false
	} else if available < len(dimensions) {
		r.Status = "complete_partial_annual"
		r.DataProfile = "annual_partial"
	}
	return r
}

func annualValuation(in AnnualInput, r *AnnualResult) (float64, bool) {
	scores := []float64{}
	ps := in.MarketCap / in.Revenue
	r.Raw["ps_annual"] = ps
	r.Components["ps_score"] = descending(ps, psPoints)
	scores = append(scores, r.Components["ps_score"])
	if positive(in.NetIncome) {
		pe := in.MarketCap / in.NetIncome
		r.Raw["pe_gaap_annual"] = pe
		r.Components["pe_score"] = descending(pe, pePoints)
		scores = append(scores, r.Components["pe_score"])
	}
	if positive(in.EnterpriseValue) && positive(in.EBITDA) {
		ev := in.EnterpriseValue / in.EBITDA
		r.Raw["ev_ebitda_annual"] = ev
		r.Components["ev_ebitda_score"] = descending(ev, evPoints)
		scores = append(scores, r.Components["ev_ebitda_score"])
	}
	return average(scores), len(scores) > 0
}

func annualProfitability(in AnnualInput, r *AnnualResult) (float64, bool) {
	scores := []float64{}
	if finite(in.OperatingIncome) {
		margin := in.OperatingIncome / in.Revenue
		r.Raw["operating_margin_annual"] = margin
		r.Components["operating_margin_score"] = ascending(margin, operatingMarginPoints)
		scores = append(scores, r.Components["operating_margin_score"])
	}
	if finite(in.NetIncome) {
		margin := in.NetIncome / in.Revenue
		r.Raw["net_margin_annual"] = margin
		r.Components["net_margin_score"] = marginScore(margin, in.NetIncome)
		scores = append(scores, r.Components["net_margin_score"])
	}
	if finite(in.OperatingCashFlow) && finite(in.CapitalExpenditures) {
		fcf := in.OperatingCashFlow - math.Abs(in.CapitalExpenditures)
		margin := fcf / in.Revenue
		r.Raw["fcf_margin_annual"] = margin
		r.Components["fcf_margin_score"] = marginScore(margin, fcf)
		scores = append(scores, r.Components["fcf_margin_score"])
	}
	return average(scores), len(scores) > 0
}

func annualGrowth(in AnnualInput, r *AnnualResult) (float64, bool) {
	if !positive(in.Revenue3YAgo) {
		return 0, false
	}
	cagr := math.Pow(in.Revenue/in.Revenue3YAgo, 1.0/3) - 1
	r.Raw["revenue_cagr_3y_annual"] = cagr
	r.Components["revenue_growth_score"] = ascending(cagr, annualGrowthPoints)
	return r.Components["revenue_growth_score"], true
}

func annualLiquidity(in AnnualInput, r *AnnualResult) (float64, bool) {
	if !positive(in.CurrentLiabilities) {
		return 0, false
	}
	scores := []float64{}
	if finite(in.CurrentAssets) {
		current := in.CurrentAssets / in.CurrentLiabilities
		r.Raw["current_ratio_annual"] = current
		r.Components["current_ratio_score"] = ascending(current, currentRatioPoints)
		scores = append(scores, r.Components["current_ratio_score"])
	}
	if finite(in.CurrentAssets) && finite(in.Inventory) {
		quick := (in.CurrentAssets - in.Inventory) / in.CurrentLiabilities
		r.Raw["quick_ratio_annual"] = quick
		r.Components["quick_ratio_score"] = ascending(quick, quickRatioPoints)
		scores = append(scores, r.Components["quick_ratio_score"])
	}
	return average(scores), len(scores) > 0
}

func annualLeverage(in AnnualInput, r *AnnualResult) (float64, bool) {
	scores := []float64{}
	if positive(in.ShareholdersEquity) {
		de := in.TotalDebt / in.ShareholdersEquity
		r.Raw["debt_to_equity_annual"] = de
		r.Components["debt_to_equity_score"] = descending(de, debtEquityPoints)
		scores = append(scores, r.Components["debt_to_equity_score"])
	}
	if positive(in.EBITDA) {
		netDebt := in.TotalDebt - in.CashAndEquivalents
		r.Raw["net_debt_ebitda_annual"] = netDebt / in.EBITDA
		r.Components["net_debt_ebitda_score"] = descending(netDebt/in.EBITDA, netDebtEBITDAPoints)
		scores = append(scores, r.Components["net_debt_ebitda_score"])
	}
	if positive(in.InterestExpense) && finite(in.OperatingIncome) {
		coverage := in.OperatingIncome / math.Abs(in.InterestExpense)
		r.Raw["interest_coverage_annual"] = coverage
		r.Components["interest_coverage_score"] = ascending(coverage, interestCoveragePoints)
		scores = append(scores, r.Components["interest_coverage_score"])
	}
	return average(scores), len(scores) > 0
}

func annualCapitalEfficiency(in AnnualInput, r *AnnualResult) (float64, bool) {
	scores := []float64{}
	if positive(in.InvestedCapital) && finite(in.OperatingIncome) {
		taxRate := .25
		if in.EffectiveTaxRate != nil && *in.EffectiveTaxRate >= 0 && *in.EffectiveTaxRate <= 1 {
			taxRate = *in.EffectiveTaxRate
		}
		roic := in.OperatingIncome * (1 - taxRate) / in.InvestedCapital
		r.Raw["roic_annual"] = roic
		r.Components["roic_score"] = ascending(roic, roicPoints)
		scores = append(scores, r.Components["roic_score"])
	}
	if positive(in.ShareholdersEquity) && finite(in.NetIncome) {
		roe := in.NetIncome / in.ShareholdersEquity
		r.Raw["roe_annual"] = roe
		r.Components["roe_score"] = ascending(roe, roePoints)
		scores = append(scores, r.Components["roe_score"])
	}
	return average(scores), len(scores) > 0
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func incompleteAnnual(r AnnualResult, reason string) AnnualResult {
	r.Status, r.Eligible, r.Confidence, r.ConfidenceLabel = "insufficient_data", false, 0, "insufficient"
	r.Reasons = append(r.Reasons, reason)
	return r
}

func deductAnnual(r *AnnualResult, amount int, reason string) {
	r.Confidence -= amount
	r.Reasons = append(r.Reasons, reason)
}
