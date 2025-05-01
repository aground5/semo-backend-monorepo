package controllers

import (
	"authn-server/internal/auth"
	"authn-server/internal/middlewares"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// Request structs
type MagicRequest struct {
	Email string `json:"email" form:"email"`
}

type LoginRequest struct {
	Email string `json:"email" form:"email"`
	Code  string `json:"code" form:"code"`
}

type LogoutRequest struct {
	AccessToken  string `json:"access_token" form:"access_token"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

type AutoLoginRequest struct {
	Email        string `json:"email" form:"email"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

type RefreshAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

type TwoFactorLoginRequest struct {
	ChallengeID string `json:"challenge_id" form:"challenge_id"`
	Code        string `json:"code" form:"code"`
}

type RegisterRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
	Username string `json:"username" form:"username"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" form:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" form:"email"`
}

type PasswordLoginRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

// Response structs
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type LoginResponse struct {
	Token TokenResponse `json:"tokens"`
	User  UserResponse  `json:"user"`
}

// Service interfaces
var (
	authService auth.AuthServiceInterface
)

// Init initializes the auth service
func init() {
	// Get the singleton auth service instance
	authService = auth.GetAuthService()
}

// MeHandler returns the currently authenticated user's information
// GET /authn/me
func MeHandler(c echo.Context) error {
	// SessionMiddleware를 통해 세션 가져오기
	sess, err := middlewares.GetSessionFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "세션 오류"})
	}

	// 세션에서 사용자 ID 확인
	userID, ok := sess.Values["auth_user"].(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "인증되지 않음"})
	}

	// 사용자 정보 조회
	var user models.User
	if err := repositories.DBS.Postgres.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 정보 조회 실패"})
	}

	// 사용자 정보 반환
	return c.JSON(http.StatusOK, UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	})
}

// MagicHandler handles magic code generation
// POST /authn/magic
func MagicHandler(c echo.Context) error {
	req := new(MagicRequest)
	if err := c.Bind(req); err != nil || req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
	}

	// Create context with Echo context for auth service
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Send magic code using auth service
	err := authService.SendMagicCode(ctx, req.Email, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Magic code generated"})
}

// LoginHandler handles login with magic code
// POST /authn/login
func LoginHandler(c echo.Context) error {
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil || req.Email == "" || req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and code required"})
	}

	// Get device ID
	duidStr := c.Request().Header.Get("x-duid")
	if duidStr == "" {
		duidCookie, err := c.Cookie("duid")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "device identifier required"})
		}
		duidStr = duidCookie.Value
	}

	duid, err := uuid.Parse(duidStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device identifier"})
	}

	// Get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "session error"})
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Login with auth service
	loginParams := auth.LoginParams{
		Email:     req.Email,
		Code:      req.Code,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		DeviceUID: &duid,
		SessionID: sess.ID,
	}

	tokens, user, err := authService.Login(ctx, loginParams)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Prepare response
	response := LoginResponse{
		Token: TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// LogoutHandler handles user logout
// POST /authn/logout
func LogoutHandler(c echo.Context) error {
	// SessionMiddleware를 통해 세션 가져오기
	sess, err := middlewares.GetSessionFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 세션 값 가져오기
	sessionID := sess.ID
	accessToken := ""
	refreshToken := ""
	userID := ""

	if val, ok := sess.Values["access_token"]; ok {
		accessToken, _ = val.(string)
	}
	if val, ok := sess.Values["refresh_token"]; ok {
		refreshToken, _ = val.(string)
	}
	if val, ok := sess.Values["auth_user"]; ok {
		userID, _ = val.(string)
	}

	// Echo 컨텍스트 포함한 컨텍스트 생성
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// 인증 서비스로 로그아웃 처리
	err = authService.Logout(ctx, sessionID, accessToken, refreshToken, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "로그아웃 완료"})
}

// AutoLoginHandler handles automatic login with refresh token
// POST /authn/auto_login
func AutoLoginHandler(c echo.Context) error {
	req := new(AutoLoginRequest)
	if err := c.Bind(req); err != nil || req.Email == "" || req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and refresh_token required"})
	}

	// Get device ID
	duidStr := c.Request().Header.Get("x-duid")
	if duidStr == "" {
		duidCookie, err := c.Cookie("duid")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "device identifier required"})
		}
		duidStr = duidCookie.Value
	}

	duid, err := uuid.Parse(duidStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device identifier"})
	}

	// Get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "session error"})
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Auto login with auth service
	deviceInfo := auth.DeviceInfo{
		DeviceUID: &duid,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		SessionID: sess.ID,
	}

	tokens, err := authService.AutoLogin(ctx, req.Email, req.RefreshToken, deviceInfo)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

