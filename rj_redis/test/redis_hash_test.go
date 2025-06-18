package test

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

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

// BaseModel 基礎模型
type BaseModel struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrderItem 訂單項目
type OrderItem struct {
	OrderID   uint `json:"order_id"`
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
	BaseModel
}

// Order 訂單結構
type Order struct {
	OrderID    uint            `json:"order_id"`
	UserID     uint            `json:"user_id"`
	OrderItems []OrderItem     `json:"order_items"`
	Amount     decimal.Decimal `json:"amount"`
	OrderDate  time.Time       `json:"order_date"`
	BaseModel
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

// convertFieldsToProduct 將 map[string]any 轉換為 Product
func (s *RedisHashTestSuite) convertFieldsToProduct(fields map[string]any) (Product, error) {
	// 轉換 product_id
	productID, ok := fields["product_id"].(float64)
	if !ok {
		return Product{}, fmt.Errorf("invalid product_id type")
	}

	// 轉換 stock
	stock, ok := fields["stock"].(float64)
	if !ok {
		return Product{}, fmt.Errorf("invalid stock type")
	}

	// 轉換 reserved
	reserved, ok := fields["reserved"].(float64)
	if !ok {
		return Product{}, fmt.Errorf("invalid reserved type")
	}

	// 轉換 price
	priceStr, ok := fields["price"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid price type")
	}
	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		return Product{}, err
	}

	// 轉換 detail.weight
	weightStr, ok := fields["detail.weight"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid detail.weight type")
	}
	weight, err := decimal.NewFromString(weightStr)
	if err != nil {
		return Product{}, err
	}

	// 轉換 detail.dimensions
	lengthStr, ok := fields["detail.dimensions.length"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid detail.dimensions.length type")
	}
	length, err := decimal.NewFromString(lengthStr)
	if err != nil {
		return Product{}, err
	}

	widthStr, ok := fields["detail.dimensions.width"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid detail.dimensions.width type")
	}
	width, err := decimal.NewFromString(widthStr)
	if err != nil {
		return Product{}, err
	}

	heightStr, ok := fields["detail.dimensions.height"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid detail.dimensions.height type")
	}
	height, err := decimal.NewFromString(heightStr)
	if err != nil {
		return Product{}, err
	}

	// 轉換 detail.tags
	tagsStr, ok := fields["detail.tags"].(string)
	if !ok {
		return Product{}, fmt.Errorf("invalid detail.tags type")
	}
	tags := strings.Split(tagsStr, ",")

	return Product{
		ProductID: int(productID),
		Code:      fields["code"].(string),
		Name:      fields["name"].(string),
		Price:     price,
		Stock:     int(stock),
		Reserved:  int(reserved),
		Category:  fields["category"].(string),
		Detail: ProductDetail{
			Description: fields["detail.description"].(string),
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
	// 生成100個隨機商品
	products := s.generateRandomProducts(100)

	// 準備批量設置的數據
	items := make(map[string]any)
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		items[key] = product
	}

	// 批量設置
	err := s.cache.HMSetMulti(s.ctx, items)
	require.NoError(s.T(), err)

	// 驗證設置的值
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)

		// 獲取所有字段
		allFields, err := s.cache.HGetAll(s.ctx, key)
		require.NoError(s.T(), err)

		// 轉換回 Product 結構
		retrievedProduct, err := s.convertFieldsToProduct(allFields)
		require.NoError(s.T(), err)

		// 驗證數據
		require.True(s.T(), product.Equal(retrievedProduct))
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

func (s *RedisHashTestSuite) TestHMSetMultiWithArray() {
	// 創建一個訂單
	order := Order{
		OrderID:   1001,
		UserID:    2001,
		Amount:    decimal.NewFromFloat(491.75),
		OrderDate: time.Now(),
		BaseModel: BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderItems: []OrderItem{
			{
				OrderID:   1001,
				ProductID: 101,
				Quantity:  2,
				BaseModel: BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
			{
				OrderID:   1001,
				ProductID: 102,
				Quantity:  1,
				BaseModel: BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			},
		},
	}

	// 準備批量寫入資料
	batchItems := make(map[string]any)
	batchItems["order:1001"] = order

	// 批量寫入
	err := s.cache.HMSetMulti(s.ctx, batchItems)
	require.NoError(s.T(), err)

	// 讀取數據
	fields, err := s.cache.HGetAll(s.ctx, "order:1001")
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), fields)

	// 打印平坦化後的字段
	fmt.Println("平坦化後的字段：")
	for k, v := range fields {
		fmt.Printf("%s: %s\n", k, v)
	}

	// 驗證基本字段
	require.Equal(s.T(), "1001", fields["order_id"])
	require.Equal(s.T(), "2001", fields["user_id"])
	require.Equal(s.T(), "491.75", fields["amount"])

	// 驗證數組字段
	// 注意：數組會被轉換為逗號分隔的字符串
	require.Contains(s.T(), fields, "order_items")
	itemsStr := fields["order_items"]
	fmt.Printf("order_items 字段的值：%s\n", itemsStr)
}

func (s *RedisHashTestSuite) TestHMGetAll() {
	// 生成100個隨機商品
	products := s.generateRandomProducts(100)

	// 準備批量設置的數據
	items := make(map[string]any)
	keys := make([]string, 0, len(products))
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		items[key] = product
		keys = append(keys, key)
	}

	// 批量設置
	err := s.cache.HMSetMulti(s.ctx, items)
	require.NoError(s.T(), err)

	// 批量獲取所有商品
	allProducts, err := s.cache.HMGetAll(s.ctx, keys...)
	require.NoError(s.T(), err)
	require.Len(s.T(), allProducts, len(products))

	// 驗證每個商品的數據
	for _, product := range products {
		key := fmt.Sprintf("product:%d", product.ProductID)
		fields, exists := allProducts[key]
		require.True(s.T(), exists, "商品數據應該存在")

		// 轉換回 Product 結構
		retrievedProduct, err := s.convertFieldsToProduct(fields)
		require.NoError(s.T(), err)

		// 驗證數據
		require.True(s.T(), product.Equal(retrievedProduct))
	}
}

func TestRedisHashSuite(t *testing.T) {
	suite.Run(t, new(RedisHashTestSuite))
}
