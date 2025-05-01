package auth

import (
	"authn-server/configs"
	"authn-server/internal/logics"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// AuthService는 인증 기능을 제공하는 메인 서비스입니다.
type AuthService struct {
	tokenManager  TokenManagerInterface
	emailVerifier EmailVerifierInterface
	otpService    OtpServiceInterface
}

var (
	// 서비스 전역 인스턴스
	authService AuthServiceInterface
)

// GetAuthService는 전역 AuthService 인스턴스를 반환합니다.
func GetAuthService() AuthServiceInterface {
	if authService == nil {
		authService = NewAuthService()
	}
	return authService
}

// NewAuthService는 인증 서비스의 새 인스턴스를 생성합니다.
func NewAuthService() AuthServiceInterface {
	// 토큰 관리자, 이메일 검증기, OTP 서비스 생성
	tokenManager := GetTokenManager()
	emailVerifier := GetEmailVerifier()
	otpService := GetOtpService()

	return &AuthService{
		tokenManager:  tokenManager,
		emailVerifier: emailVerifier,
		otpService:    otpService,
	}
}

// Register는 새 사용자를 등록합니다.
func (svc *AuthService) Register(ctx context.Context, params RegisterParams) (*models.User, error) {
	// 1. 이미 등록된 사용자인지 확인
	var existingUser models.User
	result := repositories.DBS.Postgres.Where("email = ?", params.Email).First(&existingUser)
	if result.Error == nil {
		return nil, NewAuthError(ErrEmailAlreadyExists, "이미 등록된 이메일입니다")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("사용자 조회 중 오류 발생: %w", result.Error)
	}

	// 2. 비밀번호 해싱
	hashedPassword, salt, err := HashPassword(params.Password)
	if err != nil {
		return nil, fmt.Errorf("비밀번호 해싱 실패: %w", err)
	}

	// 3. 고유 ID 생성
	uid, err := logics.UniqueIDSvc.GenerateID("u")
	if err != nil {
		return nil, fmt.Errorf("사용자 ID 생성 실패: %w", err)
	}

	// 사용자 이름이 없으면 이메일에서 추출
	username := params.Username
	if username == "" {
		username = ExtractUsernameFromEmail(params.Email)
	}

	// 4. 사용자 생성
	user := models.User{
		ID:            uid,
		Email:         params.Email,
		Username:      username,
		Password:      hashedPassword,
		Hash:          salt,
		EmailVerified: false,
	}

	if err := repositories.DBS.Postgres.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("사용자 생성 실패: %w", err)
	}

	// 5. 이메일 인증 토큰 생성 및 전송
	token, err := svc.emailVerifier.GenerateVerificationToken(params.Email)
	if err != nil {
		configs.Logger.Error("인증 토큰 생성 실패", zap.Error(err))
	} else {
		// 비동기 이메일 발송
		go func() {
			if err := svc.emailVerifier.SendVerificationEmail(params.Email, user.Name, token); err != nil {
				configs.Logger.Error("인증 이메일 발송 실패", zap.Error(err))
			}
		}()
	}

	// 6. 감사 로그 기록
	userID := user.ID
	LogUserAction(ctx, AuditLogTypeUserRegistered, params.Email, params.IP, params.UserAgent, &userID)

	return &user, nil
}

// VerifyEmail은 이메일 인증 토큰을 확인합니다.
func (svc *AuthService) VerifyEmail(token string) (*models.User, error) {
	// 1. 토큰으로 이메일 조회
	email, err := svc.emailVerifier.VerifyToken(token)
	if err != nil {
		return nil, err
	}

	// 2. 이메일로 사용자 조회
	var user models.User
	if err := repositories.DBS.Postgres.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("사용자 조회 중 오류 발생: %w", err)
	}

	// 3. 이미 인증된 경우 처리
	if user.EmailVerified {
		// 토큰 삭제 (더 이상 필요 없음)
		svc.emailVerifier.DeleteTokens(token, email)
		return &user, nil
	}

	// 4. 이메일 인증 상태 업데이트
	if err := repositories.DBS.Postgres.Model(&user).Update("email_verified", true).Error; err != nil {
		return nil, fmt.Errorf("사용자 업데이트 중 오류 발생: %w", err)
	}

	// 5. 토큰 삭제
	svc.emailVerifier.DeleteTokens(token, email)

	// 6. 감사 로그 기록
	userID := user.ID
	LogUserAction(context.Background(), AuditLogTypeEmailVerified, email, "", "", &userID)

	return &user, nil
}

