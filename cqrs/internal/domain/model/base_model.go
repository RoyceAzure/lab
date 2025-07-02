package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	IsDeleted bool           `gorm:"not null;default:false"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
