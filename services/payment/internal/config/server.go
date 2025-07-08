package config

type ServerConfig struct {
	HTTP HTTPConfig `yaml:"http"`
	GRPC GRPCConfig `yaml:"grpc"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type GRPCConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}