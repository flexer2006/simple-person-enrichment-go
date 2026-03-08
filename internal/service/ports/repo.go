package ports

import (
	"context"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"

	"github.com/google/uuid"
)

type Repositories interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
}
