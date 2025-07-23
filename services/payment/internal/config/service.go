package config

type ServiceConfig struct {
	Name                string `yaml:"name"`
	Environment         string `yaml:"environment"`
	Version             string `yaml:"version"`
	ClientURL           string `yaml:"client_url"`
	StripeSecretKey     string `yaml:"stripe_secret_key"`
	StripeWebhookSecret string `yaml:"stripe_webhook_secret"`
	EnableTestEndpoints bool   `yaml:"enable_test_endpoints"`
}