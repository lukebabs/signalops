package userapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout = 60 * time.Second
	defaultGrant   = "password"
	AuthModeToken  = "token"
	AuthModeAPIKey = "api_key"
)

// Config contains the Syncratic user-facade API boundary configuration.
// ClientSecret is the Syncratic API key for the configured non-browser token flow.
type Config struct {
	APIBaseURL    string
	AuthMode      string
	TokenURL      string
	TokenGrant    string
	ClientID      string
	ClientSecret  string
	Username      string
	Password      string
	TokenAudience string
	HTTPClient    *http.Client
}

// Client calls the Syncratic user-facing facade described by docs/syncratic_user_api_v1.yaml.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type SearchRequest struct {
	Query   string         `json:"query"`
	Limit   int            `json:"limit,omitempty"`
	Scope   string         `json:"scope,omitempty"`
	Filters map[string]any `json:"filters,omitempty"`
}

type SearchResponse struct {
	QueryID string           `json:"query_id,omitempty"`
	Status  string           `json:"status,omitempty"`
	Results []map[string]any `json:"results,omitempty"`
	Raw     json.RawMessage  `json:"-"`
}

type AskRequest struct {
	Question        string              `json:"question"`
	K               int                 `json:"k,omitempty"`
	Scope           string              `json:"scope,omitempty"`
	Filters         map[string]any      `json:"filters,omitempty"`
	ThreadMode      string              `json:"thread_mode,omitempty"`
	IncludeRefs     *bool               `json:"include_refs,omitempty"`
	DirectReasoning *bool               `json:"direct_reasoning,omitempty"`
	ExternalContext *AskExternalContext `json:"external_context,omitempty"`
	GraphEnabled    *bool               `json:"graph_enabled,omitempty"`
	KEEEnabled      *bool               `json:"kee_enabled,omitempty"`
}

type AskExternalContext struct {
	Items []AskExternalContextItem `json:"items,omitempty"`
}

type AskExternalContextItem struct {
	Title    string `json:"title,omitempty"`
	SourceID string `json:"source_id,omitempty"`
	Text     string `json:"text"`
}

type AskResponse struct {
	QueryID       string           `json:"query_id,omitempty"`
	Answer        string           `json:"answer,omitempty"`
	Confidence    NumericFloat     `json:"confidence,omitempty"`
	EvidenceCount int              `json:"evidence_count,omitempty"`
	Citations     []map[string]any `json:"citations,omitempty"`
	Raw           json.RawMessage  `json:"-"`
}

type NumericFloat float64

func (f *NumericFloat) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" || string(raw) == `""` {
		*f = 0
		return nil
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		*f = NumericFloat(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return err
	}
	*f = NumericFloat(parsed)
	return nil
}

type InsightListResponse struct {
	Insights    []map[string]any `json:"insights,omitempty"`
	ResultCount int              `json:"result_count,omitempty"`
	Raw         json.RawMessage  `json:"-"`
}

func ConfigFromEnv() Config {
	return Config{
		APIBaseURL:    os.Getenv("SYNCRATIC_API_BASE_URL"),
		AuthMode:      firstNonEmpty(os.Getenv("SYNCRATIC_AUTH_MODE"), AuthModeToken),
		TokenURL:      os.Getenv("SYNCRATIC_TOKEN_URL"),
		TokenGrant:    firstNonEmpty(os.Getenv("SYNCRATIC_TOKEN_GRANT"), defaultGrant),
		ClientID:      os.Getenv("SYNCRATIC_CLIENT_ID"),
		ClientSecret:  os.Getenv("SYNCRATIC_CLIENT_SECRET"),
		Username:      os.Getenv("SYNCRATIC_USERNAME"),
		Password:      os.Getenv("SYNCRATIC_PASSWORD"),
		TokenAudience: os.Getenv("SYNCRATIC_TOKEN_AUDIENCE"),
	}
}

