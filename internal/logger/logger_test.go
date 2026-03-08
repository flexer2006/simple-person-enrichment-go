package logger

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGlobalAndSet(t *testing.T) {
	globalMu.Lock()
	global = nil
	globalMu.Unlock()

	first := Global()
	if first == nil {
		t.Fatal("expected non-nil global logger")
	}

	second := Global()
	if first != second {
		t.Error("Global() should return the same instance")
	}

	alt, err := NewDevelopment()
	if err != nil {
		t.Fatalf("NewDevelopment failed: %v", err)
	}
	SetGlobal(alt)
	if Global() != alt {
		t.Error("SetGlobal did not override")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	l := NewConsole(InfoLevel, false)
	ctx2 := WithLogger(ctx, l)
	if FromContext(ctx2) != l {
		t.Error("FromContext did not retrieve logger")
	}

	ctx3 := WithFields(ctx2, zap.String("foo", "bar"))
	l2 := FromContext(ctx3)
	if l2 == l {
		t.Error("WithFields should produce new logger instance")
	}
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "")
	id, ok := RequestID(ctx)
	if !ok || id == "" {
		t.Error("RequestID should be present and non-empty")
	}
	if !IsValidUUID(id) {
		t.Error("generated id is not valid uuid")
	}

	manual := "123e4567-e89b-12d3-a456-426614174000"
	ctx2 := WithRequestID(ctx, manual)
	got, ok2 := RequestID(ctx2)
	if !ok2 || got != manual {
		t.Error("WithRequestID should preserve valid id")
	}
}

func TestLoggerFromContextAddsRequestID(t *testing.T) {
	buf := &strings.Builder{}
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zap.NewAtomicLevelAt(zapcore.DebugLevel))
	custom := &Logger{zap.New(core)}

	ctx := context.Background()
	ctx = WithLogger(ctx, custom)
	ctx = WithRequestID(ctx, "")

	Info(ctx, "hello")
	out := buf.String()
	if !strings.Contains(out, "request_id") {
		t.Error("logged output should contain request_id field")
	}
}
