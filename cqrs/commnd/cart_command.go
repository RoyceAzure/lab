package command

import "github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"

const (
	CartCreatedCommandName CommandType = "CartCreated"
	CartUpdatedCommandName CommandType = "CartUpdated"
	CartDeletedCommandName CommandType = "CartDeleted"
)

type CartCreatedCommand struct {
	BaseCommand
	UserID uint             `json:"user_id"`
	Items  []model.CartItem `json:"items"`
}

func (c *CartCreatedCommand) Type() CommandType {
	return CartCreatedCommandName
}

// 直接替換購物車Items狀態
type CartUpdatedCommand struct {
	BaseCommand
	UserID uint             `json:"user_id"`
	Items  []model.CartItem `json:"items"`
}

func (c *CartUpdatedCommand) Type() CommandType {
	return CartUpdatedCommandName
}

//刪除整個購物車
type CartDeletedCommand struct {
	BaseCommand
	UserID uint `json:"user_id"`
}

func (c *CartDeletedCommand) Type() CommandType {
	return CartDeletedCommandName
}
