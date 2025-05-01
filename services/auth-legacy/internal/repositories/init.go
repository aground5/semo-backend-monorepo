package repositories

import (
	"authn-server/configs"
	"authn-server/internal/loggers"
	"authn-server/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/authzed/authzed-go/v1"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dbs struct {
	Redis    *redis.Client
	Postgres *gorm.DB
	SpiceDB  *authzed.Client
}

var DBS Dbs

func Init() {
	initRedis()
	initPostgres()
}

// initRedis initializes the Redis connection
func initRedis() {
	opt := &redis.Options{
		Addr:     configs.Configs.Redis.Addresses[0],
		Username: configs.Configs.Redis.Username,
		Password: configs.Configs.Redis.Password, // if Redis requires authentication
		DB:       configs.Configs.Redis.Database, // use default DB
	}

	// TLS가 true이면 TLSConfig 설정
	if configs.Configs.Redis.Tls {
		opt.TLSConfig = &tls.Config{
			// 필요 시, 인증서 검사 비활성화:
			// InsecureSkipVerify: true,
			// 혹은 CA 인증서 등을 설정하려면 RootCAs 설정 등 추가
		}
	}

	DBS.Redis = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := DBS.Redis.Ping(ctx).Result()
	if err != nil {
		configs.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
		return
	}

	configs.Logger.Info("Redis connected successfully", zap.String("result", result))
}

// initPostgres initializes the PostgreSQL connection
func initPostgres() {
	host, port, err := net.SplitHostPort(configs.Configs.Postgres.Address)
	if err != nil {
		configs.Logger.Fatal("Invalid Postgres address", zap.Error(err))
		return
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s",
		host,
		configs.Configs.Postgres.Username,
		configs.Configs.Postgres.Password,
		configs.Configs.Postgres.Database,
		port,
	)

	var logLevel logger.LogLevel
	if configs.Configs.Logs.LogLevel == "DEBUG" {
		logLevel = logger.LogLevel(4)
	} else if configs.Configs.Logs.LogLevel == "INFO" {
		logLevel = logger.LogLevel(4)
	} else if configs.Configs.Logs.LogLevel == "WARN" {
		logLevel = logger.LogLevel(3)
	} else if configs.Configs.Logs.LogLevel == "ERROR" {
		logLevel = logger.LogLevel(2)
	} else {
		logLevel = logger.LogLevel(1)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: loggers.NewZapGormLogger(logLevel, 200*time.Millisecond, false),
	})
	if err != nil {
		configs.Logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
		return
	}

	// 자동 마이그레이션 실행
	err = autoMigrateInOrder(db)
	if err != nil {
		panic("Failed to migrate database")
	}

	DBS.Postgres = db
	configs.Logger.Info("PostgreSQL connected successfully")
}

func autoMigrateInOrder(db *gorm.DB) error {
	// 의존 관계에 따른 마이그레이션 순서
	modelsInOrder := []interface{}{
		&models.User{},
		&models.AuditLog{},
		&models.Organization{},
		&models.TokenGroup{},
		&models.Token{},
		&models.Activity{},
		&models.LoginAttempt{},
		&models.BlockedIP{},
		&models.DeviceFingerprint{},
		// 새로 추가된 모델
		&models.HoneypotAccount{},
		&models.HoneypotActivity{},
		&models.CaptchaChallenge{},
		&models.CaptchaVerification{},
		&models.TwoFactorSecret{},
		&models.TwoFactorAttempt{},
		&models.TwoFactorChallenge{},
		&models.TrustedDevice{},
		&models.UnknownDeviceAlert{},
		&models.Notification{},
		&models.NotificationPreference{},
	}

	for _, model := range modelsInOrder {
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}
