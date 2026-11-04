CREATE TABLE users (
    id          uuid PRIMARY KEY,
    display_name text NOT NULL DEFAULT '',
    wrapped_key bytea NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE identities (
    provider         text NOT NULL,
    provider_user_id text NOT NULL,
    user_id          uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    email            text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, provider_user_id)
);

CREATE INDEX identities_user_id_idx ON identities (user_id);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE resumes (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text NOT NULL,
    template_id text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX resumes_user_id_idx ON resumes (user_id);

-- Each save appends a revision. The JSON content is encrypted; the latest
-- revision is the current version.
CREATE TABLE resume_revisions (
    id         uuid PRIMARY KEY,
    resume_id  uuid NOT NULL REFERENCES resumes (id) ON DELETE CASCADE,
    ciphertext bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX resume_revisions_resume_id_idx ON resume_revisions (resume_id, created_at);
