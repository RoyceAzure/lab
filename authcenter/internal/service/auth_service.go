package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/RoyceAzure/lab/authcenter/internal/config"
	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/auth/google_auth"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	"github.com/RoyceAzure/lab/authcenter/internal/util"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/RoyceAzure/rj/util/crypt"
	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	"github.com/RoyceAzure/rj/util/random"
	er "github.com/RoyceAzure/rj/util/rj_error"
	"github.com/google/uuid"
)

type IAuthService interface {
	// AuthGoogleLogin 處理Google登入認證並返回用戶登入資訊
	//
	// 參數:
	//   - ctx: 上下文，包含請求相關資訊
	//   - googleID: Google提供的ID令牌
	//
	// 返回值:
	//   - *model.LoginResponseModel: 包含訪問令牌、刷新令牌和用戶資訊的響應模型
	//   - error: 可能發生的錯誤
	//
	// 錯誤:
	//   - er.UnauthenticatedCode 401: Google ID驗證失敗
	//   - er.UnauthorizedCode 403: 使用者不合法
	//   - 500: 用戶驗證、會話獲取或創建過程中的錯誤
	AuthGoogleLogin(ctx context.Context, googleID string) (*model.LoginResponseModel, error)
	// CheckUserValidate 驗證User合法性
	//
	// 參數:
	//   - email: User email
	//
	// 返回值:
	//   - *model.UserModel: UserModel
	//
	// 錯誤:
	//   - UserNotFoundCode: 用戶不存在
	//   - UserDisabledCode: 用戶已禁用
	CheckUserValidate(ctx context.Context, email string) (*model.UserModel, error)
	CreateAccessToken(ctx context.Context, upn string, userID uuid.UUID) (string, *token.Payload[uuid.UUID], error)
	//ValidateUserSession 檢查session是否有效
	//檢查過期  is_active欄位有效
	//若session無效或者過期  執行logout  並要求前端重新登入
	ValidateSession(ctx context.Context, session *model.UserSession) error
	// ReNewToken 使用刷新令牌生成新的訪問令牌
	//
	// 參數:
	//   - ctx: 上下文，包含請求相關資訊
	//   - refreshToken: 用戶提供的刷新令牌
	//
	// 返回值:
	//   - string: 新生成的訪問令牌
	//
	// 錯誤:
	//   - er.UnauthenticatedCode 401: 刷新令牌無效或已過期
	//   - er.UnauthorizedCode 403: 用戶無權限或會話被撤銷
	//   - er.InternalErrorCode 500: 內部處理錯誤
	ReNewToken(ctx context.Context, refreshToken string) (string, error)
	// Me 取得當前登入user資訊
	// 錯誤:
	//   - er.UnauthorizedCode 403: 未授權
	Me(ctx context.Context) (*model.UserModel, error)
	// Logout 使用刷新令牌登出並撤銷用戶會話
	//
	// 參數:
	//   - ctx: 包含請求相關資訊的上下文
	//   - refreshToken: 用戶的刷新令牌
	//
	// 返回值:
	//   - error: 可能發生的錯誤
	//
	// 錯誤:
	//   - er.UnauthenticatedCode 401: 刷新令牌無效或格式錯誤
	//   - er.UnauthorizedCode 403: 找不到對應的會話
	//   - er.InternalErrorCode 500: 刪除會話時發生內部錯誤
	Logout(ctx context.Context, refreshToken string) error
	// GetUserPermissions 函數用於檢索指定使用者 ID 的權限列表。
	//
	// 參數：
	//
	//	ctx context.Context: 操作的上下文，允許取消和設定超時。
	//	userID int32: 要檢索權限的使用者 ID。
	//
	// 返回值：
	//
	//	[]model.PermissionModel: 一個 PermissionModel 的切片，表示使用者的權限。如果使用者沒有任何權限，則返回一個空切片。
	//  bool : 是否為adimn
	//	error: 操作過程中遇到的錯誤。可能的錯誤包括：
	//	  - er.InvalidArgumentCode 460: 如果提供的 userID 無效（小於 1）。
	//	  - er.InternalErrorCode 500: 如果資料庫操作過程中發生錯誤。
	GetUserPermissions(ctx context.Context, userID uuid.UUID) (bool, []model.PermissionModel, error)
}

type AuthService struct {
	dbDao              db.IStore
	userService        IUserService
	sessionService     ISessionService
	mailService        IMailService
	googleAuthVerifier google_auth.IAuthVerifier
	tokenMaker         token.Maker[uuid.UUID]
}

