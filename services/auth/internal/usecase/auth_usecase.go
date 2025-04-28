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
	emailUseCase    interfaces.EmailUseCase
}

// NewAuthUseCase 새 인증 유스케이스 생성
func NewAuthUseCase(
	logger *zap.Logger,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	emailUC interfaces.EmailUseCase,
) *AuthUseCase {
	return &AuthUseCase{
		logger:          logger,
		userRepository:  userRepo,
		auditRepository: auditRepo,
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
	hashedPassword, err := HashPassword(params.Password)
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
		content := map[string]interface{}{
			"email":      params.Email,
			"ip":         params.IP,
			"user_agent": params.UserAgent,
		}

		auditLog := &entity.AuditLog{
			UserID:  user.ID,
			Type:    "USER_REGISTERED",
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

func (uc *SessionUseCase) Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	// 1. 매직 코드 확인
	valid, err := uc.otpUseCase.VerifyMagicCode(ctx, params.Email, params.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("매직 코드 확인 실패: %w", err)
	}

	if !valid {
		// 로그인 실패 감사 로그 기록
		uc.logLoginAttempt(ctx, "", params.Email, "LOGIN_FAILED", "유효하지 않은 매직 코드", params.IP, params.UserAgent)
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

	if err := uc.userRepository.Update(ctx, user); err != nil {
		uc.logger.Warn("마지막 로그인 시간 업데이트 실패", zap.Error(err))
	}

	// 8. 로그인 성공 감사 로그 기록
	uc.logLoginAttempt(ctx, user.ID, params.Email, "LOGIN_SUCCESS", "매직 코드 로그인", params.IP, params.UserAgent)

	// 9. 토큰 응답 생성 (만료 시간은 30분으로 가정)
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	return tokens, user, nil
}

// LoginWithPassword 비밀번호로 로그인
func (uc *SessionUseCase) LoginWithPassword(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, params.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logLoginAttempt(ctx, "", params.Email, "LOGIN_FAILED", "존재하지 않는 이메일", params.IP, params.UserAgent)
			return nil, nil, fmt.Errorf("존재하지 않는 이메일입니다")
		}
		return nil, nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 비밀번호 검증
	if !uc.verifyPassword(user.Password, params.Password) {
		uc.logLoginAttempt(ctx, user.ID, params.Email, "LOGIN_FAILED", "잘못된 비밀번호", params.IP, params.UserAgent)
		return nil, nil, fmt.Errorf("잘못된 비밀번호입니다")
	}

	// 3. 이메일 인증 확인
	if !user.EmailVerified {
		uc.logLoginAttempt(ctx, user.ID, params.Email, "LOGIN_FAILED", "이메일 미인증", params.IP, params.UserAgent)
		return nil, nil, fmt.Errorf("이메일 인증이 필요합니다")
	}

	// 4. 토큰 그룹 조회 또는 생성
	tokenGroup, err := uc.findOrCreateTokenGroup(ctx, user.ID)
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
	uc.logLoginAttempt(ctx, user.ID, params.Email, "LOGIN_SUCCESS", "비밀번호 로그인", params.IP, params.UserAgent)

	// 10. 토큰 응답 생성
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	return tokens, user, nil
}

// AutoLogin 자동 로그인 (리프레시 토큰 사용)
func (uc *SessionUseCase) AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logLoginAttempt(ctx, "", email, "AUTO_LOGIN_FAILED", "존재하지 않는 이메일", deviceInfo.IP, deviceInfo.UserAgent)
			return nil, fmt.Errorf("존재하지 않는 이메일입니다")
		}
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 리프레시 토큰 검증 및 토큰 갱신
	newTokens, err := uc.RefreshTokens(ctx, refreshToken, user.ID, deviceInfo.SessionID)
	if err != nil {
		uc.logLoginAttempt(ctx, user.ID, email, "AUTO_LOGIN_FAILED", fmt.Sprintf("토큰 갱신 실패: %v", err), deviceInfo.IP, deviceInfo.UserAgent)
		return nil, fmt.Errorf("자동 로그인 실패: %w", err)
	}

	// 3. 자동 로그인 성공 감사 로그 기록
	uc.logLoginAttempt(ctx, user.ID, email, "AUTO_LOGIN_SUCCESS", "자동 로그인", deviceInfo.IP, deviceInfo.UserAgent)

	return newTokens, nil
}

