package api

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const testAuthIssuer = "https://auth.syncratic.co/realms/syncratic"
const testAuthAudience = "signalops-api"

type testAuthFixture struct {
	key     *rsa.PrivateKey
	kid     string
	jwks    *httptest.Server
	now     time.Time
	authCfg AuthConfig
}

func newTestAuthFixture(t *testing.T) *testAuthFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &testAuthFixture{key: key, kid: "test-key", now: time.Date(2026, 7, 9, 3, 40, 0, 0, time.UTC)}
	fixture.jwks = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{fixture.jwk()}})
	}))
	t.Cleanup(fixture.jwks.Close)
	fixture.authCfg = AuthConfig{Enabled: true, Issuer: testAuthIssuer, JWKSURL: fixture.jwks.URL, Audience: testAuthAudience, Now: func() time.Time { return fixture.now }}
	return fixture
}

func (f *testAuthFixture) jwk() map[string]string {
	public := f.key.Public().(*rsa.PublicKey)
	eBytes := big.NewInt(int64(public.E)).Bytes()
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": f.kid,
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

func (f *testAuthFixture) token(t *testing.T, claims map[string]any) string {
	t.Helper()
	baseClaims := map[string]any{
		"iss":                testAuthIssuer,
		"sub":                "user-123",
		"preferred_username": "operator-auth",
		"email":              "operator@example.test",
		"tenant_id":          "tenant-local",
		"aud":                []string{testAuthAudience},
		"exp":                f.now.Add(1 * time.Hour).Unix(),
		"realm_access":       map[string]any{"roles": []string{roleViewer}},
	}
	for key, value := range claims {
		baseClaims[key] = value
	}
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": f.kid}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	claimsJSON, err := json.Marshal(baseClaims)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func withBearer(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestAuthEnabledLeavesHealthPublic(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{ServiceName: "test-gateway", Auth: fixture.authCfg})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuthEnabledRejectsProtectedRouteWithoutBearer(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: &fakeQueryRepository{}})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/alerts?tenant_id=tenant-local", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestAuthEnabledAllowsViewerRead(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{validAlertRecord()}}})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleViewer}}})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodGet, "/v1/alerts?tenant_id=tenant-local", nil), token))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuthEnabledRejectsMissingTenantClaim(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: &fakeQueryRepository{}})
	token := fixture.token(t, map[string]any{"tenant_id": ""})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodGet, "/v1/alerts", nil), token))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "missing_tenant_claim") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestAuthEnabledRejectsTenantMismatch(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: &fakeQueryRepository{}})
	token := fixture.token(t, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodGet, "/v1/alerts?tenant_id=tenant-other", nil), token))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "tenant_mismatch") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestAuthEnabledRejectsViewerLifecycleMutation(t *testing.T) {
	fixture := newTestAuthFixture(t)
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{validAlertRecord()}}})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleViewer}}})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withBearer(httptest.NewRequest(http.MethodPost, "/v1/alerts/alert-1/acknowledge", nil), token))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuthEnabledUsesTokenActorForLifecycleMutation(t *testing.T) {
	fixture := newTestAuthFixture(t)
	repo := &fakeQueryRepository{alerts: []storage.AlertLedgerRecord{validAlertRecord()}}
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, QueryRepository: repo})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleAdmin}}})
	req := httptest.NewRequest(http.MethodPost, "/v1/alerts/alert-1/acknowledge", bytes.NewBufferString(`{"actor":"body-actor","note":"triaged"}`))
	req.Header.Set("X-SignalOps-Actor", "header-actor")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, withBearer(req, token))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Alert alertDTO `json:"alert"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Alert.AcknowledgedBy != "operator-auth" {
		t.Fatalf("AcknowledgedBy = %q", body.Alert.AcknowledgedBy)
	}
	if !bytes.Contains(body.Alert.Metadata, []byte(`"actor":"operator-auth"`)) {
		t.Fatalf("metadata = %s", string(body.Alert.Metadata))
	}
}