var (
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrSessionInactive = errors.New("session is not active")
)

func NewAuthService(dbDao db.IStore, userService IUserService, sessionService ISessionService, mailService IMailService, tokenMaker token.Maker[uuid.UUID], googleAuthVerifier google_auth.IAuthVerifier) IAuthService {
	if reflect.ValueOf(dbDao).IsNil() {
		panic("auth service initialization failed: dbDao cannot be nil")
	}
	if reflect.ValueOf(sessionService).IsNil() {
		panic("auth service initialization failed: sessionService cannot be nil")
	}
	if reflect.ValueOf(userService).IsNil() {
		panic("auth service initialization failed: userService cannot be nil")
	}
	if reflect.ValueOf(mailService).IsNil() {
		panic("auth service initialization failed: mailService cannot be nil")
	}
	if reflect.ValueOf(googleAuthVerifier).IsNil() {
		panic("auth service initialization failed: googleAuthVerifier cannot be nil")
	}
	if reflect.ValueOf(tokenMaker).IsNil() {
		panic("auth service initialization failed: tokenMaker cannot be nil")
	}

	return &AuthService{
		dbDao:              dbDao,
		userService:        userService,
		sessionService:     sessionService,
		mailService:        mailService,
		googleAuthVerifier: googleAuthVerifier,
		tokenMaker:         tokenMaker,
	}
}

// AuthGoogleLogin 處理Google登入認證並返回用戶登入資訊
//
// 參數:
//   - ctx: 上下文，包含請求相關資訊
//   - googleID: Google提供的ID令牌
//
// 返回值:
//   - *model.LoginResponseModel: 包含訪問令牌、刷新令牌和用戶資訊的響應模型
//   - error: 可能發生的錯誤
//
// 錯誤:
//   - er.UnauthenticatedCode 401: Google ID驗證失敗
//   - er.UnauthorizedCode 403: 使用者不合法
//   - 500: 用戶驗證、會話獲取或創建過程中的錯誤
func (a *AuthService) AuthGoogleLogin(ctx context.Context, googleID string) (*model.LoginResponseModel, error) {
	//認證google id
	authUserInfo, err := a.googleAuthVerifier.VerifyIDToken(ctx, googleID)
	if err != nil {
		return nil, er.New(er.UnauthenticatedCode, err.Error())
	}

	userModel, err := a.CheckUserValidate(ctx, authUserInfo.Email)
	if err != nil {
		return nil, er.New(er.UnauthorizedCode, err.Error())
	}

	//取得用戶額外資訊
	deviceInfo := util.GetDeviceInfoFromContext(ctx)

	//取得用戶session
	userSession, err := a.sessionService.GetUserSessionByReqInfo(ctx, userModel.ID, deviceInfo.IPAddress, deviceInfo.DeviceType, deviceInfo.Region, deviceInfo.UserAgent)

	//session不存在,有錯誤  或者session不合法就重建
	if err != nil || a.ValidateSession(ctx, userSession) != nil {
		newSession, err := a.createdUserSession(ctx, *userModel, deviceInfo)
		if err != nil {
			return nil, err
		}
		userSession = newSession
	}

	return &model.LoginResponseModel{
		AccessToken:  userSession.AccessToken,
		RefreshToken: userSession.RefreshToken,
		User:         *userModel,
	}, nil
}

// CheckUserValidate 驗證User合法性
//
// 參數:
//   - email: User email
//
// 返回值:
//   - *model.UserModel: UserModel
//
// 錯誤:
//   - UserNotFoundCode 470: 用戶不存在
//   - UserDisabledCode 471: 用戶已禁用
func (authService *AuthService) CheckUserValidate(ctx context.Context, email string) (*model.UserModel, error) {
	userEntity, err := authService.dbDao.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, er.New(er.UserNotFoundCode, err.Error())
	}

	if !userEntity.IsActive {
		return nil, er.New(er.UserDisabledCode, "user is not actice")
	}

	return convertRepoUsertToModel(&userEntity), nil
}

// ValidateSessionByRequestInfo 檢查用戶會話是否有效
func (authService *AuthService) ValidateSession(ctx context.Context, session *model.UserSession) error {
	if session.RevokedAt != nil && !session.RevokedAt.IsZero() {
		return er.New(er.UnauthorizedCode, ErrSessionRevoked.Error())
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return er.New(er.UnauthorizedCode, ErrSessionExpired.Error())
	}

	if !session.IsActive {
		return er.New(er.UnauthorizedCode, ErrSessionInactive.Error())
	}

	return nil
}

