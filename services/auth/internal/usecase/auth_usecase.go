package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/errors"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthUseCase 인증 유스케이스 구현체
type AuthUseCase struct {
	logger          *zap.Logger
	userRepository  repository.UserRepository
	tokenRepository repository.TokenRepository
	auditRepository repository.AuditLogRepository
	tokenUseCase    interfaces.TokenUseCase
	otpUseCase      interfaces.OTPUseCase
	emailUseCase    interfaces.EmailUseCase
}

// NewAuthUseCase 새 인증 유스케이스 생성
func NewAuthUseCase(
	logger *zap.Logger,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	auditRepo repository.AuditLogRepository,
	tokenUC interfaces.TokenUseCase,
	otpUC interfaces.OTPUseCase,
	emailUC interfaces.EmailUseCase,
) interfaces.AuthUseCase {
	return &AuthUseCase{
		logger:          logger,
		userRepository:  userRepo,
		tokenRepository: tokenRepo,
		auditRepository: auditRepo,
		tokenUseCase:    tokenUC,
		otpUseCase:      otpUC,
		emailUseCase:    emailUC,
	}
}

// Register 사용자 회원가입
func (uc *AuthUseCase) Register(ctx context.Context, params dto.RegisterParams) (*entity.User, error) {
	// 1. 이메일 형식 검증
	if !isValidEmail(params.Email) {
		return nil, fmt.Errorf("유효하지 않은 이메일 형식: %s", params.Email)
	}

	// 2. 비밀번호 강도 검증
	if err := validatePasswordStrength(params.Password); err != nil {
		return nil, err
	}

	// 3. 이미 존재하는 이메일인지 확인
	existingUser, err := uc.userRepository.FindByEmail(ctx, params.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("이메일 중복 확인 실패: %w", err)
	}

	if existingUser != nil {
		return nil, fmt.Errorf("이미 등록된 이메일입니다: %s", params.Email)
	}

	// 4. 비밀번호 해싱
	hashedPassword, salt, err := HashPassword(params.Password)
	if err != nil {
		return nil, fmt.Errorf("비밀번호 해싱 실패: %w", err)
	}

	// 사용자 이름이 없으면 이메일에서 추출
	username := params.Username
	if username == "" {
		username = ExtractUsernameFromEmail(params.Email)
	}

	// 5. 사용자 생성
	user := &entity.User{
		Email:         params.Email,
		Password:      hashedPassword,
		Salt:          salt,
		Username:      username,
		Name:          username,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := uc.userRepository.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("사용자 생성 실패: %w", err)
	}

	// 6. 이메일 인증 토큰 생성 및 전송
	go func() {
		bgCtx := context.Background()
		token, err := uc.emailUseCase.GenerateVerificationToken(bgCtx, params.Email)
		if err != nil {
			uc.logger.Error("인증 토큰 생성 실패", zap.Error(err))
			return
		}

		if err := uc.emailUseCase.SendVerificationEmail(bgCtx, params.Email, username, token); err != nil {
			uc.logger.Error("인증 이메일 발송 실패", zap.Error(err))
		}
	}()

	// 7. 감사 로그 기록
	go func() {
		content := map[string]interface{}{
			"email":      params.Email,
			"ip":         params.IP,
			"user_agent": params.UserAgent,
		}

		auditLog := &entity.AuditLog{
			UserID:  &user.ID,
			Type:    entity.AuditLogTypeUserRegistered,
			Content: content,
		}

		if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
			uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
		}
	}()

	return user, nil
}

// VerifyEmail 이메일 인증
func (uc *AuthUseCase) VerifyEmail(ctx context.Context, token string) (*entity.User, error) {
	// 1. 토큰으로 이메일 조회
	email, err := uc.emailUseCase.VerifyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// 2. 이메일로 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 3. 이미 인증된 경우 처리
	if user.EmailVerified {
		// 인증이 이미 완료되었으면 토큰만 삭제하고 성공 반환
		if err := uc.emailUseCase.DeleteTokens(ctx, token, email); err != nil {
			uc.logger.Warn("이미 인증된 토큰 삭제 실패", zap.Error(err))
		}
		return user, nil
	}

	// 4. 이메일 인증 상태 업데이트
	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := uc.userRepository.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("사용자 업데이트 실패: %w", err)
	}

	// 5. 토큰 삭제
	if err := uc.emailUseCase.DeleteTokens(ctx, token, email); err != nil {
		uc.logger.Warn("토큰 삭제 실패", zap.Error(err))
	}

	// 6. 감사 로그 기록 (emailUseCase.DeleteTokens에서 이미 기록됨)

	return user, nil
}

