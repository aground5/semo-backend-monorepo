// services/auth/internal/usecase/session_usecase.go
package usecase

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/constants"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/interfaces"
	"go.uber.org/zap"
)

// SessionConfig 세션 관련 설정
type SessionConfig struct {
	SessionExpiry int    // 세션 만료 시간 (시간)
	SessionSecret string // 세션 암호화 비밀키
}

// SessionUseCase 세션 관리 유스케이스 구현체
type SessionUseCase struct {
	logger          *zap.Logger
	config          SessionConfig
	cacheRepository repository.CacheRepository
	userRepository  repository.UserRepository
	auditRepository repository.AuditLogRepository
	sessionStore    sessions.Store
}

// NewSessionUseCase 새 세션 유스케이스 생성
func NewSessionUseCase(
	logger *zap.Logger,
	config SessionConfig,
	cacheRepo repository.CacheRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
) interfaces.SessionUseCase {
	// 쿠키 저장소 초기화 (CookieStore)
	store := sessions.NewCookieStore([]byte(config.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   config.SessionExpiry * 3600, // 시간 단위를 초 단위로 변환
		HttpOnly: true,
		Secure:   true, // HTTPS만 허용
		SameSite: http.SameSiteDefaultMode,
	}

	return &SessionUseCase{
		logger:          logger,
		config:          config,
		cacheRepository: cacheRepo,
		userRepository:  userRepo,
		auditRepository: auditRepo,
		sessionStore:    store,
	}
}

// CreateSession은 사용자를 위한 새 세션을 생성합니다
func (uc *SessionUseCase) CreateSession(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (string, error) {
	// 세션 ID 생성
	sessionID := uuid.New().String()

	// Redis에 세션 데이터 저장 (백엔드 저장소)
	sessionData := map[string]string{
		"user_id":    userID,
		"device_id":  deviceInfo.DeviceUID.String(),
		"ip":         deviceInfo.IP,
		"user_agent": deviceInfo.UserAgent,
		"created_at": time.Now().Format(time.RFC3339),
	}

	// 만료 시간 설정
	expiry := uc.config.SessionExpiry
	if expiry <= 0 {
		expiry = constants.SessionExpiry // 기본값 사용
	}
	expiryDuration := time.Duration(expiry) * time.Hour

	// Redis에 세션 정보 저장
	sessionKey := fmt.Sprintf("%s%s", constants.SessionPrefix, sessionID)
	for k, v := range sessionData {
		key := fmt.Sprintf("%s:%s", sessionKey, k)
		if err := uc.cacheRepository.Set(ctx, key, v, expiryDuration); err != nil {
			uc.logger.Error("세션 데이터 저장 실패",
				zap.String("key", k),
				zap.Error(err),
			)
			return "", fmt.Errorf("세션 데이터 저장 실패: %w", err)
		}
	}

	// 감사 로그 기록
	content := map[string]interface{}{
		"session_id":  sessionID,
		"ip":          deviceInfo.IP,
		"user_agent":  deviceInfo.UserAgent,
		"device_id":   deviceInfo.DeviceUID.String(),
		"expire_time": time.Now().Add(expiryDuration).Format(time.RFC3339),
	}

	auditLog := &entity.AuditLog{
		UserID:  &userID,
		Type:    "SESSION_CREATED",
		Content: content,
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Warn("세션 생성 감사 로그 저장 실패", zap.Error(err))
	}

	return sessionID, nil
}

// ValidateSession은 세션 ID의 유효성을 검증합니다
func (uc *SessionUseCase) ValidateSession(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("세션 ID가 비어 있습니다")
	}

	// Redis에서 세션 정보 확인
	sessionKey := fmt.Sprintf("%s%s:user_id", constants.SessionPrefix, sessionID)
	userID, err := uc.cacheRepository.Get(ctx, sessionKey)

	if err != nil {
		// 캐시에 세션이 없음
		if uc.cacheRepository.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("세션 검증 중 오류 발생: %w", err)
	}

	if userID == "" {
		return false, nil
	}

	// 세션 유효성 확인 완료
	return true, nil
}

// GetSession은 세션 정보를 조회합니다
func (uc *SessionUseCase) GetSession(ctx context.Context, sessionID string) (*sessions.Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("세션 ID가 비어 있습니다")
	}

	// Redis에서 세션 정보 조회
	sessionKey := fmt.Sprintf("%s%s", constants.SessionPrefix, sessionID)
	userIDKey := fmt.Sprintf("%s:user_id", sessionKey)
	userID, err := uc.cacheRepository.Get(ctx, userIDKey)

	if err != nil {
		if uc.cacheRepository.IsNotFound(err) {
			return nil, fmt.Errorf("세션을 찾을 수 없습니다")
		}
		return nil, fmt.Errorf("세션 조회 실패: %w", err)
	}

	// 새 세션 객체 생성
	session := sessions.NewSession(uc.sessionStore, SessionKey)
	session.ID = sessionID
	session.Values = make(map[interface{}]interface{})
	session.Values["user_id"] = userID

	// 만료 시간 설정
	expiry := uc.config.SessionExpiry
	if expiry <= 0 {
		expiry = constants.SessionExpiry
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   expiry * 3600, // 시간 단위를 초 단위로 변환
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
	}

	// 추가 세션 데이터 조회 (필요한 경우)
	deviceIDKey := fmt.Sprintf("%s:device_id", sessionKey)
	ipKey := fmt.Sprintf("%s:ip", sessionKey)

	deviceID, _ := uc.cacheRepository.Get(ctx, deviceIDKey)
	ip, _ := uc.cacheRepository.Get(ctx, ipKey)

	if deviceID != "" {
		session.Values["device_id"] = deviceID
	}

	if ip != "" {
		session.Values["ip"] = ip
	}

	return session, nil
}

