-- name: UpsertDataSource :one
INSERT INTO data_sources (user_id, provider, access_token, refresh_token, expires_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, provider)
DO UPDATE SET access_token = EXCLUDED.access_token,
              refresh_token = EXCLUDED.refresh_token,
              expires_at = EXCLUDED.expires_at
RETURNING id, user_id, provider, created_at;

-- name: GetDataSourceByUserAndProvider :one
SELECT id, user_id, provider, access_token, refresh_token, expires_at, created_at
FROM data_sources
WHERE user_id = $1 AND provider = $2;

-- name: ListDataSourcesByUser :many
SELECT id, user_id, provider, expires_at, created_at
FROM data_sources
WHERE user_id = $1;

-- name: ListDataSourcesByProvider :many
SELECT id, user_id, provider, created_at
FROM data_sources
WHERE provider = $1;