// RefreshAccessTokenHandler handles refresh token requests
// POST /authn/refresh
func RefreshAccessTokenHandler(c echo.Context) error {
	req := new(RefreshAccessTokenRequest)
	if err := c.Bind(req); err != nil || req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
	}

	// Get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Get user ID from session if available
	userID := ""
	if val, ok := sess.Values["auth_user"]; ok {
		userID, _ = val.(string)
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Refresh tokens with auth service
	tokens, err := authService.RefreshTokens(ctx, req.RefreshToken, userID, sess.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

// RegisterHandler handles user registration
// POST /authn/register
func RegisterHandler(c echo.Context) error {
	req := new(RegisterRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Basic validation
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	// Email format validation
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "please enter a valid email address"})
	}

	// Password strength validation
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Register with auth service
	params := auth.RegisterParams{
		Email:     req.Email,
		Password:  req.Password,
		Username:  req.Username,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}

	user, err := authService.Register(ctx, params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Registration complete. Please check your email for verification.",
		"user_id": user.ID,
		"email":   user.Email,
	})
}

// VerifyEmailHandler handles email verification
// POST /authn/verify-email
func VerifyEmailHandler(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		req := new(VerifyEmailRequest)
		if err := c.Bind(req); err != nil || req.Token == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "verification token required"})
		}
		token = req.Token
	}

	// Verify email with auth service
	user, err := authService.VerifyEmail(token)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "Email verification complete. You can now log in.",
		"verified": true,
		"email":    user.Email,
	})
}

// ResendVerificationHandler resends verification email
// POST /authn/resend-verification
func ResendVerificationHandler(c echo.Context) error {
	req := new(ResendVerificationRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email required"})
	}

	// Resend verification email with auth service
	err := authService.ResendVerificationEmail(req.Email)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Verification email resent. Please check your inbox.",
	})
}

// PasswordLoginHandler handles login with email and password
// POST /authn/login-password
func PasswordLoginHandler(c echo.Context) error {
	req := new(PasswordLoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	// Get device ID
	duidStr := c.Request().Header.Get("x-duid")
	if duidStr == "" {
		duidCookie, err := c.Cookie("duid")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "device identifier required"})
		}
		duidStr = duidCookie.Value
	}

	duid, err := uuid.Parse(duidStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device identifier"})
	}

	// Get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "session error"})
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Login with password using auth service
	loginParams := auth.LoginParams{
		Email:     req.Email,
		Password:  req.Password,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		DeviceUID: &duid,
		SessionID: sess.ID,
	}

	tokens, user, err := authService.LoginWithPassword(ctx, loginParams)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Prepare response
	response := LoginResponse{
		Token: TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
		User: UserResponse{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GenerateTokensAfter2FAHandler generates tokens after 2FA verification
// POST /authn/generate-tokens-after-2fa
func GenerateTokensAfter2FAHandler(c echo.Context) error {
	// Parse user ID from query or JSON
	userID := c.QueryParam("user_id")
	if userID == "" {
		var req struct {
			UserID string `json:"user_id"`
		}
		if err := c.Bind(&req); err != nil || req.UserID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		}
		userID = req.UserID
	}

	// Get device ID
	duidStr := c.Request().Header.Get("x-duid")
	if duidStr == "" {
		duidCookie, err := c.Cookie("duid")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "device identifier required"})
		}
		duidStr = duidCookie.Value
	}

	duid, err := uuid.Parse(duidStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid device identifier"})
	}

	// Get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "session error"})
	}

	// Create device info
	deviceInfo := auth.DeviceInfo{
		DeviceUID: &duid,
		IP:        c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		SessionID: sess.ID,
	}

	// Create context with Echo context
	ctx := context.WithValue(c.Request().Context(), "echo", &c)

	// Generate tokens after 2FA verification
	tokens, err := authService.GenerateTokensAfter2FA(ctx, userID, deviceInfo)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}