func New(cfg Config) (*Client, error) {
	cfg.APIBaseURL = strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	cfg.AuthMode = firstNonEmpty(strings.TrimSpace(cfg.AuthMode), AuthModeToken)
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.TokenGrant = firstNonEmpty(strings.TrimSpace(cfg.TokenGrant), defaultGrant)
	if cfg.APIBaseURL == "" {
		return nil, errors.New("syncratic api base url is required")
	}
	switch cfg.AuthMode {
	case AuthModeAPIKey:
		if strings.TrimSpace(cfg.ClientSecret) == "" {
			return nil, errors.New("syncratic client secret api key is required for api_key auth mode")
		}
	case AuthModeToken:
		if cfg.TokenURL == "" {
			return nil, errors.New("syncratic token url is required")
		}
	default:
		return nil, fmt.Errorf("syncratic auth mode %q is invalid", cfg.AuthMode)
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{cfg: cfg, httpClient: httpClient}, nil
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return SearchResponse{}, errors.New("search query is required")
	}
	var out SearchResponse
	raw, err := c.postJSON(ctx, "/api/v1/search", req, &out)
	out.Raw = raw
	return out, err
}

func (c *Client) Ask(ctx context.Context, req AskRequest) (AskResponse, error) {
	if strings.TrimSpace(req.Question) == "" {
		return AskResponse{}, errors.New("ask question is required")
	}
	var out AskResponse
	raw, err := c.postJSON(ctx, "/api/v1/ask", req, &out)
	out.Raw = raw
	return out, err
}

func (c *Client) ListInsights(ctx context.Context, query url.Values) (InsightListResponse, error) {
	path := "/api/v1/insights"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var out InsightListResponse
	raw, err := c.getJSON(ctx, path, &out)
	out.Raw = raw
	return out, err
}

func (c *Client) Token(ctx context.Context) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", c.cfg.TokenGrant)
	if strings.TrimSpace(c.cfg.ClientID) != "" {
		form.Set("client_id", strings.TrimSpace(c.cfg.ClientID))
	}
	if strings.TrimSpace(c.cfg.ClientSecret) != "" {
		form.Set("client_secret", strings.TrimSpace(c.cfg.ClientSecret))
	}
	if strings.TrimSpace(c.cfg.TokenAudience) != "" {
		form.Set("audience", strings.TrimSpace(c.cfg.TokenAudience))
	}
	if c.cfg.TokenGrant == "password" {
		if strings.TrimSpace(c.cfg.Username) == "" || strings.TrimSpace(c.cfg.Password) == "" {
			return TokenResponse{}, errors.New("syncratic username and password are required for password grant")
		}
		form.Set("username", strings.TrimSpace(c.cfg.Username))
		form.Set("password", strings.TrimSpace(c.cfg.Password))
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("request syncratic token: %w", err)
	}
	defer resp.Body.Close()
	raw, err := readLimited(resp.Body)
	if err != nil {
		return TokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenResponse{}, fmt.Errorf("syncratic token request failed: status %d: %s", resp.StatusCode, compactBody(raw))
	}
	var token TokenResponse
	if err := json.Unmarshal(raw, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("decode syncratic token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, errors.New("syncratic token response missing access_token")
	}
	return token, nil
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if c.cfg.AuthMode == AuthModeAPIKey {
		return strings.TrimSpace(c.cfg.ClientSecret), nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.token, nil
	}
	token, err := c.Token(ctx)
	if err != nil {
		return "", err
	}
	c.token = token.AccessToken
	if token.ExpiresIn > 0 {
		c.expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	} else {
		c.expiresAt = time.Now().Add(5 * time.Minute)
	}
	return c.token, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, dest any) ([]byte, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIBaseURL+path, bytes.NewReader(rawPayload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	return c.doJSON(ctx, httpReq, dest)
}

func (c *Client) getJSON(ctx context.Context, path string, dest any) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.APIBaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.doJSON(ctx, httpReq, dest)
}

func (c *Client) doJSON(ctx context.Context, httpReq *http.Request, dest any) ([]byte, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	if c.cfg.AuthMode == AuthModeAPIKey {
		httpReq.Header.Set("X-API-Key", token)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request syncratic api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := readLimited(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, fmt.Errorf("syncratic api request failed: status %d: %s", resp.StatusCode, compactBody(raw))
	}
	if dest != nil {
		if err := json.Unmarshal(raw, dest); err != nil {
			return raw, fmt.Errorf("decode syncratic api response: %w", err)
		}
	}
	return raw, nil
}

func readLimited(body io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(body, 1<<20))
}

func compactBody(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > 512 {
		return text[:512]
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
