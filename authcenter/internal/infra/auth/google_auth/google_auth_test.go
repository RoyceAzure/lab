package google_auth

import (
	"context"
	"testing"
	"time"
)

func TestGoogleAuthVerifier_VerifyIDToken(t *testing.T) {
	// 從環境變數獲取測試用的token和clientID
	testIDToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjgyMWYzYmM2NmYwNzUxZjc4NDA2MDY3OTliMWFkZjllOWZiNjBkZmIiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJhenAiOiIyMjc0MDQ4NzEwOTMtYW8xc2R0MXUyMThoa3UzdWs5Z200c2xicnVpM3U0ZG0uYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJhdWQiOiIyMjc0MDQ4NzEwOTMtYW8xc2R0MXUyMThoa3UzdWs5Z200c2xicnVpM3U0ZG0uYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJzdWIiOiIxMDc5NTc1OTQ5MDcwNzA0MjUyNDQiLCJlbWFpbCI6InJveWNld25hZ0BnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiYXRfaGFzaCI6IkV2dzNnNl9YekxUS3FvVVQzeHZnT1EiLCJuYW1lIjoiUm95Y2UgV25hZyIsInBpY3R1cmUiOiJodHRwczovL2xoMy5nb29nbGV1c2VyY29udGVudC5jb20vYS9BQ2c4b2NKc01hcGhGTlB1amZHYXowOUxia1VNSUE0bG9tMFk1OEk3UUtRbnQwYW84UWlhaGc9czk2LWMiLCJnaXZlbl9uYW1lIjoiUm95Y2UiLCJmYW1pbHlfbmFtZSI6IlduYWciLCJpYXQiOjE3NDMwNjQ2NDksImV4cCI6MTc0MzA2ODI0OX0.QLnLYBUwE5Dm2HMYSmL8HjXJqEwfJkBVOkse0cyjItGonGAcodPpMyaH13W1GGCSScSM5B_Fyl8Jsas7OEry50efJmKTW2FrKavnHaI2MtR4dUmuc_UZYZA1rfTW4KuExpC-yjyznqWazNvCS1Puq_0Qa4cS8TTnezN1CHiMpxVN34TowkFAuWXzjXut3sZCwZIadQD6XDwVcOSokW9cYI94doMrEo3ZYdP9fFXiqaohTuu20HU1H3deTHKo8ErnYtf6B1jugHjO54rZnDuqYmGEdciU1_JyD-A1PUHgRXfZzrAk_n9ehnBBAiJMGM0Ut6N7g2BxxYPutMSYwoIB7Q"
	clientID := "227404871093-ao1sdt1u218hku3uk9gm4slbrui3u4dm.apps.googleusercontent.com"

	if testIDToken == "" || clientID == "" {
		t.Skip("TEST_GOOGLE_ID_TOKEN 或 GOOGLE_CLIENT_ID 環境變數未設置，跳過測試")
	}

	// 創建驗證器
	verifier := NewGoogleAuthVerifier(clientID)

	// 設置超時上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 驗證token
	userInfo, err := verifier.VerifyIDToken(ctx, testIDToken)

	// 檢查結果
	if err != nil {
		t.Fatalf("驗證 ID Token 失敗: %v", err)
	}

	if userInfo == nil {
		t.Fatal("驗證成功但未返回用戶信息")
	}

	// 檢查返回的用戶信息是否包含基本欄位
	if userInfo.Email == "" {
		t.Error("用戶信息中缺少 Email")
	}

	if userInfo.ID == "" {
		t.Error("用戶信息中缺少 ID")
	}

	t.Logf("驗證成功，用戶: %s (%s)", userInfo.Name, userInfo.Email)
}
