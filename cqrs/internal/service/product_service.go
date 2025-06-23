package service

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
)

type ProductServiceError error

var (
	errProductNotFound       ProductServiceError = errors.New("product_not_found")
	errProductStockNotEnough ProductServiceError = errors.New("product_stock_not_enough")
)

type ProductService struct {
	productRepo *db.ProductRepo
}

func NewProductService(productRepo *db.ProductRepo) *ProductService {
	return &ProductService{productRepo: productRepo}
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

	if stock < quantity {
		return false, nil
	}

	return true, nil
}

func (s *ProductService) AddProductStock(ctx context.Context, productID string, quantity uint) error {
	return s.productRepo.DeltaProductStock(ctx, productID, int64(quantity))
}

// 扣除庫存
// 會先檢查庫存是否足夠
// 錯誤:
//   - errProductStockNotEnough: 庫存不足
//   - errProductNotFound: 商品不存在
//   - err: 其他錯誤
func (s *ProductService) SubProductStock(ctx context.Context, productID string, quantity uint) error {
	isEnough, err := s.CheckProductStockEnough(ctx, productID, quantity)
	if err != nil {
		return err
	}
	if !isEnough {
		return errProductStockNotEnough
	}
	return s.productRepo.DeltaProductStock(ctx, productID, int64(-quantity))
}
