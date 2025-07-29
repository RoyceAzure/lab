package service

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
)

// IProductService 定義商品服務的介面
type IProductService interface {
	// CreateProduct 創建商品
	// 返回值:
	//   - *model.Product: 創建後的商品
	// 錯誤:
	//   - err: 其他錯誤
	CreateProduct(ctx context.Context, product *model.Product) error

	// GetProduct 取得商品
	// 返回值:
	//   - *model.Product: 商品
	// 錯誤:
	//   - err: 其他錯誤
	GetProduct(ctx context.Context, productID string) (*model.Product, error)

	// CheckProductStockEnough 檢查庫存是否足夠
	// 返回值:
	//   - bool: true 表示庫存足夠，false 表示庫存不足
	// 錯誤:
	//   - errProductNotFound: 商品不存在
	//   - err: 其他錯誤
	CheckProductStockEnough(ctx context.Context, productID string, quantity uint) (bool, error)

	// AddProductStock 增加商品庫存
	// 返回值:
	//   - int: 更新後的庫存數量
	// 錯誤:
	//   - errProductNotFound: 商品不存在
	//   - err: 其他錯誤
	AddProductStock(ctx context.Context, productID string, quantity uint) (int, error)

	// SubProductStock 扣除庫存
	// 會先檢查庫存是否足夠，使用原子操作進行扣減
	// 錯誤:
	//   - errProductStockNotEnough: 庫存不足
	//   - errProductNotFound: 商品不存在
	//   - err: 其他錯誤
	SubProductStock(ctx context.Context, productID string, quantity uint) error

	// GetProductStock 取得商品庫存
	// 返回值:
	//   - int: 商品庫存數量
	// 錯誤:
	//   - errProductNotFound: 商品不存在
	//   - err: 其他錯誤
	GetProductStock(ctx context.Context, productID string) (int, error)

	// DeleteProduct 刪除商品
	// 錯誤:
	//   - errProductNotFound: 商品不存在
	//   - err: 其他錯誤
	DeleteProduct(ctx context.Context, productID string) error
}

type ProductService struct {
	productRepo db.IProductRepository
}

// NewProductService 創建新的商品服務實例
func NewProductService(productRepo db.IProductRepository) *ProductService {
	return &ProductService{productRepo: productRepo}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *model.Product) error {
	return s.productRepo.CreateProduct(ctx, product)
}

func (s *ProductService) GetProduct(ctx context.Context, productID string) (*model.Product, error) {
	return s.productRepo.GetProductByID(ctx, productID)
}

// 檢查庫存是否足夠
// 錯誤:
//   - errProductNotFound: 商品不存在
//   - err: 其他錯誤
func (s *ProductService) CheckProductStockEnough(ctx context.Context, productID string, quantity uint) (bool, error) {
	stock, err := s.productRepo.GetProductStock(ctx, productID)
	if err != nil {
		return false, err
	}

	if stock < int(quantity) {
		return false, nil
	}

	return true, nil
}

func (s *ProductService) AddProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	return s.productRepo.AddProductStock(ctx, productID, quantity)
}

// 扣除庫存
// 會先檢查庫存是否足夠
// 錯誤:
//   - errProductStockNotEnough: 庫存不足
//   - errProductNotFound: 商品不存在
//   - err: 其他錯誤
func (s *ProductService) SubProductStock(ctx context.Context, productID string, quantity uint) error {
	_, err := s.productRepo.DeductProductStock(ctx, productID, quantity)
	return err
}

func (s *ProductService) GetProductStock(ctx context.Context, productID string) (int, error) {
	return s.productRepo.GetProductStock(ctx, productID)
}

func (s *ProductService) DeleteProduct(ctx context.Context, productID string) error {
	return s.productRepo.HardDeleteProduct(ctx, productID)
}

var _ IProductService = (*ProductService)(nil)
