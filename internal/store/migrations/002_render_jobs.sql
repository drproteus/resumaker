-- Render jobs are claimed by workers with SELECT ... FOR UPDATE SKIP LOCKED.
-- Rendered output is encrypted like the resume content it was built from.
CREATE TABLE render_jobs (
    id          uuid PRIMARY KEY,
    resume_id   uuid NOT NULL REFERENCES resumes (id) ON DELETE CASCADE,
    revision_id uuid NOT NULL REFERENCES resume_revisions (id) ON DELETE CASCADE,
    status      text NOT NULL DEFAULT 'queued',
    error       text NOT NULL DEFAULT '',
    html        bytea,
    pdf         bytea,
    created_at  timestamptz NOT NULL DEFAULT now(),
    started_at  timestamptz,
    finished_at timestamptz,
    CONSTRAINT render_jobs_status_check CHECK (status IN ('queued', 'running', 'succeeded', 'failed'))
);

-- One outstanding job per resume; the worker always renders the revision it was queued for.
CREATE UNIQUE INDEX render_jobs_active_idx ON render_jobs (resume_id)
    WHERE status IN ('queued', 'running');

CREATE INDEX render_jobs_queue_idx ON render_jobs (created_at) WHERE status = 'queued';
CREATE INDEX render_jobs_resume_idx ON render_jobs (resume_id, created_at DESC);
