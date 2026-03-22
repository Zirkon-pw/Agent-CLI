package logging

import (
	. "github.com/docup/agentctl/internal/infra/logging"
	"log/slog"
	"testing"
)

func TestSetup_Default(t *testing.T) {
	Setup(false)
	if !slog.Default().Enabled(nil, slog.LevelInfo) {
		t.Error("info level should be enabled")
	}
}

func TestSetup_Verbose(t *testing.T) {
	Setup(true)
	if !slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("debug level should be enabled in verbose mode")
	}
}

func TestSetup_NonVerbose_NoDebug(t *testing.T) {
	Setup(false)
	if slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("debug level should NOT be enabled in non-verbose mode")
	}
}
