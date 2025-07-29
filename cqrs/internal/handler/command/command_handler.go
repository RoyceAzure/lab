package handler

import (
	"context"
	"errors"
	"fmt"

	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/redis/go-redis/v9"
)

type HandlerError error

var (
	errHandlerNotFound HandlerError = errors.New("handler not found")
)

type HandlerFunc func(ctx context.Context, cmd cmd_model.Command) error

func (f HandlerFunc) HandleCommand(ctx context.Context, cmd cmd_model.Command) error {
	return f(ctx, cmd)
}

type Handler interface {
	HandleCommand(ctx context.Context, cmd cmd_model.Command) error
}

type HandlerDispatcher struct {
	handlers     map[cmd_model.CommandType]Handler
	commandCache *redis.Client
}

func NewHandlerDispatcher(handlers map[cmd_model.CommandType]Handler, commandCache *redis.Client) *HandlerDispatcher {
	return &HandlerDispatcher{handlers: handlers, commandCache: commandCache}
}

func (d *HandlerDispatcher) HandleCommand(ctx context.Context, cmd cmd_model.Command) error {
	// 檢查命令是否已經處理過
	if d.commandCache != nil {
		commandKey := fmt.Sprintf("%s:%s", cmd.Type(), cmd.GetID())
		_, err := d.commandCache.Get(ctx, commandKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}
	}

	handler, ok := d.handlers[cmd.Type()]
	if !ok {
		return errHandlerNotFound
	}
	return handler.HandleCommand(ctx, cmd)
}

// 訂單命令處理器
func NewOrderHandler(orderHandler *OrderCommandHandler) Handler {
	return &HandlerDispatcher{
		handlers: map[cmd_model.CommandType]Handler{
			cmd_model.OrderCreatedCommandName:   HandlerFunc(orderHandler.HandleOrderCreated),
			cmd_model.OrderConfirmedCommandName: HandlerFunc(orderHandler.HandleOrderConfirmed),
			cmd_model.OrderShippedCommandName:   HandlerFunc(orderHandler.OrderShippedCommand),
			cmd_model.OrderCancelledCommandName: HandlerFunc(orderHandler.OrderCancelledCommand),
			cmd_model.OrderRefundedCommandName:  HandlerFunc(orderHandler.OrderRefundedCommand),
		},
	}
}

func NewCartCommandHandler(userService *service.UserService, productService service.IProductService, orderService service.IOrderService, cartRepo *redis_repo.CartRepo, userOrderRepo *db.UserOrderRepo, eventDB *eventdb.EventDao, kafkaProducer producer.Producer, orderProducer producer.Producer) Handler {
	cartHandler := newCartCommandHandler(userService, productService, orderService, cartRepo, userOrderRepo, eventDB, kafkaProducer, orderProducer)
	return &HandlerDispatcher{
		handlers: map[cmd_model.CommandType]Handler{
			cmd_model.CartCreatedCommandName:   HandlerFunc(cartHandler.HandleCartCreated),
			cmd_model.CartUpdatedCommandName:   HandlerFunc(cartHandler.HandleCartUpdated),
			cmd_model.CartDeletedCommandName:   HandlerFunc(cartHandler.HandleCartDeleted),
			cmd_model.CartConfirmedCommandName: HandlerFunc(cartHandler.HandleCartConfirmed),
		},
	}
}
