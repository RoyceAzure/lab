package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/RoyceAzure/rj/infra/mail"
)

type IMailService interface {
	SendVertifyEmail(ctx context.Context, data EmailVerificationData) error
}

type MailService struct {
	mail.EmailSender
}

// EmailVerificationData 認證信的數據結構
type EmailVerificationData struct {
	UserName        string // 使用者名稱
	Email           string // 使用者信箱
	VerificationURL string // 認證連結
	CompanyName     string // 公司名稱
	SupportEmail    string // 客服信箱
	ExpiryMinutes   int    // 連結有效時間(分鐘)
}

// NewMailService 初始化 mail service
// 參數:
//
//	sender_name: 寄件者屬名
//	fromEmailAddress: 寄件者郵件地址
//	fromEmailPassword: 寄件者郵件密碼
func NewMailService(sender_name, fromEmailAddress, fromEmailPassword string) IMailService {
	return &MailService{
		mail.NewGmailSender(sender_name, fromEmailAddress, fromEmailPassword),
	}
}

func (m *MailService) SendVertifyEmail(ctx context.Context, data EmailVerificationData) error {
	html, err := GenerateEmailHTML(data)
	if err != nil {
		return err
	}

	return m.SendEmail("Email 認證", html, []string{data.Email}, nil, nil, nil)
}

// GenerateEmailHTML 生成 HTML 格式的認證信
func GenerateEmailHTML(data EmailVerificationData) (string, error) {
	tmpl, err := template.New("emailHTML").Parse(emailVertifyTemplate)
	if err != nil {
		return "", fmt.Errorf("解析 HTML 模板失敗: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("執行 HTML 模板失敗: %w", err)
	}

	return buf.String(), nil
}

// HTML 模板
const emailVertifyTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email 認證</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #007bff; color: white; padding: 20px; text-align: center; }
        .content { padding: 30px; background-color: #f9f9f9; }
        .button { display: inline-block; padding: 12px 30px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { padding: 20px; text-align: center; font-size: 12px; color: #666; }
        .warning { color: #e74c3c; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>歡迎加入 {{.CompanyName}}</h1>
        </div>
        
        <div class="content">
            <p>感謝您註冊 {{.CompanyName}} 帳號！</p>
            
            <p>為了確保您的帳號安全，請點擊下方按鈕完成 Email 認證：</p>
            
            <div style="text-align: center;">
                <a href="{{.VerificationURL}}" class="button">點擊認證 Email</a>
            </div>
            
            <p>或複製以下連結到瀏覽器中開啟：</p>
            <p style="word-break: break-all; background-color: #e9e9e9; padding: 10px; border-radius: 3px;">
                {{.VerificationURL}}
            </p>
            
            <p class="warning">
                ⚠️ 此認證連結將在 {{.ExpiryMinutes}} 分鐘後失效，請盡快完成認證。
            </p>
            
            <p>如果您沒有註冊此帳號，請忽略此郵件或聯繫我們的客服。</p>
        </div>
        
        <div class="footer">
            <p>此郵件由系統自動發送，請勿直接回覆。</p>
            <p>如有疑問，請聯繫客服：<a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>
            <p>&copy; {{.CompanyName}} - 版權所有</p>
        </div>
    </div>
</body>
</html>
`
