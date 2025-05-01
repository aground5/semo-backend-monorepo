package auth

import (
	"authn-server/configs"
	"authn-server/internal/models"
	"authn-server/internal/repositories"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// TokenManager는 액세스 및 리프레시 토큰을 관리합니다.
type TokenManager struct{}

var (
	// TokenManager 전역 인스턴스
	tokenManager TokenManagerInterface
)

// GetTokenManager는 전역 TokenManager 인스턴스를 반환합니다.
func GetTokenManager() TokenManagerInterface {
	if tokenManager == nil {
		tokenManager = NewTokenManager()
	}
	return tokenManager
}

// NewTokenManager는 TokenManager 인스턴스를 생성합니다.
func NewTokenManager() TokenManagerInterface {
	return &TokenManager{}
}

// GenerateAccessToken은 사용자 ID와 이메일로 새 액세스 토큰을 생성합니다.
func (tm *TokenManager) GenerateAccessToken(user *models.User) (string, error) {
	privateKeyPEM := configs.Configs.Secrets.EcdsaPrivateKey
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return "", errors.New("failed to decode PEM block containing EC private key")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(configs.Configs.Authn.AccessJwtExpireMin) * time.Minute)

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"name":  user.Name,
		"email": user.Email,
		"iss":   configs.Configs.Service.ServiceName,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// GenerateRefreshToken은 새 리프레시 토큰을 생성합니다.
func (tm *TokenManager) GenerateRefreshToken(groupID uint) (string, *models.Token) {
	tokenStr := GenerateRandomString(64)
	now := time.Now()
	expiresAt := now.Add(time.Duration(configs.Configs.Authn.RefreshTokenExpireMin) * time.Minute)
	tokenRecord := &models.Token{
		GroupID:   groupID,
		Token:     tokenStr,
		ExpiresAt: expiresAt,
	}
	return tokenStr, tokenRecord
}

// ValidateRefreshToken은 리프레시 토큰의 유효성을 검사하고 새 토큰을 발급합니다.
func (tm *TokenManager) ValidateRefreshToken(refreshToken string) (uint, *models.User, string, error) {
	// 1) 토큰 레코드 조회
	var tokenRecord models.Token
	if dbErr := repositories.DBS.Postgres.Where("token = ?", refreshToken).First(&tokenRecord).Error; dbErr != nil {
		return 0, nil, "", NewAuthError(ErrInvalidToken, "토큰이 유효하지 않거나 취소되었습니다")
	}

	// 2) 토큰 만료 확인
	if time.Now().After(tokenRecord.ExpiresAt) {
		_ = tm.RevokeTokenGroup(tokenRecord.GroupID)
		return 0, nil, "", NewAuthError(ErrTokenExpired, "토큰이 만료되었습니다")
	}

	// 3) 토큰이 가장 최근에 생성된 것인지 확인
	var count int64
	if dbErr := repositories.DBS.Postgres.Model(&models.Token{}).
		Where("group_id = ? AND created_at > ?", tokenRecord.GroupID, tokenRecord.CreatedAt).
		Count(&count).Error; dbErr != nil {
		return 0, nil, "", NewAuthErrorWithCause(ErrInvalidToken, "토큰 검증 중 오류가 발생했습니다", dbErr)
	}

	// 4) 최신 토큰이 아니면 토큰 그룹 취소
	if count > 0 {
		_ = tm.RevokeTokenGroup(tokenRecord.GroupID)
		return 0, nil, "", NewAuthError(ErrInvalidToken, "이 리프레시 토큰은 그룹에서 최신 토큰이 아닙니다")
	}

	// 5) 토큰 그룹 조회
	var tokenGroup models.TokenGroup
	if dbErr := repositories.DBS.Postgres.First(&tokenGroup, tokenRecord.GroupID).Error; dbErr != nil {
		return 0, nil, "", NewAuthError(ErrInvalidToken, "토큰 그룹을 찾을 수 없습니다")
	}

	// 6) 사용자 조회
	var user models.User
	if dbErr := repositories.DBS.Postgres.First(&user, "id = ?", tokenGroup.UserID).Error; dbErr != nil {
		return 0, nil, "", NewAuthError(ErrUserNotFound, "사용자를 찾을 수 없습니다")
	}

	// 7) 새 리프레시 토큰 발급
	newRefreshToken, newTokenRecord := tm.GenerateRefreshToken(tokenRecord.GroupID)
	if dbErr := repositories.DBS.Postgres.Create(newTokenRecord).Error; dbErr != nil {
		return 0, nil, "", NewAuthErrorWithCause(ErrInvalidToken, "토큰 레코드 삽입 실패", dbErr)
	}

	return tokenGroup.ID, &user, newRefreshToken, nil
}

// RevokeTokenGroup은 토큰 그룹을 취소합니다 (모든 관련 토큰 무효화).
func (tm *TokenManager) RevokeTokenGroup(tokenGroupID uint) error {
	var tokenGroup models.TokenGroup
	tokenGroup.ID = tokenGroupID
	if err := repositories.DBS.Postgres.Delete(&tokenGroup).Error; err != nil {
		return fmt.Errorf("failed to delete token group: %w", err)
	}
	return nil
}

// RevokeAccessToken은 Redis에 토큰을 취소 목록에 추가하여 액세스 토큰을 취소합니다.
func (tm *TokenManager) RevokeAccessToken(accessToken string) error {
	hashValue := sha256.Sum256([]byte(accessToken))
	hashStr := hex.EncodeToString(hashValue[:])
	redisKey := fmt.Sprintf("%s%s", RevokedTokenPrefix, hashStr)

	ctx := context.Background()
	err := repositories.DBS.Redis.Set(ctx, redisKey, "true",
		time.Duration(configs.Configs.Authn.AccessJwtExpireMin)*time.Minute).Err()

	return err
}
