package registry

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"worktree/pkg/config"
)

const (
	registryFileName = ".registry.json"
)

// Worktree represents a single worktree instance
type Worktree struct {
	Branch          string            `json:"branch"`
	Normalized      string            `json:"normalized"`
	Created         time.Time         `json:"created"`
	Projects        []string          `json:"projects"`
	Ports           map[string]int    `json:"ports"`
	ComposeProject  string            `json:"compose_project,omitempty"`  // Deprecated: use ComposeProjects
	ComposeProjects map[string]string `json:"compose_projects,omitempty"` // Per-service compose project names
	YoloMode        bool              `json:"yolo_mode,omitempty"`        // YOLO mode: Claude works autonomously when solution is clear
}

// GetComposeProject returns the compose project name for a specific service
// Falls back to the legacy ComposeProject field if per-service names are not set
func (w *Worktree) GetComposeProject(service string) string {
	if w.ComposeProjects != nil && w.ComposeProjects[service] != "" {
		return w.ComposeProjects[service]
	}
	// Fallback to legacy single compose project name
	return w.ComposeProject
}

// Registry manages all worktree instances and port allocations
type Registry struct {
	Worktrees  map[string]*Worktree `json:"worktrees"`
	PortRanges map[string][2]int    `json:"port_ranges"`
	mu         sync.RWMutex
	filePath   string
}

// BuildPortRanges constructs port ranges from WorktreeConfig
// All port ranges must be defined in the configuration file
func BuildPortRanges(workCfg *config.WorktreeConfig) map[string][2]int {
	ranges := make(map[string][2]int)

	if workCfg == nil {
		return ranges
	}

	// Read all configured port ranges
	for serviceName, portCfg := range workCfg.Ports {
		if portRange := portCfg.GetPortRange(); portRange != nil {
			ranges[serviceName] = *portRange
		}
	}

	return ranges
}

// Load loads the registry from disk, or creates a new one if it doesn't exist
// workCfg is optional - if provided, port ranges are loaded from configuration
func Load(worktreeDir string, workCfg *config.WorktreeConfig) (*Registry, error) {
	registryPath := filepath.Join(worktreeDir, registryFileName)

	// Build port ranges from config (with defaults as fallback)
	portRanges := BuildPortRanges(workCfg)

	r := &Registry{
		Worktrees:  make(map[string]*Worktree),
		PortRanges: portRanges,
		filePath:   registryPath,
	}

	// If registry doesn't exist, return empty registry
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return r, nil
	}

	// Read existing registry
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	// Override with configured port ranges (config is source of truth)
	r.PortRanges = portRanges
	r.filePath = registryPath

	return r, nil
}

// Save persists the registry to disk
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure worktrees directory exists
	worktreeDir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write atomically by writing to temp file and renaming
	tempPath := r.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	if err := os.Rename(tempPath, r.filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// Add adds a new worktree to the registry
func (r *Registry) Add(wt *Worktree) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Worktrees[wt.Normalized]; exists {
		return fmt.Errorf("worktree '%s' already exists in registry", wt.Normalized)
	}

	r.Worktrees[wt.Normalized] = wt
	return nil
}

// Remove removes a worktree from the registry
func (r *Registry) Remove(normalized string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Worktrees[normalized]; !exists {
		return fmt.Errorf("worktree '%s' not found in registry", normalized)
	}

	delete(r.Worktrees, normalized)
	return nil
}

// Get retrieves a worktree by normalized name
func (r *Registry) Get(normalized string) (*Worktree, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wt, exists := r.Worktrees[normalized]
	return wt, exists
}

// List returns all worktrees sorted by creation time
func (r *Registry) List() []*Worktree {
	r.mu.RLock()
	defer r.mu.RUnlock()

	worktrees := make([]*Worktree, 0, len(r.Worktrees))
	for _, wt := range r.Worktrees {
		worktrees = append(worktrees, wt)
	}

	return worktrees
}

// FindAvailablePort finds an available port for a service
func (r *Registry) FindAvailablePort(service string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	portRange, ok := r.PortRanges[service]
	if !ok {
		// List available services for better error message
		availableServices := make([]string, 0, len(r.PortRanges))
		for svc := range r.PortRanges {
			availableServices = append(availableServices, svc)
		}
		return 0, fmt.Errorf("unknown service: %s\nAvailable services: %v", service, availableServices)
	}

	minPort, maxPort := portRange[0], portRange[1]

	// Collect used ports for this service from registry
	usedPorts := make(map[int]bool)
	for _, wt := range r.Worktrees {
		if port, ok := wt.Ports[service]; ok {
			usedPorts[port] = true
		}
	}

	// Find first available port
	for port := minPort; port <= maxPort; port++ {
		if !usedPorts[port] && isPortAvailable(port) {
			return port, nil
		}
	}

	// Build detailed error message showing what's allocated
	allocatedInfo := make([]string, 0)
	for _, wt := range r.Worktrees {
		if port, ok := wt.Ports[service]; ok {
			allocatedInfo = append(allocatedInfo, fmt.Sprintf("%s: %d", wt.Normalized, port))
		}
	}

	errorMsg := fmt.Sprintf("no available ports in range %d-%d for service %s", minPort, maxPort, service)
	if len(allocatedInfo) > 0 {
		errorMsg += fmt.Sprintf("\nCurrently allocated:\n  %s", strings.Join(allocatedInfo, "\n  "))
	}

	return 0, fmt.Errorf(errorMsg)
}

// AllocatePorts allocates ports for all specified services
func (r *Registry) AllocatePorts(services []string) (map[string]int, error) {
	ports := make(map[string]int)

	for _, service := range services {
		port, err := r.FindAvailablePort(service)
		if err != nil {
			return nil, err
		}
		ports[service] = port
	}

	return ports, nil
}

// isPortAvailable checks if a port is available by attempting to bind to it
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// NormalizeBranchName converts a branch name to a filesystem-safe directory name
func NormalizeBranchName(branch string) string {
	// Remove refs/heads/ prefix if present
	branch = strings.TrimPrefix(branch, "refs/heads/")

	// Convert to lowercase
	normalized := strings.ToLower(branch)

	// Replace slashes with hyphens
	normalized = strings.ReplaceAll(normalized, "/", "-")

	// Replace underscores with hyphens
	normalized = strings.ReplaceAll(normalized, "_", "-")

	// Replace dots with hyphens (for version numbers like v2.0.0)
	normalized = strings.ReplaceAll(normalized, ".", "-")

	// Remove special characters, keep alphanumeric and hyphens
	normalized = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(normalized, "")

	// Remove leading/trailing hyphens
	normalized = strings.Trim(normalized, "-")

	// Collapse multiple hyphens
	normalized = regexp.MustCompile(`-+`).ReplaceAllString(normalized, "-")

	return normalized
}
