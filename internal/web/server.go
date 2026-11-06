// Package web serves the resume app's HTML pages and handles sign-in.
package web

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/store"
)

//go:embed templates static
var assets embed.FS

const (
	sessionCookie = "session"
	maxFormBytes  = 1 << 20
)

type Options struct {
	// SecureCookies sets the Secure flag on cookies. Enable it when serving over HTTPS.
	SecureCookies bool
}

type Server struct {
	app       *app.Service
	providers []*auth.Provider
	pages     map[string]*template.Template
	static    fs.FS
	secure    bool
}

// page is the data passed to every template. User is nil when signed out.
type page struct {
	Title string
	User  *store.User
	Data  any
}

func New(opts Options, svc *app.Service, providers []*auth.Provider) (*Server, error) {
	pages := map[string]*template.Template{}
	for _, name := range []string{"about.html", "login.html", "resumes.html", "resume_form.html"} {
		t, err := template.New("").ParseFS(assets, "templates/layout.html", "templates/"+name)
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	return &Server{
		app:       svc,
		providers: providers,
		pages:     pages,
		static:    static,
		secure:    opts.SecureCookies,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(s.static)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /about", s.handleAbout)
	mux.HandleFunc("GET /login", s.handleLogin)
	mux.HandleFunc("GET /auth/{provider}/login", s.handleAuthStart)
	mux.HandleFunc("GET /auth/{provider}/callback", s.handleAuthCallback)
	mux.HandleFunc("POST /logout", s.handleLogout)

	mux.HandleFunc("GET /resumes", s.requireUser(s.handleResumeList))
	mux.HandleFunc("GET /resumes/new", s.requireUser(s.handleResumeNew))
	mux.HandleFunc("POST /resumes", s.requireUser(s.handleResumeCreate))
	mux.HandleFunc("GET /resumes/{id}", s.requireUser(s.handleResumeEdit))
	mux.HandleFunc("POST /resumes/{id}", s.requireUser(s.handleResumeUpdate))
	mux.HandleFunc("POST /resumes/{id}/duplicate", s.requireUser(s.handleResumeDuplicate))
	mux.HandleFunc("POST /resumes/{id}/tailor", s.requireUser(s.handleResumeTailor))
	mux.HandleFunc("POST /resumes/{id}/delete", s.requireUser(s.handleResumeDelete))
	mux.HandleFunc("GET /resumes/{id}/preview", s.requireUser(s.handlePreview))

	// Rejects cross-site form posts, so no CSRF tokens are needed.
	csrf := http.NewCrossOriginProtection()
	return securityHeaders(csrf.Handler(mux))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

type userHandler func(w http.ResponseWriter, r *http.Request, user store.User)

// requireUser passes the signed-in user to h, or redirects to the login page.
func (s *Server) requireUser(h userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		h(w, r, user)
	}
}

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return store.User{}, false
	}
	user, err := s.app.UserForSession(r.Context(), c.Value)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			log.Printf("session lookup: %v", err)
		}
		return store.User{}, false
	}
	return user, true
}

func (s *Server) provider(slug string) *auth.Provider {
	for _, p := range s.providers {
		if p.Slug == slug {
			return p
		}
	}
	return nil
}

func (s *Server) render(w http.ResponseWriter, status int, name string, p page) {
	var buf bytes.Buffer
	if err := s.pages[name].ExecuteTemplate(&buf, "layout", p); err != nil {
		s.fail(w, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	http.Error(w, "Something went wrong.", http.StatusInternalServerError)
}
