package postgres

import (
	"context"
	"fmt"

	repoports "github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	logger "github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities/database"
	"go.uber.org/zap"
)

type Adapter struct {
	db           database.PostgresProvider
	repositories repoports.Repositories
}

func NewPostgresAdapter(db database.PostgresProvider) *Adapter {
	return &Adapter{
		db:           db,
		repositories: NewRepositories(db),
	}
}

func (p *Adapter) Repositories() repoports.Repositories {
	return p.repositories
}

func (p *Adapter) DB() database.PostgresProvider {
	return p.db
}

func (p *Adapter) Close(ctx context.Context) {
	logger.Info(ctx, "closing PostgreSQL adapter")
	p.db.Close(ctx)
}

func (p *Adapter) Ping(ctx context.Context) error {
	logger.Debug(ctx, "pinging PostgreSQL database")
	if err := p.db.Ping(ctx); err != nil {
		logger.Error(ctx, "failed to ping PostgreSQL database", zap.Error(err))
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}

func (p *Adapter) DSN() string {
	return p.db.GetDSN()
}
