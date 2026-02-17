package ui

import (
	"fmt"
	"github.com/braunmar/worktree/pkg/config"

	"github.com/fatih/color"
)

var (
	// Color functions
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
)

// Success prints a success message
func Success(message string) {
	fmt.Printf("%s %s\n", green("✅"), message)
}

// Error prints an error message
func Error(message string) {
	fmt.Printf("%s %s\n", red("❌"), message)
}

// Warning prints a warning message
func Warning(message string) {
	fmt.Printf("%s %s\n", yellow("⚠️ "), message)
}

// Info prints an info message
func Info(message string) {
	fmt.Printf("%s %s\n", blue("ℹ️ "), message)
}

// Section prints a section header
func Section(title string) {
	fmt.Printf("\n%s %s\n\n", cyan("»"), bold(title))
}

// Rocket prints a message with a rocket emoji
func Rocket(message string) {
	fmt.Printf("%s %s\n", "🚀", message)
}

// Loading prints a loading message
func Loading(message string) {
	fmt.Printf("%s %s\n", "⏳", message)
}

// CheckMark prints a check mark with a message
func CheckMark(message string) {
	fmt.Printf("  %s %s\n", green("✅"), message)
}

// CrossMark prints a cross mark with a message
func CrossMark(message string) {
	fmt.Printf("  %s %s\n", red("❌"), message)
}

// ShowPortsFromConfig displays port mapping from configuration
func ShowPortsFromConfig(hostname string, instance int, ports map[string]int, portConfigs map[string]config.PortConfig) {
	if len(portConfigs) == 0 {
		// Fallback to showing instance number only
		fmt.Printf("\n%s Instance %d configured\n\n", "📍", instance)
		return
	}

	fmt.Printf("\n%s Services (Instance %d):\n", "📍", instance)

	// Display ports in order (if config preserves order, or alphabetically)
	// Skip entries without a name (used only for env var export)
	for envName, portCfg := range portConfigs {
		// Skip if name is empty or URL is null/empty
		if portCfg.Name == "" || portCfg.Name == "null" {
			continue
		}
		if portCfg.URL == "" || portCfg.URL == "null" {
			continue
		}

		port, exists := ports[envName]
		if !exists {
			continue
		}

		url := portCfg.GetURL(hostname, port)
		if url == "" || url == "null" {
			continue
		}
		fmt.Printf("   %s %s\n", blue(portCfg.Name+":"), url)
	}

	fmt.Println()
}

// PrintHeader prints a header message
func PrintHeader(message string) {
	fmt.Printf("\n%s\n", bold(message))
}

// PrintStep prints a numbered step
func PrintStep(number int, message string) {
	fmt.Printf("   %s %s\n", cyan(fmt.Sprintf("%d.", number)), message)
}

// PrintCommand prints a command to run
func PrintCommand(command string) {
	fmt.Printf("      %s\n", magenta(command))
}

// PrintNextSteps prints next steps section
func PrintNextSteps() {
	fmt.Printf("\n%s\n", bold("Next steps:"))
}

// PrintStatusLine prints a status line with label and value
func PrintStatusLine(label, value string) {
	fmt.Printf("  %s %s\n", cyan(label+":"), value)
}

// PrintTable prints a simple table row
func PrintTable(col1, col2 string) {
	fmt.Printf("%-20s %s\n", col1, col2)
}

// NewLine prints a new line
func NewLine() {
	fmt.Println()
}

// Progress prints a progress indicator with current/total counts
func Progress(current, total int, message string) {
	fmt.Printf("%s %s... (%d/%d)\n", "⏳", message, current, total)
}

// ProgressWithName prints a progress indicator for a named item
func ProgressWithName(current, total int, itemName, action string) {
	fmt.Printf("%s %s %s... (%d/%d)\n", "⏳", action, itemName, current, total)
}

// Bold returns a bold-formatted string
func Bold(text string) string {
	return bold(text)
}
