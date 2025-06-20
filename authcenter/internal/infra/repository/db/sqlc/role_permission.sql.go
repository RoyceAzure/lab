// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: role_permission.sql

package sqlc

import (
	"context"
)

const assignPermissionToRole = `-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
`

type AssignPermissionToRoleParams struct {
	RoleID       int32 `json:"role_id"`
	PermissionID int32 `json:"permission_id"`
}

func (q *Queries) AssignPermissionToRole(ctx context.Context, arg AssignPermissionToRoleParams) error {
	_, err := q.db.Exec(ctx, assignPermissionToRole, arg.RoleID, arg.PermissionID)
	return err
}

const createRolePermissionIfNotExists = `-- name: CreateRolePermissionIfNotExists :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT (role_id, permission_id) DO NOTHING
`

type CreateRolePermissionIfNotExistsParams struct {
	RoleID       int32 `json:"role_id"`
	PermissionID int32 `json:"permission_id"`
}

func (q *Queries) CreateRolePermissionIfNotExists(ctx context.Context, arg CreateRolePermissionIfNotExistsParams) error {
	_, err := q.db.Exec(ctx, createRolePermissionIfNotExists, arg.RoleID, arg.PermissionID)
	return err
}

const getRolePermissions = `-- name: GetRolePermissions :many
SELECT p.id, p.name, p.resource, p.actions, p.description FROM permissions p
JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = $1
`

func (q *Queries) GetRolePermissions(ctx context.Context, roleID int32) ([]Permission, error) {
	rows, err := q.db.Query(ctx, getRolePermissions, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Permission{}
	for rows.Next() {
		var i Permission
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Resource,
			&i.Actions,
			&i.Description,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removePermissionFromRole = `-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2
`

type RemovePermissionFromRoleParams struct {
	RoleID       int32 `json:"role_id"`
	PermissionID int32 `json:"permission_id"`
}

func (q *Queries) RemovePermissionFromRole(ctx context.Context, arg RemovePermissionFromRoleParams) error {
	_, err := q.db.Exec(ctx, removePermissionFromRole, arg.RoleID, arg.PermissionID)
	return err
}
