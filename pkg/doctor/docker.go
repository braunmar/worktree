package doctor

import (
	"bytes"
	"os/exec"
	"strings"
)

// CheckDocker checks Docker installation and availability
func CheckDocker() DockerHealth {
	health := DockerHealth{}

	// Check if docker command exists
	versionCmd := exec.Command("docker", "--version")
	var versionOut bytes.Buffer
	versionCmd.Stdout = &versionOut

	if err := versionCmd.Run(); err != nil {
		health.Error = "Docker not installed or not in PATH"
		return health
	}

	health.Installed = true
	health.Version = strings.TrimSpace(versionOut.String())

	// Check if daemon is running
	psCmd := exec.Command("docker", "ps")
	if err := psCmd.Run(); err != nil {
		health.Error = "Docker daemon not running"
		return health
	}

	health.Running = true

	// Check docker compose
	composeCmd := exec.Command("docker", "compose", "version")
	health.ComposeAvailable = composeCmd.Run() == nil

	return health
}
