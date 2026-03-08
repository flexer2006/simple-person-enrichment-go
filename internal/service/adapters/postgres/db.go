package postgres

import (
	"context"
	"fmt"

	"github.com/flexer2006/case-person-enrichment-go/internal/database/postgres"
	"github.com/flexer2006/case-person-enrichment-go/internal/logger"
	repoports "github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	"go.uber.org/zap"
)

type Adapter struct {
	db           postgres.Provider
	repositories repoports.Repositories
}

func NewPostgresAdapter(db postgres.Provider) *Adapter {
	return &Adapter{
		db:           db,
		repositories: NewRepositories(db),
	}
}

func (p *Adapter) Repositories() repoports.Repositories {
	return p.repositories
}

func (p *Adapter) DB() postgres.Provider {
	return p.db
}

func (p *Adapter) Close(ctx context.Context) {
	logger.Info(ctx, "closing PostgreSQL adapter")
	p.db.Close(ctx)
}

func (p *Adapter) Ping(ctx context.Context) error {
	logger.Debug(ctx, "pinging PostgreSQL database")
	err := p.db.Ping(ctx)
	if err != nil {
		logger.Error(ctx, "failed to ping PostgreSQL database", zap.Error(err))
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}

func (p *Adapter) DSN() string {
	return p.db.GetDSN()
}
