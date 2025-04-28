package usecase

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/constants"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// TokenConfig 토큰 관련 설정
type TokenConfig struct {
	ServiceName        string // 서비스 이름
	JwtPrivateKey      string // ECDSA 개인키 (PEM 형식)
	JwtPublicKey       string // ECDSA 공개키 (PEM 형식)
	AccessTokenExpiry  int    // 액세스 토큰 만료 시간 (분)
	RefreshTokenExpiry int    // 리프레시 토큰 만료 시간 (분)
}

// TokenUseCase 토큰 유스케이스 구현체
type TokenUseCase struct {
	logger          *zap.Logger
	config          TokenConfig
	tokenRepository repository.TokenRepository
	userRepository  repository.UserRepository
	cacheRepository repository.CacheRepository
	auditRepository repository.AuditLogRepository
}

// NewTokenUseCase 새 토큰 유스케이스 생성
func NewTokenUseCase(
	logger *zap.Logger,
	config TokenConfig,
	tokenRepo repository.TokenRepository,
	userRepo repository.UserRepository,
	cacheRepo repository.CacheRepository,
	auditRepo repository.AuditLogRepository,
) interfaces.TokenUseCase {
	return &TokenUseCase{
		logger:          logger,
		config:          config,
		tokenRepository: tokenRepo,
		userRepository:  userRepo,
		cacheRepository: cacheRepo,
		auditRepository: auditRepo,
	}
}

