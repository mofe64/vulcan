package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, _ := pgxpool.ParseConfig(url)
	cfg.MaxConns = 20
	return pgxpool.NewWithConfig(ctx, cfg)
}
