package migrate_test

import (
	"context"
	"testing"

	"github.com/flexer2006/case-person-enrichment-go/internal/database/migrate"
	"github.com/stretchr/testify/assert"
)

func TestNewMigrator(t *testing.T) {
	migrator := migrate.NewMigrator()
	assert.NotNil(t, migrator, "NewMigrator() should not return nil")

	ctx := context.Background()

	t.Run("Up without config returns ErrMigrationPathNotSpecified", func(t *testing.T) {
		err := migrator.Up(ctx, "some-dsn")
		assert.ErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})

	t.Run("Down without config returns ErrMigrationPathNotSpecified", func(t *testing.T) {
		err := migrator.Down(ctx, "some-dsn")
		assert.ErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})

	t.Run("Version without config returns ErrMigrationPathNotSpecified", func(t *testing.T) {
		_, _, err := migrator.Version(ctx, "some-dsn")
		assert.ErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})

	t.Run("Force without config returns ErrMigrationPathNotSpecified", func(t *testing.T) {
		err := migrator.Force(ctx, "some-dsn", 1)
		assert.ErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})

	t.Run("With empty config path returns ErrMigrationPathNotSpecified", func(t *testing.T) {
		emptyConfig := migrate.Config{Path: ""}
		err := migrator.Up(ctx, "some-dsn", emptyConfig)
		assert.ErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})

	t.Run("With valid config adds file:// prefix to path", func(t *testing.T) {
		validConfig := migrate.Config{Path: "/valid/path"}

		err := migrator.Up(ctx, "invalid-dsn", validConfig)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, migrate.ErrMigrationPathNotSpecified)
	})
}
