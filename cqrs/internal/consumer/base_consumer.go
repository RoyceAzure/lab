package consumer

import (
	"context"
	"errors"
	"log"

	"github.com/RoyceAzure/lab/cqrs/internal/command"
	command_handler "github.com/RoyceAzure/lab/cqrs/internal/command/handler"
	"github.com/RoyceAzure/lab/cqrs/internal/event"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/event/handler"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
)

type ConsumerError error

var (
	ErrConsumerClosed      = errors.New("consumer closed")
	ErrCommandTypeNotFound = errors.New("command type not found")
)

// ConsumeData to event.Event interface
type EventDataAdapter struct {
	ConsumeData
}

func newEventDataAdapter(data ConsumeData) *EventDataAdapter {
	return &EventDataAdapter{data}
}

func (e *EventDataAdapter) Type() event.EventType {
	return event.EventType(e.ConsumeData.Type())
}

func (e *EventDataAdapter) GetID() string {
	return e.ConsumeData.GetID()
}

// event.Event to ConsumeData interface
type EventToConsumeDataAdapter struct {
	event.Event
}

func newEventToConsumeDataAdapter(evt event.Event) *EventToConsumeDataAdapter {
	return &EventToConsumeDataAdapter{evt}
}

func (e *EventToConsumeDataAdapter) Type() string {
	return string(e.Event.Type())
}

func (e *EventToConsumeDataAdapter) GetID() string {
	return e.Event.GetID()
}

// event_handler.Handler to ConsumerHandler interface
type eventHandlerAdapter struct {
	originalHandler event_handler.Handler
}

func newEventHandlerAdapter(handler event_handler.Handler) *eventHandlerAdapter {
	return &eventHandlerAdapter{handler}
}
func (a *eventHandlerAdapter) Handle(ctx context.Context, data ConsumeData) error {
	return a.originalHandler.HandleEvent(ctx, newEventDataAdapter(data))
}

// command.Command to ConsumeData interface
type CommandToConsumeDataAdapter struct {
	command.Command
}

func newCommandToConsumeDataAdapter(cmd command.Command) *CommandToConsumeDataAdapter {
	return &CommandToConsumeDataAdapter{cmd}
}

func (c *CommandToConsumeDataAdapter) Type() string {
	return string(c.Command.Type())
}

func (c *CommandToConsumeDataAdapter) GetID() string {
	return c.Command.GetID()
}

// ConsumeData to command.Command interface
type ConsumeDataToCommandAdapter struct {
	ConsumeData
}

func newConsumeDataToCommandAdapter(data ConsumeData) *ConsumeDataToCommandAdapter {
	return &ConsumeDataToCommandAdapter{data}
}

func (c *ConsumeDataToCommandAdapter) Type() command.CommandType {
	return command.CommandType(c.ConsumeData.Type())
}

func (c *ConsumeDataToCommandAdapter) GetID() string {
	return c.ConsumeData.GetID()
}

// command_handler.Handler to ConsumerHandler interface
type commandHandlerAdapter struct {
	originalHandler command_handler.Handler
}

func newCommandHandlerAdapter(handler command_handler.Handler) *commandHandlerAdapter {
	return &commandHandlerAdapter{handler}
}

func (a *commandHandlerAdapter) Handle(ctx context.Context, data ConsumeData) error {
	return a.originalHandler.HandleCommand(ctx, newConsumeDataToCommandAdapter(data))
}

type IBaseConsumer interface {
	Start(ctx context.Context) error
	Stop()
}

type baseConsumer struct {
	consumer  consumer.Consumer
	handler   ConsumerHandler
	closeChan chan struct{}
}

type ConsumeData interface {
	Type() string
	GetID() string
}

type ConsumerHandler interface {
	Handle(ctx context.Context, data ConsumeData) error
	transformData(msg message.Message) (ConsumeData, error)
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

	close(c.closeChan)
	c.consumer.Close()
}
