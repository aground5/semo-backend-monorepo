package configs

type S3Config struct {
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
}
