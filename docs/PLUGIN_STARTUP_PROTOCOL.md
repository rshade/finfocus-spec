# Plugin Startup Protocol

This document defines the startup protocol for FinFocus plugins, including port
configuration, environment variables, command-line flags, and port announcement.

## Overview

FinFocus plugins are gRPC servers that the core orchestrator (`finfocus-core`)
spawns and communicates with. The startup protocol ensures:

1. **Port Discovery**: Core can discover which port the plugin is listening on
2. **Configuration**: Plugins receive runtime configuration via environment variables
3. **Multi-Plugin Support**: Multiple plugins can run concurrently without port conflicts

## Port Configuration

### Priority Order

The plugin port is determined with the following priority:

| Priority | Source | Description |
|----------|--------|-------------|
| 1 | `--port` flag | Command-line argument (highest priority) |
| 2 | `FINFOCUS_PLUGIN_PORT` | Environment variable |
| 3 | Ephemeral | OS assigns an available port |

### Command-Line Flag: `--port`

The `--port` flag is the recommended way for orchestrators to assign ports:

```bash
# Assign specific port
./my-plugin --port 50051

# Plugin output:
# PORT=50051
```

**Implementation**:

```go
import (
    "os"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func main() {
    os.Exit(pluginsdk.Run(pluginsdk.ServeConfig{
        Plugin:     &MyPlugin{},
        PluginInfo: pluginsdk.NewPluginInfo("my-plugin", "v1.0.0"),
    }))
}
```

`Run()` parses `--port` on the handshake path and calls `Serve()`. `--help`,
`--version`, and `dry-run` go through ax-go and never write `PORT=<n>`.

Legacy binaries that still call `flag.Parse()`, `ParsePortFlag()`, and `Serve()`
keep working; new plugins should use `Run()`.

### Environment Variable: `FINFOCUS_PLUGIN_PORT`

When `--port` is not specified, the plugin checks `FINFOCUS_PLUGIN_PORT`:

```bash
export FINFOCUS_PLUGIN_PORT=50052
./my-plugin

# Plugin output:
# PORT=50052
```

**Important**: The generic `PORT` environment variable is **NOT supported**. This is
intentional to avoid conflicts when multiple plugins run on the same machine with
a shared environment.

### Ephemeral Port (Default)

When neither `--port` nor `FINFOCUS_PLUGIN_PORT` is set, the OS assigns an
available port:

```bash
./my-plugin

# Plugin output:
# PORT=54321  (random available port)
```

This is the recommended approach for dynamic orchestration where the core spawns
plugins and reads their announced ports.

## Port Announcement

Upon successful startup, the plugin **MUST** print its listening port to stdout:

```text
PORT=<port>
```

This announcement allows the orchestrator to:

1. Discover the port (especially for ephemeral ports)
2. Confirm the plugin has started successfully
3. Establish gRPC connection to the plugin

### Implementation Detail

The `pluginsdk.Serve()` function handles port announcement automatically:

```go
// Internal implementation (users don't need to do this)
fmt.Fprintf(os.Stdout, "PORT=%d\n", tcpAddr.Port)
```

### Parsing the Announcement

Orchestrators should parse the port announcement as follows:

```go
// Read first line from plugin stdout
line, _ := bufio.NewReader(pluginStdout).ReadString('\n')
if strings.HasPrefix(line, "PORT=") {
    port, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(line), "PORT="))
    // Connect to plugin at 127.0.0.1:port
}
```

## Environment Variables

### Complete Reference

| Variable | Purpose | Default | Example |
|----------|---------|---------|---------|
| `FINFOCUS_PLUGIN_PORT` | gRPC server port | Ephemeral | `50051` |
| `FINFOCUS_LOG_LEVEL` | Log verbosity | `info` | `debug`, `info`, `warn`, `error` |
| `FINFOCUS_LOG_FORMAT` | Log output format | `json` | `json`, `text` |
| `FINFOCUS_LOG_FILE` | Log file path | stderr | `/var/log/plugin.log` |
| `FINFOCUS_TRACE_ID` | Distributed trace ID | (none) | `abc123def456` |
| `FINFOCUS_TEST_MODE` | Enable test mode | `false` | `true`, `false` |

### Port Configuration

```bash
# Set plugin port via environment
export FINFOCUS_PLUGIN_PORT=50051
./my-plugin
```

**Behavior**:

- Only positive integers are valid
- Invalid values (non-numeric, zero, negative) are treated as "not set"
- Falls back to ephemeral port when invalid

### Logging Configuration

