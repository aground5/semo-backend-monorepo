package configs

type Secrets struct {
	GrpcToken     string `yaml:"grpc_token"`
	SessionSecret string `yaml:"session_secret"`
	CursorSecret  string `yaml:"cursor_secret"`
}
