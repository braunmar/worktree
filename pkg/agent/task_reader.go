package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadTaskFile reads .task.md from the specified directory
func ReadTaskFile(dir string) (string, error) {
	taskPath := filepath.Join(dir, ".task.md")

	content, err := os.ReadFile(taskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Task file is optional
		}
		return "", fmt.Errorf("failed to read .task.md: %w", err)
	}

	return string(content), nil
}

// InjectTaskIntoSkill replaces {task} placeholder with task content
func InjectTaskIntoSkill(skill string, taskContent string) string {
	return strings.ReplaceAll(skill, "{task}", taskContent)
}
