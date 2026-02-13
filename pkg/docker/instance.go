package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// IsFeatureRunning checks if a specific feature worktree is running
func IsFeatureRunning(projectName, featureName string) bool {
	// Container name format: {project-name}-{feature-name}-app-1
	prefix := fmt.Sprintf("%s-%s-", projectName, featureName)

	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", prefix), "--format", "{{.Names}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false
	}

	output := strings.TrimSpace(stdout.String())
	return output != ""
}

// GetRunningFeatures returns a list of running feature names
func GetRunningFeatures(projectName string) ([]string, error) {
	prefix := projectName + "-"
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", prefix), "--format", "{{.Names}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	featuresMap := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Extract feature name from container name
		// Format: {project-name}-{feature-name}-{service}-1
		if strings.HasPrefix(line, prefix) {
			// Remove "{project-name}-" prefix
			rest := strings.TrimPrefix(line, prefix)

			// Split by "-" and take all parts except the last two (service-1)
			parts := strings.Split(rest, "-")
			if len(parts) >= 3 {
				// Reconstruct feature name (everything except last 2 parts)
				featureName := strings.Join(parts[:len(parts)-2], "-")
				featuresMap[featureName] = true
			}
		}
	}

	// Convert map to slice
	features := make([]string, 0, len(featuresMap))
	for feature := range featuresMap {
		features = append(features, feature)
	}

	return features, nil
}

// StopFeature stops a specific feature worktree using a multi-tier approach
// It stops containers for all active projects (backend, frontend, etc.)
// projectInfo maps project directory to its compose project name
func StopFeature(projectName, featureName string, worktreePath string, projectInfo map[string]string) error {
	defaultComposeProject := fmt.Sprintf("%s-%s", projectName, featureName)

	// Tier 1: Try docker compose down in each project directory with correct compose project name
	allStopped := true
	for projectDir, composeName := range projectInfo {
		fullPath := worktreePath + "/" + projectDir
		if err := stopViaCompose(fullPath, composeName); err != nil {
			allStopped = false
		}
	}
	if allStopped && len(projectInfo) > 0 {
		return nil
	}

	// Tier 2: Try docker compose with explicit project names (no directory needed)
	allStopped = true
	for _, composeName := range projectInfo {
		if err := stopViaComposeProject(composeName); err != nil {
			allStopped = false
		}
	}
	if allStopped && len(projectInfo) > 0 {
		return nil
	}

	// Tier 3: Fall back to stopping individual containers by compose project names
	for _, composeName := range projectInfo {
		if err := stopContainersByName(composeName); err != nil {
			// Continue trying other projects even if one fails
		}
	}

	// Tier 4: Last resort - try with default compose project name (for legacy compatibility)
	if err := stopContainersByName(defaultComposeProject); err == nil {
		return nil
	}

	// If all methods fail, return error but allow removal to continue
	return fmt.Errorf("unable to stop services (docker may not be available)")
}

// stopViaCompose runs docker compose down in the specified directory
func stopViaCompose(dir string, composeProject string) error {
	cmd := exec.Command("docker", "compose", "-p", composeProject, "down", "--remove-orphans")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose down failed: %s", stderr.String())
	}
	return nil
}

// stopViaComposeProject runs docker compose down with explicit project name
func stopViaComposeProject(composeProject string) error {
	cmd := exec.Command("docker", "compose", "-p", composeProject, "down", "--remove-orphans")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose -p down failed: %s", stderr.String())
	}
	return nil
}

// stopContainersByName finds and stops containers by name pattern
func stopContainersByName(composeProject string) error {
	// Find running containers
	prefix := composeProject + "-"
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("name=%s", prefix))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	containerIDs := strings.Fields(strings.TrimSpace(stdout.String()))
	if len(containerIDs) == 0 {
		// No containers running - this is fine
		return nil
	}

	// Stop containers
	args := append([]string{"stop"}, containerIDs...)
	stopCmd := exec.Command("docker", args...)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	// Remove containers
	args = append([]string{"rm"}, containerIDs...)
	rmCmd := exec.Command("docker", args...)
	if err := rmCmd.Run(); err != nil {
		// Warn but don't fail - containers are stopped
		return nil
	}

	return nil
}

// GetFeatureContainerStatus returns the status of containers for a feature
func GetFeatureContainerStatus(projectName, featureName string) (map[string]string, error) {
	prefix := fmt.Sprintf("%s-%s-", projectName, featureName)

	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", prefix), "--format", "{{.Names}}:{{.Status}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	status := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			// Extract service name from container name
			// Format: {project-name}-{feature-name}-{service}-1
			name := parts[0]

			// Remove prefix to get service name
			rest := strings.TrimPrefix(name, prefix)
			serviceParts := strings.Split(rest, "-")
			if len(serviceParts) >= 2 {
				// Service is everything except the last part (which is the replica number)
				service := strings.Join(serviceParts[:len(serviceParts)-1], "-")
				status[service] = parts[1]
			}
		}
	}

	return status, nil
}
