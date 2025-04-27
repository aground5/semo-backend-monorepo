package db

import (
	"fmt"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Config 데이터베이스 설정
type Config struct {
	Driver          string
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	SSLMode         string
}

// NewPostgresDB PostgreSQL 데이터베이스 연결을 생성합니다.
func NewPostgresDB(config Config, zapLogger *zap.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Name,
		config.SSLMode,
	)

	// GORM 로거 설정
	gormLogger := logger.NewGormLogger(
		zapLogger,
		gormlogger.Info,
		3*time.Second, // Slow SQL 임계값
		true,          // ErrRecordNotFound 무시
	)

	// DB 연결
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 실패: %w", err)
	}

	// 연결 풀 설정
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("SQL DB 인스턴스 획득 실패: %w", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// 연결 테스트
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("데이터베이스 핑 실패: %w", err)
	}

	zapLogger.Info("데이터베이스 연결 성공",
		zap.String("driver", config.Driver),
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.String("database", config.Name),
		zap.Int("max_open_conns", config.MaxOpenConns),
		zap.Int("max_idle_conns", config.MaxIdleConns),
		zap.Duration("conn_max_lifetime", config.ConnMaxLifetime),
	)

	return db, nil
}
