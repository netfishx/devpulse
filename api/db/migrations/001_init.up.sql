-- users
CREATE TABLE users (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email       text UNIQUE NOT NULL,
    name        text NOT NULL,
    avatar_url  text,
    password    text NOT NULL,
    created_at  timestamptz DEFAULT now(),
    updated_at  timestamptz DEFAULT now()
);

-- data_sources: user-linked OAuth tokens
CREATE TABLE data_sources (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id       bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      text NOT NULL,
    access_token  bytea NOT NULL,
    refresh_token bytea,
    expires_at    timestamptz,
    created_at    timestamptz DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_data_sources_user_id ON data_sources (user_id);

-- activities: raw events from external platforms
CREATE TABLE activities (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source      text NOT NULL,
    type        text NOT NULL,
    payload     jsonb,
    occurred_at timestamptz NOT NULL,
    created_at  timestamptz DEFAULT now()
);

CREATE INDEX idx_activities_user_occurred ON activities (user_id, occurred_at DESC);
CREATE INDEX idx_activities_payload ON activities USING gin (payload jsonb_path_ops);

-- daily_summaries: aggregated snapshots
CREATE TABLE daily_summaries (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date            date NOT NULL,
    total_commits   int DEFAULT 0,
    total_prs       int DEFAULT 0,
    coding_minutes  int DEFAULT 0,
    top_repos       jsonb,
    top_languages   jsonb,
    UNIQUE (user_id, date)
);

CREATE INDEX idx_daily_summaries_user_id ON daily_summaries (user_id);
