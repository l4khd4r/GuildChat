package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	Database DatabaseConfig
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
	return Config {
		Port : os.Getenv("PORT"),
		Database: DatabaseConfig{
			Host: os.Getenv("DB_HOST"),
			Port: os.Getenv("DB_PORT"),
			User: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name: os.Getenv("DB_NAME"),
			SSLMode: os.Getenv("DB_SSLMODE"),
		},
	}
}
