package middlewares

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"semo-server/internal/logics"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const userIDKey = "user_id"
const emailIDKey = "user_email"

// JWTMiddleware는 요청 헤더의 Authorization에서 Bearer 토큰을 추출한 후,
// 토큰에서 iss(claim)를 확인하여 해당 issuer에 맞는 공개키를 가져오고,
// 이후 토큰을 검증한 후 sub 클레임(사용자 ID)을 컨텍스트에 저장합니다.
func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Authorization 헤더에서 토큰 추출
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header format"})
		}
		tokenStr := parts[1]

		// 검증 전 토큰을 파싱하여 iss(claim)를 추출
		parser := new(jwt.Parser)
		tokenUnverified, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unable to parse token"})
		}
		claims, ok := tokenUnverified.Claims.(jwt.MapClaims)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unable to parse claims"})
		}
		issuer, ok := claims["iss"].(string)
		if !ok || issuer == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "issuer (iss) claim missing"})
		}

		// issuer에 해당하는 공개키를 gRPC를 통해 가져옴
		pkService, err := logics.NewPublicKeyService(issuer)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "failed to initialize public key service: " + err.Error()})
		}
		defer pkService.Close()

		publicKeyPEM, err := pkService.GetPublicKey()
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "failed to retrieve public key: " + err.Error()})
		}

		// 가져온 공개키를 이용하여 토큰 검증
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, errors.New("unexpected signing method")
			}
			block, _ := pem.Decode([]byte(publicKeyPEM))
			if block == nil {
				return nil, errors.New("failed to decode PEM block containing public key")
			}
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

		// 토큰이 유효하면 sub 클레임(사용자 ID)를 컨텍스트에 저장
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			sub, ok := claims["sub"].(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "sub claim not found in token"})
			}
			email, ok := claims["email"].(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "email claim not found in token"})
			}
			c.Set(userIDKey, sub)
			c.Set(emailIDKey, email)
			return next(c)
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
	}
}

// GetUserIDFromContext는 미들웨어에서 저장한 user_id를 컨텍스트에서 추출합니다.
func GetUserIDFromContext(c echo.Context) (string, error) {
	uid := c.Get(userIDKey)
	if uid == nil {
		return "", errors.New("user id not found in context")
	}
	userID, ok := uid.(string)
	if !ok {
		return "", errors.New("user id has invalid type")
	}
	return userID, nil
}

// GetEmailFromContext 미들웨어에서 저장한 user_id를 컨텍스트에서 추출합니다.
func GetEmailFromContext(c echo.Context) (string, error) {
	email := c.Get(emailIDKey)
	if email == nil {
		return "", errors.New("user id not found in context")
	}
	emailStr, ok := email.(string)
	if !ok {
		return "", errors.New("user id has invalid type")
	}
	return emailStr, nil
}
