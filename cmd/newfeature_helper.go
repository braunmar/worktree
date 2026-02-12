package cmd

import "worktree/pkg/config"

// getClaudeWorkingProject returns the project configured as Claude's working directory
// from the given preset projects (not all projects in config)
func getClaudeWorkingProject(workCfg *config.WorktreeConfig, presetProjects []string) string {
	// First, check if any project in the preset has claude_working_dir: true
	for _, projectName := range presetProjects {
		if project, exists := workCfg.Projects[projectName]; exists && project.ClaudeWorkingDir {
			return projectName
		}
	}

	// Default to first project in the preset
	if len(presetProjects) > 0 {
		return presetProjects[0]
	}

	return ""
}
