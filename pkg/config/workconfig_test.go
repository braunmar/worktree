package config

import (
	"fmt"
	"math"
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
				Ports: map[string]PortConfig{
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
				Ports: map[string]PortConfig{
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
				Ports: map[string]PortConfig{
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
				Ports: map[string]PortConfig{
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
				Ports: map[string]PortConfig{
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
				Ports: map[string]PortConfig{
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
		{Ports: map[string]PortConfig{}}, // Empty ports
		{Ports: map[string]PortConfig{
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

// TestPortConfig_GetValue tests the GetValue method
func TestPortConfig_GetValue(t *testing.T) {
	tests := []struct {
		name     string
		portCfg  PortConfig
		instance int
		want     string
	}{
		{
			name: "port expression",
			portCfg: PortConfig{
				Port: "3000 + {instance}",
			},
			instance: 1,
			want:     "3001",
		},
		{
			name: "string value template",
			portCfg: PortConfig{
				Value: "myproject-{instance}",
			},
			instance: 2,
			want:     "myproject-2",
		},
		{
			name: "static port",
			portCfg: PortConfig{
				Port: "8080",
			},
			instance: 0,
			want:     "8080",
		},
		{
			name:     "empty config",
			portCfg:  PortConfig{},
			instance: 0,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.portCfg.GetValue(tt.instance)
			if got != tt.want {
				t.Errorf("GetValue(%d) = %q, want %q", tt.instance, got, tt.want)
			}
		})
	}
}

// TestPortConfig_GetPortRange tests port range extraction
func TestPortConfig_GetPortRange(t *testing.T) {
	tests := []struct {
		name    string
		portCfg PortConfig
		want    *[2]int
	}{
		{
			name: "explicit range",
			portCfg: PortConfig{
				Port:  "3000 + {instance}",
				Range: &[2]int{3000, 3100},
			},
			want: &[2]int{3000, 3100},
		},
		{
			name: "port expression with placeholder",
			portCfg: PortConfig{
				Port: "5000 + {instance}",
			},
			want: &[2]int{5000, 5100}, // ExtractBasePort replaces {instance} with 0 and extracts base
		},
		{
			name: "static port",
			portCfg: PortConfig{
				Port: "8080",
			},
			want: &[2]int{8080, 8180},
		},
		{
			name:    "no range available",
			portCfg: PortConfig{},
			want:    nil,
		},
		{
			name: "explicit range takes precedence",
			portCfg: PortConfig{
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
