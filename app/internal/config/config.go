package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	Database DatabaseConfig
	JWT      JWTConfig
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Port: os.Getenv("PORT"),
		Database: DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			PrivateKeyPath: getenv("JWT_PRIVATE_KEY_PATH", "../keys/private.pem"),
			PublicKeyPath:  getenv("JWT_PUBLIC_KEY_PATH", "../keys/public.pem"),
		},
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
