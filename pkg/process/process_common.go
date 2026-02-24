package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readPID reads a PID from a file created by StartBackground.
func readPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("PID file not found: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file %s: %w", pidFile, err)
	}
	return pid, nil
}
