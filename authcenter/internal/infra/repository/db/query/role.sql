-- name: CreateRole :one
INSERT INTO roles (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = $1;

-- name: UpdateRole :one
UPDATE roles
SET name = $2, description = $3
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY id ASC LIMIT $1 OFFSET $2;

-- name: CreateRoleIfNotExists :exec
INSERT INTO roles (id, name, description)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO NOTHING;