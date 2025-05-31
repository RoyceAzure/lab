package service

import (
	"context"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	er "github.com/RoyceAzure/rj/util/rj_error"
)

type IRoleService interface {
	// CreateRole 創建新角色
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的角色參數
	CreateRole(ctx context.Context, arg model.RoleModel) (*model.RoleModel, error)
	// CreateRole 創建新角色 if not exists
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的角色參數
	CreateRoleIfNotExists(ctx context.Context, arg model.RoleModel) error
}

// RoleService 實現角色服務
type RoleService struct {
	dbDao db.IStore
}

// NewRoleService 創建新的角色服務實例
func NewRoleService(dbDao db.IStore) IRoleService {
	if dbDao == nil {
		panic("role service missing required dependency db dao")
	}

	return &RoleService{
		dbDao: dbDao,
	}
}

// 將服務層模型轉換為儲存庫參數
func convertRoleModelToRepo(arg model.RoleModel) sqlc.CreateRoleParams {
	return sqlc.CreateRoleParams{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.StringToPgTextV5(arg.Description),
	}
}

// 將儲存庫模型轉換為服務層模型
func convertRoleRepoToModel(arg sqlc.Role) model.RoleModel {
	return model.RoleModel{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.PgTextToStringV5(arg.Description),
	}
}

// CreateRole 創建新角色
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的角色參數
func (r *RoleService) CreateRole(ctx context.Context, arg model.RoleModel) (*model.RoleModel, error) {
	// 驗證必要參數
	if arg.Name == "" {
		return nil, er.New(er.BadRequestCode, "角色名稱不能為空")
	}

	roleEntity, err := r.dbDao.CreateRole(ctx, convertRoleModelToRepo(arg))
	if err != nil {
		return nil, er.New(er.InternalErrorCode, "創建角色失敗:"+err.Error())
	}

	roleModel := convertRoleRepoToModel(roleEntity)
	return &roleModel, nil
}

// 將服務層模型轉換為儲存庫參數
func convertRoleModelToNotExistsRepo(arg model.RoleModel) sqlc.CreateRoleIfNotExistsParams {
	return sqlc.CreateRoleIfNotExistsParams{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.StringToPgTextV5(arg.Description),
	}
}

// CreateRole 創建新角色 if not exists
// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的角色參數
func (r *RoleService) CreateRoleIfNotExists(ctx context.Context, arg model.RoleModel) error {
	// 驗證必要參數
	if arg.Name == "" {
		return er.New(er.BadRequestCode, "角色名稱不能為空")
	}

	err := r.dbDao.CreateRoleIfNotExists(ctx, convertRoleModelToNotExistsRepo(arg))
	if err != nil {
		return er.New(er.InternalErrorCode, "創建角色失敗:"+err.Error())
	}

	return nil
}
