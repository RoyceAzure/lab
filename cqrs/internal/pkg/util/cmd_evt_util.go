package util

import (
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func GenerateOrderIDByTimestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()) // 簡化版
}

func GenerateOrderIDByUUID() string {
	return uuid.New().String()
}

func CalculateOrderAmount(orderItems []model.OrderItemData) decimal.Decimal {
	amount := decimal.Zero
	for _, item := range orderItems {
		amount = amount.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Quantity))))
	}
	return amount
}

func CacheOrderToOrderItemData(cartCache *model.Cart) []model.OrderItemData {
	orderItems := make([]model.OrderItemData, len(cartCache.OrderItems))
	for i, item := range cartCache.OrderItems {
		orderItems[i] = model.OrderItemData{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return orderItems
}

func OrderAggregateToOrder(orderAggregate *evt_model.OrderAggregate) *model.Order {
	orderItems := make([]model.OrderItem, len(orderAggregate.OrderItems))
	for i, item := range orderAggregate.OrderItems {
		orderItems[i] = model.OrderItem{
			OrderID:   orderAggregate.OrderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return &model.Order{
		OrderID:    orderAggregate.OrderID,
		UserID:     orderAggregate.UserID,
		OrderItems: orderItems,
		Amount:     orderAggregate.Amount,
		OrderDate:  orderAggregate.OrderDate,
		State:      orderAggregate.State,
	}
}

func OrderItemDataToOrderItem(orderItemData *model.OrderItemData) *model.OrderItem {
	return &model.OrderItem{
		OrderID:   orderItemData.OrderID,
		ProductID: orderItemData.ProductID,
		Quantity:  orderItemData.Quantity,
	}
}