// ResendVerificationEmail 이메일 인증 재발송
func (uc *AuthUseCase) ResendVerificationEmail(ctx context.Context, email string) error {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("등록되지 않은 이메일입니다: %s", email)
		}
		return fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 이미 인증된 경우
	if user.EmailVerified {
		return fmt.Errorf("이미 인증된 이메일입니다: %s", email)
	}

	// 3. 새 인증 토큰 생성 및 발송
	token, err := uc.emailUseCase.GenerateVerificationToken(ctx, email)
	if err != nil {
		return fmt.Errorf("인증 토큰 생성 실패: %w", err)
	}

	if err := uc.emailUseCase.SendVerificationEmail(ctx, email, user.Name, token); err != nil {
		return fmt.Errorf("인증 이메일 발송 실패: %w", err)
	}

	return nil
}

// Login 로그인
func (uc *AuthUseCase) Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	// 1. 매직 코드 확인
	valid, err := uc.otpUseCase.VerifyMagicCode(ctx, params.Email, params.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("매직 코드 확인 실패: %w", err)
	}

	if !valid {
		// 로그인 실패 감사 로그 기록
		go func() {
			content := map[string]interface{}{
				"email":      params.Email,
				"ip":         params.IP,
				"user_agent": params.UserAgent,
				"reason":     "유효하지 않은 매직 코드",
			}

			auditLog := &entity.AuditLog{
				Type:    entity.AuditLogTypeLoginFailed,
				Content: content,
			}

			if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
				uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
			}
		}()

		return nil, nil, fmt.Errorf("유효하지 않거나 만료된 코드입니다")
	}

	// 2. 사용자 조회 또는 생성
	user, err := uc.findOrCreateUser(ctx, params.Email)
	if err != nil {
		return nil, nil, err
	}

	// 3. 토큰 그룹 조회 또는 생성
	tokenGroup, err := uc.findOrCreateTokenGroup(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// 4. 액세스 토큰 생성
	accessToken, err := uc.tokenUseCase.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 5. 리프레시 토큰 생성
	refreshToken, tokenRecord, err := uc.tokenUseCase.GenerateRefreshToken(ctx, tokenGroup.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("리프레시 토큰 생성 실패: %w", err)
	}

	// 6. 리프레시 토큰 저장
	if err := uc.tokenRepository.Create(ctx, tokenRecord); err != nil {
		return nil, nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 7. 마지막 로그인 시간 업데이트
	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now

	if err := uc.userRepository.Update(ctx, user); err != nil {
		uc.logger.Warn("마지막 로그인 시간 업데이트 실패", zap.Error(err))
	}

	// 8. 로그인 성공 감사 로그 기록
	go func() {
		content := map[string]interface{}{
			"email":      params.Email,
			"ip":         params.IP,
			"user_agent": params.UserAgent,
			"login_type": "magic_code",
		}

		auditLog := &entity.AuditLog{
			UserID:  &user.ID,
			Type:    entity.AuditLogTypeLoginSuccess,
			Content: content,
		}

		if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
			uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
		}
	}()

	// 9. 토큰 응답 생성 (만료 시간은 30분으로 가정)
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	return tokens, user, nil
}

// 헬퍼 함수들

// findOrCreateUser 사용자를 이메일로 찾거나 새로 생성
func (uc *AuthUseCase) findOrCreateUser(ctx context.Context, email string) (*entity.User, error) {
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err == nil {
		// 사용자 찾음
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 예상치 못한 오류
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 사용자 없음, 새로 생성
	username := ExtractUsernameFromEmail(email)
	now := time.Now()

	newUser := &entity.User{
		Email:         email,
		Username:      username,
		Name:          username,
		EmailVerified: true, // 매직 링크로 로그인했으므로 이메일은 인증됨
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := uc.userRepository.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("사용자 생성 실패: %w", err)
	}

	return newUser, nil
}

// findOrCreateTokenGroup 토큰 그룹을 찾거나 새로 생성
func (uc *AuthUseCase) findOrCreateTokenGroup(ctx context.Context, userID string) (*entity.TokenGroup, error) {
	tokenGroup, err := uc.tokenRepository.FindGroupByUserID(ctx, userID)
	if err == nil && tokenGroup != nil {
		// 토큰 그룹 찾음
		return tokenGroup, nil
	}

	// 새 토큰 그룹 생성
	newTokenGroup := &entity.TokenGroup{
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	if err := uc.tokenRepository.CreateGroup(ctx, newTokenGroup); err != nil {
		return nil, fmt.Errorf("토큰 그룹 생성 실패: %w", err)
	}

	return newTokenGroup, nil
}

// isValidEmail 이메일 형식 검증
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// validatePasswordStrength 비밀번호 강도 검증
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("비밀번호는 최소 8자 이상이어야 합니다")
	}

	// 대문자 포함 여부
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// 소문자 포함 여부
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// 숫자 포함 여부
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	// 특수문자 포함 여부
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	// 3가지 이상의 조합 검증
	count := 0
	if hasUpper {
		count++
	}
	if hasLower {
		count++
	}
	if hasNumber {
		count++
	}
	if hasSpecial {
		count++
	}

	if count < 3 {
		return fmt.Errorf("비밀번호는 대문자, 소문자, 숫자, 특수문자 중 3가지 이상을 포함해야 합니다")
	}

	return nil
}

