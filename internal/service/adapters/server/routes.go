package server

import (
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"

	"github.com/gofiber/fiber/v3"
)

func Setup(app *fiber.App, api ports.API, repositories ports.Repositories) {
	personHandler := NewPersonHandler(api, repositories)
	v1 := app.Group("/api/v1")
	persons := v1.Group("/persons")
	persons.Get("/", personHandler.GetPersons)
	persons.Get("/:id", personHandler.GetPersonByID)
	persons.Post("/", personHandler.CreatePerson)
	persons.Put("/:id", personHandler.UpdatePerson)
	persons.Patch("/:id", personHandler.UpdatePerson)
	persons.Delete("/:id", personHandler.DeletePerson)
	persons.Post("/:id/enrich", personHandler.EnrichPerson)
}
