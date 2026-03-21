package cli

import (
	"flag"
	"fmt"
	"sandbox-runtime/internal/manager"
	"sandbox-runtime/internal/sandbox"
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
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	var mem int
	var cpu int
	var pids int
	fs.IntVar(&mem, "memory", 0, "memory limit in MB")
	fs.IntVar(&cpu, "cpu", 0, "cpu percentage (1-100)")
	fs.IntVar(&pids, "pids", 0, "max number of processes")
	if err := fs.Parse(args); err != nil {
		fmt.Println("error parsing flags:", err)
		return
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Println("usage: sandbox run <bundlePath> [command] [args...] [--memory=N] [--cpu=N] [--pids=N]")
		return
	}
	bundlePath := remaining[0]

	// Optional override
	var cmd string
	var cmdArgs []string
	if len(remaining) > 1 {
		cmd = remaining[1]
		cmdArgs = remaining[2:]
	}

	var res *sandbox.ResourceSpec
	if mem > 0 || cpu > 0 || pids > 0 {
		res = &sandbox.ResourceSpec{
			MemoryMB: mem,
			CPU:      cpu,
			Pids:     pids,
		}
	}

	sb, err := c.mgr.CreateSandbox(manager.CreateSandboxRequest{
		BundlePath: bundlePath,
		Command:    cmd,
		Args:       cmdArgs,
		Resources:  res,
	})
	if err != nil {
		fmt.Println("error creating sandbox:", err)
		return
	}

	sb, err = c.mgr.StartSandbox(sb.ID)
	if err != nil {
		fmt.Println("error starting sandbox:", err)
		return
	}
	fmt.Printf("started sandbox: id=%s pid=%d state=%v\n", sb.ID, sb.PID, sb.State)
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
	fmt.Printf("ID: %s\nState: %v\n", sb.ID, sb.State)
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
