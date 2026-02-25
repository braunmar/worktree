package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCalculatePort_ValidExpressions tests valid port calculation expressions
func TestCalculatePort_ValidExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		instance   int
		want       int
		wantErr    bool
	}{
		// Simple addition with {instance}
		{
			name:       "3000 + {instance} with instance 0",
			expression: "3000 + {instance}",
			instance:   0,
			want:       3000,
			wantErr:    false,
		},
		{
			name:       "3000 + {instance} with instance 1",
			expression: "3000 + {instance}",
			instance:   1,
			want:       3001,
			wantErr:    false,
		},
		{
			name:       "3000 + {instance} with instance 2",
			expression: "3000 + {instance}",
			instance:   2,
			want:       3002,
			wantErr:    false,
		},
		// Complex expression with multiplication
		{
			name:       "4510 + {instance} * 50 with instance 0",
			expression: "4510 + {instance} * 50",
			instance:   0,
			want:       4510,
			wantErr:    false,
		},
		{
			name:       "4510 + {instance} * 50 with instance 1",
			expression: "4510 + {instance} * 50",
			instance:   1,
			want:       4560,
			wantErr:    false,
		},
		{
			name:       "4510 + {instance} * 50 with instance 2",
			expression: "4510 + {instance} * 50",
			instance:   2,
			want:       4610,
			wantErr:    false,
		},
		// Static port (no placeholder)
		{
			name:       "8080 static port",
			expression: "8080",
			instance:   0,
			want:       8080,
			wantErr:    false,
		},
		{
			name:       "8080 static port with non-zero instance",
			expression: "8080",
			instance:   5,
			want:       8080,
			wantErr:    false,
		},
		// Edge case: instance = 0
		{
			name:       "edge case instance 0",
			expression: "5000 + {instance}",
			instance:   0,
			want:       5000,
			wantErr:    false,
		},
		// Large instance numbers
		{
			name:       "large instance number",
			expression: "3000 + {instance}",
			instance:   100,
			want:       3100,
			wantErr:    false,
		},
		{
			name:       "large instance with multiplication",
			expression: "5000 + {instance} * 100",
			instance:   50,
			want:       10000,
			wantErr:    false,
		},
		// Extra whitespace handling
		{
			name:       "whitespace in expression",
			expression: "  3000  +  {instance}  ",
			instance:   1,
			want:       3001,
			wantErr:    false,
		},
		{
			name:       "whitespace in complex expression",
			expression: "  4510  +  {instance}  *  50  ",
			instance:   2,
			want:       4610,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePort(tt.expression, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculatePort(%q, %d) error = %v, wantErr %v", tt.expression, tt.instance, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CalculatePort(%q, %d) = %d, want %d", tt.expression, tt.instance, got, tt.want)
			}
		})
	}
}

// TestCalculatePort_InvalidExpressions tests invalid or edge case expressions
func TestCalculatePort_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		instance   int
		want       int  // Expected return value (likely 0 for invalid expressions)
		wantErr    bool // Whether an error is expected
	}{
		{
			name:       "empty string",
			expression: "",
			instance:   0,
			want:       0,
			wantErr:    true,
		},
		{
			name:       "invalid string",
			expression: "invalid",
			instance:   0,
			want:       0,
			wantErr:    true,
		},
		{
			name:       "double addition operator",
			expression: "3000 + + 1000",
			instance:   0,
			want:       0,
			wantErr:    true, // Invalid expression format
		},
		{
			name:       "non-numeric value",
			expression: "abc + {instance}",
			instance:   0,
			want:       0,
			wantErr:    true,
		},
		{
			name:       "only placeholder",
			expression: "{instance}",
			instance:   5,
			want:       5,
			wantErr:    false,
		},
		{
			name:       "negative base port",
			expression: "-1000 + {instance}",
			instance:   0,
			want:       0,
			wantErr:    true, // Should fail validation
		},
		{
			name:       "port exceeding valid range",
			expression: "70000 + {instance}",
			instance:   0,
			want:       0,
			wantErr:    true, // Should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePort(tt.expression, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculatePort(%q, %d) error = %v, wantErr %v", tt.expression, tt.instance, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CalculatePort(%q, %d) = %d, want %d", tt.expression, tt.instance, got, tt.want)
			}
		})
	}
}

// TestCalculatePort_PortRangeValidation tests port range boundaries
func TestCalculatePort_PortRangeValidation(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		instance   int
		want       int
		wantErr    bool
	}{
		{
			name:       "port less than 1",
			expression: "0 + {instance}",
			instance:   0,
			want:       0,
			wantErr:    true, // Should error on invalid port
		},
		{
			name:       "minimum valid port",
			expression: "1 + {instance}",
			instance:   0,
			want:       1,
			wantErr:    false,
		},
		{
			name:       "maximum valid port",
			expression: "65535 + {instance}",
			instance:   0,
			want:       65535,
			wantErr:    false,
		},
		{
			name:       "port exceeding maximum",
			expression: "65535 + {instance}",
			instance:   1,
			want:       0,
			wantErr:    true, // Should error on invalid port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePort(tt.expression, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculatePort(%q, %d) error = %v, wantErr %v", tt.expression, tt.instance, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CalculatePort(%q, %d) = %d, want %d", tt.expression, tt.instance, got, tt.want)
			}
		})
	}
}

// TestCalculatePort_Overflow tests potential integer overflow scenarios
func TestCalculatePort_Overflow(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		instance   int
		want       int
		wantErr    bool
	}{
		{
			name:       "very large multiplication exceeds port range",
			expression: "10000 + {instance} * 1000",
			instance:   100,
			want:       0,
			wantErr:    true, // 110000 exceeds valid port range
		},
		{
			name:       "potential overflow with max int instance",
			expression: "1000 + {instance}",
			instance:   math.MaxInt32,
			want:       0,
			wantErr:    true, // Result will exceed valid port range
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePort(tt.expression, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculatePort(%q, %d) error = %v, wantErr %v", tt.expression, tt.instance, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CalculatePort(%q, %d) = %d, want %d", tt.expression, tt.instance, got, tt.want)
			}
		})
	}
}

