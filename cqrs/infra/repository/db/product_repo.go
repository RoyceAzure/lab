package db

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ProductRepo struct {
	ProductCache *redis_cache.RedisCache
	db           *DbDao
}

func NewProductRepo(db *DbDao, redisCache *redis_cache.RedisCache) *ProductRepo {
	return &ProductRepo{db: db, ProductCache: redisCache}
}

// Create - 創建商品
func (s *ProductRepo) CreateProduct(product *model.Product) error {
	return s.db.Create(product).Error
}

// Read - 根據ID查詢商品
func (s *ProductRepo) GetProductByID(id uint) (*model.Product, error) {
	redisKey := fmt.Sprintf("%d", id)
	product, err := s.ProductCache.HGetAll(context.Background(), redisKey)
	if err != nil {
		return nil, err
	}

	if len(product) > 0 {
		productModel, err := convertRedisMapToProduct(id, product)
		if err != nil {
			return nil, err
		}
		return productModel, nil
	}

	var productFromDB model.Product
	err = s.db.First(&productFromDB, id).Error
	if err != nil {
		return nil, err
	}
	return &productFromDB, nil
}

// convertRedisMapToProduct 將 Redis 的 map[string]string 轉換為 model.Product
func convertRedisMapToProduct(id uint, productMap map[string]any) (*model.Product, error) {
	price, err := decimal.NewFromString(productMap["price"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %v", err)
	}

	stock, err := strconv.ParseUint(productMap["stock"].(string), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stock format: %v", err)
	}

	reserved, err := strconv.ParseUint(productMap["reserved"].(string), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid reserved format: %v", err)
	}

	createdAt, err := time.Parse(time.RFC3339, productMap["created_at"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid created_at format: %v", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, productMap["updated_at"].(string))
	if err != nil {
		return nil, fmt.Errorf("invalid updated_at format: %v", err)
	}

	return &model.Product{
		ProductID:   id,
		Code:        productMap["code"].(string),
		Name:        productMap["name"].(string),
		Price:       price,
		Stock:       uint(stock),
		Reserved:    uint(reserved),
		Category:    productMap["category"].(string),
		Description: productMap["description"].(string),
		BaseModel: model.BaseModel{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}, nil
}

// Read - 根據商品代碼查詢
func (s *ProductRepo) GetProductByCode(code string) (*model.Product, error) {
	var product model.Product
	err := s.db.Where("code = ?", code).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func convertProductFromRedis(key string, product map[string]any) (*model.Product, error) {
	productID, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return nil, err
	}
	price, err := decimal.NewFromString(product["price"].(string))
	if err != nil {
		return nil, err
	}
	stock, err := strconv.ParseUint(product["stock"].(string), 10, 64)
	if err != nil {
		return nil, err
	}
	return &model.Product{
		ProductID:   uint(productID),
		Code:        product["code"].(string),
		Name:        product["name"].(string),
		Price:       price,
		Stock:       uint(stock),
		Category:    product["category"].(string),
		Description: product["description"].(string),
	}, nil
}

// Read - 查詢所有商品
func (s *ProductRepo) GetAllProducts() ([]model.Product, error) {
	var products []model.Product
	productsMap, err := s.ProductCache.HMGetAll(context.Background(), "*")
	if err != nil {
		return nil, err
	}

	for key, product := range productsMap {
		product, err := convertProductFromRedis(key, product)
		if err != nil {
			return nil, err
		}
		products = append(products, *product)
	}

	if len(products) == 0 {
		err := s.db.Find(&products).Error
		if err != nil {
			return nil, err
		}
	}

	return products, err
}

// Read - 根據分類查詢商品
func (s *ProductRepo) GetProductsByCategory(category string) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("category = ?", category).Find(&products).Error
	return products, err
}

// Read - 根據價格範圍查詢商品
func (s *ProductRepo) GetProductsByPriceRange(minPrice, maxPrice uint) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("price BETWEEN ? AND ?", minPrice, maxPrice).Find(&products).Error
	return products, err
}

// Read - 查詢有庫存的商品
func (s *ProductRepo) GetProductsInStock() ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("stock > 0").Find(&products).Error
	return products, err
}

// Read - 根據名稱搜尋商品（模糊搜尋）
func (s *ProductRepo) SearchProductsByName(name string) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("name LIKE ?", "%"+name+"%").Find(&products).Error
	return products, err
}

// Update - 更新商品
func (s *ProductRepo) UpdateProduct(product *model.Product) error {
	return s.db.Save(product).Error
}

// Update - 部分更新商品
func (s *ProductRepo) UpdateProductFields(id uint, updates map[string]interface{}) error {
	return s.db.Model(&model.Product{}).Where("product_id = ?", id).Updates(updates).Error
}

// Update - 更新庫存
func (s *ProductRepo) UpdateStock(id uint, stock uint) error {
	return s.db.Model(&model.Product{}).Where("product_id = ?", id).Update("stock", stock).Error
}

// Update - 增加庫存
func (s *ProductRepo) AddStock(id uint, quantity uint) error {
	return s.db.Model(&model.Product{}).Where("product_id = ?", id).Update("stock", gorm.Expr("stock + ?", quantity)).Error
}

// Update - 減少庫存
func (s *ProductRepo) ReduceStock(id uint, quantity uint) error {
	return s.db.Model(&model.Product{}).Where("product_id = ? AND stock >= ?", id, quantity).Update("stock", gorm.Expr("stock - ?", quantity)).Error
}

// Delete - 軟刪除商品
func (s *ProductRepo) DeleteProduct(id uint) error {
	return s.db.Delete(&model.Product{}, id).Error
}

// Delete - 硬刪除商品
func (s *ProductRepo) HardDeleteProduct(id uint) error {
	return s.db.Unscoped().Delete(&model.Product{}, id).Error
}

// 分頁查詢商品
func (s *ProductRepo) GetProductsPaginated(page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	offset := (page - 1) * pageSize

	// 計算總數
	s.db.Model(&model.Product{}).Count(&total)

	// 分頁查詢
	err := s.db.Offset(offset).Limit(pageSize).Find(&products).Error

	return products, total, err
}

// 根據條件分頁查詢
func (s *ProductRepo) GetProductsPaginatedWithCondition(page, pageSize int, condition map[string]interface{}) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	offset := (page - 1) * pageSize
	query := s.db.Model(&model.Product{})

	// 應用條件
	for key, value := range condition {
		query = query.Where(key+" = ?", value)
	}

	// 計算總數
	query.Count(&total)

	// 分頁查詢
	err := query.Offset(offset).Limit(pageSize).Find(&products).Error

	return products, total, err
}

// 批量創建商品
func (s *ProductRepo) CreateProductsBatch(products []model.Product) error {
	return s.db.Create(&products).Error
}

// 批量更新商品價格
func (s *ProductRepo) UpdateProductsPriceBatch(category string, priceMultiplier float64) error {
	return s.db.Model(&model.Product{}).Where("category = ?", category).Update("price", gorm.Expr("price * ?", priceMultiplier)).Error
}

// 取得低庫存商品
func (s *ProductRepo) GetLowStockProducts(threshold uint) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("stock <= ?", threshold).Find(&products).Error
	return products, err
}

// 取得熱門商品（根據訂單項目統計）
func (s *ProductRepo) GetPopularProducts(limit int) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Preload("OrderItems").
		Joins("LEFT JOIN order_items ON products.product_id = order_items.product_id").
		Group("products.product_id").
		Order("COUNT(order_items.id) DESC").
		Limit(limit).
		Find(&products).Error
	return products, err
}
