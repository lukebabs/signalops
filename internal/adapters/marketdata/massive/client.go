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
