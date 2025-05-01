package configs

import (
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Tconfigs struct {
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
	SpiceDB  SpiceDBConfig  `yaml:"spicedb"`
	Email    EmailConfig    `yaml:"email"`
	Service  ServiceConfig  `yaml:"service"`
	Logs     LogsConfig     `yaml:"logs"`
	Secrets  Secrets        `yaml:"secrets"`
	Authn    AuthnConfig    `yaml:"authn"`
}

var Configs Tconfigs

func Init(ConfigPath *string) {
	var configPath string
	if ConfigPath == nil || *ConfigPath == "" {
		// Find default config locations
		// 1. Check for ./configs.yaml (relative to working directory)
		// 2. Check for config in same directory as executable
		// 3. Check relative to this source file
		if _, err := os.Stat("./configs.yaml"); err == nil {
			configPath = "./configs.yaml"
		} else if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			candidatePath := filepath.Join(execDir, "configs.yaml")
			if _, err := os.Stat(candidatePath); err == nil {
				configPath = candidatePath
			} else {
				// Fallback to config relative to this source file
				_, b, _, _ := runtime.Caller(0)
				basePath := filepath.Dir(b)
				configPath = filepath.Join(basePath, "file", "configs.yaml")
			}
		} else {
			// Final fallback
			_, b, _, _ := runtime.Caller(0)
			basePath := filepath.Dir(b)
			configPath = filepath.Join(basePath, "file", "configs.yaml")
		}
	} else {
		configPath = *ConfigPath
	}

	load(configPath)
	InitLogger()
}

func load(ConfigsPath string) {
	yamlFile, err := os.ReadFile(ConfigsPath)
	if err != nil {
		// If Logger is not initialized yet, print to stderr
		if Logger == nil {
			os.Stderr.WriteString("Error reading config file: " + err.Error() + "\n")
		} else {
			Logger.Error("Error reading config file", zap.Error(err))
		}
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &Configs)
	if err != nil {
		if Logger == nil {
			os.Stderr.WriteString("Error parsing config file: " + err.Error() + "\n")
		} else {
			Logger.Error("Error parsing config file", zap.Error(err))
		}
		os.Exit(1)
	}
}
