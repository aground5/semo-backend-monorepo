package repositories

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	// "semo-server/configs-legacy" // 기존 레거시 설정 import 제거
	apilegacyconfig "semo-server/config" // 새 설정 패키지 import (main.go와 동일한 패키지 경로 사용)
	"semo-server/internal/loggers"
	"semo-server/internal/models"
	"time"

	"github.com/authzed/authzed-go/v1"
	awsconfig "github.com/aws/aws-sdk-go-v2/config" // alias 추가하여 이름 충돌 방지
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"

	"go.mongodb.org/mongo-driver/mongo"
)

type dbs struct {
	Redis    *redis.Client
	Postgres *gorm.DB
	S3       *s3.Client
	SpiceDB  *authzed.Client
	MongoDB  *mongo.Client
}

// Singleton 패턴으로 한번만 초기화
var DBS dbs

// Init 함수가 새로운 설정 객체(appCfg)를 받도록 수정
func Init(logger *zap.Logger, appCfg *apilegacyconfig.Config) {
	initRedis(logger, appCfg)
	initPostgres(logger, appCfg)
	initS3(logger, appCfg)
	initSpiceDB(logger, appCfg)
	//initMongoDB(appCfg) // 필요하다면 MongoDB 초기화도 수정
}

// initRedis initializes the Redis connection
// initRedis 함수도 새로운 설정 객체(appCfg)를 받도록 수정
func initRedis(logger *zap.Logger, appCfg *apilegacyconfig.Config) {
	if len(appCfg.Redis.Addresses) == 0 { // Addresses 슬라이스가 비어있는지 확인
		logger.Error("Redis addresses are not configured.")
		// Redis 연결이 필수적인 경우 여기서 Fatal 처리하거나,
		// 선택적인 경우 연결을 시도하지 않고 반환할 수 있습니다.
		// 여기서는 일단 에러 로깅 후 반환하도록 처리합니다.
		// DBS.Redis = nil // 또는 Redis를 사용하지 않는다는 표시
		return
	}

	opt := &redis.Options{
		Addr:     appCfg.Redis.Addresses[0], // 새로운 설정 객체 사용
		Username: appCfg.Redis.Username,      // 새로운 설정 객체 사용
		Password: appCfg.Redis.Password,      // 새로운 설정 객체 사용
		DB:       appCfg.Redis.Database,      // 새로운 설정 객체 사용
	}

	if appCfg.Redis.TLS { // 새로운 설정 객체 사용
		opt.TLSConfig = &tls.Config{}
	}

	DBS.Redis = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := DBS.Redis.Ping(ctx).Result()
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
		return
	}

	logger.Info("Redis connected successfully", zap.String("result", result))
}

// initPostgres 함수도 새로운 설정 객체(appCfg)를 받도록 수정
func initPostgres(logger *zap.Logger, appCfg *apilegacyconfig.Config) {
	// configs.Configs.Postgres.Address 대신 appCfg.Database 사용
	// api-legacy.yaml에는 database.host 와 database.port 가 분리되어 있음
	dbAddr := fmt.Sprintf("%s:%d", appCfg.Database.Host, appCfg.Database.Port)

	host, port, err := net.SplitHostPort(dbAddr) // dbAddr 사용
	if err != nil {
		logger.Fatal("Invalid Postgres address", zap.Error(err), zap.String("address", dbAddr))
		return
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", // sslmode=disable 추가 또는 설정에 따라 변경
		host,
		appCfg.Database.User,     // 새로운 설정 객체 사용
		appCfg.Database.Password, // 새로운 설정 객체 사용
		appCfg.Database.Name,     // 새로운 설정 객체 사용
		port,
	)

	gormLogger := loggers.NewZapGormLogger(
		logger, 
		gLogger.Info,
		200*time.Millisecond,
		true,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
		return
	}

	err = autoMigrateInOrder(db)
	if err != nil {
		// panic("Failed to migrate database") // panic 대신 Fatal 로깅
				logger.Fatal("Failed to migrate database", zap.Error(err))
				return
	}

	DBS.Postgres = db
	logger.Info("PostgreSQL connected successfully")
}

func autoMigrateInOrder(db *gorm.DB) error {
	// ... (기존 코드 유지) ...
	modelsInOrder := []interface{}{
		&models.Profile{},
		&models.Team{},
		&models.TeamMember{},
		&models.Attribute{},
		&models.Item{},
		&models.UserTests{},
		&models.File{},
		&models.AttributeValue{},
		&models.Activity{},
		&models.Notification{},
		&models.Comment{},
		&models.Evaluate{},
		&models.Entry{},
		&models.Share{},
	}

	for _, model := range modelsInOrder {
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}


// initS3 함수도 새로운 설정 객체(appCfg)를 받도록 수정
func initS3(logger *zap.Logger, appCfg *apilegacyconfig.Config) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(appCfg.S3.Region), // 새로운 설정 객체 사용
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				appCfg.S3.AccessKey,  // 새로운 설정 객체 사용
				appCfg.S3.SecretKey,  // 새로운 설정 객체 사용
				"",
			),
		),
	)
	if err != nil {
		logger.Fatal("AWS S3 설정 로드 실패", zap.Error(err))
		return
	}

	DBS.S3 = s3.NewFromConfig(awsCfg)
	logger.Info("S3 클라이언트가 성공적으로 초기화되었습니다")
}

// initSpiceDB 함수도 새로운 설정 객체(appCfg)를 받도록 수정
func initSpiceDB(logger *zap.Logger, appCfg *apilegacyconfig.Config) {
	client, err := authzed.NewClient(
		appCfg.SpiceDB.Address, // 새로운 설정 객체 사용
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenAuth{token: appCfg.SpiceDB.Token}), // 새로운 설정 객체 사용
	)
	if err != nil {
		logger.Fatal("failed to create authzed client", zap.Error(err))
		return
	}

	DBS.SpiceDB = client
	// configs.Logger.Info 대신 logger.Info 사용
	logger.Info("SpiceDB 클라이언트가 성공적으로 초기화되었습니다")
}

type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t tokenAuth) RequireTransportSecurity() bool {
	return false
}

// func initMongoDB() { ... } // 필요하다면 이 함수도 수정