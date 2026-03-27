package config

import (
	"os"
)

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	ServerPort     string
	GoogleClientID string
	JWTSecret      string
	// Storage
	StorageProvider  string
	S3Bucket         string
	S3Region         string
	S3AccessKey      string
	S3SecretKey      string
	S3Endpoint       string
	LocalStoragePath string
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "fitreg"),
		ServerPort:     getEnvMulti([]string{"PORT", "SERVER_PORT"}, "8080"),
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		StorageProvider:  getEnv("STORAGE_PROVIDER", "local"),
		S3Bucket:         getEnv("S3_BUCKET", ""),
		S3Region:         getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:      getEnv("AWS_ACCESS_KEY_ID", ""),
		S3SecretKey:      getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3Endpoint:       getEnv("S3_ENDPOINT", ""),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "./uploads"),
	}
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?parseTime=true&loc=UTC"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvMulti(keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return fallback
}
