package repository

import (
	"context"
)

// MailRepository는 이메일 발송을 위한 인터페이스입니다.
type MailRepository interface {
	// SendMail 이메일을 발송합니다.
	SendMail(ctx context.Context, to string, subject string, body string) error

	// SendMailWithAttachment 첨부 파일이 있는 이메일을 발송합니다.
	SendMailWithAttachment(ctx context.Context, to string, subject string, body string, attachments map[string][]byte) error
}
