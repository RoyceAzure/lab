package handler

import (
	"context"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/command"
	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/redis_repo"
)

// 處理cart 領域相關事件
// state based，直接修改redis購物車資料
type CartEventHandler struct {
	cartRepo *redis_repo.CartRepo
}

func NewCartEventHandler(cartRepo *redis_repo.CartRepo) *CartEventHandler {
	return &CartEventHandler{cartRepo: cartRepo}
}

// 處理CartCreatedEvent
// 寫入redis cart 資料
func (h *CartEventHandler) HandleCartCreated(ctx context.Context, evt event.Event) error {
	var e *event.CartCreatedEvent
	var ok bool
	if e, ok = evt.(*event.CartCreatedEvent); !ok {
		return errUnknownEventFormat
	}

	orderItems := []model.CartItem{}
	for _, item := range e.Items {
		orderItems = append(orderItems, model.CartItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	cart := model.Cart{
		UserID:     e.UserID,
		OrderItems: orderItems,
	}

	err := h.cartRepo.Create(ctx, &cart)
	if err != nil {
		return err
	}

	return nil
}

func (h *CartEventHandler) HandleCartFailed(ctx context.Context, evt event.Event) error {
	var e *event.CartFailedEvent
	var ok bool
	if e, ok = evt.(*event.CartFailedEvent); !ok {
		return errUnknownEventFormat
	}

	//先簡單印出
	// TODO: 後續要發紀錄並發送訊息給前端
	fmt.Println("CartFailedEvent", e.UserID, e.Message)

	return nil
}

func (h *CartEventHandler) HandleCartUpdated(ctx context.Context, evt event.Event) error {
	var e *event.CartUpdatedEvent
	var ok bool
	if e, ok = evt.(*event.CartUpdatedEvent); !ok {
		return errUnknownEventFormat
	}

	for _, detail := range e.Details {
		var quantity int
		switch detail.Action {
		case command.CartUpdatedActionAdd:
			quantity = detail.Quantity
		case command.CartUpdatedActionSub:
			quantity = -detail.Quantity
		}
		err := h.cartRepo.Add(ctx, e.UserID, detail.ProductID, quantity)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *CartEventHandler) HandleCartDeleted(ctx context.Context, evt event.Event) error {
	var e *event.CartDeletedEvent
	var ok bool
	if e, ok = evt.(*event.CartDeletedEvent); !ok {
		return errUnknownEventFormat
	}

	err := h.cartRepo.Clear(ctx, e.UserID)
	if err != nil {
		return err
	}

	return nil
}
