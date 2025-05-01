package auth

import (
	"authn-server/configs"
	"authn-server/internal/logics"
	"authn-server/internal/repositories"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strings"
	"time"
)

// OtpService는 일회용 패스워드 및 매직 코드 기능을 제공합니다.
type OtpService struct{}

var (
	// OtpService 전역 인스턴스
	otpService OtpServiceInterface
)

// GetOtpService는 전역 OtpService 인스턴스를 반환합니다.
func GetOtpService() OtpServiceInterface {
	if otpService == nil {
		otpService = NewOtpService()
	}
	return otpService
}

// NewOtpService는 OtpService 인스턴스를 생성합니다.
func NewOtpService() OtpServiceInterface {
	return &OtpService{}
}

// SendMagicCode는 사용자에게 매직 코드를 생성하고 이메일로 전송합니다.
func (svc *OtpService) SendMagicCode(email, ip, userAgent string) error {
	// 6자리 무작위 코드 생성
	code := strings.ToUpper(GenerateRandomCode(6))

	// Redis에 코드 저장 (5분 TTL)
	key := fmt.Sprintf("%s%s", MagicCodePrefix, strings.ToLower(email))
	ctx := context.Background()
	err := repositories.DBS.Redis.Set(ctx, key, code, 5*time.Minute).Err()
	if err != nil {
		configs.Logger.Error("Redis에 매직 코드 저장 실패", zap.Error(err))
		return fmt.Errorf("매직 코드 저장 실패: %w", err)
	}

	// 이메일 발송
	err = logics.EmailSvc.SendEmail(
		configs.Configs.Email.SenderEmail,
		email,
		"계정 접속 코드",
		logics.EmailSvc.GenerateWelcomeEmailHTML(email[:strings.Index(email, "@")], code),
	)
	if err != nil {
		configs.Logger.Error("이메일 발송 실패", zap.Error(err))
		return fmt.Errorf("이메일 발송 실패: %w", err)
	}

	// 감사 로그 기록
	LogUserAction(context.Background(), AuditLogTypeMagicCodeGenerated, email, ip, userAgent, nil)

	configs.Logger.Info("매직 코드 생성 완료")
	return nil
}

// VerifyMagicCode는 사용자가 제출한 매직 코드의 유효성을 검사합니다.
func (svc *OtpService) VerifyMagicCode(email, code string) (bool, error) {
	key := fmt.Sprintf("%s%s", MagicCodePrefix, strings.ToLower(email))
	ctx := context.Background()

	storedCode, err := repositories.DBS.Redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // 코드가 없거나 만료됨
		}
		return false, err // Redis 오류
	}

	// 코드가 일치하면 삭제하고 true 반환
	if storedCode == code {
		repositories.DBS.Redis.Del(ctx, key)
		return true, nil
	}

	return false, nil
}
