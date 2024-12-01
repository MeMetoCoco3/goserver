-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at, revoked_at)VALUES(
	$1,
	NOW(),
	NOW(),
	$2,
	$3,
	NULL
)
RETURNING *;

-- name: GetRefreshToken :one 
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: GetUserWithToken :one
SELECT users.* FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1
AND revoked_at IS NULL
AND expires_at > NOW();

-- name: DeleteRefreshToken :one
UPDATE refresh_tokens 
SET revoked_at = NOW() 
WHERE token = $1
RETURNING *;


