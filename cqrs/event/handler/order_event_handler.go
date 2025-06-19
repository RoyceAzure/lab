package handler

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/RoyceAzure/lab/cqrs/service"
)

type OrderEventHandlerError error

var (
	errUnknownEvent OrderEventHandlerError = errors.New("unknown event")
)

// 處理order 領域相關事件
type OrderEventHandler struct {
	orderService *service.OrderService
}

func NewOrderEventHandler(orderService *service.OrderService) *OrderEventHandler {
	return &OrderEventHandler{orderService: orderService}
}

// 從kafka接收event
// 接收到的事件皆為冪等
// 更新投影db
// TODO : 優化 repo批次更新
// TODO :snapshot 機制另外更新
func (h *OrderEventHandler) HandleEvent(ctx context.Context, evt event.Event) error {
	switch evt.Type() {
	case event.OrderCreatedEventName:
		// 處理 OrderConfirmedEvent
		if e, ok := evt.(*event.OrderCreatedEvent); ok {
			return h.HandleOrderCreatedEvent(ctx, e)
		}
	case event.OrderConfirmedEventName:
		// 處理 OrderConfirmedEvent
		if e, ok := evt.(*event.OrderConfirmedEvent); ok {
			return h.HandleOrderConfirmedEvent(ctx, e)
		}
	case event.OrderShippedEventName:
		// 處理 OrderShippedEvent
		if e, ok := evt.(*event.OrderShippedEvent); ok {
			return h.HandleOrderShippedEvent(ctx, e)
		}
	case event.OrderCancelledEventName:
		// 處理 OrderCancelledEvent
		if e, ok := evt.(*event.OrderCancelledEvent); ok {
			return h.HandleOrderCancelledEvent(ctx, e)
		}
	case event.OrderRefundedEventName:
		// 處理 OrderRefundedEvent
		if e, ok := evt.(*event.OrderRefundedEvent); ok {
			return h.HandleOrderRefundedEvent(ctx, e)
		}
	default:
		return errUnknownEvent
	}
	return nil
}

func (h *OrderEventHandler) HandleOrderCreatedEvent(ctx context.Context, evt *event.OrderCreatedEvent) error {
	orderItems := []model.OrderItem{}
	for _, item := range evt.Items {
		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order := model.Order{
		OrderID:    evt.OrderID,
		UserID:     evt.UserID,
		State:      evt.State,
		Amount:     evt.Amount,
		OrderItems: orderItems,
		BaseModel: model.BaseModel{
			CreatedAt: evt.CreatedAt,
		},
	}

	_, err := h.orderService.UpdateOrder(ctx, &order)
	if err != nil {
		return err
	}

	return nil
}

func (h *OrderEventHandler) HandleOrderConfirmedEvent(ctx context.Context, evt *event.OrderConfirmedEvent) error {
	order, err := h.orderService.GetOrder(ctx, evt.OrderID)
	if err != nil {
		return err
	}

	order.State = evt.State
	order.UpdatedAt = evt.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 物流相關領域事件  需要更新
func (h *OrderEventHandler) HandleOrderShippedEvent(ctx context.Context, evt *event.OrderShippedEvent) error {
	order, err := h.orderService.GetOrder(ctx, evt.OrderID)
	if err != nil {
		return err
	}

	order.State = evt.State
	order.UpdatedAt = evt.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，需要更新庫存 (product 領域事件，需要更新庫存)
func (h *OrderEventHandler) HandleOrderCancelledEvent(ctx context.Context, evt *event.OrderCancelledEvent) error {
	order, err := h.orderService.GetOrder(ctx, evt.OrderID)
	if err != nil {
		return err
	}

	order.State = evt.State
	order.IsDeleted = true
	order.UpdatedAt = evt.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，金流事件handler 需要負責退退款處理
func (h *OrderEventHandler) HandleOrderRefundedEvent(ctx context.Context, evt *event.OrderRefundedEvent) error {
	order, err := h.orderService.GetOrder(ctx, evt.OrderID)
	if err != nil {
		return err
	}

	order.State = evt.State
	order.UpdatedAt = evt.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}
