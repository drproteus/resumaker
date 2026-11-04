package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	RenderQueued    = "queued"
	RenderRunning   = "running"
	RenderSucceeded = "succeeded"
	RenderFailed    = "failed"
)

type RenderJob struct {
	ID         uuid.UUID
	ResumeID   uuid.UUID
	RevisionID uuid.UUID
	Status     string
	Error      string
	CreatedAt  time.Time
	FinishedAt *time.Time
}

// ClaimedRenderJob includes the owner so the worker can decrypt the content.
type ClaimedRenderJob struct {
	RenderJob
	UserID uuid.UUID
}

// RenderSource is everything needed to render one revision of one resume.
type RenderSource struct {
	User       User
	TemplateID string
	Ciphertext []byte
}

// EnqueueRender queues a render of a revision, or returns the resume's
// outstanding job if one is already queued or running.
func (s *Store) EnqueueRender(ctx context.Context, resumeID, revisionID uuid.UUID) (RenderJob, error) {
	job := RenderJob{
		ID:         uuid.New(),
		ResumeID:   resumeID,
		RevisionID: revisionID,
		Status:     RenderQueued,
	}
	err := s.db.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO render_jobs (id, resume_id, revision_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (resume_id) WHERE status IN ('queued', 'running') DO NOTHING
			RETURNING id, resume_id, revision_id, status, error, created_at, finished_at
		)
		SELECT id, resume_id, revision_id, status, error, created_at, finished_at FROM inserted
		UNION ALL
		SELECT id, resume_id, revision_id, status, error, created_at, finished_at FROM render_jobs
		WHERE resume_id = $2 AND status IN ('queued', 'running')
		LIMIT 1`,
		job.ID, resumeID, revisionID,
	).Scan(&job.ID, &job.ResumeID, &job.RevisionID, &job.Status, &job.Error, &job.CreatedAt, &job.FinishedAt)
	if err != nil {
		return RenderJob{}, err
	}
	return job, nil
}

// ClaimRenderJob atomically takes the oldest queued job. It returns ErrNotFound
// when the queue is empty.
func (s *Store) ClaimRenderJob(ctx context.Context) (ClaimedRenderJob, error) {
	var job ClaimedRenderJob
	err := s.db.QueryRow(ctx, `
		WITH next AS (
			SELECT id FROM render_jobs
			WHERE status = 'queued'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE render_jobs j
		SET status = 'running', started_at = now()
		FROM next
		WHERE j.id = next.id
		RETURNING j.id, j.resume_id, j.revision_id, j.created_at,
			(SELECT r.user_id FROM resumes r WHERE r.id = j.resume_id)`,
	).Scan(&job.ID, &job.ResumeID, &job.RevisionID, &job.CreatedAt, &job.UserID)
	return job, notFound(err)
}

// RenderSource loads the owner, template and encrypted content for a job.
func (s *Store) RenderSource(ctx context.Context, resumeID, revisionID uuid.UUID) (RenderSource, error) {
	var src RenderSource
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.display_name, u.wrapped_key, r.template_id, rv.ciphertext
		FROM resume_revisions rv
		JOIN resumes r ON r.id = rv.resume_id
		JOIN users u ON u.id = r.user_id
		WHERE rv.id = $1 AND r.id = $2`,
		revisionID, resumeID,
	).Scan(&src.User.ID, &src.User.DisplayName, &src.User.WrappedKey, &src.TemplateID, &src.Ciphertext)
	return src, notFound(err)
}

func (s *Store) CompleteRenderJob(ctx context.Context, jobID uuid.UUID, html, pdf []byte) error {
	_, err := s.db.Exec(ctx, `
		UPDATE render_jobs
		SET status = 'succeeded', html = $2, pdf = $3, error = '', finished_at = now()
		WHERE id = $1`,
		jobID, html, pdf)
	return err
}

func (s *Store) FailRenderJob(ctx context.Context, jobID uuid.UUID, message string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE render_jobs
		SET status = 'failed', error = $2, finished_at = now()
		WHERE id = $1`,
		jobID, message)
	return err
}

// RequeueStaleRenderJobs returns jobs left running by a worker that died.
func (s *Store) RequeueStaleRenderJobs(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE render_jobs
		SET status = 'queued', started_at = NULL
		WHERE status = 'running' AND started_at < now() - make_interval(secs => $1)`,
		olderThan.Seconds())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListRenderJobs returns a resume's jobs, newest first, scoped to the owner.
func (s *Store) ListRenderJobs(ctx context.Context, userID, resumeID uuid.UUID, limit int) ([]RenderJob, error) {
	rows, err := s.db.Query(ctx, `
		SELECT j.id, j.resume_id, j.revision_id, j.status, j.error, j.created_at, j.finished_at
		FROM render_jobs j
		JOIN resumes r ON r.id = j.resume_id
		WHERE j.resume_id = $1 AND r.user_id = $2
		ORDER BY j.created_at DESC
		LIMIT $3`,
		resumeID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RenderJob
	for rows.Next() {
		var j RenderJob
		if err := rows.Scan(&j.ID, &j.ResumeID, &j.RevisionID, &j.Status, &j.Error, &j.CreatedAt, &j.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// RenderArtifact returns a finished job and its encrypted output, scoped to the owner.
func (s *Store) RenderArtifact(ctx context.Context, userID, jobID uuid.UUID) (RenderJob, []byte, []byte, error) {
	var j RenderJob
	var html, pdf []byte
	err := s.db.QueryRow(ctx, `
		SELECT j.id, j.resume_id, j.revision_id, j.status, j.error, j.created_at, j.finished_at, j.html, j.pdf
		FROM render_jobs j
		JOIN resumes r ON r.id = j.resume_id
		WHERE j.id = $1 AND r.user_id = $2 AND j.status = 'succeeded'`,
		jobID, userID,
	).Scan(&j.ID, &j.ResumeID, &j.RevisionID, &j.Status, &j.Error, &j.CreatedAt, &j.FinishedAt, &html, &pdf)
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderJob{}, nil, nil, ErrNotFound
	}
	if err != nil {
		return RenderJob{}, nil, nil, err
	}
	return j, html, pdf, nil
}
