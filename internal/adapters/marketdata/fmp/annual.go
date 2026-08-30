package fmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AnnualFinancialSnapshot is the immutable provider response boundary for the
// annual VC/DOSM profile. Ratios and key metrics remain provider-reference
// facts; scoring derives its ratios locally from the raw annual statements.
type AnnualFinancialSnapshot struct {
	Ticker              string
	Periods             []AnnualFinancialPeriod
	RatioReferences     map[string]json.RawMessage
	KeyMetricReferences map[string]json.RawMessage
	ProviderRequestIDs  []string
	RetrievedAt         time.Time
}

type AnnualFinancialPeriod struct {
	FiscalYear string
	PeriodEnd  time.Time
	AcceptedAt time.Time
	Income     json.RawMessage
	Balance    json.RawMessage
	CashFlow   json.RawMessage
}

// GetAnnualFinancialSnapshot retrieves exactly five annual periods from each
// Starter-supported statement/reference endpoint. It deliberately does not
// call quarterly or TTM endpoints.
func (c *Client) GetAnnualFinancialSnapshot(ctx context.Context, ticker string) (AnnualFinancialSnapshot, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return AnnualFinancialSnapshot{}, fmt.Errorf("ticker is required")
	}
	requestTicker := normalizeFMPRequestSymbol(ticker)
	const limit = "5"
	paths := []string{
		"/stable/income-statement",
		"/stable/balance-sheet-statement",
		"/stable/cash-flow-statement",
		"/stable/ratios",
		"/stable/key-metrics",
	}
	statementQuery := url.Values{"symbol": {requestTicker}, "period": {"annual"}, "limit": {limit}}
	var income, balance, cash []json.RawMessage
	if err := c.get(ctx, paths[0], statementQuery, &income); err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	if err := c.get(ctx, paths[1], statementQuery, &balance); err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	if err := c.get(ctx, paths[2], statementQuery, &cash); err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	var ratios, metrics []json.RawMessage
	if err := c.get(ctx, paths[3], statementQuery, &ratios); err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	if err := c.get(ctx, paths[4], statementQuery, &metrics); err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	periods, err := mergeAnnualPeriods(income, balance, cash)
	if err != nil {
		return AnnualFinancialSnapshot{}, err
	}
	if len(periods) == 0 {
		return AnnualFinancialSnapshot{}, fmt.Errorf("FMP annual financial response is empty")
	}
	return AnnualFinancialSnapshot{
		Ticker: ticker, Periods: periods,
		RatioReferences:     annualReferenceByPeriod(ratios),
		KeyMetricReferences: annualReferenceByPeriod(metrics),
		ProviderRequestIDs:  paths,
		RetrievedAt:         time.Now().UTC(),
	}, nil
}

func mergeAnnualPeriods(income, balance, cash []json.RawMessage) ([]AnnualFinancialPeriod, error) {
	periods := map[string]*AnnualFinancialPeriod{}
	for _, source := range []struct {
		rows  []json.RawMessage
		apply func(*AnnualFinancialPeriod, json.RawMessage)
	}{
		{income, func(x *AnnualFinancialPeriod, raw json.RawMessage) { x.Income = raw }},
		{balance, func(x *AnnualFinancialPeriod, raw json.RawMessage) { x.Balance = raw }},
		{cash, func(x *AnnualFinancialPeriod, raw json.RawMessage) { x.CashFlow = raw }},
	} {
		for _, raw := range source.rows {
			key, fiscalYear, periodEnd, acceptedAt, err := annualPeriodIdentity(raw)
			if err != nil {
				continue
			}
			period := periods[key]
			if period == nil {
				period = &AnnualFinancialPeriod{FiscalYear: fiscalYear, PeriodEnd: periodEnd, AcceptedAt: acceptedAt}
				periods[key] = period
			}
			if acceptedAt.After(period.AcceptedAt) {
				period.AcceptedAt = acceptedAt
			}
			source.apply(period, raw)
		}
	}
	out := make([]AnnualFinancialPeriod, 0, len(periods))
	for _, period := range periods {
		if len(period.Income) > 0 && len(period.Balance) > 0 && len(period.CashFlow) > 0 {
			out = append(out, *period)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeriodEnd.After(out[j].PeriodEnd) })
	if len(out) > 5 {
		out = out[:5]
	}
	return out, nil
}

func annualReferenceByPeriod(rows []json.RawMessage) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for _, raw := range rows {
		key, _, _, _, err := annualPeriodIdentity(raw)
		if err == nil && key != "" {
			out[key] = raw
		}
	}
	return out
}

func annualPeriodIdentity(raw json.RawMessage) (string, string, time.Time, time.Time, error) {
	var metadata struct {
		Date         string `json:"date"`
		CalendarYear string `json:"calendarYear"`
		FiscalYear   string `json:"fiscalYear"`
		AcceptedDate string `json:"acceptedDate"`
		FilingDate   string `json:"fillingDate"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	periodEnd, err := parseFMPDate(metadata.Date)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	fiscalYear := strings.TrimSpace(metadata.FiscalYear)
	if fiscalYear == "" {
		fiscalYear = strings.TrimSpace(metadata.CalendarYear)
	}
	if fiscalYear == "" {
		fiscalYear = periodEnd.Format("2006")
	}
	acceptedAt, _ := parseFMPDateTime(firstNonEmpty(metadata.AcceptedDate, metadata.FilingDate))
	return periodEnd.Format("2006-01-02"), fiscalYear, periodEnd, acceptedAt, nil
}

func parseFMPDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func parseFMPDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid FMP date %q", value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
