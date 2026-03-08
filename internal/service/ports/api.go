package ports

import (
	"context"
)

// NOTE: the PersonAPI interface was removed.  Business logic
// now lives in a service layer (see app.PersonService) instead of an
// adapter.  Handlers should call the service directly.
// The remaining API interfaces describe external enrichment
// providers only.

// AgeAPI wraps the external age prediction service.
type AgeAPI interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
}

// GenderAPI wraps the external gender prediction service.
type GenderAPI interface {
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
}

// NationalityAPI wraps the external nationality prediction service.
type NationalityAPI interface {
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
}

// API groups all external enrichment adapter interfaces used by
// the application.  there is no longer a Person() sub‑interface.
//
// In the future we might remove this type entirely and wire clients
// directly into the service constructor.
type API interface {
	Age() AgeAPI
	Gender() GenderAPI
	Nationality() NationalityAPI
}
