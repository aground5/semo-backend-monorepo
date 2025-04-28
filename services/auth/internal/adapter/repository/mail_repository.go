package repository

import (
	"context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/mail"
)

// MailRepository SMTP 메일 저장소 어댑터
type MailRepository struct {
	client *mail.SMTPClient
}

// NewMailRepository 메일 레포지토리 어댑터 생성
func NewMailRepository(client *mail.SMTPClient) repository.MailRepository {
	return &MailRepository{
		client: client,
	}
}

// SendMail 이메일 발송
func (m *MailRepository) SendMail(ctx context.Context, to, subject, body string) error {
	return m.client.SendMail(ctx, to, subject, body)
}

// SendMailWithAttachment 첨부 파일이 있는 이메일 발송
func (m *MailRepository) SendMailWithAttachment(ctx context.Context, to, subject, body string, attachments map[string][]byte) error {
	return m.client.SendMailWithAttachment(ctx, to, subject, body, attachments)
}
