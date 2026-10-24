package config

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

func LoadMinIOConfig() *MinIOConfig {
	return &MinIOConfig{
		Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin_pass"),
		UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		Region:    getEnv("MINIO_REGION", "us-east-1"),
	}
}