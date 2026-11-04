package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/config"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	keys, err := envelope.NewKeyring(cfg.MasterKey)
	if err != nil {
		return err
	}
	svc := app.New(st, keys)

	var providers []*auth.Provider
	if cfg.GitHub.Enabled() {
		providers = append(providers, auth.NewGitHub(cfg.GitHub.ClientID, cfg.GitHub.ClientSecret, cfg.BaseURL+"/auth/github/callback"))
	}
	if cfg.Google.Enabled() {
		providers = append(providers, auth.NewGoogle(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.BaseURL+"/auth/google/callback"))
	}

	srv, err := web.New(web.Options{
		SecureCookies: strings.HasPrefix(cfg.BaseURL, "https://"),
	}, svc, providers)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("listening on %s", cfg.ListenAddr)
	return httpServer.ListenAndServe()
}
