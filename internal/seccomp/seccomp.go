package seccomp

import (
	"fmt"
	"sandbox-runtime/internal/sandbox"
	"strings"
)

// Apply installs the seccomp filter for the current process.
func Apply(cfg sandbox.SeccompConfig) error {
	return apply(cfg)
}

func validateConfig(cfg sandbox.SeccompConfig) error {
	if len(cfg.AllowedSyscalls) == 0 {
		return fmt.Errorf("seccomp: no allowed syscalls defined")
	}
	switch cfg.DefaultAction {
	case sandbox.SeccompActionErrno, sandbox.SeccompActionKill:
	default:
		return fmt.Errorf("seccomp: unknown default action: %s", cfg.DefaultAction)
	}
	for i, name := range cfg.AllowedSyscalls {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("seccomp: empty syscall name at index %d", i)
		}
	}
	return nil
}
