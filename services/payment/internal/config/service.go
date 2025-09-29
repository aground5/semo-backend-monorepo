package config

type ServiceConfig struct {
	Name                string         `yaml:"name"`
	Environment         string         `yaml:"environment"`
	Version             string         `yaml:"version"`
	ClientURL           string         `yaml:"client_url"`
	ClientURLs          []string       `yaml:"client_urls"`
	StripeSecretKey     string         `yaml:"stripe_secret_key"`
	StripeWebhookSecret string         `yaml:"stripe_webhook_secret"`
	EnableTestEndpoints bool           `yaml:"enable_test_endpoints"`
	Supabase            SupabaseConfig `yaml:"supabase"`
	Toss                TossConfig     `yaml:"toss"`
}

func (s *ServiceConfig) AllowedClientOrigins() []string {
	if len(s.ClientURLs) > 0 {
		return append([]string(nil), s.ClientURLs...)
	}
	if s.ClientURL != "" {
		return []string{s.ClientURL}
	}
	return []string{}
}

func (s *ServiceConfig) PrimaryClientURL() string {
	if len(s.ClientURLs) > 0 {
		return s.ClientURLs[0]
	}
	return s.ClientURL
}

type SupabaseConfig struct {
	JWTSecret  string `yaml:"jwt_secret"`
	ProjectURL string `yaml:"project_url"`
	APIKey     string `yaml:"api_key"`
}

type TossConfig struct {
	SecretKey     string `yaml:"secret_key"`
	ClientKey     string `yaml:"client_key"`
	WebhookSecret string `yaml:"webhook_secret"`
	PlansFile     string `yaml:"plans_file"`
}
