package dto

// TokenInfo 表示令牌資訊
type TokenInfo struct {
	Value     string `json:"value"`
	ExpiresIn int    `json:"expires_in"`
}

// UserDTO 表示用戶資訊
type UserDTO struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Name       *string `json:"name"`
	GoogleID   *string `json:"google_id"`
	LineUserID *string `json:"line_user_id"`
	FacebookID *string `json:"facebook_id"`
	Account    *string `json:"account"`
	IsActive   bool    `json:"is_active"`
	IsAdmin    bool    `json:"is_admin"`
}

// LoginResponse 表示登入響應的完整結構
type LoginResponse struct {
	AccessToken  TokenInfo `json:"access_token"`
	RefreshToken TokenInfo `json:"refresh_token"`
	User         UserDTO   `json:"user"`
}

//user permissions
type UserPermissionsDTO struct {
	Permissions []PermissionsDTO `json:"resource"`
	IsAdmin     bool             `json:"is_admin"`
}

type PermissionsDTO struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

type GoogleLoginDTO struct {
	IdToken string `json:"id_token"`
}

type RefreshTokenDTO struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateVertifyUserEmailLinkDTO struct {
	Email string `json:"email"`
}

type LinkedUserAccountAndPasswordDTO struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AccountAndPasswordLoginDTO struct {
	Account  string `json:"account"`  //帳號
	Password string `json:"password"` //密碼明文
}
