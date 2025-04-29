package configs

type OpenaiConfig struct {
	ApiKey       string `yaml:"api_key"`
	ProjectID    string `yaml:"project_id"`
	XaiApiKey    string `yaml:"xai_api_key"`
	GeminiApiKey string `yaml:"gemini_api_key"`
}
