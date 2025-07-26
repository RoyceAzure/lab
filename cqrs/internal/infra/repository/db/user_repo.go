package db

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
)

type UserRepo struct {
	dbDao *DbDao
}

func NewUserRepo(dbDao *DbDao) *UserRepo {
	return &UserRepo{dbDao: dbDao}
}

// Create - 創建用戶
func (s *UserRepo) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := s.dbDao.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// Read - 根據ID查詢用戶
func (s *UserRepo) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	var user model.User
	err := s.dbDao.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Read - 查詢所有用戶
func (s *UserRepo) GetAllUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User
	err := s.dbDao.Find(&users).Error
	return users, err
}

func (s *UserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := s.dbDao.Where("user_email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update - 更新用戶
func (s *UserRepo) UpdateUser(ctx context.Context, user *model.User) error {
	return s.dbDao.Save(user).Error
}

// Delete - 軟刪除用戶
func (s *UserRepo) DeleteUser(ctx context.Context, id int) error {
	return s.dbDao.Delete(&model.User{}, id).Error
}

// Delete - 硬刪除用戶
func (s *UserRepo) HardDeleteUser(ctx context.Context, id int) error {
	return s.dbDao.Unscoped().Delete(&model.User{}, id).Error
}
