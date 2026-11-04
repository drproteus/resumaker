package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	DisplayName string
	WrappedKey  []byte
}

// UserByIdentity returns the user linked to an external account.
func (s *Store) UserByIdentity(ctx context.Context, provider, providerUserID string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.display_name, u.wrapped_key
		FROM identities i
		JOIN users u ON u.id = i.user_id
		WHERE i.provider = $1 AND i.provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&u.ID, &u.DisplayName, &u.WrappedKey)
	return u, notFound(err)
}

// CreateUser inserts a user and links it to an external account in one transaction.
func (s *Store) CreateUser(ctx context.Context, u User, provider, providerUserID, email string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO users (id, display_name, wrapped_key) VALUES ($1, $2, $3)`,
		u.ID, u.DisplayName, u.WrappedKey); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO identities (provider, provider_user_id, user_id, email) VALUES ($1, $2, $3, $4)`,
		provider, providerUserID, u.ID, email); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	return err
}

// UserBySession returns the user for an unexpired session.
func (s *Store) UserBySession(ctx context.Context, tokenHash []byte) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.display_name, u.wrapped_key
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`,
		tokenHash,
	).Scan(&u.ID, &u.DisplayName, &u.WrappedKey)
	return u, notFound(err)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
