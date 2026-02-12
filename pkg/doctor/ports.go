package doctor

import (
	"fmt"
	"net"
	"worktree/pkg/config"
	"worktree/pkg/registry"
)

// CheckPorts checks port allocations for conflicts and range violations
func CheckPorts(reg *registry.Registry, workCfg *config.WorktreeConfig) PortReport {
	report := PortReport{
		PortRanges: make(map[string]PortRangeInfo),
	}

	// Build map of allocated ports
	allocatedPorts := make(map[int]PortAllocation)
	for _, wt := range reg.List() {
		for service, port := range wt.Ports {
			allocatedPorts[port] = PortAllocation{
				Feature: wt.Normalized,
				Service: service,
				Port:    port,
			}
			report.TotalAllocated++
		}
	}

	// Check each allocated port against configured ranges
	for port, alloc := range allocatedPorts {
		// Find which service this port belongs to
		portCfg, ok := workCfg.Ports[alloc.Service]
		if !ok {
			continue // Service not in config (might be custom)
		}

		// Check if port is in range
		if portCfg.Range != nil {
			min, max := (*portCfg.Range)[0], (*portCfg.Range)[1]
			if port < min || port > max {
				report.OutOfRange = append(report.OutOfRange, PortOutOfRange{
					Service: alloc.Service,
					Port:    port,
					Feature: alloc.Feature,
					Range:   *portCfg.Range,
				})
			}
		}

		// Check if port is actually available (might be in use by another process)
		if !isPortAvailable(port) {
			// Port is in use - check if it might be by Docker
			// We'll skip this check for now as it could be the worktree's own containers
			// report.Conflicts = append(report.Conflicts, PortConflict{...})
		}
	}

	// Calculate port range statistics
	for serviceName, portCfg := range workCfg.Ports {
		if portCfg.Range == nil {
			continue // Skip services without explicit ranges
		}

		min, max := (*portCfg.Range)[0], (*portCfg.Range)[1]
		rangeSize := max - min + 1

		// Count allocated ports in this range
		allocated := 0
		for _, wt := range reg.List() {
			if port, ok := wt.Ports[serviceName]; ok {
				if port >= min && port <= max {
					allocated++
				}
			}
		}

		report.PortRanges[serviceName] = PortRangeInfo{
			Min:       min,
			Max:       max,
			Allocated: allocated,
			Available: rangeSize - allocated,
		}

		report.TotalAvailable += (rangeSize - allocated)
	}

	return report
}

// PortAllocation tracks which feature/service uses a port
type PortAllocation struct {
	Feature string
	Service string
	Port    int
}

// isPortAvailable checks if a port is available by trying to bind to it
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
