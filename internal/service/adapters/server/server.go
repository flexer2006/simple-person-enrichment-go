package server

import (
	"context"
	"fmt"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/ports"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"

	_ "github.com/flexer2006/case-person-enrichment-go/docs/swagger"
)

// @title Person Enrichment API
// @version 1.0
// @description API for managing and enriching person data with external services
// @contact.name API Support
// @contact.email andrewgo1133official@gmail.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @BasePath /api/v1

type Server struct {
	app    *fiber.App
	config domain.Config
}

func New(config domain.Config, api ports.API, repositories ports.Repositories) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  config.Server.ReadTimeout,
		WriteTimeout: config.Server.WriteTimeout,
		AppName:      "Person Enrichment Service",
	})
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
	Setup(app, api, repositories)
	return &Server{
		app:    app,
		config: config,
	}
}

func (s *Server) Start(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	utilities.Info(ctx, "starting HTTP server", zap.String("address", address))
	go func() {
		if err := s.app.Listen(address); err != nil {
			utilities.Error(ctx, "failed to start HTTP server", zap.Error(err))
		}
	}()
	<-ctx.Done()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	utilities.Info(ctx, "stopping HTTP server")
	if err := s.app.ShutdownWithContext(ctx); err != nil {
		utilities.Error(ctx, "failed to shutdown HTTP server gracefully", zap.Error(err))
		return fmt.Errorf("failed to shutdown HTTP server gracefully: %w", err)
	}
	return nil
}

func (s *Server) GetConfig() domain.Config {
	return s.config
}
