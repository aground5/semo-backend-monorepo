package middlewares

import (
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AdminMiddleware는 특정 사용자(관리자)만 접근할 수 있는 API를 위한 미들웨어입니다.
// JWTMiddleware 다음에 체인으로 사용해야 합니다.
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// JWTMiddleware에서 설정한 userID를 가져옵니다.
		userID, err := GetUserIDFromContext(c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "인증 정보를 찾을 수 없습니다"})
		}

		// 해당 사용자가 관리자인지 확인합니다.
		// 실제 구현에서는 사용자 역할(role) 테이블이나 별도의 관리자 목록을 참조해야 합니다.
		// 여기서는 단순화를 위해 하드코딩된 목록을 사용합니다.
		var user models.User
		if err := repositories.DBS.Postgres.First(&user, "id = ?", userID).Error; err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "사용자를 찾을 수 없습니다"})
		}

		// 관리자 확인 - 실제 구현에서는 사용자 역할 테이블 확인이 필요합니다
		// 임시로 이메일 도메인이 "admin.com"인 사용자만 관리자로 취급
		isAdmin := false

		// '@' 뒤의 도메인 확인
		for _, adminDomain := range []string{"admin.com", "company.com"} {
			if len(user.Email) > len(adminDomain)+1 && user.Email[len(user.Email)-len(adminDomain)-1:] == "@"+adminDomain {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "관리자 권한이 필요합니다"})
		}

		return next(c)
	}
}