package redis_repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type ProductRepoError error

var (
	ErrProductNotFound       ProductRepoError = errors.New("product not found")
	ErrProductStockNotEnough ProductRepoError = errors.New("product stock not enough")
)

type ProductRepo struct {
	productCache *redis.Client
}

func NewProductRepo(productCache *redis.Client) *ProductRepo {
	return &ProductRepo{productCache: productCache}
}

// redis 商品庫存
// 商品庫存先統一使用redis 當作唯一真相來源
// 結構:
//
//	商品ID: {
//		stock: 100,
//	}
//
//	商品ID: {
//		stock: 100,
//	}
func generateProductStockKey(productID string) string {
	return fmt.Sprintf("product:%s:stock", productID)
}

func (s *ProductRepo) CreateProductStock(ctx context.Context, productID string, stock uint) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.HSet(ctx, redisKey, "stock", stock).Err()
	if err != nil {
		return err
	}
	return nil
}

// // 取得商品資訊q
// func (s *ProductRepo) GetProduct(ctx context.Context, productID string) (*model.Product, error) {
// 	redisKey := generateProductStockKey(productID)
// 	product, err := s.productCache.HGet(ctx, redisKey, "product").Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return product, nil
// }

// 取得 庫存商品數量
// 錯誤:
//   - ProductNotFound: 商品不存在
//   - err: 其他錯誤
func (s *ProductRepo) GetProductStock(ctx context.Context, productID string) (uint, error) {
	redisKey := generateProductStockKey(productID)
	stock, err := s.productCache.HGet(ctx, redisKey, "stock").Result()
	if err != nil {
		return 0, err
	}

	if stock == "" {
		return 0, ErrProductNotFound
	}

	stockInt, err := strconv.ParseUint(stock, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(stockInt), nil
}

// 修改庫存商品數量
func (s *ProductRepo) AddProductStock(ctx context.Context, productID string, quantity uint) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.HIncrBy(ctx, redisKey, "stock", int64(quantity)).Err()
	if err != nil {
		return err
	}
	return nil
}

// DeleteProductStock 直接刪除商品資料
func (s *ProductRepo) DeleteProductStock(ctx context.Context, productID string) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.Del(ctx, redisKey).Err()
	if err != nil {
		return err
	}
	return nil
}

// 原子性扣減庫存
func (s *ProductRepo) DeductProductStock(ctx context.Context, productID string, quantity uint) (int64, error) {
	redisKey := generateProductStockKey(productID)

	const stockDeductionScript = `
	local key = KEYS[1]
	local quantity = tonumber(ARGV[1])
	local field = ARGV[2]
	
	if redis.call('EXISTS', key) == 0 then
		return -1
	end
	
	local current_stock = redis.call('HGET', key, field)
	if not current_stock then
		return -1
	end
	
	current_stock = tonumber(current_stock)
	
	if current_stock < quantity then
		return -2  -- 表示庫存不足
	end
	
	local new_stock = redis.call('HINCRBY', key, field, -quantity)
	return new_stock
	`

	result, err := s.productCache.Eval(ctx, stockDeductionScript, []string{redisKey}, quantity, "stock").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to deduct stock: %w", err)
	}

	resultInt, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}

	switch {
	case resultInt == -1:
		return 0, fmt.Errorf("%w: product with id %s not found", ErrProductNotFound, productID)
	case resultInt == -2:
		return 0, fmt.Errorf("%w: product with id %s stock not enough", ErrProductStockNotEnough, productID)
	default:
		return resultInt, nil // 返回扣減後的庫存（可能為 0，表示剛好扣完）
	}
}
