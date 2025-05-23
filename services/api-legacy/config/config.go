package config

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/config"
	"github.com/wekeepgrowing/semo-backend-monorepo/pkg/logger"
	"go.uber.org/zap"
)

// Config defines the structure for all configuration settings.
type Config struct {
	Service     Service     `yaml:"service"`
	Server      Server      `yaml:"server"`
	Database    Database    `yaml:"database"`
	Log         Log         `yaml:"log"`
	Cors        Cors        `yaml:"cors"`
	Services    Services    `yaml:"services"`
	Redis       Redis       `yaml:"redis"`
	SpiceDB     SpiceDB     `yaml:"spicedb"`
	Organization Organization `yaml:"organization"`
	Email       Email       `yaml:"email"`
	Secrets     Secrets     `yaml:"secrets"`
	S3          S3          `yaml:"s3"`
	AIExecutor  AIExecutor  `yaml:"ai_executor"`
	Logger      *zap.Logger
}

// Service holds configuration for the service itself.
type Service struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// Server holds configuration for the HTTP and gRPC servers.
type Server struct {
	Port    string `yaml:"port"`
	GRPCPort string `yaml:"grpc_port"`
	Timeout string `yaml:"timeout"`
	Debug   bool   `yaml:"debug"`
}

// Database holds configuration for the database connection.
type Database struct {
	Driver          string `yaml:"driver"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Name            string `yaml:"name"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

// Log holds configuration for logging.
type Log struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Cors holds configuration for CORS settings.
type Cors struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// Services holds configuration for other microservices.
type Services struct {
	Auth    Microservice `yaml:"auth"`
	GeoLite Microservice `yaml:"geolite"`
}

// Microservice holds configuration for a single microservice.
type Microservice struct {
	Host     string `yaml:"host"`
	HTTPPort string `yaml:"http_port"`
	GRPCPort string `yaml:"grpc_port"`
	Timeout  string `yaml:"timeout"`
}

// Redis holds configuration for Redis.
type Redis struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Database  int      `yaml:"database"`
	TLS       bool     `yaml:"tls"`
}

// SpiceDB holds configuration for SpiceDB.
type SpiceDB struct {
	Address string `yaml:"address"`
	Token   string `yaml:"token"`
}

// Organization holds configuration for the organization.
type Organization struct {
	ID string `yaml:"id"`
}