// ReNewToken 使用刷新令牌生成新的訪問令牌
//
// 參數:
//   - ctx: 上下文，包含請求相關資訊
//   - refreshToken: 用戶提供的刷新令牌
//
// 返回值:
//   - string: 新生成的訪問令牌
//
// 錯誤:
//   - er.UnauthenticatedCode 401: 刷新令牌無效或已過期
//   - er.UnauthorizedCode 403: 用戶無權限或會話被撤銷
//   - er.InternalErrorCode 500: 內部處理錯誤
//
// 流程說明:
//  1. 驗證刷新令牌的有效性
//  2. 檢查用戶是否有效
//  3. 檢查會話是否存在
//  4. 驗證會話的有效性
//  5. 確認提供的刷新令牌與會話中存儲的一致
//  6. 創建新的訪問令牌
//  7. 更新會話中的令牌資訊
func (authService *AuthService) ReNewToken(ctx context.Context, refreshToken string) (string, error) {
	payload, err := authService.tokenMaker.VertifyToken(refreshToken)
	if err != nil {
		return "", er.New(er.UnauthenticatedCode, "token invalid")
	}

	//檢查使用者合法性
	userModel, err := authService.CheckUserValidate(ctx, payload.UPN)
	if err != nil {
		return "", er.New(er.UnauthorizedCode, "unauthorized")
	}

	//檢查session存在
	userSession, err := authService.sessionService.GetUserSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", er.New(er.UnauthorizedCode, "unauthorized")
	}

	//檢查session合法
	if err := authService.ValidateSession(ctx, userSession); err != nil {
		authService.sessionService.DeleteSession(ctx, userSession.ID)
		return "", er.New(er.UnauthorizedCode, "unauthorized")
	}

	//檢查refreshtoken 重放攻擊
	if refreshToken != userSession.RefreshToken {
		authService.sessionService.DeleteSession(ctx, userSession.ID)
		return "", er.New(er.UnauthorizedCode, "unauthorized")
	}

	accessToken, _, err := authService.CreateAccessToken(ctx, userModel.Email, userModel.ID)
	if err != nil {
		return "", er.New(er.InternalErrorCode, err.Error())
	}

	//更新session
	_, err = authService.sessionService.UpdateSessionTokens(ctx, userSession.ID, accessToken, userSession.RefreshToken)
	if err != nil {
		return "", er.New(er.InternalErrorCode, err.Error())
	}

	return accessToken, nil
}

func (authService *AuthService) CreateAccessToken(ctx context.Context, upn string, userID uuid.UUID) (string, *token.Payload[uuid.UUID], error) {
	return authService.tokenMaker.CreateToken(upn, userID, 24*time.Hour)
}

func (authService *AuthService) ValidateUserAndOperator(ctx context.Context, operator *model.UserModel, opted uuid.UUID) bool {
	if operator.IsAdmin {
		return true
	}
	return operator.ID == opted
}

// Logout 使用刷新令牌登出並撤銷用戶會話
//
// 參數:
//   - ctx: 包含請求相關資訊的上下文
//   - refreshToken: 用戶的刷新令牌
//
// 返回值:
//   - error: 可能發生的錯誤
//
// 錯誤:
//   - er.UnauthenticatedCode 401: 刷新令牌無效或格式錯誤
//   - er.UnauthorizedCode 403: 找不到對應的會話
//   - er.InternalErrorCode 500: 刪除會話時發生內部錯誤
func (authService *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// 驗證刷新令牌的格式和簽名，但可以忽略過期時間
	payload, err := authService.tokenMaker.VertifyToken(refreshToken)
	if err != nil && !errors.Is(err, token.ErrExpiredToken) {
		return er.New(er.UnauthenticatedCode, "unauthticated")
	}

	// 查找對應的會話
	session, err := authService.sessionService.GetUserSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return er.New(er.UnauthorizedCode, "session not found")
	}

	// 驗證會話屬於當前用戶
	if payload.UserId != session.UserID {
		return er.New(er.UnauthorizedCode, "unauthorized")
	}

	// 刪除會話
	err = authService.sessionService.DeleteSession(ctx, session.ID)
	if err != nil {
		return er.New(er.InternalErrorCode, "failed to delete session")
	}

	return nil
}

