package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	ProjectName     string                     `yaml:"project_name"`
	Hostname        string                     `yaml:"hostname"`
	Projects        map[string]ProjectConfig   `yaml:"projects"`
	Presets         map[string]PresetConfig    `yaml:"presets"`
	DefaultPreset   string                     `yaml:"default_preset"`
	MaxInstances    int                        `yaml:"max_instances"`
	AutoFixtures    bool                       `yaml:"auto_fixtures"`
	Symlinks        []FileLink                 `yaml:"symlinks"`
	Copies          []FileLink                 `yaml:"copies"`
	EnvVariables    map[string]EnvVarConfig    `yaml:"env_variables"`
	GeneratedFiles  map[string][]GeneratedFile `yaml:"generated_files"`
	ScheduledAgents ScheduledAgents            `yaml:"scheduled_agents"` // NEW: Scheduled agent tasks
}

// EnvVarConfig represents an environment variable configuration entry (port, string template, or display-only)
type EnvVarConfig struct {
	Name  string  `yaml:"name"`
	URL   string  `yaml:"url"`
	Port  string  `yaml:"port"`  // Expression like "3000 + {instance}" or null for non-port configs
	Value string  `yaml:"value"` // String template for non-port configs like COMPOSE_PROJECT_NAME
	Env   string  `yaml:"env"`   // Environment variable name to export
	Range *[2]int `yaml:"range"` // Optional explicit range [min, max] for port allocation
}

// ProjectConfig represents a single project configuration
type ProjectConfig struct {
	Executor           string `yaml:"executor"` // "docker" (default) or "process"
	Dir                string `yaml:"dir"`
	MainBranch         string `yaml:"main_branch"`
	StartPreCommand    string `yaml:"start_pre_command"` // Runs before start_command
	StartCommand       string `yaml:"start_command"`
	StartPostCommand   string `yaml:"start_post_command"`   // Runs after start_command (fixtures, seed, etc.)
	StopPreCommand     string `yaml:"stop_pre_command"`     // Runs before stopping services
	StopPostCommand    string `yaml:"stop_post_command"`    // Runs after stopping services
	RestartPreCommand  string `yaml:"restart_pre_command"`  // Runs before the full restart cycle
	RestartPostCommand string `yaml:"restart_post_command"` // Runs after the full restart cycle
	ClaudeWorkingDir   bool   `yaml:"claude_working_dir"`
}

// GetExecutor returns the executor type, defaulting to "docker" if not set.
func (p *ProjectConfig) GetExecutor() string {
	if p.Executor == "" {
		return "docker"
	}
	return p.Executor
}

// PresetConfig represents a preset configuration
type PresetConfig struct {
	Projects    []string `yaml:"projects"`
	Description string   `yaml:"description"`
}

// GeneratedFile represents a file to be auto-generated in a worktree
type GeneratedFile struct {
	Path     string `yaml:"path"`     // File path relative to project directory
	Template string `yaml:"template"` // Template content with {PLACEHOLDER} substitution
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

	// Validate port ranges
	for name, portCfg := range c.EnvVariables {
		if portCfg.Range != nil {
			if portCfg.Range[0] < 1 || portCfg.Range[1] > 65535 {
				return fmt.Errorf("port %s: range [%d, %d] outside valid range 1-65535",
					name, portCfg.Range[0], portCfg.Range[1])
			}
			if portCfg.Range[0] >= portCfg.Range[1] {
				return fmt.Errorf("port %s: invalid range [%d, %d] - min must be < max",
					name, portCfg.Range[0], portCfg.Range[1])
			}
		}

		// Validate port expressions are parseable
		if portCfg.Port != "" && portCfg.Range != nil {
			// Try to parse expression to catch syntax errors early
			_, err := CalculatePort(portCfg.Port, 0)
			if err != nil {
				return fmt.Errorf("port %s: invalid expression '%s': %w",
					name, portCfg.Port, err)
			}
		}
	}

	// Validate default_preset exists
	if c.DefaultPreset != "" {
		if _, exists := c.Presets[c.DefaultPreset]; !exists {
			return fmt.Errorf("default_preset '%s' does not exist in presets map", c.DefaultPreset)
		}
	}

	// Validate hostname
	if c.Hostname == "" {
		c.Hostname = "localhost"
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

// portExpression represents a parsed port expression
type portExpression struct {
	base       int
	hasOffset  bool
	offset     int
	hasMult    bool
	multiplier int
}

// parseExpression parses port expressions into structured format
// Supports formats: "3000", "3000 + 5", "3000 + 2 * 50"
func parseExpression(expr string) (*portExpression, error) {
	expr = strings.TrimSpace(expr)
	result := &portExpression{}

	// Handle "base + value * multiplier" format (e.g., "4510 + 2 * 50")
	if strings.Contains(expr, "+") && strings.Contains(expr, "*") {
		parts := strings.Split(expr, "+")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid expression format: expected 'base + offset * multiplier'")
		}

		var base int
		n, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &base)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("failed to parse base port: %w", err)
		}
		result.base = base

		// Parse multiplication in second part
		multParts := strings.Split(strings.TrimSpace(parts[1]), "*")
		if len(multParts) != 2 {
			return nil, fmt.Errorf("invalid multiplication format: expected 'factor1 * factor2'")
		}

		var factor1, factor2 int
		n1, err1 := fmt.Sscanf(strings.TrimSpace(multParts[0]), "%d", &factor1)
		if err1 != nil || n1 != 1 {
			return nil, fmt.Errorf("failed to parse first factor: %w", err1)
		}
		n2, err2 := fmt.Sscanf(strings.TrimSpace(multParts[1]), "%d", &factor2)
		if err2 != nil || n2 != 1 {
			return nil, fmt.Errorf("failed to parse second factor: %w", err2)
		}
		result.hasOffset = true
		result.hasMult = true
		result.offset = factor1
		result.multiplier = factor2
		return result, nil
	}

	// Handle simple "base + offset" format
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid expression format: expected 'base + offset'")
		}

		var base, offset int
		n1, err1 := fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &base)
		if err1 != nil || n1 != 1 {
			return nil, fmt.Errorf("failed to parse base port: %w", err1)
		}
		n2, err2 := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &offset)
		if err2 != nil || n2 != 1 {
			return nil, fmt.Errorf("failed to parse offset: %w", err2)
		}
		result.base = base
		result.hasOffset = true
		result.offset = offset
		return result, nil
	}

	// Just a number
	var port int
	n, err := fmt.Sscanf(expr, "%d", &port)
	if err != nil || n != 1 {
		return nil, fmt.Errorf("failed to parse port number: %w", err)
	}
	result.base = port
	return result, nil
}

