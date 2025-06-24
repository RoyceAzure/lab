package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache"
)

type HandlerError error

var (
	errHandlerNotFound HandlerError = errors.New("handler not found")
)

type HandlerFunc func(ctx context.Context, cmd command.Command) error

func (f HandlerFunc) HandleCommand(ctx context.Context, cmd command.Command) error {
	return f(ctx, cmd)
}

type Handler interface {
	HandleCommand(ctx context.Context, cmd command.Command) error
}

type HandlerDispatcher struct {
	handlers     map[command.CommandType]Handler
	commandCache redis_cache.Cache
}

func NewHandlerDispatcher(handlers map[command.CommandType]Handler, commandCache redis_cache.Cache) *HandlerDispatcher {
	return &HandlerDispatcher{handlers: handlers, commandCache: commandCache}
}

func (d *HandlerDispatcher) HandleCommand(ctx context.Context, cmd command.Command) error {
	// 檢查命令是否已經處理過
	commandKey := fmt.Sprintf("%s:%s", cmd.Type(), cmd.GetID())
	_, err := d.commandCache.Get(ctx, commandKey)
	if err != nil {
		return err
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
		handlers: map[command.CommandType]Handler{
			command.OrderCreatedCommandName:   HandlerFunc(orderHandler.HandleOrderCreated),
			command.OrderConfirmedCommandName: HandlerFunc(orderHandler.HandleOrderConfirmed),
			command.OrderShippedCommandName:   HandlerFunc(orderHandler.OrderShippedCommand),
			command.OrderCancelledCommandName: HandlerFunc(orderHandler.OrderCancelledCommand),
			command.OrderRefundedCommandName:  HandlerFunc(orderHandler.OrderRefundedCommand),
		},
	}
}

func NewCartCommandHandler(userService *service.UserService, productService *service.ProductService, kafkaProducer producer.Producer) Handler {
	cartHandler := newCartCommandHandler(userService, productService, kafkaProducer)
	return &HandlerDispatcher{
		handlers: map[command.CommandType]Handler{
			command.CartCreatedCommandName: HandlerFunc(cartHandler.HandleCartCreated),
			command.CartUpdatedCommandName: HandlerFunc(cartHandler.HandleCartUpdated),
			command.CartDeletedCommandName: HandlerFunc(cartHandler.HandleCartDeleted),
		},
	}
}
