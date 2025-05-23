package configs

type RedisConfig struct {
	Addresses []string `yaml:"addresses"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Database  int      `yaml:"database"`
	Tls       bool     `yaml:"tls"`
}