// calculateFromExpression calculates the final port value from a parsed expression
func (pe *portExpression) calculateFromExpression() (int, error) {
	var result int
	if pe.hasMult {
		result = pe.base + (pe.offset * pe.multiplier)
	} else if pe.hasOffset {
		result = pe.base + pe.offset
	} else {
		result = pe.base
	}

	if result < 1 || result > 65535 {
		return 0, fmt.Errorf("calculated port %d is out of valid range (1-65535)", result)
	}
	return result, nil
}

// CalculatePort evaluates a port expression like "3000 + {instance}" or "4510 + {instance} * 50"
func CalculatePort(expression string, instance int) (int, error) {
	// Replace {instance} with actual value
	expr := strings.ReplaceAll(expression, "{instance}", fmt.Sprintf("%d", instance))

	// Parse the expression
	parsed, err := parseExpression(expr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse expression '%s': %w", expression, err)
	}

	// Calculate the final port value
	result, err := parsed.calculateFromExpression()
	if err != nil {
		return 0, fmt.Errorf("invalid port for expression '%s': %w", expression, err)
	}

	return result, nil
}

// GetPortURL generates the URL for a port configuration
func (pc *EnvVarConfig) GetURL(hostname string, port int) string {
	url := strings.ReplaceAll(pc.URL, "{host}", hostname)
	url = strings.ReplaceAll(url, "{port}", fmt.Sprintf("%d", port))
	return url
}

