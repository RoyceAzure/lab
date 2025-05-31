package google_auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type IAuthVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*UserInfo, error)
}

// GoogleAuthVerifier 驗證Google登入token並取得用戶資訊
type GoogleAuthVerifier struct {
	ClientID string // 你的Google Client ID
}

// Google UserInfo 存放用戶資訊
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// NewGoogleAuthVerifier 建立一個新的驗證器
func NewGoogleAuthVerifier(clientID string) *GoogleAuthVerifier {
	return &GoogleAuthVerifier{
		ClientID: clientID,
	}
}

// VerifyIDToken 驗證從前端傳來的ID token
func (g *GoogleAuthVerifier) VerifyIDToken(ctx context.Context, idToken string) (*UserInfo, error) {
	// 使用Google的token驗證端點
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending verification request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid token")
	}

	var tokenInfo struct {
		Aud string `json:"aud"` // Client ID
		UserInfo
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// 驗證這個token是發給你的應用的
	if tokenInfo.Aud != g.ClientID {
		return nil, errors.New("token was not issued for this application")
	}

	return &tokenInfo.UserInfo, nil
}