// Email holds configuration for email settings.
type Email struct {
	SMTPHost    string `yaml:"smtp_host"`
	SMTPPort    int    `yaml:"smtp_port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	SenderEmail string `yaml:"sender_email"`
}

// Secrets holds configuration for various secrets.
type Secrets struct {
	GRPCHTolen    string `yaml:"grpc_token"`
	SessionSecret string `yaml:"session_secret"`
}

// S3 holds configuration for S3 bucket.
type S3 struct {
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
}

// AIExecutor holds configuration for the AI Executor.
type AIExecutor struct {
	Path                       string `yaml:"path"`
	OpenaiAPIKey               string `yaml:"openai_api_key"`
	OpenaiModel                string `yaml:"openai_model"`
	ContextSize                string `yaml:"context_size"`
	OpenaiEndpoint             string `yaml:"openai_endpoint"`
	AnthropicAPIKey            string `yaml:"anthropic_api_key"`
	AnthropicModel             string `yaml:"anthropic_model"`
	GoogleGenerativeAIAPIKey string `yaml:"google_generative_ai_api_key"`
	GoogleModel                string `yaml:"google_model"`
	GrokAPIKey                 string `yaml:"grok_api_key"`
	GrokModel                  string `yaml:"grok_model"`
	GrokEndpoint               string `yaml:"grok_endpoint"`
	LangfusePublicKey          string `yaml:"langfuse_public_key"`
	LangfuseSecretKey          string `yaml:"langfuse_secret_key"`
	LangfuseBaseURL            string `yaml:"langfuse_baseurl"`
	EnableTelemetry            string `yaml:"enable_telemetry"`
	LogLevel                   string `yaml:"log_level"`
	EnableFileLogging          string `yaml:"enable_file_logging"`
	LogDir                     string `yaml:"log_dir"`
	LogMaxSize                 string `yaml:"log_max_size"`
	LogMaxFiles                string `yaml:"log_max_files"`
}

var (
	// AppConfig holds the application's configuration.
	AppConfig *Config
)

// Load loads the configuration from the YAML file.
func Load() (*Config, error) {
	cfg, err := config.Load("api-legacy")
	if err != nil {
		return nil, err
	}

	appConfig := &Config{}

	// Service
	appConfig.Service.Name = cfg.GetString("service.name")
	appConfig.Service.Version = cfg.GetString("service.version")

	// Server
	appConfig.Server.Port = cfg.GetString("server.port")
	appConfig.Server.GRPCPort = cfg.GetString("server.grpc_port")
	appConfig.Server.Timeout = cfg.GetString("server.timeout")
	appConfig.Server.Debug = cfg.GetBool("server.debug")

	// Database
	appConfig.Database.Driver = cfg.GetString("database.driver")
	appConfig.Database.Host = cfg.GetString("database.host")
	appConfig.Database.Port = cfg.GetInt("database.port")
	appConfig.Database.Name = cfg.GetString("database.name")
	appConfig.Database.User = cfg.GetString("database.user")
	appConfig.Database.Password = cfg.GetString("database.password")
	appConfig.Database.MaxOpenConns = cfg.GetInt("database.max_open_conns")
	appConfig.Database.MaxIdleConns = cfg.GetInt("database.max_idle_conns")
	appConfig.Database.ConnMaxLifetime = cfg.GetString("database.conn_max_lifetime")

	// Log
	appConfig.Log.Level = cfg.GetString("log.level")
	appConfig.Log.Format = cfg.GetString("log.format")
	appConfig.Log.Output = cfg.GetString("log.output")

	// Cors
	appConfig.Cors.AllowedOrigins = cfg.GetStringSlice("cors.allowed_origins")
	appConfig.Cors.AllowedMethods = cfg.GetStringSlice("cors.allowed_methods")
	appConfig.Cors.AllowedHeaders = cfg.GetStringSlice("cors.allowed_headers")
	appConfig.Cors.ExposedHeaders = cfg.GetStringSlice("cors.exposed_headers")
	appConfig.Cors.AllowCredentials = cfg.GetBool("cors.allow_credentials")
	appConfig.Cors.MaxAge = cfg.GetInt("cors.max_age")

	// Services - Auth
	appConfig.Services.Auth.Host = cfg.GetString("services.auth.host")
	appConfig.Services.Auth.HTTPPort = cfg.GetString("services.auth.http_port")
	appConfig.Services.Auth.GRPCPort = cfg.GetString("services.auth.grpc_port")
	appConfig.Services.Auth.Timeout = cfg.GetString("services.auth.timeout")

	// Services - GeoLite
	appConfig.Services.GeoLite.Host = cfg.GetString("services.geolite.host")
	appConfig.Services.GeoLite.HTTPPort = cfg.GetString("services.geolite.http_port")
	appConfig.Services.GeoLite.GRPCPort = cfg.GetString("services.geolite.grpc_port")
	appConfig.Services.GeoLite.Timeout = cfg.GetString("services.geolite.timeout")

	// Redis
	appConfig.Redis.Addresses = cfg.GetStringSlice("redis.addresses")
	appConfig.Redis.Username = cfg.GetString("redis.username")
	appConfig.Redis.Password = cfg.GetString("redis.password")
	appConfig.Redis.Database = cfg.GetInt("redis.database")
	appConfig.Redis.TLS = cfg.GetBool("redis.tls")

	// SpiceDB
	appConfig.SpiceDB.Address = cfg.GetString("spicedb.address")
	appConfig.SpiceDB.Token = cfg.GetString("spicedb.token")

	// Organization
	appConfig.Organization.ID = cfg.GetString("organization.id")

	// Email
	appConfig.Email.SMTPHost = cfg.GetString("email.smtp_host")
	appConfig.Email.SMTPPort = cfg.GetInt("email.smtp_port")
	appConfig.Email.Username = cfg.GetString("email.username")
	appConfig.Email.Password = cfg.GetString("email.password")
	appConfig.Email.SenderEmail = cfg.GetString("email.sender_email")

	// Secrets
	appConfig.Secrets.GRPCHTolen = cfg.GetString("secrets.grpc_token")
	appConfig.Secrets.SessionSecret = cfg.GetString("secrets.session_secret")

	// S3
	appConfig.S3.Region = cfg.GetString("s3.region")
	appConfig.S3.BucketName = cfg.GetString("s3.bucket_name")
	appConfig.S3.AccessKey = cfg.GetString("s3.access_key")
	appConfig.S3.SecretKey = cfg.GetString("s3.secret_key")

	// AIExecutor
	appConfig.AIExecutor.Path = cfg.GetString("ai_executor.path")
	appConfig.AIExecutor.OpenaiAPIKey = cfg.GetString("ai_executor.openai_api_key")
	appConfig.AIExecutor.OpenaiModel = cfg.GetString("ai_executor.openai_model")
	appConfig.AIExecutor.ContextSize = cfg.GetString("ai_executor.context_size")
	appConfig.AIExecutor.OpenaiEndpoint = cfg.GetString("ai_executor.openai_endpoint")
	appConfig.AIExecutor.AnthropicAPIKey = cfg.GetString("ai_executor.anthropic_api_key")
	appConfig.AIExecutor.AnthropicModel = cfg.GetString("ai_executor.anthropic_model")
	appConfig.AIExecutor.GoogleGenerativeAIAPIKey = cfg.GetString("ai_executor.google_generative_ai_api_key")
	appConfig.AIExecutor.GoogleModel = cfg.GetString("ai_executor.google_model")
	appConfig.AIExecutor.GrokAPIKey = cfg.GetString("ai_executor.grok_api_key")
	appConfig.AIExecutor.GrokModel = cfg.GetString("ai_executor.grok_model")
	appConfig.AIExecutor.GrokEndpoint = cfg.GetString("ai_executor.grok_endpoint")
	appConfig.AIExecutor.LangfusePublicKey = cfg.GetString("ai_executor.langfuse_public_key")
	appConfig.AIExecutor.LangfuseSecretKey = cfg.GetString("ai_executor.langfuse_secret_key")
	appConfig.AIExecutor.LangfuseBaseURL = cfg.GetString("ai_executor.langfuse_baseurl")
	appConfig.AIExecutor.EnableTelemetry = cfg.GetString("ai_executor.enable_telemetry")
	appConfig.AIExecutor.LogLevel = cfg.GetString("ai_executor.log_level")
	appConfig.AIExecutor.EnableFileLogging = cfg.GetString("ai_executor.enable_file_logging")
	appConfig.AIExecutor.LogDir = cfg.GetString("ai_executor.log_dir")
	appConfig.AIExecutor.LogMaxSize = cfg.GetString("ai_executor.log_max_size")
	appConfig.AIExecutor.LogMaxFiles = cfg.GetString("ai_executor.log_max_files")

	// Logger
	loggerConfig := logger.Config{
		Level:       appConfig.Log.Level,
		Format:      appConfig.Log.Format,
		Output:      appConfig.Log.Output,
		Development: appConfig.Server.Debug,
	}
	appConfig.Logger, err = logger.NewZapLogger(loggerConfig)
	if err != nil {
		return nil, err
	}

	AppConfig = appConfig
	return appConfig, nil
}