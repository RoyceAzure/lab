package service

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/cqrs/internal/model"
	"github.com/shopspring/decimal"
)

var (
	ErrProductReservedNotEnough = errors.New("product reserved is not enough")
	ErrCartNotExist             = errors.New("cart is not exist")
	ErrOrderNotExist            = errors.New("order is not exist")
	ErrProductNotFound          = errors.New("product not found")
)

type OrderService struct {
	orderRepo   *db.OrderRepo
	productRepo *db.ProductRepo
	cartRepo    *redis_repo.CartRepo
}

// 購物車階段 只會寫入到redis, 不會寫入到db，所有購物車資料都要去redis取
func NewOrderService(orderRepo *db.OrderRepo, productRepo *db.ProductRepo, cartRepo *redis_repo.CartRepo) *OrderService {
	return &OrderService{orderRepo: orderRepo, productRepo: productRepo, cartRepo: cartRepo}
}

// 檢查商品預留數量是否足夠，於創建Order時使用
func (o *OrderService) IsProductReservedEnough(ctx context.Context, productID string, quantity int) error {
	// 檢查商品預留數量是否足夠
	// 檢查redis 該購物車是否有商品
	// 取出購物車商品數量
	product, err := o.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	if product.Reserved < uint(quantity) {
		return ErrProductReservedNotEnough
	}
	return nil
}

func (o *OrderService) CalculateOrderAmount(ctx context.Context, orderItems ...model.OrderItemData) (decimal.Decimal, error) {
	amount := decimal.NewFromInt(0)
	for _, orderItem := range orderItems {
		product, err := o.productRepo.GetProductByID(ctx, orderItem.ProductID)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if product == nil {
			return decimal.Decimal{}, ErrProductNotFound
		}
		amount = amount.Add(product.Price.Mul(decimal.NewFromInt(int64(orderItem.Quantity))))
	}
	return amount, nil
}

func TransferOrderItemDataToOrderItem(orderItemsDatas ...model.OrderItemData) ([]model.OrderItem, error) {
	orderItems := []model.OrderItem{}
	for _, orderItemData := range orderItemsDatas {
		orderItems = append(orderItems, model.OrderItem{
			OrderID:   orderItemData.OrderID,
			ProductID: orderItemData.ProductID,
			Quantity:  orderItemData.Quantity,
		})
	}
	return orderItems, nil
}

func (o *OrderService) TransferOrderItemToOrderItemData(ctx context.Context, orderItems ...model.OrderItem) ([]model.OrderItemData, error) {
	orderItemsData := []model.OrderItemData{}
	for _, orderItem := range orderItems {
		product, err := o.productRepo.GetProductByID(ctx, orderItem.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, ErrProductNotFound
		}
		orderItemsData = append(orderItemsData, model.OrderItemData{
			OrderID:     orderItem.OrderID,
			ProductID:   orderItem.ProductID,
			Quantity:    orderItem.Quantity,
			Price:       product.Price,
			Amount:      product.Price.Mul(decimal.NewFromInt(int64(orderItem.Quantity))),
			ProductName: product.Name,
		})
	}
	return orderItemsData, nil
}

func (o *OrderService) CalculateOrderAmountFromEntity(ctx context.Context, orderItems ...model.OrderItem) (decimal.Decimal, error) {
	orderItemsData, err := o.TransferOrderItemToOrderItemData(ctx, orderItems...)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return o.CalculateOrderAmount(ctx, orderItemsData...)
}

// 當修改購屋車商品數量時，檢查商品預留數量是否足夠
func (o *OrderService) IsProductStockEnoughForUpdate(ctx context.Context, orderID string, productID string, quantity int) error {

	bais := 10 // 假設原本購屋車A商品數量為10

	product, err := o.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	if product.Reserved+uint(bais) < uint(quantity) {
		return ErrProductReservedNotEnough
	}
	return nil
}

func (o *OrderService) CalculateCartAmount(ctx context.Context, cartItems ...model.CartItem) (decimal.Decimal, error) {
	amount := decimal.NewFromInt(0)
	for _, cartItem := range cartItems {
		product, err := o.productRepo.GetProductByID(ctx, cartItem.ProductID)
		if err != nil {
			return decimal.Decimal{}, err
		}
		amount = amount.Add(product.Price.Mul(decimal.NewFromInt(int64(cartItem.Quantity))))
	}
	return amount, nil
}

func (o *OrderService) GetOrder(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := o.orderRepo.GetOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotExist
	}

	return order, nil
}

func (o *OrderService) UpdateOrder(ctx context.Context, order *model.Order) (*model.Order, error) {
	err := o.orderRepo.UpdateOrder(order)
	if err != nil {
		return nil, err
	}

	return o.GetOrder(ctx, order.OrderID)
}
