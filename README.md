# Secure Sandbox

A runtime for secure, isolated workload execution, featuring a custom control plane, Linux namespace isolation, cgroup-based resource enforcement, and syscall-level sandboxing.

---

## Requirements

This project relies on Linux kernel primitives and libseccomp via CGO.

You must have:

* **Linux host environment** (native or VM)
* **Go installed** (matching your module version)
* **CGO enabled**
* **C toolchain** (e.g., `gcc`, `build-essential`)
* **libseccomp development library**

### Setup

Ensure CGO is enabled:

```
go env -w CGO_ENABLED=1
```

Install required system packages (Debian/Ubuntu example):

```
sudo apt-get update
sudo apt-get install -y build-essential pkg-config libseccomp-dev
```

> Note: This runtime depends on Linux-only features such as namespaces, cgroups, and seccomp. It will not run on macOS or Windows without a Linux VM.

---

## Overview

This runtime provides a minimal, security-focused execution environment for running workloads in isolated sandboxes on a single host. The system is designed around strict isolation boundaries, explicit control plane ownership, and in-depth defense mechanisms.

The runtime leverages native Linux primitives including namespaces, cgroups, and seccomp to constrain process behavior and reduce the attack surface of executed workloads.

---

## Architecture

```
sandbox (CLI client)
   ↓
HTTP over Unix domain socket (/run/sandboxd.sock)
   ↓
sandboxd (daemon / control plane)
   ↓
Manager (lifecycle orchestration)
   ↓
init process (re-exec boundary)
   ↓
Linux primitives (namespaces, cgroups, seccomp, mounts)
   ↓
workload
```

---

## Components

### sandboxd (Daemon)

The daemon is the control plane responsible for:

* Managing sandbox lifecycle
* Enforcing isolation boundaries
* Coordinating resource and security constraints
* Exposing a local HTTP API over a Unix domain socket

The daemon is the single authority for all sandbox operations.

---

### sandbox (CLI)

The CLI is a stateless client that:

* Translates user input into API requests
* Communicates with the daemon over a Unix socket
* Displays execution results and sandbox state

---

### Manager

The Manager encapsulates lifecycle operations:

* CreateSandbox
* StartSandbox
* StopSandbox
* GetSandbox
* ListSandboxes

It coordinates between the state store, cgroup subsystem, and execution layer.

---

### Init Process (Re-exec Boundary)

Workloads are executed through a dedicated init process using a re-exec model:

```
sandboxd → exec sandbox init → initproc → workload
```

This ensures that namespace setup, filesystem transitions, and security policies are applied within the correct process context before executing the workload.

---

## Isolation Model

The runtime enforces isolation using native Linux primitives:

### Namespaces

* Mount namespace
* PID namespace
* UTS namespace
* Extensible to network and user namespaces

### Cgroups

* CPU limits
* Memory limits
* PID limits

### Filesystem

* Root filesystem isolation
* pivot_root for sandboxed execution environment

### Seccomp

* Syscall filtering to restrict kernel surface area
* Default-deny policy with explicit allow rules

---

## API

The daemon exposes a resource-oriented HTTP API over a Unix domain socket.

### Endpoints

```
POST   /sandboxes              Create and start a sandbox
GET    /sandboxes              List sandboxes
GET    /sandboxes/{id}         Inspect sandbox
POST   /sandboxes/{id}/stop    Stop sandbox
POST   /shutdown               Shutdown daemon
```

---

## Build

```
go build -o sandboxd ./cmd/sandboxd
go build -o sandbox  ./cmd/sandbox
```

---

## Execution

### Start daemon

```
sudo ./sandboxd
```

### Run workload

```
sudo ./sandbox run ./bundle <command> [args]
```

### List sandboxes

```
sudo ./sandbox list
```

### Inspect sandbox

```
sudo ./sandbox inspect <id> [--logs]
```

### Stop sandbox

```
sudo ./sandbox stop <id>
```

### Shutdown daemon

```
sudo ./sandbox shutdown
```

---

## Bundle Format

A bundle defines the filesystem and execution configuration required for a sandbox. Each bundle must include a required `config.json` file and a `rootfs/` directory containing the filesystem that will become the sandbox root.

### Expected Structure

```
bundle/
  config.json
  rootfs/
    bin/
    proc/
    tmp/
    dev/
```

The runtime expects `rootfs/` to contain at least a **minimal Linux filesystem** sufficient enough to execute the configured workload after `pivot_root` is applied.

At minimum, the bundle should provide:

* `bin/` for executables needed inside the sandbox
* `proc/` as a mount point for `/proc`
* `tmp/` for temporary files
* `dev/` for basic device-related paths used by the environment

The runtime does not require a full distribution image, but it does expect the root filesystem to be **self-sufficient for the command being executed**.

### Configuration File

`config.json` is required and defines the default workload entrypoint and resource constraints.

Example:

```json
{
  "command": "ls",
  "args": ["-l"],
  "resources": {
    "cpu": 50,
    "memoryMB": 100,
    "pids": 64,
    "timeoutSec": 30
  }
}
```

The values in `config.json` act as the default execution contract for the bundle. Command, arguments, and resource settings may be overridden at runtime through flags via the CLI.
