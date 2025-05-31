package service

import (
	"context"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	er "github.com/RoyceAzure/rj/util/rj_error"
)

// AssignRoleToUser 將角色指派給使用者
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
type IUserRoleService interface {
	AssignRoleToUser(ctx context.Context, arg model.UserRoleModel) error
}

// UserRoleService 實現使用者角色服務
type UserRoleService struct {
	dbDao db.IStore
}

// NewUserRoleService 創建新的使用者角色服務實例
func NewUserRoleService(dbDao db.IStore) IUserRoleService {
	if dbDao == nil {
		panic("user_role service missing required dependency db dao")
	}

	return &UserRoleService{
		dbDao: dbDao,
	}
}

// 將服務層模型轉換為儲存庫參數
func convertUserRoleModelToRepo(arg model.UserRoleModel) sqlc.AssignRoleToUserParams {
	return sqlc.AssignRoleToUserParams{
		UserID: pgutil.UUIDToPgUUIDV5(arg.UserID),
		RoleID: arg.RoleID,
	}
}

// AssignRoleToUser 將角色指派給使用者
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
func (ur *UserRoleService) AssignRoleToUser(ctx context.Context, arg model.UserRoleModel) error {
	// 驗證必要參數
	if arg.RoleID <= 0 {
		return er.New(er.BadRequestCode, "角色ID無效")
	}

	err := ur.dbDao.AssignRoleToUser(ctx, convertUserRoleModelToRepo(arg))
	if err != nil {
		return er.New(er.InternalErrorCode, "指派角色給使用者失敗:"+err.Error())
	}

	return nil
}
