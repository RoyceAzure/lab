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

// BeforeDelete GORM 的 hook，在軟刪除前將 IsDeleted 設置為 true
func (b *BaseModel) BeforeDelete(tx *gorm.DB) error {
	if !tx.Statement.Unscoped {
		return tx.Update("is_deleted", true).Error
	}
	return nil
}

// BeforeUpdate GORM 的 hook，在恢復刪除時將 IsDeleted 設置為 false
func (b *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("DeletedAt") {
		if undelete, ok := tx.Statement.Get("gorm:undelete"); ok && undelete.(bool) {
			return tx.Update("is_deleted", false).Error
		}
	}
	return nil
}