// Me 取得當前登入user資訊
// 錯誤:
//   - er.UnauthorizedCode 403: 未授權
func (authService *AuthService) Me(ctx context.Context) (*model.UserModel, error) {
	payload := util.GetTokenPayloadFromContext[uuid.UUID](ctx)
	if payload == nil {
		return nil, er.New(er.UnauthorizedCode, "unauthorized")
	}

	userModel, err := authService.userService.GetUserByID(ctx, payload.UserId)
	if err != nil {
		return nil, er.New(er.UnauthorizedCode, "unauthorized")
	}

	return userModel, nil
}

// GetUserPermissions 函數用於檢索指定使用者 ID 的權限列表。
//
// 參數：
//
//	ctx context.Context: 操作的上下文，允許取消和設定超時。
//	userID int32: 要檢索權限的使用者 ID。
//
// 返回值：
//
//	[]model.PermissionModel: 一個 PermissionModel 的切片，表示使用者的權限。如果使用者沒有任何權限，則返回一個空切片。
//	error: 操作過程中遇到的錯誤。可能的錯誤包括：
//	  - er.InvalidArgumentCode 460: 如果提供的 userID 無效（小於 1）。
//	  - er.InternalErrorCode 500: 如果資料庫操作過程中發生錯誤。
func (a *AuthService) GetUserPermissions(ctx context.Context, userID uuid.UUID) (bool, []model.PermissionModel, error) {
	userModel, err := a.userService.GetUserByID(ctx, userID)
	if err != nil {
		return false, nil, er.New(er.InternalErrorCode, err.Error())
	}

	permissionEntities, err := a.dbDao.GetUserPermissions(ctx, pgutil.UUIDToPgUUIDV5(userID))
	if err != nil {
		return false, nil, er.New(er.InternalErrorCode, err.Error())
	}

	var permissions []model.PermissionModel
	for _, pEntity := range permissionEntities {
		permissions = append(permissions, convertPermissionToModel(pEntity))
	}

	return userModel.IsActive, permissions, nil
}

func convertPermissionToModel(p sqlc.Permission) model.PermissionModel {
	return model.PermissionModel{
		ID:          p.ID,
		Name:        p.Name,
		Description: pgutil.PgTextToStringV5(p.Description),
		Resource:    p.Resource,
		Actions:     p.Actions,
	}

}

// createdUserSession 創建userSession
// 錯誤:
//   - er.InternalErrorCode 500: access token創建錯誤
//   - er.InternalErrorCode 500: refresh token創建錯誤
//   - er.InternalErrorCode 500: user session創建錯誤
func (authService *AuthService) createdUserSession(ctx context.Context, user model.UserModel, deviceInfo util.DeviceInfo) (*model.UserSession, error) {
	accessToken, _, err := authService.tokenMaker.CreateToken(user.Email, user.ID, time.Duration(constants.AccessTokenDuration)*time.Hour)
	if err != nil {
		return nil, er.New(er.InternalErrorCode, "created accessToken failed")
	}

	refreshTokenDur := time.Duration(constants.RefreshTokenDuration) * time.Hour
	refreshToken, _, err := authService.tokenMaker.CreateToken(user.Email, user.ID, refreshTokenDur)
	if err != nil {
		return nil, er.New(er.InternalErrorCode, "created refreshToken failed")
	}

	s2nils := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	userSession, err := authService.sessionService.CreateSession(ctx, &model.UserSession{
		ID:             uuid.New(),
		UserID:         user.ID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		IPAddress:      deviceInfo.IPAddress,
		DeviceInfo:     deviceInfo.DeviceType,
		Region:         s2nils(deviceInfo.Region),
		UserAgent:      s2nils(deviceInfo.UserAgent),
		IsActive:       true,
		LastActivityAt: time.Now().UTC(),
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(refreshTokenDur),
	})
	if err != nil {
		return nil, er.New(er.InternalErrorCode, "created user session failed")
	}

	return userSession, nil
}

