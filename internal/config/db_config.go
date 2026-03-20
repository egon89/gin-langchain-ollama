package config

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB(url string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), url)
}
