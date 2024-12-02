-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES(
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2
)
	RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserWithID :one
SELECT * FROM users WHERE id = $1;


-- name: SetNewPassword :exec
UPDATE users SET hashed_password = $1 WHERE users.id = $2;

-- name: SetNewEmail :exec
UPDATE users SET email = $1 WHERE users.id = $2;

