package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	logger "github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type Repository struct {
	db database.PostgresProvider
}

func NewRepository(db database.PostgresProvider) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetByID(ctx context.Context, personID uuid.UUID) (*domain.Person, error) {
	logger.Debug(ctx, "getting person by ID", zap.String("id", personID.String()))
	query := `
        SELECT id, name, surname, patronymic, age, gender, gender_probability, 
               nationality, nationality_probability, created_at, updated_at
        FROM persons
        WHERE id = $1
    `
	var person domain.Person
	var patronymic, nationality, gender sql.NullString
	var age sql.NullInt32
	var genderProb, nationalityProb sql.NullFloat64
	row := r.db.Pool().QueryRow(ctx, query, personID)
	err := row.Scan(
		&person.ID,
		&person.Name,
		&person.Surname,
		&patronymic,
		&age,
		&gender,
		&genderProb,
		&nationality,
		&nationalityProb,
		&person.CreatedAt,
		&person.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Debug(ctx, "person not found", zap.String("id", personID.String()))
			return nil, fmt.Errorf("%w: id %s", domain.ErrPersonNotFound, personID)
		}
		logger.Error(ctx, "failed to get person by ID", zap.Error(err))
		return nil, fmt.Errorf("failed to get person: %w", err)
	}
	if patronymic.Valid {
		person.Patronymic = &patronymic.String
	}
	if age.Valid {
		ageVal := int(age.Int32)
		person.Age = &ageVal
	}
	if gender.Valid {
		person.Gender = &gender.String
	}
	if genderProb.Valid {
		person.GenderProbability = &genderProb.Float64
	}
	if nationality.Valid {
		person.Nationality = &nationality.String
	}
	if nationalityProb.Valid {
		person.NationalityProbability = &nationalityProb.Float64
	}
	return &person, nil
}

