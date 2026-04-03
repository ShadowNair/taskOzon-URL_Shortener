package connections

import (
	"url-shortener/internal/config"
	"url-shortener/internal/connections/postgres"
	"database/sql"
)

type Config struct {
	PostgresSQL *sql.DB
}

func New(cfg *config.Config) (*Config, error) {
	postgresSQL, err := postgres.ConnectDB(cfg.DBConfig)
	if err != nil {
		return nil, err
	}
	return &Config{
		PostgresSQL: postgresSQL,
	}, nil
}

func (c *Config) CloseAll() {
	_ = c.PostgresSQL.Close()
}