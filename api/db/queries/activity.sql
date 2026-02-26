-- name: InsertActivity :exec
INSERT INTO activities (user_id, source, type, payload, occurred_at, external_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, source, external_id) DO NOTHING;

-- name: ListActivitiesByUser :many
SELECT id, user_id, source, type, payload, occurred_at, external_id, created_at
FROM activities
WHERE user_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActivitiesByUser :one
SELECT count(*) FROM activities WHERE user_id = $1;

-- name: ListDistinctActivityUsers :many
SELECT DISTINCT user_id FROM activities;
