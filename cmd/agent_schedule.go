package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"worktree/pkg/config"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var scheduleSetupAll bool

var agentScheduleCmd = &cobra.Command{
	Use:   "schedule [task-name]",
	Short: "Set up cron/launchd scheduling for agent tasks",
	Long: `Generate cron or launchd configuration for scheduled agent tasks.

On macOS, generates launchd plist files for ~/Library/LaunchAgents/
On Linux, generates crontab entries

Examples:
  worktree agent schedule npm-audit        # Set up scheduling for one task
  worktree agent schedule --all            # Set up scheduling for all tasks`,
	Run: runAgentSchedule,
}

func runAgentSchedule(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Check if scheduled_agents section exists
	if workCfg.ScheduledAgents == nil || len(workCfg.ScheduledAgents) == 0 {
		checkError(fmt.Errorf("no scheduled_agents defined in .worktree.yml"))
	}

	// Determine which tasks to schedule
	tasksToSchedule := make(map[string]*config.AgentTask)

	if scheduleSetupAll {
		// Schedule all tasks
		for name, task := range workCfg.ScheduledAgents {
			tasksToSchedule[name] = task
		}
	} else if len(args) == 1 {
		// Schedule specific task
		taskName := args[0]
		task, exists := workCfg.ScheduledAgents[taskName]
		if !exists {
			checkError(fmt.Errorf("agent task '%s' not found in .worktree.yml", taskName))
		}
		tasksToSchedule[taskName] = task
	} else {
		checkError(fmt.Errorf("specify a task name or use --all flag"))
	}

	// Get worktree binary path
	worktreeBin, err := os.Executable()
	if err != nil {
		checkError(fmt.Errorf("failed to get worktree binary path: %w", err))
	}

	// Detect OS and generate appropriate scheduler config
	switch runtime.GOOS {
	case "darwin":
		generateLaunchdConfig(tasksToSchedule, worktreeBin, cfg.ProjectRoot)
	case "linux":
		generateCrontabConfig(tasksToSchedule, worktreeBin, cfg.ProjectRoot)
	default:
		checkError(fmt.Errorf("unsupported OS: %s (only macOS and Linux are supported)", runtime.GOOS))
	}
}

func generateLaunchdConfig(tasks map[string]*config.AgentTask, worktreeBin, projectRoot string) {
	ui.Section("Generating launchd configuration (macOS)")
	fmt.Println()

	launchAgentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")

	// Create LaunchAgents directory if it doesn't exist
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		checkError(fmt.Errorf("failed to create LaunchAgents directory: %w", err))
	}

	for taskName, task := range tasks {
		plistName := fmt.Sprintf("com.skillsetup.worktree.%s.plist", taskName)
		plistPath := filepath.Join(launchAgentsDir, plistName)

		// Convert cron schedule to launchd StartCalendarInterval
		calendarInterval := cronToLaunchdInterval(task.Schedule)

		// Generate plist content
		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>agent</string>
        <string>run</string>
        <string>%s</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>StartCalendarInterval</key>
    %s
    <key>StandardOutPath</key>
    <string>%s/logs/worktree-agent-%s.log</string>
    <key>StandardErrorPath</key>
    <string>%s/logs/worktree-agent-%s.err</string>
</dict>
</plist>
`, plistName, worktreeBin, taskName, projectRoot, calendarInterval,
			os.Getenv("HOME"), taskName, os.Getenv("HOME"), taskName)

		// Write plist file
		if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
			ui.Error(fmt.Sprintf("Failed to write %s: %v", plistPath, err))
			continue
		}

		ui.CheckMark(fmt.Sprintf("Created %s", plistPath))

		// Print load command
		fmt.Printf("  To enable: launchctl load %s\n", plistPath)
		fmt.Printf("  To start now: launchctl start %s\n", plistName)
		fmt.Println()
	}

	ui.Info("Logs will be written to ~/logs/worktree-agent-*.log")
}

func generateCrontabConfig(tasks map[string]*config.AgentTask, worktreeBin, projectRoot string) {
	ui.Section("Generating crontab entries (Linux)")
	fmt.Println()

	fmt.Println("Add the following lines to your crontab (run 'crontab -e'):")
	fmt.Println()

	for taskName, task := range tasks {
		logFile := filepath.Join(os.Getenv("HOME"), "logs", fmt.Sprintf("worktree-agent-%s.log", taskName))
		errFile := filepath.Join(os.Getenv("HOME"), "logs", fmt.Sprintf("worktree-agent-%s.err", taskName))

		fmt.Printf("# %s: %s\n", task.Name, task.Description)
		fmt.Printf("%s cd %s && %s agent run %s >> %s 2>> %s\n\n",
			task.Schedule, projectRoot, worktreeBin, taskName, logFile, errFile)
	}

	ui.Info("Logs will be written to ~/logs/worktree-agent-*.log")
}

// cronToLaunchdInterval converts a cron expression to launchd StartCalendarInterval
func cronToLaunchdInterval(cron string) string {
	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return "<dict><key>Hour</key><integer>9</integer><key>Minute</key><integer>0</integer></dict>"
	}

	minute, hour, dayOfMonth, month, dayOfWeek := parts[0], parts[1], parts[2], parts[3], parts[4]

	// Build calendar interval
	interval := "<dict>"

	if minute != "*" {
		interval += fmt.Sprintf("<key>Minute</key><integer>%s</integer>", minute)
	}

	if hour != "*" {
		interval += fmt.Sprintf("<key>Hour</key><integer>%s</integer>", hour)
	}

	if dayOfMonth != "*" {
		interval += fmt.Sprintf("<key>Day</key><integer>%s</integer>", dayOfMonth)
	}

	if month != "*" {
		monthNum := parseMonth(month)
		interval += fmt.Sprintf("<key>Month</key><integer>%d</integer>", monthNum)
	}

	if dayOfWeek != "*" {
		weekdayNum := parseWeekday(dayOfWeek)
		interval += fmt.Sprintf("<key>Weekday</key><integer>%d</integer>", weekdayNum)
	}

	interval += "</dict>"

	return interval
}

func parseMonth(month string) int {
	months := map[string]int{
		"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
		"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
	}

	if num, exists := months[strings.ToUpper(month)]; exists {
		return num
	}

	// Try parsing as number
	var num int
	fmt.Sscanf(month, "%d", &num)
	return num
}

func parseWeekday(weekday string) int {
	weekdays := map[string]int{
		"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
	}

	if num, exists := weekdays[strings.ToUpper(weekday)]; exists {
		return num
	}

	// Try parsing as number
	var num int
	fmt.Sscanf(weekday, "%d", &num)
	return num
}

func init() {
	agentScheduleCmd.Flags().BoolVar(&scheduleSetupAll, "all", false, "Set up scheduling for all agent tasks")
	agentCmd.AddCommand(agentScheduleCmd)
}
