package massive

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// FundamentalSnapshot is the current provider-backed financial input for the
// VC/DOSM engine. Values are preserved verbatim by the caller for auditability.
type FundamentalSnapshot struct {
	Ticker                 string    `json:"ticker"`
	FilingDate             time.Time `json:"filing_date"`
	RevenueTTM             float64   `json:"revenue_ttm"`
	Revenue3YAgo           float64   `json:"revenue_3y_ago"`
	NetIncomeTTM           float64   `json:"net_income_gaap_ttm"`
	EBITDATTM              float64   `json:"ebitda_provider_ttm"`
	OperatingIncomeTTM     float64   `json:"operating_income_ttm"`
	OperatingCashFlowTTM   float64   `json:"operating_cash_flow_ttm"`
	CapitalExpendituresTTM float64   `json:"capital_expenditures_ttm"`
	TotalDebt              float64   `json:"total_debt"`
	Cash                   float64   `json:"cash_and_equivalents"`
	Equity                 float64   `json:"shareholders_equity"`
	InvestedCapital        float64   `json:"invested_capital"`
	MarketCap              float64   `json:"market_cap"`
	EnterpriseValue        float64   `json:"enterprise_value"`
	ProviderRequestIDs     []string  `json:"provider_request_ids"`
}

type fundamentalsResponse struct {
	RequestID string `json:"request_id"`
	Results   []struct {
		Tickers                                []string `json:"tickers"`
		FilingDate                             string   `json:"filing_date"`
		Revenue                                float64  `json:"revenue"`
		ConsolidatedNetIncomeLoss              float64  `json:"consolidated_net_income_loss"`
		EBITDA                                 float64  `json:"ebitda"`
		OperatingIncome                        float64  `json:"operating_income"`
		NetCashFlowFromOperatingActivities     float64  `json:"net_cash_flow_from_operating_activities"`
		NetCashFlowFromInvestingActivities     float64  `json:"net_cash_flow_from_investing_activities"`
		CapitalExpenditures                    float64  `json:"capital_expenditures"`
		DebtCurrent                            float64  `json:"debt_current"`
		LongTermDebtAndCapitalLeaseObligations float64  `json:"long_term_debt_and_capital_lease_obligations"`
		CashAndEquivalents                     float64  `json:"cash_and_equivalents"`
		TotalEquity                            float64  `json:"total_equity"`
		TotalLiabilitiesAndEquity              float64  `json:"total_liabilities_and_equity"`
		MarketCap                              float64  `json:"market_cap"`
		EnterpriseValue                        float64  `json:"enterprise_value"`
	} `json:"results"`
}

func (c *Client) GetFundamentalSnapshot(ctx context.Context, ticker string) (FundamentalSnapshot, error) {
	ticker = normalizeSymbol(ticker)
	if ticker == "" {
		return FundamentalSnapshot{}, fmt.Errorf("ticker is required")
	}
	query := url.Values{"ticker": []string{ticker}, "timeframe": []string{"ttm"}, "limit": []string{"1"}}
	var income, cash, balance, ratios, annualIncome fundamentalsResponse
	if err := c.getJSON(ctx, "/stocks/financials/v1/income-statements", query, &income); err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("get income statement: %w", err)
	}
	if err := c.getJSON(ctx, "/stocks/financials/v1/cash-flow-statements", query, &cash); err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("get cash flow statement: %w", err)
	}
	if err := c.getJSON(ctx, "/stocks/financials/v1/balance-sheets", query, &balance); err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("get balance sheet: %w", err)
	}
	if err := c.getJSON(ctx, "/stocks/financials/v1/ratios", query, &ratios); err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("get financial ratios: %w", err)
	}
	annualQuery := url.Values{"ticker": []string{ticker}, "timeframe": []string{"annual"}, "limit": []string{"4"}}
	if err := c.getJSON(ctx, "/stocks/financials/v1/income-statements", annualQuery, &annualIncome); err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("get annual income statement: %w", err)
	}
	if len(income.Results) == 0 || len(cash.Results) == 0 || len(balance.Results) == 0 || len(ratios.Results) == 0 {
		return FundamentalSnapshot{}, fmt.Errorf("fundamental response is incomplete")
	}
	i, cf, b, ra := income.Results[0], cash.Results[0], balance.Results[0], ratios.Results[0]
	revenue3Y := 0.0
	if len(annualIncome.Results) >= 4 {
		revenue3Y = annualIncome.Results[len(annualIncome.Results)-1].Revenue
	}
	filing := latestFiling(i.FilingDate, cf.FilingDate, b.FilingDate, ra.FilingDate)
	return FundamentalSnapshot{Ticker: ticker, FilingDate: filing, RevenueTTM: i.Revenue, Revenue3YAgo: revenue3Y, NetIncomeTTM: i.ConsolidatedNetIncomeLoss, EBITDATTM: i.EBITDA, OperatingIncomeTTM: i.OperatingIncome, OperatingCashFlowTTM: cf.NetCashFlowFromOperatingActivities, CapitalExpendituresTTM: cf.CapitalExpenditures, TotalDebt: b.DebtCurrent + b.LongTermDebtAndCapitalLeaseObligations, Cash: b.CashAndEquivalents, Equity: b.TotalEquity, InvestedCapital: b.TotalLiabilitiesAndEquity - b.CashAndEquivalents, MarketCap: ra.MarketCap, EnterpriseValue: ra.EnterpriseValue, ProviderRequestIDs: compactIDs(income.RequestID, cash.RequestID, balance.RequestID, ratios.RequestID, annualIncome.RequestID)}, nil
}

func latestFiling(values ...string) time.Time {
	var out time.Time
	for _, value := range values {
		t, _ := time.Parse("2006-01-02", strings.TrimSpace(value))
		if t.After(out) {
			out = t
		}
	}
	return out
}
func compactIDs(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
