package api

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	roleViewer   = "signalops:viewer"
	roleOperator = "signalops:operator"
	roleAdmin    = "signalops:admin"
)

type authContextKey struct{}

// AuthConfig controls optional OIDC/JWT enforcement for gateway routes.
type AuthConfig struct {
	Enabled      bool
	Issuer       string
	JWKSURL      string
	Audience     string
	HTTPClient   *http.Client
	Now          func() time.Time
	JWKSCacheTTL time.Duration
}

// Principal is the authenticated SignalOps operator context derived from a JWT.
type Principal struct {
	Subject  string
	Actor    string
	TenantID string
	Roles    map[string]struct{}
}

func principalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(authContextKey{}).(Principal)
	return principal, ok
}

type authVerifier struct {
	cfg    AuthConfig
	client *http.Client
	now    func() time.Time

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func newAuthVerifier(cfg AuthConfig) *authVerifier {
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &authVerifier{cfg: cfg, client: client, now: now}
}

func authMiddleware(next http.Handler, cfg AuthConfig) http.Handler {
	if !cfg.Enabled {
		return next
	}
	verifier := newAuthVerifier(cfg)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r) || !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		principal, err := verifier.authenticate(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		if strings.TrimSpace(principal.TenantID) == "" {
			writeError(w, http.StatusForbidden, "missing_tenant_claim", "token must include tenant_id")
			return
		}
		if !tenantMatchesRequest(r, principal.TenantID) {
			writeError(w, http.StatusForbidden, "tenant_mismatch", "request tenant does not match token tenant")
			return
		}
		if !authorizedForRequest(r, principal) {
			writeError(w, http.StatusForbidden, "insufficient_role", "token does not include the required SignalOps role")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey{}, principal)))
	})
}

func isPublicRoute(r *http.Request) bool {
	return r.Method == http.MethodGet && (r.URL.Path == "/healthz" || r.URL.Path == "/readyz")
}

func tenantMatchesRequest(r *http.Request, tenantID string) bool {
	if queryTenant := strings.TrimSpace(r.URL.Query().Get("tenant_id")); queryTenant != "" && queryTenant != tenantID {
		return false
	}
	if pathTenant := tenantFromPath(r.URL.Path); pathTenant != "" && pathTenant != tenantID {
		return false
	}
	return true
}

func tenantFromPath(path string) string {
	prefix := "/v1/tenants/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	idx := strings.IndexByte(rest, '/')
	if idx < 0 {
		return rest
	}
	return rest[:idx]
}

func authorizedForRequest(r *http.Request, principal Principal) bool {
	if isLifecycleMutationRoute(r) {
		return hasAnyRole(principal, roleOperator, roleAdmin)
	}
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		return hasAnyRole(principal, roleViewer, roleOperator, roleAdmin)
	}
	return true
}

func isLifecycleMutationRoute(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" {
		return false
	}
	switch parts[1] {
	case "alerts":
		switch parts[3] {
		case "acknowledge", "resolve", "suppress":
			return true
		}
	case "insights":
		switch parts[3] {
		case "review", "dismiss", "archive":
			return true
		}
	}
	return false
}

func hasAnyRole(principal Principal, roles ...string) bool {
	for _, role := range roles {
		if _, ok := principal.Roles[role]; ok {
			return true
		}
	}
	return false
}

func (v *authVerifier) authenticate(r *http.Request) (Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return Principal{}, errors.New("missing bearer token")
	}
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) || strings.TrimSpace(strings.TrimPrefix(header, prefix)) == "" {
		return Principal{}, errors.New("authorization header must be Bearer token")
	}
	return v.verifyJWT(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
}

func (v *authVerifier) verifyJWT(token string) (Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Principal{}, errors.New("token must be a compact JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Principal{}, errors.New("token header is not valid base64url")
	}
	var header struct {
		Alg string `json:"alg"`
		KID string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Principal{}, errors.New("token header is not valid JSON")
	}
	if header.Alg != "RS256" {
		return Principal{}, errors.New("token alg must be RS256")
	}
	if strings.TrimSpace(header.KID) == "" {
		return Principal{}, errors.New("token kid is required")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Principal{}, errors.New("token payload is not valid base64url")
	}
	claims, err := decodeJWTClaims(payloadBytes)
	if err != nil {
		return Principal{}, err
	}
	if claims.Issuer != v.cfg.Issuer {
		return Principal{}, errors.New("token issuer is invalid")
	}
	now := v.now()
	if claims.ExpiresAt <= 0 || now.After(time.Unix(claims.ExpiresAt, 0)) {
		return Principal{}, errors.New("token is expired")
	}
	if claims.NotBefore > 0 && now.Before(time.Unix(claims.NotBefore, 0)) {
		return Principal{}, errors.New("token is not valid yet")
	}
	if !claims.HasAudience(v.cfg.Audience) {
		return Principal{}, errors.New("token audience is invalid")
	}
	key, err := v.key(header.KID)
	if err != nil {
		return Principal{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Principal{}, errors.New("token signature is not valid base64url")
	}
	digest := sha256Sum(parts[0] + "." + parts[1])
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest, signature); err != nil {
		return Principal{}, errors.New("token signature is invalid")
	}
	actor := firstNonEmptyClaim(claims.PreferredUsername, claims.Email, claims.Subject)
	if actor == "" {
		return Principal{}, errors.New("token subject is required")
	}
	return Principal{Subject: claims.Subject, Actor: actor, TenantID: claims.TenantID, Roles: claims.RoleSet(v.cfg.Audience)}, nil
}

