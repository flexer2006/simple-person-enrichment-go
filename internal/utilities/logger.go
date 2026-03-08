package utilities

import (
	"context"
	"os"
	"regexp"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel = zapcore.Level

const (
	DebugLevel LogLevel = zapcore.DebugLevel
	InfoLevel  LogLevel = zapcore.InfoLevel
	WarnLevel  LogLevel = zapcore.WarnLevel
	ErrorLevel LogLevel = zapcore.ErrorLevel
	FatalLevel LogLevel = zapcore.FatalLevel
)

type Logger struct {
	*zap.Logger
}

var (
	global   *Logger
	once     sync.Once
	globalMu sync.RWMutex
)

func Global() *Logger {
	globalMu.RLock()
	lg := global
	globalMu.RUnlock()
	if lg != nil {
		return lg
	}

	once.Do(func() {
		l, err := NewProduction()
		if err != nil {
			l = NewConsole(InfoLevel, true)
		}
		globalMu.Lock()
		global = l
		globalMu.Unlock()
	})

	return global
}

func SetGlobal(l *Logger) {
	if l == nil {
		return
	}
	globalMu.Lock()
	global = l
	globalMu.Unlock()
}

func NewDevelopment() (*Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{zapLogger}, nil
}

func NewProduction() (*Logger, error) {
	cfg := zap.NewProductionConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{zapLogger}, nil
}

func NewConsole(level LogLevel, json bool) *Logger {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var enc zapcore.Encoder
	if json {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	atomic := zap.NewAtomicLevelAt(zapcore.Level(level))
	core := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), atomic)
	return &Logger{zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))}
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	if l == nil {
		return l
	}
	return &Logger{l.Logger.With(fields...)}
}

func (l *Logger) Sync() error {
	if l == nil || l.Logger == nil {
		return nil
	}
	return l.Logger.Sync()
}

type ctxKey string

const (
	loggerKey    ctxKey = "logger"
	requestIDKey ctxKey = "request_id"
)

func WithLogger(ctx context.Context, log *Logger) context.Context {
	if ctx == nil || log == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, log)
}

func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(loggerKey); v != nil {
		if l, ok := v.(*Logger); ok {
			return l
		}
	}
	return nil
}

func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	l := loggerFromContext(ctx)
	if l == nil {
		return ctx
	}
	return WithLogger(ctx, l.With(fields...))
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func WithRequestID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id == "" || !IsValidUUID(id) {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

func IsValidUUID(id string) bool {
	return id != "" && uuidRegex.MatchString(id)
}

func GenerateRequestID() string {
	return uuid.New().String()
}

func loggerFromContext(ctx context.Context) *Logger {
	l := FromContext(ctx)
	if l == nil {
		l = Global()
	}
	if id, ok := RequestID(ctx); ok && id != "" {
		return l.With(zap.String(string(requestIDKey), id))
	}
	return l
}

func Log(ctx context.Context, level LogLevel, msg string, fields ...zap.Field) {
	l := loggerFromContext(ctx)
	if l == nil {
		return
	}
	switch level {
	case DebugLevel:
		l.Debug(msg, fields...)
	case InfoLevel:
		l.Info(msg, fields...)
	case WarnLevel:
		l.Warn(msg, fields...)
	case ErrorLevel:
		l.Error(msg, fields...)
	case FatalLevel:
		l.Fatal(msg, fields...)
	default:
		l.Info(msg, fields...)
	}
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	Log(ctx, DebugLevel, msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) { Log(ctx, InfoLevel, msg, fields...) }
func Warn(ctx context.Context, msg string, fields ...zap.Field) { Log(ctx, WarnLevel, msg, fields...) }

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	Log(ctx, ErrorLevel, msg, fields...)
}

func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	Log(ctx, FatalLevel, msg, fields...)
}
