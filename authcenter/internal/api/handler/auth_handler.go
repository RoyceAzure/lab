package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RoyceAzure/lab/authcenter/internal/api/dto"
	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/lab/authcenter/internal/model"
	"github.com/RoyceAzure/lab/authcenter/internal/service"
	"github.com/RoyceAzure/lab/authcenter/internal/util"
	"github.com/RoyceAzure/rj/api"
	er "github.com/RoyceAzure/rj/util/rj_error"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService service.IAuthService
}

func NewAuthHandler(authService service.IAuthService) *AuthHandler {
	if authService == nil {
		panic("authService cannot be nil")
	}
	return &AuthHandler{
		authService: authService,
	}
}

// @Summary google login
// @use google idtoken to login
// @Tags auth
// @Accept json
// @Produce json
// @Param id_token body dto.GoogleLoginDTO true "google id token"
// @Success 200 {object} api.Response{data=dto.LoginResponse} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 403 {object} api.ResponseError{data=string} "UnauthorizedCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/login/google [post]
func (a *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var loginDTO dto.GoogleLoginDTO
	if err := json.NewDecoder(r.Body).Decode(&loginDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), nil, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()

	loginRes, err := a.authService.AuthGoogleLogin(ctx, loginDTO.IdToken)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, dto.LoginResponse{
		AccessToken: dto.TokenInfo{
			Value:     loginRes.AccessToken,
			ExpiresIn: int(constants.AccessTokenDuration) * 3600,
		},
		RefreshToken: dto.TokenInfo{
			Value:     loginRes.RefreshToken,
			ExpiresIn: int(constants.RefreshTokenDuration) * 3600,
		},
		User: convertUserModelToDTO(loginRes.User),
	}, nil)
}

// @Summary account and password login
// @use account and password to login
// @Tags auth
// @Accept json
// @Produce json
// @Param accountInfo body dto.AccountAndPasswordLoginDTO true "account and password"
// @Success 200 {object} api.Response{data=dto.LoginResponse} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 403 {object} api.ResponseError{data=string} "UnauthorizedCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/login/account [post]
func (a *AuthHandler) AccountAndPasswordLogin(w http.ResponseWriter, r *http.Request) {
	var loginDTO dto.AccountAndPasswordLoginDTO
	if err := json.NewDecoder(r.Body).Decode(&loginDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), nil, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()

	loginRes, err := a.authService.AccountAndPasswordLogin(ctx, loginDTO.Account, loginDTO.Password)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, dto.LoginResponse{
		AccessToken: dto.TokenInfo{
			Value:     loginRes.AccessToken,
			ExpiresIn: int(constants.AccessTokenDuration) * 3600,
		},
		RefreshToken: dto.TokenInfo{
			Value:     loginRes.RefreshToken,
			ExpiresIn: int(constants.RefreshTokenDuration) * 3600,
		},
		User: convertUserModelToDTO(loginRes.User),
	}, nil)
}

// convertUserModelToDTO 將 UserModel 轉換為 UserDTO
//
// 參數:
//   - model: 用戶模型數據
//
// 返回值:
//   - UserDTO: 轉換後的用戶數據傳輸對象
func convertUserModelToDTO(model model.UserModel) dto.UserDTO {
	return dto.UserDTO{
		ID:         model.ID.String(),
		Email:      model.Email,
		Name:       model.Name,
		GoogleID:   model.GoogleID,
		LineUserID: model.LineUserID,
		FacebookID: model.FacebookID,
		Account:    model.Account,
		IsActive:   model.IsActive,
		IsAdmin:    model.IsAdmin,
	}
}

// @Summary renew token
// @use refresh token to renew access token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body dto.RefreshTokenDTO true "refresh token"
// @Success 200 {object} api.Response{data=dto.TokenInfo} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 403 {object} api.ResponseError{data=string} "UnauthorizedCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/refresh-token [post]
func (a *AuthHandler) ReNewToken(w http.ResponseWriter, r *http.Request) {
	var refreshTokenDTO dto.RefreshTokenDTO
	if err := json.NewDecoder(r.Body).Decode(&refreshTokenDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), err, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()

	accessToken, err := a.authService.ReNewToken(ctx, refreshTokenDTO.RefreshToken)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, dto.TokenInfo{
		Value:     accessToken,
		ExpiresIn: int(constants.AccessTokenDuration) * 3600,
	}, nil)
}

// @Summary get current login user info
// @use get current login user info
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=dto.UserDTO} "success"
// @Failure 403 {object} api.ResponseError{data=string} "UnauthorizedCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Security     ApiKeyAuth
// @Router /auth/me [get]
func (a *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userModel, err := a.authService.Me(ctx)
	if err != nil || userModel == nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, convertUserModelToDTO(*userModel), nil)
}

