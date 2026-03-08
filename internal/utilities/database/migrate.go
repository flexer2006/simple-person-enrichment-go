package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/golang-migrate/migrate/v4"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

var ErrMigrationPathNotSpecified = errors.New("migration path not specified")

type Provider interface {
	Up(ctx context.Context, dsn string) error
	Down(ctx context.Context, dsn string) error
	Version(ctx context.Context, dsn string) (uint, bool, error)
}

type MigrateConfig struct {
	Path string
}

type Migrator struct {
	cfg MigrateConfig
}

func NewMig(cfg MigrateConfig) *Migrator {
	return &Migrator{cfg: cfg}
}

func (m *Migrator) Up(ctx context.Context, dsn string) error {
	path, err := m.migrationPath()
	if err != nil {
		return err
	}

	migrator, err := migrate.New(path, dsn)
	if err != nil {
		utilities.Error(ctx, "failed to create migration instance", zap.Error(err), zap.String("path", path))
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.close(ctx, migrator)

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		utilities.Error(ctx, "failed to apply migrations", zap.Error(err))
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	utilities.Info(ctx, "database migrations applied")
	return nil
}

func (m *Migrator) Down(ctx context.Context, dsn string) error {
	path, err := m.migrationPath()
	if err != nil {
		return err
	}

	migrator, err := migrate.New(path, dsn)
	if err != nil {
		utilities.Error(ctx, "failed to create migration instance", zap.Error(err), zap.String("path", path))
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.close(ctx, migrator)

	if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		utilities.Error(ctx, "failed to rollback migrations", zap.Error(err))
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	utilities.Info(ctx, "database migrations rolled back")
	return nil
}

func (m *Migrator) Version(ctx context.Context, dsn string) (uint, bool, error) {
	path, err := m.migrationPath()
	if err != nil {
		return 0, false, err
	}

	migrator, err := migrate.New(path, dsn)
	if err != nil {
		utilities.Error(ctx, "failed to create migration instance", zap.Error(err), zap.String("path", path))
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.close(ctx, migrator)

	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		utilities.Error(ctx, "failed to get migration version", zap.Error(err))
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return version, dirty, nil
}

func (m *Migrator) migrationPath() (string, error) {
	if m.cfg.Path == "" {
		return "", ErrMigrationPathNotSpecified
	}
	return fmt.Sprintf("file://%s", m.cfg.Path), nil
}

func (m *Migrator) close(ctx context.Context, migrator *migrate.Migrate) {
	srcErr, dbErr := migrator.Close()
	if srcErr != nil {
		utilities.Error(ctx, "failed to close migration source", zap.Error(srcErr))
	}
	if dbErr != nil {
		utilities.Error(ctx, "failed to close migration database", zap.Error(dbErr))
	}
}
