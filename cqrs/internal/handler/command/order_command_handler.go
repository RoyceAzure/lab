package handler

import (
	"context"
	"errors"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/cqrs/internal/pkg/util"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
)

type OrrderCommandError error

var (
	errOrderCommand OrrderCommandError = errors.New("order_command_error")
)

type OrderCommandHandler struct {
	userService          *service.UserService
	eventDao             *eventdb.EventDao
	productRepo          redis_repo.IProductRedisRepository
	orderService         *service.OrderService
	toOrderEventProducer producer.Producer
}

// 只處理Order, 不處理Cart
func NewOrderCommandHandler(orderService *service.OrderService, userService *service.UserService, eventDao *eventdb.EventDao, productRepo redis_repo.IProductRedisRepository, toOrderEventProducer producer.Producer) *OrderCommandHandler {
	return &OrderCommandHandler{orderService: orderService, userService: userService, eventDao: eventDao, productRepo: productRepo, toOrderEventProducer: toOrderEventProducer}
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

	orderID := util.GenerateOrderIDByTimestamp()
	orderCreatedEvent := evt_model.NewOrderCreatedEvent(orderID, c.UserID, orderID, time.Now().UTC(), c.Items, amount, model.OrderStatusPending)

	err = h.eventDao.AppendEvent(ctx, orderCreatedEvent.EventID, eventdb.GenerateOrderStreamID(orderID), string(evt_model.OrderCreatedEventName), orderCreatedEvent)
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

	orderConfirmedEvent := evt_model.NewOrderConfirmedEvent(order.OrderID, model.OrderStatusPending, model.OrderStatusConfirmed)

	err = h.eventDao.AppendEvent(ctx, orderConfirmedEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderConfirmedEventName), orderConfirmedEvent)
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

	orderShippedEvent := evt_model.NewOrderShippedEvent(order.OrderID, c.TrackingCode, c.Carrier, model.OrderStatusConfirmed, model.OrderStatusShipped)

	err = h.eventDao.AppendEvent(ctx, orderShippedEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderShippedEventName), orderShippedEvent)
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

	orderCancelledEvent := evt_model.NewOrderCancelledEvent(order.OrderID, c.Message, model.OrderStatusConfirmed, model.OrderStatusCancelled)

	err = h.eventDao.AppendEvent(ctx, orderCancelledEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderCancelledEventName), orderCancelledEvent)
	if err != nil {
		return err
	}

	err = h.eventDao.AppendEvent(ctx, orderCancelledEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderCancelledEventName), orderCancelledEvent)
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

	orderRefundedEvent := evt_model.NewOrderRefundedEvent(order.OrderID, amount, model.OrderStatusShipped, model.OrderStatusRefunded)

	err = h.eventDao.AppendEvent(ctx, orderRefundedEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderRefundedEventName), orderRefundedEvent)
	if err != nil {
		return err
	}

	err = h.eventDao.AppendEvent(ctx, orderRefundedEvent.EventID, eventdb.GenerateOrderStreamID(order.OrderID), string(evt_model.OrderRefundedEventName), orderRefundedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	return nil
}
