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
	UpdateUserAccountAndPassword(ctx context.Context, email string, account string, password string) error
	ActiveUser(ctx context.Context, id uuid.UUID) error
	CreateUser(ctx context.Context, arg *model.UserModel) (*model.UserModel, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.UserModel, error)
	// CreateUserWithRole 創建用戶並分配角色
	// 參數:
	//   - ctx: 上下文
	//   - user: 用戶資訊
	//
	// 錯誤:
	//   - er.InternalErrorCode 500: 內部處理錯誤
	//   - er.InvalidArgumentCode 460: email已經存在
	CreateUserWithRole(ctx context.Context, user *model.CreateUserWithRoleModel) error
	// CheckUserExistsByEmail 檢查用戶是否存在
	// 回傳:
	//   - true: 存在
	//   - false: 不存在
	//   - error: 錯誤
	CheckUserExistsByEmail(ctx context.Context, email string) (bool, error)
	// GetUserByAccountAndPassword 根據帳號和密碼獲取用戶信息
	// 如果帳號不存在，會返回401錯誤
	// 參數:
	//   - ctx: 上下文
	//   - account: 帳號
	//   - password: 密碼明文
	//
	// 回傳:
	//   - user: 用戶信息
	//   - error: 錯誤
	//
	// 錯誤:
	//   - er.UnauthenticatedCode 401: 帳號或密碼錯誤
	//   - er.InvalidOperationCode 460: 帳號未綁定密碼
	//   - er.InternalErrorCode 500: 內部處理錯誤
	GetUserByAccountAndPassword(ctx context.Context, account, password string) (*model.UserModel, error)
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
	if err != nil {
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return convertRepoUsertToModel(&userEntity), nil
}

// GetUserByEmail 根據email獲取用戶信息
// 找不到用戶時，error 會是 nil, user 會是 nil
// 回傳:
//   - user: 用戶信息
//   - error: 錯誤
func (u *UserService) GetUserByEmail(ctx context.Context, email string) (*model.UserModel, error) {
	userEntity, err := u.dbDao.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
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
func (u *UserService) UpdateUserAccountAndPassword(ctx context.Context, email string, account string, password string) error {
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
		Email:        email,
		Account:      pgutil.StringToPgTextV5(&account),
		PasswordHash: pgutil.StringToPgTextV5(&hashPassword),
	})
	if err != nil {
		return er.New(er.InternalErrorCode, err.Error())
	}
	return nil
}

// CreateUserWithRole 創建用戶並分配角色
// 參數:
//   - ctx: 上下文
//   - user: 用戶資訊
//
// 錯誤:
//   - er.InternalErrorCode 500: 內部處理錯誤
//   - er.InvalidArgumentCode 460: email已經存在
func (u *UserService) CreateUserWithRole(ctx context.Context, user *model.CreateUserWithRoleModel) error {
	// 檢查email是否已存在
	existingUser, err := u.CheckUserExistsByEmail(ctx, user.Email)
	if err != nil {
		return err
	}
	if existingUser {
		return er.New(er.InvalidArgumentCode, "email already exists")
	}

	userEntity := sqlc.CreateUserParams{
		ID:        pgutil.UUIDToPgUUIDV5(user.ID),
		Email:     user.Email,
		IsAdmin:   user.IsAdmin,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
	roleEntity := sqlc.AssignRoleToUserParams{
		UserID: pgutil.UUIDToPgUUIDV5(user.ID),
		RoleID: user.RoleID,
	}

	err = u.dbDao.ExecTx(ctx, func(q *sqlc.Queries) error {
		_, err := q.CreateUser(ctx, userEntity)
		if err != nil {
			return err
		}

		err = q.AssignRoleToUser(ctx, roleEntity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return er.New(er.InternalErrorCode, "create user failed")
	}

	return nil
}

// CheckUserExistsByEmail 檢查用戶是否存在
// 回傳:
//   - true: 存在
//   - false: 不存在
//   - error: 錯誤
func (u *UserService) CheckUserExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := u.dbDao.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

// GetUserByAccountAndPassword 根據帳號和密碼獲取用戶信息
// 如果帳號不存在，會返回401錯誤
// 參數:
//   - ctx: 上下文
//   - account: 帳號
//   - password: 密碼明文
//
// 回傳:
//   - user: 用戶信息
//   - error: 錯誤
//
// 錯誤:
//   - er.UnauthenticatedCode 401: 帳號或密碼錯誤
//   - er.InvalidOperationCode 460: 帳號未綁定密碼
//   - er.InternalErrorCode 500: 內部處理錯誤
func (u *UserService) GetUserByAccountAndPassword(ctx context.Context, account, password string) (*model.UserModel, error) {
	userEntity, err := u.dbDao.GetUserByAccount(ctx, pgutil.StringToPgTextV5(&account))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, er.New(er.NotFoundCode, "account not found")
		}
		return nil, err
	}

	if !userEntity.IsActive {
		return nil, er.New(er.UnauthenticatedCode, "account is not active")
	}

	hashPassword := pgutil.PgTextToStringV5(userEntity.PasswordHash)
	if hashPassword == nil {
		return nil, er.New(er.InvalidOperationCode, "account is not linked with password")
	}

	err = crypt.CheckPassword(password, *hashPassword)
	if err != nil {
		return nil, er.New(er.UnauthenticatedCode, "帳號或密碼錯誤")
	}

	return convertRepoUsertToModel(&userEntity), nil
}
