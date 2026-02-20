package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestWarningF(t *testing.T) {
	output := captureOutput(func() {
		WarningF("warning %d: %s", 1, "test")
	})
	if !strings.Contains(output, "warning 1: test") {
		t.Errorf("Expected formatted warning, got: %s", output)
	}
	if !strings.Contains(output, "⚠️") {
		t.Errorf("Expected warning emoji, got: %s", output)
	}
}

func TestErrorF(t *testing.T) {
	output := captureOutput(func() {
		ErrorF("error %d: %s", 404, "not found")
	})
	if !strings.Contains(output, "error 404: not found") {
		t.Errorf("Expected formatted error, got: %s", output)
	}
	if !strings.Contains(output, "❌") {
		t.Errorf("Expected error emoji, got: %s", output)
	}
}

func TestInfoF(t *testing.T) {
	output := captureOutput(func() {
		InfoF("info: %s=%d", "count", 42)
	})
	if !strings.Contains(output, "info: count=42") {
		t.Errorf("Expected formatted info, got: %s", output)
	}
	if !strings.Contains(output, "ℹ️") {
		t.Errorf("Expected info emoji, got: %s", output)
	}
}

func TestSuccessF(t *testing.T) {
	output := captureOutput(func() {
		SuccessF("completed %d tasks", 5)
	})
	if !strings.Contains(output, "completed 5 tasks") {
		t.Errorf("Expected formatted success, got: %s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("Expected success emoji, got: %s", output)
	}
}

func TestFatalError(t *testing.T) {
	// FatalError calls os.Exit, so we can only test nil case
	t.Run("nil error does not exit", func(t *testing.T) {
		// This should not panic or exit
		FatalError(nil)
	})
}

func TestFatal(t *testing.T) {
	// Fatal calls os.Exit(1), so we cannot test it directly in a unit test
	// We can only verify the format by testing the underlying Error function
	t.Run("uses correct format", func(t *testing.T) {
		output := captureOutput(func() {
			Error(fmt.Sprintf("fatal: %s", "test"))
		})
		if !strings.Contains(output, "fatal: test") {
			t.Errorf("Expected formatted message, got: %s", output)
		}
	})
}
