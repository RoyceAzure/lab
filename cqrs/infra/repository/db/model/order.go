package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	OrderStatusPending   uint = 0 // 待處理
	OrderStatusConfirmed uint = 1 // 已確認
	OrderStatusShipped   uint = 2 // 已出貨
	OrderStatusCancelled uint = 3 // 已取消
	OrderStatusRefunded  uint = 4 // 已退款
)

type Order struct {
	OrderID    string          `gorm:"primaryKey;type:varchar(255)" json:"order_id"`
	UserID     uint            `gorm:"not null" json:"user_id"`                                           // 外鍵，關聯到 User
	OrderItems []OrderItem     `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order_items"` // 一對多，級聯刪除
	Amount     decimal.Decimal `gorm:"not null;type:decimal(10,2)" json:"amount"`
	OrderDate  time.Time       `gorm:"not null" json:"order_date"`
	State      uint            `gorm:"not null;default:0" json:"state"`
	BaseModel
}

type OrderItem struct {
	OrderID   string `gorm:"primaryKey;type:varchar(255)" json:"order_id"`   // 外鍵，關聯到 Order
	ProductID string `gorm:"primaryKey;type:varchar(255)" json:"product_id"` // 外鍵，關聯到 Product
	Quantity  int    `gorm:"not null" json:"quantity"`
	BaseModel
}

type Cart struct {
	CartID     uuid.UUID       `json:"cart_id"`
	UserID     uint            `json:"user_id"` // 外鍵，關聯到 User
	OrderItems []CartItem      `json:"order_items"`
	Amount     decimal.Decimal `json:"amount"`
}

type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// for command and event
type OrderItemData struct {
	OrderID     string          `json:"order_id"`
	ProductID   string          `json:"product_id"`
	Quantity    int             `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Amount      decimal.Decimal `json:"amount"`
	ProductName string          `json:"product_name"`
}
