package server

import (
	"context"
	"fmt"

	_ "github.com/flexer2006/pes-api/docs/swagger"
	"github.com/flexer2006/pes-api/internal/service/config"
	"github.com/flexer2006/pes-api/internal/service/logger"
	"github.com/flexer2006/pes-api/internal/service/ports"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// @title PES API
// @version 1.0
// @BasePath /api/v1

type Server struct {
	app    *fiber.App
	config config.Config
}

func New(config config.Config, api ports.API, repositories ports.Repositories) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
		AppName:      "PES",
	})
	personHandler := newPersonHandler(api, repositories)
	v1 := app.Group("/api/v1")
	persons := v1.Group("/persons")
	app.Get("/swagger", func(c fiber.Ctx) error {
		r := c.Redirect()
		r.Status(fiber.StatusFound)
		return r.To("/swagger/swagger.html")
	})
	app.Get("/swagger/swagger.html", func(c fiber.Ctx) error {
		return c.SendFile("./docs/swagger/swagger.html")
	})
	app.Get("/swagger/swagger.json", func(c fiber.Ctx) error {
		return c.SendFile("./docs/swagger/swagger.json")
	})
	persons.Get("/", personHandler.getPersons)
	persons.Get("/:id", personHandler.getPersonByID)
	persons.Post("/", personHandler.createPerson)
	persons.Put("/:id", personHandler.updatePerson)
	persons.Patch("/:id", personHandler.updatePerson)
	persons.Delete("/:id", personHandler.deletePerson)
	persons.Post("/:id/enrich", personHandler.enrichPerson)
	return new(Server{
		app:    app,
		config: config,
	})
}

func (s *Server) Start(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	logger.Info(ctx, "starting HTTP server", zap.String("address", address))
	go func() {
		if err := s.app.Listen(address); err != nil {
			logger.Error(ctx, "failed to start HTTP server", zap.Error(err))
		}
	}()
	<-ctx.Done()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping HTTP server")
	if err := s.app.ShutdownWithContext(ctx); err != nil {
		logger.Error(ctx, "failed to shutdown HTTP server gracefully", zap.Error(err))
		return fmt.Errorf("failed to shutdown HTTP server gracefully: %w", err)
	}
	return nil
}
