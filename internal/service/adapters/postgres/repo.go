package postgres

import (
	"context"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities/database"

	"github.com/google/uuid"
)

var _ ports.Repositories = (*Repositories)(nil)

type Repositories struct {
	personRepo *Repository
}

func NewRepositories(db database.PostgresProvider) *Repositories {
	return &Repositories{
		personRepo: NewRepository(db),
	}
}

func (r *Repositories) GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
	return r.personRepo.GetByID(ctx, id)
}

func (r *Repositories) GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error) {
	return r.personRepo.GetPersons(ctx, filter, offset, limit)
}

func (r *Repositories) CreatePerson(ctx context.Context, person *domain.Person) error {
	return r.personRepo.CreatePerson(ctx, person)
}

func (r *Repositories) UpdatePerson(ctx context.Context, person *domain.Person) error {
	return r.personRepo.UpdatePerson(ctx, person)
}

func (r *Repositories) DeletePerson(ctx context.Context, id uuid.UUID) error {
	return r.personRepo.DeletePerson(ctx, id)
}

func (r *Repositories) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	return r.personRepo.ExistsByID(ctx, id)
}
