package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	"github.com/RoyceAzure/rj/util/crypt"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	er "github.com/RoyceAzure/rj/util/rj_error"
	"github.com/google/uuid"
)

type IUserService interface {
	GetUserByEmail(ctx context.Context, email string) (*model.UserModel, error)
	UpdateUserAccountAndPassword(ctx context.Context, id uuid.UUID, account string, password string) error
	ActiveUser(ctx context.Context, id uuid.UUID) error
	CreateUser(ctx context.Context, arg *model.UserModel) (*model.UserModel, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.UserModel, error)
}

type UserService struct {
	dbDao db.IStore
}

func NewUserService(dbDao db.IStore) IUserService {
	return &UserService{
		dbDao: dbDao,
	}
}

func (u *UserService) CreateUser(ctx context.Context, arg *model.UserModel) (*model.UserModel, error) {
	// 檢查email是否已存在
	existingUser, err := u.GetUserByEmail(ctx, arg.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	userEntity, err := u.dbDao.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           pgutil.UUIDToPgUUIDV5(arg.ID),
		Email:        arg.Email,
		IsAdmin:      arg.IsAdmin,
		IsActive:     arg.IsActive,
		GoogleID:     pgutil.StringToPgTextV5(arg.GoogleID),
		FacebookID:   pgutil.StringToPgTextV5(arg.FacebookID),
		Name:         pgutil.StringToPgTextV5(arg.Name),
		LineUserID:   pgutil.StringToPgTextV5(arg.LineUserID),
		LineLinkedAt: pgutil.TimeToPgTimestamptzV5(arg.LineLinkedAt),
		CreatedAt:    arg.CreatedAt,
		Account:      pgutil.StringToPgTextV5(arg.Account),
		PasswordHash: pgutil.StringToPgTextV5(arg.HashPassword),
	})

	return convertRepoUsertToModel(&userEntity), nil
}

func (u *UserService) GetUserByEmail(ctx context.Context, email string) (*model.UserModel, error) {
	userEntity, err := u.dbDao.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return convertRepoUsertToModel(&userEntity), nil
}

func (u *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.UserModel, error) {
	userEntity, err := u.dbDao.GetUserByID(ctx, pgutil.UUIDToPgUUIDV5(id))
	if err != nil {
		return nil, err
	}

	return convertRepoUsertToModel(&userEntity), nil
}

func (u *UserService) CheckUserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := u.dbDao.GetUserByID(ctx, pgutil.UUIDToPgUUIDV5(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (u *UserService) ActiveUser(ctx context.Context, id uuid.UUID) error {
	err := u.dbDao.ActiveUser(ctx, pgutil.UUIDToPgUUIDV5(id))
	if err != nil {
		return err
	}
	return nil
}

// UpdateUserAccountAndPassword 更新用戶帳號和密碼
// 參數:
//   - ctx: 上下文
//   - id: 用戶ID
//   - account: 帳號
//   - password: 密碼明文
//
// 錯誤:
//   - er.InternalErrorCode 500: 內部處理錯誤
//   - er.InvalidArgumentCode 460: 帳號或密碼為空
//   - er.InvalidArgumentCode 460: 密碼長度不足
//   - er.InvalidArgumentCode 460: 密碼強度不足
func (u *UserService) UpdateUserAccountAndPassword(ctx context.Context, id uuid.UUID, account string, password string) error {
	if account == "" || password == "" {
		return er.New(er.InvalidArgumentCode, "account or password is empty")
	}

	err := crypt.ValidateStringPassword(password)
	if err != nil {
		return er.New(er.InvalidArgumentCode, err.Error())
	}

	hashPassword, err := crypt.HashPassword(password)
	if err != nil {
		return er.New(er.InternalErrorCode, err.Error())
	}

	err = u.dbDao.UpdateUserAccountAndPassword(ctx, sqlc.UpdateUserAccountAndPasswordParams{
		ID:           pgutil.UUIDToPgUUIDV5(id),
		Account:      pgutil.StringToPgTextV5(&account),
		PasswordHash: pgutil.StringToPgTextV5(&hashPassword),
	})
	if err != nil {
		return er.New(er.InternalErrorCode, err.Error())
	}
	return nil
}

// 將 repository 模型轉換為服務層模型
func convertRepoUsertToModel(u *sqlc.User) *model.UserModel {
	return &model.UserModel{
		ID:           u.ID.Bytes,
		Email:        u.Email,
		IsAdmin:      u.IsAdmin,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		Name:         pgutil.PgTextToStringV5(u.Name),
		Account:      pgutil.PgTextToStringV5(u.Account),
		HashPassword: pgutil.PgTextToStringV5(u.PasswordHash),
		GoogleID:     pgutil.PgTextToStringV5(u.GoogleID),
		FacebookID:   pgutil.PgTextToStringV5(u.FacebookID),
		LineUserID:   pgutil.PgTextToStringV5(u.LineUserID),
		LineLinkedAt: pgutil.PgTimestamptzToTimeV5(u.LineLinkedAt),
	}
}

// 將服務層模型轉換為 repository 模型
func convertUserModelToRepo(m *model.UserModel) *sqlc.User {
	return &sqlc.User{
		ID:           pgutil.UUIDToPgUUIDV5(m.ID),
		Email:        m.Email,
		IsAdmin:      m.IsAdmin,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		Name:         pgutil.StringToPgTextV5(m.Name),
		Account:      pgutil.StringToPgTextV5(m.Account),
		PasswordHash: pgutil.StringToPgTextV5(m.HashPassword),
		GoogleID:     pgutil.StringToPgTextV5(m.GoogleID),
		FacebookID:   pgutil.StringToPgTextV5(m.FacebookID),
		LineUserID:   pgutil.StringToPgTextV5(m.LineUserID),
		LineLinkedAt: pgutil.TimeToPgTimestamptzV5(m.LineLinkedAt),
	}
}
