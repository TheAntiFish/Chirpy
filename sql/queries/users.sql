-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_uuid(),
    time.now(),
    time.now(),
    $1
)
RETURNING *;