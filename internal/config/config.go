package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds the web app settings, read from environment variables.
type Config struct {
	ListenAddr  string
	BaseURL     string
	DatabaseURL string
	MasterKey   []byte
	GitHub      OAuthClient
	Google      OAuthClient
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
// At least one OAuth provider must be configured.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:  getenv("RESUMAKER_LISTEN_ADDR", ":8080"),
		BaseURL:     strings.TrimRight(getenv("RESUMAKER_BASE_URL", "http://localhost:8080"), "/"),
		DatabaseURL: os.Getenv("RESUMAKER_DATABASE_URL"),
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

	if !cfg.GitHub.Enabled() && !cfg.Google.Enabled() {
		return Config{}, fmt.Errorf("configure at least one OAuth provider (GITHUB_CLIENT_ID/SECRET or GOOGLE_CLIENT_ID/SECRET)")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