// ResendVerificationEmail은 이메일 인증 메일을 재발송합니다.
func (svc *AuthService) ResendVerificationEmail(email string) error {
	// 1. 사용자 조회
	var user models.User
	result := repositories.DBS.Postgres.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return NewAuthError(ErrUserNotFound, "등록되지 않은 이메일입니다")
		}
		return fmt.Errorf("사용자 조회 중 오류 발생: %w", result.Error)
	}

	// 2. 이미 인증된 경우
	if user.EmailVerified {
		return NewAuthError(ErrEmailAlreadyExists, "이미 인증된 이메일입니다")
	}

	// 3. 새 인증 토큰 생성 및 발송
	token, err := svc.emailVerifier.GenerateVerificationToken(email)
	if err != nil {
		return fmt.Errorf("인증 토큰 생성 실패: %w", err)
	}

	return svc.emailVerifier.SendVerificationEmail(email, user.Name, token)
}

// SendMagicCode는 이메일로 매직 코드를 발송합니다.
func (svc *AuthService) SendMagicCode(ctx context.Context, email, ip, userAgent string) error {
	return svc.otpService.SendMagicCode(email, ip, userAgent)
}

// Login은 매직 코드를 사용하여 사용자를 인증합니다.
func (svc *AuthService) Login(ctx context.Context, params LoginParams) (*AuthTokens, *models.User, error) {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return nil, nil, errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return nil, nil, err
	}

	// 1. 매직 코드 확인
	valid, err := svc.otpService.VerifyMagicCode(params.Email, params.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("매직 코드 확인 실패: %w", err)
	}

	if !valid {
		LogUserAction(ctx, AuditLogTypeLoginFailed, params.Email, params.IP, params.UserAgent, nil)
		return nil, nil, NewAuthError(ErrInvalidCredentials, "유효하지 않거나 만료된 코드입니다")
	}

	// 2. 사용자 조회 또는 생성
	user, err := svc.findOrCreateUser(params.Email)
	if err != nil {
		return nil, nil, err
	}

	// 3. 토큰 그룹 조회 또는 생성
	tokenGroup, err := svc.findOrCreateTokenGroup(user.ID)
	if err != nil {
		return nil, nil, err
	}

	// 4. 액세스 및 리프레시 토큰 생성
	accessToken, err := svc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	refreshToken, tokenRecord := svc.tokenManager.GenerateRefreshToken(tokenGroup.ID)

	// 5. 리프레시 토큰 저장
	if err := repositories.DBS.Postgres.Create(tokenRecord).Error; err != nil {
		return nil, nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 6. 세션에 사용자 ID 저장
	sess.Values["auth_user"] = user.ID
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, nil, err
	}

	// 7. 로그인 활동 기록
	if err := svc.recordLoginActivity(*c, sess.ID, user.ID, tokenGroup.ID, params.DeviceUID); err != nil {
		return nil, nil, err
	}

	// 8. 세션에 토큰 저장
	sess.Values["access_token"] = accessToken
	sess.Values["refresh_token"] = refreshToken
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, nil, err
	}

	// 9. 감사 로그 기록
	userID := user.ID
	LogUserAction(ctx, AuditLogTypeLoginSuccess, params.Email, params.IP, params.UserAgent, &userID)

	// 10. 토큰 응답 구성
	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute),
	}

	return tokens, user, nil
}

