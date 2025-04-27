package config

import (
	"github.com/your-org/semo-backend-monorepo/pkg/config"
	"github.com/your-org/semo-backend-monorepo/pkg/logger"
	"go.uber.org/zap"
)

// Config 인증 서비스 설정 구조체
type Config struct {
	// 서비스 기본 정보
	Service struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
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

	// JWT 설정
	JWT struct {
		Secret             string `yaml:"secret"`
		AccessTokenExpiry  string `yaml:"access_token_expiry"`
		RefreshTokenExpiry string `yaml:"refresh_token_expiry"`
	} `yaml:"jwt"`

	// 로그 설정
	Log struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"log"`

	// OAuth 설정
	OAuth struct {
		Google struct {
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RedirectURL  string `yaml:"redirect_url"`
		} `yaml:"google"`
		Github struct {
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RedirectURL  string `yaml:"redirect_url"`
		} `yaml:"github"`
	} `yaml:"oauth"`

	// 인증 설정
	Auth struct {
		PasswordMinLength int `yaml:"password_min_length"`
		HashCost          int `yaml:"hash_cost"`
	} `yaml:"auth"`

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
	cfg, err := config.Load("auth")
	if err != nil {
		return nil, err
	}

	// Config 구조체 생성
	appConfig := &Config{}

	// 서비스 정보
	appConfig.Service.Name = cfg.GetString("service.name")
	appConfig.Service.Version = cfg.GetString("service.version")

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

	// JWT 설정
	appConfig.JWT.Secret = cfg.GetString("jwt.secret")
	appConfig.JWT.AccessTokenExpiry = cfg.GetString("jwt.access_token_expiry")
	appConfig.JWT.RefreshTokenExpiry = cfg.GetString("jwt.refresh_token_expiry")

	// 로그 설정
	appConfig.Log.Level = cfg.GetString("log.level")
	appConfig.Log.Format = cfg.GetString("log.format")
	appConfig.Log.Output = cfg.GetString("log.output")

	// OAuth 설정
	appConfig.OAuth.Google.ClientID = cfg.GetString("oauth.google.client_id")
	appConfig.OAuth.Google.ClientSecret = cfg.GetString("oauth.google.client_secret")
	appConfig.OAuth.Google.RedirectURL = cfg.GetString("oauth.google.redirect_url")
	appConfig.OAuth.Github.ClientID = cfg.GetString("oauth.github.client_id")
	appConfig.OAuth.Github.ClientSecret = cfg.GetString("oauth.github.client_secret")
	appConfig.OAuth.Github.RedirectURL = cfg.GetString("oauth.github.redirect_url")

	// 인증 설정
	appConfig.Auth.PasswordMinLength = cfg.GetInt("auth.password_min_length")
	appConfig.Auth.HashCost = cfg.GetInt("auth.hash_cost")

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
