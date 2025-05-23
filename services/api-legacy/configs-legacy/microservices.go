package configs

type MicroserviceConfig struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	HttpPort string `yaml:"http_port"`
	GrpcPort string `yaml:"grpc_port"`
}
