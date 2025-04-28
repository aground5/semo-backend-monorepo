package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserUseCase 사용자 프로필, 비밀번호 관리 유스케이스 구현체
type UserUseCase struct {
	logger          *zap.Logger
	userRepository  repository.UserRepository
	tokenRepository repository.TokenRepository
	auditRepository repository.AuditLogRepository
}

// NewUserUseCase 새 사용자 유스케이스 생성
func NewUserUseCase(
	logger *zap.Logger,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	auditRepo repository.AuditLogRepository,
) *UserUseCase {
	return &UserUseCase{
		logger:          logger,
		userRepository:  userRepo,
		tokenRepository: tokenRepo,
		auditRepository: auditRepo,
	}
}

// GetUserProfile 사용자 프로필 조회
func (uc *UserUseCase) GetUserProfile(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
		}
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	return user, nil
}

// UpdateUserProfile 사용자 프로필 업데이트
func (uc *UserUseCase) UpdateUserProfile(ctx context.Context, userID string, name string) (*entity.User, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
		}
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 이름 업데이트
	user.Name = name

	// 3. 사용자 정보 업데이트
	if err := uc.userRepository.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("프로필 업데이트 실패: %w", err)
	}

	// 4. 프로필 업데이트 감사 로그 기록
	uc.logUserActivity(ctx, userID, entity.AuditLogTypeUserProfileUpdate, map[string]interface{}{
		"field": "name",
		"value": name,
	})

	return user, nil
}

// ChangePassword 비밀번호 변경
func (uc *UserUseCase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
		}
		return fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 현재 비밀번호 검증
	if !uc.verifyPassword(user.Password, currentPassword) {
		// 비밀번호 변경 실패 감사 로그 기록
		uc.logUserActivity(ctx, userID, entity.AuditLogTypePasswordChangeFailed, map[string]interface{}{
			"reason": "incorrect_current_password",
		})
		return fmt.Errorf("현재 비밀번호가 일치하지 않습니다")
	}

	// 3. 새 비밀번호 강도 검증
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 4. 새 비밀번호 해싱
	hashedPassword, _, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("비밀번호 해싱 실패: %w", err)
	}

	// 5. 비밀번호 업데이트
	user.Password = hashedPassword
	if err := uc.userRepository.Update(ctx, user); err != nil {
		return fmt.Errorf("비밀번호 업데이트 실패: %w", err)
	}

	// 6. 토큰 폐기 - 보안상 다른 세션 모두 로그아웃 처리
	// 실제 구현에서는 현재 세션은 유지하도록 조정 필요
	if err := uc.revokeUserTokens(ctx, userID); err != nil {
		uc.logger.Warn("사용자 토큰 폐기 실패",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	// 7. 비밀번호 변경 성공 감사 로그 기록
	uc.logUserActivity(ctx, userID, entity.AuditLogTypePasswordChange, nil)

	return nil
}

// RequestPasswordReset 비밀번호 재설정 요청
func (uc *UserUseCase) RequestPasswordReset(ctx context.Context, email string) error {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 보안상 사용자가 없어도 성공한 것처럼 응답
			uc.logger.Info("비밀번호 재설정 요청 - 사용자 없음", zap.String("email", email))
			return nil
		}
		return fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 비밀번호 재설정 토큰 생성
	// 실제 구현에서는 토큰 생성 및 이메일 발송 로직 필요

	// 3. 비밀번호 재설정 요청 감사 로그 기록
	uc.logUserActivity(ctx, user.ID, entity.AuditLogTypePasswordResetRequested, map[string]interface{}{
		"email": email,
	})

	return nil
}

// 보조 함수들

// logUserActivity 사용자 활동 감사 로그 기록
func (uc *UserUseCase) logUserActivity(ctx context.Context, userID string, activityType entity.AuditLogType, details map[string]interface{}) {
	// IP와 User-Agent 정보 추출
	clientIP := getContextValue(ctx, "client_ip")
	userAgent := getContextValue(ctx, "user_agent")

	// 상세 정보에 클라이언트 정보 추가
	if details == nil {
		details = make(map[string]interface{})
	}
	details["ip"] = clientIP
	details["user_agent"] = userAgent

	// 감사 로그 생성
	auditLog := entity.NewAuditLog(&userID, activityType, details)

	// 감사 로그 저장
	if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
		uc.logger.Error("감사 로그 저장 실패",
			zap.String("user_id", userID),
			zap.String("activity", string(activityType)),
			zap.Error(err),
		)
	}
}

// verifyPassword 비밀번호 검증
func (uc *UserUseCase) verifyPassword(hashedPassword, plainPassword string) bool {
	// 실제 구현에서는 bcrypt.CompareHashAndPassword 등 사용
	// 이 예제에서는 단순화를 위해 간단한 비교
	return hashedPassword == plainPassword+"_hashed"
}

// revokeUserTokens 사용자 토큰 폐기
func (uc *UserUseCase) revokeUserTokens(ctx context.Context, userID string) error {
	// 실제 구현에서는 토큰 데이터베이스에서 특정 사용자의 토큰 폐기
	// 이 예제에서는 미구현 상태
	return nil
}

// getContextValue Context에서 값 추출
func getContextValue(ctx context.Context, key string) string {
	value := ctx.Value(key)
	if value == nil {
		return "unknown"
	}
	strValue, ok := value.(string)
	if !ok {
		return "unknown"
	}
	return strValue
}
