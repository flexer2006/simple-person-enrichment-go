package api

import (
	"context"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	portsapi "github.com/flexer2006/case-person-enrichment-go/internal/service/ports"

	"github.com/google/uuid"
)

var _ portsapi.API = (*API)(nil)

type ageSvc interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
}

type genderSvc interface {
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
}

type nationalitySvc interface {
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
}

type personSvc interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
	EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error)
}

type API struct {
	ageSvc         ageSvc
	genderSvc      genderSvc
	nationalitySvc nationalitySvc
	personSvc      personSvc
}

func NewAPI(ageSvc ageSvc, genderSvc genderSvc, nationalitySvc nationalitySvc) *API {
	return &API{
		ageSvc:         ageSvc,
		genderSvc:      genderSvc,
		nationalitySvc: nationalitySvc,
	}
}

func NewDefaultAPI() *API {
	return &API{
		ageSvc:         NewAgeAPIClient(nil),
		genderSvc:      NewGenderAPIClient(nil),
		nationalitySvc: NewNationalityAPIClient(nil),
	}
}

func (a *API) Age() interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
} {
	return a.ageSvc
}

func (a *API) Gender() interface {
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
} {
	return a.genderSvc
}

func (a *API) Nationality() interface {
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
} {
	return a.nationalitySvc
}

func (a *API) Person() interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
	EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error)
} {
	return a.personSvc
}