// CreateUserByAccountAndPas 使用者使用帳號密碼創建帳號
// 必須先驗證過email，驗證過後，db會已經存在該使用者email資訊，且是起用狀態
//
// 參數：
//
//	ctx context.Context: 操作的上下文，允許取消和設定超時。
//	userID int32: 要檢索權限的使用者 ID。
//
// 返回值：
//
//	model.CreateUserByAccountAndPasResult:
//		ResultCode int32:
//		0: 成功
//		1: email已經存在，需要執行帳戶連結，返回讓用戶確認是否連結
//		UserModel  UserModel
//	error: 操作過程中遇到的錯誤。可能的錯誤包括：
//	  - er.InvalidArgumentCode 460: 如果提供的 userID 無效（小於 1）。
//	  - er.InternalErrorCode 500: 如果資料庫操作過程中發生錯誤。
func (a *AuthService) CreateUserByAccountAndPas(ctx context.Context, userModel *model.CreateUserByAccountAndPasModel) error {
	//已經驗證過email, email就必定會存在
	user, err := a.userService.GetUserByEmail(ctx, userModel.Email)
	if err != nil {
		return err
	}

	if !user.IsActive {
		return er.New(er.InternalErrorCode, "user is not active")
	}

	hashPassword, err := crypt.HashPassword(userModel.Password)
	if err != nil {
		return er.New(er.InternalErrorCode, "hash password failed")
	}

	//update user
	err = a.userService.UpdateUserAccountAndPassword(ctx, user.ID, userModel.Account, hashPassword)
	if err != nil {
		return err
	}

	return nil
}

func getVertifyEmailLink(linkCode string) string {
	return fmt.Sprintf("%s/auth/vertify-email?code=%s", config.GetConfig().AuthCenterUrl, linkCode)
}

// CreateVertifyUserEmailLink 使用者使用帳號密碼創建帳號時，需要先認證email。
func (a *AuthService) CreateVertifyUserEmailLink(ctx context.Context, email string) error {
	//要發送email認證信， 使用者點擊後就要做帳戶連結
	//將當前要創建使用者的資訊存入db, 前端回傳相關api 網址 到使用者email 讓使用者點擊後做帳戶連結
	randomLinkString := random.RandomString(25)
	//email, randomLinkString 存入db
	//用randomLinkString組合api網址資訊
	emailLink, err := a.dbDao.CreateEmailVertify(ctx, sqlc.CreateEmailVertifyParams{
		ID:        randomLinkString,
		Email:     email,
		ExpiresAt: time.Now().UTC().Add(time.Minute * 10),
		IsUsed:    false,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return er.New(er.InternalErrorCode, "create email vertify failed")
	}

	//組合email link
	link := getVertifyEmailLink(emailLink.ID)
	//發送email認證信
	//todo 需要email template
	err = a.mailService.SendVertifyEmail(ctx, &model.MailContent{
		Subject: "Email Verification",
		Content: link,
		To:      []string{email},
	})
	if err != nil {
		a.dbDao.DeleteEmailVertify(ctx, emailLink.ID)
		return er.New(er.InternalErrorCode, "send email vertify failed")
	}

	return nil
}

// VertifyUserEmailLink 驗證使用者email連結
// 參數:
//
//	ctx context.Context: 操作的上下文，允許取消和設定超時。
//	linkCode string: 要驗證的連結代碼。
//
// 返回值：
//
//	error: 操作過程中遇到的錯誤。可能的錯誤包括：
//		- er.NotFoundCode 404: 找不到連結代碼。
//		- er.InvalidOperationCode 405: 連結代碼已過期。
//		- er.InvalidOperationCode 405: 連結代碼已使用。
//	    - er.InternalErrorCode 500: 如果資料庫操作過程中發生錯誤。
func (a *AuthService) VertifyUserEmailLink(ctx context.Context, linkCode string) error {
	//檢查linkCode是否存在
	emailVertify, err := a.dbDao.GetEmailVertify(ctx, linkCode)
	if err != nil {
		return er.New(er.NotFoundCode, "email vertify link not found")
	}
	//檢查linkCode是否過期
	if emailVertify.ExpiresAt.Before(time.Now()) {
		return er.New(er.InvalidOperationCode, "email vertify link expired")
	}
	//檢查linkCode是否被使用
	if emailVertify.IsUsed {
		return er.New(er.InvalidOperationCode, "email vertify link already used")
	}
	//如果以上都沒有問題，則將linkCode對應的email取出
	user, err := a.userService.GetUserByEmail(ctx, emailVertify.Email)
	if err != nil {
		if _, ok := err.(*er.AnaError); !ok {
			return err
		}
		if !err.(*er.AnaError).Is(er.NotFoundError) {
			return err
		}
	}

	if user == nil {
		user = &model.UserModel{
			ID:        uuid.New(),
			Email:     emailVertify.Email,
			IsActive:  true,
			CreatedAt: time.Now().UTC(),
		}
		_, err = a.userService.CreateUser(ctx, user)
		if err != nil {
			return err
		}
	}

	err = a.userService.ActiveUser(ctx, user.ID)
	if err != nil {
		return err
	}

	return nil
}
