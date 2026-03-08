package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Postgres        PostgresConfig
	Migrate         MigrateConfig
	ApplyMigrations bool
}

type Database struct {
	provider PostgresProvider
	migrator Provider
}

func New(ctx context.Context, cfg Config) (*Database, error) {
	postgresDB, err := NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}
	db := &Database{
		provider: postgresDB,
		migrator: NewMig(cfg.Migrate),
	}
	if cfg.ApplyMigrations {
		if err := db.ApplyMigrations(ctx); err != nil {
			db.Close(ctx)
			return nil, err
		}
	}
	return db, nil
}

func NewWithDSN(ctx context.Context, dsn string, minConn, maxConn int, migrationsPath string, applyMigrations bool) (*Database, error) {
	postgresDB, err := NewPostgresWithDSN(ctx, dsn, minConn, maxConn)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}
	db := &Database{
		provider: postgresDB,
		migrator: NewMig(MigrateConfig{Path: migrationsPath}),
	}
	if applyMigrations {
		if err := db.ApplyMigrations(ctx); err != nil {
			db.Close(ctx)
			return nil, err
		}
	}
	return db, nil
}

func (d *Database) Pool() *pgxpool.Pool {
	return d.provider.Pool()
}

func (d *Database) Close(ctx context.Context) {
	d.provider.Close(ctx)
}

func (d *Database) ApplyMigrations(ctx context.Context) error {
	if err := d.migrator.Up(ctx, d.provider.GetDSN()); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func (d *Database) RollbackMigrations(ctx context.Context) error {
	if err := d.migrator.Down(ctx, d.provider.GetDSN()); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

func (d *Database) Provider() PostgresProvider {
	return d.provider
}

func (d *Database) GetMigrationVersion(ctx context.Context) (uint, bool, error) {
	version, dirty, err := d.migrator.Version(ctx, d.provider.GetDSN())
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

func (d *Database) Ping(ctx context.Context) error {
	if err := d.provider.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}
