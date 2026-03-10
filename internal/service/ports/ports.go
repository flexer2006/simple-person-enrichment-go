package ports

import (
	"context"

	"github.com/flexer2006/pes-api/internal/service/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories interface {
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
}

type API interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
}

type Database interface {
	Pool() *pgxpool.Pool
	Close(ctx context.Context)
	Ping(ctx context.Context) error
}
