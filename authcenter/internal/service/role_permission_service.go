package service

import (
	"context"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	er "github.com/RoyceAzure/rj/util/rj_error"
)

// IRolePermissionService 定義角色權限服務介面
type IRolePermissionService interface {
	// AssignPermissionToRole 將權限指派給角色
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的參數
	AssignPermissionToRole(ctx context.Context, arg model.RolePermissionModel) error
	// AssignPermissionToRole 將權限指派給角色
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的參數
	AssignRolePermissionIfNotExists(ctx context.Context, arg model.RolePermissionModel) error
}

// RolePermissionService 實現角色權限服務
type RolePermissionService struct {
	dbDao db.IStore
}

// NewRolePermissionService 創建新的角色權限服務實例
func NewRolePermissionService(dbDao db.IStore) IRolePermissionService {
	if dbDao == nil {
		panic("role_permission service missing required dependency db dao")
	}
	return &RolePermissionService{
		dbDao: dbDao,
	}
}

// 將服務層模型轉換為儲存庫參數
func convertRolePermissionModelToRepo(arg model.RolePermissionModel) sqlc.AssignPermissionToRoleParams {
	return sqlc.AssignPermissionToRoleParams{
		RoleID:       arg.RoleID,
		PermissionID: arg.PermissionID,
	}
}

// AssignPermissionToRole 將權限指派給角色
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
func (rp *RolePermissionService) AssignPermissionToRole(ctx context.Context, arg model.RolePermissionModel) error {
	// 驗證必要參數
	if arg.RoleID <= 0 {
		return er.New(er.BadRequestCode, "角色ID無效")
	}
	if arg.PermissionID <= 0 {
		return er.New(er.BadRequestCode, "權限ID無效")
	}

	err := rp.dbDao.AssignPermissionToRole(ctx, convertRolePermissionModelToRepo(arg))
	if err != nil {
		return er.New(er.InternalErrorCode, "指派權限給角色失敗:"+err.Error())
	}

	return nil
}

// 將服務層模型轉換為儲存庫參數
func convertRolePermissionModelToNotExistsRepo(arg model.RolePermissionModel) sqlc.CreateRolePermissionIfNotExistsParams {
	return sqlc.CreateRolePermissionIfNotExistsParams{
		RoleID:       arg.RoleID,
		PermissionID: arg.PermissionID,
	}
}

// AssignPermissionToRole 將權限指派給角色
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
func (rp *RolePermissionService) AssignRolePermissionIfNotExists(ctx context.Context, arg model.RolePermissionModel) error {
	// 驗證必要參數
	if arg.RoleID <= 0 {
		return er.New(er.BadRequestCode, "角色ID無效")
	}
	if arg.PermissionID <= 0 {
		return er.New(er.BadRequestCode, "權限ID無效")
	}

	err := rp.dbDao.CreateRolePermissionIfNotExists(ctx, convertRolePermissionModelToNotExistsRepo(arg))
	if err != nil {
		return er.New(er.InternalErrorCode, "指派權限給角色失敗:"+err.Error())
	}

	return nil
}
