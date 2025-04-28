package repository

import (
	"github.com/redis/go-redis/v9"
	domainrepo "github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/mail"
	"gorm.io/gorm"
)

// InitRepositories 모든 레포지토리를 초기화하고 컬렉션을 반환합니다
func InitRepositories(database *gorm.DB, redisClient *redis.Client, smtpClient *mail.SMTPClient) *domainrepo.Repositories {
	// 사용자 레포지토리
	userRepo := NewUserRepository(database)

	// 토큰 레포지토리
	tokenRepo := NewTokenRepository(database)

	// 감사 로그 레포지토리
	auditLogRepo := NewAuditLogRepository(database)

	// 캐시 레포지토리 (Redis 기반)
	cacheRepo := db.NewRedisRepository(redisClient)

	// 메일 레포지토리
	mailRepo := NewMailRepository(smtpClient)

	// 레포지토리 컬렉션 생성 및 반환
	return domainrepo.NewRepositories(
		userRepo,
		tokenRepo,
		auditLogRepo,
		cacheRepo,
		mailRepo,
	)
}
