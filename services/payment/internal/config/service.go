package config

type ServiceConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Version     string `yaml:"version"`
}