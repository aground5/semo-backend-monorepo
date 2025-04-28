package db

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/mail"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Infrastructure 인프라스트럭처 구조체
type Infrastructure struct {
	DB             *gorm.DB
	RedisClient    *redis.Client
	EmailTemplates *mail.EmailTemplateService
	SMTPClient     *mail.SMTPClient
}

// NewInfrastructure 인프라스트럭처 초기화
func NewInfrastructure(cfg *config.Config) (*Infrastructure, error) {
	logger := cfg.Logger
	infrastructure := &Infrastructure{}

	// 데이터베이스 연결 설정
	dbConfig := Config{
		Driver:          cfg.Database.Driver,
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Database.ConnMaxLifetime) * time.Second,
		SSLMode:         "disable", // 필요에 따라 변경
	}

	// 데이터베이스 연결
	var err error
	infrastructure.DB, err = NewPostgresDB(dbConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 실패: %w", err)
	}

	// Redis 설정
	redisConfig := RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// Redis 클라이언트 초기화
	infrastructure.RedisClient, err = NewRedisClient(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("Redis 연결 실패: %w", err)
	}

	// 이메일 템플릿 서비스 초기화
	infrastructure.EmailTemplates = mail.NewEmailTemplateService(
		cfg.Service.BaseURL,
		cfg.Email.SenderEmail,
		cfg.Service.Name,
	)

	smtpConfig := mail.SMTPConfig{
		Host:     cfg.Email.SMTPHost,
		Port:     cfg.Email.SMTPPort,
		Username: cfg.Email.SMTPUser,
		Password: cfg.Email.SMTPPass,
		From:     cfg.Email.SenderEmail,
	}

	// SMTP 클라이언트 초기화
	infrastructure.SMTPClient = mail.NewSMTPClient(smtpConfig)

	if err != nil {
		return nil, fmt.Errorf("SMTP 클라이언트 초기화 실패: %w", err)
	}

	logger.Info("인프라스트럭처 초기화 완료",
		zap.String("database", "PostgreSQL"),
		zap.String("redis", "Redis"),
		zap.String("email", "SMTP"),
	)

	return infrastructure, nil
}

// Close 모든 연결 종료
func (i *Infrastructure) Close() error {
	// DB 연결 종료
	sqlDB, err := i.DB.DB()
	if err != nil {
		return fmt.Errorf("DB 인스턴스 획득 실패: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("데이터베이스 연결 종료 실패: %w", err)
	}

	// Redis 연결 종료
	if err := i.RedisClient.Close(); err != nil {
		return fmt.Errorf("Redis 연결 종료 실패: %w", err)
	}

	config.AppConfig.Logger.Info("모든 인프라스트럭처 연결 종료됨")
	return nil
}
