// Package app holds the application logic behind the HTTP handlers.
package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/auth"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
)

const sessionTTL = 30 * 24 * time.Hour

type Service struct {
	store *store.Store
	keys  *envelope.Keyring
}

func New(st *store.Store, keys *envelope.Keyring) *Service {
	return &Service{store: st, keys: keys}
}

// SignIn returns the user linked to an identity, creating the account on first sign-in.
func (s *Service) SignIn(ctx context.Context, id auth.Identity) (store.User, error) {
	user, err := s.store.UserByIdentity(ctx, id.Provider, id.ProviderUserID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return store.User{}, err
	}

	dek, err := envelope.NewDataKey()
	if err != nil {
		return store.User{}, err
	}
	user = store.User{ID: uuid.New(), DisplayName: id.Name}
	user.WrappedKey, err = s.keys.WrapDataKey(dek, user.ID[:])
	if err != nil {
		return store.User{}, err
	}
	if err := s.store.CreateUser(ctx, user, id.Provider, id.ProviderUserID, id.Email); err != nil {
		return store.User{}, err
	}
	return user, nil
}

// StartSession creates a session and returns its token. Only a hash of the token is stored.
func (s *Service) StartSession(ctx context.Context, userID uuid.UUID) (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expires := time.Now().Add(sessionTTL)
	if err := s.store.CreateSession(ctx, hashToken(token), userID, expires); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (s *Service) UserForSession(ctx context.Context, token string) (store.User, error) {
	if token == "" {
		return store.User{}, store.ErrNotFound
	}
	return s.store.UserBySession(ctx, hashToken(token))
}

func (s *Service) EndSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, hashToken(token))
}

// dataKey returns the user's plaintext data key, which is only held in memory while needed.
func (s *Service) dataKey(user store.User) ([]byte, error) {
	return s.keys.UnwrapDataKey(user.WrappedKey, user.ID[:])
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
