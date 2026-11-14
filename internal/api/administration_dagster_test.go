package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDagsterAdministrationBootstrapsAndProxiesReadOnlyUI(t *testing.T) {
	fixture := newTestAuthFixture(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Errorf("gateway credentials leaked upstream: authorization=%q cookie=%q", r.Header.Get("Authorization"), r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("dagster-ui"))
	}))
	defer upstream.Close()
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, DagsterReadonlyURL: upstream.URL})
	token := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})

	bootstrap := httptest.NewRecorder()
	req := withBearer(httptest.NewRequest(http.MethodPost, dagsterSessionPath, nil), token)
	router.ServeHTTP(bootstrap, req)
	if bootstrap.Code != http.StatusOK {
		t.Fatalf("bootstrap status=%d body=%s", bootstrap.Code, bootstrap.Body.String())
	}
	cookie := bootstrap.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != dagsterSessionCookie {
		t.Fatalf("unexpected session cookie: %#v", cookie)
	}

	page := httptest.NewRecorder()
	pageReq := httptest.NewRequest(http.MethodGet, dagsterAdminPath, nil)
	pageReq.AddCookie(cookie[0])
	router.ServeHTTP(page, pageReq)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "dagster-ui") {
		t.Fatalf("proxy status=%d body=%s", page.Code, page.Body.String())
	}
}

func TestDagsterAdministrationRejectsNonAdminAndMutations(t *testing.T) {
	fixture := newTestAuthFixture(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()
	router := NewRouter(RouterConfig{Auth: fixture.authCfg, DagsterReadonlyURL: upstream.URL})
	nonAdmin := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{roleViewer}}})
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, withBearer(httptest.NewRequest(http.MethodPost, dagsterSessionPath, nil), nonAdmin))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("non-admin bootstrap status=%d body=%s", denied.Code, denied.Body.String())
	}
	admin := fixture.token(t, map[string]any{"realm_access": map[string]any{"roles": []string{rolePlatformSuperAdmin}}})
	bootstrap := httptest.NewRecorder()
	router.ServeHTTP(bootstrap, withBearer(httptest.NewRequest(http.MethodPost, dagsterSessionPath, nil), admin))
	cookie := bootstrap.Result().Cookies()[0]
	mutation := httptest.NewRecorder()
	mutationReq := httptest.NewRequest(http.MethodDelete, dagsterAdminPath, nil)
	mutationReq.AddCookie(cookie)
	router.ServeHTTP(mutation, mutationReq)
	if mutation.Code != http.StatusMethodNotAllowed {
		t.Fatalf("mutation status=%d body=%s", mutation.Code, mutation.Body.String())
	}
}
