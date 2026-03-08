package database

import (
	"context"
	"fmt"

	"github.com/flexer2006/case-person-enrichment-go/internal/database/migrate"
	"github.com/flexer2006/case-person-enrichment-go/internal/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Postgres        postgres.Config
	Migrate         migrate.Config
	ApplyMigrations bool
}

type Database struct {
	provider postgres.Provider
	migrator migrate.Provider
}

func New(ctx context.Context, cfg Config) (*Database, error) {
	postgresDB, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	migrator := migrate.NewAdapter(cfg.Migrate)

	database := &Database{
		provider: postgresDB,
		migrator: migrator,
	}

	if cfg.ApplyMigrations {
		if err := database.ApplyMigrations(ctx); err != nil {
			database.Close(ctx)
			return nil, err
		}
	}

	return database, nil
}

func NewWithDSN(ctx context.Context, dsn string, minConn, maxConn int, migrationsPath string, applyMigrations bool) (*Database, error) {
	postgresDB, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	migrateConfig := migrate.Config{Path: migrationsPath}
	migrator := migrate.NewAdapter(migrateConfig)

	database := &Database{
		provider: postgresDB,
		migrator: migrator,
	}

	if applyMigrations && migrationsPath != "" {
		if err := database.ApplyMigrations(ctx); err != nil {
			database.Close(ctx)
			return nil, err
		}
	}

	return database, nil
}

func (d *Database) Pool() *pgxpool.Pool {
	return d.provider.Pool()
}

func (d *Database) Close(ctx context.Context) {
	d.provider.Close(ctx)
}

func (d *Database) ApplyMigrations(ctx context.Context) error {
	dsn := d.provider.GetDSN()
	if err := d.migrator.Up(ctx, dsn); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func (d *Database) RollbackMigrations(ctx context.Context) error {
	dsn := d.provider.GetDSN()
	if err := d.migrator.Down(ctx, dsn); err != nil {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

func (d *Database) GetMigrationVersion(ctx context.Context) (uint, bool, error) {
	dsn := d.provider.GetDSN()
	version, dirty, err := d.migrator.Version(ctx, dsn)
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
