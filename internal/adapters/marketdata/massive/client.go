package massive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.massive.com"

type ClientConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

func LoadClientConfigFromEnv() ClientConfig {
	return ClientConfig{
		BaseURL: strings.TrimSpace(os.Getenv("SIGNALOPS_MASSIVE_BASE_URL")),
		APIKey:  firstNonEmptyEnv("SIGNALOPS_MASSIVE_API_KEY", "MASSIVE_API_KEY", "API_KEY"),
	}
}

func NewClient(cfg ClientConfig) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid massive base url")
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("massive api key is required")
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: parsed, apiKey: apiKey, httpClient: httpClient}, nil
}

func (c *Client) ListOptionContracts(ctx context.Context, underlying string, asOf time.Time, limit int) ([]OptionContractDailyRecord, error) {
	query := url.Values{}
	query.Set("underlying_ticker", normalizeSymbol(underlying))
	query.Set("as_of", dateKey(asOf))
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	var response optionContractsResponse
	if err := c.getJSON(ctx, "/v3/reference/options/contracts", query, &response); err != nil {
		return nil, err
	}
	return response.records(asOf), nil
}

func (c *Client) ListOptionChainSnapshot(ctx context.Context, underlying string, limit int, maxPages int) ([]OptionContractDailyRecord, error) {
	return c.ListOptionChainSnapshotFiltered(ctx, underlying, OptionChainSnapshotFilter{Limit: limit, MaxPages: maxPages})
}

// OptionChainSnapshotFilter bounds an option-chain request at the provider.
// Zero-valued dates and nil strike bounds are omitted.
type OptionChainSnapshotFilter struct {
	Limit             int
	MaxPages          int
	ExpirationDateGTE time.Time
	ExpirationDateLTE time.Time
	StrikePriceGTE    *float64
	StrikePriceLTE    *float64
}

func (c *Client) ListOptionChainSnapshotFiltered(ctx context.Context, underlying string, filter OptionChainSnapshotFilter) ([]OptionContractDailyRecord, error) {
	underlying = normalizeSymbol(underlying)
	if underlying == "" {
		return nil, errors.New("underlying symbol is required")
	}
	if filter.Limit <= 0 || filter.Limit > 250 {
		filter.Limit = 250
	}
	if filter.MaxPages <= 0 || filter.MaxPages > 20 {
		filter.MaxPages = 1
	}
	if !filter.ExpirationDateGTE.IsZero() && !filter.ExpirationDateLTE.IsZero() && filter.ExpirationDateLTE.Before(filter.ExpirationDateGTE) {
		return nil, errors.New("option-chain expiration upper bound must not precede lower bound")
	}
	if filter.StrikePriceGTE != nil && filter.StrikePriceLTE != nil && *filter.StrikePriceLTE < *filter.StrikePriceGTE {
		return nil, errors.New("option-chain strike upper bound must not precede lower bound")
	}
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", filter.Limit))
	if !filter.ExpirationDateGTE.IsZero() {
		query.Set("expiration_date.gte", dateKey(filter.ExpirationDateGTE))
	}
	if !filter.ExpirationDateLTE.IsZero() {
		query.Set("expiration_date.lte", dateKey(filter.ExpirationDateLTE))
	}
	if filter.StrikePriceGTE != nil {
		query.Set("strike_price.gte", fmt.Sprintf("%.8f", *filter.StrikePriceGTE))
	}
	if filter.StrikePriceLTE != nil {
		query.Set("strike_price.lte", fmt.Sprintf("%.8f", *filter.StrikePriceLTE))
	}
	path := fmt.Sprintf("/v3/snapshot/options/%s", url.PathEscape(underlying))
	fallback := time.Now().UTC()
	records := []OptionContractDailyRecord{}
	nextURL := ""
	for page := 0; page < filter.MaxPages; page++ {
		var response optionChainSnapshotResponse
		var err error
		if page == 0 {
			err = c.getJSON(ctx, path, query, &response)
		} else {
			err = c.getJSONURL(ctx, nextURL, &response)
		}
		if err != nil {
			return nil, err
		}
		records = append(records, response.records(underlying, fallback)...)
		nextURL = strings.TrimSpace(response.NextURL)
		if nextURL == "" {
			break
		}
	}
	return records, nil
}

func (c *Client) GetEquityDailyBar(ctx context.Context, symbol string, date time.Time) (EquityEODPriceRecord, error) {
	path := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/day/%s/%s", url.PathEscape(normalizeSymbol(symbol)), dateKey(date), dateKey(date))
	var response aggregateBarsResponse
	if err := c.getJSON(ctx, path, nil, &response); err != nil {
		return EquityEODPriceRecord{}, err
	}
	records := response.equityRecords(normalizeSymbol(symbol), date)
	if len(records) == 0 {
		return EquityEODPriceRecord{}, errors.New("massive equity aggregate response contained no bars")
	}
	return records[0], nil
}

func (c *Client) GetOptionDailyBar(ctx context.Context, optionTicker string, underlying string, date time.Time) (OptionContractDailyRecord, error) {
	optionTicker = strings.TrimSpace(optionTicker)
	path := fmt.Sprintf("/v2/aggs/ticker/%s/range/1/day/%s/%s", url.PathEscape(optionTicker), dateKey(date), dateKey(date))
	var response aggregateBarsResponse
	if err := c.getJSON(ctx, path, nil, &response); err != nil {
		return OptionContractDailyRecord{}, err
	}
	records := response.optionRecords(optionTicker, underlying, date)
	if len(records) == 0 {
		return OptionContractDailyRecord{}, errors.New("massive option aggregate response contained no bars")
	}
	return records[0], nil
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, target any) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: path})
	values := endpoint.Query()
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	values.Set("apiKey", c.apiKey)
	endpoint.RawQuery = values.Encode()
	return c.getJSONEndpoint(ctx, endpoint, target)
}

func (c *Client) getJSONURL(ctx context.Context, rawURL string, target any) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid massive next url")
	}
	values := parsed.Query()
	if values.Get("apiKey") == "" {
		values.Set("apiKey", c.apiKey)
		parsed.RawQuery = values.Encode()
	}
	return c.getJSONEndpoint(ctx, parsed, target)
}

func (c *Client) getJSONEndpoint(ctx context.Context, endpoint *url.URL, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("build massive request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("massive request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return fmt.Errorf("read massive response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("massive request failed with status %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode massive response: %w", err)
	}
	return nil
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
