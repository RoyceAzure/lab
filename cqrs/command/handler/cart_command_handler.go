package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/RoyceAzure/lab/cqrs/command"
	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/RoyceAzure/lab/cqrs/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/google/uuid"
)

type CartCommandError error

var (
	errCartCommand CartCommandError = errors.New("cart_command_format_error")
)

// 購物車事件
// topic: cart
// key: userID 一個userID只會有一個購物車
type CartCommandHandler struct {
	userService    *service.UserService
	productService *service.ProductService
	kafkaProducer  producer.Producer
}

func NewCartCommandHandler(userService *service.UserService, productService *service.ProductService, kafkaProducer producer.Producer) *CartCommandHandler {
	return &CartCommandHandler{userService: userService, productService: productService, kafkaProducer: kafkaProducer}
}

// 驗證命令
// 強一致性處理庫存狀態，一個商品同時間只有一個thread處理
// 購物車Connand 使用delta 處理庫存，所以需要處理冪等(已經由HandlerDispatcher處理)
// 庫存處理不使用event，使用redis state-based
func (h *CartCommandHandler) HandleCartCreated(ctx context.Context, cmd command.Command) error {
	var c *command.CartCreatedCommand
	var ok bool
	if c, ok = cmd.(*command.CartCreatedCommand); !ok {
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
func (h *CartCommandHandler) produceCartEvent(ctx context.Context, userID int, orderItems []model.OrderItemData) {
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
func (h *CartCommandHandler) produceCartFailedEvent(ctx context.Context, userID int, errs ...error) {
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
	eventID := uuid.New().String()
	eventBytes, err := json.Marshal(event.CartCreatedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateCartAggregateID(userID),
			EventType:   event.CartCreatedEventName,
		},
		UserID: userID,
		Items:  orderItems,
	})
	return message.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Topic: "cart",
		Value: eventBytes,
	}, err
}

func prepareCartFailedEventMessage(userID int, errs ...error) (message.Message, error) {
	eventID := uuid.New().String()
	eventBytes, err := json.Marshal(event.CartCommandFailedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateCartAggregateID(userID),
			EventType:   event.CartFailedEventName,
		},
		Message: errors.Join(errs...).Error(),
	})
	return message.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Topic: "cart",
		Value: eventBytes,
	}, err
}

// 驗證命令
// 強一致性處理庫存狀態，一個商品同時間只有一個thread處理
// 購物車Connand 使用delta 處理庫存，所以需要處理冪等(已經由HandlerDispatcher處理)
// 庫存處理不使用event，使用redis state-based
func (h *CartCommandHandler) HandleCartUpdated(ctx context.Context, cmd command.Command) error {
	var c *command.CartUpdatedCommand
	var ok bool
	if c, ok = cmd.(*command.CartUpdatedCommand); !ok {
		return errCartCommand
	}

	user, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	var (
		errsOrderItem []error
	)

	//主要庫存處理事件，同步處理庫存
	//庫存個別檢查，個別處理，避免一個商品庫存不足，導致整個購物車失敗
	for _, item := range c.Details {
		switch item.Action {
		case command.CartUpdatedActionAdd:
			// 扣庫存
			err = h.productService.SubProductStock(ctx, item.ProductID, uint(item.Quantity))
			if err != nil {
				errsOrderItem = append(errsOrderItem, err)
				continue
			}
		case command.CartUpdatedActionSub:
			// 加庫存
			err = h.productService.AddProductStock(ctx, item.ProductID, uint(item.Quantity))
			if err != nil {
				errsOrderItem = append(errsOrderItem, err)
				continue
			}
		}
	}

	//次要事件發布，有錯誤會記錄，交由後續程序處理
	go h.produceCartUpdatedEvent(ctx, user.UserID, c.Details)
	go h.produceCartFailedEvent(ctx, user.UserID, errsOrderItem...)

	return nil
}

// 更新購物車事件發布
// 發送狀態變更事件
// TODO :若是有任何失敗，需要紀錄並後續處理
func (h *CartCommandHandler) produceCartUpdatedEvent(ctx context.Context, userID int, details []command.CartUpdatedDetial) {
	msg, err := prepareCartUpdatedEventMessage(userID, details)
	if err != nil {
		return
	}

	err = h.kafkaProducer.Produce(ctx, []message.Message{msg})
	if err != nil {
		return
	}
}

func prepareCartUpdatedEventMessage(userID int, details []command.CartUpdatedDetial) (message.Message, error) {
	eventID := uuid.New().String()
	eventBytes, err := json.Marshal(event.CartUpdatedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateCartAggregateID(userID),
			EventType:   event.CartUpdatedEventName,
		},
		UserID:  userID,
		Details: details,
	})

	return message.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Topic: "cart",
		Value: eventBytes,
	}, err
}

// 購物車確認-> 進入訂單狀態後 會刪除購物車
// 或者購物車直接刪除
func (h *CartCommandHandler) HandleCartDeleted(ctx context.Context, cmd command.Command) error {
	var c *command.CartDeletedCommand
	var ok bool
	if c, ok = cmd.(*command.CartDeletedCommand); !ok {
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

func (h *CartCommandHandler) produceCartDeletedEvent(ctx context.Context, userID int) {
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
	eventID := uuid.New().String()
	eventBytes, err := json.Marshal(event.CartDeletedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateCartAggregateID(userID),
			EventType:   event.CartDeletedEventName,
		},
		UserID: userID,
	})
	return message.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Topic: "cart",
		Value: eventBytes,
	}, err
}

func generateCartAggregateID(userID int) string {
	return fmt.Sprintf("cart:%d", userID)
}
