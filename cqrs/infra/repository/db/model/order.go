package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Order struct {
	OrderID    uint            `gorm:"primaryKey" json:"order_id"`
	UserID     uint            `gorm:"not null" json:"user_id"`                                           // 外鍵，關聯到 User
	OrderItems []OrderItem     `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order_items"` // 一對多，級聯刪除
	Amount     decimal.Decimal `gorm:"not null;type:decimal(10,2)" json:"amount"`
	OrderDate  time.Time       `gorm:"not null" json:"order_date"`
	BaseModel
}

type OrderItem struct {
	OrderID   uint `gorm:"primaryKey" json:"order_id"`   // 外鍵，關聯到 Order
	ProductID uint `gorm:"primaryKey" json:"product_id"` // 外鍵，關聯到 Product
	Quantity  int  `gorm:"not null" json:"quantity"`
	BaseModel
}

type Cart struct {
	CartID     uuid.UUID       `json:"cart_id"`
	UserID     uint            `json:"user_id"` // 外鍵，關聯到 User
	OrderItems []CartItem      `json:"order_items"`
	Amount     decimal.Decimal `json:"amount"`
}

type CartItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}
