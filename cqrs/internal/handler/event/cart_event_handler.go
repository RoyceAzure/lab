package handler

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
)

// 處理cart 領域相關事件
// state based，直接修改redis購物車資料
type cartEventHandler struct {
	cartRepo *redis_repo.CartRepo
}

func newCartEventHandler(cartRepo *redis_repo.CartRepo) *cartEventHandler {
	return &cartEventHandler{cartRepo: cartRepo}
}

// 處理CartCreatedEvent
// 寫入redis cart 資料
func (h *cartEventHandler) HandleCartCreated(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.CartCreatedEvent
	var ok bool
	if e, ok = evt.(*evt_model.CartCreatedEvent); !ok {
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

func (h *cartEventHandler) HandleCartFailed(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.CartFailedEvent
	var ok bool
	if e, ok = evt.(*evt_model.CartFailedEvent); !ok {
		return errUnknownEventFormat
	}

	//先簡單印出
	// TODO: 後續要發紀錄並發送訊息給前端
	_ = e
	// fmt.Println("CartFailedEvent", e.UserID, e.Message)

	return nil
}

func (h *cartEventHandler) HandleCartUpdated(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.CartUpdatedEvent
	var ok bool
	if e, ok = evt.(*evt_model.CartUpdatedEvent); !ok {
		return errUnknownEventFormat
	}

	for _, detail := range e.Details {
		var quantity int
		switch detail.Action {
		case cmd_model.CartAddItem:
			quantity = detail.Quantity
		case cmd_model.CartSubItem:
			quantity = -detail.Quantity
		}
		err := h.cartRepo.Delta(ctx, e.UserID, detail.ProductID, quantity)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *cartEventHandler) HandleCartDeleted(ctx context.Context, evt evt_model.Event) error {
	var e *evt_model.CartDeletedEvent
	var ok bool
	if e, ok = evt.(*evt_model.CartDeletedEvent); !ok {
		return errUnknownEventFormat
	}

	err := h.cartRepo.Clear(ctx, e.UserID)
	if err != nil {
		return err
	}

	return nil
}
