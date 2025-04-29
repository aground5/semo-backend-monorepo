package configs

type MongoDBConfig struct {
	Uri      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
