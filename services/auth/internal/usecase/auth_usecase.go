package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthUseCase 회원가입, 이메일 인증 관련 유스케이스 구현체
type AuthUseCase struct {
	logger          *zap.Logger
	userRepository  repository.UserRepository
	auditRepository repository.AuditLogRepository
	tokenRepository repository.TokenRepository
	auditLogUseCase interfaces.AuditLogUseCase
	emailUseCase    interfaces.EmailUseCase
	tokenUseCase    interfaces.TokenUseCase
	otpUseCase      interfaces.OTPUseCase
}

// NewAuthUseCase 새 인증 유스케이스 생성
func NewAuthUseCase(
	logger *zap.Logger,
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	auditRepo repository.AuditLogRepository,
	auditLogUC interfaces.AuditLogUseCase,
	emailUC interfaces.EmailUseCase,
	tokenUC interfaces.TokenUseCase,
	otpUC interfaces.OTPUseCase,
) *AuthUseCase {
	return &AuthUseCase{
		logger:          logger,
		userRepository:  userRepo,
		tokenRepository: tokenRepo,
		auditRepository: auditRepo,
		auditLogUseCase: auditLogUC,
		emailUseCase:    emailUC,
		tokenUseCase:    tokenUC,
		otpUseCase:      otpUC,
	}
}

// Register 사용자 회원가입
func (uc *AuthUseCase) Register(ctx context.Context, params dto.RegisterParams) (*entity.User, error) {
	// 1. 이메일 형식 검증
	if !isValidEmail(params.Email) {
		return nil, fmt.Errorf("유효하지 않은 이메일 형식: %s", params.Email)
	}

	// 2. 비밀번호 강도 검증
	if err := ValidatePasswordStrength(params.Password); err != nil {
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
		ip := ctx.Value("client_ip")
		if ip == nil {
			ip = "unknown"
		}

		userAgent := ctx.Value("user_agent")
		if userAgent == nil {
			userAgent = "unknown"
		}

		content := map[string]interface{}{
			"email":      params.Email,
			"ip":         ip,
			"user_agent": userAgent,
		}

		if err := uc.auditLogUseCase.AddLog(context.Background(), entity.AuditLogTypeUserRegistered, content, &user.ID); err != nil {
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

	if err := uc.userRepository.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("사용자 업데이트 실패: %w", err)
	}

	// 5. 토큰 삭제
	if err := uc.emailUseCase.DeleteTokens(ctx, token, email); err != nil {
		uc.logger.Warn("토큰 삭제 실패", zap.Error(err))
	}

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

func (uc *AuthUseCase) Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	// 1. 매직 코드 확인
	valid, err := uc.otpUseCase.VerifyMagicCode(ctx, params.Email, params.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("매직 코드 확인 실패: %w", err)
	}

	if !valid {
		// 로그인 실패 감사 로그 기록
		uc.logLoginAttempt(ctx, "", params.Email, "유효하지 않은 매직 코드", entity.AuditLogTypeLoginFailed)
		return nil, nil, fmt.Errorf("유효하지 않거나 만료된 코드입니다")
	}

	// 2. 사용자 조회 또는 생성
	user, err := uc.findOrCreateUser(ctx, params.Email)
	if err != nil {
		return nil, nil, err
	}

	// 3. 토큰 그룹 조회 또는 생성
	tokenGroup, err := uc.tokenRepository.FindOrCreateTokenGroup(ctx, user.ID)
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

	if err := uc.userRepository.Update(ctx, user); err != nil {
		uc.logger.Warn("마지막 로그인 시간 업데이트 실패", zap.Error(err))
	}

	// 8. 로그인 성공 감사 로그 기록
	uc.logLoginAttempt(ctx, user.ID, params.Email, "매직 코드 로그인", entity.AuditLogTypeLoginSuccess)

	// 9. 토큰 응답 생성 (만료 시간은 30분으로 가정)
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return tokens, user, nil
}

// LoginWithPassword 비밀번호로 로그인
func (uc *AuthUseCase) LoginWithPassword(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, params.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logLoginAttempt(ctx, "", params.Email, "존재하지 않는 이메일", entity.AuditLogTypeLoginFailed)
			return nil, nil, fmt.Errorf("존재하지 않는 이메일입니다")
		}
		return nil, nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 비밀번호 검증
	err = VerifyPassword(user.Password, params.Password, user.Salt)
	if err != nil {
		uc.logLoginAttempt(ctx, user.ID, params.Email, "잘못된 비밀번호", entity.AuditLogTypeLoginFailed)
		return nil, nil, fmt.Errorf("잘못된 비밀번호입니다")
	}

	// 3. 이메일 인증 확인
	if !user.EmailVerified {
		uc.logLoginAttempt(ctx, user.ID, params.Email, "이메일 미인증", entity.AuditLogTypeLoginFailed)
		return nil, nil, fmt.Errorf("이메일 인증이 필요합니다")
	}

	// 4. 토큰 그룹 조회 또는 생성
	tokenGroup, err := uc.tokenRepository.FindOrCreateTokenGroup(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// 5. 액세스 토큰 생성
	accessToken, err := uc.tokenUseCase.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 6. 리프레시 토큰 생성
	refreshToken, tokenRecord, err := uc.tokenUseCase.GenerateRefreshToken(ctx, tokenGroup.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("리프레시 토큰 생성 실패: %w", err)
	}

	// 7. 리프레시 토큰 저장
	if err := uc.tokenRepository.Create(ctx, tokenRecord); err != nil {
		return nil, nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 8. 마지막 로그인 시간 업데이트
	now := time.Now()
	user.LastLoginAt = &now

	if err := uc.userRepository.Update(ctx, user); err != nil {
		uc.logger.Warn("마지막 로그인 시간 업데이트 실패", zap.Error(err))
	}

	// 9. 로그인 성공 감사 로그 기록
	uc.logLoginAttempt(ctx, user.ID, params.Email, "비밀번호 로그인", entity.AuditLogTypeLoginSuccess)

	// 10. 토큰 응답 생성
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return tokens, user, nil
}

// AutoLogin 자동 로그인 (리프레시 토큰 사용)
func (uc *AuthUseCase) AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logLoginAttempt(ctx, "", email, "존재하지 않는 이메일", entity.AuditLogTypeAutoLoginFailed)
			return nil, fmt.Errorf("존재하지 않는 이메일입니다")
		}
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 리프레시 토큰 검증 및 토큰 갱신
	newTokens, err := uc.tokenUseCase.RefreshTokens(ctx, refreshToken, user.ID, deviceInfo.SessionID)
	if err != nil {
		uc.logLoginAttempt(ctx, user.ID, email, fmt.Sprintf("토큰 갱신 실패: %v", err), entity.AuditLogTypeAutoLoginFailed)
		return nil, fmt.Errorf("자동 로그인 실패: %w", err)
	}

	// 3. 자동 로그인 성공 감사 로그 기록
	uc.logLoginAttempt(ctx, user.ID, email, "자동 로그인", entity.AuditLogTypeAutoLoginSuccess)

	return newTokens, nil
}

