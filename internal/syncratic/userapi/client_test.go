package userapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("SYNCRATIC_API_BASE_URL", "https://portal.example")
	t.Setenv("SYNCRATIC_TOKEN_URL", "https://auth.example/token")
	t.Setenv("SYNCRATIC_TOKEN_GRANT", "password")
	t.Setenv("SYNCRATIC_CLIENT_ID", "client")
	t.Setenv("SYNCRATIC_CLIENT_SECRET", "api-key")
	t.Setenv("SYNCRATIC_USERNAME", "user@example.com")
	t.Setenv("SYNCRATIC_PASSWORD", "secret")
	t.Setenv("SYNCRATIC_TOKEN_AUDIENCE", "syncratic-user-api")
	cfg := ConfigFromEnv()
	if cfg.APIBaseURL != "https://portal.example" || cfg.TokenURL != "https://auth.example/token" || cfg.ClientSecret != "api-key" || cfg.TokenAudience != "syncratic-user-api" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestTokenPasswordGrantUsesConfiguredAPIKeyAsClientSecret(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token-1","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()
	client, err := New(Config{APIBaseURL: "https://portal.example", TokenURL: server.URL, TokenGrant: "password", ClientID: "signalops", ClientSecret: "api-key", Username: "user@example.com", Password: "password", TokenAudience: "syncratic-user-api", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	token, err := client.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "token-1" {
		t.Fatalf("token = %+v", token)
	}
	for key, want := range map[string]string{"grant_type": "password", "client_id": "signalops", "client_secret": "api-key", "username": "user@example.com", "password": "password", "audience": "syncratic-user-api"} {
		if got := form.Get(key); got != want {
			t.Fatalf("form[%s] = %q, want %q", key, got, want)
		}
	}
}

func TestSearchAttachesBearerAndCachesToken(t *testing.T) {
	var tokenRequests int32
	var searchAuth string
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenRequests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"cached-token","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		searchAuth = r.Header.Get("Authorization")
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Query != "marketops context" || req.Limit != 3 {
			t.Fatalf("request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_id":"query-1","status":"complete","results":[{"title":"result"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client, err := New(Config{APIBaseURL: server.URL, TokenURL: server.URL + "/token", TokenGrant: "client_credentials", ClientSecret: "api-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		resp, err := client.Search(context.Background(), SearchRequest{Query: "marketops context", Limit: 3})
		if err != nil {
			t.Fatal(err)
		}
		if resp.QueryID != "query-1" || len(resp.Results) != 1 || len(resp.Raw) == 0 {
			t.Fatalf("response = %+v", resp)
		}
	}
	if searchAuth != "Bearer cached-token" {
		t.Fatalf("authorization = %q", searchAuth)
	}
	if atomic.LoadInt32(&tokenRequests) != 1 {
		t.Fatalf("token requests = %d", tokenRequests)
	}
}

func TestAskAndListInsights(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
	})
	mux.HandleFunc("/api/v1/ask", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_id":"ask-1","answer":"answer","confidence":0.8,"evidence_count":2}`))
	})
	mux.HandleFunc("/api/v1/insights", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "2" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result_count":1,"insights":[{"insight_id":"ins-1"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client, err := New(Config{APIBaseURL: server.URL, TokenURL: server.URL + "/token", TokenGrant: "client_credentials", ClientSecret: "api-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	ask, err := client.Ask(context.Background(), AskRequest{Question: "What changed?", K: 4})
	if err != nil {
		t.Fatal(err)
	}
	if ask.QueryID != "ask-1" || ask.Answer != "answer" || ask.EvidenceCount != 2 {
		t.Fatalf("ask = %+v", ask)
	}
	insights, err := client.ListInsights(context.Background(), url.Values{"limit": []string{"2"}})
	if err != nil {
		t.Fatal(err)
	}
	if insights.ResultCount != 1 || len(insights.Insights) != 1 {
		t.Fatalf("insights = %+v", insights)
	}
}

func TestPasswordGrantRequiresUsernameAndPassword(t *testing.T) {
	client, err := New(Config{APIBaseURL: "https://portal.example", TokenURL: "https://auth.example/token", TokenGrant: "password", ClientSecret: "api-key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "username and password") {
		t.Fatalf("err = %v", err)
	}
}

func TestConfigFromEnvDefaultsGrant(t *testing.T) {
	for _, key := range []string{"SYNCRATIC_API_BASE_URL", "SYNCRATIC_TOKEN_URL", "SYNCRATIC_TOKEN_GRANT", "SYNCRATIC_CLIENT_ID", "SYNCRATIC_CLIENT_SECRET", "SYNCRATIC_USERNAME", "SYNCRATIC_PASSWORD", "SYNCRATIC_TOKEN_AUDIENCE"} {
		_ = os.Unsetenv(key)
	}
	cfg := ConfigFromEnv()
	if cfg.TokenGrant != defaultGrant {
		t.Fatalf("grant = %q", cfg.TokenGrant)
	}
}
