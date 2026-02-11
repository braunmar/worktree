package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorktreeConfig represents the .worktree.yml configuration
type WorktreeConfig struct {
	Projects       map[string]ProjectConfig    `yaml:"projects"`
	Presets        map[string]PresetConfig     `yaml:"presets"`
	DefaultPreset  string                      `yaml:"default_preset"`
	MaxInstances   int                         `yaml:"max_instances"`
	AutoFixtures   bool                        `yaml:"auto_fixtures"`
	AutoMigrations bool                        `yaml:"auto_migrations"`
	Ports          map[string]PortConfig       `yaml:"ports"`
}

// PortConfig represents a port/service display configuration
type PortConfig struct {
	Name  string `yaml:"name"`
	URL   string `yaml:"url"`
	Port  string `yaml:"port"`  // Expression like "3000 + {instance}" or null for non-port configs
	Value string `yaml:"value"` // String template for non-port configs like COMPOSE_PROJECT_NAME
	Env   string `yaml:"env"`   // Environment variable name to export
}

// ProjectConfig represents a single project configuration
type ProjectConfig struct {
	Dir                string `yaml:"dir"`
	MainBranch         string `yaml:"main_branch"`
	StartCommand       string `yaml:"start_command"`
	MigrationCommand   string `yaml:"migration_command"`
	PostCommand        string `yaml:"post_command"` // Runs after start (fixtures, seed, etc.)
	ClaudeWorkingDir   bool   `yaml:"claude_working_dir"`
}

// PresetConfig represents a preset configuration
type PresetConfig struct {
	Projects    []string `yaml:"projects"`
	Description string   `yaml:"description"`
}

// LoadWorktreeConfig loads the .worktree.yml configuration file
func LoadWorktreeConfig(projectRoot string) (*WorktreeConfig, error) {
	configPath := filepath.Join(projectRoot, ".worktree.yml")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s\nRun 'worktree init-config' to create a default configuration", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config WorktreeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *WorktreeConfig) Validate() error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("no projects defined")
	}

	if len(c.Presets) == 0 {
		return fmt.Errorf("no presets defined")
	}

	// Validate that preset projects exist
	for presetName, preset := range c.Presets {
		for _, projectName := range preset.Projects {
			if _, exists := c.Projects[projectName]; !exists {
				return fmt.Errorf("preset '%s' references undefined project '%s'", presetName, projectName)
			}
		}
	}

	return nil
}

// GetPreset returns a preset by name, or the default preset
func (c *WorktreeConfig) GetPreset(name string) (*PresetConfig, error) {
	if name == "" {
		name = c.DefaultPreset
	}

	preset, exists := c.Presets[name]
	if !exists {
		return nil, fmt.Errorf("preset '%s' not found", name)
	}

	return &preset, nil
}

// GetProjectsForPreset returns the list of projects for a preset
func (c *WorktreeConfig) GetProjectsForPreset(presetName string) ([]ProjectConfig, error) {
	preset, err := c.GetPreset(presetName)
	if err != nil {
		return nil, err
	}

	var projects []ProjectConfig
	for _, projectName := range preset.Projects {
		project, exists := c.Projects[projectName]
		if !exists {
			return nil, fmt.Errorf("project '%s' not found", projectName)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// ReplaceInstancePlaceholder replaces {instance} placeholder in command
func ReplaceInstancePlaceholder(command string, instance int) string {
	return strings.ReplaceAll(command, "{instance}", fmt.Sprintf("%d", instance))
}

// CalculatePort evaluates a port expression like "3000 + {instance}"
func CalculatePort(expression string, instance int) int {
	// Replace {instance} with actual value
	expr := strings.ReplaceAll(expression, "{instance}", fmt.Sprintf("%d", instance))
	expr = strings.TrimSpace(expr)

	// Simple evaluation: handle "base + instance" format
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) == 2 {
			var base, offset int
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &base)
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &offset)
			return base + offset
		}
	}

	// Just a number
	var port int
	fmt.Sscanf(expr, "%d", &port)
	return port
}

// GetPortURL generates the URL for a port configuration
func (pc *PortConfig) GetURL(instance int) string {
	port := CalculatePort(pc.Port, instance)
	return strings.ReplaceAll(pc.URL, "{port}", fmt.Sprintf("%d", port))
}

// GetValue calculates the value for this port config (either port or string template)
func (pc *PortConfig) GetValue(instance int) string {
	if pc.Port != "" {
		// Port calculation
		port := CalculatePort(pc.Port, instance)
		return fmt.Sprintf("%d", port)
	} else if pc.Value != "" {
		// String template (e.g., "skillsetup-inst{instance}")
		return strings.ReplaceAll(pc.Value, "{instance}", fmt.Sprintf("%d", instance))
	}
	return ""
}

// ExportEnvVars exports all configured environment variables for the given instance
func (c *WorktreeConfig) ExportEnvVars(instance int) map[string]string {
	envVars := make(map[string]string)

	// Always export INSTANCE_ID
	envVars["INSTANCE_ID"] = fmt.Sprintf("%d", instance)

	for _, portCfg := range c.Ports {
		if portCfg.Env != "" {
			value := portCfg.GetValue(instance)
			if value != "" {
				envVars[portCfg.Env] = value
			}
		}
	}

	return envVars
}

// GetClaudeWorkingProject returns the project configured as Claude's working directory
func (c *WorktreeConfig) GetClaudeWorkingProject() string {
	for name, project := range c.Projects {
		if project.ClaudeWorkingDir {
			return name
		}
	}
	// Default to first project if none specified
	for name := range c.Projects {
		return name
	}
	return ""
}
