package server

import (
	"strconv"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/logger"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type personHandler struct {
	repositories ports.Repositories
	api          ports.API
}

func newPersonHandler(api ports.API, repositories ports.Repositories) *personHandler {
	return new(personHandler{api: api, repositories: repositories})
}

// GetPersons godoc
// @Summary List persons
// @Description Filtered, paginated list
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
func (h *personHandler) getPersons(c fiber.Ctx) error {
	reqCtx := c.Context()
	limit, offset := parseIntQuery(c, "limit", 10, 1, 0), parseIntQuery(c, "offset", 0, 0, 0)
	filter := make(map[string]any)
	for _, field := range []string{"name", "surname", "patronymic", "gender", "nationality"} {
		if value := c.Query(field); value != "" {
			filter[field] = value
		}
	}
	if ageStr := c.Query("age"); ageStr != "" {
		if age, err := strconv.Atoi(ageStr); err == nil {
			filter["age"] = age
		}
	}
	persons, total, err := h.repositories.GetPersons(reqCtx, filter, offset, limit)
	if err := repoError(reqCtx, err, "failed to get persons", c); err != nil {
		return err
	}
	if err := c.Status(fiber.StatusOK).JSON(fiber.Map{
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
// @Summary Retrieve person
// @Description By UUID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 200 {object} domain.Person "Successfully retrieved person"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [get]
func (h *personHandler) getPersonByID(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	reqCtx := c.Context()
	person, err := h.repositories.GetByID(reqCtx, id)
	if err := repoError(reqCtx, err, "failed to get person", c); err != nil {
		return err
	}
	return c.Status(fiber.StatusOK).JSON(person)
}

// CreatePerson godoc
// @Summary Create person
// @Description Create a person
// @Tags persons
// @Accept json
// @Produce json
// @Param person body domain.Person true "Person object to be created"
// @Success 201 {object} domain.Person "Successfully created person"
// @Failure 400 {object} map[string]string "Bad request - Invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons [post]
func (h *personHandler) createPerson(c fiber.Ctx) error {
	person, err := bindBody[domain.Person](c)
	if err != nil {
		_ = respondError(c, err)
		return err
	}
	if err := validatePerson(&person); err != nil {
		_ = respondError(c, err)
		return err
	}
	if person.ID == uuid.Nil {
		person.ID = uuid.New()
	}
	reqCtx := c.Context()
	if err := h.repositories.CreatePerson(reqCtx, &person); err != nil {
		logger.Error(reqCtx, "failed to create person", zap.Error(err))
		_ = respondError(c, err)
		return err
	}
	if err := c.Status(fiber.StatusCreated).JSON(person); err != nil {
		return err
	}
	return nil
}

// UpdatePerson godoc
// @Summary Update person
// @Description Modify existing record
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Param person body domain.Person true "Person data to update"
// @Success 200 {object} domain.Person "Successfully updated person"
// @Failure 400 {object} map[string]string "Bad request - Invalid input"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [put]
func (h *personHandler) updatePerson(c fiber.Ctx) error {
	reqCtx := c.Context()
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	exists, err := h.repositories.ExistsByID(reqCtx, id)
	if err := repoError(reqCtx, err, "failed to check if person exists", c); err != nil {
		return err
	}
	if !exists {
		return respondError(c, domain.ErrPersonNotFound)
	}
	person, err := bindBody[domain.Person](c)
	if err != nil {
		return respondError(c, err)
	}
	person.ID = id
	if err := validatePerson(&person); err != nil {
		return respondError(c, err)
	}
	if err := h.repositories.UpdatePerson(reqCtx, &person); err != nil {
		logger.Error(reqCtx, "failed to update person", zap.Error(err))
		return respondError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(person)
}

// DeletePerson godoc
// @Summary Remove person
// @Description By UUID
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 204 "Successfully deleted"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id} [delete]
func (h *personHandler) deletePerson(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	reqCtx := c.Context()
	if err := h.repositories.DeletePerson(reqCtx, id); err != nil {
		if err := repoError(reqCtx, err, "failed to delete person", c); err != nil {
			return err
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// EnrichPerson godoc
// @Summary Enrich person
// @Description Add age, gender, nationality
// @Tags persons
// @Accept json
// @Produce json
// @Param id path string true "Person UUID" format(uuid)
// @Success 200 {object} domain.Person "Successfully enriched person"
// @Failure 400 {object} map[string]string "Bad request - Invalid UUID"
// @Failure 404 {object} map[string]string "Person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /persons/{id}/enrich [post]
// helper used internally to avoid repeating nearly-identical enrichment branches.
// K is the concrete type returned by the external API (int for age, string for
// gender/nationality).  `existing` must be a pointer to a pointer field on
// Person, so the caller can pass &person.Age, &person.Gender, etc.
func (h *personHandler) enrichPerson(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}
	reqCtx := c.Context()
	person, err := h.repositories.GetByID(reqCtx, id)
	if err := repoError(reqCtx, err, "failed to get person", c); err != nil {
		return err
	}
	enrichValue(reqCtx, person, &person.Age, person.Name, h.api.GetAgeByName,
		func(p *domain.Person, v int, prob float64) { p.Age = &v },
		"age")
	enrichValue(reqCtx, person, &person.Gender, person.Name, h.api.GetGenderByName,
		func(p *domain.Person, v string, prob float64) { p.Gender = &v; p.GenderProbability = &prob },
		"gender")
	enrichValue(reqCtx, person, &person.Nationality, person.Name, h.api.GetNationalityByName,
		func(p *domain.Person, v string, prob float64) { p.Nationality = &v; p.NationalityProbability = &prob },
		"nationality")
	err = h.repositories.UpdatePerson(reqCtx, person)
	if err != nil {
		logger.Error(reqCtx, "failed to save enriched data", zap.Error(err))
		return respondError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(person)
}
