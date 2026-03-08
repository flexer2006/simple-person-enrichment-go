package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/service/app"
	"github.com/flexer2006/case-person-enrichment-go/internal/service/domain"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/flexer2006/case-person-enrichment-go/internal/utilities/database"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	initialLogger := utilities.NewConsole(utilities.InfoLevel, true)
	utilities.SetGlobal(initialLogger)
	defer safeSyncLogger(initialLogger)
	cfg, err := utilities.Load[domain.Config](ctx, utilities.LoadOptions{
		ConfigPath: "./deploy/.env",
	})
	if err != nil {
		utilities.Error(ctx, "failed to load configuration", zap.Error(err))
		return err
	}
	finalLogger, err := setupLogger(ctx, cfg.Logger.Model)
	if err != nil {
		return err
	}
	utilities.SetGlobal(finalLogger)
	shutdownTimeout, err := time.ParseDuration(cfg.Graceful.ShutdownTimeout)
	if err != nil {
		utilities.Error(ctx, "invalid graceful shutdown timeout, defaulting to 5s", zap.Error(err))
		shutdownTimeout = 5 * time.Second
	}
	utilities.Info(ctx, "initializing database")
	data, err := database.New(ctx, database.Config{
		Postgres: database.PostgresConfig{
			Host:     cfg.Postgres.Host,
			Port:     cfg.Postgres.Port,
			User:     cfg.Postgres.User,
			Password: cfg.Postgres.Password,
			Database: cfg.Postgres.Database,
			SSLMode:  cfg.Postgres.SSLMode,
			MinConns: cfg.Postgres.PoolMinConns,
			MaxConns: cfg.Postgres.PoolMaxConns,
		},
		Migrate: database.MigrateConfig{
			Path: cfg.Migrations.Path,
		},
		ApplyMigrations: true,
	})
	if err != nil {
		utilities.Error(ctx, "failed to initialize database", zap.Error(err))
		return err
	}
	if err := data.Ping(ctx); err != nil {
		utilities.Error(ctx, "database ping failed", zap.Error(err))
		return err
	}
	version, dirty, err := data.GetMigrationVersion(ctx)
	if err != nil {
		utilities.Warn(ctx, "failed to get migration version", zap.Error(err))
	} else if dirty {
		utilities.Warn(ctx, "database has dirty migration", zap.Uint("version", version))
	} else {
		utilities.Info(ctx, "current migration version", zap.Uint("version", version))
	}
	utilities.Info(ctx, "database initialized successfully")
	application, err := app.NewApplication(ctx, cfg, data.Provider(), nil)
	if err != nil {
		utilities.Error(ctx, "failed to initialize application", zap.Error(err))
		return err
	}
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()
	go func() {
		if err := application.Start(appCtx); err != nil {
			utilities.Error(ctx, "application stopped with error", zap.Error(err))
			appCancel()
		}
	}()
	logStartupInfo(ctx, cfg)
	err = utilities.Wait(ctx, shutdownTimeout,
		func(ctx context.Context) error {
			appCancel()
			return application.Stop(ctx)
		},
		func(ctx context.Context) error {
			utilities.Info(ctx, "closing database connection")
			data.Close(ctx)
			return nil
		},
	)
	if err != nil {
		utilities.Error(ctx, "shutdown hooks returned error", zap.Error(err))
	}
	utilities.Info(ctx, "service shutdown complete")
	return nil
}

func safeSyncLogger(logger *utilities.Logger) {
	if err := logger.Sync(); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "sync /dev/stderr: invalid argument") ||
			strings.Contains(errMsg, "sync /dev/stdout: invalid argument") {
			return
		}
		if n, writeErr := fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err); writeErr != nil {
			panic(fmt.Sprintf("failed to write error message to stderr: %v", writeErr))
		} else if n == 0 {
			panic("failed to write error message to stderr: zero bytes written")
		}
	}
}

func setupLogger(ctx context.Context, model string) (*utilities.Logger, error) {
	var logger *utilities.Logger
	var err error
	switch model {
	case "development":
		logger, err = utilities.NewDevelopment()
	case "production":
		logger, err = utilities.NewProduction()
	default:
		utilities.Warn(ctx, "unknown logger model, using development", zap.String("model", model))
		logger, err = utilities.NewDevelopment()
	}
	if err != nil {
		utilities.Error(ctx, "failed to initialize logger with config", zap.Error(err))
		return nil, err
	}
	return logger, nil
}

func logStartupInfo(ctx context.Context, cfg *domain.Config) {
	utilities.Info(ctx, "service started",
		zap.String("environment", cfg.Logger.Model),
		zap.String("log_level", cfg.Logger.Level),
		zap.String("startup_time", time.Now().Format(time.RFC3339)),
		zap.Object("server_config", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			enc.AddString("host", cfg.Server.Host)
			enc.AddInt("port", cfg.Server.Port)
			enc.AddDuration("read_timeout", cfg.Server.ReadTimeout)
			enc.AddDuration("write_timeout", cfg.Server.WriteTimeout)
			return nil
		})),
	)
}
