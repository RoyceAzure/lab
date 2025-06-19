package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	command "github.com/RoyceAzure/lab/cqrs/commnd"
	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/service"
	"github.com/google/uuid"
)

type OrrderCommandError error

var (
	errOrderCommand OrrderCommandError = errors.New("order_command_error")
)

type OrderCommandHandler struct {
	userService  *service.UserService
	eventDao     *eventdb.EventDao
	productRepo  *db.ProductRepo
	orderService *service.OrderService
}

// 只處理Order, 不處理Cart
func NewOrderCommandHandler(orderService *service.OrderService, userService *service.UserService, eventDao *eventdb.EventDao, productRepo *db.ProductRepo) *OrderCommandHandler {
	return &OrderCommandHandler{orderService: orderService, userService: userService, eventDao: eventDao, productRepo: productRepo}
}

func (h *OrderCommandHandler) HandleCommand(ctx context.Context, cmd command.Command) error {
	switch cmd.Type() {
	case command.OrderCreatedCommandName:
		if c, ok := cmd.(*command.OrderCreatedCommand); ok {
			return h.HandleOrderCreated(ctx, c)
		}
	case command.OrderConfirmedCommandName:
		if c, ok := cmd.(*command.OrderConfirmedCommand); ok {
			return h.HandleOrderConfirmed(ctx, c)
		}
	case command.OrderShippedCommandName:
		if c, ok := cmd.(*command.OrderShippedCommand); ok {
			return h.OrderShippedCommandName(ctx, c)
		}
	case command.OrderCancelledCommandName:
		if c, ok := cmd.(*command.OrderCancelledCommand); ok {
			return h.OrderCancelledCommandName(ctx, c)
		}
	case command.OrderRefundedCommandName:
		if c, ok := cmd.(*command.OrderRefundedCommand); ok {
			return h.OrderRefundedCommandName(ctx, c)
		}
	default:
		return errOrderCommand
	}
	return nil
}

// 驗證命令
// 最終一致性儲存與發布命令
func (h *OrderCommandHandler) HandleOrderCreated(ctx context.Context, cmd *command.OrderCreatedCommand) error {
	user, err := h.userService.GetUser(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	// 驗證商品是否存在
	// 計算訂單總金額
	amount, err := h.orderService.CalculateOrderAmount(cmd.Items...)
	if err != nil {
		return err
	}

	orderID := generateOrderID()
	eventID := uuid.New().String()
	orderCreatedEvent := event.OrderCreatedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(orderID),
			EventType:   event.OrderCreatedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID: orderID,
		UserID:  user.UserID,
		Items:   cmd.Items,
		Amount:  amount,
		State:   model.OrderStatusPending,
	}

	err = h.eventDao.AppendEvent(ctx, orderCreatedEvent.AggregateID, string(event.OrderCreatedEventName), orderCreatedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	// err = h.eventDao.PublishEvent(ctx, orderCreatedEvent)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (h *OrderCommandHandler) HandleOrderConfirmed(ctx context.Context, cmd *command.OrderConfirmedCommand) error {
	user, err := h.userService.GetUser(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderConfirmedEvent := event.OrderConfirmedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   event.OrderConfirmedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID: order.OrderID,
		UserID:  user.UserID,
		State:   model.OrderStatusConfirmed,
	}
	err = h.eventDao.AppendEvent(ctx, orderConfirmedEvent.AggregateID, string(event.OrderConfirmedEventName), orderConfirmedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderShippedCommandName(ctx context.Context, cmd *command.OrderShippedCommand) error {
	user, err := h.userService.GetUser(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	//驗證TrackingCode
	//驗證Carrier

	eventID := uuid.New().String()
	orderShippedEvent := event.OrderShippedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   event.OrderShippedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID:      order.OrderID,
		UserID:       user.UserID,
		TrackingCode: cmd.TrackingCode,
		Carrier:      cmd.Carrier,
		State:        model.OrderStatusShipped,
	}

	err = h.eventDao.AppendEvent(ctx, orderShippedEvent.AggregateID, string(event.OrderShippedEventName), orderShippedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderCancelledCommandName(ctx context.Context, cmd *command.OrderCancelledCommand) error {
	user, err := h.userService.GetUser(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderCancelledEvent := event.OrderCancelledEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   event.OrderCancelledEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID: order.OrderID,
		UserID:  user.UserID,
		Message: cmd.Message,
		State:   model.OrderStatusCancelled,
	}

	err = h.eventDao.AppendEvent(ctx, orderCancelledEvent.AggregateID, string(event.OrderCancelledEventName), orderCancelledEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderRefundedCommandName(ctx context.Context, cmd *command.OrderRefundedCommand) error {
	user, err := h.userService.GetUser(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	amount, err := h.orderService.CalculateOrderAmountFromEntity(order.OrderItems...)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderRefundedEvent := event.OrderRefundedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   event.OrderRefundedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID: order.OrderID,
		UserID:  user.UserID,
		Amount:  amount,
		State:   model.OrderStatusRefunded,
	}

	err = h.eventDao.AppendEvent(ctx, orderRefundedEvent.AggregateID, string(event.OrderRefundedEventName), orderRefundedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func generateOrderID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()) // 簡化版
}
func generateOrderAggregateID(orderID string) string {
	return fmt.Sprintf("order-%s", orderID)
}
