package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
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

// 驗證命令
// 最終一致性儲存與發布命令
func (h *OrderCommandHandler) HandleOrderCreated(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.OrderCreatedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.OrderCreatedCommand); !ok {
		return errOrderCommand
	}

	_, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	// 驗證商品是否存在
	// 計算訂單總金額
	amount, err := h.orderService.CalculateOrderAmount(ctx, c.Items...)
	if err != nil {
		return err
	}

	orderID := generateOrderID()
	eventID := uuid.New().String()
	orderCreatedEvent := evt_model.OrderCreatedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(orderID),
			EventType:   evt_model.OrderCreatedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		Items:     c.Items,
		Amount:    amount,
		FromState: model.OrderStatusPending,
		ToState:   model.OrderStatusPending,
	}

	err = h.eventDao.AppendEvent(ctx, orderCreatedEvent.AggregateID, string(evt_model.OrderCreatedEventName), orderCreatedEvent)
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

func (h *OrderCommandHandler) HandleOrderConfirmed(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.OrderConfirmedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.OrderConfirmedCommand); !ok {
		return errOrderCommand
	}

	_, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, c.OrderID)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderConfirmedEvent := evt_model.OrderConfirmedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   evt_model.OrderConfirmedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		FromState: model.OrderStatusPending,
		ToState:   model.OrderStatusConfirmed,
	}
	err = h.eventDao.AppendEvent(ctx, orderConfirmedEvent.AggregateID, string(evt_model.OrderConfirmedEventName), orderConfirmedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderShippedCommand(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.OrderShippedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.OrderShippedCommand); !ok {
		return errOrderCommand
	}

	_, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, c.OrderID)
	if err != nil {
		return err
	}

	//驗證TrackingCode
	//驗證Carrier

	eventID := uuid.New().String()
	orderShippedEvent := evt_model.OrderShippedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   evt_model.OrderShippedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		TrackingCode: c.TrackingCode,
		Carrier:      c.Carrier,
		FromState:    model.OrderStatusConfirmed,
		ToState:      model.OrderStatusShipped,
	}

	err = h.eventDao.AppendEvent(ctx, orderShippedEvent.AggregateID, string(evt_model.OrderShippedEventName), orderShippedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderCancelledCommand(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.OrderCancelledCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.OrderCancelledCommand); !ok {
		return errOrderCommand
	}

	_, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, c.OrderID)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderItems, err := h.orderService.TransferOrderItemToOrderItemData(ctx, order.OrderItems...)
	if err != nil {
		return err
	}

	orderCancelledEvent := evt_model.OrderCancelledEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   evt_model.OrderCancelledEventName,
			CreatedAt:   time.Now().UTC(),
		},
		Items:     orderItems,
		Message:   c.Message,
		FromState: model.OrderStatusConfirmed,
		ToState:   model.OrderStatusCancelled,
	}

	err = h.eventDao.AppendEvent(ctx, orderCancelledEvent.AggregateID, string(evt_model.OrderCancelledEventName), orderCancelledEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}

func (h *OrderCommandHandler) OrderRefundedCommand(ctx context.Context, cmd cmd_model.Command) error {
	var c *cmd_model.OrderRefundedCommand
	var ok bool
	if c, ok = cmd.(*cmd_model.OrderRefundedCommand); !ok {
		return errOrderCommand
	}

	_, err := h.userService.GetUser(ctx, c.UserID)
	if err != nil {
		return err
	}

	order, err := h.orderService.GetOrder(ctx, c.OrderID)
	if err != nil {
		return err
	}

	amount, err := h.orderService.CalculateOrderAmountFromEntity(ctx, order.OrderItems...)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	orderRefundedEvent := evt_model.OrderRefundedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     eventID,
			AggregateID: generateOrderAggregateID(order.OrderID),
			EventType:   evt_model.OrderRefundedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		Amount:    amount,
		FromState: model.OrderStatusShipped,
		ToState:   model.OrderStatusRefunded,
	}

	err = h.eventDao.AppendEvent(ctx, orderRefundedEvent.AggregateID, string(evt_model.OrderRefundedEventName), orderRefundedEvent)
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
