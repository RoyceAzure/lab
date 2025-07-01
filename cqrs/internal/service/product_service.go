package service

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
)

type ProductService struct {
	productRepo *redis_repo.ProductRepo
}

func NewProductService(productRepo *redis_repo.ProductRepo) *ProductService {
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

// 將購物車的商品資訊轉換為訂單的商品資訊
// func (s *ProductService) CacheOrderToOrderItemData(cartCache *model.Cart) []model.OrderItemData {
// 	orderItems := make([]model.OrderItemData, len(cartCache.OrderItems))
// 	for i, item := range cartCache.OrderItems {
// 		product, err := s.productRepo.GetProductStock(ctx, item.ProductID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		orderItems[i] = model.OrderItemData{
// 			ProductID: item.ProductID,
// 			Quantity:  item.Quantity,
// 		}
// 	}
// 	return orderItems
// }
