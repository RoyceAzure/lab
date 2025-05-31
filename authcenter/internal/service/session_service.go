package service

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"
	"time"

	db "github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	er "github.com/RoyceAzure/rj/util/rj_error"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ISessionService interface {
	// CreateSession 創建新的會話記錄
	//
	// 參數:
	//   - ctx: 上下文，包含請求相關資訊
	//   - session: 包含會話資訊的模型
	//
	// 返回值:
	//   - *model.UserSession: 創建成功後的會話模型，包含數據庫生成的資訊
	//   - error: 可能發生的錯誤
	//
	// 錯誤:
	//   - 數據庫錯誤 500: 在創建會話過程中可能發生的數據庫錯誤
	CreateSession(ctx context.Context, session *model.UserSession) (*model.UserSession, error)
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) error
	// GetUserSessionByReqInfo 根據用戶ID、IP地址和設備名稱從數據庫獲取用戶會話信息
	//
	// 參數:
	//   - ctx: 上下文，用於控制請求生命週期
	//   - userID: 用戶ID
	//   - ipAddress: 客戶端IP地址
	//   - deviceName: 客戶端設備名稱
	//   - region: 客戶端區域
	//   - userAgent: 客戶端Agent
	//
	// 返回:
	//   - *model.UserSession: 用戶會話詳細信息
	//   - error: 操作過程中可能發生的錯誤
	//
	// 可能的錯誤:
	//   - 數據不存在 462
	//   - 數據庫操作錯誤 500
	GetUserSessionByReqInfo(ctx context.Context, userID uuid.UUID, ipAddress netip.Addr, deviceName, region, userAgent string) (*model.UserSession, error)
	// 可能的錯誤:
	//   - 數據不存在 462
	//   - 數據庫操作錯誤 500
	GetUserSessionByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error)
	// 可能的錯誤:
	//   - 數據不存在 462
	//   - 數據庫操作錯誤 500
	GetUserSessionByAccessToken(ctx context.Context, accessToken string) (*model.UserSession, error)
	// 可能的錯誤:
	//   - 數據庫操作錯誤 500
	UpdateSessionTokens(ctx context.Context, sessionID uuid.UUID, accessToken, refreshToken string) (*model.UserSession, error)
	// ForceClearAllSessions 強制清除所有用戶會話記錄
	// 通常在系統關閉或重啟前調用，以確保會話表格清空
	//
	// 參數:
	//   - ctx: 上下文，用於跟踪請求或設置截止日期
	//
	// 返回:
	//   - error: 操作可能返回的錯誤
	//
	// 可能的錯誤:
	//   - 數據庫操作錯誤 500
	ForceClearAllSessions(ctx context.Context) error
}

// SessionService 實現會話服務
type SessionService struct {
	dbDao db.IStore
}

// NewSessionService 創建新的會話服務實例
func NewSessionService(dbDao db.IStore) ISessionService {
	return &SessionService{
		dbDao: dbDao,
	}
}

// CreateSession 創建新的會話記錄
//
// 參數:
//   - ctx: 上下文，包含請求相關資訊
//   - session: 包含會話資訊的模型
//
// 返回值:
//   - *model.UserSession: 創建成功後的會話模型，包含數據庫生成的資訊
//   - error: 可能發生的錯誤
//
// 錯誤:
//   - 數據庫錯誤 500: 在創建會話過程中可能發生的數據庫錯誤
func (s *SessionService) CreateSession(ctx context.Context, session *model.UserSession) (*model.UserSession, error) {
	// 轉換為 repository 層使用的參數
	params := sqlc.CreateSessionParams{
		ID:           pgutil.UUIDToPgUUIDV5(session.ID),
		UserID:       pgutil.UUIDToPgUUIDV5(session.UserID),
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		IpAddress:    session.IPAddress,
		DeviceInfo:   session.DeviceInfo,
		Region:       pgutil.StringToPgTextV5(session.Region),
		UserAgent:    pgutil.StringToPgTextV5(session.UserAgent),
		ExpiresAt:    session.ExpiresAt,
	}

	repoSession, err := s.dbDao.CreateSession(ctx, params)
	if err != nil {
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return convertRepoSessionToModel(repoSession), nil
}

// DeleteSession 刪除指定的會話
func (s *SessionService) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	pgUUID := pgutil.UUIDToPgUUIDV5(sessionID)
	return s.dbDao.DeleteSession(ctx, pgUUID)
}

// DeleteExpiredSessions 刪除所有過期的會話
func (s *SessionService) DeleteExpiredSessions(ctx context.Context) error {
	return s.dbDao.DeleteExpiredSessions(ctx)
}

