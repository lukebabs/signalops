// Package fmp provides the financial-statement boundary used by VC/DOSM.
package fmp

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

const DefaultBaseURL = "https://financialmodelingprep.com"

type ClientConfig struct {
	BaseURL, APIKey string
	HTTPClient      *http.Client
}
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	calls      int
}

type FundamentalSnapshot struct {
	Ticker                                                                                 string
	FilingDate                                                                             time.Time
	RevenueTTM, Revenue3YAgo, NetIncomeTTM, EBITDATTM, OperatingIncomeTTM                  float64
	OperatingCashFlowTTM, CapitalExpendituresTTM, TotalDebt, Cash, Equity, InvestedCapital float64
	MarketCap, EnterpriseValue                                                             float64
	EffectiveTaxRate                                                                       float64
	ProviderRequestIDs                                                                     []string
}

func LoadClientConfigFromEnv() ClientConfig {
	key := strings.TrimSpace(os.Getenv("SIGNALOPS_FMP_API_KEY"))
	if key == "" {
		key = strings.TrimSpace(os.Getenv("fmp_apikey"))
	}
	return ClientConfig{BaseURL: strings.TrimSpace(os.Getenv("SIGNALOPS_FMP_BASE_URL")), APIKey: key}
}
func NewClient(cfg ClientConfig) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, errors.New("invalid fmp base url")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("fmp api key is required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: u, apiKey: strings.TrimSpace(cfg.APIKey), httpClient: hc}, nil
}
func (c *Client) Calls() int { return c.calls }

type statement struct {
	Date               string  `json:"date"`
	FilingDate         string  `json:"fillingDate"`
	Revenue            float64 `json:"revenue"`
	NetIncome          float64 `json:"netIncome"`
	EBITDA             float64 `json:"ebitda"`
	OperatingIncome    float64 `json:"operatingIncome"`
	OperatingCashFlow  float64 `json:"netCashProvidedByOperatingActivities"`
	CapitalExpenditure float64 `json:"capitalExpenditure"`
	TotalDebt          float64 `json:"totalDebt"`
	Cash               float64 `json:"cashAndCashEquivalents"`
	Equity             float64 `json:"totalStockholdersEquity"`
	TotalAssets        float64 `json:"totalAssets"`
}
type profile struct {
	MarketCap    float64 `json:"mktCap"`
	MarketCapAlt float64 `json:"marketCap"`
}

func (c *Client) GetFundamentalSnapshot(ctx context.Context, ticker string) (FundamentalSnapshot, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return FundamentalSnapshot{}, errors.New("ticker is required")
	}
	var income, cash, balance []statement
	var annual []statement
	var profiles []profile
	paths := []string{"/stable/income-statement-ttm", "/stable/cash-flow-statement-ttm", "/stable/balance-sheet-statement-ttm", "/stable/profile", "/stable/income-statement"}
	if err := c.get(ctx, paths[0], url.Values{"symbol": {ticker}}, &income); err != nil {
		return FundamentalSnapshot{}, err
	}
	if err := c.get(ctx, paths[1], url.Values{"symbol": {ticker}}, &cash); err != nil {
		return FundamentalSnapshot{}, err
	}
	if err := c.get(ctx, paths[2], url.Values{"symbol": {ticker}}, &balance); err != nil {
		return FundamentalSnapshot{}, err
	}
	if err := c.get(ctx, paths[3], url.Values{"symbol": {ticker}}, &profiles); err != nil {
		return FundamentalSnapshot{}, err
	}
	if err := c.get(ctx, paths[4], url.Values{"symbol": {ticker}, "period": {"annual"}, "limit": {"4"}}, &annual); err != nil {
		return FundamentalSnapshot{}, err
	}
	if len(income) == 0 || len(cash) == 0 || len(balance) == 0 || len(profiles) == 0 || len(annual) < 4 {
		return FundamentalSnapshot{}, errors.New("fmp fundamental response is incomplete")
	}
	marketCap := profiles[0].MarketCap
	if marketCap == 0 {
		marketCap = profiles[0].MarketCapAlt
	}
	if marketCap <= 0 {
		return FundamentalSnapshot{}, errors.New("fmp market capitalization is unavailable")
	}
	i, cf, b := income[0], cash[0], balance[0]
	filing := latest(i.FilingDate, cf.FilingDate, b.FilingDate)
	return FundamentalSnapshot{Ticker: ticker, FilingDate: filing, RevenueTTM: i.Revenue, Revenue3YAgo: annual[len(annual)-1].Revenue, NetIncomeTTM: i.NetIncome, EBITDATTM: i.EBITDA, OperatingIncomeTTM: i.OperatingIncome, OperatingCashFlowTTM: cf.OperatingCashFlow, CapitalExpendituresTTM: cf.CapitalExpenditure, TotalDebt: b.TotalDebt, Cash: b.Cash, Equity: b.Equity, InvestedCapital: b.TotalAssets - b.Cash, MarketCap: marketCap, EnterpriseValue: marketCap + b.TotalDebt - b.Cash, ProviderRequestIDs: paths}, nil
}
func (c *Client) get(ctx context.Context, path string, q url.Values, target any) error {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	c.calls++
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fmp request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fmp request failed with status %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode fmp response: %w", err)
	}
	return nil
}
func latest(values ...string) time.Time {
	var out time.Time
	for _, value := range values {
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, strings.TrimSpace(value)); err == nil && t.After(out) {
				out = t
				break
			}
		}
	}
	return out
}
