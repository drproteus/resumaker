package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Resume struct {
	ID         uuid.UUID
	Name       string
	TemplateID string
	UpdatedAt  time.Time
}

// Revision is one saved version of a resume's encrypted content.
type Revision struct {
	ID         uuid.UUID
	Ciphertext []byte
}

func (s *Store) ListResumes(ctx context.Context, userID uuid.UUID) ([]Resume, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, template_id, updated_at FROM resumes WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Resume
	for rows.Next() {
		var r Resume
		if err := rows.Scan(&r.ID, &r.Name, &r.TemplateID, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateResume(ctx context.Context, userID uuid.UUID, r Resume, rev Revision) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO resumes (id, user_id, name, template_id) VALUES ($1, $2, $3, $4)`,
		r.ID, userID, r.Name, r.TemplateID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO resume_revisions (id, resume_id, ciphertext) VALUES ($1, $2, $3)`,
		rev.ID, r.ID, rev.Ciphertext); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetResume returns a resume and its latest revision, scoped to the owner.
func (s *Store) GetResume(ctx context.Context, userID, id uuid.UUID) (Resume, Revision, error) {
	var r Resume
	var rev Revision
	err := s.db.QueryRow(ctx, `
		SELECT r.id, r.name, r.template_id, r.updated_at, rv.id, rv.ciphertext
		FROM resumes r
		JOIN resume_revisions rv ON rv.resume_id = r.id
		WHERE r.id = $1 AND r.user_id = $2
		ORDER BY rv.created_at DESC
		LIMIT 1`,
		id, userID,
	).Scan(&r.ID, &r.Name, &r.TemplateID, &r.UpdatedAt, &rev.ID, &rev.Ciphertext)
	return r, rev, notFound(err)
}

// UpdateResume changes a resume's metadata and appends a revision.
func (s *Store) UpdateResume(ctx context.Context, userID uuid.UUID, r Resume, rev Revision) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE resumes SET name = $3, template_id = $4, updated_at = now() WHERE id = $1 AND user_id = $2`,
		r.ID, userID, r.Name, r.TemplateID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO resume_revisions (id, resume_id, ciphertext) VALUES ($1, $2, $3)`,
		rev.ID, r.ID, rev.Ciphertext); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeleteResume(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM resumes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
