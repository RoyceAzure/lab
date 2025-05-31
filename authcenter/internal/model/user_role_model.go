package model

import "github.com/google/uuid"

// UserRoleModel 是服務層使用的使用者角色關聯模型
type UserRoleModel struct {
	UserID uuid.UUID
	RoleID int32
}
