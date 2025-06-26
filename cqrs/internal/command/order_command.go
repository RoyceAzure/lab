package command

import (
	"github.com/RoyceAzure/lab/cqrs/internal/model"
)

type OrderCreatedCommand struct {
	BaseCommand
	UserID int                   `json:"user_id"`
	Items  []model.OrderItemData `json:"items"`
}

func (c *OrderCreatedCommand) Type() CommandType {
	return OrderCreatedCommandName
}

type OrderConfirmedCommand struct {
	BaseCommand
	OrderID string `json:"order_id"`
	UserID  int    `json:"user_id"`
}

func (c *OrderConfirmedCommand) Type() CommandType {
	return OrderConfirmedCommandName
}

type OrderShippedCommand struct {
	BaseCommand
	OrderID      string `json:"order_id"`
	UserID       int    `json:"user_id"`
	TrackingCode string `json:"tracking_code"` // 物流追蹤號
	Carrier      string `json:"carrier"`       // 物流商
}

func (c *OrderShippedCommand) Type() CommandType {
	return OrderShippedCommandName
}

type OrderCancelledCommand struct {
	BaseCommand
	OrderID string `json:"order_id"`
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
}

func (c *OrderCancelledCommand) Type() CommandType {
	return OrderCancelledCommandName
}

type OrderRefundedCommand struct {
	BaseCommand
	OrderID string `json:"order_id"`
	UserID  int    `json:"user_id"`
}

func (c *OrderRefundedCommand) Type() CommandType {
	return OrderRefundedCommandName
}
