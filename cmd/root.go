package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"  // overridden by -ldflags at build time
	commit  = "none" // overridden by -ldflags at build time
)

var rootCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees for multi-instance development",
	Long: `Worktree Manager - A CLI tool for managing git worktrees in multi-instance development environments.

This tool helps you create, manage, and remove coordinated git worktrees for multiple
projects, integrated with multi-instance Docker setups.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Resolve version: ldflags take priority, then module build info (go install @version),
	// then VCS info embedded by go build (vcs.revision setting).
	if info, ok := debug.ReadBuildInfo(); ok {
		if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		if commit == "none" {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					commit = s.Value[:7]
					break
				}
			}
		}
	}
	rootCmd.Version = version + " (" + commit + ")"

	// Add global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(stopAllCmd)
	rootCmd.AddCommand(rebaseCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(portsCmd)
	rootCmd.AddCommand(newFeatureCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(yoloCmd)
	rootCmd.AddCommand(agentCmd)

	// Customize help template
	rootCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
