-- name: CreateUser :one
INSERT INTO users (email, name, password)
VALUES ($1, $2, $3)
RETURNING id, email, name, avatar_url, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, name, avatar_url, password, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, avatar_url, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $2, avatar_url = $3, updated_at = now()
WHERE id = $1
RETURNING id, email, name, avatar_url, created_at, updated_at;
