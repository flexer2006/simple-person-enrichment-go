package enrichment

import (
	api "github.com/flexer2006/case-person-enrichment-go/internal/service/adapters/enrichment/services"
	apiports "github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
)

var _ apiports.API = (*Enrichment)(nil)

type Enrichment struct {
	impl *api.API
}

func NewEnrichment(apiImpl *api.API) *Enrichment {
	return &Enrichment{impl: apiImpl}
}

func NewDefaultEnrichment() *Enrichment {
	return &Enrichment{impl: api.NewDefaultAPI()}
}

func (e *Enrichment) Age() apiports.AgeAPI {
	return e.impl.Age()
}

func (e *Enrichment) Gender() apiports.GenderAPI {
	return e.impl.Gender()
}

func (e *Enrichment) Nationality() apiports.NationalityAPI {
	return e.impl.Nationality()
}

// Person enrichment was previously part of the bundled API
// interface, but the service layer now owns person-related logic.
// We keep this adapter around only for the three external clients.
