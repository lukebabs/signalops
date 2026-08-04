package fmp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type EarningsCalendarRecord struct {
	Symbol           string   `json:"symbol"`
	Date             string   `json:"date"`
	EPSActual        *float64 `json:"epsActual"`
	EPSEstimated     *float64 `json:"epsEstimated"`
	RevenueActual    *float64 `json:"revenueActual"`
	RevenueEstimated *float64 `json:"revenueEstimated"`
	LastUpdated      string   `json:"lastUpdated"`
}

func (c *Client) GetEarningsCalendar(ctx context.Context, from, to time.Time) ([]EarningsCalendarRecord, error) {
	if from.IsZero() || to.IsZero() || to.Before(from) {
		return nil, fmt.Errorf("valid earnings calendar window is required")
	}
	var rows []EarningsCalendarRecord
	if err := c.get(ctx, "/stable/earnings-calendar", url.Values{"from": {from.UTC().Format("2006-01-02")}, "to": {to.UTC().Format("2006-01-02")}}, &rows); err != nil {
		return nil, err
	}
	out := make([]EarningsCalendarRecord, 0, len(rows))
	for _, row := range rows {
		row.Symbol = strings.ToUpper(strings.TrimSpace(row.Symbol))
		if row.Symbol != "" && row.Date != "" {
			out = append(out, row)
		}
	}
	return out, nil
}
