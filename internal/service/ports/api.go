package ports

import "context"

type AgeAPI interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
}

type GenderAPI interface {
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
}

type NationalityAPI interface {
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
}

type API interface {
	Age() AgeAPI
	Gender() GenderAPI
	Nationality() NationalityAPI
}
