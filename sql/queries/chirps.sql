-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES(
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2
)
RETURNING *;

-- name: GetChirps :many
SELECT * FROM chirps ORDER BY created_at ASC;

-- name: GetChirpsDesc :many
SELECT * FROM chirps ORDER BY created_at DESC;

-- name: GetChirp :one
SELECT * FROM chirps WHERE id = $1;

-- name: GetChirpByAuthor :many
SELECT * FROM chirps WHERE user_id = $1 ORDER BY created_at ASC;

-- name: GetChirpByAuthorDesc :many
SELECT * FROM chirps WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteChirpByID :exec
DELETE FROM chirps WHERE id = $1;

-- name: DeleteChirps :exec
TRUNCATE TABLE chirps CASCADE;
