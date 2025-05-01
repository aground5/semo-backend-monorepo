package config

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger"
	"go.uber.org/zap"
)

// Config API 서비스 설정 구조체
type Config struct {
	// 서비스 기본 정보
	Service struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"service"`

	// 서버 설정
	Server struct {
		// HTTP 서버 설정
		HTTP struct {
			Port    string `yaml:"port"`
			Timeout int    `yaml:"timeout"`
			Debug   bool   `yaml:"debug"`
		} `yaml:"http"`

		// gRPC 서버 설정
		GRPC struct {
			Port    string `yaml:"port"`
			Timeout int    `yaml:"timeout"`
		} `yaml:"grpc"`
	} `yaml:"server"`

	// 데이터베이스 설정
	Database struct {
		Driver          string `yaml:"driver"`
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		Name            string `yaml:"name"`
		User            string `yaml:"user"`
		Password        string `yaml:"password"`
		MaxOpenConns    int    `yaml:"max_open_conns"`
		MaxIdleConns    int    `yaml:"max_idle_conns"`
		ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	} `yaml:"database"`

	// Redis 설정
	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	// JWT 설정
	JWT struct {
		Secret             string `yaml:"secret"`
		PrivateKey         string `yaml:"private_key"`
		PublicKey          string `yaml:"public_key"`
		AccessTokenExpiry  int    `yaml:"access_token_expiry"`
		RefreshTokenExpiry int    `yaml:"refresh_token_expiry"`
	} `yaml:"jwt"`

	// 로그 설정
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"log"`

	// S3 설정
	S3 struct {
		Endpoint  string `yaml:"endpoint"`
		Region    string `yaml:"region"`
		Bucket    string `yaml:"bucket"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"s3"`

	// Auth 설정
	Auth struct {
		Endpoint string `yaml:"endpoint"`
	} `yaml:"auth"`

	// AI 설정
	AI struct {
		OpenAIAPIKey      string `yaml:"openai_api_key"`
		AnthropicAPIKey   string `yaml:"anthropic_api_key"`
		GeminiAPIKey      string `yaml:"gemini_api_key"`
		DefaultModel      string `yaml:"default_model"`
		EmbeddingEndpoint string `yaml:"embedding_endpoint"`
	} `yaml:"ai"`

	// 로거 인스턴스
	Logger *zap.Logger
}

var (
	// AppConfig는 어플리케이션 전체에서 사용하는 설정 인스턴스입니다.
	AppConfig *Config
)

// Load 설정 파일 로드
func Load() (*Config, error) {
	// pkg/config 패키지를 사용하여 설정 파일 로드
	cfg, err := config.Load("api")
	if err != nil {
		return nil, err
	}

	// Config 구조체 생성
	appConfig := &Config{}

	// 서비스 정보
	appConfig.Service.Name = cfg.GetString("service.name")
	appConfig.Service.Version = cfg.GetString("service.version")
	appConfig.Service.BaseURL = cfg.GetString("service.base_url")

	// HTTP 서버 설정
	appConfig.Server.HTTP.Port = cfg.GetString("server.port")
	appConfig.Server.HTTP.Timeout = cfg.GetInt("server.timeout")
	appConfig.Server.HTTP.Debug = cfg.GetBool("server.debug")

	// gRPC 서버 설정
	appConfig.Server.GRPC.Port = cfg.GetString("server.grpc.port")
	appConfig.Server.GRPC.Timeout = cfg.GetInt("server.grpc.timeout")

	// 데이터베이스 설정
	appConfig.Database.Driver = cfg.GetString("database.driver")
	appConfig.Database.Host = cfg.GetString("database.host")
	appConfig.Database.Port = cfg.GetInt("database.port")
	appConfig.Database.Name = cfg.GetString("database.name")
	appConfig.Database.User = cfg.GetString("database.user")
	appConfig.Database.Password = cfg.GetString("database.password")
	appConfig.Database.MaxOpenConns = cfg.GetInt("database.max_open_conns")
	appConfig.Database.MaxIdleConns = cfg.GetInt("database.max_idle_conns")
	appConfig.Database.ConnMaxLifetime = cfg.GetInt("database.conn_max_lifetime")

	// Redis 설정
	appConfig.Redis.Host = cfg.GetString("redis.host")
	appConfig.Redis.Port = cfg.GetInt("redis.port")
	appConfig.Redis.Password = cfg.GetString("redis.password")
	appConfig.Redis.DB = cfg.GetInt("redis.db")

	// JWT 설정
	appConfig.JWT.Secret = cfg.GetString("jwt.secret")
	appConfig.JWT.PrivateKey = cfg.GetString("jwt.private_key")
	appConfig.JWT.PublicKey = cfg.GetString("jwt.public_key")
	appConfig.JWT.AccessTokenExpiry = cfg.GetInt("jwt.access_token_expiry")
	appConfig.JWT.RefreshTokenExpiry = cfg.GetInt("jwt.refresh_token_expiry")

	// 로그 설정
	appConfig.Log.Level = cfg.GetString("log.level")
	appConfig.Log.Format = cfg.GetString("log.format")
	appConfig.Log.Output = cfg.GetString("log.output")

	// S3 설정
	appConfig.S3.Endpoint = cfg.GetString("s3.endpoint")
	appConfig.S3.Region = cfg.GetString("s3.region")
	appConfig.S3.Bucket = cfg.GetString("s3.bucket")
	appConfig.S3.AccessKey = cfg.GetString("s3.access_key")
	appConfig.S3.SecretKey = cfg.GetString("s3.secret_key")

	// Auth 설정
	appConfig.Auth.Endpoint = cfg.GetString("auth.endpoint")

	// AI 설정
	appConfig.AI.OpenAIAPIKey = cfg.GetString("ai.openai_api_key")
	appConfig.AI.AnthropicAPIKey = cfg.GetString("ai.anthropic_api_key")
	appConfig.AI.GeminiAPIKey = cfg.GetString("ai.gemini_api_key")
	appConfig.AI.DefaultModel = cfg.GetString("ai.default_model")
	appConfig.AI.EmbeddingEndpoint = cfg.GetString("ai.embedding_endpoint")

	// 로거 설정
	loggerConfig := logger.Config{
		Level:       appConfig.Log.Level,
		Format:      appConfig.Log.Format,
		Output:      appConfig.Log.Output,
		Development: appConfig.Server.HTTP.Debug,
	}

	// 로거 생성
	appConfig.Logger, err = logger.NewZapLogger(loggerConfig)
	if err != nil {
		return nil, err
	}

	// 전역 변수에 설정
	AppConfig = appConfig

	return appConfig, nil
}
