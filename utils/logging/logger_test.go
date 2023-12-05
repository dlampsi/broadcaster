package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func Test_NewLogger(t *testing.T) {
	t.Parallel()

	logger := NewLogger("", true)
	if logger == nil {
		t.Fatal("expected logger to never be nil")
	}
}

func Test_Context(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger1 := FromContext(ctx)
	if logger1 == nil {
		t.Fatal("expected logger to never be nil")
	}

	ctx = ContextWithLogger(ctx, logger1)

	logger2 := FromContext(ctx)
	if logger1 != logger2 {
		t.Errorf("expected %#v to be %#v", logger1, logger2)
	}
}

func Test_toZapLevel(t *testing.T) {
	t.Parallel()

	f := func(level string, expected zapcore.Level) {
		t.Helper()
		resp := toZapLevel(level)
		require.Equal(t, expected, resp)
	}

	f("debug", zapcore.DebugLevel)
	f("info", zapcore.InfoLevel)
	f("warning", zapcore.WarnLevel)
	f("error", zapcore.ErrorLevel)
	f("fatal", zapcore.FatalLevel)
	f("fakeone", zapcore.InfoLevel)
}
