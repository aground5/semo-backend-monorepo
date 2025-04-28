package repository

// Repositories 모든 레포지토리 인터페이스의 컬렉션
type Repositories struct {
	User     UserRepository
	Token    TokenRepository
	AuditLog AuditLogRepository
	Cache    CacheRepository
	Mail     MailRepository
}

// NewRepositories 모든 레포지토리를 포함하는 컬렉션 생성
func NewRepositories(
	userRepo UserRepository,
	tokenRepo TokenRepository,
	auditLogRepo AuditLogRepository,
	cacheRepo CacheRepository,
	mailRepo MailRepository,
) *Repositories {
	return &Repositories{
		User:     userRepo,
		Token:    tokenRepo,
		AuditLog: auditLogRepo,
		Cache:    cacheRepo,
		Mail:     mailRepo,
	}
}
