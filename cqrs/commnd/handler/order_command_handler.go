package handler

import (
	command "github.com/RoyceAzure/lab/cqrs/commnd"
	"github.com/RoyceAzure/lab/cqrs/event"
	"github.com/RoyceAzure/lab/cqrs/service"
)

type OrderCommandHandler struct {
	orderService *service.OrderService
}

// 只有當訂單確認，庫存數量才會減少
func NewOrderCommandHandler(orderService *service.OrderService) *OrderCommandHandler {
	return &OrderCommandHandler{orderService: orderService}
}

func (h *OrderCommandHandler) HandleAddProduct(command *command.AddProductCommand) error {
	err := h.orderService.IsProductReservedEnough(command.ProductID, command.Quantity)
	if err != nil {
		return err
	}

	_ = event.ProductAddedEvent{
		ProductID: command.ProductID,
		Quantity:  command.Quantity,
	}

	//發布事件
	return nil
}

// func (h *OrderCommandHandler) HandleUpdateProduct(command *command.UpdateProductCommand) error {
// 	//要先檢查redis 該購物車是否有商品
// 	cart, err := h.orderService.GetCacheCart(command.Ctx, command.UserID)
// 	if err != nil {
// 		return err
// 	}
// 	if cart == nil {
// 		return errors.New("購物車不存在")
// 	}

// 	_ = event.ProductUpdatedEvent{
// 		ProductID: command.ProductID,
// 		Quantity:  command.Quantity,
// 	}

// 	return nil
// }
