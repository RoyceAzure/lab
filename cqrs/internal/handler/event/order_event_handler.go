package handler

import (
	"context"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/pkg/util"
)

// 處理order事件
type OrderEventHandler struct {
	orderRepo db.IOrderRepository
}

func newOrderEventHandler(orderRepo db.IOrderRepository) *OrderEventHandler {
	return &OrderEventHandler{orderRepo: orderRepo}
}

func (h *OrderEventHandler) HandleOrderCreated(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderCreatedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderCreatedEvent); !ok {
		return errUnknownEventFormat
	}

	order := util.OrderCreatedEventToOrder(e)
	err := h.orderRepo.CreateOrder(ctx, order)
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
	err := h.orderRepo.UpdateOrderState(ctx, e.AggregateID, uint(e.ToState))
	if err != nil {
		return err
	}

	fmt.Println("HandleOrderConfirmed", e)
	return nil
}

// TODO: 物流相關領域事件  需要更新
func (h *OrderEventHandler) HandleOrderShipped(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderShippedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderShippedEvent); !ok {
		return errUnknownEventFormat
	}

	fmt.Println("HandleOrderShipped", e)
	return nil
}

// TODO: 退款後，需要更新庫存 (product 領域事件，需要更新庫存)
func (h *OrderEventHandler) HandleOrderCancelled(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderCancelledEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderCancelledEvent); !ok {
		return errUnknownEventFormat
	}

	err := h.orderRepo.UpdateOrderState(ctx, e.AggregateID, uint(e.ToState))
	if err != nil {
		return err
	}

	fmt.Println("HandleOrderCancelled", e)
	return nil
}

// TODO: 退款後，金流事件handler 需要負責退退款處理
func (h *OrderEventHandler) HandleOrderRefunded(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderRefundedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderRefundedEvent); !ok {
		return errUnknownEventFormat
	}

	err := h.orderRepo.UpdateOrderState(ctx, e.AggregateID, uint(e.ToState))
	if err != nil {
		return err
	}

	fmt.Println("HandleOrderRefunded", e)
	return nil
}
