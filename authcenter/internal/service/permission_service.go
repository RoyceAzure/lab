package service

import (
	"context"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	"github.com/RoyceAzure/lab/authcenter/internal/util"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	er "github.com/RoyceAzure/rj/util/rj_error"
)

type IPermissionService interface {
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的參數
	CreatePermission(ctx context.Context, arg model.PermissionModel) (*model.PermissionModel, error)
	// 錯誤:
	//   - er.InternalErrorCode 500: 資料庫操作錯誤
	//   - er.BadRequestCode 400: 無效的參數
	CreatePermissionIfNotExists(ctx context.Context, arg model.PermissionModel) error
}

type PermissionService struct {
	dbDao db.IStore
}

func NewPermissionService(dbDao db.IStore) IPermissionService {
	if dbDao == nil {
		panic("permission service missing reqired depency db dao")
	}

	return &PermissionService{
		dbDao: dbDao,
	}
}

func convertPermissionModelToRepo(arg model.PermissionModel) sqlc.CreatePermissionParams {
	return sqlc.CreatePermissionParams{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.StringToPgTextV5(arg.Description),
		Resource:    arg.Resource,
		Actions:     arg.Actions,
	}
}

func convertPermissioRepoToModel(arg sqlc.Permission) model.PermissionModel {
	return model.PermissionModel{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.PgTextToStringV5(arg.Description),
		Resource:    arg.Resource,
		Actions:     arg.Actions,
	}
}

// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
func (p *PermissionService) CreatePermission(ctx context.Context, arg model.PermissionModel) (*model.PermissionModel, error) {
	if util.IsStringEmpty(arg.Name) {
		return nil, er.New(er.BadRequestCode, "權限名稱無效")
	}
	if util.IsStringEmpty(arg.Resource) {
		return nil, er.New(er.BadRequestCode, "權限資源無效")
	}
	if util.IsStringEmpty(arg.Actions) {
		return nil, er.New(er.BadRequestCode, "權限動作無效")
	}

	permissionEntity, err := p.dbDao.CreatePermission(ctx, convertPermissionModelToRepo(arg))
	if err != nil {
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	permissionModel := convertPermissioRepoToModel(permissionEntity)
	return &permissionModel, nil
}

func convertPermissionModelToNotExistsRepo(arg model.PermissionModel) sqlc.CreatePermissionIfNotExistsParams {
	return sqlc.CreatePermissionIfNotExistsParams{
		ID:          arg.ID,
		Name:        arg.Name,
		Description: pgutil.StringToPgTextV5(arg.Description),
		Resource:    arg.Resource,
		Actions:     arg.Actions,
	}
}

// 錯誤:
//   - er.InternalErrorCode 500: 資料庫操作錯誤
//   - er.BadRequestCode 400: 無效的參數
func (p *PermissionService) CreatePermissionIfNotExists(ctx context.Context, arg model.PermissionModel) error {
	if util.IsStringEmpty(arg.Name) {
		return er.New(er.BadRequestCode, "權限名稱無效")
	}
	if util.IsStringEmpty(arg.Resource) {
		return er.New(er.BadRequestCode, "權限資源無效")
	}
	if util.IsStringEmpty(arg.Actions) {
		return er.New(er.BadRequestCode, "權限動作無效")
	}

	err := p.dbDao.CreatePermissionIfNotExists(ctx, convertPermissionModelToNotExistsRepo(arg))
	if err != nil {
		return er.New(er.InternalErrorCode, err.Error())
	}

	return nil
}
