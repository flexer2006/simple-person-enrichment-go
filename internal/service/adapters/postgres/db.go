package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flexer2006/pes-api/internal/service/domain"
	"github.com/flexer2006/pes-api/internal/service/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresConfig struct {
	Host, User, Password, Database, SSLMode string //nolint:gosec
	Port, MinConns, MaxConns                int
}

type Database struct {
	pool *pgxpool.Pool
}

func NewDatabase(ctx context.Context, cfg PostgresConfig) (*Database, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.User == "" || cfg.Database == "" {
		err := domain.ErrInvalidConfiguration
		logger.Error(ctx, "invalid database configuration", zap.Error(err))
		return nil, err
	}
	logger.Info(ctx, "connecting to postgres database")
	poolCfg, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode))
	if err != nil {
		logger.Error(ctx, "parse config failed", zap.Error(err))
		return nil, fmt.Errorf("setup db: parse config: %w", err)
	}
	poolCfg.MinConns, poolCfg.MaxConns, poolCfg.ConnConfig.ConnectTimeout, poolCfg.HealthCheckPeriod = clamp32(cfg.MinConns), clamp32(cfg.MaxConns), 5*time.Second, 1*time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Error(ctx, "create pool failed", zap.Error(err))
		return nil, fmt.Errorf("setup db: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		logger.Error(ctx, "ping failed", zap.Error(err))
		return nil, fmt.Errorf("setup db: ping: %w", err)
	}
	logger.Info(ctx, "connected to postgres database")
	return new(Database{pool: pool}), nil
}

func (d *Database) Pool() *pgxpool.Pool { return d.pool }

func (d *Database) Close(ctx context.Context) {
	if d.pool != nil {
		logger.Info(ctx, "closing postgres database connection")
		d.pool.Close()
	}
}

func (d *Database) Ping(ctx context.Context) error {
	if d.pool == nil {
		return fmt.Errorf("ping: pool nil")
	}
	return d.pool.Ping(ctx)
}

func RunMigration(ctx context.Context, path string, cfg PostgresConfig) error {
	mig, err := migrate.New("file://"+path, fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode))
	if err != nil {
		logger.Error(ctx, "failed to create migration instance", zap.Error(err), zap.String("path", path))
		return fmt.Errorf("migrations: %w", err)
	}
	srcErr, dbErr := mig.Close()
	if srcErr != nil {
		logger.Error(ctx, "failed to close migration source", zap.Error(srcErr))
	}
	if dbErr != nil {
		logger.Error(ctx, "failed to close migration database", zap.Error(dbErr))
	}
	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error(ctx, "migration apply failed", zap.Error(err))
		return fmt.Errorf("migrations: %w", err)
	}
	logger.Info(ctx, "database migrations applied")
	return nil
}
