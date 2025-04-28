package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/constants"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// OTPUseCase OTP 유스케이스 구현체
type OTPUseCase struct {
	logger          *zap.Logger
	cacheRepository repository.CacheRepository
	mailRepository  repository.MailRepository
	auditRepository repository.AuditLogRepository
}

// NewOTPUseCase 새 OTP 유스케이스 생성
func NewOTPUseCase(
	logger *zap.Logger,
	cacheRepo repository.CacheRepository,
	mailRepo repository.MailRepository,
	auditRepo repository.AuditLogRepository,
) interfaces.OTPUseCase {
	return &OTPUseCase{
		logger:          logger,
		cacheRepository: cacheRepo,
		mailRepository:  mailRepo,
		auditRepository: auditRepo,
	}
}

// SendMagicCode 매직 코드 발송
func (uc *OTPUseCase) SendMagicCode(ctx context.Context, email, ip, userAgent string) error {
	// 6자리 랜덤 코드 생성
	code := strings.ToUpper(GenerateRandomCode(6))

	// 캐시에 코드 저장 (5분 TTL)
	key := fmt.Sprintf("%s%s", constants.MagicCodePrefix, strings.ToLower(email))
	expiry := time.Duration(constants.MagicCodeExpiry) * time.Minute

	err := uc.cacheRepository.Set(ctx, key, code, expiry)
	if err != nil {
		uc.logger.Error("매직 코드 저장 실패",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("매직 코드 저장 실패: %w", err)
	}

	// 이름 추출 (이메일의 @ 앞 부분)
	name := ExtractUsernameFromEmail(email)

	// 이메일 내용 생성
	subject := "로그인 코드"
	body := fmt.Sprintf(`
		<h1>안녕하세요, %s님!</h1>
		<p>로그인을 위한 인증 코드입니다:</p>
		<h2 style="font-size: 24px; letter-spacing: 5px; text-align: center; padding: 10px; background-color: #f0f0f0; border-radius: 4px;">%s</h2>
		<p>이 코드는 %d분 동안 유효합니다.</p>
	`, name, code, constants.MagicCodeExpiry)

	// 이메일 발송
	err = uc.mailRepository.SendMail(ctx, email, subject, body)
	if err != nil {
		uc.logger.Error("매직 코드 이메일 발송 실패",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("이메일 발송 실패: %w", err)
	}

	// 감사 로그 기록
	content := map[string]interface{}{
		"email":      email,
		"ip":         ip,
		"user_agent": userAgent,
	}

	if err := uc.auditRepository.Create(ctx, &entity.AuditLog{
		Type:      entity.AuditLogTypeMagicCodeGenerated,
		Content:   content,
		Timestamp: time.Now(),
	}); err != nil {
		uc.logger.Warn("매직 코드 생성 감사 로그 저장 실패",
			zap.String("email", email),
			zap.Error(err),
		)
	}

	uc.logger.Info("매직 코드 생성 완료", zap.String("email", email))
	return nil
}

// VerifyMagicCode 매직 코드 검증
func (uc *OTPUseCase) VerifyMagicCode(ctx context.Context, email, code string) (bool, error) {
	key := fmt.Sprintf("%s%s", constants.MagicCodePrefix, strings.ToLower(email))

	// 저장된 코드 조회
	storedCode, err := uc.cacheRepository.Get(ctx, key)
	if err != nil {
		if uc.cacheRepository.IsNotFound(err) {
			return false, nil // 코드가 없거나 만료됨
		}
		return false, fmt.Errorf("매직 코드 조회 실패: %w", err)
	}

	// 대소문자 구분 없이 코드 비교
	if strings.EqualFold(storedCode, code) {
		// 검증 성공 시 코드 삭제
		if err := uc.cacheRepository.Delete(ctx, key); err != nil {
			uc.logger.Warn("사용된 매직 코드 삭제 실패",
				zap.String("email", email),
				zap.Error(err),
			)
		}
		return true, nil
	}

	return false, nil
}

// GenerateRandomCode 랜덤 코드 생성
func (uc *OTPUseCase) GenerateRandomCode(length int) string {
	return GenerateRandomCode(length)
}
