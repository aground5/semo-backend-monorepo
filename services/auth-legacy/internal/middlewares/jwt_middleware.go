package middlewares

import (
	"authn-server/configs"
	"authn-server/internal/auth"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// userIDKey는 컨텍스트에 저장할 때 사용할 키입니다.
const userIDKey = "user_id"

// JWTMiddleware는 요청 헤더의 Authorization에서 Bearer 토큰을 추출하여 검증한 후,
// JWT의 "sub" 클레임(사용자 ID)을 컨텍스트에 저장합니다.
// ES256 (ECDSA) 기반의 서명을 검증합니다.
func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header format"})
		}
		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// 검증: ES256 기반 서명 방식이어야 합니다.
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, errors.New("unexpected signing method")
			}

			// PEM 형식의 공개키 로드
			publicKeyPEM := configs.Configs.Secrets.EcdsaPublicKey
			block, _ := pem.Decode([]byte(publicKeyPEM))
			if block == nil {
				return nil, errors.New("failed to decode PEM block containing public key")
			}

			// x509 표준에 따른 공개키 파싱
			pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			pubKey, ok := pubKeyInterface.(*ecdsa.PublicKey)
			if !ok {
				return nil, errors.New("public key is not an ECDSA public key")
			}
			return pubKey, nil
		})
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}

		// 토큰이 유효하다면 "sub" 클레임을 추출하여 컨텍스트에 저장합니다.
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			sub, ok := claims["sub"].(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "sub claim not found in token"})
			}
			// 컨텍스트에 사용자 ID 저장 (다른 핸들러에서 c.Get("user_id")로 접근 가능)
			c.Set(userIDKey, sub)
			return next(c)
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}
}

// GetUserIDFromContext는 미들웨어에 의해 설정된 user_id 값을 컨텍스트에서 추출합니다.
func GetUserIDFromContext(c echo.Context) (string, error) {
	uid := c.Get(userIDKey)
	if uid == nil {
		return "", auth.NewAuthError(auth.ErrInvalidToken, "user id not found in context")
	}
	userID, ok := uid.(string)
	if !ok {
		return "", auth.NewAuthError(auth.ErrInvalidToken, "user id has invalid type")
	}
	return userID, nil
}