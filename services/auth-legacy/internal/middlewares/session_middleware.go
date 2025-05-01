package middlewares

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// sessionKeyContext는 컨텍스트에 세션을 저장할 때 사용할 키입니다.
const sessionKeyContext = "session_data"

// SessionMiddleware는 HTTP 요청에서 세션을 가져와 컨텍스트에 저장합니다.
// 세션을 가져오는 중 에러가 발생하면 클라이언트 세션 쿠키를 초기화합니다.
func SessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			// 세션 오류 발생 시 세션 쿠키 초기화
			resetSessionCookie(c)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "세션 오류가 발생했습니다. 다시 로그인해주세요.", "dont_raise_error": "true"})
		}

		// 세션을 컨텍스트에 저장
		c.Set(sessionKeyContext, sess)
		return next(c)
	}
}

// resetSessionCookie는 클라이언트의 세션 쿠키를 초기화합니다.
func resetSessionCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = "session"
	cookie.Value = ""
	cookie.Path = "/"
	cookie.MaxAge = -1 // 즉시 만료
	cookie.HttpOnly = true
	c.SetCookie(cookie)
}

// GetSessionFromContext는 컨텍스트에서 세션을 가져옵니다.
func GetSessionFromContext(c echo.Context) (*sessions.Session, error) {
	sessionData := c.Get(sessionKeyContext)
	if sessionData == nil {
		// 컨텍스트에 세션이 없으면 직접 가져오기 시도
		sess, err := session.Get("session", c)
		if err != nil {
			return nil, err
		}
		return sess, nil
	}

	sess, ok := sessionData.(*sessions.Session)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "세션 타입이 유효하지 않습니다")
	}

	return sess, nil
}
