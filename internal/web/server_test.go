package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"earthmanlabs/resumaker/internal/auth"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	providers := []*auth.Provider{
		auth.NewGitHub("client-id", "client-secret", "http://localhost/auth/github/callback"),
	}
	srv, err := New(Options{}, nil, providers)
	if err != nil {
		t.Fatal(err)
	}
	return srv.Handler()
}

func TestResumesRequireSignIn(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/resumes", nil))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestLoginPageListsProviders(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Continue with GitHub") {
		t.Error("login page does not list GitHub")
	}
}

func TestAboutPageListsTemplates(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/about", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Resume styles",
		"Classic",
		"Modern",
		"Sidebar",
		"Comic",
		"Dingbats",
		"Privacy",
		`href="/about"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("about page missing %q", want)
		}
	}
}

func TestAuthStartSetsPKCEAndStateCookies(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github/login", nil))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	for _, want := range []string{"https://github.com/login/oauth/authorize", "code_challenge_method=S256", "state="} {
		if !strings.Contains(loc, want) {
			t.Errorf("redirect %q missing %q", loc, want)
		}
	}
	cookies := map[string]bool{}
	for _, c := range rec.Result().Cookies() {
		cookies[c.Name] = c.HttpOnly
	}
	for _, name := range []string{oauthStateCookie, oauthVerifierCookie} {
		if httpOnly, ok := cookies[name]; !ok || !httpOnly {
			t.Errorf("cookie %s missing or not HttpOnly", name)
		}
	}
}

func TestCallbackWithoutLoginCookiesRedirectsToLogin(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github/callback?code=abc&state=xyz", nil))

	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestCrossSiteFormPostIsRejected(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))

	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("unexpected CSP %q", csp)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
}
