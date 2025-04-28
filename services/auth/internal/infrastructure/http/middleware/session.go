package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// SessionKey는 컨텍스트에 세션을 저장할 때 사용하는 키입니다.
const (
	SessionKey = "session"
)

// SessionMiddleware는 세션 관리를 담당하는 미들웨어입니다.
type SessionMiddleware struct {
	sessionUseCase interfaces.SessionUseCase
	logger         *zap.Logger
}

// NewSessionMiddleware는 새로운 세션 미들웨어를 생성합니다.
func NewSessionMiddleware(sessionUC interfaces.SessionUseCase, logger *zap.Logger) *SessionMiddleware {
	return &SessionMiddleware{
		sessionUseCase: sessionUC,
		logger:         logger,
	}
}

// Handle는 HTTP 요청에서 세션을 관리하는 핸들러 함수를 반환합니다.
func (m *SessionMiddleware) Handle() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 세션 가져오기
			sess, err := session.Get(SessionKey, c)
			if err != nil {
				// 세션 오류 발생 시 세션 초기화
				m.resetSession(c)
				m.logger.Warn("세션 검증 실패",
					zap.Error(err),
					zap.String("ip", c.RealIP()),
				)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":            "세션이 만료되었습니다. 다시 로그인해주세요.",
					"dont_raise_error": "true",
				})
			}

			// 세션에서 사용자 ID 가져오기
			userID, ok := sess.Values["user_id"].(string)
			if !ok || userID == "" {
				m.resetSession(c)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "세션에 사용자 정보가 없습니다.",
				})
			}

			// 세션 유효성 검증 (유스케이스에 위임)
			valid, err := m.sessionUseCase.ValidateSession(c.Request().Context(), sess.ID)
			if err != nil || !valid {
				m.resetSession(c)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "세션이 만료되었거나 유효하지 않습니다.",
				})
			}

			// 세션 정보를 컨텍스트에 저장
			c.Set(SessionKey, sess)
			c.Set(UserIDKey, userID)

			// 세션 갱신 (만료 시간 연장)
			if err := m.sessionUseCase.RefreshSession(c.Request().Context(), sess.ID); err != nil {
				m.logger.Warn("세션 갱신 실패", zap.Error(err))
			}

			return next(c)
		}
	}
}

// resetSession은 클라이언트의 세션을 초기화합니다.
func (m *SessionMiddleware) resetSession(c echo.Context) {
	sess, _ := session.Get(SessionKey, c)
	sess.Values = make(map[interface{}]interface{})
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1, // 세션 즉시 만료
		HttpOnly: true,
		Secure:   c.Request().TLS != nil, // HTTPS 연결에서만 Secure 설정
		SameSite: http.SameSiteStrictMode,
	}
	sess.Save(c.Request(), c.Response())
}
