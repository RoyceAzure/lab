-- name: CreateSession :one
INSERT INTO user_sessions (
    id, user_id, access_token, refresh_token, ip_address, 
    device_info, region, user_agent, last_activity_at, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, $9
) RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM user_sessions WHERE id = $1 LIMIT 1;

-- name: GetSessionByUserID :one
SELECT * FROM user_sessions WHERE user_id = $1 LIMIT 1;

-- name: GetSessionByAccessToken :one
SELECT * FROM user_sessions WHERE access_token = $1 AND is_active = true LIMIT 1;

-- name: GetSessionByRefreshToken :one
SELECT * FROM user_sessions WHERE refresh_token = $1 AND is_active = true LIMIT 1;


-- name: GetSessionByRequestInfo :one
SELECT * FROM user_sessions 
WHERE 
    user_id = $1  
    AND  ip_address = $2
    AND  device_info = $3
    AND
    (sqlc.narg('region')::text IS NULL OR region = sqlc.narg('region')::text)
    AND
    (sqlc.narg('user_agent')::text IS NULL OR user_agent = sqlc.narg('user_agent')::text)
    AND is_active = true
LIMIT 1;

-- name: ListActiveSessionsByUserID :many
SELECT * FROM user_sessions 
WHERE user_id = $1 AND is_active = true 
ORDER BY created_at DESC;

-- name: UpdateSessionActivity :exec
UPDATE user_sessions 
SET last_activity_at = CURRENT_TIMESTAMP 
WHERE id = $1;

-- name: UpdateSessionTokens :one
UPDATE user_sessions 
SET access_token = $1, refresh_token = $2, last_activity_at = CURRENT_TIMESTAMP 
WHERE id = $3 
RETURNING *;

-- name: RevokeSession :exec
UPDATE user_sessions 
SET is_active = false, revoked_at = CURRENT_TIMESTAMP 
WHERE id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE user_sessions 
SET is_active = false, revoked_at = CURRENT_TIMESTAMP 
WHERE user_id = $1 AND id != $2;

-- name: DeleteExpiredSessions :exec
DELETE FROM user_sessions 
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: DeleteSession :exec
DELETE FROM user_sessions WHERE id = $1;

-- name: DeleteSessionByUserID :exec
DELETE FROM user_sessions WHERE user_id = $1;

-- name: ForceClearAllSessions :exec
TRUNCATE TABLE user_sessions;