-- A row means the feature is enabled for the user. Features are switched on and
-- off with the admin command, so a missing row means the feature is off.
CREATE TABLE user_features (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feature    text NOT NULL,
    enabled_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, feature)
);
