package repositories

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"semo-server/configs"
	"semo-server/internal/loggers"
	"semo-server/internal/models"
	"time"

	"github.com/authzed/authzed-go/v1"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func Init() {
	initRedis()
	initPostgres()
	initS3()
	initSpiceDB()
	//initMongoDB()
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

	// Create custom GORM logger
	gormLogger := loggers.NewZapGormLogger(
		logger.Info,                    // LogLevel
		200*time.Millisecond,           // SlowThreshold
		true,                           // IgnoreRecordNotFoundError
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		configs.Logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
		return
	}

	//자동 마이그레이션 실행
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

func initS3() {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(configs.Configs.S3.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				configs.Configs.S3.AccessKey,
				configs.Configs.S3.SecretKey,
				"",
			),
		),
	)
	if err != nil {
		configs.Logger.Fatal("AWS S3 설정 로드 실패", zap.Error(err))
		return
	}

	// S3 클라이언트 생성
	DBS.S3 = s3.NewFromConfig(cfg)
	configs.Logger.Info("S3 클라이언트가 성공적으로 초기화되었습니다")
}

func initSpiceDB() {
	client, err := authzed.NewClient(
		configs.Configs.SpiceDB.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenAuth{token: configs.Configs.SpiceDB.Token}),
	)
	if err != nil {
		configs.Logger.Fatal("failed to create authzed client", zap.Error(err))
		return
	}

	DBS.SpiceDB = client
	configs.Logger.Info("SpiceDB 클라이언트가 성공적으로 초기화되었습니다")
}

// tokenAuth implements the PerRPCCredentials interface.
type tokenAuth struct {
	token string
}

// GetRequestMetadata adds the authorization header with the token.
func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

// RequireTransportSecurity returns false because we're using insecure credentials.
// If you were using TLS, you would typically return true.
func (t tokenAuth) RequireTransportSecurity() bool {
	return false
}

// initMongoDB initializes the MongoDB connection
func initMongoDB() {
	// Set up connection options with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create MongoDB connection options
	clientOptions := options.Client().
		ApplyURI(configs.Configs.MongoDB.Uri).
		SetAuth(options.Credential{
			Username: configs.Configs.MongoDB.Username,
			Password: configs.Configs.MongoDB.Password,
		})

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		configs.Logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		return
	}

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		configs.Logger.Fatal("Failed to ping MongoDB", zap.Error(err))
		return
	}

	// Set the client and database in the global variable
	DBS.MongoDB = client

	configs.Logger.Info("MongoDB connected successfully")
}