// @Summary logout
// @use logout current login user
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body dto.RefreshTokenDTO true "refresh token"
// @Success 200 {object} api.Response{data=string} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 403 {object} api.ResponseError{data=string} "UnauthorizedCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/logout [post]
func (a *AuthHandler) LogOut(w http.ResponseWriter, r *http.Request) {
	var refreshTokenDTO dto.RefreshTokenDTO
	if err := json.NewDecoder(r.Body).Decode(&refreshTokenDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), err, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()
	err := a.authService.Logout(ctx, refreshTokenDTO.RefreshToken)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, nil, nil)
}

func convertGetPermissionModelToDTO(isAdmin bool, permissionModels []model.PermissionModel) dto.UserPermissionsDTO {
	m := map[string][]string{}
	for _, permissionModel := range permissionModels {
		if slice, ok := m[permissionModel.Resource]; ok {
			m[permissionModel.Resource] = append(slice, permissionModel.Actions)
		} else {
			m[permissionModel.Resource] = []string{permissionModel.Actions}
		}
	}

	var pers []dto.PermissionsDTO

	for k, v := range m {
		pers = append(pers, dto.PermissionsDTO{
			Resource: k,
			Actions:  v,
		})
	}

	return dto.UserPermissionsDTO{
		Permissions: pers,
		IsAdmin:     isAdmin,
	}
}

// @Summary Get user permissions
// @use Get user permissions
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=dto.UserPermissionsDTO} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 460 {object} api.ResponseError{data=string} "InvalidArgumentCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Security     ApiKeyAuth
// @Router /auth/permissions [get]
func (a *AuthHandler) Permissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	payload := util.GetTokenPayloadFromContext[uuid.UUID](ctx)
	if payload == nil {
		api.ErrorJSON(w, int(er.UnauthenticatedCode), errors.New("token is invalidate"), er.ErrStrMap[er.UnauthenticatedCode])
		return
	}

	isAdmin, permissionModels, err := a.authService.GetUserPermissions(ctx, payload.UserId)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, convertGetPermissionModelToDTO(isAdmin, permissionModels), nil)
}

// @Summary 創建email認證連結
// @use 創建email認證連結
// @Tags Register
// @Accept json
// @Produce json
// @Param email body dto.CreateVertifyUserEmailLinkDTO true "email"
// @Success 200 {object} api.Response{data=string} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 460 {object} api.ResponseError{data=string} "InvalidArgumentCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/create-vertify-email [post]
func (a *AuthHandler) CreateVertifyUserEmailLink(w http.ResponseWriter, r *http.Request) {
	var createVertifyUserEmailLinkDTO dto.CreateVertifyUserEmailLinkDTO
	if err := json.NewDecoder(r.Body).Decode(&createVertifyUserEmailLinkDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), err, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()

	err := a.authService.CreateVertifyUserEmailLink(ctx, createVertifyUserEmailLinkDTO.Email)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, nil, nil)
}

// @Summary 驗證email認證連結
// @use 驗證email認證連結
// @Tags Register
// @Accept json
// @Produce json
// @Param code query string true "code"
// @Success 200 {object} api.Response{data=string} "success"
// @Failure 401 {object} api.ResponseError{data=string} "UnauthenticatedCode"
// @Failure 460 {object} api.ResponseError{data=string} "InvalidArgumentCode"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/vertify-email [get]
func (a *AuthHandler) VertifyUserEmailLink(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	ctx := r.Context()

	err := a.authService.VertifyUserEmailLink(ctx, code)
	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, nil, nil)
}

// @Summary 使用帳號密碼連結email，必須要先認證email
// @use 使用帳號密碼連結email
// @Tags Register
// @Accept json
// @Produce json
// @Param accountInfo body dto.LinkedUserAccountAndPasswordDTO true "account info"
// @Success 200 {object} api.Response{data=string} "success"
// @Failure 500 {object} api.ResponseError{data=string} "Internal server error"
// @Router /auth/linkedUser [post]
func (a *AuthHandler) LinkedUserAccountAndPassword(w http.ResponseWriter, r *http.Request) {
	var linkedUserAccountAndPasswordDTO dto.LinkedUserAccountAndPasswordDTO
	if err := json.NewDecoder(r.Body).Decode(&linkedUserAccountAndPasswordDTO); err != nil {
		api.ErrorJSON(w, int(er.BadRequestCode), err, er.ErrStrMap[er.BadRequestCode])
		return
	}

	ctx := r.Context()

	err := a.authService.LinkUserAccountAndPas(ctx, &model.CreateUserByAccountAndPasModel{
		Account:  linkedUserAccountAndPasswordDTO.Account,
		Password: linkedUserAccountAndPasswordDTO.Password,
		Email:    linkedUserAccountAndPasswordDTO.Email,
	})

	if err != nil {
		if anaErr, ok := err.(*er.AnaError); ok {
			api.ErrorJSON(w, int(anaErr.Code), anaErr, er.ErrStrMap[anaErr.Code])
		} else {
			api.ErrorJSON(w, int(er.InternalErrorCode), err, er.ErrStrMap[er.InternalErrorCode])
		}
		return
	}

	api.SuccessJSON(w, nil, nil)
}
