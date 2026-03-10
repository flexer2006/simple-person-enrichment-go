package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flexer2006/pes-api/internal/service/adapters/enrichment"
	"github.com/flexer2006/pes-api/internal/service/adapters/postgres"
	"github.com/flexer2006/pes-api/internal/service/adapters/server"
	"github.com/flexer2006/pes-api/internal/service/config"
	"github.com/flexer2006/pes-api/internal/service/logger"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, configPath string) error {
	logger.SetGlobal(logger.NewConsole(logger.InfoLevel, true))
	defer func() { _ = logger.Global().Sync() }()
	logger.Info(ctx, "loading configuration")
	cfg := new(config.Config)
	if configPath != "" {
		if fi, err := os.Stat(configPath); err == nil && !fi.IsDir() {
			if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
				logger.Error(ctx, "config read failed", zap.Error(err), zap.String("path", configPath))
				return fmt.Errorf("read config %s: %w", configPath, err)
			}
		}
	}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		logger.Error(ctx, "env read failed", zap.Error(err))
		return fmt.Errorf("read env: %w", err)
	}
	if loggable, ok := any(cfg).(interface{ LogFields() []zap.Field }); ok {
		logger.Info(ctx, "configuration loaded", loggable.LogFields()...)
	} else {
		logger.Info(ctx, "configuration loaded")
	}
	var err error
	var finalLogger *zap.Logger
	switch cfg.Logger.Model {
	case "production":
		finalLogger, err = logger.NewProduction()
	default:
		if cfg.Logger.Model != "development" {
			logger.Warn(ctx, "unknown logger model, using development", zap.String("model", cfg.Logger.Model))
		}
		finalLogger, err = logger.NewDevelopment()
	}
	if err != nil {
		logger.Error(ctx, "init logger", zap.Error(err))
		return err
	}
	logger.SetGlobal(finalLogger)
	logger.Info(ctx, "initializing database")
	dbCfg := postgres.PostgresConfig{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
		MinConns: cfg.Postgres.PoolMinConns,
		MaxConns: cfg.Postgres.PoolMaxConns,
	}
	if cfg.Migrations.Path != "" {
		if err := postgres.RunMigration(ctx, cfg.Migrations.Path, dbCfg); err != nil {
			logger.Error(ctx, "migration failed", zap.Error(err))
			return err
		}
	}
	db, err := postgres.NewDatabase(ctx, dbCfg)
	if err != nil {
		logger.Error(ctx, "init db", zap.Error(err))
		return err
	}
	if err := db.Ping(ctx); err != nil {
		logger.Error(ctx, "db ping", zap.Error(err))
		return err
	}
	logger.Info(ctx, "database ready")
	logger.Info(ctx, "initializing application")
	repos := postgres.New(db)
	apiAdapter := enrichment.NewAPI()
	httpServer := server.New(*cfg, apiAdapter, repos)
	logger.Info(ctx, "application initialized successfully")
	shutdownTimeout := 5 * time.Second
	if d, err := time.ParseDuration(cfg.Graceful.ShutdownTimeout); err == nil {
		shutdownTimeout = d
	} else {
		logger.Error(ctx, "bad duration, defaulting", zap.Error(err))
	}
	appCtx, cancel := context.WithCancel(ctx)
	go func() {
		logger.Info(appCtx, "starting application")
		if err := httpServer.Start(appCtx); err != nil {
			logger.Error(appCtx, "app stopped", zap.Error(err))
			cancel()
		}
	}()
	defer cancel()
	logger.Info(ctx, "service started",
		zap.String("env", cfg.Logger.Model),
		zap.String("level", cfg.Logger.Level),
	)
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, shutdownTimeout)
	defer shutdownCancel()
	var g errgroup.Group
	g.Go(func() error {
		cancel()
		logger.Info(shutdownCtx, "stopping application")
		stopTimeout := 5 * time.Second
		if d, err := time.ParseDuration(cfg.Graceful.ShutdownTimeout); err == nil {
			stopTimeout = d
		} else {
			logger.Error(shutdownCtx, "invalid graceful shutdown timeout, using default", zap.Error(err))
		}
		stopCtx, stopCancel := context.WithTimeout(shutdownCtx, stopTimeout)
		defer stopCancel()
		if err := httpServer.Stop(stopCtx); err != nil {
			logger.Error(stopCtx, "error stopping HTTP server", zap.Error(err))
		}
		if db != nil {
			logger.Info(stopCtx, "closing db")
			db.Close(stopCtx)
		}
		logger.Info(stopCtx, "application stopped")
		return nil
	})
	err = g.Wait()
	if err != nil {
		logger.Error(ctx, "shutdown error", zap.Error(err))
	}
	logger.Info(ctx, "shutdown complete")
	return err
}
