package db

import (
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
)

type UserOrderRepo struct {
	dbDao *DbDao
}

func NewUserOrderRepo(dbDao *DbDao) *UserOrderRepo {
	return &UserOrderRepo{dbDao: dbDao}
}

// GetUserOrderByID - 根據 ID 查詢用戶訂單關聯
func (r *UserOrderRepo) GetUserOrderByID(id int) (*model.UserOrder, error) {
	var userOrder model.UserOrder
	err := r.dbDao.First(&userOrder, id).Error
	if err != nil {
		return nil, err
	}
	return &userOrder, nil
}

// ListUserOrders - 查詢所有用戶訂單關聯
func (r *UserOrderRepo) ListUserOrders() ([]model.UserOrder, error) {
	var userOrders []model.UserOrder
	err := r.dbDao.Find(&userOrders).Error
	return userOrders, err
}

// ListUserOrdersByUserID - 根據用戶 ID 查詢訂單關聯
func (r *UserOrderRepo) ListUserOrdersByUserID(userID int) ([]model.UserOrder, error) {
	var userOrders []model.UserOrder
	err := r.dbDao.Where("user_id = ?", userID).Find(&userOrders).Error
	return userOrders, err
}

// CreateUserOrder - 創建用戶訂單關聯
func (r *UserOrderRepo) CreateUserOrder(userOrder *model.UserOrder) (*model.UserOrder, error) {
	if err := r.dbDao.Create(userOrder).Error; err != nil {
		return nil, err
	}
	return userOrder, nil
}

// DeleteUserOrder - 刪除用戶訂單關聯
func (r *UserOrderRepo) DeleteUserOrder(id int) error {
	return r.dbDao.Delete(&model.UserOrder{}, id).Error
}

// HardDeleteUserOrder - 硬刪除用戶訂單關聯
func (r *UserOrderRepo) HardDeleteUserOrder(id int) error {
	return r.dbDao.Unscoped().Delete(&model.UserOrder{}, id).Error
}
