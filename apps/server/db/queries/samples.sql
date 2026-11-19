-- name: CreateSample :one
INSERT INTO samples (id, title)
VALUES ($1, $2)
RETURNING id, title, created_at;

-- name: GetSampleByID :one
SELECT id, title, created_at
FROM samples
WHERE id = $1;

-- name: ListSamples :many
SELECT id, title, created_at
FROM samples
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
