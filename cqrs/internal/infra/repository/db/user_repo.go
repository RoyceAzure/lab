package db

import (
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
)

type UserRepo struct {
	dbDao *DbDao
}

func NewUserRepo(dbDao *DbDao) *UserRepo {
	return &UserRepo{dbDao: dbDao}
}

// Create - 創建用戶
func (s *UserRepo) CreateUser(user *model.User) (*model.User, error) {
	if err := s.dbDao.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// Read - 根據ID查詢用戶
func (s *UserRepo) GetUserByID(id int) (*model.User, error) {
	var user model.User
	err := s.dbDao.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Read - 查詢所有用戶
func (s *UserRepo) GetAllUsers() ([]model.User, error) {
	var users []model.User
	err := s.dbDao.Find(&users).Error
	return users, err
}

// Read - 根據Email查詢用戶
func (s *UserRepo) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := s.dbDao.Where("user_email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update - 更新用戶
func (s *UserRepo) UpdateUser(user *model.User) error {
	return s.dbDao.Save(user).Error
}

// Update - 部分更新用戶
func (s *UserRepo) PatchUserFields(id int, updates map[string]interface{}) error {
	return s.dbDao.Model(&model.User{}).Where("user_id = ?", id).Updates(updates).Error
}

// Delete - 軟刪除用戶
func (s *UserRepo) DeleteUser(id int) error {
	return s.dbDao.Delete(&model.User{}, id).Error
}

// Delete - 硬刪除用戶
func (s *UserRepo) HardDeleteUser(id int) error {
	return s.dbDao.Unscoped().Delete(&model.User{}, id).Error
}

// 分頁查詢
func (s *UserRepo) GetUsersPaginated(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	offset := (page - 1) * pageSize

	// 先計算總數
	s.dbDao.Model(&model.User{}).Count(&total)

	// 分頁查詢
	err := s.dbDao.Offset(offset).Limit(pageSize).Find(&users).Error

	return users, total, err
}
