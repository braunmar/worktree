package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"worktree/pkg/config"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var agentInstallServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install agent scheduler as system service",
	Long: `Install the agent scheduler as a system service.

On macOS: Creates a launchd plist in ~/Library/LaunchAgents/
On Linux: Creates a systemd service in ~/.config/systemd/user/
On Windows: Creates a Windows service (requires admin)

The service will start automatically on boot and run the scheduler daemon.

Examples:
  worktree agent install-service`,
	Run: runAgentInstallService,
}

var agentUninstallServiceCmd = &cobra.Command{
	Use:   "uninstall-service",
	Short: "Uninstall agent scheduler system service",
	Long: `Remove the agent scheduler system service.

Examples:
  worktree agent uninstall-service`,
	Run: runAgentUninstallService,
}

func runAgentInstallService(cmd *cobra.Command, args []string) {
	cfg, err := config.New()
	checkError(err)

	// Get worktree binary path
	worktreeBin, err := os.Executable()
	if err != nil {
		checkError(fmt.Errorf("failed to get worktree binary path: %w", err))
	}

	ui.Section("Installing Agent Scheduler Service")
	fmt.Println()

	switch runtime.GOOS {
	case "darwin":
		installLaunchdService(worktreeBin, cfg.ProjectRoot)
	case "linux":
		installSystemdService(worktreeBin, cfg.ProjectRoot)
	case "windows":
		ui.Error("Windows service installation not yet implemented")
		fmt.Println()
		fmt.Println("For now, use Task Scheduler to run:")
		fmt.Printf("  %s agent daemon\n", worktreeBin)
	default:
		checkError(fmt.Errorf("unsupported OS: %s", runtime.GOOS))
	}
}

func runAgentUninstallService(cmd *cobra.Command, args []string) {
	ui.Section("Uninstalling Agent Scheduler Service")
	fmt.Println()

	switch runtime.GOOS {
	case "darwin":
		uninstallLaunchdService()
	case "linux":
		uninstallSystemdService()
	case "windows":
		ui.Error("Windows service uninstallation not yet implemented")
	default:
		checkError(fmt.Errorf("unsupported OS: %s", runtime.GOOS))
	}
}

func installLaunchdService(worktreeBin, projectRoot string) {
	plistName := "com.skillsetup.worktree.scheduler.plist"
	launchAgentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	plistPath := filepath.Join(launchAgentsDir, plistName)

	// Create LaunchAgents directory if it doesn't exist
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		checkError(fmt.Errorf("failed to create LaunchAgents directory: %w", err))
	}

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
        <string>daemon</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/logs/worktree-scheduler.log</string>
    <key>StandardErrorPath</key>
    <string>%s/logs/worktree-scheduler.err</string>
</dict>
</plist>
`, plistName, worktreeBin, projectRoot, os.Getenv("HOME"), os.Getenv("HOME"))

	// Write plist file
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		checkError(fmt.Errorf("failed to write plist: %w", err))
	}

	ui.CheckMark(fmt.Sprintf("Created %s", plistPath))
	fmt.Println()

	// Load the service
	ui.Loading("Loading service...")
	loadCmd := exec.Command("launchctl", "load", plistPath)
	if output, err := loadCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to load service: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
		fmt.Println()
		fmt.Println("To load manually:")
		fmt.Printf("  launchctl load %s\n", plistPath)
	} else {
		ui.CheckMark("Service loaded and started")
	}

	fmt.Println()
	ui.Success("✅ Agent scheduler service installed")
	fmt.Println()
	fmt.Println("Service commands:")
	fmt.Printf("  Start:   launchctl start %s\n", plistName)
	fmt.Printf("  Stop:    launchctl stop %s\n", plistName)
	fmt.Printf("  Unload:  launchctl unload %s\n", plistPath)
	fmt.Printf("  Logs:    tail -f ~/logs/worktree-scheduler.log\n")
}

func uninstallLaunchdService() {
	plistName := "com.skillsetup.worktree.scheduler.plist"
	launchAgentsDir := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents")
	plistPath := filepath.Join(launchAgentsDir, plistName)

	// Unload the service
	ui.Loading("Unloading service...")
	unloadCmd := exec.Command("launchctl", "unload", plistPath)
	if output, err := unloadCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Service may not be loaded: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Service unloaded")
	}

	// Remove plist file
	if err := os.Remove(plistPath); err != nil {
		if os.IsNotExist(err) {
			ui.Warning("Service file not found (may already be uninstalled)")
		} else {
			checkError(fmt.Errorf("failed to remove plist: %w", err))
		}
	} else {
		ui.CheckMark(fmt.Sprintf("Removed %s", plistPath))
	}

	fmt.Println()
	ui.Success("✅ Agent scheduler service uninstalled")
}

func installSystemdService(worktreeBin, projectRoot string) {
	serviceName := "worktree-scheduler.service"
	systemdDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	servicePath := filepath.Join(systemdDir, serviceName)

	// Create systemd user directory if it doesn't exist
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		checkError(fmt.Errorf("failed to create systemd directory: %w", err))
	}

	// Generate service content
	service := fmt.Sprintf(`[Unit]
