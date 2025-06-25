package db

import (
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
)

// 購物車階段 只會寫入到redis, 不會寫入到db，所有購物車資料都要去redis取
type OrderRepo struct {
	db *DbDao
}

func NewOrderRepo(db *DbDao) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create - 創建訂單 db
func (s *OrderRepo) CreateOrder(order *model.Order) error {
	return s.db.Create(order).Error
}

// Read - 根據ID查詢訂單
func (s *OrderRepo) GetOrderByID(id string) (*model.Order, error) {
	var order model.Order
	err := s.db.Preload("OrderItems").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Read - 根據用戶ID查詢訂單
func (s *OrderRepo) GetOrdersByUserID(userID int) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").Where("user_id = ?", userID).Find(&orders).Error
	return orders, err
}

// Read - 查詢所有訂單
func (s *OrderRepo) GetAllOrders() ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").Find(&orders).Error
	return orders, err
}

// Read - 根據日期範圍查詢訂單
func (s *OrderRepo) GetOrdersByDateRange(startDate, endDate model.Order) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").
		Where("order_date BETWEEN ? AND ?", startDate, endDate).
		Find(&orders).Error
	return orders, err
}

// Read - 根據金額範圍查詢訂單
func (s *OrderRepo) GetOrdersByAmountRange(minAmount, maxAmount float64) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Preload("OrderItems").
		Where("amount BETWEEN ? AND ?", minAmount, maxAmount).
		Find(&orders).Error
	return orders, err
}

// Update - 更新訂單
func (s *OrderRepo) UpdateOrder(order *model.Order) error {
	return s.db.Save(order).Error
}

// Update - 部分更新訂單
func (s *OrderRepo) UpdateOrderFields(id string, updates map[string]interface{}) error {
	return s.db.Model(&model.Order{}).Where("order_id = ?", id).Updates(updates).Error
}

// Update - 更新訂單金額
func (s *OrderRepo) UpdateOrderAmount(id string, amount float64) error {
	return s.db.Model(&model.Order{}).Where("order_id = ?", id).Update("amount", amount).Error
}

// Delete - 軟刪除訂單
func (s *OrderRepo) DeleteOrder(id string) error {
	return s.db.Delete(&model.Order{}, id).Error
}

// Delete - 硬刪除訂單
func (s *OrderRepo) HardDeleteOrder(id string) error {
	return s.db.Unscoped().Delete(&model.Order{}, id).Error
}

// 分頁查詢訂單
func (s *OrderRepo) GetOrdersPaginated(page, pageSize int) ([]model.Order, int64, error) {
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
func (s *OrderRepo) GetOrdersPaginatedWithCondition(page, pageSize int, condition map[string]interface{}) ([]model.Order, int64, error) {
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
func (s *OrderRepo) CreateOrdersBatch(orders []model.Order) error {
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
func (s *OrderRepo) GetTopSellingProducts(limit int) ([]model.Product, error) {
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
func (s *OrderRepo) GetOrderItems(orderID uint) ([]model.OrderItem, error) {
	var items []model.OrderItem
	err := s.db.Where("order_id = ?", orderID).Find(&items).Error
	return items, err
}

// 新增訂單項目
func (s *OrderRepo) AddOrderItem(item *model.OrderItem) error {
	return s.db.Create(item).Error
}

// 更新訂單項目數量
func (s *OrderRepo) UpdateOrderItemQuantity(orderID, productID uint, quantity int) error {
	return s.db.Model(&model.OrderItem{}).
		Where("order_id = ? AND product_id = ?", orderID, productID).
		Update("quantity", quantity).Error
}

// 刪除訂單項目
func (s *OrderRepo) DeleteOrderItem(orderID, productID uint) error {
	return s.db.Where("order_id = ? AND product_id = ?", orderID, productID).
		Delete(&model.OrderItem{}).Error
}
