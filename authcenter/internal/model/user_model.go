package model

import (
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	ID           uuid.UUID
	Email        string
	IsAdmin      bool
	IsActive     bool
	LineUserID   *string
	GoogleID     *string
	FacebookID   *string
	Account      *string
	HashPassword *string
	Name         *string
	LineLinkedAt *time.Time
	CreatedAt    time.Time
}

type CreateUserModel struct {
	GoogleID     string
	Email        string
	Name         string
	IsAdmin      bool
	LineUserID   *string
	LineLinkedAt *time.Time
	CreatedAt    time.Time
}

type CreateUserByAccountAndPasResult struct {
	ResultCode int32
	UserModel  UserModel
}

type CreateUserByAccountAndPasModel struct {
	Email    string
	Account  string
	Password string
}

type UpdateUserModel struct {
	ID           int32
	Email        string
	Name         string
	IsAdmin      bool
	LineUserID   *string
	LineLinkedAt *string
}
