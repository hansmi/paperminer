package core

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zapcore"
)

func TestNewLoggingSetup(t *testing.T) {
	got, err := NewLoggingSetup()

	if err != nil {
		t.Errorf("NewLoggingSetup() failed: %v", err)
	}

	if name := got.Logger().Name(); name != "" {
		t.Errorf("Non-empty name %q", name)
	}

	if diff := cmp.Diff(zapcore.InfoLevel, got.Logger().Level()); diff != "" {
		t.Errorf("Level diff (-want +got):\n%s", diff)
	}

	got.SetLevel(zapcore.DebugLevel)

	if diff := cmp.Diff(zapcore.DebugLevel, got.Logger().Level()); diff != "" {
		t.Errorf("Level diff (-want +got):\n%s", diff)
	}
}