Description=Worktree Agent Scheduler
After=network.target

[Service]
Type=simple
ExecStart=%s agent daemon
WorkingDirectory=%s
Restart=always
RestartSec=10
StandardOutput=append:%s/logs/worktree-scheduler.log
StandardError=append:%s/logs/worktree-scheduler.err

[Install]
WantedBy=default.target
`, worktreeBin, projectRoot, os.Getenv("HOME"), os.Getenv("HOME"))

	// Write service file
	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		checkError(fmt.Errorf("failed to write service file: %w", err))
	}

	ui.CheckMark(fmt.Sprintf("Created %s", servicePath))
	fmt.Println()

	// Reload systemd
	ui.Loading("Reloading systemd...")
	reloadCmd := exec.Command("systemctl", "--user", "daemon-reload")
	if output, err := reloadCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to reload systemd: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Systemd reloaded")
	}

	// Enable the service
	ui.Loading("Enabling service...")
	enableCmd := exec.Command("systemctl", "--user", "enable", serviceName)
	if output, err := enableCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to enable service: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Service enabled")
	}

	// Start the service
	ui.Loading("Starting service...")
	startCmd := exec.Command("systemctl", "--user", "start", serviceName)
	if output, err := startCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to start service: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
		fmt.Println()
		fmt.Println("To start manually:")
		fmt.Printf("  systemctl --user start %s\n", serviceName)
	} else {
		ui.CheckMark("Service started")
	}

	fmt.Println()
	ui.Success("✅ Agent scheduler service installed")
	fmt.Println()
	fmt.Println("Service commands:")
	fmt.Printf("  Start:   systemctl --user start %s\n", serviceName)
	fmt.Printf("  Stop:    systemctl --user stop %s\n", serviceName)
	fmt.Printf("  Restart: systemctl --user restart %s\n", serviceName)
	fmt.Printf("  Status:  systemctl --user status %s\n", serviceName)
	fmt.Printf("  Logs:    journalctl --user -u %s -f\n", serviceName)
}

func uninstallSystemdService() {
	serviceName := "worktree-scheduler.service"
	systemdDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	servicePath := filepath.Join(systemdDir, serviceName)

	// Stop the service
	ui.Loading("Stopping service...")
	stopCmd := exec.Command("systemctl", "--user", "stop", serviceName)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Service may not be running: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Service stopped")
	}

	// Disable the service
	ui.Loading("Disabling service...")
	disableCmd := exec.Command("systemctl", "--user", "disable", serviceName)
	if output, err := disableCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Service may not be enabled: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Service disabled")
	}

	// Remove service file
	if err := os.Remove(servicePath); err != nil {
		if os.IsNotExist(err) {
			ui.Warning("Service file not found (may already be uninstalled)")
		} else {
			checkError(fmt.Errorf("failed to remove service file: %w", err))
		}
	} else {
		ui.CheckMark(fmt.Sprintf("Removed %s", servicePath))
	}

	// Reload systemd
	ui.Loading("Reloading systemd...")
	reloadCmd := exec.Command("systemctl", "--user", "daemon-reload")
	if output, err := reloadCmd.CombinedOutput(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to reload systemd: %v", err))
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
		}
	} else {
		ui.CheckMark("Systemd reloaded")
	}

	fmt.Println()
	ui.Success("✅ Agent scheduler service uninstalled")
}

func init() {
	agentCmd.AddCommand(agentInstallServiceCmd)
	agentCmd.AddCommand(agentUninstallServiceCmd)
}
