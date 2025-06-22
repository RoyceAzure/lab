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

func (h *OrderEventHandler) HandleOrderCreated(ctx context.Context, evt event.Event) error {
	var e *event.OrderCreatedEvent
	var ok bool
	if e, ok = evt.(*event.OrderCreatedEvent); !ok {
		return errUnknownEvent
	}

	orderItems := []model.OrderItem{}
	for _, item := range e.Items {
		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order := model.Order{
		OrderID:    e.OrderID,
		UserID:     e.UserID,
		State:      e.State,
		Amount:     e.Amount,
		OrderItems: orderItems,
		BaseModel: model.BaseModel{
			CreatedAt: e.CreatedAt,
		},
	}

	_, err := h.orderService.UpdateOrder(ctx, &order)
	if err != nil {
		return err
	}

	return nil
}

func (h *OrderEventHandler) HandleOrderConfirmed(ctx context.Context, evt event.Event) error {
	var e *event.OrderConfirmedEvent
	var ok bool
	if e, ok = evt.(*event.OrderConfirmedEvent); !ok {
		return errUnknownEvent
	}

	order, err := h.orderService.GetOrder(ctx, e.OrderID)
	if err != nil {
		return err
	}

	order.State = e.State
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 物流相關領域事件  需要更新
func (h *OrderEventHandler) HandleOrderShipped(ctx context.Context, evt event.Event) error {
	var e *event.OrderShippedEvent
	var ok bool
	if e, ok = evt.(*event.OrderShippedEvent); !ok {
		return errUnknownEvent
	}

	order, err := h.orderService.GetOrder(ctx, e.OrderID)
	if err != nil {
		return err
	}

	order.State = e.State
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，需要更新庫存 (product 領域事件，需要更新庫存)
func (h *OrderEventHandler) HandleOrderCancelled(ctx context.Context, evt event.Event) error {
	var e *event.OrderCancelledEvent
	var ok bool
	if e, ok = evt.(*event.OrderCancelledEvent); !ok {
		return errUnknownEvent
	}

	order, err := h.orderService.GetOrder(ctx, e.OrderID)
	if err != nil {
		return err
	}

	order.State = e.State
	order.IsDeleted = true
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，金流事件handler 需要負責退退款處理
func (h *OrderEventHandler) HandleOrderRefunded(ctx context.Context, evt event.Event) error {
	var e *event.OrderRefundedEvent
	var ok bool
	if e, ok = evt.(*event.OrderRefundedEvent); !ok {
		return errUnknownEvent
	}

	order, err := h.orderService.GetOrder(ctx, e.OrderID)
	if err != nil {
		return err
	}

	order.State = e.State
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}
