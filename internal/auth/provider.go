// Package auth implements OAuth 2.0 sign-in with GitHub and Google.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	ghoauth "golang.org/x/oauth2/github"
)

// Identity is an account as reported by an identity provider.
type Identity struct {
	Provider       string
	ProviderUserID string
	Name           string
	Email          string
}

// Provider is an OAuth identity provider. Logins use PKCE on top of the state parameter.
type Provider struct {
	Slug   string
	Label  string
	config *oauth2.Config
	fetch  func(ctx context.Context, client *http.Client) (Identity, error)
}

// Login is an authorization request. The caller keeps State and Verifier
// (for example in short-lived cookies) and passes them back to Complete.
type Login struct {
	URL      string
	State    string
	Verifier string
}

func NewGitHub(clientID, clientSecret, redirectURL string) *Provider {
	return &Provider{
		Slug:  "github",
		Label: "GitHub",
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"read:user"},
			Endpoint:     ghoauth.Endpoint,
		},
		fetch: fetchGitHub,
	}
}

func NewGoogle(clientID, clientSecret, redirectURL string) *Provider {
	return &Provider{
		Slug:  "google",
		Label: "Google",
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
		fetch: fetchGoogle,
	}
}

// Begin starts a login and returns the URL to redirect the user to.
func (p *Provider) Begin() (Login, error) {
	state, err := randomString()
	if err != nil {
		return Login{}, err
	}
	verifier := oauth2.GenerateVerifier()
	return Login{
		URL:      p.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)),
		State:    state,
		Verifier: verifier,
	}, nil
}

// Complete exchanges an authorization code for the signed-in user's identity.
func (p *Provider) Complete(ctx context.Context, code, verifier string) (Identity, error) {
	tok, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, fmt.Errorf("%s token exchange: %w", p.Slug, err)
	}
	id, err := p.fetch(ctx, p.config.Client(ctx, tok))
	if err != nil {
		return Identity{}, fmt.Errorf("%s profile: %w", p.Slug, err)
	}
	id.Provider = p.Slug
	return id, nil
}

func fetchGitHub(ctx context.Context, client *http.Client) (Identity, error) {
	var u struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &u); err != nil {
		return Identity{}, err
	}
	name := u.Name
	if name == "" {
		name = u.Login
	}
	return Identity{
		ProviderUserID: strconv.FormatInt(u.ID, 10),
		Name:           name,
		Email:          u.Email,
	}, nil
}

func fetchGoogle(ctx context.Context, client *http.Client) (Identity, error) {
	var u struct {
		Sub   string `json:"sub"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &u); err != nil {
		return Identity{}, err
	}
	return Identity{
		ProviderUserID: u.Sub,
		Name:           u.Name,
		Email:          u.Email,
	}, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: %s: %s", url, resp.Status, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func randomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
