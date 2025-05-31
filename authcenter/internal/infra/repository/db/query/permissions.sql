-- name: CreatePermission :one
INSERT INTO permissions (id, name, description, resource, actions)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPermissionByID :one
SELECT * FROM permissions WHERE id = $1;

-- name: GetPermissionByName :one
SELECT * FROM permissions WHERE name = $1;

-- name: UpdatePermission :one
UPDATE permissions
SET name = $2, description = $3, resource = $4, actions=$5
WHERE id = $1
RETURNING *;

-- name: DeletePermission :exec
DELETE FROM permissions WHERE id = $1;

-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY id ASC LIMIT $1 OFFSET $2;


-- name: CreatePermissionIfNotExists :exec
INSERT INTO permissions (id, name, description, resource, actions)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (name) DO NOTHING;