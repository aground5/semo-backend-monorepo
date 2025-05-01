package configs

type Secrets struct {
	GrpcToken       string `yaml:"grpc_token"`
	SessionSecret   string `yaml:"session_secret"`
	EcdsaPrivateKey string `yaml:"ecdsa_private_key"`
	EcdsaPublicKey  string `yaml:"ecdsa_public_key"`
}
