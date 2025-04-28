package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/constants"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
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
	appURL           string
	emailSenderEmail string
}

// NewEmailUseCase 새 이메일 유스케이스 생성
func NewEmailUseCase(
	logger *zap.Logger,
	cacheRepo repository.CacheRepository,
	mailRepo repository.MailRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	appURL string,
	emailSenderEmail string,
) interfaces.EmailUseCase {
	return &EmailUseCase{
		logger:           logger,
		cacheRepository:  cacheRepo,
		mailRepository:   mailRepo,
		userRepository:   userRepo,
		auditRepository:  auditRepo,
		appURL:           appURL,
		emailSenderEmail: emailSenderEmail,
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
	// 인증 링크 생성
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", uc.appURL, token)

	// 이메일 데이터 생성
	data := dto.VerificationEmailData{
		Name:             name,
		VerificationLink: verificationLink,
		Token:            token,
		ExpireHours:      constants.VerificationTokenExpiry,
	}

	// 이메일 내용 생성
	subject := "이메일 주소 인증"
	body := fmt.Sprintf(`
		<h1>안녕하세요, %s님!</h1>
		<p>아래 링크를 클릭하여 이메일 주소를 인증해 주세요:</p>
		<p><a href="%s">이메일 인증하기</a></p>
		<p>또는 다음 코드를 입력하세요: %s</p>
		<p>이 링크는 %d시간 동안 유효합니다.</p>
	`, name, verificationLink, token, constants.VerificationTokenExpiry)

	// 이메일 발송
	err := uc.mailRepository.SendMail(ctx, email, subject, body)
	if err != nil {
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
