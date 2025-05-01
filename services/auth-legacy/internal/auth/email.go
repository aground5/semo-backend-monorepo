package auth

import (
	"authn-server/configs"
	"authn-server/internal/logics"
	"authn-server/internal/repositories"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

// EmailVerifier는 이메일 인증 토큰을 관리합니다.
type EmailVerifier struct{}

var (
	// EmailVerifier 전역 인스턴스
	emailVerifier EmailVerifierInterface
)

// GetEmailVerifier는 전역 EmailVerifier 인스턴스를 반환합니다.
func GetEmailVerifier() EmailVerifierInterface {
	if emailVerifier == nil {
		emailVerifier = NewEmailVerifier()
	}
	return emailVerifier
}

// NewEmailVerifier는 EmailVerifier 인스턴스를 생성합니다.
func NewEmailVerifier() EmailVerifierInterface {
	return &EmailVerifier{}
}

// GenerateVerificationToken은 이메일 인증용 토큰을 생성하고 Redis에 저장합니다.
func (ev *EmailVerifier) GenerateVerificationToken(email string) (string, error) {
	token := GenerateRandomString(32)
	ctx := context.Background()

	// 양방향 매핑 저장
	tokenKey := fmt.Sprintf("%s%s", TokenKeyPrefix, token)
	emailKey := fmt.Sprintf("%s%s", EmailKeyPrefix, email)

	pipe := repositories.DBS.Redis.Pipeline()
	pipe.Set(ctx, tokenKey, email, 24*time.Hour)
	pipe.Set(ctx, emailKey, token, 24*time.Hour)
	_, err := pipe.Exec(ctx)

	return token, err
}

// VerifyToken은 인증 토큰을 확인하고 연결된 이메일을 반환합니다.
func (ev *EmailVerifier) VerifyToken(token string) (string, error) {
	ctx := context.Background()

	tokenKey := fmt.Sprintf("%s%s", TokenKeyPrefix, token)
	email, err := repositories.DBS.Redis.Get(ctx, tokenKey).Result()

	if err != nil {
		if err == redis.Nil {
			return "", NewAuthError(ErrInvalidToken, "유효하지 않거나 만료된 인증 토큰입니다")
		}
		return "", fmt.Errorf("토큰 조회 중 오류 발생: %w", err)
	}

	return email, nil
}

// SendVerificationEmail은 사용자에게 인증 이메일을 발송합니다.
func (ev *EmailVerifier) SendVerificationEmail(email, name, token string) error {
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s",
		configs.Configs.Service.BaseURL, token)

	subject := "이메일 주소 인증"
	body := logics.EmailSvc.GenerateVerificationEmailHTML(name, verificationLink, token)

	return logics.EmailSvc.SendEmail(
		configs.Configs.Email.SenderEmail,
		email,
		subject,
		body,
	)
}

// DeleteTokens는 이메일 인증 후 토큰과 관련 키를 삭제합니다.
func (ev *EmailVerifier) DeleteTokens(token, email string) error {
	ctx := context.Background()

	tokenKey := fmt.Sprintf("%s%s", TokenKeyPrefix, token)
	emailKey := fmt.Sprintf("%s%s", EmailKeyPrefix, email)

	pipe := repositories.DBS.Redis.Pipeline()
	pipe.Del(ctx, tokenKey)
	pipe.Del(ctx, emailKey)
	_, err := pipe.Exec(ctx)

	return err
}
