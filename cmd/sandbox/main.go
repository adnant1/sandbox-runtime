package main

import (
	"fmt"
	"os"
	"sandbox-runtime/internal/cli"
	"sandbox-runtime/internal/initproc"
)

// Unix domain socket used by sandboxd
const socketPath = "/run/sandboxd.sock"

// Main entrypoint into the sandbox CLI
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

	cli := cli.New(socketPath)
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "cli error: %v\n", err)
		os.Exit(1)
	}
}
