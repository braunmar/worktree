package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileLink represents a file or directory to symlink or copy
type FileLink struct {
	Source string `yaml:"source"` // Path relative to project root
	Target string `yaml:"target"` // Path relative to worktree root
}

// WorktreeConfig represents the .worktree.yml configuration
type WorktreeConfig struct {
	ProjectName   string                      `yaml:"project_name"`
	Hostname      string                      `yaml:"hostname"`
	Projects      map[string]ProjectConfig    `yaml:"projects"`
	Presets       map[string]PresetConfig     `yaml:"presets"`
	DefaultPreset string                      `yaml:"default_preset"`
	MaxInstances  int                         `yaml:"max_instances"`
	AutoFixtures  bool                        `yaml:"auto_fixtures"`
	Symlinks      []FileLink                  `yaml:"symlinks"`
	Copies        []FileLink                  `yaml:"copies"`
	Ports         map[string]PortConfig       `yaml:"ports"`
}

// PortConfig represents a port/service display configuration
type PortConfig struct {
	Name  string   `yaml:"name"`
	URL   string   `yaml:"url"`
	Port  string   `yaml:"port"`  // Expression like "3000 + {instance}" or null for non-port configs
	Value string   `yaml:"value"` // String template for non-port configs like COMPOSE_PROJECT_NAME
	Env   string   `yaml:"env"`   // Environment variable name to export
	Range *[2]int  `yaml:"range"` // Optional explicit range [min, max] for port allocation
}

// ProjectConfig represents a single project configuration
type ProjectConfig struct {
	Dir              string `yaml:"dir"`
	MainBranch       string `yaml:"main_branch"`
	StartCommand     string `yaml:"start_command"`
	PostCommand      string `yaml:"post_command"` // Runs after start (fixtures, seed, etc.)
	ClaudeWorkingDir bool   `yaml:"claude_working_dir"`
}

// PresetConfig represents a preset configuration
type PresetConfig struct {
	Projects    []string `yaml:"projects"`
	Description string   `yaml:"description"`
}

// validateProjectName validates that project name only contains alphanumeric characters and hyphens
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project_name cannot be empty")
	}

	// Check each character
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("project_name '%s' contains invalid character '%c'. Only alphanumeric characters and hyphens are allowed", name, char)
		}
	}

	// Check it doesn't start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("project_name '%s' cannot start or end with a hyphen", name)
	}

	return nil
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

	// Set default hostname if not provided
	if config.Hostname == "" {
		config.Hostname = "localhost"
	}

	// Validate project name is provided
	if config.ProjectName == "" {
		return nil, fmt.Errorf("project_name is required in .worktree.yml configuration file")
	}

	// Validate project name format
	if err := validateProjectName(config.ProjectName); err != nil {
		return nil, err
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
func (pc *PortConfig) GetURL(hostname string, port int) string {
	url := strings.ReplaceAll(pc.URL, "{host}", hostname)
	url = strings.ReplaceAll(url, "{port}", fmt.Sprintf("%d", port))
	return url
}

// GetValue calculates the value for this port config (either port or string template)
func (pc *PortConfig) GetValue(instance int) string {
	if pc.Port != "" {
		// Port calculation
		port := CalculatePort(pc.Port, instance)
		return fmt.Sprintf("%d", port)
	} else if pc.Value != "" {
		// String template (e.g., "myproject-inst{instance}")
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

// GetPortRange extracts port range from either explicit Range field or Port expression
func (pc *PortConfig) GetPortRange() *[2]int {
	// 1. If explicit range defined, use it
	if pc.Range != nil {
		return pc.Range
	}

	// 2. If port expression exists, extract base and calculate range
	if pc.Port != "" {
		base := extractBasePort(pc.Port)
		if base > 0 {
			return &[2]int{base, base + 100} // Default: 100 port range
		}
	}

	// 3. No range available
	return nil
}

// extractBasePort extracts base port from expressions like "3000 + {instance}"
func extractBasePort(expr string) int {
	// Handle "base + {instance}" format
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) >= 1 {
			var base int
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &base)
			return base
		}
	}

	// Handle plain number
	var port int
	fmt.Sscanf(strings.TrimSpace(expr), "%d", &port)
	return port
}

// GetServiceURL returns the formatted URL for a service by port env name
// Returns empty string if port not found or URL not configured
func (c *WorktreeConfig) GetServiceURL(portEnvName string, ports map[string]int) string {
	portCfg, exists := c.Ports[portEnvName]
	if !exists || portCfg.URL == "" {
		return ""
	}

	port, exists := ports[portEnvName]
	if !exists {
		return ""
	}

	return portCfg.GetURL(c.Hostname, port)
}

// GetDisplayableServices returns a list of services that should be displayed
// Returns map of service name -> URL
func (c *WorktreeConfig) GetDisplayableServices(ports map[string]int) map[string]string {
	services := make(map[string]string)

	for envName, portCfg := range c.Ports {
		// Skip if name or URL not configured (these are not meant to be displayed)
		if portCfg.Name == "" || portCfg.URL == "" {
			continue
		}

		port, exists := ports[envName]
		if !exists {
			continue
		}

		url := portCfg.GetURL(c.Hostname, port)
		services[portCfg.Name] = url
	}

	return services
}

// CalculateRelativePath calculates relative path from worktree to project root
// Example: worktrees/feature-name -> ../..
func CalculateRelativePath(worktreeDepth int) string {
	if worktreeDepth <= 0 {
		return "."
	}
	result := ".."
	for i := 1; i < worktreeDepth; i++ {
		result += "/.."
	}
	return result
}

// GetPortServiceNames returns list of all service names that need port allocation
// Services must have both 'env' and 'range' fields to be included
// This excludes template-only services like COMPOSE_PROJECT_NAME
func (c *WorktreeConfig) GetPortServiceNames() []string {
	var services []string
	for name, portCfg := range c.Ports {
		// Only include services that need port allocation (have both env and range)
		if portCfg.Env != "" && portCfg.Range != nil {
			services = append(services, name)
		}
	}
	return services
}

// GetComposeProjectTemplate returns the template for compose project names
// Returns "{project}-{feature}" as default if not configured
func (c *WorktreeConfig) GetComposeProjectTemplate() string {
	if portCfg, exists := c.Ports["COMPOSE_PROJECT_NAME"]; exists && portCfg.Value != "" {
		return portCfg.Value
	}
	// Default template for backward compatibility
	return "{project}-{feature}"
}

// ReplaceComposeProjectPlaceholders replaces placeholders in a compose project name template
// Supported placeholders: {project}, {feature}, {service}
func (c *WorktreeConfig) ReplaceComposeProjectPlaceholders(template, featureName, serviceName string) string {
	result := template
	result = strings.ReplaceAll(result, "{project}", c.ProjectName)
	result = strings.ReplaceAll(result, "{feature}", featureName)
	result = strings.ReplaceAll(result, "{service}", serviceName)
	return result
}
