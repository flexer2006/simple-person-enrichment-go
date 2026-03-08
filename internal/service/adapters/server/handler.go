package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	logger "github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func sendJSONResponse(ctx fiber.Ctx, status int, body interface{}) error {
	if err := ctx.Status(status).JSON(body); err != nil {
		return fmt.Errorf("failed to send JSON response: %w", err)
	}
	return nil
}

func sendError(ctx fiber.Ctx, status int, msg string) error {
	return sendJSONResponse(ctx, status, fiber.Map{"error": msg})
}

func parseUUIDParam(ctx fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := ctx.Params(name)
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("missing %s parameter", name)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

type PersonHandler struct {
	api          ports.API
	repositories ports.Repositories
}

func NewPersonHandler(api ports.API, repositories ports.Repositories) *PersonHandler {
	return &PersonHandler{
		api:          api,
		repositories: repositories,
	}
}

// GetPersons godoc
// @Summary Get list of persons
// @Description Get a list of persons with filtering and pagination
// @Tags persons
// @Accept json
// @Produce json
// @Param limit query int false "Page size limit" default(10) minimum(1)
// @Param offset query int false "Page offset" default(0) minimum(0)
// @Param name query string false "Filter by name"
// @Param surname query string false "Filter by surname"
// @Param patronymic query string false "Filter by patronymic"
// @Param gender query string false "Filter by gender"
// @Param nationality query string false "Filter by nationality"
// @Param age query int false "Filter by age"
// @Success 200 {object} map[string]interface{} "Successfully retrieved persons list"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons [get]
func (h *PersonHandler) GetPersons(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	logger.Debug(requestCtx, "handling get persons request")

	limitStr := ctx.Query("limit", "10")
	offsetStr := ctx.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	filter := make(map[string]any)
	for _, field := range []string{"name", "surname", "patronymic", "gender", "nationality"} {
		if value := ctx.Query(field); value != "" {
			filter[field] = value
		}
	}

	if ageStr := ctx.Query("age"); ageStr != "" {
		if age, err := strconv.Atoi(ageStr); err == nil {
			filter["age"] = age
		}
	}

	persons, total, err := h.repositories.GetPersons(requestCtx, filter, offset, limit)
	if err != nil {
		logger.Error(requestCtx, "failed to get persons", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to retrieve persons"); err != nil {
			return err
		}
		return fmt.Errorf("failed to get persons: %w", err)
	}

	if err := sendJSONResponse(ctx, fiber.StatusOK, fiber.Map{
		"data":   persons,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}); err != nil {
		return err
	}
	return nil
}

// GetPersonByID godoc
// @Summary Get person by ID
// @Description Get person details by UUID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 200 {object} entities.Person "Successfully retrieved person"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [get]
func (h *PersonHandler) GetPersonByID(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	idParam := ctx.Params("id")

	logger.Debug(requestCtx, "handling get person by ID request", zap.String("id", idParam))

	personID, err := parseUUIDParam(ctx, "id")
	if err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid UUID format"); err != nil {
			return err
		}
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	person, err := h.repositories.GetByID(requestCtx, personID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if err := sendError(ctx, fiber.StatusNotFound, "Person not found"); err != nil {
				return err
			}
			return fmt.Errorf("person not found: %w", err)
		}
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to get person"); err != nil {
			return err
		}
		return fmt.Errorf("failed to get person: %w", err)
	}

	if err := sendJSONResponse(ctx, fiber.StatusOK, person); err != nil {
		return err
	}
	return nil
}

// CreatePerson godoc
// @Summary Create new person
// @Description Create a new person with the input data
// @Tags persons
// @Accept json
// @Produce json
// @Param person body entities.Person true "Person object to be created"
// @Success 201 {object} entities.Person "Successfully created person"
// @Failure 400 {object} map[string]string "Bad request - Invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons [post]
func (h *PersonHandler) CreatePerson(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	logger.Debug(requestCtx, "handling create person request")

	var person domain.Person
	if err := ctx.Bind().Body(&person); err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid request body"); err != nil {
			return err
		}
		return fmt.Errorf("invalid request body: %w", err)
	}

	if person.Name == "" || person.Surname == "" {
		if err := sendError(ctx, fiber.StatusBadRequest, "Name and surname are required"); err != nil {
			return err
		}
		return fmt.Errorf("%w", domain.ErrNameSurnameRequired)
	}

	if person.ID == uuid.Nil {
		person.ID = uuid.New()
	}

	if err := h.repositories.CreatePerson(requestCtx, &person); err != nil {
		logger.Error(requestCtx, "failed to create person", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to create person"); err != nil {
			return err
		}
		return fmt.Errorf("failed to create person: %w", err)
	}

	if err := sendJSONResponse(ctx, fiber.StatusCreated, person); err != nil {
		return err
	}
	return nil
}

