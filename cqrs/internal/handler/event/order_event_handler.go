package handler

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
)

// 處理order事件
// 儲存到eventdb  需要跟infra order eventdb
type OrderEventHandler struct {
	orderService *service.OrderService
}

func NewOrderEventHandler(orderService *service.OrderService) *OrderEventHandler {
	return &OrderEventHandler{orderService: orderService}
}

func (h *OrderEventHandler) HandleOrderCreated(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderCreatedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderCreatedEvent); !ok {
		return errUnknownEventFormat
	}

	orderItems := []model.OrderItem{}
	for _, item := range e.Items {
		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order := model.Order{
		OrderID:    e.AggregateID,
		State:      e.ToState,
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

func (h *OrderEventHandler) HandleOrderConfirmed(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderConfirmedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderConfirmedEvent); !ok {
		return errUnknownEventFormat
	}

	order, err := h.orderService.GetOrder(ctx, e.AggregateID)
	if err != nil {
		return err
	}

	order.State = e.ToState
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 物流相關領域事件  需要更新
func (h *OrderEventHandler) HandleOrderShipped(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderShippedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderShippedEvent); !ok {
		return errUnknownEventFormat
	}

	order, err := h.orderService.GetOrder(ctx, e.AggregateID)
	if err != nil {
		return err
	}

	order.State = e.ToState
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，需要更新庫存 (product 領域事件，需要更新庫存)
func (h *OrderEventHandler) HandleOrderCancelled(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderCancelledEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderCancelledEvent); !ok {
		return errUnknownEventFormat
	}

	order, err := h.orderService.GetOrder(ctx, e.AggregateID)
	if err != nil {
		return err
	}

	order.State = e.ToState
	order.IsDeleted = true
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 退款後，金流事件handler 需要負責退退款處理
func (h *OrderEventHandler) HandleOrderRefunded(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderRefundedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderRefundedEvent); !ok {
		return errUnknownEventFormat
	}

	order, err := h.orderService.GetOrder(ctx, e.AggregateID)
	if err != nil {
		return err
	}

	order.State = e.ToState
	order.UpdatedAt = e.CreatedAt

	_, err = h.orderService.UpdateOrder(ctx, order)
	if err != nil {
		return err
	}

	return nil
}
