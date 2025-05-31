package service

import (
	"context"

	"github.com/RoyceAzure/lab/authcenter/internal/model"
	"github.com/RoyceAzure/rj/infra/mail"
)

type IMailService interface {
	SendVertifyEmail(ctx context.Context, mailContent *model.MailContent) error
}

type MailService struct {
	mail.EmailSender
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

func (m *MailService) SendVertifyEmail(ctx context.Context, mailContent *model.MailContent) error {
	return m.SendEmail(mailContent.Subject, mailContent.Content, mailContent.To, mailContent.CC, mailContent.BCC, mailContent.AttachFiles)
}