func (r *Repository) GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error) {
	logger.Debug(ctx, "getting persons with filter",
		zap.Any("filter", filter),
		zap.Int("offset", offset),
		zap.Int("limit", limit))
	baseQuery := `FROM persons WHERE 1=1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	dataQuery := `
        SELECT id, name, surname, patronymic, age, gender, gender_probability, 
               nationality, nationality_probability, created_at, updated_at
    ` + baseQuery
	var args []interface{}
	argNum := 1
	var conditions []string
	for field, value := range filter {
		switch field {
		case "name", "surname", "patronymic", "gender", "nationality":
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", field, argNum))
			args = append(args, fmt.Sprintf("%%%v%%", value))
		case "age":
			conditions = append(conditions, fmt.Sprintf("%s = $%d", field, argNum))
			args = append(args, value)
		default:
			logger.Warn(ctx, "ignoring unknown filter field", zap.String("field", field))
			continue
		}
		argNum++
	}
	if len(conditions) > 0 {
		filterCondition := " AND " + strings.Join(conditions, " AND ")
		countQuery += filterCondition
		dataQuery += filterCondition
	}
	var total int
	err := r.db.Pool().QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		logger.Error(ctx, "failed to count persons", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count persons: %w", err)
	}
	if total == 0 {
		return []*domain.Person{}, 0, nil
	}
	dataQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)
	rows, err := r.db.Pool().Query(ctx, dataQuery, args...)
	if err != nil {
		logger.Error(ctx, "failed to query persons", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to query persons: %w", err)
	}
	defer rows.Close()
	var persons []*domain.Person
	for rows.Next() {
		var person domain.Person
		var patronymic, gender, nationality sql.NullString
		var age sql.NullInt32
		var genderProb, nationalityProb sql.NullFloat64
		err := rows.Scan(
			&person.ID,
			&person.Name,
			&person.Surname,
			&patronymic,
			&age,
			&gender,
			&genderProb,
			&nationality,
			&nationalityProb,
			&person.CreatedAt,
			&person.UpdatedAt,
		)
		if err != nil {
			logger.Error(ctx, "failed to scan person row", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to scan person row: %w", err)
		}
		if patronymic.Valid {
			person.Patronymic = &patronymic.String
		}
		if age.Valid {
			ageVal := int(age.Int32)
			person.Age = &ageVal
		}
		if gender.Valid {
			person.Gender = &gender.String
		}
		if genderProb.Valid {
			person.GenderProbability = &genderProb.Float64
		}
		if nationality.Valid {
			person.Nationality = &nationality.String
		}
		if nationalityProb.Valid {
			person.NationalityProbability = &nationalityProb.Float64
		}
		persons = append(persons, &person)
	}
	if rows.Err() != nil {
		logger.Error(ctx, "error iterating through rows", zap.Error(rows.Err()))
		return nil, 0, fmt.Errorf("error iterating through rows: %w", rows.Err())
	}
	return persons, total, nil
}

func (r *Repository) CreatePerson(ctx context.Context, person *domain.Person) error {
	logger.Debug(ctx, "creating new person", zap.String("name", person.Name), zap.String("surname", person.Surname))
	if person.ID == uuid.Nil {
		person.ID = uuid.New()
	}
	now := time.Now().UTC()
	person.CreatedAt, person.UpdatedAt = now, now
	query := `
        INSERT INTO persons (
            id, name, surname, patronymic, age, gender, gender_probability, 
            nationality, nationality_probability, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `
	_, err := r.db.Pool().Exec(ctx, query,
		person.ID,
		person.Name,
		person.Surname,
		person.Patronymic,
		person.Age,
		person.Gender,
		person.GenderProbability,
		person.Nationality,
		person.NationalityProbability,
		person.CreatedAt,
		person.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			logger.Error(ctx, "person with this ID already exists",
				zap.String("id", person.ID.String()),
				zap.Error(err))
			return fmt.Errorf("%w: ID %s", domain.ErrPersonAlreadyExists, person.ID)
		}
		logger.Error(ctx, "failed to create person", zap.Error(err))
		return fmt.Errorf("failed to create person: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePerson(ctx context.Context, person *domain.Person) error {
	logger.Debug(ctx, "updating person", zap.String("id", person.ID.String()))
	person.UpdatedAt = time.Now().UTC()
	query := `
        UPDATE persons
        SET name = $2, surname = $3, patronymic = $4, age = $5, 
            gender = $6, gender_probability = $7, nationality = $8, 
            nationality_probability = $9, updated_at = $10
        WHERE id = $1
    `
	result, err := r.db.Pool().Exec(ctx, query,
		person.ID,
		person.Name,
		person.Surname,
		person.Patronymic,
		person.Age,
		person.Gender,
		person.GenderProbability,
		person.Nationality,
		person.NationalityProbability,
		person.UpdatedAt,
	)
	if err != nil {
		logger.Error(ctx, "failed to update person", zap.Error(err))
		return fmt.Errorf("failed to update person: %w", err)
	}
	if result.RowsAffected() == 0 {
		logger.Error(ctx, "person not found for update", zap.String("id", person.ID.String()))
		return fmt.Errorf("%w: id %s", domain.ErrPersonNotFound, person.ID)
	}
	return nil
}

func (r *Repository) DeletePerson(ctx context.Context, personID uuid.UUID) error {
	logger.Debug(ctx, "deleting person", zap.String("id", personID.String()))
	query := `DELETE FROM persons WHERE id = $1`
	result, err := r.db.Pool().Exec(ctx, query, personID)
	if err != nil {
		logger.Error(ctx, "failed to delete person", zap.Error(err))
		return fmt.Errorf("failed to delete person: %w", err)
	}
	if result.RowsAffected() == 0 {
		logger.Debug(ctx, "person not found for deletion", zap.String("id", personID.String()))
		return fmt.Errorf("%w: id %s", domain.ErrPersonNotFound, personID)
	}
	return nil
}

func (r *Repository) ExistsByID(ctx context.Context, personID uuid.UUID) (bool, error) {
	logger.Debug(ctx, "checking if person exists", zap.String("id", personID.String()))
	query := `SELECT EXISTS(SELECT 1 FROM persons WHERE id = $1)`
	var exists bool
	if err := r.db.Pool().QueryRow(ctx, query, personID).Scan(&exists); err != nil {
		logger.Error(ctx, "failed to check if person exists", zap.Error(err))
		return false, fmt.Errorf("failed to check if person exists: %w", err)
	}
	return exists, nil
}
