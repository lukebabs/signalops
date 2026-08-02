package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/adapters/marketdata/fmp"
	"github.com/lukebabs/signalops/internal/adapters/marketdata/massive"
	"github.com/lukebabs/signalops/internal/marketops/financial"
	"github.com/lukebabs/signalops/internal/storage"
)

func loadDerivedFundamentals(ctx context.Context, repo storage.MarketOpsFinancialRepository, fmpClient *fmp.Client, massiveClient *massive.Client, tenant, symbol string, session time.Time, price float64, persist bool) (fmp.FundamentalSnapshot, error) {
	quarters, err := fmpClient.GetQuarterlyStatements(ctx, symbol)
	if err != nil {
		return fmp.FundamentalSnapshot{}, err
	}
	var income, cash, balance []financial.Quarter
	for _, q := range quarters {
		switch q.StatementType {
		case "income-statement":
			income = append(income, q)
		case "cash-flow-statement":
			cash = append(cash, q)
		case "balance-sheet-statement":
			balance = append(balance, q)
		}
	}
	ref, err := massiveClient.GetTickerDetails(ctx, symbol)
	if err != nil {
		return fmp.FundamentalSnapshot{}, err
	}
	shares := ref.ShareClassSharesOutstanding
	if shares == 0 {
		shares = ref.WeightedSharesOutstanding
	}
	derived, err := financial.Derive(financial.Input{Ticker: strings.ToUpper(symbol), EvaluationDate: session.Add(24 * time.Hour).Add(-time.Nanosecond), Price: price, MarketCap: ref.MarketCap, SharesOutstanding: shares, Income: income, CashFlow: cash, Balance: balance})
	if err != nil {
		return fmp.FundamentalSnapshot{}, err
	}
	if persist {
		for _, q := range quarters {
			normalized, _ := json.Marshal(q)
			raw := q.Raw
			if len(raw) == 0 {
				raw = []byte(`{}`)
			}
			id := financialStatementID(tenant, symbol, q)
			if err := repo.UpsertMarketOpsFinancialStatement(ctx, storage.MarketOpsFinancialStatementRecord{StatementID: id, TenantID: tenant, Symbol: symbol, StatementType: q.StatementType, PeriodEnd: q.PeriodEnd, AcceptedAt: q.AcceptedAt, Period: q.Period, NormalizedJSON: normalized, RawJSON: raw}); err != nil {
				return fmp.FundamentalSnapshot{}, err
			}
		}
		input, _ := json.Marshal(map[string]any{"market_provider": "massive", "financial_provider": "fmp-quarterly", "market_cap": ref.MarketCap, "shares_outstanding": shares})
		derivedJSON, _ := json.Marshal(derived)
		if err := repo.UpsertMarketOpsFinancialSnapshot(ctx, storage.MarketOpsFinancialSnapshotRecord{FinancialSnapshotID: stable("financial", tenant, symbol, session.Format("2006-01-02"), derived.AvailableAt.Format(time.RFC3339Nano)), TenantID: tenant, Symbol: symbol, SnapshotVersion: financial.SnapshotVersion, EvaluationDate: session, AvailableAt: derived.AvailableAt, StatementIDs: derived.StatementIDs, InputJSON: input, DerivedJSON: derivedJSON}); err != nil {
			return fmp.FundamentalSnapshot{}, err
		}
	}
	return fmp.FundamentalSnapshot{Ticker: symbol, FilingDate: derived.AvailableAt, RevenueTTM: derived.RevenueTTM, Revenue3YAgo: derived.Revenue3YAgo, NetIncomeTTM: derived.NetIncomeTTM, EBITDATTM: derived.EBITDATTM, OperatingIncomeTTM: derived.OperatingIncomeTTM, OperatingCashFlowTTM: derived.OperatingCashFlowTTM, CapitalExpendituresTTM: derived.CapitalExpendituresTTM, TotalDebt: derived.TotalDebt, Cash: derived.Cash, Equity: derived.Equity, InvestedCapital: derived.InvestedCapital, MarketCap: derived.MarketCap, EnterpriseValue: derived.EnterpriseValue, EffectiveTaxRate: derived.EffectiveTaxRate, ProviderRequestIDs: derived.StatementIDs}, nil
}
func financialStatementID(tenant, symbol string, q financial.Quarter) string {
	return stable("finstmt", tenant, strings.ToUpper(symbol), q.StatementType, q.PeriodEnd.Format("2006-01-02"), q.AcceptedAt.Format(time.RFC3339Nano))
}
