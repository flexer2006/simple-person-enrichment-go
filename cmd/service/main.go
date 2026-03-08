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
	initialLogger := utilities.NewConsole(utilities.InfoLevel, true)
	utilities.SetGlobal(initialLogger)

	ctx := context.Background()
	var exitCode int

	func() {
		defer func() {
			if err := initialLogger.Sync(); err != nil {
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
		}()
		var cfgPath = "./deploy/.env"
		cfg, err := utilities.Load[domain.Config](ctx, utilities.LoadOptions{
			ConfigPath: cfgPath,
		})
		if err != nil {
			utilities.Error(ctx, "failed to load configuration", zap.Error(err))
			exitCode = 1
			return
		}

		var finalLogger *utilities.Logger
		switch cfg.Logger.Model {
		case "development":
			finalLogger, err = utilities.NewDevelopment()
		case "production":
			finalLogger, err = utilities.NewProduction()
		default:
			utilities.Warn(ctx, "unknown logger model, using development", zap.String("model", cfg.Logger.Model))
			finalLogger, err = utilities.NewDevelopment()
		}

		if err != nil {
			utilities.Error(ctx, "failed to initialize logger with config", zap.Error(err))
			exitCode = 1
			return
		}

		utilities.SetGlobal(finalLogger)
		var defaultTimeout = 5 * time.Second
		shutdownTimeout, err := time.ParseDuration(cfg.Graceful.ShutdownTimeout)
		if err != nil {
			utilities.Error(ctx, "invalid graceful shutdown timeout", zap.Error(err))
			shutdownTimeout = defaultTimeout
		}

		dbConfig := database.Config{
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
		}

		utilities.Info(ctx, "initializing database")
		data, err := database.New(ctx, dbConfig)
		if err != nil {
			utilities.Error(ctx, "failed to initialize database", zap.Error(err))
			exitCode = 1
			return
		}

		if err := data.Ping(ctx); err != nil {
			utilities.Error(ctx, "database ping failed", zap.Error(err))
			exitCode = 1
			return
		}

		version, dirty, err := data.GetMigrationVersion(ctx)
		if err != nil {
			utilities.Warn(ctx, "failed to get migration version", zap.Error(err))
		} else {
			if dirty {
				utilities.Warn(ctx, "database has dirty migration", zap.Uint("version", version))
			} else {
				utilities.Info(ctx, "current migration version", zap.Uint("version", version))
			}
		}

		utilities.Info(ctx, "database initialized successfully")

		application, err := app.NewApplication(ctx, cfg, data.Provider(), nil)
		if err != nil {
			utilities.Error(ctx, "failed to initialize application", zap.Error(err))
			exitCode = 1
			return
		}

		appCtx, appCancel := context.WithCancel(ctx)
		defer appCancel()

		go func() {
			if err := application.Start(appCtx); err != nil {
				utilities.Error(ctx, "application stopped with error", zap.Error(err))
				exitCode = 1
				appCancel()
			}
		}()

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

		if err := utilities.Wait(ctx, shutdownTimeout,
			func(ctx context.Context) error {
				appCancel()
				return application.Stop(ctx)
			},
			func(ctx context.Context) error {
				utilities.Info(ctx, "closing database connection")
				data.Close(ctx)
				return nil
			},
		); err != nil {
			utilities.Error(ctx, "shutdown hooks returned error", zap.Error(err))
		}
		utilities.Info(ctx, "service shutdown complete")
	}()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
