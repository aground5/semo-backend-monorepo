package configs

type LogsConfig struct {
	LogPath    string `yaml:"log_path"`
	LogLevel   string `yaml:"log_level"`
	StdoutOnly bool   `yaml:"stdout_only"`
}