```bash
# Development setup
export FINFOCUS_LOG_LEVEL=debug
export FINFOCUS_LOG_FORMAT=text
./my-plugin

# Production setup
export FINFOCUS_LOG_LEVEL=info
export FINFOCUS_LOG_FORMAT=json
export FINFOCUS_LOG_FILE=/var/log/finfocus/aws-plugin.log
./my-plugin
```

**Log Level Fallback**: If `FINFOCUS_LOG_LEVEL` is not set, the SDK checks the
legacy `LOG_LEVEL` environment variable for backwards compatibility.

### Distributed Tracing

```bash
# Pass trace ID from orchestrator to plugin
export FINFOCUS_TRACE_ID=abc123def456
./my-plugin
```

The trace ID is automatically:

- Attached to all log entries
- Propagated in gRPC metadata
- Included in response headers

### Test Mode

```bash
# Enable test mode
export FINFOCUS_TEST_MODE=true
./my-plugin
```

**Behavior**:

- Only the exact string `"true"` enables test mode
- Any other value (including `"TRUE"`, `"1"`, `"yes"`) disables test mode
- A warning is logged for values other than `"true"` or `"false"`

## Multi-Plugin Orchestration

When running multiple plugins on the same host, avoid port conflicts:

### Option 1: Explicit Port Assignment (Recommended for Static Setups)

```bash
# Plugin 1 (AWS)
./aws-plugin --port 50051 &

# Plugin 2 (Azure)
./azure-plugin --port 50052 &

# Plugin 3 (GCP)
./gcp-plugin --port 50053 &
```

### Option 2: Ephemeral Ports (Recommended for Dynamic Orchestration)

```bash
# Start plugins and capture their ports
./aws-plugin &    # Announces PORT=43291
./azure-plugin &  # Announces PORT=39102
./gcp-plugin &    # Announces PORT=51847
```

The orchestrator reads each plugin's stdout to discover the assigned port.

### Option 3: Environment Variables per Plugin

```bash
# Each plugin in its own environment
FINFOCUS_PLUGIN_PORT=50051 ./aws-plugin &
FINFOCUS_PLUGIN_PORT=50052 ./azure-plugin &
FINFOCUS_PLUGIN_PORT=50053 ./gcp-plugin &
```

## Startup Sequence

### Plugin Startup Flow

```text
┌─────────────────────────────────────────────────────────────┐
│                    Plugin Startup Flow                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Parse command-line flags                                │
│     └─ pluginsdk.Run() handshake path (--port / serve)      │
│                                                             │
│  2. Resolve port (--port → env var → ephemeral)            │
│     └─ ServeConfig.Port (set from --port) or env var       │
│                                                             │
│  3. Create TCP listener on 127.0.0.1:port                  │
│     └─ Binds to loopback only for security                 │
│                                                             │
│  4. Announce port to stdout                                 │
│     └─ fmt.Fprintf(os.Stdout, "PORT=%d\n", port)           │
│                                                             │
│  5. Register gRPC services                                  │
│     └─ CostSourceServiceServer                             │
│                                                             │
│  6. Start serving (blocking)                                │
│     └─ grpcServer.Serve(listener)                          │
│                                                             │
│  7. Handle shutdown signal                                  │
│     └─ grpcServer.GracefulStop() on context cancellation   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Orchestrator Integration Flow

```text
┌─────────────────────────────────────────────────────────────┐
│               Orchestrator Integration Flow                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Spawn plugin process                                    │
│     └─ exec.Command("./my-plugin", "--port", "50051")      │
│                                                             │
│  2. Read PORT announcement from stdout                      │
│     └─ Parse "PORT=<port>\n"                               │
│                                                             │
│  3. Establish gRPC connection                               │
│     └─ grpc.Dial("127.0.0.1:<port>")                       │
│                                                             │
│  4. Call Name() to verify plugin identity                   │
│     └─ client.Name(ctx, &NameRequest{})                    │
│                                                             │
│  5. Register plugin in orchestrator registry                │
│     └─ Associate plugin with provider/region               │
│                                                             │
│  6. Route requests to plugin                                │
│     └─ GetProjectedCost, GetActualCost, etc.               │
│                                                             │
│  7. Shutdown: cancel context, wait for graceful stop        │
│     └─ Plugin calls GracefulStop() automatically           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Graceful Shutdown

The `pluginsdk.Serve()` function handles graceful shutdown automatically:

```go
import (
    "context"
    "errors"
    "os"
    "os/signal"
    "syscall"

    "github.com/rs/zerolog/log"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt,
        syscall.SIGTERM,
    )
    defer cancel()

    if err := pluginsdk.Serve(ctx, config); err != nil {
        if errors.Is(err, context.Canceled) {
            log.Info().Msg("Plugin shutdown complete")
            return
        }
        log.Fatal().Err(err).Msg("Server error")
    }
}
```

