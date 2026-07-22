package massive

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// EquityQuote is the most recent intraday price, or the most recent completed
// daily close when the market is closed or intraday aggregates are unavailable.
type EquityQuote struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Timestamp     time.Time `json:"timestamp"`
	MarketStatus  string    `json:"market_status"`
	Stale         bool      `json:"stale"`
	PreviousClose *float64  `json:"previous_close,omitempty"`
	Change        *float64  `json:"change,omitempty"`
	ChangePercent *float64  `json:"change_percent,omitempty"`
	Week52Low     *float64  `json:"week52_low,omitempty"`
	Week52High    *float64  `json:"week52_high,omitempty"`
}

func (c *Client) GetEquityQuote(ctx context.Context, symbol string) (EquityQuote, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return EquityQuote{}, fmt.Errorf("symbol is required")
	}

	today := time.Now().UTC()
	quote := EquityQuote{Symbol: symbol, MarketStatus: "intraday"}
	var effectiveDay time.Time
	var minute aggregateBarsResponse
	minutePath := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/minute/%s/%s", url.PathEscape(symbol), dateKey(today), dateKey(today))
	minuteQuery := url.Values{"sort": []string{"desc"}, "limit": []string{"1"}}
	if err := c.getJSON(ctx, minutePath, minuteQuery, &minute); err == nil {
		records := minute.equityRecords(symbol, today)
		if len(records) > 0 && records[0].Close != nil {
			quote.Price = *records[0].Close
			quote.Timestamp = today
			if len(minute.Results) > 0 && minute.Results[0].Timestamp != nil {
				quote.Timestamp = time.UnixMilli(*minute.Results[0].Timestamp).UTC()
			}
			effectiveDay = today
		}
	}

	// Minute aggregates are unavailable before the session opens and after a
	// non-trading day. In that case, present the latest completed daily close.
	if effectiveDay.IsZero() {
		for offset := 0; offset <= 7; offset++ {
			candidate := today.AddDate(0, 0, -offset)
			var daily aggregateBarsResponse
			path := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/day/%s/%s", url.PathEscape(symbol), dateKey(candidate), dateKey(candidate))
			if err := c.getJSON(ctx, path, nil, &daily); err != nil {
				continue
			}
			records := daily.equityRecords(symbol, candidate)
			if len(records) == 0 || records[0].Close == nil {
				continue
			}
			quote.Price = *records[0].Close
			quote.Timestamp = candidate
			if len(daily.Results) > 0 && daily.Results[0].Timestamp != nil {
				quote.Timestamp = time.UnixMilli(*daily.Results[0].Timestamp).UTC()
			}
			quote.MarketStatus = "end_of_day"
			quote.Stale = offset > 0
			effectiveDay = candidate
			break
		}
	}
	if effectiveDay.IsZero() {
		return EquityQuote{}, fmt.Errorf("quote response contained no price")
	}

	for offset := 1; offset <= 7; offset++ {
		candidate := effectiveDay.AddDate(0, 0, -offset)
		var prior aggregateBarsResponse
		path := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/day/%s/%s", url.PathEscape(symbol), dateKey(candidate), dateKey(candidate))
		if err := c.getJSON(ctx, path, nil, &prior); err != nil {
			continue
		}
		records := prior.equityRecords(symbol, candidate)
		if len(records) > 0 && records[0].Close != nil {
			value := *records[0].Close
			quote.PreviousClose = &value
			break
		}
	}

	start := effectiveDay.AddDate(0, 0, -365)
	var year aggregateBarsResponse
	path := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/day/%s/%s", url.PathEscape(symbol), dateKey(start), dateKey(effectiveDay))
	if err := c.getJSON(ctx, path, url.Values{"limit": []string{"50000"}}, &year); err == nil {
		var low, high float64
		found := false
		for _, bar := range year.Results {
			if bar.Low == nil || bar.High == nil {
				continue
			}
			if !found || *bar.Low < low {
				low = *bar.Low
			}
			if !found || *bar.High > high {
				high = *bar.High
			}
			found = true
		}
		if found {
			quote.Week52Low = &low
			quote.Week52High = &high
		}
	}
	if quote.PreviousClose != nil {
		change := quote.Price - *quote.PreviousClose
		percent := change / *quote.PreviousClose * 100
		quote.Change = &change
		quote.ChangePercent = &percent
	}
	return quote, nil
}

func NormalizeQuoteSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
