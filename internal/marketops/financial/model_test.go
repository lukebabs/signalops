package financial

import (
	"testing"
	"time"
)

func TestDeriveUsesFourRollingQuartersForTTMAndNormalizesCapex(t *testing.T) {
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	income, cash := []Quarter{}, []Quarter{}
	for i := 0; i < 4; i++ {
		date := end.AddDate(0, -3*i, 0)
		accepted := date.AddDate(0, 0, 30)
		income = append(income, Quarter{StatementType: "income-statement", PeriodEnd: date, AcceptedAt: accepted, Revenue: 100, OperatingIncome: 20, PretaxIncome: 20, TaxExpense: 5, NetIncome: 15, EBITDA: 25})
		cash = append(cash, Quarter{StatementType: "cash-flow-statement", PeriodEnd: date, AcceptedAt: accepted, OperatingCashFlow: 30, CapitalExpenditure: -4})
	}
	balance := []Quarter{{StatementType: "balance-sheet-statement", PeriodEnd: end, AcceptedAt: end.AddDate(0, 0, 30), Cash: 10, TotalDebt: 30, Equity: 80, TotalAssets: 140}}
	s, err := Derive(Input{Ticker: "TEST", EvaluationDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Price: 10, MarketCap: 200, Income: income, CashFlow: cash, Balance: balance})
	if err != nil {
		t.Fatal(err)
	}
	if s.RevenueTTM != 400 || s.CapitalExpendituresTTM != 16 || s.FreeCashFlowTTM != 104 || s.EnterpriseValue != 220 || s.InvestedCapital != 100 || s.EffectiveTaxRate != .25 {
		t.Fatalf("snapshot=%+v", s)
	}
	if s.GrowthStatus != "unavailable_requires_16_quarters" {
		t.Fatalf("unexpected growth status: %s", s.GrowthStatus)
	}
	if len(s.StatementIDs) != 9 {
		t.Fatalf("statement ids=%d", len(s.StatementIDs))
	}
}
