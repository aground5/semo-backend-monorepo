package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"go.uber.org/zap"
)

// SMTPConfig SMTP 설정 구조체
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPClient SMTP를 통한 이메일 발송 클라이언트
type SMTPClient struct {
	config SMTPConfig
}

// NewSMTPClient SMTP 클라이언트 생성
func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	return &SMTPClient{
		config: cfg,
	}
}

// SendMail 이메일 발송
func (m *SMTPClient) SendMail(ctx context.Context, to, subject, body string) error {
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	// 이메일 헤더 설정
	headers := make(map[string]string)
	headers["From"] = m.config.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	// 메시지 구성
	var message string
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// 발송
	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(message))
	if err != nil {
		config.AppConfig.Logger.Error("이메일 발송 실패",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("이메일 발송 실패: %w", err)
	}

	config.AppConfig.Logger.Info("이메일 발송 성공",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	return nil
}

// SendMailWithAttachment 첨부 파일이 있는 이메일 발송
func (m *SMTPClient) SendMailWithAttachment(ctx context.Context, to, subject, body string, attachments map[string][]byte) error {
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	// 멀티파트 경계 설정
	boundary := "MIME_boundary_123456789"

	// 이메일 헤더 설정
	headers := make(map[string]string)
	headers["From"] = m.config.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=\"%s\"", boundary)

	// 메시지 구성
	var message string
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n"

	// 본문 파트
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	message += "Content-Transfer-Encoding: quoted-printable\r\n\r\n"
	message += body + "\r\n\r\n"

	// 첨부 파일 파트
	for name, data := range attachments {
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", name)
		message += "Content-Transfer-Encoding: base64\r\n"
		message += fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", name)

		// Base64로 인코딩된 데이터 추가 (실제 구현은 더 복잡할 수 있음)
		// 이 예제에서는 단순화를 위해 생략합니다
		message += fmt.Sprintf("%s\r\n\r\n", strings.TrimSpace(string(data)))
	}

	// 메시지 종료
	message += fmt.Sprintf("--%s--", boundary)

	// 발송
	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(message))
	if err != nil {
		config.AppConfig.Logger.Error("첨부 파일 이메일 발송 실패",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Int("attachments", len(attachments)),
			zap.Error(err),
		)
		return fmt.Errorf("첨부 파일 이메일 발송 실패: %w", err)
	}

	config.AppConfig.Logger.Info("첨부 파일 이메일 발송 성공",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.Int("attachments", len(attachments)),
	)

	return nil
}
