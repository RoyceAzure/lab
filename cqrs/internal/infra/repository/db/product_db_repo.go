package db

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"gorm.io/gorm"
)

var (
	// ErrProductNotFound 商品不存在
	ErrProductNotFound = errors.New("product not found")
	// ErrProductStockNotEnough 商品庫存不足
	ErrProductStockNotEnough = errors.New("product stock not enough")
)

/*
Redis當作唯一真相來源
異步寫入DB
任何異動消息要加上timestamp 後續消費者只會根據timestamp來判斷是否要更新
*/
type ProductDBRepo struct {
	db *DbDao
}

func NewProductDBRepo(db *DbDao) *ProductDBRepo {
	return &ProductDBRepo{db: db}
}

func (s *ProductDBRepo) CreateProduct(product *model.Product) error {
	return s.db.Create(product).Error
}

func (s *ProductDBRepo) GetProductByID(ctx context.Context, productID string) (*model.Product, error) {
	var productFromDB model.Product
	err := s.db.First(&productFromDB, productID).Error
	if err != nil {
		return nil, err
	}
	return &productFromDB, nil
}

func (s *ProductDBRepo) GetProductStock(ctx context.Context, productID string) (int, error) {
	var productFromDB model.Product
	err := s.db.First(&productFromDB, productID).Error
	if err != nil {
		return 0, err
	}
	return int(productFromDB.Stock), nil
}

func (s *ProductDBRepo) AddProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	var currentStock int
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 先查詢當前庫存
		var product model.Product
		if err := tx.WithContext(ctx).Where("product_id = ?", productID).First(&product).Error; err != nil {
			return err
		}

		// 更新庫存
		if err := tx.WithContext(ctx).Model(&model.Product{}).
			Where("product_id = ?", productID).
			Update("stock", gorm.Expr("stock + ?", quantity)).Error; err != nil {
			return err
		}

		currentStock = int(product.Stock) + int(quantity)
		return nil
	})

	if err != nil {
		return 0, err
	}
	return currentStock, nil
}

func (s *ProductDBRepo) DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	var currentStock int
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 先查詢當前庫存
		var product model.Product
		if err := tx.WithContext(ctx).Where("product_id = ?", productID).First(&product).Error; err != nil {
			return err
		}

		// 檢查庫存是否足夠
		if product.Stock < quantity {
			return ErrProductStockNotEnough
		}

		// 更新庫存
		if err := tx.WithContext(ctx).Model(&model.Product{}).
			Where("product_id = ?", productID).
			Update("stock", gorm.Expr("stock - ?", quantity)).Error; err != nil {
			return err
		}

		currentStock = int(product.Stock) - int(quantity)
		return nil
	})

	if err != nil {
		return 0, err
	}
	return currentStock, nil
}

// convertRedisMapToProduct 將 Redis 的 map[string]string 轉換為 model.Product

func (s *ProductDBRepo) GetProductByCode(ctx context.Context, code string) (*model.Product, error) {
	var product model.Product
	err := s.db.Where("code = ?", code).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (s *ProductDBRepo) GetAllProducts(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Find(&products).Error
	if err != nil {
		return nil, err
	}

	return products, err
}

// Read - 查詢有庫存的商品
func (s *ProductDBRepo) GetProductsInStock(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Where("stock > 0").Find(&products).Error
	return products, err
}

// Update - 更新商品
func (s *ProductDBRepo) UpdateProduct(ctx context.Context, product *model.Product) error {
	return s.db.Save(product).Error
}

// Update - 部分更新商品

// Update - 更新reserved庫存
func (s *ProductDBRepo) UpdateReserved(ctx context.Context, id string, reserved uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先鎖定記錄
		var product model.Product
		if err := tx.Set("gorm:for_update", true).
			Where("product_id = ?", id).
			First(&product).Error; err != nil {
			return err
		}

		// 更新 reserved 值
		return tx.Model(&model.Product{}).
			Where("product_id = ?", id).
			Update("reserved", reserved).Error
	})
}

// Update - 更新庫存
func (s *ProductDBRepo) UpdateStock(ctx context.Context, id string, stock uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先鎖定記錄
		var product model.Product
		if err := tx.Set("gorm:for_update", true).
			Where("product_id = ?", id).
			First(&product).Error; err != nil {
			return err
		}

		// 更新 stock 值
		return tx.Model(&model.Product{}).
			Where("product_id = ?", id).
			Update("stock", stock).Error
	})
}

// Delete - 硬刪除商品
func (s *ProductDBRepo) HardDeleteProduct(ctx context.Context, id string) error {
	return s.db.Unscoped().Delete(&model.Product{}, id).Error
}

// 分頁查詢商品
func (s *ProductDBRepo) GetProductsPaginated(ctx context.Context, page, pageSize int) ([]model.Product, int64, error) {
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
func (s *ProductDBRepo) GetProductsPaginatedWithCondition(ctx context.Context, page, pageSize int, condition map[string]interface{}) ([]model.Product, int64, error) {
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
func (s *ProductDBRepo) CreateProductsBatch(ctx context.Context, products []model.Product) error {
	return s.db.Create(&products).Error
}

// 批量更新商品價格
func (s *ProductDBRepo) UpdateProductsPriceBatch(ctx context.Context, category string, priceMultiplier float64) error {
	return s.db.Model(&model.Product{}).Where("category = ?", category).Update("price", gorm.Expr("price * ?", priceMultiplier)).Error
}