// UpdatePerson godoc
// @Summary Update person
// @Description Update an existing person
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Param person body entities.Person true "Person data to update"
// @Success 200 {object} entities.Person "Successfully updated person"
// @Failure 400 {object} map[string]string "Bad request - Invalid input"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [put]
func (h *PersonHandler) UpdatePerson(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	idParam := ctx.Params("id")

	logger.Debug(requestCtx, "handling update person request", zap.String("id", idParam))

	personID, err := parseUUIDParam(ctx, "id")
	if err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid UUID format"); err != nil {
			return err
		}
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	exists, err := h.repositories.ExistsByID(requestCtx, personID)
	if err != nil {
		logger.Error(requestCtx, "failed to check if person exists", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to check if person exists"); err != nil {
			return err
		}
		return fmt.Errorf("failed to check if person exists: %w", err)
	}

	if !exists {
		if err := sendError(ctx, fiber.StatusNotFound, "Person not found"); err != nil {
			return err
		}
		return fmt.Errorf("%w", domain.ErrPersonNotFound)
	}

	var person domain.Person
	if err := ctx.Bind().Body(&person); err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid request body"); err != nil {
			return err
		}
		return fmt.Errorf("invalid request body: %w", err)
	}

	person.ID = personID

	if person.Name == "" || person.Surname == "" {
		if err := sendError(ctx, fiber.StatusBadRequest, "Name and surname are required"); err != nil {
			return err
		}
		return fmt.Errorf("%w", domain.ErrNameSurnameRequired)
	}

	if err := h.repositories.UpdatePerson(requestCtx, &person); err != nil {
		logger.Error(requestCtx, "failed to update person", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to update person"); err != nil {
			return err
		}
		return fmt.Errorf("failed to update person: %w", err)
	}

	if err := sendJSONResponse(ctx, fiber.StatusOK, person); err != nil {
		return err
	}
	return nil
}

// DeletePerson godoc
// @Summary Delete person
// @Description Delete a person by UUID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 204 "Successfully deleted"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [delete]
func (h *PersonHandler) DeletePerson(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	idParam := ctx.Params("id")

	logger.Debug(requestCtx, "handling delete person request", zap.String("id", idParam))

	personID, err := parseUUIDParam(ctx, "id")
	if err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid UUID format"); err != nil {
			return err
		}
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	if err := h.repositories.DeletePerson(requestCtx, personID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			if err := sendError(ctx, fiber.StatusNotFound, "Person not found"); err != nil {
				return err
			}
			return fmt.Errorf("person not found: %w", err)
		}
		logger.Error(requestCtx, "failed to delete person", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to delete person"); err != nil {
			return err
		}
		return fmt.Errorf("failed to delete person: %w", err)
	}

	if err := ctx.SendStatus(fiber.StatusNoContent); err != nil {
		return fmt.Errorf("failed to send status: %w", err)
	}
	return nil
}

// EnrichPerson godoc
// @Summary Enrich person data
// @Description Enrich person with age, gender, and nationality data from external APIs
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 200 {object} entities.Person "Successfully enriched person"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id}/enrich [post]
func (h *PersonHandler) EnrichPerson(ctx fiber.Ctx) error {
	requestCtx := ctx.Context()
	idParam := ctx.Params("id")

	logger.Debug(requestCtx, "handling enrich person request", zap.String("id", idParam))

	personID, err := parseUUIDParam(ctx, "id")
	if err != nil {
		if err := sendError(ctx, fiber.StatusBadRequest, "Invalid UUID format"); err != nil {
			return err
		}
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	person, err := h.repositories.GetByID(requestCtx, personID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if err := sendError(ctx, fiber.StatusNotFound, "Person not found"); err != nil {
				return err
			}
			return fmt.Errorf("person not found: %w", err)
		}
		logger.Error(requestCtx, "failed to get person", zap.Error(err))
		if err := sendError(ctx, fiber.StatusInternalServerError, "Failed to get person"); err != nil {
			return err
		}
		return fmt.Errorf("failed to get person: %w", err)
	}

	if person.Age == nil {
		age, probability, err := h.api.Age().GetAgeByName(requestCtx, person.Name)
		if err == nil {
			person.Age = &age
			logger.Debug(requestCtx, "enriched with age data",
				zap.Int("age", age),
				zap.Float64("probability", probability))
		} else {
			logger.Warn(requestCtx, "failed to get age data", zap.Error(err))
		}
	}

	if person.Gender == nil {
		gender, probability, err := h.api.Gender().GetGenderByName(requestCtx, person.Name)
		if err == nil {
			person.Gender = &gender
			person.GenderProbability = &probability
			logger.Debug(requestCtx, "enriched with gender data",
				zap.String("gender", gender),
				zap.Float64("probability", probability))
		} else {
			logger.Warn(requestCtx, "failed to get gender data", zap.Error(err))
		}
	}

	if person.Nationality == nil {
		nationality, probability, err := h.api.Nationality().GetNationalityByName(requestCtx, person.Name)
		if err == nil {
			person.Nationality = &nationality
			person.NationalityProbability = &probability
			logger.Debug(requestCtx, "enriched with nationality data",
				zap.String("nationality", nationality),
				zap.Float64("probability", probability))
		} else {
			logger.Warn(requestCtx, "failed to get nationality data", zap.Error(err))
		}
	}

	err = h.repositories.UpdatePerson(requestCtx, person)
	if err != nil {
		logger.Error(requestCtx, "failed to save enriched data", zap.Error(err))
		if serr := sendError(ctx, fiber.StatusInternalServerError, "Failed to save enriched data"); serr != nil {
			return serr
		}
		return fmt.Errorf("failed to save enriched data: %w", err)
	}

	if err := sendJSONResponse(ctx, fiber.StatusOK, person); err != nil {
		return err
	}
	return nil
}