// TestExtractBasePort tests base port extraction from expressions
// Note: ExtractBasePort replaces {instance} with 0 before parsing
func TestExtractBasePort(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       int
		wantErr    bool
	}{
		{
			name:       "simple addition expression with placeholder",
			expression: "3000 + {instance}",
			want:       3000,
			wantErr:    false,
		},
		{
			name:       "complex multiplication expression with placeholder",
			expression: "4510 + {instance} * 50",
			want:       4510,
			wantErr:    false,
		},
		{
			name:       "simple addition expression numeric",
			expression: "3000 + 0",
			want:       3000,
			wantErr:    false,
		},
		{
			name:       "static port",
			expression: "8080",
			want:       8080,
			wantErr:    false,
		},
		{
			name:       "with whitespace",
			expression: "  5000  +  {instance}  ",
			want:       5000,
			wantErr:    false,
		},
		{
			name:       "empty string",
			expression: "",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "invalid expression",
			expression: "invalid",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "just a zero",
			expression: "0",
			want:       0,
			wantErr:    true, // Base port 0 is invalid
		},
		{
			name:       "placeholder only becomes zero",
			expression: "{instance}",
			want:       0,
			wantErr:    true, // Base port 0 is invalid
		},
		{
			name:       "base port above valid range",
			expression: "70000",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "negative base port",
			expression: "-100 + {instance}",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "valid minimum port",
			expression: "1",
			want:       1,
			wantErr:    false,
		},
		{
			name:       "valid maximum port",
			expression: "65535",
			want:       65535,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBasePort(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBasePort(%q) error = %v, wantErr %v", tt.expression, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ExtractBasePort(%q) = %d, want %d", tt.expression, got, tt.want)
			}
		})
	}
}

// TestExportEnvVars tests environment variable export functionality
func TestExportEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		config   *WorktreeConfig
		instance int
		want     map[string]string
	}{
		{
			name: "basic port configs",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"FE_PORT": {
						Name: "Frontend",
						Port: "3000 + {instance}",
						Env:  "FE_PORT",
					},
					"BE_PORT": {
						Name: "Backend",
						Port: "8080 + {instance}",
						Env:  "BE_PORT",
					},
				},
			},
			instance: 1,
			want: map[string]string{
				"INSTANCE": "1",
				"FE_PORT":  "3001",
				"BE_PORT":  "8081",
			},
		},
		{
			name: "complex port calculation",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"PG_PORT": {
						Name: "PostgreSQL",
						Port: "5432 + {instance} * 10",
						Env:  "PG_PORT",
					},
				},
			},
			instance: 2,
			want: map[string]string{
				"INSTANCE": "2",
				"PG_PORT":  "5452",
			},
		},
		{
			name: "static port",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"REDIS_PORT": {
						Name: "Redis",
						Port: "6379",
						Env:  "REDIS_PORT",
					},
				},
			},
			instance: 5,
			want: map[string]string{
				"INSTANCE":   "5",
				"REDIS_PORT": "6379",
			},
		},
		{
			name: "string template (non-port value)",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"COMPOSE_PROJECT_NAME": {
						Value: "myproject-inst{instance}",
						Env:   "COMPOSE_PROJECT_NAME",
					},
				},
			},
			instance: 3,
			want: map[string]string{
				"INSTANCE":             "3",
				"COMPOSE_PROJECT_NAME": "myproject-inst3",
			},
		},
		{
			name: "mixed port and value configs",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"FE_PORT": {
						Name: "Frontend",
						Port: "3000 + {instance}",
						Env:  "FE_PORT",
					},
					"COMPOSE_PROJECT_NAME": {
						Value: "proj-{instance}",
						Env:   "COMPOSE_PROJECT_NAME",
					},
				},
			},
			instance: 0,
			want: map[string]string{
				"INSTANCE":             "0",
				"FE_PORT":              "3000",
				"COMPOSE_PROJECT_NAME": "proj-0",
			},
		},
		{
			name: "port config without env var",
			config: &WorktreeConfig{
				EnvVariables: map[string]EnvVarConfig{
					"FE_PORT": {
						Name: "Frontend",
						Port: "3000 + {instance}",
						Env:  "FE_PORT",
					},
					"INTERNAL_PORT": {
						Name: "Internal",
						Port: "9000 + {instance}",
						Env:  "", // No env var
					},
				},
			},
			instance: 1,
			want: map[string]string{
				"INSTANCE": "1",
				"FE_PORT":  "3001",
				// INTERNAL_PORT not exported
			},
		},
		{
			name:     "empty config",
			config:   &WorktreeConfig{},
			instance: 0,
			want: map[string]string{
				"INSTANCE": "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ExportEnvVars(tt.instance)

			// Check all expected vars are present with correct values
			for key, wantVal := range tt.want {
				gotVal, exists := got[key]
				if !exists {
					t.Errorf("ExportEnvVars() missing key %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("ExportEnvVars() key %q = %q, want %q", key, gotVal, wantVal)
				}
			}

			// Check no unexpected vars are present
			for key := range got {
				if _, expected := tt.want[key]; !expected {
					t.Errorf("ExportEnvVars() unexpected key %q = %q", key, got[key])
				}
			}
		})
	}
}