// LoginWithPassword는 이메일과 비밀번호로 사용자를 인증합니다.
func (svc *AuthService) LoginWithPassword(ctx context.Context, params LoginParams) (*AuthTokens, *models.User, error) {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return nil, nil, errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return nil, nil, err
	}

	// 1. 이메일로 사용자 조회
	var user models.User
	result := repositories.DBS.Postgres.Where("email = ?", params.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			LogUserAction(ctx, AuditLogTypeLoginFailed, params.Email, params.IP, params.UserAgent, nil)
			return nil, nil, NewAuthError(ErrInvalidCredentials, "이메일 또는 비밀번호가 올바르지 않습니다")
		}
		return nil, nil, fmt.Errorf("사용자 조회 중 오류 발생: %w", result.Error)
	}

	// 2. 이메일 인증 확인
	if !user.EmailVerified {
		// 인증 상태 확인 및 처리
		return nil, nil, NewAuthError(ErrEmailNotVerified, "이메일 인증이 필요합니다")
	}

	// 3. 비밀번호 검증
	if err := VerifyPassword(user.Password, params.Password, user.Hash); err != nil {
		LogUserAction(ctx, AuditLogTypeLoginFailed, params.Email, params.IP, params.UserAgent, nil)
		return nil, nil, NewAuthError(ErrInvalidCredentials, "이메일 또는 비밀번호가 올바르지 않습니다")
	}

	// 4. 토큰 그룹 조회 또는 생성
	tokenGroup, err := svc.findOrCreateTokenGroup(user.ID)
	if err != nil {
		return nil, nil, err
	}

	// 5. 액세스 및 리프레시 토큰 생성
	accessToken, err := svc.tokenManager.GenerateAccessToken(&user)
	if err != nil {
		return nil, nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	refreshToken, tokenRecord := svc.tokenManager.GenerateRefreshToken(tokenGroup.ID)

	// 6. 리프레시 토큰 저장
	if err := repositories.DBS.Postgres.Create(tokenRecord).Error; err != nil {
		return nil, nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 7. 세션에 사용자 ID 저장
	sess.Values["auth_user"] = user.ID
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, nil, err
	}

	// 8. 로그인 활동 기록
	if err := svc.recordLoginActivity(*c, sess.ID, user.ID, tokenGroup.ID, params.DeviceUID); err != nil {
		return nil, nil, err
	}

	// 9. 세션에 토큰 저장
	sess.Values["access_token"] = accessToken
	sess.Values["refresh_token"] = refreshToken
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, nil, err
	}

	// 10. 감사 로그 기록
	userID := user.ID
	LogUserAction(ctx, AuditLogTypeLoginSuccess, params.Email, params.IP, params.UserAgent, &userID)

	// 11. 토큰 응답 구성
	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute),
	}

	return tokens, &user, nil
}

// AutoLogin은 리프레시 토큰을 사용하여 자동 로그인합니다.
func (svc *AuthService) AutoLogin(ctx context.Context, email, refreshToken string, deviceInfo DeviceInfo) (*AuthTokens, error) {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return nil, errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return nil, err
	}

	// 1. 리프레시 토큰 검증 및 갱신
	tokenGroupID, user, newRefreshToken, err := svc.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		LogUserAction(ctx, AuditLogTypeRefreshTokenInvalid, email, deviceInfo.IP, deviceInfo.UserAgent, nil)
		return nil, err
	}

	// 2. 요청 이메일과 토큰 사용자 이메일 일치 확인
	if email != user.Email {
		_ = svc.tokenManager.RevokeTokenGroup(tokenGroupID)
		return nil, NewAuthError(ErrInvalidCredentials, "요청 사용자와 토큰 사용자가 일치하지 않습니다")
	}

	// 3. 새 액세스 토큰 생성
	accessToken, err := svc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 4. 세션에 사용자 ID 저장
	sess.Values["auth_user"] = user.ID
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, err
	}

	// 5. 로그인 활동 기록
	if err := svc.recordLoginActivity(*c, sess.ID, user.ID, tokenGroupID, deviceInfo.DeviceUID); err != nil {
		return nil, err
	}

	// 6. 세션에 토큰 저장
	sess.Values["access_token"] = accessToken
	sess.Values["refresh_token"] = newRefreshToken
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, err
	}

	// 7. 감사 로그 기록
	userID := user.ID
	LogUserAction(ctx, AuditLogTypeAutoLoginSuccess, email, deviceInfo.IP, deviceInfo.UserAgent, &userID)

	// 8. 토큰 응답 구성
	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute),
	}

	return tokens, nil
}

