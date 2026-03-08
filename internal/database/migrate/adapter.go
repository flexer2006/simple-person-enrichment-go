package migrate

import (
	"context"
)

// Provider определяет интерфейс для провайдера миграций.
type Provider interface {
	// Up выполняет все доступные миграции.
	Up(ctx context.Context, dsn string) error
	// Down откатывает все миграции.
	Down(ctx context.Context, dsn string) error
	// Version возвращает текущую версию миграции и статус "грязный".
	Version(ctx context.Context, dsn string) (uint, bool, error)
}

// Adapter адаптирует Migrator к интерфейсу Provider.
type Adapter struct {
	migrator *Migrator
	config   Config
}

// NewAdapter создает новый адаптер для мигратора.
func NewAdapter(config Config) *Adapter {
	return &Adapter{
		migrator: NewMigrator(),
		config:   config,
	}
}

// Up реализует Provider.Up.
func (a *Adapter) Up(ctx context.Context, dsn string) error {
	return a.migrator.Up(ctx, dsn, a.config)
}

// Down реализует Provider.Down.
func (a *Adapter) Down(ctx context.Context, dsn string) error {
	return a.migrator.Down(ctx, dsn, a.config)
}

// Version реализует Provider.Version.
func (a *Adapter) Version(ctx context.Context, dsn string) (uint, bool, error) {
	return a.migrator.Version(ctx, dsn, a.config)
}
