package configs

import (
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type configs struct {
	Postgres      PostgresConfig       `yaml:"postgres"`
	MongoDB       MongoDBConfig        `yaml:"mongodb"`
	Redis         RedisConfig          `yaml:"redis"`
	SpiceDB       SpiceDBConfig        `yaml:"spicedb"`
	Email         EmailConfig          `yaml:"email"`
	Service       ServiceConfig        `yaml:"service"`
	Logs          LogsConfig           `yaml:"logs"`
	Secrets       Secrets              `yaml:"secrets"`
	Microservices []MicroserviceConfig `yaml:"microservices"`
	S3            S3Config             `yaml:"s3"`
	Openai        OpenaiConfig         `yaml:"openai"`
	AiExecutor    AiExecutor           `yaml:"ai_executor"`
	Organization  OrganizationConfig   `yaml:"organization"`
}

var Configs configs

func Init(ConfigPath *string) {
	var configPath string
	if ConfigPath == nil {
		_, b, _, _ := runtime.Caller(0)
		BasePath := filepath.Dir(b)
		configPath = BasePath + "/file/configs.yaml"
	} else {
		configPath = *ConfigPath
	}
	load(configPath)

	InitLogger()
}

func load(ConfigsPath string) {
	yamlFile, err := os.ReadFile(ConfigsPath)
	if err != nil {
		Logger.Error("Unmarshal", zap.Error(err))
	}
	err = yaml.Unmarshal(yamlFile, &Configs)
	if err != nil {
		Logger.Error("Unmarshal", zap.Error(err))
	}
}
