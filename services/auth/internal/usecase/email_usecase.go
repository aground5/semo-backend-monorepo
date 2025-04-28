package usecase

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/mail"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/constants"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// EmailUseCase 이메일 인증 유스케이스 구현체
type EmailUseCase struct {
	logger           *zap.Logger
	cacheRepository  repository.CacheRepository
	mailRepository   repository.MailRepository
	userRepository   repository.UserRepository
	auditRepository  repository.AuditLogRepository
	smtpClient       *smtp.Client
	emailTemplate    *mail.EmailTemplateService
	appURL           string
	emailSenderEmail string
	companyName      string
}

// NewEmailUseCase 새 이메일 유스케이스 생성
func NewEmailUseCase(
	logger *zap.Logger,
	cacheRepo repository.CacheRepository,
	mailRepo repository.MailRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	smtpClient *smtp.Client,
	emailTemplate *mail.EmailTemplateService,
	appURL string,
	emailSenderEmail string,
	companyName string,
) interfaces.EmailUseCase {
	return &EmailUseCase{
		logger:           logger,
		cacheRepository:  cacheRepo,
		mailRepository:   mailRepo,
		userRepository:   userRepo,
		auditRepository:  auditRepo,
		smtpClient:       smtpClient,
		emailTemplate:    emailTemplate,
		appURL:           appURL,
		emailSenderEmail: emailSenderEmail,
		companyName:      companyName,
	}
}

// GenerateVerificationToken 이메일 인증 토큰 생성
func (uc *EmailUseCase) GenerateVerificationToken(ctx context.Context, email string) (string, error) {
	// 랜덤 토큰 생성
	token := GenerateRandomString(32)

	// 양방향 매핑 저장
	tokenKey := fmt.Sprintf("%s%s", constants.TokenKeyPrefix, token)
	emailKey := fmt.Sprintf("%s%s", constants.EmailKeyPrefix, email)

	expiry := time.Duration(constants.VerificationTokenExpiry) * time.Hour

	// 트랜잭션으로 두 키 모두 저장
	items := map[string]string{
		tokenKey: email,
		emailKey: token,
	}

	if err := uc.cacheRepository.SetMulti(ctx, items, expiry); err != nil {
		uc.logger.Error("이메일 인증 토큰 생성 실패",
			zap.String("email", email),
			zap.Error(err),
		)
		return "", fmt.Errorf("인증 토큰 저장 실패: %w", err)
	}

	return token, nil
}

// VerifyToken 인증 토큰 확인
func (uc *EmailUseCase) VerifyToken(ctx context.Context, token string) (string, error) {
	tokenKey := fmt.Sprintf("%s%s", constants.TokenKeyPrefix, token)

	// 토큰으로 이메일 조회
	email, err := uc.cacheRepository.Get(ctx, tokenKey)
	if err != nil {
		if uc.cacheRepository.IsNotFound(err) {
			return "", fmt.Errorf("유효하지 않거나 만료된 인증 토큰")
		}
		return "", fmt.Errorf("토큰 조회 실패: %w", err)
	}

	return email, nil
}

// SendVerificationEmail 인증 이메일 발송
func (uc *EmailUseCase) SendVerificationEmail(ctx context.Context, email, name, token string) error {
	// 이메일 서비스를 사용하여 인증 이메일 발송
	html := uc.emailTemplate.GenerateVerificationEmailHTML(name, token)
	if err := uc.mailRepository.SendMail(ctx, email, "[SEMO] 이메일 인증 코드 입니다.", html); err != nil {
		uc.logger.Error("인증 이메일 발송 실패",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("이메일 발송 실패: %w", err)
	}

	return nil
}

// DeleteTokens 인증 완료 후 토큰 삭제
func (uc *EmailUseCase) DeleteTokens(ctx context.Context, token, email string) error {
	tokenKey := fmt.Sprintf("%s%s", constants.TokenKeyPrefix, token)
	emailKey := fmt.Sprintf("%s%s", constants.EmailKeyPrefix, email)

	// 두 키 모두 삭제
	keys := []string{tokenKey, emailKey}
	if err := uc.cacheRepository.DeleteMulti(ctx, keys); err != nil {
		uc.logger.Error("토큰 삭제 실패",
			zap.String("token", token),
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("토큰 삭제 실패: %w", err)
	}

	// 감사 로그 생성
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err == nil && user != nil {
		// 유저가 있는 경우에만 감사 로그 기록
		content := map[string]interface{}{
			"email": email,
			"token": token,
		}
		auditLog := &entity.AuditLog{
			UserID:  &user.ID,
			Type:    entity.AuditLogTypeEmailVerified,
			Content: content,
		}
		if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
			uc.logger.Warn("이메일 인증 감사 로그 저장 실패", zap.Error(err))
		}
	}

	return nil
}
