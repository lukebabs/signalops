// Package financial builds immutable, locally-derived GAAP financial snapshots.
package financial

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const SnapshotVersion = "financial-v1.1-ttm"

type Quarter struct {
	StatementType                                                                      string    `json:"statement_type"`
	PeriodEnd                                                                          time.Time `json:"period_end"`
	AcceptedAt                                                                         time.Time `json:"accepted_at"`
	Period                                                                             string    `json:"period"`
	Revenue, GrossProfit, OperatingIncome, PretaxIncome, TaxExpense, NetIncome, EBITDA float64
	OperatingCashFlow, CapitalExpenditure                                              float64
	Cash, TotalDebt, Equity, TotalAssets, SharesOutstanding                            float64
	Raw                                                                                []byte `json:"raw"`
}

type Input struct {
	Ticker                              string
	EvaluationDate                      time.Time
	Price, MarketCap, SharesOutstanding float64
	Income, CashFlow, Balance           []Quarter
}
type Snapshot struct {
	Ticker                                                                                                                                                                               string `json:"ticker"`
	EvaluationDate, AvailableAt                                                                                                                                                          time.Time
	Price, MarketCap, SharesOutstanding, EnterpriseValue                                                                                                                                 float64
	RevenueTTM, Revenue3YAgo, GrossProfitTTM, OperatingIncomeTTM, PretaxIncomeTTM, TaxExpenseTTM, NetIncomeTTM, EBITDATTM, OperatingCashFlowTTM, CapitalExpendituresTTM, FreeCashFlowTTM float64
	Cash, TotalDebt, Equity, InvestedCapital, EffectiveTaxRate, NOPAT, RevenueCAGR3Y                                                                                                     float64
	GrowthStatus                                                                                                                                                                         string    `json:"growth_status"`
	StatementIDs                                                                                                                                                                         []string  `json:"statement_ids"`
	Raw                                                                                                                                                                                  []Quarter `json:"selected_quarters"`
}

func Derive(in Input) (Snapshot, error) {
	if in.Ticker == "" || in.EvaluationDate.IsZero() || in.Price <= 0 || in.MarketCap <= 0 {
		return Snapshot{}, fmt.Errorf("ticker, evaluation date, price, and market cap are required")
	}
	income := selectQuarters(in.Income, in.EvaluationDate, 4)
	cash := selectQuarters(in.CashFlow, in.EvaluationDate, 4)
	balance := selectQuarters(in.Balance, in.EvaluationDate, 1)
	if len(income) < 4 || len(cash) < 4 || len(balance) < 1 {
		return Snapshot{}, fmt.Errorf("four accepted income/cash-flow quarters and one balance quarter are required for TTM")
	}
	s := Snapshot{Ticker: in.Ticker, EvaluationDate: in.EvaluationDate, Price: in.Price, MarketCap: in.MarketCap, SharesOutstanding: in.SharesOutstanding, Cash: balance[0].Cash, TotalDebt: balance[0].TotalDebt, Equity: balance[0].Equity, GrowthStatus: "unavailable_requires_16_quarters"}
	s.AvailableAt = latest(income[0].AcceptedAt, cash[0].AcceptedAt, balance[0].AcceptedAt)
	for _, q := range income[:4] {
		s.RevenueTTM += q.Revenue
		s.GrossProfitTTM += q.GrossProfit
		s.OperatingIncomeTTM += q.OperatingIncome
		s.PretaxIncomeTTM += q.PretaxIncome
		s.TaxExpenseTTM += q.TaxExpense
		s.NetIncomeTTM += q.NetIncome
		s.EBITDATTM += q.EBITDA
		s.Raw = append(s.Raw, q)
	}
	for _, q := range cash[:4] {
		s.OperatingCashFlowTTM += q.OperatingCashFlow
		s.CapitalExpendituresTTM += math.Abs(q.CapitalExpenditure)
		s.Raw = append(s.Raw, q)
	}
	s.FreeCashFlowTTM = s.OperatingCashFlowTTM - s.CapitalExpendituresTTM
	s.EnterpriseValue = s.MarketCap + s.TotalDebt - s.Cash
	s.InvestedCapital = s.TotalDebt + s.Equity - s.Cash
	s.EffectiveTaxRate = .25
	if s.PretaxIncomeTTM > 0 {
		s.EffectiveTaxRate = math.Max(0, math.Min(.4, s.TaxExpenseTTM/s.PretaxIncomeTTM))
	}
	s.NOPAT = s.OperatingIncomeTTM * (1 - s.EffectiveTaxRate)
	for _, q := range append(append([]Quarter{}, income...), cash...) {
		s.StatementIDs = append(s.StatementIDs, id(q))
	}
	s.StatementIDs = append(s.StatementIDs, id(balance[0]))
	return s, nil
}
func selectQuarters(rows []Quarter, asOf time.Time, n int) []Quarter {
	byPeriod := map[string]Quarter{}
	for _, q := range rows {
		if q.AcceptedAt.After(asOf) || q.PeriodEnd.IsZero() {
			continue
		}
		k := q.StatementType + "|" + q.PeriodEnd.Format("2006-01-02")
		if old, ok := byPeriod[k]; !ok || q.AcceptedAt.After(old.AcceptedAt) {
			byPeriod[k] = q
		}
	}
	out := make([]Quarter, 0, len(byPeriod))
	for _, q := range byPeriod {
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeriodEnd.After(out[j].PeriodEnd) })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
func id(q Quarter) string {
	return q.StatementType + ":" + q.PeriodEnd.Format("2006-01-02") + ":" + q.AcceptedAt.UTC().Format(time.RFC3339)
}
func latest(v ...time.Time) time.Time {
	var out time.Time
	for _, x := range v {
		if x.After(out) {
			out = x
		}
	}
	return out
}
