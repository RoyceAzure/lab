package event

import "time"

type OrderCreatedEvent struct {
	OrderID   uint
	UserID    uint
	Items     []OrderItemData
	CreatedAt time.Time
}

type OrderItemData struct {
	ProductID uint
	Quantity  int
}

type ProductAddedEvent struct {
	OrderID   uint
	ProductID uint
	Quantity  int
}

type ProductUpdatedEvent struct {
	OrderID   uint
	ProductID uint
	Quantity  int
}

type ProductDeletedEvent struct {
	OrderID   uint
	ProductID uint
}
