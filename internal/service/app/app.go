package app

import (
	"context"
	"fmt"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/enrichment"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/postgres"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/server"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	dbpkg "github.com/flexer2006/case-person-enrichment-go/internal/utilities/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PersonService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
	EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error)
}

type Application struct {
	config        *domain.Config
	db            dbpkg.PostgresProvider
	httpServer    *server.Server
	personService PersonService
}

func NewApplication(ctx context.Context, config *domain.Config, database dbpkg.PostgresProvider, apiAdapter ports.API) (*Application, error) {
	utilities.Info(ctx, "initializing application")
	pgAdapter := postgres.NewPostgresAdapter(database)
	if apiAdapter == nil {
		apiAdapter = enrichment.NewDefaultEnrichment()
	}
	app := &Application{
		config:        config,
		db:            database,
		httpServer:    server.New(*config, apiAdapter, pgAdapter.Repositories()),
		personService: NewPersonService(pgAdapter.Repositories(), apiAdapter),
	}
	utilities.Info(ctx, "application initialized successfully")
	return app, nil
}

func (a *Application) Start(ctx context.Context) error {
	utilities.Info(ctx, "starting application")
	if err := a.httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

func (a *Application) Stop(ctx context.Context) error {
	utilities.Info(ctx, "stopping application")
	shutdownTimeout, err := time.ParseDuration(a.config.Graceful.ShutdownTimeout)
	if err != nil {
		shutdownTimeout = 5 * time.Second
		utilities.Warn(ctx, "invalid graceful shutdown timeout, using default",
			zap.String("default", shutdownTimeout.String()))
	}
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()
	if err := a.httpServer.Stop(ctx); err != nil {
		utilities.Error(ctx, "error stopping HTTP server", zap.Error(err))
	}
	if a.db != nil {
		a.db.Close(ctx)
	}
	utilities.Info(ctx, "application stopped")
	return nil
}

func (a *Application) PersonService() PersonService {
	return a.personService
}

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
	person, err := s.repositories.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get person: %w", err)
	}
	if person.Age == nil {
		if age, _, err := s.apiAdapter.Age().GetAgeByName(ctx, person.Name); err == nil {
			person.Age = &age
		} else {
			utilities.Warn(ctx, "failed to enrich with age data", zap.Error(err))
		}
	}
	if person.Gender == nil {
		if gender, probability, err := s.apiAdapter.Gender().GetGenderByName(ctx, person.Name); err == nil {
			person.Gender, person.GenderProbability = &gender, &probability
		} else {
			utilities.Warn(ctx, "failed to enrich with gender data", zap.Error(err))
		}
	}
	if person.Nationality == nil {
		nationality, probability, err := s.apiAdapter.Nationality().GetNationalityByName(ctx, person.Name)
		if err == nil {
			person.Nationality, person.NationalityProbability = &nationality, &probability
		} else {
			utilities.Warn(ctx, "failed to enrich with nationality data", zap.Error(err))
		}
	}
	if err = s.repositories.UpdatePerson(ctx, person); err != nil {
		return nil, fmt.Errorf("failed to save enriched person data: %w", err)
	}
	return person, nil
}
