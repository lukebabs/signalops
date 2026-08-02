package fmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/marketops/financial"
)

type quarterlyResponse struct {
	Date, AcceptedDate, Period                                                                            string
	Revenue, GrossProfit, OperatingIncome, IncomeBeforeTax, IncomeTaxExpense, NetIncome, EBITDA           float64
	NetCashProvidedByOperatingActivities, OperatingCashFlow, CapitalExpenditure                           float64
	CashAndCashEquivalents, TotalDebt, TotalStockholdersEquity, TotalAssets, CommonStockSharesOutstanding float64
}

func (c *Client) GetQuarterlyStatements(ctx context.Context, ticker string) ([]financial.Quarter, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return nil, fmt.Errorf("ticker is required")
	}
	paths := []string{"/stable/income-statement", "/stable/balance-sheet-statement", "/stable/cash-flow-statement"}
	out := []financial.Quarter{}
	for _, path := range paths {
		var rows []quarterlyResponse
		if err := c.get(ctx, path, url.Values{"symbol": {ticker}, "period": {"quarter"}, "limit": {"4"}}, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			end, err := time.Parse("2006-01-02", row.Date)
			if err != nil {
				continue
			}
			location, _ := time.LoadLocation("America/New_York")
			accepted, err := time.ParseInLocation("2006-01-02 15:04:05", row.AcceptedDate, location)
			if err != nil {
				continue
			}
			raw, _ := json.Marshal(row)
			out = append(out, financial.Quarter{StatementType: strings.TrimPrefix(path, "/stable/"), PeriodEnd: end, AcceptedAt: accepted.UTC(), Period: row.Period, Revenue: row.Revenue, GrossProfit: row.GrossProfit, OperatingIncome: row.OperatingIncome, PretaxIncome: row.IncomeBeforeTax, TaxExpense: row.IncomeTaxExpense, NetIncome: row.NetIncome, EBITDA: row.EBITDA, OperatingCashFlow: firstNonZero(row.NetCashProvidedByOperatingActivities, row.OperatingCashFlow), CapitalExpenditure: row.CapitalExpenditure, Cash: row.CashAndCashEquivalents, TotalDebt: row.TotalDebt, Equity: row.TotalStockholdersEquity, TotalAssets: row.TotalAssets, SharesOutstanding: row.CommonStockSharesOutstanding, Raw: raw})
		}
	}
	return out, nil
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