// TestExportEnvVars_InstanceAlwaysPresent verifies INSTANCE is always exported
func TestExportEnvVars_InstanceAlwaysPresent(t *testing.T) {
	configs := []*WorktreeConfig{
		{}, // Empty config
		{EnvVariables: map[string]EnvVarConfig{}}, // Empty ports
		{EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {Port: "3000", Env: "FE_PORT"},
		}},
	}

	instances := []int{0, 1, 5, 100}

	for configIdx, config := range configs {
		for _, instance := range instances {
			t.Run(fmt.Sprintf("config_%d_instance_%d", configIdx, instance), func(t *testing.T) {
				envVars := config.ExportEnvVars(instance)

				instanceVal, exists := envVars["INSTANCE"]
				if !exists {
					t.Error("INSTANCE environment variable not found")
					return
				}

				expectedVal := fmt.Sprintf("%d", instance)
				if instanceVal != expectedVal {
					t.Errorf("INSTANCE = %q, want %q", instanceVal, expectedVal)
				}
			})
		}
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *WorktreeConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &WorktreeConfig{
				Projects: map[string]ProjectConfig{
					"frontend": {Dir: "frontend"},
				},
				Presets: map[string]PresetConfig{
					"default": {Projects: []string{"frontend"}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing projects",
			config: &WorktreeConfig{
				Projects: map[string]ProjectConfig{},
				Presets: map[string]PresetConfig{
					"default": {Projects: []string{}},
				},
			},
			wantErr: true,
		},
		{
			name: "missing presets",
			config: &WorktreeConfig{
				Projects: map[string]ProjectConfig{
					"frontend": {Dir: "frontend"},
				},
				Presets: map[string]PresetConfig{},
			},
			wantErr: true,
		},
		{
			name: "preset references undefined project",
			config: &WorktreeConfig{
				Projects: map[string]ProjectConfig{
					"frontend": {Dir: "frontend"},
				},
				Presets: map[string]PresetConfig{
					"default": {Projects: []string{"backend"}}, // backend doesn't exist
				},
			},
			wantErr: true,
		},
		{
			name: "multiple projects and presets valid",
			config: &WorktreeConfig{
				Projects: map[string]ProjectConfig{
					"frontend": {Dir: "frontend"},
					"backend":  {Dir: "backend"},
				},
				Presets: map[string]PresetConfig{
					"default": {Projects: []string{"frontend", "backend"}},
					"fe-only": {Projects: []string{"frontend"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEnvVarConfig_GetValue tests the GetValue method
func TestEnvVarConfig_GetValue(t *testing.T) {
	tests := []struct {
		name     string
		portCfg  EnvVarConfig
		instance int
		hostname string
		envVars  map[string]string
		want     string
	}{
		{
			name: "port expression",
			portCfg: EnvVarConfig{
				Port: "3000 + {instance}",
			},
			instance: 1,
			hostname: "localhost",
			want:     "3001",
		},
		{
			name: "string value template",
			portCfg: EnvVarConfig{
				Value: "myproject-{instance}",
			},
			instance: 2,
			hostname: "localhost",
			want:     "myproject-2",
		},
		{
			name: "static port",
			portCfg: EnvVarConfig{
				Port: "8080",
			},
			instance: 0,
			hostname: "localhost",
			want:     "8080",
		},
		{
			name:     "empty config",
			portCfg:  EnvVarConfig{},
			instance: 0,
			hostname: "localhost",
			want:     "",
		},
		{
			name: "{host} substituted with hostname",
			portCfg: EnvVarConfig{
				Value: "http://{host}:{FE_PORT}/auth/callback",
			},
			instance: 0,
			hostname: "localhost",
			envVars:  map[string]string{"FE_PORT": "3005"},
			want:     "http://localhost:3005/auth/callback",
		},
		{
			name: "{host} substituted with custom hostname",
			portCfg: EnvVarConfig{
				Value: "http://{host}:{FE_PORT}/auth/callback",
			},
			instance: 0,
			hostname: "dev.myapp.internal",
			envVars:  map[string]string{"FE_PORT": "3006"},
			want:     "http://dev.myapp.internal:3006/auth/callback",
		},
		{
			name: "{host} in value with no port vars",
			portCfg: EnvVarConfig{
				Value: "http://{host}/healthz",
			},
			instance: 0,
			hostname: "staging.example.com",
			want:     "http://staging.example.com/healthz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := tt.envVars
			if envVars == nil {
				envVars = map[string]string{}
			}
			got := tt.portCfg.GetValue(tt.instance, envVars, tt.hostname)
			if got != tt.want {
				t.Errorf("GetValue(%d) = %q, want %q", tt.instance, got, tt.want)
			}
		})
	}
}

// TestEnvVarConfig_GetPortRange tests port range extraction
func TestEnvVarConfig_GetPortRange(t *testing.T) {
	tests := []struct {
		name    string
		portCfg EnvVarConfig
		want    *[2]int
	}{
		{
			name: "explicit range",
			portCfg: EnvVarConfig{
				Port:  "3000 + {instance}",
				Range: &[2]int{3000, 3100},
			},
			want: &[2]int{3000, 3100},
		},
		{
			name: "port expression with placeholder",
			portCfg: EnvVarConfig{
				Port: "5000 + {instance}",
			},
			want: &[2]int{5000, 5100}, // ExtractBasePort replaces {instance} with 0 and extracts base
		},
		{
			name: "static port",
			portCfg: EnvVarConfig{
				Port: "8080",
			},
			want: &[2]int{8080, 8180},
		},
		{
			name:    "no range available",
			portCfg: EnvVarConfig{},
			want:    nil,
		},
		{
			name: "explicit range takes precedence",
			portCfg: EnvVarConfig{
				Port:  "3000 + {instance}",
				Range: &[2]int{4000, 4500},
			},
			want: &[2]int{4000, 4500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.portCfg.GetPortRange()

			if tt.want == nil {
				if got != nil {
					t.Errorf("GetPortRange() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("GetPortRange() = nil, want %v", tt.want)
				return
			}

			if got[0] != tt.want[0] || got[1] != tt.want[1] {
				t.Errorf("GetPortRange() = [%d, %d], want [%d, %d]", got[0], got[1], tt.want[0], tt.want[1])
			}
		})
	}
}

// TestValidateProjectName tests project name validation
func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "myproject", false},
		{"valid with hyphens", "my-project-name", false},
		{"valid alphanumeric", "proj1", false},
		{"valid mixed case", "MyProject", false},
		{"empty string", "", true},
		{"starts with hyphen", "-project", true},
		{"ends with hyphen", "project-", true},
		{"contains underscore", "my_project", true},
		{"contains dot", "my.project", true},
		{"contains space", "my project", true},
		{"contains slash", "my/project", true},
		{"contains special char", "proj@name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestLoadWorktreeConfig tests loading config from a YAML file
func TestLoadWorktreeConfig(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		_, err := LoadWorktreeConfig("/nonexistent/path")
		if err == nil {
			t.Error("expected error for missing config file")
		}
	})

	t.Run("valid config loads correctly", func(t *testing.T) {
		dir := t.TempDir()
		content := `project_name: testproject
projects:
  backend:
    dir: backend
    main_branch: main
presets:
  default:
    projects: [backend]
default_preset: default
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadWorktreeConfig(dir)
		if err != nil {
			t.Fatalf("LoadWorktreeConfig() error = %v", err)
		}
		if cfg.ProjectName != "testproject" {
			t.Errorf("ProjectName = %q, want %q", cfg.ProjectName, "testproject")
		}
		if cfg.Hostname != "localhost" {
			t.Errorf("Hostname = %q, want %q (default)", cfg.Hostname, "localhost")
		}
	})

	t.Run("missing project_name returns error", func(t *testing.T) {
		dir := t.TempDir()
		content := `projects:
  backend:
    dir: backend
presets:
  default:
    projects: [backend]
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadWorktreeConfig(dir)
		if err == nil {
			t.Error("expected error for missing project_name")
		}
	})

	t.Run("invalid project_name returns error", func(t *testing.T) {
		dir := t.TempDir()
		content := `project_name: invalid_name
projects:
  backend:
    dir: backend
presets:
  default:
    projects: [backend]
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadWorktreeConfig(dir)
		if err == nil {
			t.Error("expected error for invalid project_name with underscore")
		}
	})

	t.Run("custom hostname is preserved", func(t *testing.T) {
		dir := t.TempDir()
		content := `project_name: myproj
hostname: myhost.local
projects:
  backend:
    dir: backend
presets:
  default:
    projects: [backend]
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadWorktreeConfig(dir)
		if err != nil {
			t.Fatalf("LoadWorktreeConfig() error = %v", err)
		}
		if cfg.Hostname != "myhost.local" {
			t.Errorf("Hostname = %q, want %q", cfg.Hostname, "myhost.local")
		}
	})
}

// TestProjectConfigPerProjectLinks tests per-project symlinks and copies parsing from YAML
func TestProjectConfigPerProjectLinks(t *testing.T) {
	t.Run("per-project symlinks and copies are parsed", func(t *testing.T) {
		dir := t.TempDir()
		content := `project_name: testproject
projects:
  backend:
    dir: backend
    symlinks:
      - source: ".env.backend.shared"
        target: ".env"
    copies:
      - source: "config/template.yml"
        target: "config/local.yml"
  frontend:
    dir: frontend
    symlinks:
      - source: ".env.frontend.shared"
        target: ".env.local"
presets:
  default:
    projects: [backend, frontend]
default_preset: default
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadWorktreeConfig(dir)
		if err != nil {
			t.Fatalf("LoadWorktreeConfig() error = %v", err)
		}

		backend := cfg.Projects["backend"]
		if len(backend.Symlinks) != 1 {
			t.Fatalf("backend.Symlinks: got %d, want 1", len(backend.Symlinks))
		}
		if backend.Symlinks[0].Source != ".env.backend.shared" {
			t.Errorf("backend.Symlinks[0].Source = %q, want %q", backend.Symlinks[0].Source, ".env.backend.shared")
		}
		if backend.Symlinks[0].Target != ".env" {
			t.Errorf("backend.Symlinks[0].Target = %q, want %q", backend.Symlinks[0].Target, ".env")
		}
		if len(backend.Copies) != 1 {
			t.Fatalf("backend.Copies: got %d, want 1", len(backend.Copies))
		}
		if backend.Copies[0].Source != "config/template.yml" {
			t.Errorf("backend.Copies[0].Source = %q, want %q", backend.Copies[0].Source, "config/template.yml")
		}

		frontend := cfg.Projects["frontend"]
		if len(frontend.Symlinks) != 1 {
			t.Fatalf("frontend.Symlinks: got %d, want 1", len(frontend.Symlinks))
		}
		if frontend.Symlinks[0].Target != ".env.local" {
			t.Errorf("frontend.Symlinks[0].Target = %q, want %q", frontend.Symlinks[0].Target, ".env.local")
		}
		if len(frontend.Copies) != 0 {
			t.Errorf("frontend.Copies: got %d, want 0", len(frontend.Copies))
		}
	})

	t.Run("projects without per-project links have empty slices", func(t *testing.T) {
		dir := t.TempDir()
		content := `project_name: testproject
projects:
  backend:
    dir: backend
presets:
  default:
    projects: [backend]
default_preset: default
`
		if err := os.WriteFile(filepath.Join(dir, ".worktree.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadWorktreeConfig(dir)
		if err != nil {
			t.Fatalf("LoadWorktreeConfig() error = %v", err)
		}

		backend := cfg.Projects["backend"]
		if len(backend.Symlinks) != 0 {
			t.Errorf("backend.Symlinks: got %d, want 0", len(backend.Symlinks))
		}
		if len(backend.Copies) != 0 {
			t.Errorf("backend.Copies: got %d, want 0", len(backend.Copies))
		}
	})
}

// TestGetPreset tests preset retrieval
func TestGetPreset(t *testing.T) {
	cfg := &WorktreeConfig{
		DefaultPreset: "default",
		Presets: map[string]PresetConfig{
			"default":  {Projects: []string{"backend", "frontend"}},
			"backend":  {Projects: []string{"backend"}},
			"frontend": {Projects: []string{"frontend"}},
		},
	}

	t.Run("get named preset", func(t *testing.T) {
		preset, err := cfg.GetPreset("backend")
		if err != nil {
			t.Fatalf("GetPreset() error = %v", err)
		}
		if len(preset.Projects) != 1 || preset.Projects[0] != "backend" {
			t.Errorf("GetPreset(\"backend\") = %v, want [backend]", preset.Projects)
		}
	})

	t.Run("empty name returns default preset", func(t *testing.T) {
		preset, err := cfg.GetPreset("")
		if err != nil {
			t.Fatalf("GetPreset(\"\") error = %v", err)
		}
		if len(preset.Projects) != 2 {
			t.Errorf("GetPreset(\"\") returned %d projects, want 2", len(preset.Projects))
		}
	})

	t.Run("nonexistent preset returns error", func(t *testing.T) {
		_, err := cfg.GetPreset("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent preset")
		}
	})
}

// TestGetProjectsForPreset tests project list retrieval for a preset
func TestGetProjectsForPreset(t *testing.T) {
	cfg := &WorktreeConfig{
		DefaultPreset: "fullstack",
		Projects: map[string]ProjectConfig{
			"backend":  {Dir: "backend"},
			"frontend": {Dir: "frontend"},
		},
		Presets: map[string]PresetConfig{
			"fullstack": {Projects: []string{"backend", "frontend"}},
			"beonly":    {Projects: []string{"backend"}},
		},
	}

	t.Run("returns projects for preset", func(t *testing.T) {
		projects, err := cfg.GetProjectsForPreset("beonly")
		if err != nil {
			t.Fatalf("GetProjectsForPreset() error = %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("expected 1 project, got %d", len(projects))
		}
		if projects[0].Dir != "backend" {
			t.Errorf("expected Dir=backend, got %q", projects[0].Dir)
		}
	})

	t.Run("returns all projects for fullstack preset", func(t *testing.T) {
		projects, err := cfg.GetProjectsForPreset("fullstack")
		if err != nil {
			t.Fatalf("GetProjectsForPreset() error = %v", err)
		}
		if len(projects) != 2 {
			t.Errorf("expected 2 projects, got %d", len(projects))
		}
	})

	t.Run("nonexistent preset returns error", func(t *testing.T) {
		_, err := cfg.GetProjectsForPreset("missing")
		if err == nil {
			t.Error("expected error for missing preset")
		}
	})
}

// TestGetURL tests URL generation from port config
func TestGetURL(t *testing.T) {
	tests := []struct {
		name     string
		cfg      EnvVarConfig
		hostname string
		port     int
		want     string
	}{
		{
			name:     "http URL template",
			cfg:      EnvVarConfig{URL: "http://{host}:{port}"},
			hostname: "localhost",
			port:     3000,
			want:     "http://localhost:3000",
		},
		{
			name:     "custom hostname",
			cfg:      EnvVarConfig{URL: "http://{host}:{port}/api"},
			hostname: "myhost.local",
			port:     8080,
			want:     "http://myhost.local:8080/api",
		},
		{
			name:     "no placeholders",
			cfg:      EnvVarConfig{URL: "http://fixed-url"},
			hostname: "localhost",
			port:     3000,
			want:     "http://fixed-url",
		},
		{
			name:     "empty URL",
			cfg:      EnvVarConfig{URL: ""},
			hostname: "localhost",
			port:     3000,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetURL(tt.hostname, tt.port)
			if got != tt.want {
				t.Errorf("GetURL(%q, %d) = %q, want %q", tt.hostname, tt.port, got, tt.want)
			}
		})
	}
}

// TestGetDisplayableServices tests displayable service retrieval
func TestGetDisplayableServices(t *testing.T) {
	cfg := &WorktreeConfig{
		Hostname: "localhost",
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {
				Name: "Frontend",
				URL:  "http://{host}:{port}",
				Port: "3000 + {instance}",
			},
			"BE_PORT": {
				Name: "Backend API",
				URL:  "http://{host}:{port}/api",
				Port: "8080 + {instance}",
			},
			"PG_PORT": {
				// No Name or URL - should not be displayed
				Port: "5432 + {instance}",
			},
		},
	}

	ports := map[string]int{
		"FE_PORT": 3001,
		"BE_PORT": 8081,
		"PG_PORT": 5433,
	}

	t.Run("returns only services with name and URL", func(t *testing.T) {
		services := cfg.GetDisplayableServices(ports)
		if _, ok := services["Frontend"]; !ok {
			t.Error("expected Frontend in displayable services")
		}
		if _, ok := services["Backend API"]; !ok {
			t.Error("expected Backend API in displayable services")
		}
		if len(services) != 2 {
			t.Errorf("expected 2 displayable services, got %d: %v", len(services), services)
		}
	})

	t.Run("correct URL generation", func(t *testing.T) {
		services := cfg.GetDisplayableServices(ports)
		if got := services["Frontend"]; got != "http://localhost:3001" {
			t.Errorf("Frontend URL = %q, want %q", got, "http://localhost:3001")
		}
		if got := services["Backend API"]; got != "http://localhost:8081/api" {
			t.Errorf("Backend API URL = %q, want %q", got, "http://localhost:8081/api")
		}
	})
}

// TestGetServiceURL tests per-service URL lookup
func TestGetServiceURL(t *testing.T) {
	cfg := &WorktreeConfig{
		Hostname: "localhost",
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {Name: "Frontend", URL: "http://{host}:{port}"},
			"NO_URL":  {Name: "No URL service"},
		},
	}

	ports := map[string]int{"FE_PORT": 3001}

	t.Run("returns URL for configured service", func(t *testing.T) {
		url := cfg.GetServiceURL("FE_PORT", ports)
		if url != "http://localhost:3001" {
			t.Errorf("GetServiceURL() = %q, want %q", url, "http://localhost:3001")
		}
	})

	t.Run("returns empty for service without URL", func(t *testing.T) {
		url := cfg.GetServiceURL("NO_URL", ports)
		if url != "" {
			t.Errorf("GetServiceURL(NO_URL) = %q, want empty", url)
		}
	})

	t.Run("returns empty for nonexistent service", func(t *testing.T) {
		url := cfg.GetServiceURL("MISSING", ports)
		if url != "" {
			t.Errorf("GetServiceURL(MISSING) = %q, want empty", url)
		}
	})

	t.Run("returns empty when port not allocated", func(t *testing.T) {
		url := cfg.GetServiceURL("FE_PORT", map[string]int{})
		if url != "" {
			t.Errorf("GetServiceURL() with missing port = %q, want empty", url)
		}
	})
}

// TestGetPortServiceNames tests port service name extraction
func TestGetPortServiceNames(t *testing.T) {
	r1 := [2]int{3000, 3100}
	r2 := [2]int{8080, 8180}

	cfg := &WorktreeConfig{
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {Env: "FE_PORT", Range: &r1},
			"BE_PORT": {Env: "BE_PORT", Range: &r2},
			"CALC":    {Env: "CALC", Port: "9000 + {instance}"}, // no Range - excluded
			"NOENV":   {Range: &r1},                             // no Env - excluded
			"TMPL":    {Env: "TMPL", Value: "proj-{instance}"},  // value only - excluded
		},
	}

	services := cfg.GetPortServiceNames()
	sort.Strings(services)

	if len(services) != 2 {
		t.Errorf("GetPortServiceNames() returned %d services, want 2: %v", len(services), services)
	}
	if services[0] != "BE_PORT" || services[1] != "FE_PORT" {
		t.Errorf("GetPortServiceNames() = %v, want [BE_PORT FE_PORT]", services)
	}
}

// TestGetComposeProjectTemplate tests compose project template retrieval
func TestGetComposeProjectTemplate(t *testing.T) {
	t.Run("returns configured template", func(t *testing.T) {
		cfg := &WorktreeConfig{
			EnvVariables: map[string]EnvVarConfig{
				"COMPOSE_PROJECT_NAME": {
					Value: "{project}-{feature}-{service}",
					Env:   "COMPOSE_PROJECT_NAME",
				},
			},
		}
		got := cfg.GetComposeProjectTemplate()
		if got != "{project}-{feature}-{service}" {
			t.Errorf("GetComposeProjectTemplate() = %q, want %q", got, "{project}-{feature}-{service}")
		}
	})

	t.Run("returns default when not configured", func(t *testing.T) {
		cfg := &WorktreeConfig{}
		got := cfg.GetComposeProjectTemplate()
		if got != "{project}-{feature}" {
			t.Errorf("GetComposeProjectTemplate() = %q, want %q", got, "{project}-{feature}")
		}
	})

	t.Run("returns default when value is empty", func(t *testing.T) {
		cfg := &WorktreeConfig{
			EnvVariables: map[string]EnvVarConfig{
				"COMPOSE_PROJECT_NAME": {Env: "COMPOSE_PROJECT_NAME"},
			},
		}
		got := cfg.GetComposeProjectTemplate()
		if got != "{project}-{feature}" {
			t.Errorf("GetComposeProjectTemplate() = %q, want %q", got, "{project}-{feature}")
		}
	})
}

// TestReplaceComposeProjectPlaceholders tests placeholder replacement in compose templates
func TestReplaceComposeProjectPlaceholders(t *testing.T) {
	cfg := &WorktreeConfig{ProjectName: "project"}

	tests := []struct {
		name        string
		template    string
		featureName string
		serviceName string
		want        string
	}{
		{
			name:        "all placeholders replaced",
			template:    "{project}-{feature}-{service}",
			featureName: "feature-user-auth",
			serviceName: "backend",
			want:        "project-feature-user-auth-backend",
		},
		{
			name:        "only project and feature",
			template:    "{project}-{feature}",
			featureName: "feature-reports",
			serviceName: "frontend",
			want:        "project-feature-reports",
		},
		{
			name:        "no placeholders",
			template:    "fixed-name",
			featureName: "anything",
			serviceName: "anything",
			want:        "fixed-name",
		},
		{
			name:        "service only",
			template:    "{service}-svc",
			featureName: "feature-x",
			serviceName: "api",
			want:        "api-svc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ReplaceComposeProjectPlaceholders(tt.template, tt.featureName, tt.serviceName)
			if got != tt.want {
				t.Errorf("ReplaceComposeProjectPlaceholders() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReplaceInstancePlaceholder tests {instance} substitution in commands
func TestReplaceInstancePlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		instance int
		want     string
	}{
		{"simple replacement", "docker run -p {instance}:80 nginx", 3, "docker run -p 3:80 nginx"},
		{"multiple occurrences", "{instance}-app-{instance}", 5, "5-app-5"},
		{"no placeholder", "docker compose up -d", 2, "docker compose up -d"},
		{"zero instance", "port-{instance}", 0, "port-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceInstancePlaceholder(tt.command, tt.instance)
			if got != tt.want {
				t.Errorf("ReplaceInstancePlaceholder(%q, %d) = %q, want %q", tt.command, tt.instance, got, tt.want)
			}
		})
	}
}

// TestGetClaudeWorkingProject tests Claude working directory project selection
func TestGetClaudeWorkingProject(t *testing.T) {
	t.Run("returns project marked as claude_working_dir", func(t *testing.T) {
		cfg := &WorktreeConfig{
			Projects: map[string]ProjectConfig{
				"backend":  {Dir: "backend", ClaudeWorkingDir: false},
				"frontend": {Dir: "frontend", ClaudeWorkingDir: true},
			},
		}
		got := cfg.GetClaudeWorkingProject()
		if got != "frontend" {
			t.Errorf("GetClaudeWorkingProject() = %q, want %q", got, "frontend")
		}
	})

	t.Run("returns any project when none marked", func(t *testing.T) {
		cfg := &WorktreeConfig{
			Projects: map[string]ProjectConfig{
				"backend": {Dir: "backend"},
			},
		}
		got := cfg.GetClaudeWorkingProject()
		if got == "" {
			t.Error("GetClaudeWorkingProject() returned empty string, expected a project name")
		}
	})

	t.Run("returns empty for no projects", func(t *testing.T) {
		cfg := &WorktreeConfig{Projects: map[string]ProjectConfig{}}
		got := cfg.GetClaudeWorkingProject()
		if got != "" {
			t.Errorf("GetClaudeWorkingProject() = %q, want empty string", got)
		}
	})
}

// TestCalculateRelativePath tests relative path calculation
func TestCalculateRelativePath(t *testing.T) {
	tests := []struct {
		name  string
		depth int
		want  string
	}{
		{"depth 0 returns dot", 0, "."},
		{"negative depth returns dot", -1, "."},
		{"depth 1 returns ..", 1, ".."},
		{"depth 2 returns ../..", 2, "../.."},
		{"depth 3 returns ../../..", 3, "../../.."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRelativePath(tt.depth)
			if got != tt.want {
				t.Errorf("CalculateRelativePath(%d) = %q, want %q", tt.depth, got, tt.want)
			}
		})
	}
}

// TestGenerateFiles tests file generation from templates
func TestGenerateFiles(t *testing.T) {
	t.Run("generates file with placeholder substitution", func(t *testing.T) {
		featureDir := t.TempDir()
		projectDir := filepath.Join(featureDir, "backend")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := &WorktreeConfig{
			Projects: map[string]ProjectConfig{
				"backend": {Dir: "backend"},
			},
			GeneratedFiles: map[string][]GeneratedFile{
				"backend": {
					{Path: ".env.local", Template: "PORT={BE_PORT}\nFEATURE={FEATURE_NAME}\n"},
				},
			},
		}

		envVars := map[string]string{
			"BE_PORT":      "8081",
			"FEATURE_NAME": "feature-test",
		}

		if err := cfg.GenerateFiles("backend", featureDir, envVars); err != nil {
			t.Fatalf("GenerateFiles() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(projectDir, ".env.local"))
		if err != nil {
			t.Fatalf("failed to read generated file: %v", err)
		}

		got := string(data)
		want := "PORT=8081\nFEATURE=feature-test\n"
		if got != want {
			t.Errorf("generated file content = %q, want %q", got, want)
		}
	})

	t.Run("no-op when project has no generated files", func(t *testing.T) {
		cfg := &WorktreeConfig{
			Projects:       map[string]ProjectConfig{"backend": {Dir: "backend"}},
			GeneratedFiles: map[string][]GeneratedFile{},
		}
		if err := cfg.GenerateFiles("backend", t.TempDir(), map[string]string{}); err != nil {
			t.Errorf("GenerateFiles() for project with no files error = %v", err)
		}
	})

	t.Run("returns error for nonexistent project", func(t *testing.T) {
		cfg := &WorktreeConfig{
			Projects: map[string]ProjectConfig{},
			GeneratedFiles: map[string][]GeneratedFile{
				"missing": {{Path: "file.txt", Template: "content"}},
			},
		}
		err := cfg.GenerateFiles("missing", t.TempDir(), map[string]string{})
		if err == nil {
			t.Error("expected error for project not in Projects map")
		}
	})
}

// oauthConfig returns a WorktreeConfig that mirrors the real OAuth redirect URI setup
// in .worktree.yml: FE_PORT is a static expression "3000" (allocated from range),
// and the OAuth vars use {FE_PORT} in their value template.
func oauthConfig() *WorktreeConfig {
	feRange := [2]int{3000, 3100}
	beRange := [2]int{8080, 8180}
	return &WorktreeConfig{
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {
				Port:  "3000",
				Env:   "FE_PORT",
				Range: &feRange,
			},
			"BE_PORT": {
				Port:  "8080",
				Env:   "BE_PORT",
				Range: &beRange,
			},
			"GOOGLE_OAUTH_REDIRECT_URI": {
				Value: "http://localhost:{FE_PORT}/oauth/callback/google",
				Env:   "GOOGLE_OAUTH_REDIRECT_URI",
			},
			"OUTLOOK_OAUTH_REDIRECT_URI": {
				Value: "http://localhost:{FE_PORT}/oauth/callback/outlook",
				Env:   "OUTLOOK_OAUTH_REDIRECT_URI",
			},
			"REACT_APP_API_BASE_URL": {
				Value: "http://localhost:{BE_PORT}",
				Env:   "REACT_APP_API_BASE_URL",
			},
		},
	}
}

// TestResolveValueVars_PropagatesAllocatedPorts is the regression test for the bug:
// ExportEnvVars computes value templates against base port expressions (e.g. FE_PORT=3000)
// before the actual allocated port (e.g. 3002) is applied. ResolveValueVars must update
// the value vars to use the real ports.
func TestResolveValueVars_PropagatesAllocatedPorts(t *testing.T) {
	cfg := oauthConfig()

	// Simulate what start.go / newfeature.go do:
	// 1. ExportEnvVars computes base values (FE_PORT=3000 from expression "3000")
	envVars := cfg.ExportEnvVars(2) // instance 2

	// 2. After ExportEnvVars, FE_PORT is 3000 (the base expression value).
	//    The OAuth vars are already set to ...localhost:3000...
	if envVars["GOOGLE_OAUTH_REDIRECT_URI"] != "http://localhost:3000/oauth/callback/google" {
		t.Fatalf("precondition: expected base port 3000 before override, got %q", envVars["GOOGLE_OAUTH_REDIRECT_URI"])
	}

	// 3. Override with the actually allocated port from the registry (e.g. 3002)
	envVars["FE_PORT"] = "3002"
	envVars["BE_PORT"] = "8082"

	// 4. Without ResolveValueVars, OAuth URIs still reference :3000 — the bug.
	//    ResolveValueVars must fix them.
	cfg.ResolveValueVars(2, envVars)

	tests := []struct {
		key  string
		want string
	}{
		{"GOOGLE_OAUTH_REDIRECT_URI", "http://localhost:3002/oauth/callback/google"},
		{"OUTLOOK_OAUTH_REDIRECT_URI", "http://localhost:3002/oauth/callback/outlook"},
		{"REACT_APP_API_BASE_URL", "http://localhost:8082"},
	}
	for _, tt := range tests {
		if got := envVars[tt.key]; got != tt.want {
			t.Errorf("ResolveValueVars: %s = %q, want %q", tt.key, got, tt.want)
		}
	}
}

// TestResolveValueVars_BasePortWhenNoOverride verifies that calling ResolveValueVars
// without any port override does not change existing correct values.
func TestResolveValueVars_BasePortWhenNoOverride(t *testing.T) {
	cfg := oauthConfig()
	envVars := cfg.ExportEnvVars(0)

	// No override — ports stay at base (3000 / 8080)
	cfg.ResolveValueVars(0, envVars)

	if got := envVars["GOOGLE_OAUTH_REDIRECT_URI"]; got != "http://localhost:3000/oauth/callback/google" {
		t.Errorf("GOOGLE_OAUTH_REDIRECT_URI = %q, want localhost:3000", got)
	}
	if got := envVars["REACT_APP_API_BASE_URL"]; got != "http://localhost:8080" {
		t.Errorf("REACT_APP_API_BASE_URL = %q, want localhost:8080", got)
	}
}

// TestResolveValueVars_MultipleInstances verifies different allocated ports
// produce different URI values.
func TestResolveValueVars_MultipleInstances(t *testing.T) {
	cfg := oauthConfig()

	cases := []struct {
		allocatedFE int
		want        string
	}{
		{3000, "http://localhost:3000/oauth/callback/google"},
		{3001, "http://localhost:3001/oauth/callback/google"},
		{3007, "http://localhost:3007/oauth/callback/google"},
	}

	for _, tc := range cases {
		envVars := cfg.ExportEnvVars(0)
		envVars["FE_PORT"] = fmt.Sprintf("%d", tc.allocatedFE)
		cfg.ResolveValueVars(0, envVars)

		got := envVars["GOOGLE_OAUTH_REDIRECT_URI"]
		if got != tc.want {
			t.Errorf("allocated FE_PORT=%d: GOOGLE_OAUTH_REDIRECT_URI = %q, want %q",
				tc.allocatedFE, got, tc.want)
		}
	}
}

// TestResolveValueVars_OnlyValueVarsUpdated verifies that ResolveValueVars does not
// alter port-based env vars — it must only touch Value-template entries.
func TestResolveValueVars_OnlyValueVarsUpdated(t *testing.T) {
	cfg := oauthConfig()
	envVars := cfg.ExportEnvVars(1)
	envVars["FE_PORT"] = "3005" // simulate registry override

	cfg.ResolveValueVars(1, envVars)

	// Port var itself must not be changed by ResolveValueVars
	if got := envVars["FE_PORT"]; got != "3005" {
		t.Errorf("FE_PORT must stay %q after ResolveValueVars, got %q", "3005", got)
	}
}

// TestResolveValueVars_EmptyConfig is a no-op safety check.
func TestResolveValueVars_EmptyConfig(t *testing.T) {
	cfg := &WorktreeConfig{}
	envVars := map[string]string{"FE_PORT": "3001"}
	cfg.ResolveValueVars(0, envVars) // must not panic
	if envVars["FE_PORT"] != "3001" {
		t.Errorf("FE_PORT changed unexpectedly")
	}
}

// TestGetComputedVars_IncludesAllResolvedVars verifies that GetComputedVars returns
// all fully-resolved env vars: port vars, value-template vars (OAuth URIs, API URLs),
// and aliases — everything that has no unresolved {placeholder} tokens.
func TestGetComputedVars_IncludesAllResolvedVars(t *testing.T) {
	cfg := oauthConfig()
	envVars := cfg.ExportEnvVars(0)
	envVars["FE_PORT"] = "3003"
	envVars["BE_PORT"] = "8083"
	cfg.ResolveValueVars(0, envVars)

	computed := cfg.GetComputedVars(envVars)

	want := map[string]string{
		"FE_PORT":                    "3003",
		"BE_PORT":                    "8083",
		"GOOGLE_OAUTH_REDIRECT_URI":  "http://localhost:3003/oauth/callback/google",
		"OUTLOOK_OAUTH_REDIRECT_URI": "http://localhost:3003/oauth/callback/outlook",
		"REACT_APP_API_BASE_URL":     "http://localhost:8083",
	}
	for k, wantVal := range want {
		if got := computed[k]; got != wantVal {
			t.Errorf("GetComputedVars[%q] = %q, want %q", k, got, wantVal)
		}
	}
	if len(computed) != len(want) {
		t.Errorf("GetComputedVars returned %d entries, want %d: %v", len(computed), len(want), computed)
	}
}

// TestGetComputedVars_IncludesHostResolvedVars verifies that value-template vars using
// {host} are fully resolved (via GetValue) and included in computed_vars.
func TestGetComputedVars_IncludesHostResolvedVars(t *testing.T) {
	feRange := [2]int{3005, 3104}
	cfg := &WorktreeConfig{
		Hostname: "localhost",
		EnvVariables: map[string]EnvVarConfig{
			"APP_PORT": {Port: "8085", Env: "APP_PORT", Range: &[2]int{8085, 8184}},
			"FE_PORT":  {Port: "3005", Env: "FE_PORT", Range: &feRange},
			"GOOGLE_REDIRECT_URI": {
				Value: "http://{host}:{FE_PORT}/auth/callback",
				Env:   "GOOGLE_REDIRECT_URI",
			},
			"HTTP_CORS_ALLOW_ORIGINS": {
				Value: "http://{host}:{FE_PORT}",
				Env:   "HTTP_CORS_ALLOW_ORIGINS",
			},
		},
	}

	envVars := cfg.ExportEnvVars(0)
	envVars["APP_PORT"] = "8086"
	envVars["FE_PORT"] = "3006"
	cfg.ResolveValueVars(0, envVars)

	computed := cfg.GetComputedVars(envVars)

	want := map[string]string{
		"APP_PORT":                "8086",
		"FE_PORT":                 "3006",
		"GOOGLE_REDIRECT_URI":     "http://localhost:3006/auth/callback",
		"HTTP_CORS_ALLOW_ORIGINS": "http://localhost:3006",
	}
	for k, wantVal := range want {
		if got := computed[k]; got != wantVal {
			t.Errorf("computed[%q] = %q, want %q", k, got, wantVal)
		}
	}
	if len(computed) != len(want) {
		t.Errorf("expected %d entries, got %d: %v", len(want), len(computed), computed)
	}
}

// TestGetComputedVars_ExcludesUnresolvedTemplates verifies that entries whose resolved
// value still contains '{' (e.g. COMPOSE_PROJECT_NAME with {project}/{feature}/{service})
// are excluded — they cannot be substituted by the normal env-var resolution flow.
func TestGetComputedVars_ExcludesUnresolvedTemplates(t *testing.T) {
	r := [2]int{3000, 3100}
	cfg := &WorktreeConfig{
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {Port: "3000", Env: "FE_PORT", Range: &r},
			"COMPOSE_PROJECT_NAME": {
				Value: "{project}-{feature}-{service}",
				Env:   "COMPOSE_PROJECT_NAME",
			},
		},
	}
	envVars := cfg.ExportEnvVars(0)
	// After ExportEnvVars, COMPOSE_PROJECT_NAME = "{project}-{feature}-{service}" (unresolved)

	computed := cfg.GetComputedVars(envVars)

	if _, ok := computed["COMPOSE_PROJECT_NAME"]; ok {
		t.Errorf("GetComputedVars must exclude COMPOSE_PROJECT_NAME with unresolved template, got %v", computed)
	}
	// FE_PORT should still be present
	if _, ok := computed["FE_PORT"]; !ok {
		t.Error("GetComputedVars must include FE_PORT")
	}
}

// TestGetComputedVars_PortOnlyConfigIncludesPorts verifies that a port-only config
// (no Value templates) returns the port vars in computed_vars.
func TestGetComputedVars_PortOnlyConfigIncludesPorts(t *testing.T) {
	cfg := &WorktreeConfig{
		EnvVariables: map[string]EnvVarConfig{
			"FE_PORT": {Port: "3000", Env: "FE_PORT"},
			"BE_PORT": {Port: "8080", Env: "BE_PORT"},
		},
	}
	envVars := cfg.ExportEnvVars(0)
	envVars["FE_PORT"] = "3002"
	envVars["BE_PORT"] = "8082"

	computed := cfg.GetComputedVars(envVars)

	if got := computed["FE_PORT"]; got != "3002" {
		t.Errorf("computed[FE_PORT] = %q, want %q", got, "3002")
	}
	if got := computed["BE_PORT"]; got != "8082" {
		t.Errorf("computed[BE_PORT] = %q, want %q", got, "8082")
	}
	if len(computed) != 2 {
		t.Errorf("expected 2 entries, got %d: %v", len(computed), computed)
	}
}

// TestGetComputedVars_MissingEnvVarSkipped verifies that a Value-template entry
// whose env var is not yet in the envVars map is silently skipped.
func TestGetComputedVars_MissingEnvVarSkipped(t *testing.T) {
	cfg := &WorktreeConfig{
		EnvVariables: map[string]EnvVarConfig{
			"MY_URL": {Value: "http://example.com", Env: "MY_URL"},
		},
	}
	// envVars does not contain MY_URL
	computed := cfg.GetComputedVars(map[string]string{})
	if len(computed) != 0 {
		t.Errorf("GetComputedVars with missing var = %v, want empty", computed)
	}
}

// TestValidate_PortRangeAndDefaultPreset tests the remaining Validate branches
func TestValidate_PortRangeAndDefaultPreset(t *testing.T) {
	validBase := func() *WorktreeConfig {
		return &WorktreeConfig{
			Projects: map[string]ProjectConfig{
				"backend": {Dir: "backend"},
			},
			Presets: map[string]PresetConfig{
				"default": {Projects: []string{"backend"}},
			},
		}
	}

	t.Run("invalid port range: min out of 1-65535", func(t *testing.T) {
		cfg := validBase()
		r := [2]int{0, 100} // min=0 is invalid
		cfg.EnvVariables = map[string]EnvVarConfig{
			"FE_PORT": {Range: &r},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for port range with min=0")
		}
	})

	t.Run("invalid port range: max out of 1-65535", func(t *testing.T) {
		cfg := validBase()
		r := [2]int{1000, 70000} // max=70000 is invalid
		cfg.EnvVariables = map[string]EnvVarConfig{
			"FE_PORT": {Range: &r},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for port range with max=70000")
		}
	})

	t.Run("invalid port range: min >= max", func(t *testing.T) {
		cfg := validBase()
		r := [2]int{5000, 5000} // min == max is invalid
		cfg.EnvVariables = map[string]EnvVarConfig{
			"FE_PORT": {Range: &r},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for port range where min >= max")
		}
	})

	t.Run("invalid port expression with range", func(t *testing.T) {
		cfg := validBase()
		r := [2]int{3000, 3100}
		cfg.EnvVariables = map[string]EnvVarConfig{
			"FE_PORT": {Port: "not-a-number", Range: &r},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for invalid port expression")
		}
	})

	t.Run("default_preset nonexistent returns error", func(t *testing.T) {
		cfg := validBase()
		cfg.DefaultPreset = "nonexistent"
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for nonexistent default_preset")
		}
	})

	t.Run("default_preset matching preset is valid", func(t *testing.T) {
		cfg := validBase()
		cfg.DefaultPreset = "default"
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("valid port range passes", func(t *testing.T) {
		cfg := validBase()
		r := [2]int{3000, 3100}
		cfg.EnvVariables = map[string]EnvVarConfig{
			"FE_PORT": {Port: "3000 + {instance}", Range: &r},
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
}
