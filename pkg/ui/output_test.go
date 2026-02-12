package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/braunmar/worktree/pkg/config"
)

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSuccess(t *testing.T) {
	output := captureOutput(func() {
		Success("test message")
	})
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("Expected output to contain checkmark emoji, got: %s", output)
	}
}

func TestError(t *testing.T) {
	output := captureOutput(func() {
		Error("error message")
	})
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected output to contain 'error message', got: %s", output)
	}
	if !strings.Contains(output, "❌") {
		t.Errorf("Expected output to contain cross emoji, got: %s", output)
	}
}

func TestWarning(t *testing.T) {
	output := captureOutput(func() {
		Warning("warning message")
	})
	if !strings.Contains(output, "warning message") {
		t.Errorf("Expected output to contain 'warning message', got: %s", output)
	}
	if !strings.Contains(output, "⚠️") {
		t.Errorf("Expected output to contain warning emoji, got: %s", output)
	}
}

func TestInfo(t *testing.T) {
	output := captureOutput(func() {
		Info("info message")
	})
	if !strings.Contains(output, "info message") {
		t.Errorf("Expected output to contain 'info message', got: %s", output)
	}
	if !strings.Contains(output, "ℹ️") {
		t.Errorf("Expected output to contain info emoji, got: %s", output)
	}
}

func TestSection(t *testing.T) {
	output := captureOutput(func() {
		Section("Test Section")
	})
	if !strings.Contains(output, "Test Section") {
		t.Errorf("Expected output to contain 'Test Section', got: %s", output)
	}
	if !strings.Contains(output, "»") {
		t.Errorf("Expected output to contain section marker, got: %s", output)
	}
}

func TestRocket(t *testing.T) {
	output := captureOutput(func() {
		Rocket("launching")
	})
	if !strings.Contains(output, "launching") {
		t.Errorf("Expected output to contain 'launching', got: %s", output)
	}
	if !strings.Contains(output, "🚀") {
		t.Errorf("Expected output to contain rocket emoji, got: %s", output)
	}
}

func TestLoading(t *testing.T) {
	output := captureOutput(func() {
		Loading("loading...")
	})
	if !strings.Contains(output, "loading...") {
		t.Errorf("Expected output to contain 'loading...', got: %s", output)
	}
	if !strings.Contains(output, "⏳") {
		t.Errorf("Expected output to contain hourglass emoji, got: %s", output)
	}
}

func TestCheckMark(t *testing.T) {
	output := captureOutput(func() {
		CheckMark("done")
	})
	if !strings.Contains(output, "done") {
		t.Errorf("Expected output to contain 'done', got: %s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("Expected output to contain checkmark, got: %s", output)
	}
}

func TestCrossMark(t *testing.T) {
	output := captureOutput(func() {
		CrossMark("failed")
	})
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected output to contain 'failed', got: %s", output)
	}
	if !strings.Contains(output, "❌") {
		t.Errorf("Expected output to contain cross, got: %s", output)
	}
}

func TestBold(t *testing.T) {
	result := Bold("test")
	// Bold wraps with ANSI codes, so just check it's not empty
	if result == "" {
		t.Error("Expected Bold to return non-empty string")
	}
}

func TestShowPortsFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		portConfigs map[string]config.EnvVarConfig
		ports       map[string]int
		expectEmpty bool
	}{
		{
			name:        "empty config shows fallback",
			portConfigs: map[string]config.EnvVarConfig{},
			ports:       map[string]int{},
			expectEmpty: false, // Should show "Instance N configured"
		},
		{
			name: "valid port config shows services",
			portConfigs: map[string]config.EnvVarConfig{
				"APP_PORT": {
					Name: "Backend API",
					URL:  "http://{host}:{port}",
				},
			},
			ports: map[string]int{
				"APP_PORT": 8080,
			},
			expectEmpty: false,
		},
		{
			name: "skips entries without name",
			portConfigs: map[string]config.EnvVarConfig{
				"APP_PORT": {
					Name: "",
					URL:  "http://{host}:{port}",
				},
			},
			ports: map[string]int{
				"APP_PORT": 8080,
			},
			expectEmpty: false,
		},
		{
			name: "skips entries without URL",
			portConfigs: map[string]config.EnvVarConfig{
				"APP_PORT": {
					Name: "Backend API",
					URL:  "",
				},
			},
			ports: map[string]int{
				"APP_PORT": 8080,
			},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				ShowPortsFromConfig("localhost", 1, tt.ports, tt.portConfigs)
			})
			if output == "" && !tt.expectEmpty {
				t.Errorf("Expected non-empty output for %s", tt.name)
			}
		})
	}
}

func TestPrintHeader(t *testing.T) {
	output := captureOutput(func() {
		PrintHeader("Test Header")
	})
	if !strings.Contains(output, "Test Header") {
		t.Errorf("Expected output to contain 'Test Header', got: %s", output)
	}
}

func TestPrintStep(t *testing.T) {
	output := captureOutput(func() {
		PrintStep(1, "First step")
	})
	if !strings.Contains(output, "First step") {
		t.Errorf("Expected output to contain 'First step', got: %s", output)
	}
	if !strings.Contains(output, "1.") {
		t.Errorf("Expected output to contain step number, got: %s", output)
	}
}

func TestPrintCommand(t *testing.T) {
	output := captureOutput(func() {
		PrintCommand("make build")
	})
	if !strings.Contains(output, "make build") {
		t.Errorf("Expected output to contain 'make build', got: %s", output)
	}
}

func TestPrintStatusLine(t *testing.T) {
	output := captureOutput(func() {
		PrintStatusLine("Status", "Running")
	})
	if !strings.Contains(output, "Status") {
		t.Errorf("Expected output to contain 'Status', got: %s", output)
	}
	if !strings.Contains(output, "Running") {
		t.Errorf("Expected output to contain 'Running', got: %s", output)
	}
}

func TestPrintTable(t *testing.T) {
	output := captureOutput(func() {
		PrintTable("Key", "Value")
	})
	if !strings.Contains(output, "Key") {
		t.Errorf("Expected output to contain 'Key', got: %s", output)
	}
	if !strings.Contains(output, "Value") {
		t.Errorf("Expected output to contain 'Value', got: %s", output)
	}
}

func TestNewLine(t *testing.T) {
	output := captureOutput(func() {
		NewLine()
	})
	if output != "\n" {
		t.Errorf("Expected single newline, got: %q", output)
	}
}

func TestProgress(t *testing.T) {
	output := captureOutput(func() {
		Progress(1, 3, "processing")
	})
	if !strings.Contains(output, "processing") {
		t.Errorf("Expected output to contain 'processing', got: %s", output)
	}
	if !strings.Contains(output, "(1/3)") {
		t.Errorf("Expected output to contain progress counts, got: %s", output)
	}
}

func TestProgressWithName(t *testing.T) {
	output := captureOutput(func() {
		ProgressWithName(2, 5, "backend", "building")
	})
	if !strings.Contains(output, "building") {
		t.Errorf("Expected output to contain 'building', got: %s", output)
	}
	if !strings.Contains(output, "backend") {
		t.Errorf("Expected output to contain 'backend', got: %s", output)
	}
	if !strings.Contains(output, "(2/5)") {
		t.Errorf("Expected output to contain progress counts, got: %s", output)
	}
}
