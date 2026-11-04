package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the web app and worker settings, read from environment variables.
type Config struct {
	ListenAddr  string
	BaseURL     string
	DatabaseURL string
	MasterKey   []byte
	GitHub      OAuthClient
	Google      OAuthClient

	// Worker settings.
	RenderConcurrency int
	RenderTimeout     time.Duration
	ChromePath        string
}

type OAuthClient struct {
	ClientID     string
	ClientSecret string
}

func (c OAuthClient) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// Load reads configuration from the environment.
//
//	RESUMAKER_LISTEN_ADDR      listen address (default :8080)
//	RESUMAKER_BASE_URL         public URL, used for OAuth redirects (default http://localhost:8080)
//	RESUMAKER_DATABASE_URL     PostgreSQL connection string (required)
//	RESUMAKER_MASTER_KEY       base64-encoded 32-byte key (required)
//	GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET
//	GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET
//
// The worker settings are RESUMAKER_RENDER_CONCURRENCY (default 2),
// RESUMAKER_RENDER_TIMEOUT (default 60s) and RESUMAKER_CHROME_PATH.
// Use RequireProvider in the web app to check that sign-in is configured.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:  getenv("RESUMAKER_LISTEN_ADDR", ":8080"),
		BaseURL:     strings.TrimRight(getenv("RESUMAKER_BASE_URL", "http://localhost:8080"), "/"),
		DatabaseURL: os.Getenv("RESUMAKER_DATABASE_URL"),
		ChromePath:  os.Getenv("RESUMAKER_CHROME_PATH"),
		GitHub: OAuthClient{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		},
		Google: OAuthClient{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("RESUMAKER_DATABASE_URL is required")
	}

	key, err := base64.StdEncoding.DecodeString(os.Getenv("RESUMAKER_MASTER_KEY"))
	if err != nil || len(key) != 32 {
		return Config{}, errors.New("RESUMAKER_MASTER_KEY must be a base64-encoded 32-byte key")
	}
	cfg.MasterKey = key

	if cfg.RenderConcurrency, err = positiveInt("RESUMAKER_RENDER_CONCURRENCY", 2); err != nil {
		return Config{}, err
	}
	if cfg.RenderTimeout, err = positiveDuration("RESUMAKER_RENDER_TIMEOUT", 60*time.Second); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// RequireProvider reports whether at least one OAuth provider is configured.
func (c Config) RequireProvider() error {
	if !c.GitHub.Enabled() && !c.Google.Enabled() {
		return errors.New("configure at least one OAuth provider (GITHUB_CLIENT_ID/SECRET or GOOGLE_CLIENT_ID/SECRET)")
	}
	return nil
}

func positiveInt(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return n, nil
}

func positiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, for example 60s", name)
	}
	return d, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
