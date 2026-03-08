package api

import "github.com/flexer2006/case-person-enrichment-go/internal/service/ports"

type ageSvc = ports.AgeAPI
type genderSvc = ports.GenderAPI
type nationalitySvc = ports.NationalityAPI

type API struct {
	ageSvc         ageSvc
	genderSvc      genderSvc
	nationalitySvc nationalitySvc
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

func (a *API) Age() ports.AgeAPI {
	return a.ageSvc
}

func (a *API) Gender() ports.GenderAPI {
	return a.genderSvc
}

func (a *API) Nationality() ports.NationalityAPI {
	return a.nationalitySvc
}
