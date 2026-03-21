package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sandbox-runtime/internal/api"
	"sandbox-runtime/internal/cgroups"
	"sandbox-runtime/internal/config"
	"sandbox-runtime/internal/manager"
	"sandbox-runtime/internal/state"
	"syscall"
	"time"
)

// Unix domain socket used by sandboxd
const socketPath = "/run/sandboxd.sock"

// entry point for the sandbox daemon (sandboxd).
//
// sandboxd acts as the control plane for the runtime. It owns the Manager,
// which is reponsible for all sandbox lifecyle operations (run, list, inspect, stop).
func main() {
	// Root directory containing all sandbox info, logs
	absRootDir, err := filepath.Abs("./sandbox-data")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve root directory: %v\n", err)
		os.Exit(1)
	}
	cfg := config.Config{
		RootDir: absRootDir,
	}
	store := state.New()
	cg := cgroups.New("/sys/fs/cgroup")
	mgr := manager.New(store, cg, cfg)

	server := api.New(mgr, socketPath)
	server.Debug = true // Development mode

	// Channel to capture OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		fmt.Println("sandboxd starting on", socketPath)

		if err := server.Start(); err != nil {
			fmt.Println("server error:", err)
			os.Exit(1)
		}
	}()
	select {
	case sig := <-sigCh:
		fmt.Println("received signal:", sig)
	case <-server.ShutdownCh:
		fmt.Println("received shutdown request")
	}

	// Cleanup sandboxes
	sandboxes, _ := mgr.ListSandboxes()
	for _, sb := range sandboxes {
		_, _ = mgr.StopSandbox(sb.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("shutdown error:", err)
	}
	fmt.Println("sandboxd exited cleanly")
}
