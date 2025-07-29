package db

import (
	"context"
	"fmt"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 購物車階段 只會寫入到redis, 不會寫入到db，所有購物車資料都要去redis取
type OrderRepo struct {
	db *DbDao
}

func NewOrderRepo(db *DbDao) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create - 創建訂單 db
func (s *OrderRepo) CreateOrder(ctx context.Context, order *model.Order) error {
	return s.db.Create(order).Error
}

// Read - 根據ID查詢訂單
func (s *OrderRepo) GetOrderByID(ctx context.Context, id string) (*model.Order, error) {
	var order model.Order
	err := s.db.Preload("OrderItems").First(&order, "order_id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Read - 根據用戶ID查詢訂單
func (s *OrderRepo) GetOrdersByUserID(ctx context.Context, userID int) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").Where("user_id = ?", userID).Find(&orders).Error
	return orders, err
}

// Read - 查詢所有訂單
func (s *OrderRepo) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").Find(&orders).Error
	return orders, err
}

// Read - 根據日期範圍查詢訂單
func (s *OrderRepo) GetOrdersByDateRange(ctx context.Context, startDate, endDate model.Order) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").
		Where("order_date BETWEEN ? AND ?", startDate, endDate).
		Find(&orders).Error
	return orders, err
}

// Read - 根據金額範圍查詢訂單
func (s *OrderRepo) GetOrdersByAmountRange(ctx context.Context, minAmount, maxAmount float64) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").
		Where("amount BETWEEN ? AND ?", minAmount, maxAmount).
		Find(&orders).Error
	return orders, err
}

// Update - 更新訂單
func (s *OrderRepo) UpdateOrder(ctx context.Context, order *model.Order) error {
	return s.db.Save(order).Error
}

// Update - 更新訂單狀態
func (s *OrderRepo) UpdateOrderState(ctx context.Context, id string, state uint) error {
	return s.db.Model(&model.Order{}).Where("order_id = ?", id).Update("state", state).Error
}

// Update - 更新訂單金額
func (s *OrderRepo) UpdateOrderAmount(ctx context.Context, id string, amount float64) error {
	return s.db.Model(&model.Order{}).Where("order_id = ?", id).Update("amount", amount).Error
}

// Delete - 軟刪除訂單
func (s *OrderRepo) DeleteOrder(ctx context.Context, id string) error {
	return s.db.Where("order_id = ?", id).Delete(&model.Order{}).Error
}

// Delete - 硬刪除訂單
func (s *OrderRepo) HardDeleteOrder(ctx context.Context, id string) error {
	return s.db.Unscoped().Where("order_id = ?", id).Delete(&model.Order{}).Error
}

// 分頁查詢訂單
func (s *OrderRepo) GetOrdersPaginated(ctx context.Context, page, pageSize int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	offset := (page - 1) * pageSize

	// 計算總數
	s.db.Model(&model.Order{}).Count(&total)

	// 分頁查詢
	err := s.db.Preload("OrderItems").Offset(offset).Limit(pageSize).Find(&orders).Error

	return orders, total, err
}

// 根據條件分頁查詢
func (s *OrderRepo) GetOrdersPaginatedWithCondition(ctx context.Context, page, pageSize int, condition map[string]interface{}) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	offset := (page - 1) * pageSize
	query := s.db.Model(&model.Order{})

	// 應用條件
	for key, value := range condition {
		query = query.Where(key+" = ?", value)
	}

	// 計算總數
	query.Count(&total)

	// 分頁查詢
	err := query.Preload("OrderItems").Offset(offset).Limit(pageSize).Find(&orders).Error

	return orders, total, err
}

// 批量創建訂單
func (s *OrderRepo) CreateOrdersBatch(ctx context.Context, orders []model.Order) error {
	return s.db.Create(&orders).Error
}

