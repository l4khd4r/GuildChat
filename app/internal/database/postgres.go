package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// dsnBody is the DSN without its scheme, so that callers needing a different
// one (the migrate driver wants pgx5://) can prefix their own.
func (c Config) dsnBody() string {
	return fmt.Sprintf("%s:%s@%s:%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

func (c Config) DSN() string {
	return "postgres://" + c.dsnBody()
}

func NewPostgresPool(cfg Config) (*pgxpool.Pool, error) {
	dsn := cfg.DSN()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}
