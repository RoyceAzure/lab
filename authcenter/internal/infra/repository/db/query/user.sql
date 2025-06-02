-- name: CreateUser :one
INSERT INTO users (id, google_id, facebook_id, email, account, password_hash, name, line_user_id, line_linked_at, created_at,is_admin, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;


-- name: GetUserByAccount :one
SELECT * FROM users WHERE account = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByAccountAndPassword :one
SELECT * FROM users WHERE account = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $2, name = $3, line_user_id = $4, line_linked_at = $5, is_admin=$6
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;


-- name: ActiveUser :exec
UPDATE users
SET is_active = TRUE 
WHERE id = $1;

-- name: DeActiveUser :exec
UPDATE users
SET is_active = FALSE 
WHERE id = $1;

-- name: DeActiveUserAdmin :exec
UPDATE users
SET is_admin = FALSE 
WHERE id = $1;

-- name: ActiveUserAdmin :exec
UPDATE users
SET is_admin = TRUE 
WHERE id = $1;


-- name: CreateUserIfNotExists :exec
INSERT INTO users (id, google_id, facebook_id, email, account, password_hash, name, line_user_id, line_linked_at, created_at,is_admin, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (email) DO NOTHING;


-- name: UpdateUserAccountAndPassword :exec
UPDATE users
SET account = $2, password_hash = $3
WHERE email = $1;