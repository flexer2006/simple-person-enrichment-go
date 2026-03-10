package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flexer2006/pes-api/internal/service/domain"
	"github.com/flexer2006/pes-api/internal/service/logger"
	"github.com/flexer2006/pes-api/internal/service/ports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const (
	personCols = `id, name, surname, patronymic, age, gender, gender_probability, nationality, nationality_probability, created_at, updated_at`
	baseFrom   = `FROM persons WHERE 1=1`
)

type Repository struct {
	db ports.Database
}

func New(db ports.Database) ports.Repositories {
	return new(Repository{db: db})
}

func (r *Repository) GetByID(ctx context.Context, personID uuid.UUID) (*domain.Person, error) {
	p, err := scanPerson(r.db.Pool().QueryRow(ctx, fmt.Sprintf("SELECT %s %s AND id = $1", personCols, baseFrom), personID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound(ctx, personID)
		}
		return nil, wrapError(ctx, "failed to get person by ID", err)
	}
	return p, nil
}

func (r *Repository) GetPersons(ctx context.Context, filter map[string]any, offset, limit int) ([]*domain.Person, int, error) {
	allowed := map[string]string{
		"name":        "name",
		"surname":     "surname",
		"patronymic":  "patronymic",
		"gender":      "gender",
		"nationality": "nationality",
		"age":         "age",
	}
	filterCondition, args := buildFilter(filter, allowed)
	dataQuery := fmt.Sprintf("SELECT %s %s%s", personCols, baseFrom, filterCondition)
	var total int
	if err := r.db.Pool().QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) %s%s", baseFrom, filterCondition), args...).Scan(&total); err != nil {
		return nil, 0, wrapError(ctx, "failed to count persons", err)
	}
	if total == 0 {
		return []*domain.Person{}, 0, nil
	}
	startIdx := len(args) + 1
	dataQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", startIdx, startIdx+1)
	args = append(args, limit, offset)
	rows, err := r.db.Pool().Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, wrapError(ctx, "failed to query persons", err)
	}
	defer rows.Close()
	persons, err := scanRows(rows, scanPerson)
	if err != nil {
		return nil, 0, wrapError(ctx, "failed to scan persons", err)
	}
	return persons, total, nil
}

func (r *Repository) CreatePerson(ctx context.Context, person *domain.Person) error {
	if person.ID == uuid.Nil {
		person.ID = uuid.New()
	}
	now := time.Now().UTC()
	person.CreatedAt, person.UpdatedAt = now, now
	_, err := r.db.Pool().Exec(ctx, fmt.Sprintf(
		"\n        INSERT INTO persons (\n            %s\n        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)\n    ", personCols),
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
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			logger.Error(ctx, "person with this ID already exists",
				zap.String("id", person.ID.String()),
				zap.Error(err))
			return fmt.Errorf("%w: ID %s", domain.ErrPersonAlreadyExists, person.ID)
		}
		return wrapError(ctx, "failed to create person", err)
	}
	return nil
}

func (r *Repository) UpdatePerson(ctx context.Context, person *domain.Person) error {
	person.UpdatedAt = time.Now().UTC()
	result, err := r.db.Pool().Exec(ctx, `
        UPDATE persons
        SET name = $2, surname = $3, patronymic = $4, age = $5, 
            gender = $6, gender_probability = $7, nationality = $8, 
            nationality_probability = $9, updated_at = $10
        WHERE id = $1
    `,
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
		return wrapError(ctx, "failed to update person", err)
	}
	if result.RowsAffected() == 0 {
		return notFound(ctx, person.ID)
	}
	return nil
}

func (r *Repository) DeletePerson(ctx context.Context, personID uuid.UUID) error {
	result, err := r.db.Pool().Exec(ctx, `DELETE FROM persons WHERE id = $1`, personID)
	if err != nil {
		return wrapError(ctx, "failed to delete person", err)
	}
	if result.RowsAffected() == 0 {
		return notFound(ctx, personID)
	}
	return nil
}

func (r *Repository) ExistsByID(ctx context.Context, personID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM persons WHERE id = $1)`, personID).Scan(&exists); err != nil {
		return false, wrapError(ctx, "failed to check if person exists", err)
	}
	return exists, nil
}
