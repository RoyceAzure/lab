package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	command "github.com/RoyceAzure/lab/cqrs/commnd"
	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/service"
	"github.com/google/uuid"
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
		return errors.New("unknown command type")
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
	for _, item := range cmd.Items {
		product, err := h.productRepo.GetProductByID(item.ProductID)
		if err != nil {
			return err
		}
		if product == nil {
			return errors.New("product not found")
		}
		//不驗證商品數量，因為商品數量已經在cart階段嚴格處理過
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
	}

	err = h.eventDao.AppendEvent(ctx, orderCreatedEvent.AggregateID, string(event.OrderCreatedEventName), orderCreatedEvent)
	if err != nil {
		return err
	}

	// 發佈事件
	err = h.eventDao.PublishEvent(ctx, orderCreatedEvent)
	if err != nil {
		return err
	}

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
	if order == nil {
		return errors.New("order not found")
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
	}
	err = h.eventDao.AppendEvent(ctx, orderConfirmedEvent.AggregateID, string(event.OrderConfirmedEventName), orderConfirmedEvent)
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
	if order == nil {
		return errors.New("order not found")
	}

	//驗證TrackingCode
	//驗證Carrier

	eventID := uuid.New().String()
	orderShippedEvent := event.OrderShippedEvent{
		BaseEvent: event.BaseEvent{
			EventID:     eventID,
			AggregateID: cmd.OrderID,
			EventType:   event.OrderShippedEventName,
			CreatedAt:   time.Now().UTC(),
		},
		OrderID:      order.OrderID,
		UserID:       user.UserID,
		TrackingCode: cmd.TrackingCode,
		Carrier:      cmd.Carrier,
	}

	err = h.eventDao.AppendEvent(ctx, orderShippedEvent.AggregateID, string(event.OrderShippedEventName), orderShippedEvent)
	return nil
}

func (h *OrderCommandHandler) OrderCancelledCommandName(command *command.OrderConfirmedCommand) error {
	return nil
}

func (h *OrderCommandHandler) OrderRefundedCommandName(command *command.OrderConfirmedCommand) error {
	return nil
}

func generateOrderID() uint {
	return uint(time.Now().UnixNano()) // 簡化版
}
func generateOrderAggregateID(orderID uint) string {
	return fmt.Sprintf("order-%d", orderID)
}