// Logout 로그아웃
func (uc *SessionUseCase) Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error {
	// 1. 리프레시 토큰 무효화
	if err := uc.tokenRepository.InvalidateRefreshToken(ctx, refreshToken); err != nil {
		uc.logger.Warn("리프레시 토큰 무효화 실패", zap.Error(err))
	}

	// 2. 액세스 토큰 블랙리스트 추가 (선택적)
	if err := uc.tokenUseCase.BlacklistAccessToken(ctx, accessToken); err != nil {
		uc.logger.Warn("액세스 토큰 블랙리스트 추가 실패", zap.Error(err))
	}

	// 3. 로그아웃 감사 로그 기록
	ua := ctx.Value("user_agent")
	if ua == nil {
		ua = "unknown"
	}

	ip := ctx.Value("client_ip")
	if ip == nil {
		ip = "unknown"
	}

	auditLog := &entity.AuditLog{
		UserID: userID,
		Type:   "LOGOUT",
		Content: map[string]interface{}{
			"session_id": sessionID,
			"ip":         ip,
			"user_agent": ua,
		},
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Error("로그아웃 감사 로그 저장 실패", zap.Error(err))
	}

	return nil
}

// RefreshTokens 토큰 갱신
func (uc *SessionUseCase) RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*dto.AuthTokens, error) {
	// 1. 리프레시 토큰 검증
	tokenData, err := uc.tokenRepository.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("리프레시 토큰 검증 실패: %w", err)
	}

	// 2. 토큰 소유자 검증
	tokenGroup, err := uc.tokenRepository.FindGroupByID(ctx, tokenData.TokenGroupID)
	if err != nil {
		return nil, fmt.Errorf("토큰 그룹 조회 실패: %w", err)
	}

	if tokenGroup.UserID != userID {
		// 다른 사용자의 토큰인 경우 - 보안 위험으로 간주
		auditLog := &entity.AuditLog{
			UserID: userID,
			Type:   "TOKEN_MISUSE",
			Content: map[string]interface{}{
				"token_owner": tokenGroup.UserID,
				"requester":   userID,
				"token_id":    tokenData.ID,
			},
		}

		if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
			uc.logger.Error("토큰 오용 감사 로그 저장 실패", zap.Error(err))
		}

		return nil, fmt.Errorf("유효하지 않은 토큰입니다")
	}

	// 3. 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 4. 새 액세스 토큰 생성
	newAccessToken, err := uc.tokenUseCase.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 5. 새 리프레시 토큰 생성
	newRefreshToken, tokenRecord, err := uc.tokenUseCase.GenerateRefreshToken(ctx, tokenGroup.ID)
	if err != nil {
		return nil, fmt.Errorf("리프레시 토큰 생성 실패: %w", err)
	}

	// 6. 이전 리프레시 토큰 무효화
	if err := uc.tokenRepository.InvalidateRefreshToken(ctx, refreshToken); err != nil {
		uc.logger.Warn("이전 리프레시 토큰 무효화 실패", zap.Error(err))
	}

	// 7. 새 리프레시 토큰 저장
	if err := uc.tokenRepository.Create(ctx, tokenRecord); err != nil {
		return nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 8. 토큰 갱신 감사 로그 기록
	auditLog := &entity.AuditLog{
		UserID: userID,
		Type:   "TOKEN_REFRESHED",
		Content: map[string]interface{}{
			"session_id": sessionID,
		},
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Error("토큰 갱신 감사 로그 저장 실패", zap.Error(err))
	}

	// 9. 토큰 응답 생성
	tokens := &dto.AuthTokens{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	return tokens, nil
}

// GenerateTokensAfter2FA 2FA 인증 후 토큰 생성
func (uc *SessionUseCase) GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	// 1. 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 토큰 그룹 조회 또는 생성
	tokenGroup, err := uc.findOrCreateTokenGroup(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// 3. 액세스 토큰 생성
	accessToken, err := uc.tokenUseCase.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 4. 리프레시 토큰 생성
	refreshToken, tokenRecord, err := uc.tokenUseCase.GenerateRefreshToken(ctx, tokenGroup.ID)
	if err != nil {
		return nil, fmt.Errorf("리프레시 토큰 생성 실패: %w", err)
	}

	// 5. 리프레시 토큰 저장
	if err := uc.tokenRepository.Create(ctx, tokenRecord); err != nil {
		return nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 6. 2FA 인증 성공 감사 로그 기록
	auditLog := &entity.AuditLog{
		UserID: userID,
		Type:   "2FA_SUCCESS",
		Content: map[string]interface{}{
			"ip":         deviceInfo.IP,
			"user_agent": deviceInfo.UserAgent,
			"session_id": deviceInfo.SessionID,
		},
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Error("2FA 인증 감사 로그 저장 실패", zap.Error(err))
	}

	// 7. 토큰 응답 생성
	tokens := &dto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	return tokens, nil
}

// 헬퍼 함수들

// findOrCreateUser 사용자를 이메일로 찾거나 새로 생성
func (uc *SessionUseCase) findOrCreateUser(ctx context.Context, email string) (*entity.User, error) {
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
	username := extractUsernameFromEmail(email)

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

// findOrCreateTokenGroup 토큰 그룹을 찾거나 새로 생성
func (uc *SessionUseCase) findOrCreateTokenGroup(ctx context.Context, userID string) (*entity.TokenGroup, error) {
	tokenGroup, err := uc.tokenRepository.FindGroupByUserID(ctx, userID)
	if err == nil && tokenGroup != nil {
		// 토큰 그룹 찾음
		return tokenGroup, nil
	}

	// 새 토큰 그룹 생성
	newTokenGroup := &entity.TokenGroup{
		UserID: userID,
	}

	if err := uc.tokenRepository.CreateGroup(ctx, newTokenGroup); err != nil {
		return nil, fmt.Errorf("토큰 그룹 생성 실패: %w", err)
	}

	return newTokenGroup, nil
}

// verifyPassword 비밀번호 검증
func (uc *SessionUseCase) verifyPassword(hashedPassword, plainPassword string) bool {
	// 실제 검증 로직 구현 (예: bcrypt.CompareHashAndPassword)
	// 이 예제에서는 단순화를 위해 간단한 비교 사용
	return hashedPassword == plainPassword+"_hashed"
}

// logLoginAttempt 로그인 시도 감사 로그 기록
func (uc *SessionUseCase) logLoginAttempt(ctx context.Context, userID, email, logType, reason, ip, userAgent string) {
	auditLog := &entity.AuditLog{
		UserID: userID,
		Type:   logType,
		Content: map[string]interface{}{
			"email":      email,
			"ip":         ip,
			"user_agent": userAgent,
			"reason":     reason,
		},
	}

	if err := uc.auditRepository.Create(context.Background(), auditLog); err != nil {
		uc.logger.Error("로그인 시도 감사 로그 저장 실패", zap.Error(err))
	}
}

// extractUsernameFromEmail 이메일에서 사용자 이름 추출
func extractUsernameFromEmail(email string) string {
	parts := regexp.MustCompile("@").Split(email, 2)
	return parts[0]
}

// LoginWithPassword 비밀번호로 로그인
func (uc *AuthUseCase) LoginWithPassword(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	return nil, nil, fmt.Errorf("인증 유스케이스에서 미구현 기능: LoginWithPassword")
}

// AutoLogin 자동 로그인 (리프레시 토큰 사용)
func (uc *AuthUseCase) AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	return nil, fmt.Errorf("인증 유스케이스에서 미구현 기능: AutoLogin")
}

// Logout 로그아웃
func (uc *AuthUseCase) Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error {
	return fmt.Errorf("인증 유스케이스에서 미구현 기능: Logout")
}

// RefreshTokens 토큰 갱신
func (uc *AuthUseCase) RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*dto.AuthTokens, error) {
	return nil, fmt.Errorf("인증 유스케이스에서 미구현 기능: RefreshTokens")
}

// GenerateTokensAfter2FA 2FA 인증 후 토큰 생성
func (uc *AuthUseCase) GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (*dto.AuthTokens, error) {
	return nil, fmt.Errorf("인증 유스케이스에서 미구현 기능: GenerateTokensAfter2FA")
}

// Login 로그인 (SessionUseCase로 위임)
func (uc *AuthUseCase) Login(ctx context.Context, params dto.LoginParams) (*dto.AuthTokens, *entity.User, error) {
	return nil, nil, fmt.Errorf("인증 유스케이스에서 미구현 기능: Login")
}

// 헬퍼 함수

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

// ExtractUsernameFromEmail 이메일에서 사용자 이름 추출
func ExtractUsernameFromEmail(email string) string {
	parts := regexp.MustCompile("@").Split(email, 2)
	return parts[0]
}

// HashPassword 비밀번호 해싱
func HashPassword(password string) (string, error) {
	// 실제 해싱 로직 구현
	// 예: bcrypt 등 사용
	hashedPassword := password + "_hashed" // 예시용 간단 구현
	return hashedPassword, nil
}