// Logout은 사용자를 로그아웃시키고 모든 관련 토큰을 취소합니다.
func (svc *AuthService) Logout(ctx context.Context, sessionID, accessToken, refreshToken, userID string) error {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return err
	}

	// 1. 세션 삭제
	sess.Options.MaxAge = -1
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return err
	}

	// 2. 액세스 토큰 취소
	if accessToken != "" {
		_ = svc.tokenManager.RevokeAccessToken(accessToken)
	}

	// 3. 리프레시 토큰 취소
	if refreshToken != "" {
		var token models.Token
		err := repositories.DBS.Postgres.Preload("TokenGroup").Where("token = ?", refreshToken).First(&token).Error
		if err == nil && token.TokenGroup != nil {
			_ = svc.tokenManager.RevokeTokenGroup(token.TokenGroup.ID)
		}
	}

	// 4. 로그인 활동 기록 비활성화
	if sessionID != "" && userID != "" {
		if err := repositories.DBS.Postgres.Model(&models.Activity{}).
			Where("session_id = ? AND user_id = ?", sessionID, userID).
			Update("logout_at", time.Now()).
			Error; err != nil {
			return fmt.Errorf("로그인 활동 비활성화 실패: %w", err)
		}
	}

	// 5. 감사 로그 기록
	LogUserAction(ctx, AuditLogTypeLogoutSuccess, "", (*c).RealIP(), (*c).Request().UserAgent(), &userID)

	return nil
}

// RefreshTokens는 리프레시 토큰을 사용하여 액세스 토큰을 갱신합니다.
func (svc *AuthService) RefreshTokens(ctx context.Context, refreshToken, userID, sessionID string) (*AuthTokens, error) {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return nil, errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return nil, err
	}

	// 1. 세션 유효성 확인
	sessionUserID, ok := sess.Values["auth_user"].(string)
	if !ok || sessionUserID == "" {
		return nil, NewAuthError(ErrSessionInvalid, "세션이 유효하지 않거나 사용자를 찾을 수 없습니다")
	}

	// 2. 리프레시 토큰 검증
	tokenGroupID, user, newRefreshToken, err := svc.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		LogUserAction(ctx, AuditLogTypeRefreshTokenInvalid, userID, (*c).RealIP(), (*c).Request().UserAgent(), nil)
		return nil, err
	}

	// 3. 세션 사용자와 토큰 사용자 일치 확인
	if sessionUserID != user.ID {
		_ = svc.tokenManager.RevokeTokenGroup(tokenGroupID)
		return nil, NewAuthError(ErrInvalidCredentials, "세션 사용자와 토큰 사용자가 일치하지 않습니다")
	}

	// 4. 새 액세스 토큰 생성
	newAccessToken, err := svc.tokenManager.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	// 5. 세션 업데이트
	sess.Values["access_token"] = newAccessToken
	sess.Values["refresh_token"] = newRefreshToken
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, err
	}

	// 6. 감사 로그 기록
	LogUserAction(ctx, AuditLogTypeRefreshTokenSuccess, user.Email, (*c).RealIP(), (*c).Request().UserAgent(), &user.ID)

	// 7. 토큰 응답 구성
	tokens := &AuthTokens{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute),
	}

	return tokens, nil
}

