-- name: CreateEmailVertify :one
INSERT INTO email_vertify (id, email, is_valid, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetEmailVertify :one
SELECT * FROM email_vertify WHERE id = $1;

-- name: GetEmailVertifyByEmail :many
SELECT * FROM email_vertify WHERE email = $1;

-- name: DeactivateEmailVertifyByID :exec
UPDATE email_vertify 
SET is_valid = FALSE 
WHERE id = $1 
AND is_valid = TRUE;

-- name: UpdateEmailVertify :one
UPDATE email_vertify SET is_valid = $2 WHERE id = $1 RETURNING *;

-- name: DeleteEmailVertify :exec
DELETE FROM email_vertify WHERE id = $1;

