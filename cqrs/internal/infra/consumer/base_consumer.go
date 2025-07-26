package consumer

import (
	"context"
	"errors"
	"log"
	"sync"

	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	command_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/command"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/event"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type ConsumerError error

var (
	ErrConsumerClosed       = errors.New("consumer closed")
	ErrCommandTypeNotFound  = errors.New("command type not found")
	ErrUnknownEventFormat   = errors.New("unknown event format")
	ErrUnknownCommandFormat = errors.New("unknown command format")
)

// command and event handler adapter
type handlerAdapter struct {
	originalCMDHandler command_handler.Handler
	originalEVTHandler event_handler.Handler
}

func newHandlerAdapter(cmdHandler command_handler.Handler, evtHandler event_handler.Handler) *handlerAdapter {
	return &handlerAdapter{cmdHandler, evtHandler}
}

func (a *handlerAdapter) Handle(ctx context.Context, data consumeData) error {
	if data.IsCommand() {
		return a.originalCMDHandler.HandleCommand(ctx, data.Command())
	}
	return a.originalEVTHandler.HandleEvent(ctx, data.Event())
}

// consumeData is a union of command and event
type consumeData struct {
	command cmd_model.Command
	event   evt_model.Event
}

func (d consumeData) IsCommand() bool {
	return d.command != nil
}

func (d consumeData) IsEvent() bool {
	return d.event != nil
}

func (d consumeData) Command() cmd_model.Command {
	return d.command
}

func (d consumeData) Event() evt_model.Event {
	return d.event
}

type IBaseConsumer interface {
	Start(ctx context.Context) error
	Stop()
}

type baseConsumer struct {
	consumer  consumer.Consumer
	handler   ConsumerHandler
	closeOnce sync.Once
	closeChan chan struct{}
}

type ConsumerHandler interface {
	Handle(ctx context.Context, data consumeData) error
	transformData(msg message.Message) (consumeData, error)
}

func newBaseConsumer(consumer consumer.Consumer, consumerHandler ConsumerHandler) *baseConsumer {
	return &baseConsumer{consumer: consumer, handler: consumerHandler, closeChan: make(chan struct{})}
}

func (c *baseConsumer) checkIsClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *baseConsumer) Start(ctx context.Context) error {
	if c.checkIsClosed() {
		return ErrConsumerClosed
	}

	msgChan, errChan, err := c.consumer.Consume()

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-c.closeChan:
				return
			case msg := <-msgChan:
				data, err := c.handler.transformData(msg)
				if err != nil {
					log.Println("error", err)
					continue
				}

				err = c.handler.Handle(ctx, data)
				if err != nil {
					log.Println("error", err)
					continue
				}
			case err := <-errChan:
				log.Println("error", err)
			}
		}
	}()

	return nil
}

func (c *baseConsumer) Stop() {
	if c.checkIsClosed() {
		return
	}

	c.closeOnce.Do(func() {
		close(c.closeChan)
	})

	c.consumer.Close()
}
