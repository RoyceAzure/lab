package util

import (
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
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

func CartItemToOrderItem(cartCache *model.Cart) []model.OrderItem {
	orderItems := make([]model.OrderItem, len(cartCache.OrderItems))
	for i, item := range cartCache.OrderItems {
		orderItems[i] = model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return orderItems
}

func OrderItemDataToCartItem(orderItems []model.OrderItemData) []model.CartItem {
	cartItems := make([]model.CartItem, len(orderItems))
	for i, item := range orderItems {
		cartItems[i] = model.CartItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return cartItems
}