func sha256Sum(value string) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(value))
	return h.Sum(nil)
}

type jwtClaims struct {
	Issuer            string
	Subject           string
	PreferredUsername string
	Email             string
	TenantID          string
	Audience          []string
	ExpiresAt         int64
	NotBefore         int64
	RealmRoles        []string
	ResourceRoles     map[string][]string
}

func decodeJWTClaims(payload []byte) (jwtClaims, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return jwtClaims{}, errors.New("token payload is not valid JSON")
	}
	claims := jwtClaims{ResourceRoles: map[string][]string{}}
	_ = json.Unmarshal(raw["iss"], &claims.Issuer)
	_ = json.Unmarshal(raw["sub"], &claims.Subject)
	_ = json.Unmarshal(raw["preferred_username"], &claims.PreferredUsername)
	_ = json.Unmarshal(raw["email"], &claims.Email)
	_ = json.Unmarshal(raw["tenant_id"], &claims.TenantID)
	claims.ExpiresAt = int64Claim(raw["exp"])
	claims.NotBefore = int64Claim(raw["nbf"])
	claims.Audience = audienceClaim(raw["aud"])
	claims.RealmRoles = rolesClaim(raw["realm_access"])
	claims.ResourceRoles = resourceRolesClaim(raw["resource_access"])
	return claims, nil
}

func int64Claim(raw json.RawMessage) int64 {
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0
	}
	return int64(value)
}

func audienceClaim(raw json.RawMessage) []string {
	var single string
	if err := json.Unmarshal(raw, &single); err == nil && single != "" {
		return []string{single}
	}
	var many []string
	_ = json.Unmarshal(raw, &many)
	return many
}

func rolesClaim(raw json.RawMessage) []string {
	var wrapper struct {
		Roles []string `json:"roles"`
	}
	_ = json.Unmarshal(raw, &wrapper)
	return wrapper.Roles
}

func resourceRolesClaim(raw json.RawMessage) map[string][]string {
	var resources map[string]struct {
		Roles []string `json:"roles"`
	}
	_ = json.Unmarshal(raw, &resources)
	result := map[string][]string{}
	for resource, roles := range resources {
		result[resource] = roles.Roles
	}
	return result
}

func (c jwtClaims) HasAudience(audience string) bool {
	if strings.TrimSpace(audience) == "" {
		return true
	}
	for _, item := range c.Audience {
		if item == audience {
			return true
		}
	}
	return false
}

func (c jwtClaims) RoleSet(audience string) map[string]struct{} {
	roles := map[string]struct{}{}
	for _, role := range c.RealmRoles {
		roles[role] = struct{}{}
	}
	for _, role := range c.ResourceRoles[audience] {
		roles[role] = struct{}{}
	}
	return roles
}

func (v *authVerifier) key(kid string) (*rsa.PublicKey, error) {
	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}
	if err := v.refreshKeys(); err != nil {
		return nil, err
	}
	if key := v.cachedKey(kid); key != nil {
		return key, nil
	}
	return nil, errors.New("token signing key not found")
}

func (v *authVerifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.keys == nil {
		return nil
	}
	if ttl := v.cacheTTL(); ttl > 0 && v.now().After(v.fetchedAt.Add(ttl)) {
		return nil
	}
	return v.keys[kid]
}

func (v *authVerifier) cacheTTL() time.Duration {
	if v.cfg.JWKSCacheTTL > 0 {
		return v.cfg.JWKSCacheTTL
	}
	return 5 * time.Minute
}

func (v *authVerifier) refreshKeys() error {
	if strings.TrimSpace(v.cfg.JWKSURL) == "" {
		return errors.New("jwks url is required")
	}
	req, err := http.NewRequest(http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return errors.New("jwks request failed")
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return errors.New("jwks fetch failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("jwks fetch returned non-200")
	}
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return errors.New("jwks response is invalid")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, jwk := range set.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" || jwk.N == "" || jwk.E == "" {
			continue
		}
		key, err := rsaPublicKeyFromJWK(jwk.N, jwk.E)
		if err != nil {
			continue
		}
		keys[jwk.Kid] = key
	}
	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = v.now()
	v.mu.Unlock()
	return nil
}

func rsaPublicKeyFromJWK(nValue string, eValue string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nValue)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eValue)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid rsa exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func firstNonEmptyClaim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