// GetValue calculates the value for this port config (either port or string template).
// hostname is used to resolve the {host} placeholder in value templates.
// Returns empty string if calculation fails.
func (pc *EnvVarConfig) GetValue(instance int, envVars map[string]string, hostname string) string {
	if pc.Port != "" {
		// Port calculation
		port, err := CalculatePort(pc.Port, instance)
		if err != nil {
			// Log error but don't crash - return empty string
			// Caller should validate port configurations during startup
			return ""
		}
		return fmt.Sprintf("%d", port)
	} else if pc.Value != "" {
		// String template - substitute placeholders
		result := pc.Value

		// Substitute {host} with the configured hostname
		result = strings.ReplaceAll(result, "{host}", hostname)

		// Substitute {instance}
		result = strings.ReplaceAll(result, "{instance}", fmt.Sprintf("%d", instance))

		// Substitute port variables like {BE_PORT}, {FE_PORT}, etc.
		for key, value := range envVars {
			placeholder := fmt.Sprintf("{%s}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}

		return result
	}
	return ""
}

// ExportEnvVars exports all configured environment variables for the given instance
func (c *WorktreeConfig) ExportEnvVars(instance int) map[string]string {
	envVars := make(map[string]string)

	// Always export INSTANCE first
	envVars["INSTANCE"] = fmt.Sprintf("%d", instance)

	// First pass: Export all port values (both allocated and calculated ports)
	for _, portCfg := range c.EnvVariables {
		if portCfg.Env != "" && portCfg.Port != "" {
			value := portCfg.GetValue(instance, envVars, c.Hostname)
			if value != "" {
				envVars[portCfg.Env] = value
			}
		}
	}

	// Second pass: Export string templates that depend on ports
	for _, portCfg := range c.EnvVariables {
		if portCfg.Env != "" && portCfg.Value != "" {
			value := portCfg.GetValue(instance, envVars, c.Hostname)
			if value != "" {
				envVars[portCfg.Env] = value
			}
		}
	}

	return envVars
}

// GenerateFiles creates configured files for a project with templated content
// Uses the same placeholder substitution as environment variables
func (c *WorktreeConfig) GenerateFiles(projectName, featureDir string, envVars map[string]string) error {
	files, ok := c.GeneratedFiles[projectName]
	if !ok {
		return nil // No files to generate for this project
	}

	projectConfig, exists := c.Projects[projectName]
	if !exists {
		return fmt.Errorf("project '%s' not found in configuration", projectName)
	}

	projectPath := filepath.Join(featureDir, projectConfig.Dir)

	for _, file := range files {
		// Substitute placeholders in template
		content := file.Template
		for key, value := range envVars {
			placeholder := fmt.Sprintf("{%s}", key)
			content = strings.ReplaceAll(content, placeholder, value)
		}

		// Write file
		filePath := filepath.Join(projectPath, file.Path)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to generate %s: %w", file.Path, err)
		}
	}

	return nil
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
func (pc *EnvVarConfig) GetPortRange() *[2]int {
	// 1. If explicit range defined, use it
	if pc.Range != nil {
		return pc.Range
	}

	// 2. If port expression exists, extract base and calculate range
	if pc.Port != "" {
		base, err := ExtractBasePort(pc.Port)
		if err == nil && base > 0 {
			return &[2]int{base, base + 100} // Default: 100 port range
		}
	}

	// 3. No range available
	return nil
}

// ExtractBasePort extracts base port from expressions like "3000 + {instance}"
func ExtractBasePort(expr string) (int, error) {
	// Replace {instance} with 0 to get base port
	cleanedExpr := strings.ReplaceAll(expr, "{instance}", "0")

	// Parse the expression
	parsed, err := parseExpression(cleanedExpr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse expression '%s': %w", expr, err)
	}

	// Validate base port is in range
	if parsed.base < 1 || parsed.base > 65535 {
		return 0, fmt.Errorf("base port %d is out of valid range (1-65535) in expression '%s'", parsed.base, expr)
	}

	return parsed.base, nil
}

// GetServiceURL returns the formatted URL for a service by port env name
// Returns empty string if port not found or URL not configured
func (c *WorktreeConfig) GetServiceURL(portEnvName string, ports map[string]int) string {
	portCfg, exists := c.EnvVariables[portEnvName]
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

	for envName, portCfg := range c.EnvVariables {
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

// ResolveValueVars recomputes value-template env vars (e.g., GOOGLE_OAUTH_REDIRECT_URI)
// using the provided envVars map. Call this AFTER overriding ports from the registry
// so that placeholder substitution uses the actual allocated port values, not base values.
func (c *WorktreeConfig) ResolveValueVars(instance int, envVars map[string]string) {
	for _, portCfg := range c.EnvVariables {
		if portCfg.Env != "" && portCfg.Value != "" {
			value := portCfg.GetValue(instance, envVars, c.Hostname)
			if value != "" {
				envVars[portCfg.Env] = value
			}
		}
	}
}

// GetComputedVars returns all env vars that are fully resolved in the provided envVars map.
// Entries with unresolved {placeholder} tokens (e.g. COMPOSE_PROJECT_NAME={project}-{feature}-{service})
// are excluded — they are substituted separately and cannot be stored as-is.
func (c *WorktreeConfig) GetComputedVars(envVars map[string]string) map[string]string {
	result := make(map[string]string)
	for _, portCfg := range c.EnvVariables {
		if portCfg.Env == "" {
			continue
		}
		val, ok := envVars[portCfg.Env]
		if !ok {
			continue
		}
		// Skip values with unresolved placeholders like {project}, {feature}, {service}
		if strings.Contains(val, "{") {
			continue
		}
		result[portCfg.Env] = val
	}
	return result
}

// GetPortServiceNames returns list of all service names that need port allocation
// Services must have both 'env' and 'range' fields to be included
// This excludes template-only services like COMPOSE_PROJECT_NAME
func (c *WorktreeConfig) GetPortServiceNames() []string {
	var services []string
	for name, portCfg := range c.EnvVariables {
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
	if portCfg, exists := c.EnvVariables["COMPOSE_PROJECT_NAME"]; exists && portCfg.Value != "" {
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

// GetInstancePortName returns the name of the env variable used for instance number calculation.
// It finds the first env variable (alphabetically) that has both a range and a port expression.
// All ranged ports with base+{instance} expressions yield the same instance number, so any one works.
func (c *WorktreeConfig) GetInstancePortName() (string, error) {
	var names []string
	for name, cfg := range c.EnvVariables {
		if cfg.Range != nil && cfg.Port != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no port variable with a range found in env_variables; at least one port must have a range for instance calculation")
	}
	sort.Strings(names)
	return names[0], nil
}

// GetFirstProjectDir returns the Dir field of the first configured project (alphabetically).
// Used as a representative git directory for health checks.
func (c *WorktreeConfig) GetFirstProjectDir() string {
	var names []string
	for name := range c.Projects {
		names = append(names, name)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return c.Projects[names[0]].Dir
}
