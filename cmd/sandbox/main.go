package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sandbox-runtime/internal/cgroups"
	"sandbox-runtime/internal/cli"
	"sandbox-runtime/internal/config"
	"sandbox-runtime/internal/initproc"
	"sandbox-runtime/internal/manager"
	"sandbox-runtime/internal/state"
)

// Main entrypoint into the runtime
func main() {
	// Internal bootstrap path.
	//
	// This binary is re-executed by the runtime using:
	//   /proc/self/exe init <sandbox-id> <rootfs> <cmd> [args...]
	//
	// This means main() runs twice:
	// 1. User invocation (e.g. "sandbox run ...") -> normal CLI path
	// 2. Child process (argv[1] == "init")		   -> enters this branch
	//
	// The "init" path runs inside the child process AFTER namespaces are applied.
	// It performs in-namespace setup (e.g. pivot_root, mounts, etc) and then execs
	// the real workload.
	//
	// This is required because operations like pivot_root must execute in the
	// child process, not the parent (StartSandbox).
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := initproc.Run(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "init error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	absRootDir, err := filepath.Abs("./sandbox-data")
	if err != nil {
		panic(fmt.Sprintf("failed to resolve root directory: %v", err))
	}
	cfg := config.Config{
		RootDir: absRootDir,
	}
	store := state.New()
	cg := cgroups.New("/sys/fs/cgroup")
	mgr := manager.New(store, cg, cfg)
	cli := cli.New(mgr)
	cli.Run(os.Args[1:])
}
