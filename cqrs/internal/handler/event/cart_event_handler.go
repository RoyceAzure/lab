package handler

import (
	"context"

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
	//目前沒有其他事件需要處理
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
	// log.Info().Msgf("CartFailedEvent: %d, %s", e.UserID, e.Message)

	return nil
}

func (h *cartEventHandler) HandleCartUpdated(ctx context.Context, evt evt_model.Event) error {
	//目前沒有其他事件需要處理
	return nil
}

func (h *cartEventHandler) HandleCartDeleted(ctx context.Context, evt evt_model.Event) error {
	//目前沒有其他事件需要處理
	return nil
}
