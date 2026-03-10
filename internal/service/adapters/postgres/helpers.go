package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/flexer2006/pes-api/internal/service/domain"
	"github.com/flexer2006/pes-api/internal/service/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func buildFilter[K comparable, V any](filter map[K]V, columns map[K]string) (string, []any) {
	var conditions []string
	args := make([]any, 0, len(filter))
	i := 1
	for key, val := range filter {
		col, ok := columns[key]
		if !ok {
			continue
		}
		switch v := any(val).(type) {
		case string:
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", col, i))
			args = append(args, fmt.Sprintf("%%%v%%", v))
		default:
			conditions = append(conditions, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, v)
		}
		i++
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " AND " + strings.Join(conditions, " AND "), args
}

func scanPerson(row pgx.Row) (*domain.Person, error) {
	var person domain.Person
	var patronymic, nationality, gender sql.NullString
	var age sql.NullInt32
	var genderProb, nationalityProb sql.NullFloat64
	if err := row.Scan(
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
	); err != nil {
		return nil, err
	}
	person.Patronymic = sqlNullStr(patronymic)
	person.Age = sqlNullInt(age)
	person.Gender = sqlNullStr(gender)
	person.GenderProbability = sqlNullFloat(genderProb)
	person.Nationality = sqlNullStr(nationality)
	person.NationalityProbability = sqlNullFloat(nationalityProb)
	return &person, nil
}

func scanRows[T any](rows pgx.Rows, scan func(pgx.Row) (*T, error)) ([]*T, error) {
	var items []*T
	for rows.Next() {
		it, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func sqlNullStr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func sqlNullInt(ns sql.NullInt32) *int {
	if ns.Valid {
		v := int(ns.Int32)
		return &v
	}
	return nil
}

func sqlNullFloat(ns sql.NullFloat64) *float64 {
	if ns.Valid {
		return &ns.Float64
	}
	return nil
}

func clamp32(n int) int32 {
	if n <= 0 {
		return 0
	}
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(n)
}

func wrapError(ctx context.Context, msg string, err error) error {
	logger.Error(ctx, msg, zap.Error(err))
	return fmt.Errorf("%s: %w", msg, err)
}

func notFound(ctx context.Context, id uuid.UUID) error {
	logger.Debug(ctx, "record not found", zap.String("id", id.String()))
	return fmt.Errorf("%w: id %s", domain.ErrPersonNotFound, id)
}
