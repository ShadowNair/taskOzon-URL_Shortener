package config

import (
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBConfig  *PostgresConfig
	AppConfig *AppConfig
}

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DB       string
}

type AppConfig struct {
	Host        string
	Port        string
	StorageType string
}

func GetConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		DBConfig:  GetPostgresConfig(),
		AppConfig: GetAppConfig(),
	}
}

func GetPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		User:     envOrDefault("POSTGRES_USER", "shortener"),
		Password: envOrDefault("POSTGRES_PASSWORD", "shortener"),
		Host:     envOrDefault("POSTGRES_HOST", "localhost"),
		Port:     normalizePort(envOrDefault("POSTGRES_PORT", "5432")),
		DB:       envOrDefault("POSTGRES_DB", "shortener"),
	}
}

func GetAppConfig() *AppConfig {
	return &AppConfig{
		Host:        envOrDefault("APP_HOST", "0.0.0.0"),
		Port:        normalizePort(envOrDefault("APP_PORT", "8080")),
		StorageType: strings.ToLower(envOrDefault("STORAGE_TYPE", "memory")),
	}
}

func (c *AppConfig) Addr() string {
	if c == nil {
		return ":8080"
	}
	if c.Host == "" || c.Host == "0.0.0.0" {
		return ":" + c.Port
	}
	return net.JoinHostPort(c.Host, c.Port)
}

func envOrDefault(name, def string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return def
}

func normalizePort(port string) string {
	return strings.TrimPrefix(strings.TrimSpace(port), ":")
}