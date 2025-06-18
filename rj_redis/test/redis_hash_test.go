package test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ProductDetail 產品詳細資訊
type ProductDetail struct {
	Description string          `json:"description"`
	Weight      decimal.Decimal `json:"weight"`
	Dimensions  Dimensions      `json:"dimensions"`
	Tags        []string        `json:"tags"`
}

// Dimensions 產品尺寸
type Dimensions struct {
	Length decimal.Decimal `json:"length"`
	Width  decimal.Decimal `json:"width"`
	Height decimal.Decimal `json:"height"`
}

// Product 產品結構
type Product struct {
	ProductID int             `json:"product_id"`
	Code      string          `json:"code"`
	Name      string          `json:"name"`
	Price     decimal.Decimal `json:"price"`
	Stock     int             `json:"stock"`
	Reserved  int             `json:"reserved"`
	Category  string          `json:"category"`
	Detail    ProductDetail   `json:"detail"`
}

// Equal 方法用於比較兩個 Product 是否相等
func (p Product) Equal(other Product) bool {
	return p.ProductID == other.ProductID &&
		p.Code == other.Code &&
		p.Name == other.Name &&
		p.Price.Equal(other.Price) &&
		p.Stock == other.Stock &&
		p.Reserved == other.Reserved &&
		p.Category == other.Category &&
		p.Detail.Description == other.Detail.Description &&
		p.Detail.Weight.Equal(other.Detail.Weight) &&
		p.Detail.Dimensions.Length.Equal(other.Detail.Dimensions.Length) &&
		p.Detail.Dimensions.Width.Equal(other.Detail.Dimensions.Width) &&
		p.Detail.Dimensions.Height.Equal(other.Detail.Dimensions.Height) &&
		compareStringSlices(p.Detail.Tags, other.Detail.Tags)
}

// compareStringSlices 比較兩個字符串切片是否相等
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	tags := []string{"New", "Popular", "Sale", "Limited", "Premium"}
	productID := rand.Intn(1000)

	return Product{
		ProductID: productID,
		Code:      fmt.Sprintf("P%04d", productID),
		Name:      fmt.Sprintf("Product %d", productID),
		Price:     decimal.NewFromFloat(rand.Float64() * 1000).Round(2),
		Stock:     rand.Intn(1000),
		Reserved:  rand.Intn(100),
		Category:  categories[rand.Intn(len(categories))],
		Detail: ProductDetail{
			Description: fmt.Sprintf("Description for product %d", productID),
			Weight:      decimal.NewFromFloat(rand.Float64() * 10).Round(2),
			Dimensions: Dimensions{
				Length: decimal.NewFromFloat(rand.Float64() * 100).Round(2),
				Width:  decimal.NewFromFloat(rand.Float64() * 100).Round(2),
				Height: decimal.NewFromFloat(rand.Float64() * 100).Round(2),
			},
			Tags: []string{
				tags[rand.Intn(len(tags))],
				tags[rand.Intn(len(tags))],
			},
		},
	}
}

