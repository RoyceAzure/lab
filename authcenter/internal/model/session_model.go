package model

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AccessToken    string
	RefreshToken   string
	IPAddress      netip.Addr
	DeviceInfo     string
	Region         *string
	UserAgent      *string
	IsActive       bool
	LastActivityAt time.Time
	CreatedAt      time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
}
