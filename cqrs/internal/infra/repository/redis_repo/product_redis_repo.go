package redis_repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// IProductRedisRepository 定義 Redis 商品操作的介面
type IProductRedisRepository interface {
	// CreateProductStock 創建商品庫存
	CreateProductStock(ctx context.Context, productID string, stock uint) error

	// GetProductStock 取得商品庫存數量
	GetProductStock(ctx context.Context, productID string) (int, error)

	// AddProductStock 增加商品庫存數量
	AddProductStock(ctx context.Context, productID string, quantity uint) (int, error)

	// UpdateProductStock 修改商品庫存數量
	UpdateProductStock(ctx context.Context, productID string, quantity uint) error

	// DeleteProductStock 刪除商品庫存
	DeleteProductStock(ctx context.Context, productID string) error

	// DeductProductStock 原子性扣減庫存
	DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error)
}

type ProductRepoError error

var (
	ErrProductNotFound       ProductRepoError = errors.New("product not found")
	ErrProductStockNotEnough ProductRepoError = errors.New("product stock not enough")
)

/*	redis 專注商品庫存
	結構:
	商品ID: {
		stock: 100,
	}*/

type ProductRedisRepo struct {
	productCache *redis.Client
}

func NewProductRepo(productCache *redis.Client) *ProductRedisRepo {
	return &ProductRedisRepo{productCache: productCache}
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

func (s *ProductRedisRepo) CreateProductStock(ctx context.Context, productID string, stock uint) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.HSet(ctx, redisKey, "stock", stock).Err()
	if err != nil {
		return err
	}
	return nil
}

// 取得 庫存商品數量
// 錯誤:
//   - ProductNotFound: 商品不存在
//   - err: 其他錯誤
func (s *ProductRedisRepo) GetProductStock(ctx context.Context, productID string) (int, error) {
	redisKey := generateProductStockKey(productID)
	stock, err := s.productCache.HGet(ctx, redisKey, "stock").Result()
	if err != nil {
		return 0, err
	}

	if stock == "" {
		return 0, ErrProductNotFound
	}

	stockInt, err := strconv.ParseInt(stock, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(stockInt), nil
}

// 增加庫存商品數量
func (s *ProductRedisRepo) AddProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	redisKey := generateProductStockKey(productID)
	// HIncrBy 會返回增加後的值
	result := s.productCache.HIncrBy(ctx, redisKey, "stock", int64(quantity))
	if err := result.Err(); err != nil {
		return 0, err
	}
	return int(result.Val()), nil
}

// 修改庫存商品數量
func (s *ProductRedisRepo) UpdateProductStock(ctx context.Context, productID string, quantity uint) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.HSet(ctx, redisKey, "stock", quantity).Err()
	if err != nil {
		return err
	}
	return nil
}

// DeleteProductStock 直接刪除商品資料
func (s *ProductRedisRepo) DeleteProductStock(ctx context.Context, productID string) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.Del(ctx, redisKey).Err()
	if err != nil {
		return err
	}
	return nil
}

// 原子性扣減庫存
/*
	返回值:
		- 扣減後的庫存數量
		- 錯誤:
			- ErrProductNotFound: 商品不存在
			- ErrProductStockNotEnough: 庫存不足
			- err: 其他錯誤
*/
func (s *ProductRedisRepo) DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
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
		return int(resultInt), nil
	}
}

// 確保 ProductRedisRepo 實現了 ProductRedisRepository 介面
var _ IProductRedisRepository = (*ProductRedisRepo)(nil)
