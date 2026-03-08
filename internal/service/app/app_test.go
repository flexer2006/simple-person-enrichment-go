package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/app"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAPIAdapter struct {
	mock.Mock
}

func (m *mockAPIAdapter) Age() interface {
	GetAgeByName(ctx context.Context, name string) (int, float64, error)
} {
	args := m.Called()
	return args.Get(0).(interface {
		GetAgeByName(ctx context.Context, name string) (int, float64, error)
	})
}

func (m *mockAPIAdapter) Gender() interface {
	GetGenderByName(ctx context.Context, name string) (string, float64, error)
} {
	args := m.Called()
	return args.Get(0).(interface {
		GetGenderByName(ctx context.Context, name string) (string, float64, error)
	})
}

func (m *mockAPIAdapter) Nationality() interface {
	GetNationalityByName(ctx context.Context, name string) (string, float64, error)
} {
	args := m.Called()
	return args.Get(0).(interface {
		GetNationalityByName(ctx context.Context, name string) (string, float64, error)
	})
}

func (m *mockAPIAdapter) Person() interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
	CreatePerson(ctx context.Context, person *domain.Person) error
	UpdatePerson(ctx context.Context, person *domain.Person) error
	DeletePerson(ctx context.Context, id uuid.UUID) error
	EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error)
} {
	args := m.Called()
	return args.Get(0).(interface {
		GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error)
		GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error)
		CreatePerson(ctx context.Context, person *domain.Person) error
		UpdatePerson(ctx context.Context, person *domain.Person) error
		DeletePerson(ctx context.Context, id uuid.UUID) error
		EnrichPerson(ctx context.Context, id uuid.UUID) (*domain.Person, error)
	})
}

type mockPersonRepository struct {
	mock.Mock
}

func (m *mockPersonRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func (m *mockPersonRepository) GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error) {
	args := m.Called(ctx, filter, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Person), args.Int(1), args.Error(2)
}

func (m *mockPersonRepository) CreatePerson(ctx context.Context, person *domain.Person) error {
	args := m.Called(ctx, person)
	return args.Error(0)
}

func (m *mockPersonRepository) UpdatePerson(ctx context.Context, person *domain.Person) error {
	args := m.Called(ctx, person)
	return args.Error(0)
}

func (m *mockPersonRepository) DeletePerson(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockPersonRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

type mockAgeService struct {
	mock.Mock
}

func (m *mockAgeService) GetAgeByName(ctx context.Context, name string) (int, float64, error) {
	args := m.Called(ctx, name)
	return args.Int(0), args.Get(1).(float64), args.Error(2)
}

type mockGenderService struct {
	mock.Mock
}

func (m *mockGenderService) GetGenderByName(ctx context.Context, name string) (string, float64, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Get(1).(float64), args.Error(2)
}

type mockNationalityService struct {
	mock.Mock
}

func (m *mockNationalityService) GetNationalityByName(ctx context.Context, name string) (string, float64, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Get(1).(float64), args.Error(2)
}

func TestNewPersonService(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)

	service := app.NewPersonService(personRepo, apiAdapter)
	require.NotNil(t, service)
}

