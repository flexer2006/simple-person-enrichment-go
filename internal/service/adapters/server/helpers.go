package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/flexer2006/pes-api/internal/service/domain"
	"github.com/flexer2006/pes-api/internal/service/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

func parseIntQuery[T integer](ctx fiber.Ctx, name string, defaultVal, min, max T) T {
	str := ctx.Query(name)
	if str == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal
	}
	v := T(parsed)
	if min != 0 && v < min {
		return defaultVal
	}
	if max != 0 && v > max {
		return defaultVal
	}
	return v
}

func enrichValue[K any](ctx context.Context, person *domain.Person, existing **K, name string,
	getter func(context.Context, string) (K, float64, error),
	setter func(*domain.Person, K, float64),
	logField string,
) {
	if *existing != nil {
		return
	}
	v, prob, err := getter(ctx, name)
	if err == nil {
		setter(person, v, prob)
	} else {
		logger.Warn(ctx, "failed to get "+logField+" data", zap.Error(err))
	}
}

func bindBody[T any](ctx fiber.Ctx) (T, error) {
	var out T
	if err := ctx.Bind().Body(&out); err != nil {
		return out, errors.Join(domain.ErrInvalidRequestBody, err)
	}
	return out, nil
}

func statusMessage(err error) (int, string) {
	for _, entry := range [...]struct {
		target error
		msg    string
		code   int
	}{
		{target: domain.ErrPersonNotFound, msg: "Person not found", code: fiber.StatusNotFound},
		{target: domain.ErrPersonAlreadyExists, msg: "Person already exists", code: fiber.StatusConflict},
		{target: domain.ErrNameSurnameRequired, msg: "Name and surname are required", code: fiber.StatusBadRequest},
		{target: domain.ErrInvalidUUID, msg: "Invalid UUID format", code: fiber.StatusBadRequest},
		{target: domain.ErrInvalidRequestBody, msg: "Invalid request body", code: fiber.StatusBadRequest},
	} {
		if errors.Is(err, entry.target) {
			return entry.code, entry.msg
		}
	}
	return fiber.StatusInternalServerError, "Internal server error"
}

func parseUUIDParam(ctx fiber.Ctx, name string) (uuid.UUID, error) { //nolint:unparam
	idStr := ctx.Params(name)
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("%w: missing %s", domain.ErrInvalidUUID, name)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", domain.ErrInvalidUUID, err)
	}
	return id, nil
}

func validatePerson(p *domain.Person) error {
	if p.Name == "" || p.Surname == "" {
		return domain.ErrNameSurnameRequired
	}
	return nil
}

func respondError(ctx fiber.Ctx, err error) error {
	status, msg := statusMessage(err)
	return ctx.Status(status).JSON(fiber.Map{"error": msg})
}

func repoError(ctx context.Context, err error, logMsg string, fibCtx fiber.Ctx) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrPersonNotFound) {
		logger.Error(ctx, logMsg, zap.Error(err))
	}
	_ = respondError(fibCtx, err)
	return err
}
