package command

import "github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"

const (
	CartCreatedCommandName CommandType = "CartCreated"
	CartUpdatedCommandName CommandType = "CartUpdated"
	CartDeletedCommandName CommandType = "CartDeleted"
)

type CartCreatedCommand struct {
	BaseCommand
	UserID int              `json:"user_id"`
	Items  []model.CartItem `json:"items"`
}

func (c *CartCreatedCommand) Type() CommandType {
	return CartCreatedCommandName
}

type CartUpdatedCommand struct {
	BaseCommand
	UserID  int                 `json:"user_id"`
	Details []CartUpdatedDetial `json:"details"`
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

func (c *CartDeletedCommand) Type() CommandType {
	return CartDeletedCommandName
}
