package service

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/google/uuid"
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
}

// 購物車階段 只會寫入到redis, 不會寫入到db，所有購物車資料都要去redis取
func NewOrderService(orderRepo *db.OrderRepo, productRepo *db.ProductRepo) *OrderService {
	return &OrderService{orderRepo: orderRepo, productRepo: productRepo}
}

// 檢查商品預留數量是否足夠，於創建Order時使用
func (o *OrderService) IsProductReservedEnough(productID string, quantity int) error {
	// 檢查商品預留數量是否足夠
	// 檢查redis 該購物車是否有商品
	// 取出購物車商品數量
	product, err := o.productRepo.GetProductByID(productID)
	if err != nil {
		return err
	}

	if product.Reserved < uint(quantity) {
		return ErrProductReservedNotEnough
	}
	return nil
}

func (o *OrderService) CalculateOrderAmount(orderItems ...model.OrderItemData) (decimal.Decimal, error) {
	amount := decimal.NewFromInt(0)
	for _, orderItem := range orderItems {
		product, err := o.productRepo.GetProductByID(orderItem.ProductID)
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

func (o *OrderService) CalculateOrderAmountFromEntity(orderItems ...model.OrderItem) (decimal.Decimal, error) {
	amount := decimal.NewFromInt(0)
	for _, orderItem := range orderItems {
		product, err := o.productRepo.GetProductByID(orderItem.ProductID)
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

// 當修改購屋車商品數量時，檢查商品預留數量是否足夠
func (o *OrderService) IsProductStockEnoughForUpdate(orderID string, productID string, quantity int) error {

	bais := 10 // 假設原本購屋車A商品數量為10

	product, err := o.productRepo.GetProductByID(productID)
	if err != nil {
		return err
	}

	if product.Reserved+uint(bais) < uint(quantity) {
		return ErrProductReservedNotEnough
	}
	return nil
}

func (o *OrderService) CalculateCartAmount(cartItems ...model.CartItem) (decimal.Decimal, error) {
	amount := decimal.NewFromInt(0)
	for _, cartItem := range cartItems {
		product, err := o.productRepo.GetProductByID(cartItem.ProductID)
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

//購物車相關，使用全量替換方式

// 創建購物車
// 參數:
//
//	ctx: 上下文
//	userID(uint): 用戶ID
//
// 返回值:
//
//	cartId (uuid.UUID): 購物車ID
//
// 錯誤:
func (o *OrderService) CreateCacheCart(ctx context.Context, userID uint, cartItems ...model.CartItem) (uuid.UUID, error) {
	//驗證使用者是否存在
	// user, err := o.userRepo.GetUserByID(userID)
	// if err != nil {
	// 	return uuid.UUID{}, err
	// }

	amount, err := o.CalculateCartAmount(cartItems...)
	if err != nil {
		return uuid.UUID{}, err
	}

	cartId, err := o.orderRepo.CreateCacheCart(ctx, userID, model.Cart{
		UserID:     userID,
		Amount:     amount,
		OrderItems: cartItems,
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return cartId, nil
}

func (o *OrderService) GetCacheCart(ctx context.Context, userID uint) (*model.Cart, error) {
	//驗證使用者是否存在
	// user, err := o.userRepo.GetUserByID(userID)
	// if err != nil {
	// 	return uuid.UUID{}, err
	// }
	return o.orderRepo.GetCacheCart(ctx, userID)
}

// 購物車修改商品與商品數量，需要購物車OrderItem修改後的狀態
// 參數:
//
//	ctx: 上下文
//	userID(uint): 用戶ID
//	productID(uint): 商品ID
//	quantity(int): 商品數量
//
// 返回值:
func (o *OrderService) UpdateCacheCart(ctx context.Context, userID uint, cartItems ...model.CartItem) (*model.Cart, error) {
	//驗證使用者是否存在
	// user, err := o.userRepo.GetUserByID(userID)
	// if err != nil {
	// 	return uuid.UUID{}, err
	// }
	amount, err := o.CalculateCartAmount(cartItems...)
	if err != nil {
		return nil, err
	}

	cart, err := o.GetCacheCart(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, ErrCartNotExist
	}

	return o.orderRepo.UpdateCacheCart(ctx, userID, model.Cart{
		CartID:     cart.CartID,
		UserID:     userID,
		Amount:     amount,
		OrderItems: cartItems,
	})
}
