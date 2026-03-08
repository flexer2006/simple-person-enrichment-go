package utilities

import (
	"context"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/zap"
)

type LoggableConfig interface {
	LogFields() []zap.Field
}

type LoadOptions struct {
	ConfigPath string
}

func Load[T any](ctx context.Context, opts ...LoadOptions) (*T, error) {
	Info(ctx, "loading configuration")

	var cfg T

	var options LoadOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.ConfigPath != "" {
		if _, err := os.Stat(options.ConfigPath); err == nil {
			if err := cleanenv.ReadConfig(options.ConfigPath, &cfg); err != nil {
				Error(ctx, "failed to load configuration", zap.Error(err), zap.String("path", options.ConfigPath))
				return nil, fmt.Errorf("%s from file %s: %w", "failed to load configuration", options.ConfigPath, err)
			}
		}
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		Error(ctx, "failed to load configuration", zap.Error(err))
		return nil, fmt.Errorf("%s from environment: %w", "failed to load configuration", err)
	}

	if loggable, ok := any(&cfg).(LoggableConfig); ok {
		Info(ctx, "configuration loaded successfully", loggable.LogFields()...)
	} else {
		Info(ctx, "configuration loaded successfully")
	}

	return &cfg, nil
}
