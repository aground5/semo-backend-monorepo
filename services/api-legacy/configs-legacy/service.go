package configs

type ServiceConfig struct {
	HttpPort    string `yaml:"http_port"`
	GrpcPort    string `yaml:"grpc_port"`
	ServiceName string `yaml:"service_name"`
	BaseURL     string `yaml:"base_url"`
}
