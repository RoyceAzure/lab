package model

import "github.com/google/uuid"

type BaseCommand struct {
	commandID string
}

func NewBaseCommand() BaseCommand {
	return BaseCommand{
		commandID: uuid.New().String(),
	}
}

func (c *BaseCommand) GetID() string {
	return c.commandID
}

type CommandType string

const (
	OrderCreatedCommandName   CommandType = "OrderCreated"
	OrderConfirmedCommandName CommandType = "OrderConfirmed"
	OrderShippedCommandName   CommandType = "OrderShipped"
	OrderCancelledCommandName CommandType = "OrderCancelled"
	OrderRefundedCommandName  CommandType = "OrderRefunded"
)

type Command interface {
	Type() CommandType
	GetID() string
}
