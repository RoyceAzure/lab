package test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Product struct {
	ProductID int             `json:"product_id"`
	Code      string          `json:"code"`
	Name      string          `json:"name"`
	Price     decimal.Decimal `json:"price"`
	Stock     int             `json:"stock"`
	Reserved  int             `json:"reserved"`
	Category  string          `json:"category"`
}

// Equal 方法用於比較兩個 Product 是否相等
func (p Product) Equal(other Product) bool {
	return p.ProductID == other.ProductID &&
		p.Code == other.Code &&
		p.Name == other.Name &&
		p.Price.Equal(other.Price) &&
		p.Stock == other.Stock &&
		p.Reserved == other.Reserved &&
		p.Category == other.Category
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
	productID := rand.Intn(1000)
	return Product{
		ProductID: productID,
		Code:      fmt.Sprintf("P%04d", productID),
		Name:      fmt.Sprintf("Product %d", productID),
		Price:     decimal.NewFromFloat(rand.Float64() * 1000).Round(2),
		Stock:     rand.Intn(1000),
		Reserved:  rand.Intn(100),
		Category:  categories[rand.Intn(len(categories))],
	}
}

func (s *RedisHashTestSuite) generateRandomProducts(n int) []Product {
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Sports"}
	usedIDs := make(map[int]struct{}, n)
	products := make([]Product, 0, n)

	for len(products) < n {
		productID := rand.Intn(10000) // 增大範圍以降低碰撞
		if _, exists := usedIDs[productID]; exists {
			continue // 已存在則跳過
		}
		usedIDs[productID] = struct{}{}
		product := Product{
			ProductID: productID,
			Code:      fmt.Sprintf("P%04d", productID),
			Name:      fmt.Sprintf("Product %d", productID),
			Price:     decimal.NewFromFloat(rand.Float64() * 1000).Round(2),
			Stock:     rand.Intn(1000),
			Reserved:  rand.Intn(100),
			Category:  categories[rand.Intn(len(categories))],
		}
		products = append(products, product)
	}
	return products
}

// convertFieldsToProduct 將 map[string]string 轉換為 Product
func (s *RedisHashTestSuite) convertFieldsToProduct(fields map[string]string) (Product, error) {
	productID, err := strconv.Atoi(fields["product_id"])
	if err != nil {
		return Product{}, err
	}

	stock, err := strconv.Atoi(fields["stock"])
	if err != nil {
		return Product{}, err
	}

	reserved, err := strconv.Atoi(fields["reserved"])
	if err != nil {
		return Product{}, err
	}

	price, err := decimal.NewFromString(fields["price"])
	if err != nil {
		return Product{}, err
	}

	return Product{
		ProductID: productID,
		Code:      fields["code"],
		Name:      fields["name"],
		Price:     price,
		Stock:     stock,
		Reserved:  reserved,
		Category:  fields["category"],
	}, nil
}

func (s *RedisHashTestSuite) TestHashOperations() {
	// 生成100個隨機商品
	products := s.generateRandomProducts(100)

	// 測試 HMSet 和 HGet
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)

		// 使用 HMSet 直接設置整個商品結構
		err := s.cache.HMSet(s.ctx, key, product)
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

	retrievedProduct, err := s.convertFieldsToProduct(allFields)
	require.NoError(s.T(), err)
	require.True(s.T(), products[0].Equal(retrievedProduct))

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
	require.Len(s.T(), keys, 7) // 7 個字段：product_id, code, name, price, stock, reserved, category
	require.Contains(s.T(), keys, "code")
	require.Contains(s.T(), keys, "name")

	// 測試 HVals
	vals, err := s.cache.HVals(s.ctx, key)
	require.NoError(s.T(), err)
	require.Len(s.T(), vals, 7)

	// 測試 HLen
	length, err := s.cache.HLen(s.ctx, key)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(7), length)

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
	batchItems := make(map[string]any, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		batchItems[key] = product
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

		retrievedProduct, err := s.convertFieldsToProduct(fields)
		require.NoError(s.T(), err)
		require.True(s.T(), product.Equal(retrievedProduct), "產品數據不匹配")
	}
}

func TestRedisHashSuite(t *testing.T) {
	suite.Run(t, new(RedisHashTestSuite))
}