// RevokeSession은 세션을 폐기합니다
func (uc *SessionUseCase) RevokeSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("세션 ID가 비어 있습니다")
	}

	// 세션 정보 조회
	sessionKey := fmt.Sprintf("%s%s", constants.SessionPrefix, sessionID)
	userIDKey := fmt.Sprintf("%s:user_id", sessionKey)
	userID, err := uc.cacheRepository.Get(ctx, userIDKey)

	if err != nil {
		if uc.cacheRepository.IsNotFound(err) {
			// 이미 만료된 세션이므로 성공으로 처리
			return nil
		}
		return fmt.Errorf("세션 조회 실패: %w", err)
	}

	// 세션 키 삭제 (패턴 기반 삭제가 있다면 활용)
	keysToDelete := []string{
		fmt.Sprintf("%s:user_id", sessionKey),
		fmt.Sprintf("%s:device_id", sessionKey),
		fmt.Sprintf("%s:ip", sessionKey),
		fmt.Sprintf("%s:user_agent", sessionKey),
		fmt.Sprintf("%s:created_at", sessionKey),
	}

	if err := uc.cacheRepository.DeleteMulti(ctx, keysToDelete); err != nil {
		uc.logger.Error("세션 삭제 실패", zap.Error(err))
		return fmt.Errorf("세션 삭제 실패: %w", err)
	}

	// 세션 무효화 감사 로그 기록
	auditLog := &entity.AuditLog{
		UserID:  &userID,
		Type:    "SESSION_REVOKED",
		Content: map[string]interface{}{"session_id": sessionID},
	}

	if err := uc.auditRepository.Create(ctx, auditLog); err != nil {
		uc.logger.Warn("세션 폐기 감사 로그 저장 실패", zap.Error(err))
	}

	return nil
}

// RefreshSession은 세션의 만료 시간을 갱신합니다
func (uc *SessionUseCase) RefreshSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("세션 ID가 비어 있습니다")
	}

	// 세션 키 확인
	sessionKey := fmt.Sprintf("%s%s", constants.SessionPrefix, sessionID)
	userIDKey := fmt.Sprintf("%s:user_id", sessionKey)

	// 세션 존재 여부 확인
	userID, err := uc.cacheRepository.Get(ctx, userIDKey)
	if err != nil {
		if uc.cacheRepository.IsNotFound(err) {
			return fmt.Errorf("세션을 찾을 수 없습니다")
		}
		return fmt.Errorf("세션 조회 실패: %w", err)
	}

	// 만료 시간 설정
	expiry := uc.config.SessionExpiry
	if expiry <= 0 {
		expiry = constants.SessionExpiry
	}
	expiryDuration := time.Duration(expiry) * time.Hour

	// 모든 세션 키 갱신
	keysToRefresh := []string{
		userIDKey,
		fmt.Sprintf("%s:device_id", sessionKey),
		fmt.Sprintf("%s:ip", sessionKey),
		fmt.Sprintf("%s:user_agent", sessionKey),
		fmt.Sprintf("%s:created_at", sessionKey),
	}

	for _, key := range keysToRefresh {
		value, err := uc.cacheRepository.Get(ctx, key)
		if err == nil && value != "" {
			// 값이 있는 경우에만 갱신
			if err := uc.cacheRepository.Set(ctx, key, value, expiryDuration); err != nil {
				uc.logger.Warn("세션 키 갱신 실패",
					zap.String("key", key),
					zap.Error(err),
				)
			}
		}
	}

	// 세션 마지막 액세스 시간 업데이트
	lastAccessKey := fmt.Sprintf("%s:last_access", sessionKey)
	if err := uc.cacheRepository.Set(ctx, lastAccessKey, time.Now().Format(time.RFC3339), expiryDuration); err != nil {
		uc.logger.Warn("세션 마지막 액세스 시간 업데이트 실패", zap.Error(err))
	}

	return nil
}

// SessionKey는 세션 키 상수
const SessionKey = "session"
