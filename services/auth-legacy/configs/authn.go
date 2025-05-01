package configs

type AuthnConfig struct {
	SessionExpireMin      int `yaml:"session_expire_min"`
	AccessJwtExpireMin    int `yaml:"access_jwt_expire_min"`
	RefreshTokenExpireMin int `yaml:"refresh_token_expire_min"`
}
