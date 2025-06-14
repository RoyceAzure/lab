// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: user_role.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const assignRoleToUser = `-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
`

type AssignRoleToUserParams struct {
	UserID pgtype.UUID `json:"user_id"`
	RoleID int32       `json:"role_id"`
}

func (q *Queries) AssignRoleToUser(ctx context.Context, arg AssignRoleToUserParams) error {
	_, err := q.db.Exec(ctx, assignRoleToUser, arg.UserID, arg.RoleID)
	return err
}

const createUserRoleIfNotExists = `-- name: CreateUserRoleIfNotExists :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT (user_id, role_id) DO NOTHING
`

type CreateUserRoleIfNotExistsParams struct {
	UserID pgtype.UUID `json:"user_id"`
	RoleID int32       `json:"role_id"`
}

func (q *Queries) CreateUserRoleIfNotExists(ctx context.Context, arg CreateUserRoleIfNotExistsParams) error {
	_, err := q.db.Exec(ctx, createUserRoleIfNotExists, arg.UserID, arg.RoleID)
	return err
}

const getUserRoles = `-- name: GetUserRoles :many
SELECT r.id, r.name, r.description FROM roles r
JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = $1
`

func (q *Queries) GetUserRoles(ctx context.Context, userID pgtype.UUID) ([]Role, error) {
	rows, err := q.db.Query(ctx, getUserRoles, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Role{}
	for rows.Next() {
		var i Role
		if err := rows.Scan(&i.ID, &i.Name, &i.Description); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeRoleFromUser = `-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2
`

type RemoveRoleFromUserParams struct {
	UserID pgtype.UUID `json:"user_id"`
	RoleID int32       `json:"role_id"`
}

func (q *Queries) RemoveRoleFromUser(ctx context.Context, arg RemoveRoleFromUserParams) error {
	_, err := q.db.Exec(ctx, removeRoleFromUser, arg.UserID, arg.RoleID)
	return err
}