// Logout 로그아웃
func (uc *AuthUseCase) Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error {
	// 1. 리프레시 토큰 무효화
	if err := uc.tokenUseCase.RevokeRefreshToken(ctx, refreshToken); err != nil {
		uc.logger.Warn("리프레시 토큰 무효화 실패", zap.Error(err))
	}

	// 2. 액세스 토큰 무효화 (선택적)
	if err := uc.tokenUseCase.RevokeAccessToken(ctx, accessToken); err != nil {
		uc.logger.Warn("액세스 토큰 무효화 실패", zap.Error(err))
	}

	uc.logLoginAttempt(ctx, userID, "", "로그아웃", entity.AuditLogTypeLogout)

	return nil
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

	newUser := &entity.User{
		Email:         email,
		Username:      username,
		Name:          username,
		EmailVerified: true, // 매직 링크로 로그인했으므로 이메일은 인증됨
	}

	if err := uc.userRepository.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("사용자 생성 실패: %w", err)
	}

	return newUser, nil
}

// logLoginAttempt 로그인 시도 감사 로그 기록
func (uc *AuthUseCase) logLoginAttempt(ctx context.Context, userID, email, reason string, logType entity.AuditLogType) {
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	ip := ctx.Value("client_ip")
	if ip == nil {
		ip = "unknown"
	}

	userAgent := ctx.Value("user_agent")
	if userAgent == nil {
		userAgent = "unknown"
	}

	content := map[string]interface{}{
		"email":      email,
		"ip":         ip,
		"user_agent": userAgent,
		"reason":     reason,
	}

	// AuditLogUseCase의 AddLog 메소드를 사용하여 감사 로그 기록
	if err := uc.auditLogUseCase.AddLog(ctx, logType, content, userIDPtr); err != nil {
		uc.logger.Error("로그인 시도 감사 로그 저장 실패", zap.Error(err))
	}
}

// 헬퍼 함수

// isValidEmail 이메일 형식 검증
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
