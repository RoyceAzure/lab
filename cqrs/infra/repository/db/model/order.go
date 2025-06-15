package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	OrderID    uint            `gorm:"primaryKey"`
	UserID     uint            `gorm:"not null"`                                       // 外鍵，關聯到 User
	OrderItems []OrderItem     `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"` // 一對多，級聯刪除
	Amount     decimal.Decimal `gorm:"not null;type:decimal(10,2)"`
	OrderDate  time.Time       `gorm:"not null"`
	BaseModel
}

type OrderItem struct {
	OrderID   uint `gorm:"primaryKey"` // 外鍵，關聯到 Order
	ProductID uint `gorm:"primaryKey"` // 外鍵，關聯到 Product
	Quantity  int  `gorm:"not null"`
	BaseModel
}
