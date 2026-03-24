//go:build linux && cgo

package seccomp

import (
	"fmt"
	"strings"

	libseccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"

	"sandbox-runtime/internal/sandbox"
)

func apply(cfg sandbox.SeccompConfig) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	var defaultAction libseccomp.ScmpAction
	switch cfg.DefaultAction {
	case sandbox.SeccompActionErrno:
		defaultAction = libseccomp.ActErrno.SetReturnCode(int16(unix.EPERM))
	case sandbox.SeccompActionKill:
		defaultAction = libseccomp.ActKillThread
	default:
		fmt.Errorf("seccomp: unsupported default action: &s", cfg.DefaultAction)
	}

	// Create filter and add allowed syscalls
	filter, err := libseccomp.NewFilter(defaultAction)
	if err != nil {
		return fmt.Errorf("seccomp: create filer: %w", err)
	}
	for _, name := range cfg.AllowedSyscalls {
		name := strings.TrimSpace(name)

		sc, err := libseccomp.GetSyscallFromName(name)
		if err != nil {
			return fmt.Errorf("seccomp: resolve syscall %q: %w", name, err)
		}
		if err := filter.AddRule(sc, libseccomp.ActAllow); err != nil {
			return fmt.Errorf("seccomp: allow syscall %q: %w", name, err)
		}
	}

	// Load filter into kernel
	if err := filter.Load(); err != nil {
		return fmt.Errorf("seccomp: load filter: %w", err)
	}
	return nil
}