// 輔助函數 - 將 repository 模型轉換為服務層模型
func convertRepoSessionToModel(repoSession sqlc.UserSession) *model.UserSession {
	model := &model.UserSession{
		ID:             pgutil.PgUUIDToUUIDV5(repoSession.ID),
		UserID:         pgutil.PgUUIDToUUIDV5(repoSession.UserID),
		AccessToken:    repoSession.AccessToken,
		RefreshToken:   repoSession.RefreshToken,
		IPAddress:      repoSession.IpAddress,
		DeviceInfo:     repoSession.DeviceInfo,
		IsActive:       repoSession.IsActive.Bool,
		LastActivityAt: repoSession.LastActivityAt,
		CreatedAt:      repoSession.CreatedAt,
		ExpiresAt:      repoSession.ExpiresAt,
	}

	model.Region = pgutil.PgTextToStringV5(repoSession.Region)
	model.UserAgent = pgutil.PgTextToStringV5(repoSession.UserAgent)
	model.RevokedAt = pgutil.PgTimestamptzToTimeV5(repoSession.RevokedAt)

	return model
}

// GetUserSessionByReqInfo 根據用戶ID、IP地址和設備名稱從數據庫獲取用戶會話信息
//
// 參數:
//   - ctx: 上下文，用於控制請求生命週期
//   - userID: 用戶ID
//   - ipAddress: 客戶端IP地址
//   - deviceName: 客戶端設備名稱
//   - region: 客戶端區域
//   - userAgent: 客戶端Agent
//
// 返回:
//   - *model.UserSession: 用戶會話詳細信息
//   - error: 操作過程中可能發生的錯誤
//
// 可能的錯誤:
//   - 數據不存在 462
//   - 數據庫操作錯誤 500
func (s *SessionService) GetUserSessionByReqInfo(ctx context.Context, userID uuid.UUID, ipAddress netip.Addr, deviceName, region, userAgent string) (*model.UserSession, error) {
	sessionEntity, err := s.dbDao.GetSessionByRequestInfo(ctx, sqlc.GetSessionByRequestInfoParams{
		UserID:     pgutil.UUIDToPgUUIDV5(userID),
		IpAddress:  ipAddress,
		DeviceInfo: deviceName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, er.New(er.DataNotExistsCode, err.Error())
		}
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return &model.UserSession{
		ID:           pgutil.PgUUIDToUUIDV5(sessionEntity.ID),
		UserID:       pgutil.PgUUIDToUUIDV5(sessionEntity.UserID),
		AccessToken:  sessionEntity.AccessToken,
		RefreshToken: sessionEntity.RefreshToken,
		IPAddress:    sessionEntity.IpAddress,
		DeviceInfo:   sessionEntity.DeviceInfo,
		Region:       pgutil.PgTextToStringV5(sessionEntity.Region),
		UserAgent:    pgutil.PgTextToStringV5(sessionEntity.UserAgent),
		IsActive: func() bool {
			if b := pgutil.PgBoolToBoolV5(sessionEntity.IsActive); b != nil {
				return *b
			}
			return false
		}(),
		LastActivityAt: sessionEntity.LastActivityAt,
		CreatedAt:      sessionEntity.CreatedAt,
		ExpiresAt:      sessionEntity.ExpiresAt,
		RevokedAt: func() *time.Time {
			if sessionEntity.RevokedAt.Valid {
				return &sessionEntity.RevokedAt.Time
			}
			return nil
		}(),
	}, nil
}

// 可能的錯誤:
//   - 數據不存在 462
//   - 數據庫操作錯誤 500
func (s *SessionService) GetUserSessionByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error) {
	sessionEntity, err := s.dbDao.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, er.New(er.DataNotExistsCode, err.Error())
		}
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return convertRepoSessionToModel(sessionEntity), nil
}

// 可能的錯誤:
//   - 數據不存在 462
//   - 數據庫操作錯誤 500
func (s *SessionService) GetUserSessionByAccessToken(ctx context.Context, accessToken string) (*model.UserSession, error) {
	sessionEntity, err := s.dbDao.GetSessionByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, er.New(er.DataNotExistsCode, err.Error())
		}
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return convertRepoSessionToModel(sessionEntity), nil
}

// 可能的錯誤:
//   - 數據庫操作錯誤 500
func (s *SessionService) UpdateSessionTokens(ctx context.Context, sessionID uuid.UUID, accessToken, refreshToken string) (*model.UserSession, error) {
	sessionEntity, err := s.dbDao.UpdateSessionTokens(ctx, sqlc.UpdateSessionTokensParams{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
	})
	if err != nil {
		return nil, er.New(er.InternalErrorCode, err.Error())
	}

	return convertRepoSessionToModel(sessionEntity), nil
}

// ForceClearAllSessions 強制清除所有用戶會話記錄
// 通常在系統關閉或重啟前調用，以確保會話表格清空
//
// 參數:
//   - ctx: 上下文，用於跟踪請求或設置截止日期
//
// 返回:
//   - error: 操作可能返回的錯誤
//
// 可能的錯誤:
//   - 數據庫操作錯誤 500
func (s *SessionService) ForceClearAllSessions(ctx context.Context) error {
	return s.dbDao.ForceClearAllSessions(ctx)
}
