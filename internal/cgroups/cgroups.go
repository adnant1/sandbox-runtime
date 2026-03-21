package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"sandbox-runtime/internal/sandbox"
	"strconv"
	"strings"
)

// ResourceManager is responsible for managing cgroup v2 resources for
// each sandbox.
type ResourceManager struct {
	mountPoint string
	basePath   string // root directory for all sandbox cgroups
}

// New initializes a new ResourceManager
func New(mountPoint string) *ResourceManager {
	if mountPoint == "" {
		panic("resource manager: mountPoint cannot be empty")
	}
	return &ResourceManager{
		mountPoint: mountPoint,
		basePath:   filepath.Join(mountPoint, "secure-sandbox"),
	}
}

// Create creates a cgroup directory for a sandbox.
func (rm *ResourceManager) Create(id string) error {
	if id == "" {
		return fmt.Errorf("cgroup id cannot be empty")
	}
	if strings.Contains(id, "/") || strings.Contains(id, "..") {
		return fmt.Errorf("invalid cgroup id: %s", id)
	}

	// Ensure basePath exists
	if err := os.MkdirAll(rm.basePath, 0o755); err != nil {
		return fmt.Errorf("failed to create base cgroup path: %w", err)
	}

	cgroupPath := rm.sandboxPath(id)
	if err := os.MkdirAll(cgroupPath, 0o755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("cgroup already exists for sandbox %s", id)
		}
		return fmt.Errorf("failed to create cgroup: %w", err)
	}
	return nil
}

// ApplyLimits applies resource limits to a sandbox group.
// Translates high-level ResourceSpec into cgroup v2 file writes.
func (rm *ResourceManager) ApplyLimits(id string, spec sandbox.ResourceSpec) error {
	if id == "" {
		return fmt.Errorf("cgroup id cannot be empty")
	}
	if spec.MemoryMB < 0 {
		return fmt.Errorf("memory limit cannot be negative: %d", spec.MemoryMB)
	}
	if spec.CPU < 0 || spec.CPU > 100 {
		return fmt.Errorf("cpu limit must be between 0 and 100: %d", spec.CPU)
	}
	if spec.Pids < 0 {
		return fmt.Errorf("pids limit cannot be negative: %d", spec.Pids)
	}

	// "max" is the default for no limit
	memVal := "max"
	if spec.MemoryMB > 0 {
		memBytes := int64(spec.MemoryMB) * 1024 * 1024
		memVal = strconv.FormatInt(memBytes, 10)
	}
	if err := os.WriteFile(rm.memoryFile(id), []byte(memVal), 0o644); err != nil {
		return fmt.Errorf("write memory.max for sandbox %s: %w", id, err)
	}

	// cgroup v2 expects: "<quota> <peroid>"
	// set a default fixed period of 100000 microseconds
	cpuVal := "max 100000"
	if spec.CPU > 0 {
		const period = 100000
		quota := (spec.CPU * period) / 100
		cpuVal = fmt.Sprintf("%d %d", quota, period)
	}
	if err := os.WriteFile(rm.cpuFile(id), []byte(cpuVal), 0o644); err != nil {
		return fmt.Errorf("write cpu.max for sandbox %s: %w", id, err)
	}

	pidsVal := "max"
	if spec.Pids > 0 {
		pidsVal = strconv.Itoa(spec.Pids)
	}
	if err := os.WriteFile(rm.pidsFile(id), []byte(pidsVal), 0o644); err != nil {
		return fmt.Errorf("write pids.max for sandbox %s: %w", id, err)
	}

	return nil
}

// AddProcess attaches a process to the sandbox group.
func (rm *ResourceManager) AddProcess(id string, pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}

	procsPath := rm.procsFile(id)
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(procsPath, []byte(pidStr), 0o644); err != nil {
		return fmt.Errorf("failed to add pid %d to cgroup: %w", pid, err)
	}
	return nil
}

// Delete removes a sandbox group.
func (rm *ResourceManager) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("cgroup id cannot be empty")
	}

	cgroupPath := rm.sandboxPath(id)

	// Attempt to remove the cgroup directory
	// Will fail if: processes are still alive, path doesn't exist, permission denied
	if err := os.Remove(cgroupPath); err != nil {
		return fmt.Errorf("failed to delete cgroup %s: %w", id, err)
	}
	return nil
}