**Shutdown Behavior**:

1. Context cancellation triggers `grpcServer.GracefulStop()`
2. Server stops accepting new connections
3. Existing RPCs are allowed to complete
4. `Serve()` returns `context.Canceled`

## Error Handling

### Common Startup Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to listen: address already in use` | Port is taken | Use different port or ephemeral |
| `failed to listen: permission denied` | Port < 1024 requires root | Use port ≥ 1024 |
| `ParsePortFlag() returns 0 unexpectedly` | `flag.Parse()` not called | Call `flag.Parse()` first, or use `Run()` |

### Debugging Port Issues

```bash
# Check if port is in use
lsof -i :50051

# Check which process is using the port
netstat -tlnp | grep 50051

# Use ephemeral port to avoid conflicts
./my-plugin  # Let OS assign port
```

## Security Considerations

### Loopback Binding

Plugins bind to `127.0.0.1` (loopback) only:

```go
// Internal implementation
address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
listener, err := net.Listen("tcp", address)
```

This ensures:

- Plugins are not accessible from the network
- Only local processes can communicate with plugins
- Reduces attack surface in containerized environments

### Environment Variable Security

- Avoid passing secrets via environment variables when possible
- Use `FINFOCUS_TEST_MODE=true` only in test environments
- Log files (`FINFOCUS_LOG_FILE`) should have appropriate permissions

## Troubleshooting

### Windows-Specific Issues

On Windows systems, there are important differences in how `localhost` and `127.0.0.1` are resolved:

| Address | Behavior | Use Case |
|---------|----------|----------|
| `127.0.0.1` | IPv4 loopback (recommended) | Works consistently |
| `localhost` | May resolve to IPv6 (`::1`) | Avoid for explicit binding |
| `::1` | IPv6 loopback | Only if IPv6 is required |

**Common Issue**: Connection failures when orchestrator uses `localhost` but plugin binds to `127.0.0.1`.

**Solution**: Always use `127.0.0.1` explicitly in both plugin binding and orchestrator connection:

```go
// Orchestrator connecting to plugin
conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
```

### Multiple --port Flags

If a plugin receives multiple `--port` flags, only the first valid value is used:

```bash
# Only 50051 is used; 50052 is ignored
./my-plugin --port 50051 --port 50052

# Plugin output:
# PORT=50051
```

**Best Practice**: Ensure orchestrators pass only a single `--port` flag to avoid confusion.

### Debug Checklist

When plugins fail to start or connect, verify:

1. **Port availability**: `netstat -tlnp | grep <port>` (Linux) or `netstat -an | findstr <port>` (Windows)
2. **Address resolution**: Ensure both sides use `127.0.0.1` explicitly
3. **Environment variables**: `env | grep FINFOCUS` to check for conflicting settings
4. **Flag parsing**: Prefer `pluginsdk.Run()`; legacy mains must call `flag.Parse()` before `ParsePortFlag()`
5. **Firewall rules**: On Windows, check Windows Defender Firewall for localhost exceptions

## Quick Reference

### Minimal Plugin Implementation

```go
package main

import (
    "context"
    "os"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-cost-plugin" }

func (p *MyPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (
    *pbc.GetProjectedCostResponse, error) {
    return &pbc.GetProjectedCostResponse{}, nil
}

func (p *MyPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (
    *pbc.GetActualCostResponse, error) {
    return &pbc.GetActualCostResponse{}, nil
}

func (p *MyPlugin) GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (
    *pbc.GetPricingSpecResponse, error) {
    return &pbc.GetPricingSpecResponse{}, nil
}

func (p *MyPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (
    *pbc.EstimateCostResponse, error) {
    return &pbc.EstimateCostResponse{}, nil
}

func main() {
    os.Exit(pluginsdk.Run(pluginsdk.ServeConfig{
        Plugin:     &MyPlugin{},
        PluginInfo: pluginsdk.NewPluginInfo("my-cost-plugin", "v1.0.0"),
    }))
}
```

### Environment Setup Script

```bash
#!/bin/bash
# setup-plugin-env.sh

# Required
export FINFOCUS_PLUGIN_PORT=50051

# Logging (optional)
export FINFOCUS_LOG_LEVEL=info
export FINFOCUS_LOG_FORMAT=json
export FINFOCUS_LOG_FILE=/var/log/finfocus/plugin.log

# Tracing (optional)
export FINFOCUS_TRACE_ID=$(uuidgen)

# Start plugin
exec ./my-plugin "$@"
```