// GenerateTokensAfter2FA는 2FA 인증 후 토큰을 생성합니다.
func (svc *AuthService) GenerateTokensAfter2FA(ctx context.Context, userID string, deviceInfo DeviceInfo) (*AuthTokens, error) {
	// Echo 컨텍스트 및 세션 객체 추출
	c, ok := ctx.Value("echo").(*echo.Context)
	if !ok || c == nil {
		return nil, errors.New("Echo 컨텍스트를 찾을 수 없습니다")
	}

	// 세션 객체 얻기
	sess, err := getSessionFromContext(*c)
	if err != nil {
		return nil, err
	}

	// 1. 사용자 조회
	var user models.User
	if err := repositories.DBS.Postgres.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("사용자 조회 실패: %w", err)
	}

	// 2. 토큰 그룹 조회 또는 생성
	tokenGroup, err := svc.findOrCreateTokenGroup(userID)
	if err != nil {
		return nil, err
	}

	// 3. 토큰 생성
	accessToken, err := svc.tokenManager.GenerateAccessToken(&user)
	if err != nil {
		return nil, fmt.Errorf("액세스 토큰 생성 실패: %w", err)
	}

	refreshToken, tokenRecord := svc.tokenManager.GenerateRefreshToken(tokenGroup.ID)

	// 4. 리프레시 토큰 저장
	if err := repositories.DBS.Postgres.Create(tokenRecord).Error; err != nil {
		return nil, fmt.Errorf("토큰 저장 실패: %w", err)
	}

	// 5. 세션 인증
	sess.Values["auth_user"] = userID
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, err
	}

	// 6. 로그인 활동 기록
	if err := svc.recordLoginActivity(*c, deviceInfo.SessionID, userID, tokenGroup.ID, deviceInfo.DeviceUID); err != nil {
		return nil, err
	}

	// 7. 세션에 토큰 저장
	sess.Values["access_token"] = accessToken
	sess.Values["refresh_token"] = refreshToken
	if err := sess.Save((*c).Request(), (*c).Response()); err != nil {
		return nil, err
	}

	// 8. 감사 로그 기록
	LogUserAction(ctx, AuditLogTypeLoginSuccess, user.Email, deviceInfo.IP, deviceInfo.UserAgent, &userID)

	// 9. 토큰 응답 구성
	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute),
	}

	return tokens, nil
}

// 헬퍼 함수

// findOrCreateUser는 이메일로 사용자를 조회하거나 생성합니다.
func (svc *AuthService) findOrCreateUser(email string) (*models.User, error) {
	var user models.User
	err := repositories.DBS.Postgres.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 새 사용자 생성
			uid, _ := logics.GenerateUniqueID("u")
			user = models.User{
				ID:       uid,
				Email:    email,
				Name:     ExtractUsernameFromEmail(email),
				Username: ExtractUsernameFromEmail(email),
				Password: "", // 매직 로그인에서는 비밀번호 없음
				Hash:     "", // 매직 로그인에서는 해시 없음
			}
			if createErr := repositories.DBS.Postgres.Create(&user).Error; createErr != nil {
				return nil, fmt.Errorf("사용자 생성 실패: %w", createErr)
			}
			return &user, nil
		}
		return nil, err
	}
	return &user, nil
}

// findOrCreateTokenGroup은 사용자 ID로 토큰 그룹을 조회하거나 생성합니다.
func (svc *AuthService) findOrCreateTokenGroup(userID string) (*models.TokenGroup, error) {
	var tokenGroup models.TokenGroup
	err := repositories.DBS.Postgres.Where("user_id = ?", userID).First(&tokenGroup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 새 토큰 그룹 생성
			tokenGroup = models.TokenGroup{
				UserID: userID,
			}
			if createErr := repositories.DBS.Postgres.Create(&tokenGroup).Error; createErr != nil {
				return nil, fmt.Errorf("토큰 그룹 생성 실패: %w", createErr)
			}
		} else {
			return nil, err
		}
	}
	return &tokenGroup, nil
}

// recordLoginActivity는 로그인 활동을 기록합니다.
func (svc *AuthService) recordLoginActivity(c echo.Context, sessionID, userID string, tokenGroupID uint, deviceUID *uuid.UUID) error {
	activityRecord := &models.Activity{
		SessionID:    sessionID,
		UserID:       userID,
		TokenGroupID: tokenGroupID,
		IP:           c.RealIP(),
		UserAgent:    c.Request().UserAgent(),
		DeviceUID:    deviceUID,
		LoginAt:      time.Now(),
		LogoutAt:     nil,
	}

	return repositories.DBS.Postgres.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ip", "user_agent", "device_uid", "token_group_id", "login_at", "logout_at",
			"updated_at",
		}),
	}).Create(activityRecord).Error
}

// getSessionFromContext는 Echo 컨텍스트에서 세션을 얻습니다.
func getSessionFromContext(c echo.Context) (*sessions.Session, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, fmt.Errorf("세션 조회 실패: %w", err)
	}
	return sess, nil
}
