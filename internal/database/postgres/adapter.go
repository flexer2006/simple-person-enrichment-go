package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider interface {
	Pool() *pgxpool.Pool
	Close(ctx context.Context)
	Ping(ctx context.Context) error
	GetDSN() string
}
