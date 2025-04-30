package repository

import (
	domainrepo "github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db"
)

// NewRepositories 모든 레포지토리를 초기화하고 컬렉션을 반환합니다
func NewRepositories(infrastructure *db.Infrastructure) *domainrepo.Repositories {
	// 사용자 레포지토리
	userRepo := NewUserRepository(infrastructure.DB)

	// 토큰 레포지토리
	tokenRepo := NewTokenRepository(infrastructure.DB)

	// 감사 로그 레포지토리
	auditLogRepo := NewAuditLogRepository(infrastructure.DB)

	// 캐시 레포지토리 (Redis 기반)
	cacheRepo := db.NewRedisRepository(infrastructure.RedisClient)

	// 메일 레포지토리
	mailRepo := NewMailRepository(infrastructure.SMTPClient)

	// 레포지토리 컬렉션 생성 및 반환
	return domainrepo.NewRepositories(
		userRepo,
		tokenRepo,
		auditLogRepo,
		cacheRepo,
		mailRepo,
	)
}
