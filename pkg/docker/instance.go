package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// IsFeatureRunning checks if a specific feature worktree is running
func IsFeatureRunning(featureName string) bool {
	// Container name format: skillsetup-{feature-name}-app-1
	prefix := fmt.Sprintf("skillsetup-%s-", featureName)

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
func GetRunningFeatures() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--filter", "name=skillsetup-", "--format", "{{.Names}}")
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
		// Format: skillsetup-{feature-name}-{service}-1
		if strings.HasPrefix(line, "skillsetup-") {
			// Remove "skillsetup-" prefix
			rest := strings.TrimPrefix(line, "skillsetup-")

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

// StopFeature stops a specific feature worktree
func StopFeature(featureName string, worktreePath string) error {
	// Change to backend directory and run make down
	backendDir := worktreePath + "/backend"

	cmd := exec.Command("make", "down")
	cmd.Dir = backendDir

	// Set environment variable for compose project name
	cmd.Env = append(cmd.Env, fmt.Sprintf("COMPOSE_PROJECT_NAME=skillsetup-%s", featureName))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop feature: %s", stderr.String())
	}

	return nil
}

// GetFeatureContainerStatus returns the status of containers for a feature
func GetFeatureContainerStatus(featureName string) (map[string]string, error) {
	prefix := fmt.Sprintf("skillsetup-%s-", featureName)

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
			// Format: skillsetup-{feature-name}-{service}-1
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

// Legacy functions for backward compatibility during transition
// These will be removed once all commands are updated

// IsInstanceRunning checks if a specific instance is running (DEPRECATED)
func IsInstanceRunning(instance int) (bool, error) {
	containerName := fmt.Sprintf("skillsetup-inst%d-app-1", instance)

	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check docker status: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	return output == containerName, nil
}

// GetRunningInstances returns a list of running instance numbers (DEPRECATED)
func GetRunningInstances() ([]int, error) {
	cmd := exec.Command("docker", "ps", "--filter", "name=skillsetup-inst", "--format", "{{.Names}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	instances := []int{}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Extract instance number from container name
		// Format: skillsetup-inst{N}-app-1
		if strings.HasPrefix(line, "skillsetup-inst") {
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				var instanceNum int
				if _, err := fmt.Sscanf(parts[1], "inst%d", &instanceNum); err == nil {
					instances = append(instances, instanceNum)
				}
			}
		}
	}

	return instances, nil
}

// StopInstance stops a specific instance (DEPRECATED)
func StopInstance(instance int, projectRoot string) error {
	// Change to backend directory and run make down
	backendDir := projectRoot + "/backend"

	cmd := exec.Command("make", "down", fmt.Sprintf("INSTANCE=%d", instance))
	cmd.Dir = backendDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop instance: %s", stderr.String())
	}

	return nil
}

// GetContainerStatus returns the status of containers for an instance (DEPRECATED)
func GetContainerStatus(instance int) (map[string]string, error) {
	prefix := fmt.Sprintf("skillsetup-inst%d-", instance)

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
			// Format: skillsetup-inst{N}-{service}-1
			name := parts[0]
			nameParts := strings.Split(name, "-")
			if len(nameParts) >= 3 {
				service := nameParts[2]
				status[service] = parts[1]
			}
		}
	}

	return status, nil
}
