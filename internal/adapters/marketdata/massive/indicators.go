package massive

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"
)

// IndicatorValue is one adjusted daily technical-indicator observation.
// Massive uses the same timestamp/value envelope for SMA and RSI.
type IndicatorValue struct {
	Timestamp time.Time
	Value     float64
}

// SimpleMovingAverage preserves the caller-facing name for SMA values.
type SimpleMovingAverage = IndicatorValue

// RelativeStrengthIndex preserves the caller-facing name for RSI values.
type RelativeStrengthIndex = IndicatorValue

// ListSimpleMovingAverage reads a bounded daily SMA series from Massive.
func (c *Client) ListSimpleMovingAverage(ctx context.Context, symbol string, window, limit int) ([]SimpleMovingAverage, error) {
	return c.listIndicator(ctx, "sma", symbol, window, limit)
}

// ListRelativeStrengthIndex reads a bounded daily Wilder-style RSI series from
// Massive. It is intentionally provider-backed: callers do not infer RSI from
// locally retained EOD history.
func (c *Client) ListRelativeStrengthIndex(ctx context.Context, symbol string, window, limit int) ([]RelativeStrengthIndex, error) {
	return c.listIndicator(ctx, "rsi", symbol, window, limit)
}

func (c *Client) listIndicator(ctx context.Context, indicator, symbol string, window, limit int) ([]IndicatorValue, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" || window <= 0 || limit <= 0 {
		return nil, errors.New("symbol, window, and limit are required")
	}
	q := url.Values{}
	q.Set("timespan", "day")
	q.Set("adjusted", "true")
	q.Set("window", fmt.Sprintf("%d", window))
	q.Set("series_type", "close")
	q.Set("limit", fmt.Sprintf("%d", limit))
	var response struct {
		Results struct {
			Values []struct {
				Timestamp int64   `json:"timestamp"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"results"`
	}
	if err := c.getJSON(ctx, "/v1/indicators/"+indicator+"/"+url.PathEscape(symbol), q, &response); err != nil {
		return nil, err
	}
	values := make([]IndicatorValue, 0, len(response.Results.Values))
	for _, item := range response.Results.Values {
		if item.Timestamp > 0 {
			values = append(values, IndicatorValue{Timestamp: time.UnixMilli(item.Timestamp).UTC(), Value: item.Value})
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Timestamp.After(values[j].Timestamp) })
	if len(values) == 0 {
		return nil, fmt.Errorf("massive %s response contained no values", indicator)
	}
	return values, nil
}
