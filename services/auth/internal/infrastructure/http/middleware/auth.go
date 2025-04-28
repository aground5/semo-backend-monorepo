package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// 컨텍스트 키 상수
const (
	UserIDKey = "user_id"
	UserKey   = "user"
)

// JWTAuthMiddleware는 JWT 인증을 처리하는 미들웨어입니다.
// 토큰 유효성 검증과 같은 비즈니스 로직은 TokenUseCase에 위임합니다.
type JWTAuthMiddleware struct {
	tokenUseCase interfaces.TokenUseCase
	logger       *zap.Logger
}

// NewJWTAuthMiddleware는 새로운 JWT 인증 미들웨어를 생성합니다.
func NewJWTAuthMiddleware(tokenUseCase interfaces.TokenUseCase, logger *zap.Logger) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		tokenUseCase: tokenUseCase,
		logger:       logger,
	}
}

// Handle는 HTTP 요청에서 JWT 토큰을 추출하고 검증하는 핸들러 함수를 반환합니다.
func (m *JWTAuthMiddleware) Handle() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. 요청 헤더에서 토큰 추출
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "인증 토큰이 없습니다",
				})
			}

			// Bearer 토큰 형식 확인 및 추출
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "인증 헤더 형식이 올바르지 않습니다",
				})
			}
			accessToken := parts[1]

			// 2. 토큰 유스케이스를 통해 토큰 검증
			user, err := m.tokenUseCase.ValidateAccessToken(c.Request().Context(), accessToken)
			if err != nil {
				m.logger.Info("인증 실패",
					zap.String("error", err.Error()),
					zap.String("ip", c.RealIP()),
					zap.String("path", c.Request().URL.Path),
				)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "유효하지 않은 인증 토큰입니다: " + err.Error(),
				})
			}

			// 3. 검증된 사용자 정보를 컨텍스트에 저장
			c.Set(UserIDKey, user.ID)
			c.Set(UserKey, user)

			// 다음 핸들러 호출
			return next(c)
		}
	}
}