func (s *RedisHashTestSuite) generateRandomProducts(n int) []Product {
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Sports"}
	tags := []string{"New", "Popular", "Sale", "Limited", "Premium"}
	usedIDs := make(map[int]struct{}, n)
	products := make([]Product, 0, n)

	for len(products) < n {
		productID := rand.Intn(10000)
		if _, exists := usedIDs[productID]; exists {
			continue
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
			Detail: ProductDetail{
				Description: fmt.Sprintf("Description for product %d", productID),
				Weight:      decimal.NewFromFloat(rand.Float64() * 10).Round(2),
				Dimensions: Dimensions{
					Length: decimal.NewFromFloat(rand.Float64() * 100).Round(2),
					Width:  decimal.NewFromFloat(rand.Float64() * 100).Round(2),
					Height: decimal.NewFromFloat(rand.Float64() * 100).Round(2),
				},
				Tags: []string{
					tags[rand.Intn(len(tags))],
					tags[rand.Intn(len(tags))],
				},
			},
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

	weight, err := decimal.NewFromString(fields["detail.weight"])
	if err != nil {
		return Product{}, err
	}

	length, err := decimal.NewFromString(fields["detail.dimensions.length"])
	if err != nil {
		return Product{}, err
	}

	width, err := decimal.NewFromString(fields["detail.dimensions.width"])
	if err != nil {
		return Product{}, err
	}

	height, err := decimal.NewFromString(fields["detail.dimensions.height"])
	if err != nil {
		return Product{}, err
	}

	// 解析標籤
	tags := strings.Split(fields["detail.tags"], ",")

	return Product{
		ProductID: productID,
		Code:      fields["code"],
		Name:      fields["name"],
		Price:     price,
		Stock:     stock,
		Reserved:  reserved,
		Category:  fields["category"],
		Detail: ProductDetail{
			Description: fields["detail.description"],
			Weight:      weight,
			Dimensions: Dimensions{
				Length: length,
				Width:  width,
				Height: height,
			},
			Tags: tags,
		},
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

func (s *RedisHashTestSuite) TestHMSetMulti() {
	// 生成3個隨機商品
	products := s.generateRandomProducts(3)

	// 準備批量寫入資料
	batchItems := make(map[string]any, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		batchItems[key] = product
	}

	// 批量寫入所有產品
	err := s.cache.HMSetMulti(s.ctx, batchItems)
	require.NoError(s.T(), err)

	// 驗證每個產品的數據
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)

		// 讀取產品數據
		fields, err := s.cache.HGetAll(s.ctx, key)
		require.NoError(s.T(), err)
		require.NotEmpty(s.T(), fields, "產品數據不應為空")

		// 轉換並比較數據
		retrievedProduct, err := s.convertFieldsToProduct(fields)
		require.NoError(s.T(), err)
		require.True(s.T(), product.Equal(retrievedProduct), "產品數據不匹配")

		// 驗證特定字段
		require.Equal(s.T(), product.Code, fields["code"])
		require.Equal(s.T(), product.Name, fields["name"])
		require.Equal(s.T(), product.Price.String(), fields["price"])
		require.Equal(s.T(), fmt.Sprintf("%d", product.Stock), fields["stock"])
		require.Equal(s.T(), fmt.Sprintf("%d", product.Reserved), fields["reserved"])
		require.Equal(s.T(), product.Category, fields["category"])

		// 驗證巢狀結構
		require.Equal(s.T(), product.Detail.Description, fields["detail.description"])
		require.Equal(s.T(), product.Detail.Weight.String(), fields["detail.weight"])
		require.Equal(s.T(), product.Detail.Dimensions.Length.String(), fields["detail.dimensions.length"])
		require.Equal(s.T(), product.Detail.Dimensions.Width.String(), fields["detail.dimensions.width"])
		require.Equal(s.T(), product.Detail.Dimensions.Height.String(), fields["detail.dimensions.height"])
		require.Equal(s.T(), strings.Join(product.Detail.Tags, ","), fields["detail.tags"])
	}
}

func (s *RedisHashTestSuite) TestHMSetMultiWithModification() {
	// 生成3個隨機商品
	products := s.generateRandomProducts(3)

	// 準備批量寫入資料
	batchItems := make(map[string]any, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		batchItems[key] = product
	}

	// 第一次批量寫入
	err := s.cache.HMSetMulti(s.ctx, batchItems)
	require.NoError(s.T(), err)

	// 修改產品數據
	for i := range products {
		// 修改基本字段
		products[i].Name = fmt.Sprintf("Modified Product %d", products[i].ProductID)
		products[i].Price = decimal.NewFromFloat(rand.Float64() * 2000).Round(2)
		products[i].Stock = rand.Intn(2000)

		// 修改巢狀結構
		products[i].Detail.Description = fmt.Sprintf("Modified Description for product %d", products[i].ProductID)
		products[i].Detail.Weight = decimal.NewFromFloat(rand.Float64() * 20).Round(2)
		products[i].Detail.Dimensions.Length = decimal.NewFromFloat(rand.Float64() * 200).Round(2)
		products[i].Detail.Dimensions.Width = decimal.NewFromFloat(rand.Float64() * 200).Round(2)
		products[i].Detail.Dimensions.Height = decimal.NewFromFloat(rand.Float64() * 200).Round(2)
		products[i].Detail.Tags = []string{"Modified", "Updated"}
	}

	// 更新批量寫入資料
	batchItems = make(map[string]any, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		batchItems[key] = product
	}

	// 第二次批量寫入（更新）
	err = s.cache.HMSetMulti(s.ctx, batchItems)
	require.NoError(s.T(), err)

	// 驗證修改後的數據
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)

		// 讀取產品數據
		fields, err := s.cache.HGetAll(s.ctx, key)
		require.NoError(s.T(), err)
		require.NotEmpty(s.T(), fields, "產品數據不應為空")

		// 轉換並比較數據
		retrievedProduct, err := s.convertFieldsToProduct(fields)
		require.NoError(s.T(), err)
		require.True(s.T(), product.Equal(retrievedProduct), "修改後的產品數據不匹配")

		// 驗證修改後的特定字段
		require.Equal(s.T(), product.Name, fields["name"], "產品名稱不匹配")
		require.Equal(s.T(), product.Price.String(), fields["price"], "產品價格不匹配")
		require.Equal(s.T(), fmt.Sprintf("%d", product.Stock), fields["stock"], "產品庫存不匹配")

		// 驗證修改後的巢狀結構
		require.Equal(s.T(), product.Detail.Description, fields["detail.description"], "產品描述不匹配")
		require.Equal(s.T(), product.Detail.Weight.String(), fields["detail.weight"], "產品重量不匹配")
		require.Equal(s.T(), product.Detail.Dimensions.Length.String(), fields["detail.dimensions.length"], "產品長度不匹配")
		require.Equal(s.T(), product.Detail.Dimensions.Width.String(), fields["detail.dimensions.width"], "產品寬度不匹配")
		require.Equal(s.T(), product.Detail.Dimensions.Height.String(), fields["detail.dimensions.height"], "產品高度不匹配")
		require.Equal(s.T(), strings.Join(product.Detail.Tags, ","), fields["detail.tags"], "產品標籤不匹配")
	}
}

func TestRedisHashSuite(t *testing.T) {
	suite.Run(t, new(RedisHashTestSuite))
}
