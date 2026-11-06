package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrAmbiguousEmail reports that an email address is linked to more than one account.
var ErrAmbiguousEmail = errors.New("email matches more than one account")

// HasFeature reports whether a feature is enabled for a user.
func (s *Store) HasFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	var enabled bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_features WHERE user_id = $1 AND feature = $2)`,
		userID, feature).Scan(&enabled)
	return enabled, err
}

// SetFeature enables or disables a feature for a user. It returns ErrNotFound if
// the user doesn't exist.
func (s *Store) SetFeature(ctx context.Context, userID uuid.UUID, feature string, enabled bool) error {
	if !enabled {
		_, err := s.db.Exec(ctx, `DELETE FROM user_features WHERE user_id = $1 AND feature = $2`, userID, feature)
		return err
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO user_features (user_id, feature)
		SELECT id, $2 FROM users WHERE id = $1
		ON CONFLICT DO NOTHING`, userID, feature)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

// ListFeatures returns the features enabled for a user, sorted by name.
func (s *Store) ListFeatures(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT feature FROM user_features WHERE user_id = $1 ORDER BY feature`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UserAccount is a user with the emails linked to their sign-in accounts.
type UserAccount struct {
	User
	Emails []string
}

// ListUserAccounts returns every user with their linked emails, oldest first.
func (s *Store) ListUserAccounts(ctx context.Context) ([]UserAccount, error) {
	rows, err := s.db.Query(ctx, `
		SELECT u.id, u.display_name, u.wrapped_key, i.email
		FROM users u
		LEFT JOIN identities i ON i.user_id = u.id
		ORDER BY u.created_at, u.id, i.email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserAccount
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var u User
		var email *string
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.WrappedKey, &email); err != nil {
			return nil, err
		}
		i, ok := index[u.ID]
		if !ok {
			i = len(out)
			index[u.ID] = i
			out = append(out, UserAccount{User: u})
		}
		if email != nil && *email != "" {
			out[i].Emails = append(out[i].Emails, *email)
		}
	}
	return out, rows.Err()
}

// UserByEmail returns the single user linked to an email address. It returns
// ErrNotFound if none match and ErrAmbiguousEmail if several do.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT u.id, u.display_name, u.wrapped_key
		FROM identities i
		JOIN users u ON u.id = i.user_id
		WHERE lower(i.email) = lower($1)`, email)
	if err != nil {
		return User{}, err
	}
	defer rows.Close()

	var found []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.WrappedKey); err != nil {
			return User{}, err
		}
		found = append(found, u)
	}
	if err := rows.Err(); err != nil {
		return User{}, err
	}
	switch len(found) {
	case 0:
		return User{}, ErrNotFound
	case 1:
		return found[0], nil
	default:
		return User{}, fmt.Errorf("%w: %s", ErrAmbiguousEmail, email)
	}
}

// UserByID returns a user by ID, or ErrNotFound.
func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.db.QueryRow(ctx,
		`SELECT id, display_name, wrapped_key FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.DisplayName, &u.WrappedKey)
	return u, notFound(err)
}