func TestPersonServiceGetByID(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	id := uuid.New()
	expectedPerson := &domain.Person{ID: id, Name: "John Doe"}

	personRepo.On("GetByID", mock.Anything, id).Return(expectedPerson, nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	person, err := service.GetByID(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, expectedPerson, person)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceGetByIDError(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	id := uuid.New()
	expectedError := domain.ErrPersonNotFound

	personRepo.On("GetByID", mock.Anything, id).Return(nil, expectedError)

	service := app.NewPersonService(personRepo, apiAdapter)
	person, err := service.GetByID(ctx, id)

	require.Error(t, err)
	assert.Nil(t, person)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceGetPersons(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	filter := map[string]any{"name": "John"}
	expectedPersons := []*domain.Person{
		{ID: uuid.New(), Name: "John Doe"},
		{ID: uuid.New(), Name: "John Smith"},
	}
	expectedCount := 2

	personRepo.On("GetPersons", mock.Anything, filter, 0, 10).Return(expectedPersons, expectedCount, nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	persons, count, err := service.GetPersons(ctx, filter, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, expectedPersons, persons)
	assert.Equal(t, expectedCount, count)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceCreatePerson(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	person := &domain.Person{Name: "John Doe"}

	personRepo.On("CreatePerson", mock.Anything, person).Return(nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	err := service.CreatePerson(ctx, person)

	require.NoError(t, err)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceUpdatePerson(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	person := &domain.Person{ID: uuid.New(), Name: "John Doe"}

	personRepo.On("UpdatePerson", mock.Anything, person).Return(nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	err := service.UpdatePerson(ctx, person)

	require.NoError(t, err)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceDeletePerson(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ctx := context.Background()
	id := uuid.New()

	personRepo.On("DeletePerson", mock.Anything, id).Return(nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	err := service.DeletePerson(ctx, id)

	require.NoError(t, err)
	personRepo.AssertExpectations(t)
}

func TestPersonServiceEnrichPerson(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ageService := new(mockAgeService)
	genderService := new(mockGenderService)
	nationalityService := new(mockNationalityService)

	ctx := context.Background()
	id := uuid.New()
	person := &domain.Person{ID: id, Name: "John Doe"}
	expectedAge := 30
	expectedGender := "male"
	expectedNationality := "US"
	genderProb := 0.95
	nationalityProb := 0.85

	enrichedPerson := &domain.Person{
		ID:                     id,
		Name:                   "John Doe",
		Age:                    &expectedAge,
		Gender:                 &expectedGender,
		GenderProbability:      &genderProb,
		Nationality:            &expectedNationality,
		NationalityProbability: &nationalityProb,
	}

	apiAdapter.On("Age").Return(ageService)
	apiAdapter.On("Gender").Return(genderService)
	apiAdapter.On("Nationality").Return(nationalityService)

	personRepo.On("GetByID", mock.Anything, id).Return(person, nil)
	ageService.On("GetAgeByName", mock.Anything, "John Doe").Return(expectedAge, 0.9, nil)
	genderService.On("GetGenderByName", mock.Anything, "John Doe").Return(expectedGender, genderProb, nil)
	nationalityService.On("GetNationalityByName", mock.Anything, "John Doe").Return(expectedNationality, nationalityProb, nil)
	personRepo.On("UpdatePerson", mock.Anything, mock.MatchedBy(func(p *domain.Person) bool {
		return p.ID == id &&
			p.Name == "John Doe" &&
			*p.Age == expectedAge &&
			*p.Gender == expectedGender &&
			*p.GenderProbability == genderProb &&
			*p.Nationality == expectedNationality &&
			*p.NationalityProbability == nationalityProb
	})).Return(nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	result, err := service.EnrichPerson(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, enrichedPerson.ID, result.ID)
	assert.Equal(t, enrichedPerson.Name, result.Name)
	assert.Equal(t, *enrichedPerson.Age, *result.Age)
	assert.Equal(t, *enrichedPerson.Gender, *result.Gender)
	assert.Equal(t, *enrichedPerson.GenderProbability, *result.GenderProbability)
	assert.Equal(t, *enrichedPerson.Nationality, *result.Nationality)
	assert.Equal(t, *enrichedPerson.NationalityProbability, *result.NationalityProbability)

	personRepo.AssertExpectations(t)
	ageService.AssertExpectations(t)
	genderService.AssertExpectations(t)
	nationalityService.AssertExpectations(t)
}

func TestPersonServiceEnrichPersonPartialFailure(t *testing.T) {
	personRepo := new(mockPersonRepository)
	apiAdapter := new(mockAPIAdapter)
	ageService := new(mockAgeService)
	genderService := new(mockGenderService)
	nationalityService := new(mockNationalityService)
	ctx := context.Background()
	id := uuid.New()
	person := &domain.Person{ID: id, Name: "John Doe"}
	expectedGender := "male"
	genderProb := 0.95

	apiAdapter.On("Age").Return(ageService)
	apiAdapter.On("Gender").Return(genderService)
	apiAdapter.On("Nationality").Return(nationalityService)

	personRepo.On("GetByID", mock.Anything, id).Return(person, nil)
	ageService.On("GetAgeByName", mock.Anything, "John Doe").Return(0, 0.0, errors.New("age service error"))
	genderService.On("GetGenderByName", mock.Anything, "John Doe").Return(expectedGender, genderProb, nil)
	nationalityService.On("GetNationalityByName", mock.Anything, "John Doe").Return("", 0.0, errors.New("nationality service error"))

	personRepo.On("UpdatePerson", mock.Anything, mock.MatchedBy(func(p *domain.Person) bool {
		return p.ID == id &&
			p.Name == "John Doe" &&
			p.Age == nil &&
			*p.Gender == expectedGender &&
			*p.GenderProbability == genderProb &&
			p.Nationality == nil &&
			p.NationalityProbability == nil
	})).Return(nil)

	service := app.NewPersonService(personRepo, apiAdapter)
	result, err := service.EnrichPerson(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, person.ID, result.ID)
	assert.Equal(t, person.Name, result.Name)
	assert.Nil(t, result.Age)
	assert.Equal(t, expectedGender, *result.Gender)
	assert.Equal(t, genderProb, *result.GenderProbability)
	assert.Nil(t, result.Nationality)
	assert.Nil(t, result.NationalityProbability)

	personRepo.AssertExpectations(t)
	ageService.AssertExpectations(t)
	genderService.AssertExpectations(t)
	nationalityService.AssertExpectations(t)
}
