package web

import (
	"crypto/subtle"
	"net/http"
	"time"
)

const (
	oauthStateCookie    = "oauth_state"
	oauthVerifierCookie = "oauth_verifier"
	oauthCookiePath     = "/auth/"
	oauthCookieTTL      = 10 * time.Minute
)

func (s *Server) handleAuthStart(w http.ResponseWriter, r *http.Request) {
	p := s.provider(r.PathValue("provider"))
	if p == nil {
		http.NotFound(w, r)
		return
	}
	login, err := p.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.setCookie(w, oauthStateCookie, login.State, oauthCookiePath, oauthCookieTTL)
	s.setCookie(w, oauthVerifierCookie, login.Verifier, oauthCookiePath, oauthCookieTTL)
	http.Redirect(w, r, login.URL, http.StatusSeeOther)
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	p := s.provider(r.PathValue("provider"))
	if p == nil {
		http.NotFound(w, r)
		return
	}
	// The login cookies are single-use, whatever the outcome below.
	state, stateErr := r.Cookie(oauthStateCookie)
	verifier, verifierErr := r.Cookie(oauthVerifierCookie)
	s.clearCookie(w, oauthStateCookie, oauthCookiePath)
	s.clearCookie(w, oauthVerifierCookie, oauthCookiePath)

	// The user cancelled at the provider, or the login session expired or doesn't match.
	if r.URL.Query().Get("error") != "" || stateErr != nil || verifierErr != nil ||
		subtle.ConstantTimeCompare([]byte(state.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ident, err := p.Complete(r.Context(), r.URL.Query().Get("code"), verifier.Value)
	if err != nil {
		s.fail(w, err)
		return
	}
	user, err := s.app.SignIn(r.Context(), ident)
	if err != nil {
		s.fail(w, err)
		return
	}
	token, expires, err := s.app.StartSession(r.Context(), user.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/resumes", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := s.app.EndSession(r.Context(), c.Value); err != nil {
			s.fail(w, err)
			return
		}
	}
	s.clearCookie(w, sessionCookie, "/")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) setCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}
