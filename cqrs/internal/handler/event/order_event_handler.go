package handler

import (
	"context"
	"fmt"

	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
)

// 處理order事件
type OrderEventHandler struct {
	orderEventDB *eventdb.EventDao
}

func newOrderEventHandler(orderEventDB *eventdb.EventDao) *OrderEventHandler {
	return &OrderEventHandler{orderEventDB: orderEventDB}
}

func (h *OrderEventHandler) HandleOrderCreated(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderCreatedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderCreatedEvent); !ok {
		return errUnknownEventFormat
	}

	fmt.Println("HandleOrderCreated", e)
	return nil
}

func (h *OrderEventHandler) HandleOrderConfirmed(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.OrderConfirmedEvent
	var ok bool
	if e, ok = evt.(*evt_model.OrderConfirmedEvent); !ok {
		return errUnknownEventFormat
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

	fmt.Println("HandleOrderRefunded", e)
	return nil
}