// GenerateAccessToken 사용자 정보로부터 액세스 토큰 생성
func (uc *TokenUseCase) GenerateAccessToken(ctx context.Context, user *entity.User) (string, error) {
	// JWT 개인키 로드
	privateKeyPEM := uc.config.JwtPrivateKey
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return "", fmt.Errorf("EC 개인키 디코딩 실패")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		uc.logger.Error("개인키 파싱 실패", zap.Error(err))
		return "", fmt.Errorf("개인키 파싱 실패: %w", err)
	}

	// 토큰 만료 시간 설정
	now := time.Now()
	expiry := uc.config.AccessTokenExpiry
	if expiry <= 0 {
		expiry = constants.AccessTokenExpiry // 기본값 사용
	}
	expiresAt := now.Add(time.Duration(expiry) * time.Minute)

	// JWT 클레임 설정
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"name":  user.Name,
		"email": user.Email,
		"iss":   uc.config.ServiceName,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
		"type":  "access",
	}

	// 토큰 생성 및 서명
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		uc.logger.Error("액세스 토큰 서명 실패", zap.Error(err))
		return "", fmt.Errorf("액세스 토큰 서명 실패: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken 리프레시 토큰 생성
func (uc *TokenUseCase) GenerateRefreshToken(ctx context.Context, tokenGroupID uint) (string, *entity.Token, error) {
	// 랜덤 리프레시 토큰 생성
	tokenStr := GenerateRandomString(64)

	// 토큰 만료 시간 설정
	now := time.Now()
	expiry := uc.config.RefreshTokenExpiry
	if expiry <= 0 {
		expiry = constants.RefreshTokenExpiry // 기본값 사용
	}
	expiresAt := now.Add(time.Duration(expiry) * time.Minute)

	// 토큰 엔티티 생성
	token := entity.NewToken(tokenGroupID, tokenStr, "refresh", expiresAt, now)

	return tokenStr, token, nil
}

// ValidateRefreshToken 리프레시 토큰 검증
func (uc *TokenUseCase) ValidateRefreshToken(ctx context.Context, refreshToken string) (uint, *entity.User, string, error) {
	// 1) 리프레시 토큰 조회
	token, err := uc.tokenRepository.FindByToken(ctx, refreshToken)
	if err != nil {
		uc.logger.Error("리프레시 토큰 조회 실패", zap.Error(err))
		return 0, nil, "", fmt.Errorf("유효하지 않은 리프레시 토큰")
	}

	if token == nil || token.TokenType != "refresh" {
		return 0, nil, "", fmt.Errorf("유효하지 않거나 취소된 토큰")
	}

	// 2) 토큰 만료 확인
	now := time.Now()
	if now.After(token.ExpiresAt) {
		uc.RevokeTokenGroup(ctx, token.GroupID)
		return 0, nil, "", fmt.Errorf("리프레시 토큰이 만료됨")
	}

	// 3) 토큰 그룹 조회
	tokenGroup, err := uc.tokenRepository.FindGroupByID(ctx, token.GroupID)
	if err != nil {
		uc.logger.Error("토큰 그룹 조회 실패", zap.Error(err))
		return 0, nil, "", fmt.Errorf("토큰 그룹을 찾을 수 없음")
	}

	// 4) 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, tokenGroup.UserID)
	if err != nil {
		uc.logger.Error("사용자 조회 실패", zap.Error(err))
		return 0, nil, "", fmt.Errorf("사용자를 찾을 수 없음")
	}

	// 5) 새 리프레시 토큰 생성
	newRefreshToken, newToken, err := uc.GenerateRefreshToken(ctx, token.GroupID)
	if err != nil {
		return 0, nil, "", fmt.Errorf("새 리프레시 토큰 생성 실패: %w", err)
	}

	// 6) 새 리프레시 토큰 저장
	if err := uc.tokenRepository.Create(ctx, newToken); err != nil {
		uc.logger.Error("새 리프레시 토큰 저장 실패", zap.Error(err))
		return 0, nil, "", fmt.Errorf("새 리프레시 토큰 저장 실패: %w", err)
	}

	// 7) 기존 리프레시 토큰 삭제
	if err := uc.tokenRepository.Delete(ctx, token.ID); err != nil {
		uc.logger.Warn("사용된 리프레시 토큰 삭제 실패", zap.Error(err))
	}

	// 8) 감사 로그 생성
	content := map[string]interface{}{
		"user_id":        user.ID,
		"token_group_id": token.GroupID,
	}
	auditLog := &entity.AuditLog{
		UserID:  &user.ID,
		Type:    entity.AuditLogTypeTokenRefresh,
		Content: content,
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Warn("토큰 갱신 감사 로그 저장 실패", zap.Error(err))
	}

	return tokenGroup.ID, user, newRefreshToken, nil
}

// RevokeTokenGroup 토큰 그룹 폐기
func (uc *TokenUseCase) RevokeTokenGroup(ctx context.Context, tokenGroupID uint) error {
	return uc.tokenRepository.DeleteByGroup(ctx, tokenGroupID)
}

// RevokeAccessToken 액세스 토큰 폐기
func (uc *TokenUseCase) RevokeAccessToken(ctx context.Context, accessToken string) error {
	// 토큰 해시 생성
	hashValue := sha256.Sum256([]byte(accessToken))
	hashStr := hex.EncodeToString(hashValue[:])
	redisKey := fmt.Sprintf("%s%s", constants.RevokedTokenPrefix, hashStr)

	// 폐기 토큰 목록에 추가 (액세스 토큰 만료 시간까지)
	expiry := uc.config.AccessTokenExpiry
	if expiry <= 0 {
		expiry = constants.AccessTokenExpiry // 기본값 사용
	}
	expiryDuration := time.Duration(expiry) * time.Minute

	// 폐기 토큰 캐시에 저장
	if err := uc.cacheRepository.Set(ctx, redisKey, "true", expiryDuration); err != nil {
		uc.logger.Error("액세스 토큰 폐기 실패", zap.Error(err))
		return fmt.Errorf("액세스 토큰 폐기 실패: %w", err)
	}

	return nil
}

// ValidateAccessToken 액세스 토큰 검증
func (uc *TokenUseCase) ValidateAccessToken(ctx context.Context, accessToken string) (*entity.User, error) {
	// 1) 토큰 폐기 여부 확인
	hashValue := sha256.Sum256([]byte(accessToken))
	hashStr := hex.EncodeToString(hashValue[:])
	redisKey := fmt.Sprintf("%s%s", constants.RevokedTokenPrefix, hashStr)

	revoked, err := uc.cacheRepository.Get(ctx, redisKey)
	if err == nil && revoked == "true" {
		return nil, fmt.Errorf("폐기된 액세스 토큰")
	}

	// 2) JWT 공개키 로드
	publicKeyPEM := uc.config.JwtPublicKey
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("EC 공개키 디코딩 실패")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		uc.logger.Error("공개키 파싱 실패", zap.Error(err))
		return nil, fmt.Errorf("공개키 파싱 실패: %w", err)
	}

	// 3) 토큰 파싱 및 검증
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		// 서명 알고리즘 확인
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("잘못된 서명 알고리즘: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	// 4) 토큰 검증 오류 처리
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, fmt.Errorf("액세스 토큰이 만료됨")
			}
		}
		return nil, fmt.Errorf("액세스 토큰 검증 실패: %w", err)
	}

	// 5) 클레임 정보 추출
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("유효하지 않은 액세스 토큰")
	}

	// 토큰 타입 확인
	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return nil, fmt.Errorf("액세스 토큰이 아님")
	}

	// 6) 사용자 ID 추출
	userID, _ := claims["sub"].(string)
	if userID == "" {
		return nil, fmt.Errorf("토큰에 사용자 ID가 없음")
	}

	// 7) 사용자 조회
	user, err := uc.userRepository.FindByID(ctx, userID)
	if err != nil {
		uc.logger.Error("사용자 조회 실패", zap.Error(err))
		return nil, fmt.Errorf("사용자를 찾을 수 없음: %w", err)
	}

	return user, nil
}
