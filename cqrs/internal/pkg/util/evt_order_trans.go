package util

import (
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/google/uuid"
)

func GenerateOrderID() string {
	return uuid.New().String()
}

func OrderCreatedEventToOrder(evt *evt_model.OrderCreatedEvent) *model.Order {
	orderItems := make([]model.OrderItem, len(evt.Items))
	for i, item := range evt.Items {
		orderItems[i] = *OrderItemDataToOrderItem(&item)
	}

	return &model.Order{
		OrderID:    evt.OrderID,
		UserID:     evt.UserID,
		OrderItems: orderItems,
		Amount:     evt.Amount,
		OrderDate:  evt.OrderDate,
		State:      uint(evt.ToState),
	}
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
