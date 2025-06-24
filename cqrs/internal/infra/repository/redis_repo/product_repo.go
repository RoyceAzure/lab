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
	ErrProductNotFound ProductRepoError = errors.New("product not found")
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
func (s *ProductRepo) DeltaProductStock(ctx context.Context, productID string, delta int64) error {
	redisKey := generateProductStockKey(productID)
	err := s.productCache.HIncrBy(ctx, redisKey, "stock", delta).Err()
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
