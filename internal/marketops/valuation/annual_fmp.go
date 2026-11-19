package valuation

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
)

// AnnualInputFromFMP derives the v4 annual input exclusively from raw FMP
// statement rows and a separately retained canonical EOD close. It does not
// consume FMP's ratio or key-metric reference values for scoring.
func AnnualInputFromFMP(ticker string, periods []fmp.AnnualFinancialPeriod, price float64, session time.Time) (AnnualInput, time.Time, bool) {
	in := AnnualInput{Ticker: strings.ToUpper(strings.TrimSpace(ticker)), Price: price}
	if len(periods) == 0 {
		return in, time.Time{}, false
	}
	latest := periods[0]
	in.Revenue = annualNumber(latest.Income, "revenue")
	in.NetIncome = annualNumber(latest.Income, "netIncome")
	in.EBITDA = annualNumber(latest.Income, "ebitda")
	in.OperatingIncome = annualNumber(latest.Income, "operatingIncome")
	in.OperatingCashFlow = annualNumber(latest.CashFlow, "netCashProvidedByOperatingActivities")
	in.CapitalExpenditures = annualNumber(latest.CashFlow, "capitalExpenditure")
	in.TotalDebt = annualNumber(latest.Balance, "totalDebt")
	in.CashAndEquivalents = annualNumber(latest.Balance, "cashAndCashEquivalents")
	in.ShareholdersEquity = annualNumber(latest.Balance, "totalStockholdersEquity")
	in.InvestedCapital = annualNumber(latest.Balance, "totalAssets") - in.CashAndEquivalents
	in.CurrentAssets = annualNumber(latest.Balance, "totalCurrentAssets")
	in.CurrentLiabilities = annualNumber(latest.Balance, "totalCurrentLiabilities")
	in.Inventory = annualNumber(latest.Balance, "inventory")
	in.InterestExpense = annualNumber(latest.Income, "interestExpense")

	shares := annualNumber(latest.Income, "weightedAverageShsOutDil")
	if shares <= 0 {
		shares = annualNumber(latest.Income, "weightedAverageShsOut")
	}
	if positive(price) && positive(shares) {
		in.MarketCap = price * shares
		in.EnterpriseValue = in.MarketCap + in.TotalDebt - in.CashAndEquivalents
	}
	if len(periods) >= 4 {
		in.Revenue3YAgo = annualNumber(periods[3].Income, "revenue")
	}
	if preTax := annualNumber(latest.Income, "incomeBeforeTax"); positive(preTax) {
		tax := math.Abs(annualNumber(latest.Income, "incomeTaxExpense")) / preTax
		if tax >= 0 && tax <= 1 {
			in.EffectiveTaxRate = &tax
		}
	}
	availableAt := latest.AcceptedAt
	if availableAt.IsZero() {
		availableAt = latest.PeriodEnd
	}
	if !availableAt.IsZero() && !session.IsZero() && session.After(availableAt) {
		in.FinancialAgeDays = int(session.Sub(availableAt).Hours() / 24)
	}
	return in, availableAt, true
}

func annualNumber(raw json.RawMessage, field string) float64 {
	if len(raw) == 0 {
		return 0
	}
	values := map[string]json.RawMessage{}
	if json.Unmarshal(raw, &values) != nil {
		return 0
	}
	value := values[field]
	if len(value) == 0 {
		return 0
	}
	var number float64
	if json.Unmarshal(value, &number) == nil && finite(number) {
		return number
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil && finite(parsed) {
			return parsed
		}
	}
	return 0
}