// GetUserProfile 사용자 프로필 조회
func (uc *AuthUseCase) GetUserProfile(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.UserNotFoundError(userID)
	}

	return user, nil
}

// UpdateUserProfile 사용자 프로필 업데이트
func (uc *AuthUseCase) UpdateUserProfile(ctx context.Context, userID, name string) (*entity.User, error) {
	// 1) 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.UserNotFoundError(userID)
	}

	// 2) 이름 업데이트
	user.Name = name

	// 3) 사용자 정보 업데이트
	if err := uc.userRepository.Update(ctx, user); err != nil {
		return nil, errors.Wrap(err, "프로필 업데이트 실패")
	}

	// 4) 프로필 업데이트 감사 로그 기록
	go func() {
		auditLog := &entity.AuditLog{
			UserID:    user.ID,
			Type:      entity.AuditLogTypeProfileUpdate,
			IP:        getClientIP(ctx),
			UserAgent: getUserAgent(ctx),
			Timestamp: time.Now(),
		}

		if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
			uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
		}
	}()

	return user, nil
}

// ChangePassword 비밀번호 변경
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	// 1) 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		return errors.UserNotFoundError(userID)
	}

	// 2) 현재 비밀번호 검증
	if !checkPasswordHash(currentPassword, user.Password) {
		// 비밀번호 변경 실패 감사 로그 기록
		go func() {
			auditLog := &entity.AuditLog{
				UserID:    user.ID,
				Type:      entity.AuditLogTypePasswordChangeFailure,
				IP:        getClientIP(ctx),
				UserAgent: getUserAgent(ctx),
				Timestamp: time.Now(),
			}

			if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
				uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
			}
		}()

		return errors.InvalidPasswordError()
	}

	// 3) 새 비밀번호 강도 검증
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 4) 새 비밀번호 해싱
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return errors.Wrap(err, "비밀번호 해싱 실패")
	}

	// 5) 비밀번호 업데이트
	user.Password = hashedPassword
	if err := uc.userRepository.Update(ctx, user); err != nil {
		return errors.Wrap(err, "비밀번호 업데이트 실패")
	}

	// 6) 해당 사용자의 모든 토큰 폐기 (현재 세션 제외)
	// 현재 액세스 토큰은 유지하면서 나머지 토큰 그룹만 폐기하는 로직 필요
	// 이 예제에서는 단순화를 위해 모든 토큰을 폐기
	if err := uc.tokenRepository.DeleteByUser(ctx, user.ID); err != nil {
		uc.logger.Error("사용자 토큰 폐기 실패", zap.Error(err))
	}

	// 7) 비밀번호 변경 성공 감사 로그 기록
	go func() {
		auditLog := &entity.AuditLog{
			UserID: user.ID,
			Type:   entity.AuditLogTypePasswordChange,
		}

		auditLog.AddContentField("ip", getClientIP(ctx))
		auditLog.AddContentField("user_agent", getUserAgent(ctx))

		if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
			uc.logger.Error("감사 로그 저장 실패", zap.Error(err))
		}
	}()

	return nil
}

// 내부 헬퍼 함수들

// logLoginAttempt 로그인 시도 감사 로그 기록
func (uc *AuthUseCase) logLoginAttempt(ctx context.Context, userID, email string, logType entity.AuditLogType, reason string) {
	auditLog := &entity.AuditLog{
		UserID:    userID,
		Email:     email, // 사용자 ID가 없을 경우 이메일 기록
		Type:      logType,
		IP:        getClientIP(ctx),
		UserAgent: getUserAgent(ctx),
		Message:   reason,
		Timestamp: time.Now(),
	}

	if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
		uc.logger.Error("로그인 시도 감사 로그 저장 실패", zap.Error(err))
	}
}

// getClientIP 클라이언트 IP 주소 추출
func getClientIP(ctx context.Context) string {
	// Context에서 IP 정보 추출하는 로직
	// 실제 구현은 프레임워크나 미들웨어에 따라 달라질 수 있음
	ip, ok := ctx.Value("client_ip").(string)
	if !ok || ip == "" {
		return "unknown"
	}
	return ip
}

// getUserAgent 사용자 에이전트 추출
func getUserAgent(ctx context.Context) string {
	// Context에서 User-Agent 정보 추출하는 로직
	// 실제 구현은 프레임워크나 미들웨어에 따라 달라질 수 있음
	ua, ok := ctx.Value("user_agent").(string)
	if !ok || ua == "" {
		return "unknown"
	}
	return ua
}
