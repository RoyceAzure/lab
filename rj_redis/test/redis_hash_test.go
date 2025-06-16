package test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Product struct {
	ProductID uint
	Code      string
	Name      string
	Price     decimal.Decimal
	Stock     uint
	Reserved  uint
	Category  string
}

type RedisHashTestSuite struct {
	suite.Suite
	cache cache.Cache
	ctx   context.Context
}

func (s *RedisHashTestSuite) SetupSuite() {
	redisClient, err := redis_client.GetRedisClient(testRedisAddr, redis_client.WithPassword(testRedisPassword))
	require.NoError(s.T(), err)
	s.cache = redis_cache.NewRedisCache(redisClient, "test_hash")
	s.ctx = context.Background()
}

func (s *RedisHashTestSuite) TearDownSuite() {
	// 清理所有測試數據
	err := s.cache.Clear(s.ctx)
	require.NoError(s.T(), err)
}

func (s *RedisHashTestSuite) SetupTest() {
	// 每個測試方法執行前清理數據
	err := s.cache.Clear(s.ctx)
	require.NoError(s.T(), err)
}

func (s *RedisHashTestSuite) generateRandomProduct() Product {
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Sports"}
	productID := uint(rand.Intn(1000))
	return Product{
		ProductID: productID,
		Code:      fmt.Sprintf("P%04d", productID),
		Name:      fmt.Sprintf("Product %d", productID),
		Price:     decimal.NewFromFloat(rand.Float64() * 1000).Round(2),
		Stock:     uint(rand.Intn(1000)),
		Reserved:  uint(rand.Intn(100)),
		Category:  categories[rand.Intn(len(categories))],
	}
}

func (s *RedisHashTestSuite) generateRandomProducts(n int) []Product {
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Sports"}
	usedIDs := make(map[uint]struct{}, n)
	products := make([]Product, 0, n)

	for len(products) < n {
		productID := uint(rand.Intn(10000)) // 增大範圍以降低碰撞
		if _, exists := usedIDs[productID]; exists {
			continue // 已存在則跳過
		}
		usedIDs[productID] = struct{}{}
		product := Product{
			ProductID: productID,
			Code:      fmt.Sprintf("P%04d", productID),
			Name:      fmt.Sprintf("Product %d", productID),
			Price:     decimal.NewFromFloat(rand.Float64() * 1000).Round(2),
			Stock:     uint(rand.Intn(1000)),
			Reserved:  uint(rand.Intn(100)),
			Category:  categories[rand.Intn(len(categories))],
		}
		products = append(products, product)
	}
	return products
}

func (s *RedisHashTestSuite) TestHashOperations() {
	// 生成100個隨機商品
	products := s.generateRandomProducts(100)

	// 測試 HSet 和 HGet
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)

		// 設置商品資訊
		err := s.cache.HSet(s.ctx, key, "code", product.Code)
		require.NoError(s.T(), err)
		err = s.cache.HSet(s.ctx, key, "name", product.Name)
		require.NoError(s.T(), err)
		err = s.cache.HSet(s.ctx, key, "price", product.Price.String())
		require.NoError(s.T(), err)
		err = s.cache.HSet(s.ctx, key, "stock", fmt.Sprintf("%d", product.Stock))
		require.NoError(s.T(), err)
		err = s.cache.HSet(s.ctx, key, "reserved", fmt.Sprintf("%d", product.Reserved))
		require.NoError(s.T(), err)
		err = s.cache.HSet(s.ctx, key, "category", product.Category)
		require.NoError(s.T(), err)

		// 驗證設置的值
		code, err := s.cache.HGet(s.ctx, key, "code")
		require.NoError(s.T(), err)
		require.Equal(s.T(), product.Code, code)

		name, err := s.cache.HGet(s.ctx, key, "name")
		require.NoError(s.T(), err)
		require.Equal(s.T(), product.Name, name)
	}

	// 測試 HGetAll
	key := fmt.Sprintf("product:%d", products[0].ProductID)
	allFields, err := s.cache.HGetAll(s.ctx, key)
	require.NoError(s.T(), err)
	require.Len(s.T(), allFields, 6)
	require.Equal(s.T(), products[0].Code, allFields["code"])
	require.Equal(s.T(), products[0].Name, allFields["name"])

	// 測試 HExists
	exists, err := s.cache.HExists(s.ctx, key, "code")
	require.NoError(s.T(), err)
	require.True(s.T(), exists)

	exists, err = s.cache.HExists(s.ctx, key, "nonexistent")
	require.NoError(s.T(), err)
	require.False(s.T(), exists)

	// 測試 HKeys
	keys, err := s.cache.HKeys(s.ctx, key)
	require.NoError(s.T(), err)
	require.Len(s.T(), keys, 6)
	require.Contains(s.T(), keys, "code")
	require.Contains(s.T(), keys, "name")

	// 測試 HVals
	vals, err := s.cache.HVals(s.ctx, key)
	require.NoError(s.T(), err)
	require.Len(s.T(), vals, 6)

	// 測試 HLen
	length, err := s.cache.HLen(s.ctx, key)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(6), length)

	// 測試 HIncrBy
	err = s.cache.HSet(s.ctx, key, "stock", "100")
	require.NoError(s.T(), err)

	newStock, err := s.cache.HIncrBy(s.ctx, key, "stock", 50)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(150), newStock)

	// 測試 HDel
	err = s.cache.HDel(s.ctx, key, "code", "name")
	require.NoError(s.T(), err)

	exists, err = s.cache.HExists(s.ctx, key, "code")
	require.NoError(s.T(), err)
	require.False(s.T(), exists)
}

func (s *RedisHashTestSuite) TestBulkProductOperations() {
	// 生成100個隨機商品
	products := s.generateRandomProducts(100)

	// 準備批量寫入資料
	batchItems := make(map[string]map[string]any, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		batchItems[key] = map[string]any{
			"code":     product.Code,
			"name":     product.Name,
			"price":    product.Price.String(),
			"stock":    fmt.Sprintf("%d", product.Stock),
			"reserved": fmt.Sprintf("%d", product.Reserved),
			"category": product.Category,
		}
	}

	// 批量寫入所有產品
	err := s.cache.HMSetMulti(s.ctx, batchItems)
	require.NoError(s.T(), err)

	// 批量讀取所有產品
	keys := make([]string, 0, len(products))
	for _, product := range products {
		keys = append(keys, fmt.Sprintf("product:%d", product.ProductID))
	}
	batchResult, err := s.cache.HMGetAll(s.ctx, keys...)
	require.NoError(s.T(), err)
	require.Len(s.T(), batchResult, len(products), "批量讀取產品數量不正確")

	// 驗證所有產品的數據一致性
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		fields, ok := batchResult[key]
		require.True(s.T(), ok, "缺少產品: %s", key)
		require.Len(s.T(), fields, 6, "產品字段數量不正確")
		require.Equal(s.T(), product.Code, fields["code"], "產品代碼不匹配")
		require.Equal(s.T(), product.Name, fields["name"], "產品名稱不匹配")
		require.Equal(s.T(), product.Price.String(), fields["price"], "產品價格不匹配")
		require.Equal(s.T(), fmt.Sprintf("%d", product.Stock), fields["stock"], "產品庫存不匹配")
		require.Equal(s.T(), fmt.Sprintf("%d", product.Reserved), fields["reserved"], "產品預訂數量不匹配")
		require.Equal(s.T(), product.Category, fields["category"], "產品類別不匹配")
	}
}

func TestRedisHashSuite(t *testing.T) {
	suite.Run(t, new(RedisHashTestSuite))
}
