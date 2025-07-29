package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/shopspring/decimal"
)

var (
	ErrProductReservedNotEnough = errors.New("product reserved is not enough")
	ErrCartNotExist             = errors.New("cart is not exist")
	ErrOrderNotExist            = errors.New("order is not exist")
	ErrProductNotFound          = errors.New("product not found")
)

type IOrderService interface {
	CalculateOrderAmount(ctx context.Context, orderItems ...model.OrderItemData) (decimal.Decimal, error)
	SetUpOrderItemDataFromRedis(ctx context.Context, orderItems ...model.OrderItemData) ([]model.OrderItemData, error)
	TransferOrderItemToOrderItemData(ctx context.Context, orderItems ...model.OrderItem) ([]model.OrderItemData, error)
	CalculateOrderAmountFromEntity(ctx context.Context, orderItems ...model.OrderItem) (decimal.Decimal, error)
	CalculateCartAmount(ctx context.Context, cartItems ...model.CartItem) (decimal.Decimal, error)
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrder(ctx context.Context, orderID string) (*model.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]model.Order, error)
	GetAllOrders(ctx context.Context) ([]model.Order, error)
	UpdateOrder(ctx context.Context, order *model.Order) (*model.Order, error)
	UpdateOrderState(ctx context.Context, orderID string, state uint) error
	UpdateOrderAmount(ctx context.Context, orderID string, amount float64) error
	HardDeleteOrder(ctx context.Context, orderID string) error
}

type OrderService struct {
	orderRepo   db.IOrderRepository
	productRepo db.IProductRepository
}

// 購物車階段 只會寫入到redis, 不會寫入到db，所有購物車資料都要去redis取
func NewOrderService(orderRepo db.IOrderRepository, productRepo db.IProductRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo, productRepo: productRepo}
}

// 檢查商品預留數量是否足夠，於創建Order時使用
func (o *OrderService) IsProductReservedEnough(ctx context.Context, productID string, quantity int) error {
	// 檢查商品預留數量是否足夠
	// 檢查redis 該購物車是否有商品
	// 取出購物車商品數量
	product, err := o.productRepo.GetProductStock(ctx, productID)
	if err != nil {
		return err
	}

	if product < quantity {
		return ErrProductReservedNotEnough
	}
	return nil
}

/*
計算訂單總金額
*/
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

/*
還原product資訊
Redis購物車order item只儲存productID, quantity
*/
func (o *OrderService) SetUpOrderItemDataFromRedis(ctx context.Context, orderItems ...model.OrderItemData) ([]model.OrderItemData, error) {
	orderItemsWithProductInfo := make([]model.OrderItemData, 0)
	for _, item := range orderItems {
		product, err := o.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("get product info failed: %w", err)
		}
		orderItemsWithProductInfo = append(orderItemsWithProductInfo, model.OrderItemData{
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: product.Name,
			Price:       product.Price,
			Amount:      product.Price.Mul(decimal.NewFromInt(int64(item.Quantity))),
			Quantity:    item.Quantity,
		})
	}
	return orderItemsWithProductInfo, nil
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

func (o *OrderService) CreateOrder(ctx context.Context, order *model.Order) error {
	return o.orderRepo.CreateOrder(ctx, order)
}

func (o *OrderService) GetOrder(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := o.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotExist
	}

	return order, nil
}

func (o *OrderService) GetOrdersByUserID(ctx context.Context, userID int) ([]model.Order, error) {
	return o.orderRepo.GetOrdersByUserID(ctx, userID)
}

func (o *OrderService) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	return o.orderRepo.GetAllOrders(ctx)
}

func (o *OrderService) UpdateOrder(ctx context.Context, order *model.Order) (*model.Order, error) {
	err := o.orderRepo.UpdateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	return o.GetOrder(ctx, order.OrderID)
}

/*
state:

	OrderStatusPending   uint = 0 // 待處理
	OrderStatusConfirmed uint = 1 // 已確認
	OrderStatusShipped   uint = 2 // 已出貨
	OrderStatusCancelled uint = 3 // 已取消
	OrderStatusRefunded  uint = 4 // 已退款
*/
func (o *OrderService) UpdateOrderState(ctx context.Context, orderID string, state uint) error {
	return o.orderRepo.UpdateOrderState(ctx, orderID, state)
}

func (o *OrderService) UpdateOrderAmount(ctx context.Context, orderID string, amount float64) error {
	return o.orderRepo.UpdateOrderAmount(ctx, orderID, amount)
}

func (o *OrderService) HardDeleteOrder(ctx context.Context, orderID string) error {
	return o.orderRepo.HardDeleteOrder(ctx, orderID)
}

var _ IOrderService = (*OrderService)(nil)
