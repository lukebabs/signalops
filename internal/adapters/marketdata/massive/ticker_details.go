package massive

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type TickerDetails struct {
	Ticker   string
	Name     string
	Exchange string
	Market   string
	Type     string
	Active   bool
	Sector   string
	Industry string
}

type tickerDetailsResponse struct {
	Results struct {
		Ticker          string `json:"ticker"`
		Name            string `json:"name"`
		PrimaryExchange string `json:"primary_exchange"`
		Market          string `json:"market"`
		Type            string `json:"type"`
		Active          bool   `json:"active"`
		SICDescription  string `json:"sic_description"`
	} `json:"results"`
}

func (c *Client) GetTickerDetails(ctx context.Context, ticker string) (TickerDetails, error) {
	ticker = normalizeSymbol(ticker)
	if ticker == "" {
		return TickerDetails{}, errors.New("ticker is required")
	}
	var response tickerDetailsResponse
	if err := c.getJSON(ctx, "/v3/reference/tickers/"+url.PathEscape(ticker), nil, &response); err != nil {
		return TickerDetails{}, err
	}
	out := TickerDetails{Ticker: normalizeSymbol(response.Results.Ticker), Name: strings.TrimSpace(response.Results.Name), Exchange: strings.TrimSpace(response.Results.PrimaryExchange), Market: strings.ToLower(strings.TrimSpace(response.Results.Market)), Type: strings.ToLower(strings.TrimSpace(response.Results.Type)), Active: response.Results.Active, Industry: strings.TrimSpace(response.Results.SICDescription)}
	if out.Ticker == "" || out.Name == "" {
		return TickerDetails{}, fmt.Errorf("massive ticker reference response is incomplete")
	}
	return out, nil
}
