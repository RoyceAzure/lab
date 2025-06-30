package model

import "github.com/RoyceAzure/lab/cqrs/internal/domain/model"

const (
	CartCreatedCommandName   CommandType = "CartCreated"
	CartUpdatedCommandName   CommandType = "CartUpdated"
	CartDeletedCommandName   CommandType = "CartDeleted"
	CartConfirmedCommandName CommandType = "CartConfirmed"
)

type CartCreatedCommand struct {
	BaseCommand
	UserID int              `json:"user_id"`
	Items  []model.CartItem `json:"items"`
}

func NewCartCreatedCommand(userID int, items []model.CartItem) *CartCreatedCommand {
	return &CartCreatedCommand{
		BaseCommand: NewBaseCommand(),
		UserID:      userID,
		Items:       items,
	}
}

func (c *CartCreatedCommand) Type() CommandType {
	return CartCreatedCommandName
}

type CartUpdatedCommand struct {
	BaseCommand
	UserID  int                 `json:"user_id"`
	Details []CartUpdatedDetial `json:"details"`
}

func NewCartUpdatedCommand(userID int, details []CartUpdatedDetial) *CartUpdatedCommand {
	return &CartUpdatedCommand{
		BaseCommand: NewBaseCommand(),
		UserID:      userID,
		Details:     details,
	}
}

type CartUpdatedAction int

const (
	CartAddItem CartUpdatedAction = iota
	CartSubItem
)

type CartUpdatedDetial struct {
	Action    CartUpdatedAction `json:"action"`
	ProductID string            `json:"product_id"`
	Quantity  int               `json:"quantity"`
}

func (c *CartUpdatedCommand) Type() CommandType {
	return CartUpdatedCommandName
}

//刪除整個購物車
type CartDeletedCommand struct {
	BaseCommand
	UserID int `json:"user_id"`
}

func NewCartDeletedCommand(userID int) *CartDeletedCommand {
	return &CartDeletedCommand{
		BaseCommand: NewBaseCommand(),
		UserID:      userID,
	}
}

func (c *CartDeletedCommand) Type() CommandType {
	return CartDeletedCommandName
}

type CartConfirmedCommand struct {
	BaseCommand
	UserID int `json:"user_id"`
}

func NewCartConfirmedCommand(userID int) *CartConfirmedCommand {
	return &CartConfirmedCommand{
		BaseCommand: NewBaseCommand(),
		UserID:      userID,
	}
}

func (c *CartConfirmedCommand) Type() CommandType {
	return CartConfirmedCommandName
}
