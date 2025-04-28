package interfaces

import (
	"context"

	"github.com/gorilla/sessions"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/usecase/dto"
)

// SessionUseCase 세션 관리를 위한 유스케이스 인터페이스
type SessionUseCase interface {
	// CreateSession은 사용자를 위한 새 세션을 생성합니다
	CreateSession(ctx context.Context, userID string, deviceInfo dto.DeviceInfo) (string, error)

	// ValidateSession은 세션 ID의 유효성을 검증합니다
	ValidateSession(ctx context.Context, sessionID string) (bool, error)

	// GetSession은 지정된 세션 ID에 해당하는 세션 정보를 반환합니다
	GetSession(ctx context.Context, sessionID string) (*sessions.Session, error)

	// RevokeSession은 세션을 폐기합니다
	RevokeSession(ctx context.Context, sessionID string) error

	// RefreshSession은 세션의 만료 시간을 갱신합니다
	RefreshSession(ctx context.Context, sessionID string) error
}
