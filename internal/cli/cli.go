package cli

import (
	"errors"
	"flag"
	"fmt"
	"sandbox-runtime/internal/manager"
	"sandbox-runtime/internal/sandbox"
	"strings"
)

func printUsage() {
	fmt.Println("Usage: sandbox <command> [args]")
	fmt.Println("Commands: run, list, inspect, stop")
}

// CLI represents the command-line interface for interaction with the runtime.
type CLI struct {
	mgr *manager.Manager
}

func New(mgr *manager.Manager) *CLI {
	if mgr == nil {
		panic("cli: nil manager")
	}
	return &CLI{
		mgr: mgr,
	}
}

// Run parses command-line arguments and dispatches the appropriate command.
func (c *CLI) Run(args []string) {
	if len(args) == 0 {
		printUsage()
		return
	}

	cmd := args[0]
	switch cmd {
	case "run":
		c.runCommand(args[1:])
	case "list":
		c.listCommand(args[1:])
	case "inspect":
		c.inspectCommand(args[1:])
	case "stop":
		c.stopCommand(args[1:])
	default:
		fmt.Println("Unknown command:", cmd)
		printUsage()
	}
}

func (c *CLI) runCommand(args []string) {
	req, err := parseRunArgs(args)
	if err != nil {
		fmt.Println("usage: sandbox run <bundlePath> [command] [args...] [--memory=N] [--cpu=N] [--pids=N]")
		return
	}

	sb, err := c.mgr.CreateSandbox(req)
	if err != nil {
		fmt.Println("error creating sandbox:", err)
		return
	}
	sb, err = c.mgr.StartSandbox(sb.ID)
	if err != nil {
		fmt.Println("error starting sandbox:", err)
		return
	}

	fmt.Printf("started sandbox: id=%s pid=%d state=%s\n", sb.ID, sb.PID, sb.State.String())
}

func (c *CLI) listCommand(args []string) {
	sandboxes, err := c.mgr.ListSandboxes()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, sb := range sandboxes {
		fmt.Printf("%s\t%v\n", sb.ID, sb.State)
	}
}

func (c *CLI) inspectCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("missing sandbox id")
		return
	}

	id := args[0]
	sb, err := c.mgr.GetSandbox(id)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("ID: %s\nState: %s\n", sb.ID, sb.State.String())
}

func (c *CLI) stopCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("missing sandbox id")
		return
	}

	id := args[0]
	sb, err := c.mgr.StopSandbox(id)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("stopped sandbox:", sb.ID)
}

func isFlagKV(a string) bool {
	return strings.HasPrefix(a, "--memory=") ||
		strings.HasPrefix(a, "--cpu=") ||
		strings.HasPrefix(a, "--pids=")
}

// parseRunArgs parses CLI input for the `run` command into a structured
// CreateSandboxRequest, separating workload arguments from runtime
// resource override flags.
func parseRunArgs(args []string) (manager.CreateSandboxRequest, error) {
	if len(args) < 1 {
		return manager.CreateSandboxRequest{}, fmt.Errorf("missing bundle path")
	}

	bundlePath := args[0]
	rest := args[1:]

	// Split workload vs flags
	var workload []string
	var flagArgs []string
	flagStart := -1
	for i, a := range rest {
		if strings.HasPrefix(a, "--") {
			flagStart = i
			break
		}
	}
	if flagStart == 0 && len(rest) > 1 {
		// Flags were inputted before workload command + args
		// Propagate an empty error to trigger usage output only at the caller
		return manager.CreateSandboxRequest{}, errors.New("")
	}
	if flagStart == -1 {
		workload = rest
	} else {
		workload = rest[:flagStart]
		flagArgs = rest[flagStart:]
	}

	// Flag parsing
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var mem, cpu, pids int
	fs.IntVar(&mem, "memory", 0, "memory limit in MB")
	fs.IntVar(&cpu, "cpu", 0, "cpu percentage (1-100)")
	fs.IntVar(&pids, "pids", 0, "max number of processes")
	if err := fs.Parse(flagArgs); err != nil {
		return manager.CreateSandboxRequest{}, err
	}

	// Optional overrides
	var cmd string
	var cmdArgs []string
	if len(workload) > 0 {
		cmd = workload[0]
		cmdArgs = workload[1:]
	}
	var res *sandbox.ResourceSpec
	if mem > 0 || cpu > 0 || pids > 0 {
		res = &sandbox.ResourceSpec{
			MemoryMB: mem,
			CPU:      cpu,
			Pids:     pids,
		}
	}

	return manager.CreateSandboxRequest{
		BundlePath: bundlePath,
		Command:    cmd,
		Args:       cmdArgs,
		Resources:  res,
	}, nil
}
