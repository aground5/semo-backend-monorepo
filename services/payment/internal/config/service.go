package config

type ServiceConfig struct {
	Name                string         `yaml:"name"`
	Environment         string         `yaml:"environment"`
	Version             string         `yaml:"version"`
	ClientURL           string         `yaml:"client_url"`
	StripeSecretKey     string         `yaml:"stripe_secret_key"`
	StripeWebhookSecret string         `yaml:"stripe_webhook_secret"`
	EnableTestEndpoints bool           `yaml:"enable_test_endpoints"`
	Supabase            SupabaseConfig `yaml:"supabase"`
	Toss                TossConfig     `yaml:"toss"`
}

type SupabaseConfig struct {
	JWTSecret  string `yaml:"jwt_secret"`
	ProjectURL string `yaml:"project_url"`
	APIKey     string `yaml:"api_key"`
}

type TossConfig struct {
	SecretKey      string `yaml:"secret_key"`
	ClientKey      string `yaml:"client_key"`
	WebhookSecret  string `yaml:"webhook_secret"`
}