// 取得用戶的訂單統計
func (s *OrderRepo) GetUserOrderStats(userID int) (float64, int, error) {
	var totalAmount float64
	var orderCount int64

	err := s.db.Model(&model.Order{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(amount), 0) as total_amount, COUNT(*) as order_count").
		Row().
		Scan(&totalAmount, &orderCount)

	return totalAmount, int(orderCount), err
}

// 取得熱門商品（根據訂單項目統計）
func (s *OrderRepo) GetTopSellingProducts(ctx context.Context, limit int) ([]model.Product, error) {
	var products []model.Product
	err := s.db.Model(&model.Product{}).
		Joins("JOIN order_items ON products.product_id = order_items.product_id").
		Group("products.product_id").
		Order("SUM(order_items.quantity) DESC").
		Limit(limit).
		Find(&products).Error
	return products, err
}

// 取得訂單項目
func (s *OrderRepo) GetOrderItems(ctx context.Context, orderID uint) ([]model.OrderItem, error) {
	var items []model.OrderItem
	err := s.db.Where("order_id = ?", orderID).Find(&items).Error
	return items, err
}

// 新增訂單項目
func (s *OrderRepo) AddOrderItem(ctx context.Context, item *model.OrderItem) error {
	return s.db.Create(item).Error
}

// 更新訂單項目數量
func (s *OrderRepo) UpdateOrderItemQuantity(ctx context.Context, orderID, productID uint, quantity int) error {
	return s.db.Model(&model.OrderItem{}).
		Where("order_id = ? AND product_id = ?", orderID, productID).
		Update("quantity", quantity).Error
}

// 刪除訂單項目
func (s *OrderRepo) DeleteOrderItem(ctx context.Context, orderID, productID uint) error {
	return s.db.Where("order_id = ? AND product_id = ?", orderID, productID).
		Delete(&model.OrderItem{}).Error
}

// BatchUpdateResult 批次更新結果
type BatchUpdateResult struct {
	SuccessOrders []string         // 成功更新的訂單ID
	FailedOrders  map[string]error // 失敗的訂單ID及其錯誤原因
	TotalCount    int              // 總處理數量
	SuccessCount  int              // 成功數量
	FailCount     int              // 失敗數量
}

// 批量更新訂單
func (s *OrderRepo) UpdateOrdersBatch(ctx context.Context, orders []*model.Order) (*BatchUpdateResult, error) {
	if len(orders) == 0 {
		return &BatchUpdateResult{
			SuccessOrders: make([]string, 0),
			FailedOrders:  make(map[string]error),
			TotalCount:    0,
			SuccessCount:  0,
			FailCount:     0,
		}, nil
	}

	result := &BatchUpdateResult{
		SuccessOrders: make([]string, 0, len(orders)),
		FailedOrders:  make(map[string]error),
		TotalCount:    len(orders),
		SuccessCount:  0,
		FailCount:     0,
	}

	// 使用事務來確保資料一致性
	err := s.db.DB.Transaction(func(tx *gorm.DB) error {
		// 準備批次更新的訂單 ID 列表
		orderIDs := make([]string, len(orders))
		for i, order := range orders {
			orderIDs[i] = order.OrderID
		}

		// 使用 Upsert 批次更新訂單基本資訊
		for _, order := range orders {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "order_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"user_id", "amount", "order_date", "state"}),
			}).Create(order).Error; err != nil {
				// 記錄失敗的訂單
				result.FailedOrders[order.OrderID] = err
				result.FailCount++
				// 記錄錯誤但繼續處理其他訂單
				continue
			}
			// 記錄成功的訂單
			result.SuccessOrders = append(result.SuccessOrders, order.OrderID)
			result.SuccessCount++
		}

		// 處理訂單項目
		successOrderMap := make(map[string]struct{})
		for _, orderID := range result.SuccessOrders {
			successOrderMap[orderID] = struct{}{}
		}

		// 收集所有需要檢查的訂單項目
		var itemsToCheck []model.OrderItem
		for _, order := range orders {
			if _, ok := successOrderMap[order.OrderID]; ok && len(order.OrderItems) > 0 {
				itemsToCheck = append(itemsToCheck, order.OrderItems...)
			}
		}

		if len(itemsToCheck) > 0 {
			// 批次檢查訂單項目是否存在
			var existingItems []struct {
				OrderID   string
				ProductID string
			}
			if err := tx.Model(&model.OrderItem{}).
				Select("order_id, product_id").
				Where("(order_id, product_id) IN ?", generateOrderItemPKPairs(itemsToCheck)).
				Find(&existingItems).Error; err != nil {
				return err
			}

			// 建立已存在項目的映射
			existingMap := make(map[string]struct{})
			for _, item := range existingItems {
				key := fmt.Sprintf("%s-%s", item.OrderID, item.ProductID)
				existingMap[key] = struct{}{}
			}

			// 使用 Upsert 處理訂單項目
			for _, item := range itemsToCheck {
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "order_id"}, {Name: "product_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"quantity"}),
				}).Create(&item).Error; err != nil {
					// 如果訂單項目插入失敗，將相關訂單標記為失敗
					if _, exists := result.FailedOrders[item.OrderID]; !exists {
						result.FailedOrders[item.OrderID] = fmt.Errorf("failed to create order items: %v", err)
						result.FailCount++
						result.SuccessCount--
						// 從成功列表中移除
						for j, successID := range result.SuccessOrders {
							if successID == item.OrderID {
								result.SuccessOrders = append(result.SuccessOrders[:j], result.SuccessOrders[j+1:]...)
								break
							}
						}
					}
					// 繼續處理下一個項目
					continue
				}
			}
		}

		// 如果所有訂單都失敗，返回錯誤
		if result.FailCount == result.TotalCount {
			return fmt.Errorf("all orders failed to update")
		}

		return nil
	})

	if err != nil {
		// 如果事務失敗，將所有未記錄的訂單標記為失敗
		for _, order := range orders {
			if _, exists := result.FailedOrders[order.OrderID]; !exists {
				result.FailedOrders[order.OrderID] = err
				result.FailCount++
			}
		}
		result.SuccessOrders = nil
		result.SuccessCount = 0
	}

	return result, err
}

// generateOrderItemPKPairs 生成訂單項目主鍵對的列表
func generateOrderItemPKPairs(items []model.OrderItem) [][]interface{} {
	pairs := make([][]interface{}, len(items))
	for i, item := range items {
		pairs[i] = []interface{}{item.OrderID, item.ProductID}
	}
	return pairs
}
