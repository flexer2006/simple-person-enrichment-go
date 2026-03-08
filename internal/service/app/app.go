package app

import (
	"context"
	"fmt"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/database/migrate"
	pgadapter "github.com/flexer2006/case-person-enrichment-go/internal/database/postgres"
	"github.com/flexer2006/case-person-enrichment-go/internal/logger"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/enrichment"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/postgres"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/server"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Application struct {
	config        *domain.Config
	db            *pgadapter.Database
	pgAdapter     *postgres.Adapter
	apiAdapter    ports.API
	repositories  ports.Repositories
	httpServer    *server.Server
	personService *personServiceImpl
}

func NewApplication(ctx context.Context, config *domain.Config) (*Application, error) {
	logger.Info(ctx, "initializing application")

	dbConfig := pgadapter.Config{
		Host:     config.Postgres.Host,
		Port:     config.Postgres.Port,
		User:     config.Postgres.User,
		Password: config.Postgres.Password,
		Database: config.Postgres.Database,
		SSLMode:  config.Postgres.SSLMode,
		MinConns: config.Postgres.PoolMinConns,
		MaxConns: config.Postgres.PoolMaxConns,
	}

	database, err := pgadapter.New(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if config.Migrations.Path != "" {
		migrateConfig := migrate.Config{
			Path: config.Migrations.Path,
		}
		migrator := migrate.NewAdapter(migrateConfig)
		if err := migrator.Up(ctx, database.GetDSN()); err != nil {
			return nil, fmt.Errorf("failed to apply migrations: %w", err)
		}
	}

	pgAdapter := postgres.NewPostgresAdapter(database)

	apiAdapter := enrichment.NewDefaultEnrichment()

	personSvc := NewPersonService(pgAdapter.Repositories(), apiAdapter)

	httpServer := server.New(*config, apiAdapter, pgAdapter.Repositories())

	app := &Application{
		config:        config,
		db:            database,
		pgAdapter:     pgAdapter,
		apiAdapter:    apiAdapter,
		repositories:  pgAdapter.Repositories(),
		httpServer:    httpServer,
		personService: personSvc,
	}

	logger.Info(ctx, "application initialized successfully")
	return app, nil
}

func (a *Application) Start(ctx context.Context) error {
	logger.Info(ctx, "starting application")

	if err := a.httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

func (a *Application) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping application")

	shutdownTimeout, err := time.ParseDuration(a.config.Graceful.ShutdownTimeout)
	if err != nil {
		shutdownTimeout = 5 * time.Second
		logger.Warn(ctx, "invalid graceful shutdown timeout, using default",
			zap.String("default", shutdownTimeout.String()))
	}

	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := a.httpServer.Stop(ctx); err != nil {
		logger.Error(ctx, "error stopping HTTP server", zap.Error(err))
	}

	a.pgAdapter.Close(ctx)

	logger.Info(ctx, "application stopped")
	return nil
}

// Repositories returns the application's storage interface.
func (a *Application) Repositories() ports.Repositories {
	return a.repositories
}

// API returns the external API adapter.
func (a *Application) API() ports.API {
	return a.apiAdapter
}

func (a *Application) PersonService() *personServiceImpl {
	return a.personService
}

// personServiceImpl handles domain-specific operations using the
// storage and API adapters. The flattened Repositories interface makes it
// easy to call storage methods directly without the previous nested
// Person() accessor.
func NewPersonService(repositories ports.Repositories, apiAdapter ports.API) *personServiceImpl {
	return &personServiceImpl{
		repositories: repositories,
		apiAdapter:   apiAdapter,
	}
}

type personServiceImpl struct {
	repositories ports.Repositories
	apiAdapter   ports.API
}

func (s *personServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
	person, err := s.repositories.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get person by ID: %w", err)
	}
	return person, nil
}

func (s *personServiceImpl) GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error) {
	persons, count, err := s.repositories.GetPersons(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get persons: %w", err)
	}
	return persons, count, nil
}

func (s *personServiceImpl) CreatePerson(ctx context.Context, person *domain.Person) error {
	if err := s.repositories.CreatePerson(ctx, person); err != nil {
		return fmt.Errorf("failed to create person: %w", err)
	}
	return nil
}

func (s *personServiceImpl) UpdatePerson(ctx context.Context, person *domain.Person) error {
	if err := s.repositories.UpdatePerson(ctx, person); err != nil {
		return fmt.Errorf("failed to update person: %w", err)
	}
	return nil
}

func (s *personServiceImpl) DeletePerson(ctx context.Context, id uuid.UUID) error {
	if err := s.repositories.DeletePerson(ctx, id); err != nil {
		return fmt.Errorf("failed to delete person: %w", err)
	}
	return nil
}

func (s *personServiceImpl) EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
	logger.Debug(ctx, "enriching person data", zap.String("id", id.String()))

	person, err := s.repositories.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get person: %w", err)
	}

	if person.Age == nil {
		age, _, err := s.apiAdapter.Age().GetAgeByName(ctx, person.Name)
		if err == nil {
			person.Age = &age
		} else {
			logger.Warn(ctx, "failed to enrich with age data", zap.Error(err))
		}
	}

	if person.Gender == nil {
		gender, probability, err := s.apiAdapter.Gender().GetGenderByName(ctx, person.Name)
		if err == nil {
			person.Gender = &gender
			person.GenderProbability = &probability
		} else {
			logger.Warn(ctx, "failed to enrich with gender data", zap.Error(err))
		}
	}

	if person.Nationality == nil {
		nationality, probability, err := s.apiAdapter.Nationality().GetNationalityByName(ctx, person.Name)
		if err == nil {
			person.Nationality = &nationality
			person.NationalityProbability = &probability
		} else {
			logger.Warn(ctx, "failed to enrich with nationality data", zap.Error(err))
		}
	}

	err = s.repositories.UpdatePerson(ctx, person)
	if err != nil {
		return nil, fmt.Errorf("failed to save enriched person data: %w", err)
	}

	return person, nil
}
