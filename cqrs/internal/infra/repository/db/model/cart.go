package model

import (
	"github.com/shopspring/decimal"
)

type Cart struct {
	UserID     int        `json:"user_id"` // 外鍵，關聯到 User
	OrderItems []CartItem `json:"order_items"`
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
