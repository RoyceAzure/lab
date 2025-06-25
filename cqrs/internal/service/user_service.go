package service

import (
	"context"
	"errors"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
)

var (
	ErrUserNotExist = errors.New("user is not exist")
)

type UserService struct {
	userRepo *db.UserRepo
}

func NewUserService(userRepo *db.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
}

func (u *UserService) GetUser(ctx context.Context, userID int) (*model.User, error) {
	user, err := u.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotExist
	}
	return user, nil
}
