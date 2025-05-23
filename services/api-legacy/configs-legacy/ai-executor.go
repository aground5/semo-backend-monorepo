package configs

type AiExecutor struct {
	Path string `yaml:"path"`

	// OpenAI API configuration
	OpenAIAPIKey   string `yaml:"openai_api_key"`
	OpenAIModel    string `yaml:"openai_model"`
	ContextSize    string `yaml:"context_size"`
	OpenAIEndpoint string `yaml:"openai_endpoint"`

	// Anthropic API configuration
	AnthropicAPIKey string `yaml:"anthropic_api_key"`
	AnthropicModel  string `yaml:"anthropic_model"`

	// Google Gemini API configuration
	GoogleAPIKey string `yaml:"google_generative_ai_api_key"`
	GoogleModel  string `yaml:"google_model"`

	// Grok API configuration
	GrokAPIKey   string `yaml:"grok_api_key"`
	GrokModel    string `yaml:"grok_model"`
	GrokEndpoint string `yaml:"grok_endpoint"`

	// Langfuse telemetry configuration
	LangfusePublicKey string `yaml:"langfuse_public_key"`
	LangfuseSecretKey string `yaml:"langfuse_secret_key"`
	LangfuseBaseURL   string `yaml:"langfuse_baseurl"`
	EnableTelemetry   string `yaml:"enable_telemetry"`

	// Logging configuration
	LogLevel          string `yaml:"log_level"`
	EnableFileLogging string `yaml:"enable_file_logging"`
	LogDir            string `yaml:"log_dir"`
	LogMaxSize        string `yaml:"log_max_size"`
	LogMaxFiles       string `yaml:"log_max_files"`
}
