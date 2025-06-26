package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/google/uuid"
)

type CartCommandError error

var (
	errCartCommand CartCommandError = errors.New("cart_command_format_error")
)

// 購物車命令處理器
// 使用state based 處理庫存快取 product stock相關操作
// 發送cart event 到kafka，由cart event handler 處理
// topic: 由producer創建時設置
// key: userID 一個userID只會有一個購物車
type cartCommandHandler struct {
	userService    *service.UserService
	productService *service.ProductService
	kafkaProducer  producer.Producer
}

func newCartCommandHandler(userService *service.UserService, productService *service.ProductService, kafkaProducer producer.Producer) *cartCommandHandler {
	return &cartCommandHandler{userService: userService, productService: productService, kafkaProducer: kafkaProducer}
}

// 通用的事件消息準備函數
func prepareEventMessage(userID int, eventType evt_model.EventType, payload interface{}) (message.Message, error) {
	eventBytes, err := json.Marshal(payload)
	if err != nil {
		return message.Message{}, err
	}

	return message.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Value: eventBytes,
		Headers: []message.Header{
			{
				Key:   "event_type",
				Value: []byte(eventType),
			},
		},
	}, nil
}

// 驗證命令
// 強一致性處理庫存狀態，一個商品同時間只有一個thread處理
// 購物車Connand 使用delta 處理庫存，所以需要處理冪等(已經由HandlerDispatcher處理)
// 庫存處理不使用event，使用redis state-based
func (h *cartCommandHandler) HandleCartCreated(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.CartCreatedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.CartCreatedCommand); !ok {
		return errCartCommand
	}

	user, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	var (
		orderItems    []model.OrderItemData
		errsOrderItem []error
	)

	//主要庫存處理事件
	//庫存個別檢查，個別扣除，避免一個商品庫存不足，導致整個購物車失敗
	for _, item := range c.Items {
		// 扣庫存
		err = h.productService.SubProductStock(ctx, item.ProductID, uint(item.Quantity))
		if err != nil {
			errsOrderItem = append(errsOrderItem, err)
			continue
		}

		orderItems = append(orderItems, model.OrderItemData{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	//次要事件發布，有錯誤會記錄，交由後續程序處理
	go h.produceCartEvent(ctx, user.UserID, orderItems)
	go h.produceCartFailedEvent(ctx, user.UserID, errsOrderItem...)

	return nil
}

// 事件發布
// TODO :若是有任何失敗，需要紀錄並後續處理
func (h *cartCommandHandler) produceCartEvent(ctx context.Context, userID int, orderItems []model.OrderItemData) {
	msg, err := prepareCartEventMessage(userID, orderItems)
	if err != nil {
		return
	}

	err = h.kafkaProducer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return
	}
}

// 事件發布
// TODO :若是有任何失敗，需要紀錄並後續處理
func (h *cartCommandHandler) produceCartFailedEvent(ctx context.Context, userID int, errs ...error) {
	if len(errs) == 0 {
		return
	}

	msg, err := prepareCartFailedEventMessage(userID, errs...)
	if err != nil {
		return
	}

	err = h.kafkaProducer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return
	}
}

func prepareCartEventMessage(userID int, orderItems []model.OrderItemData) (message.Message, error) {
	return prepareEventMessage(userID, evt_model.CartCreatedEventName, evt_model.CartCreatedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     uuid.New().String(),
			AggregateID: generateCartAggregateID(userID),
			EventType:   evt_model.CartCreatedEventName,
		},
		UserID: userID,
		Items:  orderItems,
	})
}

func prepareCartFailedEventMessage(userID int, errs ...error) (message.Message, error) {
	return prepareEventMessage(userID, evt_model.CartFailedEventName, evt_model.CartFailedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     uuid.New().String(),
			AggregateID: generateCartAggregateID(userID),
			EventType:   evt_model.CartFailedEventName,
		},
		Message: errors.Join(errs...).Error(),
	})
}

// 驗證命令
// 強一致性處理庫存狀態，一個商品同時間只有一個thread處理
// 購物車Connand 使用delta 處理庫存，所以需要處理冪等(已經由HandlerDispatcher處理)
// 庫存處理不使用event，使用redis state-based
func (h *cartCommandHandler) HandleCartUpdated(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.CartUpdatedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.CartUpdatedCommand); !ok {
		return errCartCommand
	}

	user, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	//主要庫存處理事件，同步處理庫存
	//庫存個別檢查，個別處理，避免一個商品庫存不足，導致整個購物車失敗
	for _, item := range c.Details {
		switch item.Action {
		case cmd_model.CartAddItem:
			// 扣庫存
			err = h.productService.SubProductStock(ctx, item.ProductID, uint(item.Quantity))
		case cmd_model.CartSubItem:
			// 加庫存
			err = h.productService.AddProductStock(ctx, item.ProductID, uint(item.Quantity))
		}
		if err != nil {
			go h.produceCartFailedEvent(ctx, user.UserID, err)
		} else {
			go h.produceCartUpdatedEvent(ctx, user.UserID, []cmd_model.CartUpdatedDetial{item})
		}
	}

	return nil
}

// 更新購物車事件發布
// 發送狀態變更事件
// TODO :若是有任何失敗，需要紀錄並後續處理
func (h *cartCommandHandler) produceCartUpdatedEvent(ctx context.Context, userID int, details []cmd_model.CartUpdatedDetial) {
	msg, err := prepareCartUpdatedEventMessage(userID, details)
	if err != nil {
		return
	}

	err = h.kafkaProducer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return
	}
}

func prepareCartUpdatedEventMessage(userID int, details []cmd_model.CartUpdatedDetial) (message.Message, error) {
	return prepareEventMessage(userID, evt_model.CartUpdatedEventName, evt_model.CartUpdatedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     uuid.New().String(),
			AggregateID: generateCartAggregateID(userID),
			EventType:   evt_model.CartUpdatedEventName,
		},
		UserID:  userID,
		Details: details,
	})
}

// 購物車確認-> 進入訂單狀態後 會刪除購物車
// 或者購物車直接刪除
func (h *cartCommandHandler) HandleCartDeleted(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.CartDeletedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.CartDeletedCommand); !ok {
		return errCartCommand
	}

	user, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	//次要事件發布，有錯誤會記錄，交由後續程序處理
	go h.produceCartDeletedEvent(ctx, user.UserID)

	return nil
}

func (h *cartCommandHandler) produceCartDeletedEvent(ctx context.Context, userID int) {
	msg, err := prepareCartDeletedEventMessage(userID)
	if err != nil {
		return
	}

	err = h.kafkaProducer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return
	}
}

func prepareCartDeletedEventMessage(userID int) (message.Message, error) {
	return prepareEventMessage(userID, evt_model.CartDeletedEventName, evt_model.CartDeletedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     uuid.New().String(),
			AggregateID: generateCartAggregateID(userID),
			EventType:   evt_model.CartDeletedEventName,
		},
		UserID: userID,
	})
}

func generateCartAggregateID(userID int) string {
	return fmt.Sprintf("cart:%d", userID)
}
