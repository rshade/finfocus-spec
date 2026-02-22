# pluginsdk - FinFocus Plugin Development SDK

The `pluginsdk` package provides a comprehensive development SDK for building FinFocus plugins. It includes
the core plugin interface, helper utilities for cost calculations, structured logging with zerolog, and testing
utilities for plugin development.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Server Configuration](#server-configuration)
- [Multi-Protocol Support](#multi-protocol-support-grpc-grpc-web-connect)
- [Go Client SDK](#go-client-sdk)
- [Plugin Info (GetPluginInfo RPC)](#plugin-info-getplugininfo-rpc)
- [Environment Variables](#environment-variables)
- [Core Components](#core-components)
- [Structured Logging](#structured-logging)
- [Prometheus Metrics](#prometheus-metrics)
- [Testing Utilities](#testing-utilities)
- [Error Helpers](#error-helpers)
- [Pagination Helpers](#pagination-helpers)
- [FOCUS 1.2 Cost Records](#focus-12-cost-records)
- [FOCUS 1.3 Extensions](#focus-13-extensions)
- [Contract Commitment Dataset](#contract-commitment-dataset-focus-13)
- [Manifest Management](#manifest-management)
- [Property Mapping](#property-mapping-mapping-subpackage)
- [Dry Run Mode](#dry-run-mode)
- [Thread Safety](#thread-safety)
- [Rate Limiting](#rate-limiting)
- [Performance Tuning](#performance-tuning)
- [CORS Configuration](#cors-configuration)
- [Migration](#migration-from-finfocus-core)
- [API Reference](#api-reference)

## Installation

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
```

## Quick Start

The simplest possible plugin implementation:

```go
package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// MyPlugin implements the pluginsdk.Plugin interface.
type MyPlugin struct{}

func (p *MyPlugin) Name() string {
    return "my-cost-plugin"
}

func (p *MyPlugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    // Implement projected cost calculation
    return &pbc.GetProjectedCostResponse{}, nil
}

func (p *MyPlugin) GetActualCost(
    ctx context.Context,
    req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
    // Implement actual cost retrieval
    return &pbc.GetActualCostResponse{}, nil
}

func (p *MyPlugin) GetPricingSpec(
    ctx context.Context,
    req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
    // Implement pricing spec lookup
    return &pbc.GetPricingSpecResponse{}, nil
}

func (p *MyPlugin) EstimateCost(
    ctx context.Context,
    req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
    // Implement cost estimation
    return &pbc.EstimateCostResponse{}, nil
}

func main() {
    // Parse command-line flags (required before ParsePortFlag)
    flag.Parse()

    // Create cancellable context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle SIGINT and SIGTERM for graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        cancel()
    }()

    // Start the gRPC server
    if err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin: &MyPlugin{},
        Port:   pluginsdk.ParsePortFlag(), // Uses --port flag if provided
    }); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

## Server Configuration

The `pluginsdk.Serve()` function is the entry point for running your plugin as a gRPC server.

### Function Signature

```go
func Serve(ctx context.Context, config ServeConfig) error
```

`Serve` starts the gRPC server and blocks until the context is canceled or a fatal error occurs. It automatically handles:

- Port resolution (flag, env var, or ephemeral)
- Graceful shutdown on context cancellation
- Trace ID propagation
- Port announcement to stdout (`PORT=<port>`)

### ServeConfig

Configuration for the server is provided via the `ServeConfig` struct:

```go
type ServeConfig struct {
    // Required: The plugin implementation
    Plugin Plugin

    // Optional: TCP port to listen on.
    // If 0, uses FINFOCUS_PLUGIN_PORT env var or an ephemeral port.
    Port int

    // Optional: Registry for looking up plugins (used for Supports validation).
    // If nil, defaults to a no-op registry.
    Registry RegistryLookup

    // Optional: Custom logger.
    // If nil, a default stderr logger is used.
    Logger *zerolog.Logger

    // Optional: Additional gRPC unary interceptors.
    // Chained after the built-in tracing interceptor.
    UnaryInterceptors []grpc.UnaryServerInterceptor
}
```

### Port Resolution

The server port is determined with the following priority:

1. **Explicit configuration**: `ServeConfig.Port` (if > 0)
2. **Environment variable**: `FINFOCUS_PLUGIN_PORT` (if set)
3. **Ephemeral port**: The OS assigns an available port (if both above are 0/unset)

The generic `PORT` environment variable is **not supported** to avoid conflicts when multiple plugins run on the same machine.

### Port Announcement

Upon starting, the server prints the selected port to stdout in the format:

```text
PORT=50051
```

This allows the parent process (e.g., finfocus-core) to discover the ephemeral port.

### ParsePortFlag

The `pluginsdk` package provides a standard helper for the `--port` flag:

```go
// Returns value of --port flag (or 0 if not set)
port := pluginsdk.ParsePortFlag()
```

**Important**: You MUST call `flag.Parse()` before calling `pluginsdk.ParsePortFlag()`.

### Configuration Examples

**Using the `--port` flag:**

```bash
./my-plugin --port 50051
```

**Using Environment Variable:**

```bash
export FINFOCUS_PLUGIN_PORT=50052
./my-plugin
```

**Using Ephemeral Port:**

```bash
./my-plugin
# Output: PORT=54321
```

### Multi-Plugin Orchestration

When running multiple plugins on the same host (e.g., orchestrated by `finfocus-core`),
you must avoid port conflicts. This is why the generic `PORT` environment variable is **not supported**.

Instead, assign unique ports using the `--port` flag or let the OS assign ephemeral ports:

```bash
# Plugin 1 (AWS)
./aws-plugin --port 50051 &

# Plugin 2 (Azure)
./azure-plugin --port 50052 &
```

Or using ephemeral ports (recommended for dynamic orchestration):

```bash
# Plugins report their ports to stdout
./aws-plugin    # Output: PORT=43291
./azure-plugin  # Output: PORT=39102
```

### Graceful Shutdown

The `Serve()` function monitors the provided `context.Context`. When the context is canceled (e.g., via SIGINT/SIGTERM handling):

1. The server stops accepting new connections.
2. `grpcServer.GracefulStop()` is called.
3. Existing RPCs are allowed to complete.
4. The function returns `context.Canceled` (or the cancellation error).

This behavior ensures no in-flight requests are dropped during rolling updates or shutdown.

## Multi-Protocol Support (gRPC, gRPC-Web, Connect)

The SDK supports serving plugins over multiple protocols simultaneously using the
[Connect](https://connectrpc.com/) framework:

| Protocol     | Use Case                   | Client Support                    |
| ------------ | -------------------------- | --------------------------------- |
| **gRPC**     | Server-to-server (HTTP/2)  | All gRPC clients                  |
| **gRPC-Web** | Browser clients (HTTP/1.1) | grpc-web, @connectrpc/connect-web |
| **Connect**  | Simple JSON over HTTP      | fetch(), curl, any HTTP client    |

### Enabling Web Support

To enable browser access (gRPC-Web and Connect protocols), set `Web.Enabled = true` in your server config:

```go
err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
    Plugin: &MyPlugin{},
    Port:   8080,
    Web: pluginsdk.WebConfig{
        Enabled:              true,              // Enable web protocols
        AllowedOrigins:       []string{"*"},     // CORS origins (use specific domains in production)
        AllowCredentials:     true,              // Allow cookies/auth headers
        EnableHealthEndpoint: true,              // Add /healthz endpoint
    },
})
```

### WebConfig Options

| Field                  | Type       | Default | Description                                    |
| ---------------------- | ---------- | ------- | ---------------------------------------------- |
| `Enabled`              | `bool`     | `false` | Enable Connect/gRPC-Web protocols              |
| `AllowedOrigins`       | `[]string` | `nil`   | CORS allowed origins (empty = no CORS headers) |
| `AllowCredentials`     | `bool`     | `false` | Include credentials in CORS                    |
| `EnableHealthEndpoint` | `bool`     | `false` | Add `/healthz` health check endpoint           |
| `AllowedHeaders`       | `[]string` | `nil`   | Custom allowed headers (nil = defaults)        |
| `ExposedHeaders`       | `[]string` | `nil`   | Custom exposed headers (nil = defaults)        |

### Builder Pattern

Use the fluent builder for configuration:

```go
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithAllowCredentials(true).
    WithHealthEndpoint(true).
    WithAllowedHeaders([]string{"Content-Type", "Authorization"}).
    WithExposedHeaders([]string{"X-Request-ID"})
```

### Calling from Browsers

With `Web.Enabled = true`, you can call your plugin using simple `fetch()`:

```javascript
// Get plugin name
const response = await fetch(
  "http://localhost:8080/finfocus.v1.CostSourceService/Name",
  {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  },
);
const data = await response.json();
console.log(data.name); // "my-cost-plugin"
```

### Calling with curl

```bash
# Get plugin name
curl -X POST http://localhost:8080/finfocus.v1.CostSourceService/Name \
  -H "Content-Type: application/json" \
  -d '{}'

# Estimate cost
curl -X POST http://localhost:8080/finfocus.v1.CostSourceService/EstimateCost \
  -H "Content-Type: application/json" \
  -d '{"resource_type": "aws:ec2/instance:Instance"}'
```

### Legacy Mode

When `Web.Enabled = false` (default), the server uses pure gRPC over HTTP/2. This is suitable for:

- Server-to-server communication with gRPC clients
- Environments requiring binary protobuf only
- Maximum performance (no HTTP/1.1 overhead)

## Go Client SDK

The SDK provides a convenient Go client for communicating with FinFocus plugins:

### Quick Start

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Create client using Connect protocol (recommended)
client := pluginsdk.NewConnectClient("http://localhost:8080")

// Get plugin name
name, err := client.Name(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Plugin:", name)

// Estimate cost
resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
    ResourceType: "aws:ec2/instance:Instance",
})
fmt.Printf("Monthly cost: %s %.2f\n", resp.Currency, resp.CostMonthly)
```

### Protocol Selection

Create clients using different protocols:

```go
// Connect protocol (JSON over HTTP, best browser compatibility)
client := pluginsdk.NewConnectClient("http://localhost:8080")

// gRPC protocol (HTTP/2, best for server-to-server)
client := pluginsdk.NewGRPCClient("http://localhost:8080")

// gRPC-Web protocol (HTTP/1.1 compatible gRPC)
client := pluginsdk.NewGRPCWebClient("http://localhost:8080")
```

### Client Configuration

For advanced configuration:

```go
import (
    "net/http"
    "time"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

cfg := pluginsdk.ClientConfig{
    BaseURL:  "http://localhost:8080",
    Protocol: pluginsdk.ProtocolConnect,
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second,
    },
}
client := pluginsdk.NewClient(cfg)
```

### Available Methods

| Method                                    | Description                             |
| ----------------------------------------- | --------------------------------------- |
| `Name(ctx)`                               | Get plugin name                         |
| `Supports(ctx, resource)`                 | Check resource support                  |
| `SupportsResourceType(ctx, resourceType)` | Convenience for checking by type string |
| `EstimateCost(ctx, req)`                  | Estimate monthly cost                   |
| `GetActualCost(ctx, req)`                 | Get historical cost data                |
| `GetProjectedCost(ctx, req)`              | Get projected cost                      |
| `GetPricingSpec(ctx, req)`                | Get pricing specification               |
| `GetRecommendations(ctx, req)`            | Get cost recommendations                |
| `DismissRecommendation(ctx, req)`         | Dismiss a recommendation                |
| `GetBudgets(ctx, req)`                    | Get budget information                  |
| `Inner()`                                 | Access underlying connect client        |

### Using the Raw Connect Client

For advanced use cases, access the underlying connect-generated client:

```go
import (
    "connectrpc.com/connect"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
    "google.golang.org/protobuf/types/known/structpb"
)

inner := client.Inner()

// Attributes is a protobuf Struct, not a Go map
attrs, _ := structpb.NewStruct(map[string]any{
    "instanceType": "t3.micro",
    "region":       "us-east-1",
})

resp, err := inner.EstimateCost(ctx, connect.NewRequest(&pbc.EstimateCostRequest{
    ResourceType: "aws:ec2/instance:Instance",
    Attributes:   attrs,
}))
```

#### Example: Signal Handling

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

if err := pluginsdk.Serve(ctx, config); err != nil {
    // ...
}
```

### Error Handling

`Serve()` may return the following errors:

- **"failed to listen: address already in use"**: The configured port is taken.
- **"failed to listen: permission denied"**: The process lacks permission to bind to the port (e.g., ports < 1024).
- **context.Canceled**: The server shut down gracefully due to context cancellation.

**Common Mistake**: Calling `ParsePortFlag()` before `flag.Parse()` will always return 0,
causing the server to use an ephemeral port (or env var) unexpectedly. Always call `flag.Parse()` first.

## Plugin Info (GetPluginInfo RPC)

The `GetPluginInfo` RPC allows the Core System to retrieve metadata about the plugin, including its
version, the spec version it was compiled against, and supported providers.

### Configuration via ServeConfig

The simplest way to support `GetPluginInfo` is to provide the `PluginInfo` struct in your
`ServeConfig`. The `Serve` function will automatically handle the RPC using this data.

```go
config := pluginsdk.ServeConfig{
    Plugin: &MyPlugin{},
    // Metadata for GetPluginInfo RPC
    PluginInfo: &pluginsdk.PluginInfo{
        Name:        "aws-cost-plugin",
        Version:     "1.2.0",
        SpecVersion: pluginsdk.SpecVersion,
        Providers:   []string{"aws"},
        Metadata: map[string]string{
            "build_hash": "abc1234",
        },
    },
}
```

### Dynamic Metadata (Advanced)

If your plugin needs to determine metadata at runtime (e.g., list of providers depends on
configuration), you can implement the `PluginInfoProvider` interface directly on your plugin struct.
This takes precedence over `ServeConfig.PluginInfo`.

```go
// Implement PluginInfoProvider interface
func (p *MyPlugin) GetPluginInfo(
    ctx context.Context,
    req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
    return &pbc.GetPluginInfoResponse{
        Name:        p.Name(),
        Version:     "1.0.0",
        SpecVersion: pluginsdk.SpecVersion,
        Providers:   p.detectProviders(), // Dynamic
    }, nil
}
```

### Spec Version

You should always use the `pluginsdk.SpecVersion` constant to ensure your plugin reports the
correct SDK version it was compiled against.

### Migrating to GetPluginInfo

The `GetPluginInfo` RPC is a new requirement for all plugins. It allows the core system to
discover capabilities and validate compatibility.

#### Static Metadata (Recommended)

For most plugins, metadata is static. Use the `NewPluginInfo` helper:

```go
pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
    Plugin: &MyPlugin{},
    // Add this field:
    PluginInfo: pluginsdk.NewPluginInfo(
        "my-cost-plugin",
        "v1.0.0",
        pluginsdk.WithProviders("aws", "azure"),
        pluginsdk.WithDescription("Cost analysis for AWS and Azure"),
    ),
})
```

#### Dynamic Metadata

If your plugin's capabilities change at runtime (e.g., based on credentials), implement
the `PluginInfoProvider` interface:

```go
func (p *MyPlugin) GetPluginInfo(ctx context.Context, req *pbc.GetPluginInfoRequest) (
    *pbc.GetPluginInfoResponse, error) {
    return &pbc.GetPluginInfoResponse{
        Name:        "my-cost-plugin",
        Version:     "v1.0.0",
        SpecVersion: pluginsdk.SpecVersion,
        Providers:   p.discoverProviders(), // Dynamic logic
    }, nil
}
```

#### Backward Compatibility

Legacy plugins that do not implement `GetPluginInfo` will return a gRPC `Unimplemented`
status. Clients should handle this gracefully:

```go
info, err := client.GetPluginInfo(ctx, req)
if status.Code(err) == codes.Unimplemented {
    // Legacy plugin - assume default behavior
}
```

## Developer Experience Improvements

The SDK includes several helpers to simplify plugin development.

### Health Checking

Plugins can implement the `HealthChecker` interface to provide custom health logic:

```go
func (p *MyPlugin) Check(ctx context.Context) error {
    if err := p.db.PingContext(ctx); err != nil {
        return fmt.Errorf("database unavailable: %w", err)
    }
    return nil
}
```

The SDK automatically registers a `/healthz` endpoint when `Web.EnableHealthEndpoint` is true.

### Context Validation

Validate contexts before performing work to fail fast:

```go
func (p *MyPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
    if err := pluginsdk.ValidateContext(ctx); err != nil {
        return nil, status.Error(codes.Canceled, err.Error())
    }
    // ...
}
```

### ARN Helpers

Parse and validate cloud resource identifiers:

```go
provider := pluginsdk.DetectARNProvider(resource.Id)
if provider == "aws" { // Returns "" for unrecognized formats
    // Handle AWS resource
}

if err := pluginsdk.ValidateARNConsistency(resource.Id, "aws"); err != nil {
    return nil, status.Error(codes.InvalidArgument, "invalid ARN for provider")
}
```

### Client Timeouts

Configure timeouts when creating clients:

```go
client := pluginsdk.NewClient(pluginsdk.ClientConfig{
    BaseURL: "http://localhost:8080",
    Timeout: 2 * time.Minute, // Override default 30s
})
```

## Capability Discovery

The SDK automatically discovers capabilities by checking which interfaces your plugin implements.
This simplifies configuration and ensures metadata matches implementation.

### Automatic Discovery

Simply implement the standard interfaces:

| Interface                 | Required Method         | Capability Enum                             |
| ------------------------- | ----------------------- | ------------------------------------------- |
| `DryRunHandler`           | `HandleDryRun`          | `PLUGIN_CAPABILITY_DRY_RUN`                 |
| `RecommendationsProvider` | `GetRecommendations`    | `PLUGIN_CAPABILITY_RECOMMENDATIONS`         |
| `BudgetsProvider`         | `GetBudgets`            | `PLUGIN_CAPABILITY_BUDGETS`                 |
| `DismissProvider`         | `DismissRecommendation` | `PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS` |

```go
// Example: Implementing DryRunHandler
func (p *MyPlugin) HandleDryRun(req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
    // ...
    return &pbc.DryRunResponse{}, nil
}

func main() {
    // PLUGIN_CAPABILITY_DRY_RUN is automatically added to capabilities
    info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0")
    pluginsdk.Serve(context.Background(), pluginsdk.ServeConfig{
        Plugin: &MyPlugin{},
        PluginInfo: info,
    })
}
```

### Manual Override

To override auto-discovery (e.g., to disable a feature or when using a proxy), use `WithCapabilities()`:

```go
info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0",
    // Explicitly set capabilities (ignores interfaces)
    pluginsdk.WithCapabilities(
        pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
    ),
)
```

**Backward Compatibility**: The SDK automatically populates the legacy `metadata` map
(e.g., `"supports_dry_run": "true"`) derived from your capabilities to support older hosts.

## Environment Variables

Plugins can be configured using standard environment variables. The SDK provides backward compatibility
for legacy environment variables during the migration from PulumiCost to FinFocus.

| Variable               | Legacy Fallback          | Purpose                                          | Default       |
| ---------------------- | ------------------------ | ------------------------------------------------ | ------------- |
| `FINFOCUS_PLUGIN_PORT` | `PULUMICOST_PLUGIN_PORT` | gRPC server port (overridden by `--port`)        | Ephemeral (0) |
| `FINFOCUS_LOG_LEVEL`   | `PULUMICOST_LOG_LEVEL`   | Log verbosity (`debug`, `info`, `warn`, `error`) | `info`        |
| `FINFOCUS_LOG_FORMAT`  | `PULUMICOST_LOG_FORMAT`  | Log output format (`json`, `text`)               | `json`        |
| `FINFOCUS_LOG_FILE`    | `PULUMICOST_LOG_FILE`    | Path to log file (empty = stderr)                | stderr        |
| `FINFOCUS_TRACE_ID`    | `PULUMICOST_TRACE_ID`    | Distributed trace ID for correlation             | (none)        |
| `FINFOCUS_TEST_MODE`   | `PULUMICOST_TEST_MODE`   | Enable test mode features (`true` / `false`)     | `false`       |

### Backward Compatibility

When a `FINFOCUS_*` environment variable is not set, the SDK automatically checks for the corresponding
`PULUMICOST_*` legacy variable. If a legacy variable is used, the SDK logs a **deprecation warning**
advising migration to the new canonical variable.

### Logging Configuration

- **FINFOCUS_LOG_LEVEL**: Controls the verbosity of the logger.
  - `debug`: Detailed debugging information
  - `info`: Standard operational events
  - `warn`: Warning conditions
  - `error`: Error conditions
  - **Fallback**: If `FINFOCUS_LOG_LEVEL` is not set, `GetLogLevel()` checks `PULUMICOST_LOG_LEVEL`,
    then finally falls back to the generic `LOG_LEVEL` environment variable.
- **FINFOCUS_LOG_FORMAT**: Controls the output structure.
  - `json`: Structured JSON for production (default)
  - `text`: Human-readable text for development
- **FINFOCUS_LOG_FILE**: Redirects logs to a file instead of stderr.
- **FINFOCUS_TEST_MODE**: Enables test mode features. Only `"true"` enables test mode; all other
  values disable it. `GetTestMode()` logs a warning when the value is set but is not `"true"` or
  `"false"`.

### Distributed Tracing

- **FINFOCUS_TRACE_ID**: If set, this ID is automatically attached to the logger and propagated
  in gRPC contexts. This allows correlating plugin activity with the caller's trace.

### Helper Functions

The SDK provides getter functions to access these values type-safely:

```go
port := pluginsdk.GetPort()           // Returns int
level := pluginsdk.GetLogLevel()      // Returns string (checks FINFOCUS_LOG_LEVEL, falls back to LOG_LEVEL)
format := pluginsdk.GetLogFormat()    // Returns string
file := pluginsdk.GetLogFile()        // Returns string
traceID := pluginsdk.GetTraceID()     // Returns string
isTest := pluginsdk.GetTestMode()     // Returns bool, logs warnings for invalid values
isTest := pluginsdk.IsTestMode()      // Returns bool, silent version for repeated checks
```

## Core Components

### Plugin Interface

The `Plugin` interface defines the core methods that every FinFocus plugin must implement:

```go
type Plugin interface {
    Name() string
    GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error)
    GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error)
    GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error)
    EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error)
}
```

### Optional Interfaces

Plugins can optionally implement these interfaces to provide advanced capabilities:

**SupportsProvider** - Enables the plugin to declare which resources it supports.

```go
type SupportsProvider interface {
    Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error)
}
```

**RecommendationsProvider** - Enables the plugin to provide cost optimization recommendations.

```go
type RecommendationsProvider interface {
    GetRecommendations(ctx context.Context, req *pbc.GetRecommendationsRequest) (*pbc.GetRecommendationsResponse, error)
}
```

### BasePlugin

`BasePlugin` provides a scaffold with default implementations for all methods. Extend it and override
the methods you need:

```go
plugin := pluginsdk.NewBasePlugin("my-plugin")

// Access helpers
matcher := plugin.Matcher()      // ResourceMatcher for filtering resources
calc := plugin.Calculator()      // CostCalculator for cost computations
name := plugin.Name()            // Returns "my-plugin"
```

### ResourceMatcher

Helps filter which resources your plugin supports:

```go
matcher := pluginsdk.NewResourceMatcher()
matcher.AddProvider("aws")
matcher.AddProvider("azure")
matcher.AddResourceType("aws:ec2:Instance")
matcher.AddResourceType("azure:compute:VirtualMachine")

// Check if a resource is supported
if matcher.Supports(resource) {
    // Process the resource
}
```

**Thread Safety**: ResourceMatcher is NOT safe for concurrent use. Configure it during plugin
initialization before calling `Serve()`.

### CostCalculator

Provides utilities for cost calculations:

```go
calc := pluginsdk.NewCostCalculator()

// Convert hourly to monthly (using 730 hours/month)
monthly := calc.HourlyToMonthly(0.10)  // Returns 73.0

// Convert monthly to hourly
hourly := calc.MonthlyToHourly(73.0)   // Returns 0.10

// Create standard response
resp := calc.CreateProjectedCostResponse("USD", 0.10, "Hourly pricing")
```

### Constants

```go
pluginsdk.HoursPerMonth  // 730.0 - standard hours for monthly calculations
```

## Structured Logging

The SDK uses zerolog for structured logging with standardized field names.

### Log File Configuration

By default, plugins log to stderr. Set `FINFOCUS_LOG_FILE` to redirect logs to a file:

```bash
# Direct all plugin logs to a file
export FINFOCUS_LOG_FILE=/var/log/finfocus/plugins.log
./my-plugin

# Or per-plugin for separate log files
FINFOCUS_LOG_FILE=/var/log/finfocus/aws.log ./aws-plugin &
FINFOCUS_LOG_FILE=/var/log/finfocus/azure.log ./azure-plugin &
```

When the environment variable is not set or empty, logs go to stderr (default behavior).

### NewLogWriter

Use `NewLogWriter()` to get an `io.Writer` that respects the log file configuration:

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Get writer that respects FINFOCUS_LOG_FILE
writer := pluginsdk.NewLogWriter()

// Create logger with the configured writer
logger := pluginsdk.NewPluginLogger(
    "my-plugin",
    "v1.0.0",
    zerolog.InfoLevel,
    writer,
)
```

**Behavior**:

| Scenario                             | Result                                                     |
| ------------------------------------ | ---------------------------------------------------------- |
| `FINFOCUS_LOG_FILE` not set          | Returns `os.Stderr`                                        |
| `FINFOCUS_LOG_FILE=""` (empty)       | Returns `os.Stderr`                                        |
| `FINFOCUS_LOG_FILE=/valid/path.log`  | Returns file writer (creates if needed, appends if exists) |
| `FINFOCUS_LOG_FILE=/invalid/path`    | Logs warning to stderr, returns `os.Stderr`                |
| `FINFOCUS_LOG_FILE=/some/directory/` | Logs warning to stderr, returns `os.Stderr`                |

**File Handling**:

- Files are created with `0644` permissions
- Existing files are appended to (not truncated)
- Multiple plugins can safely write to the same log file (append mode)
- Parent directories must exist (SDK does not create them)

### Creating a Plugin Logger

```go
import "github.com/rs/zerolog"

// Create a configured plugin logger
logger := pluginsdk.NewPluginLogger(
    "my-plugin",           // Plugin name
    "v1.0.0",              // Version
    zerolog.InfoLevel,     // Log level
    os.Stderr,             // Output writer (nil for os.Stderr)
)

// Standard field constants
logger.Info().
    Str(pluginsdk.FieldTraceID, traceID).
    Str(pluginsdk.FieldOperation, "GetProjectedCost").
    Str(pluginsdk.FieldResourceType, "aws:ec2:Instance").
    Float64(pluginsdk.FieldCostMonthly, 73.0).
    Msg("calculated cost")
```

### Available Field Constants

| Constant             | Value            | Description              |
| -------------------- | ---------------- | ------------------------ |
| `FieldTraceID`       | `trace_id`       | Request trace identifier |
| `FieldComponent`     | `component`      | System component         |
| `FieldOperation`     | `operation`      | RPC operation name       |
| `FieldDurationMs`    | `duration_ms`    | Operation duration       |
| `FieldResourceURN`   | `resource_urn`   | Pulumi resource URN      |
| `FieldResourceType`  | `resource_type`  | Resource type            |
| `FieldProvider`      | `provider`       | Cloud provider           |
| `FieldRegion`        | `region`         | Cloud region             |
| `FieldPluginName`    | `plugin_name`    | Plugin identifier        |
| `FieldPluginVersion` | `plugin_version` | Plugin version           |
| `FieldCostMonthly`   | `cost_monthly`   | Monthly cost value       |
| `FieldAdapter`       | `adapter`        | Adapter name             |
| `FieldErrorCode`     | `error_code`     | Error code               |

### Trace ID Propagation

The SDK includes a gRPC interceptor for trace ID propagation:

```go
// Interceptor automatically extracts/generates trace IDs
// Already integrated into Serve() function

// Access trace ID in your handlers
traceID := pluginsdk.TraceIDFromContext(ctx)

// Manually inject trace ID (for testing)
ctx = pluginsdk.ContextWithTraceID(ctx, "abc123def456...")

// Generate a new trace ID
traceID, err := pluginsdk.GenerateTraceID()
```

### Operation Timing

```go
done := pluginsdk.LogOperation(logger, "GetProjectedCost")
defer done()  // Logs operation completion with duration
```

## Prometheus Metrics

The SDK provides optional Prometheus metrics instrumentation for monitoring plugin performance.

### Quick Start

Add the metrics interceptor to your plugin:

```go
config := pluginsdk.ServeConfig{
    Plugin: plugin,
    UnaryInterceptors: []grpc.UnaryServerInterceptor{
        pluginsdk.MetricsUnaryServerInterceptor("my-plugin"),
    },
}
```

### Exposing Metrics

#### Option A: Built-in HTTP Server (Simple)

```go
server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
    Port: 9090,  // Default: 9090
})
if err != nil {
    log.Fatal(err)
}
defer server.Shutdown(context.Background())
```

Then scrape `http://localhost:9090/metrics`.

#### Option B: Custom Registry (Production)

```go
metrics := pluginsdk.NewPluginMetrics("my-plugin")
interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

// Add to your existing HTTP server
http.Handle("/metrics", promhttp.HandlerFor(
    metrics.Registry,
    promhttp.HandlerOpts{},
))
```

### Available Metrics

| Metric                                     | Type      | Labels                                    | Description                  |
| ------------------------------------------ | --------- | ----------------------------------------- | ---------------------------- |
| `finfocus_plugin_requests_total`           | Counter   | `grpc_method`, `grpc_code`, `plugin_name` | Total gRPC requests          |
| `finfocus_plugin_request_duration_seconds` | Histogram | `grpc_method`, `plugin_name`              | Request latency distribution |

**Histogram Buckets**: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s

### Example PromQL Queries

```promql
# Request rate by method
sum(rate(finfocus_plugin_requests_total[5m])) by (grpc_method)

# Error rate
sum(rate(finfocus_plugin_requests_total{grpc_code!="OK"}[5m]))
/ sum(rate(finfocus_plugin_requests_total[5m]))

# P99 latency by method
histogram_quantile(0.99, sum(rate(finfocus_plugin_request_duration_seconds_bucket[5m])) by (le, grpc_method))
```

### Performance

Benchmark results show <1% overhead for realistic handler workloads:

```text
BenchmarkMetricsInterceptor_Overhead    1000000    1391 ns/op    0 B/op    0 allocs/op
BenchmarkMetricsInterceptor_NoMetrics   830516698  1.82 ns/op    0 B/op    0 allocs/op
```

For handlers with typical work (1ms+), the metrics overhead is under 1% of total request time.

### Metrics Functions

| Function                                    | Description                              |
| ------------------------------------------- | ---------------------------------------- |
| `NewPluginMetrics(pluginName)`              | Create metrics with custom registry      |
| `MetricsUnaryServerInterceptor(pluginName)` | Create interceptor with default registry |
| `MetricsInterceptorWithRegistry(metrics)`   | Create interceptor with custom registry  |
| `StartMetricsServer(config)`                | Start optional HTTP metrics server       |

### Metrics Constants

| Constant             | Value      | Description          |
| -------------------- | ---------- | -------------------- |
| `MetricNamespace`    | `finfocus` | Prometheus namespace |
| `MetricSubsystem`    | `plugin`   | Prometheus subsystem |
| `DefaultMetricsPort` | `9090`     | Default HTTP port    |
| `DefaultMetricsPath` | `/metrics` | Default URL path     |

## Testing Utilities

The SDK provides testing utilities for plugin development:

### TestPlugin (Quick Testing)

```go
func TestMyPlugin(t *testing.T) {
    plugin := NewMyPlugin()

    // Creates an in-process gRPC server with cleanup
    tp := pluginsdk.NewTestPlugin(t, plugin)

    // Test name
    tp.TestName("my-plugin")

    // Test projected cost (expectError=false for success)
    resource := pluginsdk.CreateTestResource("aws", "aws:ec2:Instance", nil)
    resp := tp.TestProjectedCost(resource, false)

    // Validate response
    if resp.GetCostPerMonth() != 73.0 {
        t.Errorf("unexpected cost: %f", resp.GetCostPerMonth())
    }
}
```

### TestServer (Lower-Level Control)

```go
func TestWithServer(t *testing.T) {
    plugin := NewMyPlugin()
    ts := pluginsdk.NewTestServer(t, plugin)
    defer ts.Close()

    // Use the gRPC client directly
    client := ts.Client()
    resp, err := client.Name(ctx, &pbc.NameRequest{})
    // ...
}
```

### CreateTestResource

```go
// Create a test resource with properties
resource := pluginsdk.CreateTestResource(
    "aws",
    "aws:ec2:Instance",
    map[string]string{
        "instance_type": "t2.micro",
        "region":        "us-east-1",
    },
)
```

### Conformance Testing

The pluginsdk package provides adapter functions for running conformance tests directly on your
`Plugin` implementation without manual conversion to the gRPC server interface:

```go
func TestPluginConformance(t *testing.T) {
    plugin := NewMyPlugin()

    // Run basic conformance (core functionality)
    result, err := pluginsdk.RunBasicConformance(plugin)
    if err != nil {
        t.Fatalf("Conformance test error: %v", err)
    }

    // Print formatted report
    pluginsdk.PrintConformanceReport(t, result)

    if !result.Passed() {
        t.Errorf("Basic conformance failed: %d/%d tests passed",
            result.Summary.Passed, result.Summary.Total)
    }
}
```

**Conformance Levels**:

| Level                            | Description                                | Use Case                       |
| -------------------------------- | ------------------------------------------ | ------------------------------ |
| `RunBasicConformance(plugin)`    | Core functionality, required for all       | Minimum validation             |
| `RunStandardConformance(plugin)` | Production-ready (includes error handling) | Production deployments         |
| `RunAdvancedConformance(plugin)` | High performance (strict latency limits)   | Performance-critical scenarios |

**Type Aliases**:

The package re-exports key types from `sdk/go/testing` for convenience:

```go
// Use directly from pluginsdk
var level pluginsdk.ConformanceLevel = pluginsdk.ConformanceLevelStandard
var result *pluginsdk.ConformanceResult
var summary pluginsdk.ResultSummary
```

**Complete Example**:

```go
func TestConformance(t *testing.T) {
    plugin := NewMyPlugin()

    t.Run("Basic", func(t *testing.T) {
        result, err := pluginsdk.RunBasicConformance(plugin)
        if err != nil {
            t.Fatalf("Error: %v", err)
        }
        if result.Summary.Failed > 0 {
            pluginsdk.PrintConformanceReport(t, result)
            t.Fail()
        }
    })

    t.Run("Standard", func(t *testing.T) {
        result, err := pluginsdk.RunStandardConformance(plugin)
        if err != nil {
            t.Fatalf("Error: %v", err)
        }
        pluginsdk.PrintConformanceReport(t, result)
        if result.LevelAchieved < pluginsdk.ConformanceLevelStandard {
            t.Errorf("Expected Standard conformance, achieved: %s",
                result.LevelAchievedStr)
        }
    })
}
```

### Advanced Testing with sdk/go/testing

For more advanced testing scenarios including mock plugins with error injection,
custom configurations, and performance benchmarks, use the `sdk/go/testing` package directly:

```go
import plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"

// Create configurable mock
mock := plugintesting.NewMockPlugin()
mock.ShouldErrorOnName = true  // Inject errors

// Convert plugin to server and use test harness for in-memory gRPC
plugin := NewMyPlugin()
server := pluginsdk.NewServer(plugin)
harness := plugintesting.NewTestHarness(server)
harness.Start(t)
defer harness.Stop()
```

## Error Helpers

```go
// Resource not supported
err := pluginsdk.NotSupportedError(resource)
// Returns: "resource type X from provider Y is not supported"

// No cost data available
err := pluginsdk.NoDataError("resource-id")
// Returns: "no cost data available for resource resource-id"
```

## Response Builders and Validation

### NewActualCostResponse

Create `GetActualCostResponse` using functional options for explicit control:

```go
import (
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Plugin found cost data - signal no fallback needed
resp := pluginsdk.NewActualCostResponse(
    pluginsdk.WithResults(results),
    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
)

// Plugin has no data - recommend trying other plugins
resp := pluginsdk.NewActualCostResponse(
    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
)

// Plugin cannot handle this resource type at all
resp := pluginsdk.NewActualCostResponse(
    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_REQUIRED),
)
```

### FallbackHint Enum

The `FallbackHint` enum signals to the core system whether it should query other plugins:

| Value | Description |
|-------|-------------|
| `FALLBACK_HINT_UNSPECIFIED` (0) | Default. Treated as "no fallback needed" |
| `FALLBACK_HINT_NONE` (1) | Plugin has data; do not attempt fallback |
| `FALLBACK_HINT_RECOMMENDED` (2) | Plugin has no data; core SHOULD try other plugins |
| `FALLBACK_HINT_REQUIRED` (3) | Plugin cannot handle request; core MUST try fallback |

**Important**: For actual errors (API failures, network timeouts), return a gRPC error instead
of using a fallback hint. Hints are for "no data" scenarios, not system failures.

### Validation Helpers

Validate responses before returning:

```go
// Validate actual cost response
resp := pluginsdk.NewActualCostResponse(
    pluginsdk.WithResults(results),
    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
)
if err := pluginsdk.ValidateActualCostResponse(resp); err != nil {
    return nil, status.Errorf(codes.Internal, "invalid response: %v", err)
}

// Validate recommendations
if err := pluginsdk.ValidateRecommendation(rec); err != nil {
    return nil, status.Errorf(codes.Internal, "invalid recommendation: %v", err)
}
```

Available validation functions:

- `ValidateActualCostResponse(resp)` - Validates results are non-nil with non-negative costs
- `ValidateRecommendation(rec)` - Validates recommendation has all required fields
- `ValidateResourceRecommendationInfo(res)` - Validates resource info fields
- `ValidateRecommendationImpact(impact)` - Validates impact with ISO 4217 currency

## Caching Hint Helpers (expires_at)

The SDK provides helper functions for checking whether cost data has expired,
enabling callers to manage local caches based on plugin-provided `expires_at` hints.

### Checking Expiration

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Check if a cost result has expired
for _, result := range resp.Results {
    if pluginsdk.IsActualCostExpired(result, time.Now()) {
        // Data is stale, re-fetch from plugin
        continue
    }
    // Data is fresh, use cached version
    processResult(result)
}

// Check projected cost expiration
if pluginsdk.IsProjectedCostExpired(projResp, time.Now()) {
    // Pricing data is stale, re-fetch
}
```

### Extracting Expiration Time

```go
// Get TTL for cache management
if expiresAt, ok := pluginsdk.ActualCostExpiresAt(result); ok {
    ttl := time.Until(expiresAt)
    if ttl > 0 {
        cache.SetWithTTL(cacheKey, result, ttl)
    }
    // Past expires_at means data is already stale; skip caching
}

// Get projected cost expiration
if expiresAt, ok := pluginsdk.ProjectedCostExpiresAt(projResp); ok {
    log.Info().Time("expires_at", expiresAt).Msg("pricing valid until")
}
```

### Setting expires_at (Plugin Side)

For actual cost results, set `ExpiresAt` directly or use functional options:

```go
import "google.golang.org/protobuf/types/known/timestamppb"

// Option 1: Direct field assignment
for _, r := range results {
    r.ExpiresAt = timestamppb.New(time.Now().Add(6 * time.Hour))
}

// Option 2: Functional options
pluginsdk.ApplyActualCostResultOptions(result,
    pluginsdk.WithActualCostResultExpiresAt(time.Now().Add(6 * time.Hour)),
)
```

For projected cost responses, use the `WithProjectedCostExpiresAt` option:

```go
resp := pluginsdk.NewGetProjectedCostResponse(
    pluginsdk.WithProjectedCostExpiresAt(endOfBillingCycle()),
)
```

### Behavior Summary

| Input | `IsExpired` returns | `ExpiresAt` returns |
|-------|---------------------|---------------------|
| nil result/response | `false` | `zero, false` |
| nil expires_at | `false` | `zero, false` |
| Future timestamp | `false` | `time, true` |
| Past timestamp | `true` | `time, true` |

## Pagination Helpers

The SDK provides pagination helpers for both `GetRecommendations` and `GetActualCost` RPCs.

### Plugin-Side: PaginateActualCosts

Use `PaginateActualCosts` in your plugin's `GetActualCost` handler to paginate results:

```go
func (p *MyPlugin) GetActualCost(
    ctx context.Context,
    req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
    allResults, err := p.fetchCostData(ctx, req)
    if err != nil {
        return nil, err
    }
    page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(
        allResults, req.PageSize, req.PageToken,
    )
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "%v", err)
    }
    return pluginsdk.NewActualCostResponse(
        pluginsdk.WithResults(page),
        pluginsdk.WithNextPageToken(nextToken),
        pluginsdk.WithTotalCount(totalCount),
    ), nil
}
```

Constants:

- `DefaultPageSize = 50` - Used when `page_size` is 0 with a non-empty `page_token`
  (i.e., when pagination has been explicitly started)
- `MaxPageSize = 1000` - Maximum allowed page size (larger values are clamped)
- When both `page_size <= 0` AND `page_token == ""` (proto3 defaults), the helper
  returns all results for backward compatibility with legacy/unpaginated callers

### Host-Side: ActualCostIterator

Use `ActualCostIterator` to consume paginated responses without managing tokens manually:

```go
iter := pluginsdk.NewActualCostIterator(ctx,
    func(ctx context.Context, pageToken string, pageSize int32) (*pbc.GetActualCostResponse, error) {
        return client.GetActualCost(ctx, &pbc.GetActualCostRequest{
            ResourceId: "i-abc123",
            PageSize:   pageSize,
            PageToken:  pageToken,
        })
    },
    100, // page size
)

for iter.Next() {
    record := iter.Record()
    // process record
}
if err := iter.Err(); err != nil {
    // handle error
}
fmt.Printf("Total records: %d\n", iter.TotalCount())
```

### Response Options

- `WithNextPageToken(token)` - Sets the continuation token on the response
- `WithTotalCount(count)` - Sets the total record count on the response

## FOCUS 1.2 Cost Records

The SDK includes a comprehensive `FocusRecordBuilder` for constructing FinOps FOCUS 1.2 compliant cost records.
FOCUS (FinOps Open Cost and Usage Specification) is a standard for cloud billing data.

### Quick Start

```go
import (
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

builder := pluginsdk.NewFocusRecordBuilder()

// Set mandatory fields
builder.WithIdentity("AWS", "123456789012", "Production Account")
builder.WithBillingPeriod(billingStart, billingEnd, "USD")
builder.WithChargePeriod(chargeStart, chargeEnd)
builder.WithChargeDetails(
    pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
    pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
)
builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")
builder.WithFinancials(73.0, 80.0, 70.0, "USD", "INV-2025-11")
builder.WithContractedCost(65.0)  // New in FOCUS 1.2

record, err := builder.Build()
```

### FOCUS 1.2 Column Coverage

The `FocusCostRecord` proto message implements all 57 columns defined in FOCUS 1.2:

| Category             | Mandatory | Recommended | Conditional |
| -------------------- | --------- | ----------- | ----------- |
| Identity & Hierarchy | 3         | 0           | 4           |
| Billing Period       | 3         | 0           | 0           |
| Charge Period        | 2         | 0           | 0           |
| Charge Details       | 3         | 1           | 0           |
| Pricing              | 0         | 0           | 10          |
| Service & Product    | 1         | 0           | 3           |
| Resource Details     | 0         | 0           | 3           |
| SKU Details          | 0         | 0           | 4           |
| Location             | 0         | 0           | 3           |
| Financial Amounts    | 2         | 0           | 3           |
| Consumption/Usage    | 0         | 0           | 2           |
| Commitment Discounts | 0         | 0           | 7           |
| Capacity Reservation | 0         | 0           | 2           |
| Invoice              | 0         | 0           | 2           |
| Metadata             | 0         | 0           | 1           |
| **Total**            | **14**    | **1**       | **42**      |

### Builder Methods by Category

**Mandatory Fields (14 columns)**:

| Method                                                         | Fields Set                                            | FOCUS Section |
| -------------------------------------------------------------- | ----------------------------------------------------- | ------------- |
| `WithIdentity(provider, accountID, accountName)`               | ProviderName, BillingAccountId, BillingAccountName    | 2.1           |
| `WithBillingPeriod(start, end, currency)`                      | BillingPeriodStart, BillingPeriodEnd, BillingCurrency | 2.2           |
| `WithChargePeriod(start, end)`                                 | ChargePeriodStart, ChargePeriodEnd                    | 2.3           |
| `WithChargeDetails(chargeCat, pricingCat)`                     | ChargeCategory, PricingCategory                       | 2.4           |
| `WithChargeClassification(class, desc, freq)`                  | ChargeClass, ChargeDescription                        | 2.4           |
| `WithService(category, name)`                                  | ServiceCategory, ServiceName                          | 2.6           |
| `WithFinancials(billed, list, effective, currency, invoiceID)` | BilledCost, ListCost, EffectiveCost                   | 2.10          |
| `WithContractedCost(cost)`                                     | ContractedCost                                        | 3.20          |

**Conditional Fields - New in FOCUS 1.2**:

| Method                                                   | Fields Set                                       | FOCUS Section |
| -------------------------------------------------------- | ------------------------------------------------ | ------------- |
| `WithBillingAccountType(type)`                           | BillingAccountType                               | 3.3           |
| `WithSubAccountType(type)`                               | SubAccountType                                   | 3.45          |
| `WithCapacityReservation(id, status)`                    | CapacityReservationId, CapacityReservationStatus | 3.6, 3.7      |
| `WithCommitmentDiscountDetails(qty, status, type, unit)` | CommitmentDiscountQuantity, Status, Type, Unit   | 3.14-3.19     |
| `WithContractedUnitPrice(price)`                         | ContractedUnitPrice                              | 3.21          |
| `WithPricingCurrency(currency)`                          | PricingCurrency                                  | 3.34          |
| `WithPricingCurrencyPrices(contracted, effective, list)` | PricingCurrency\*Prices                          | 3.35-3.37     |
| `WithPublisher(publisher)`                               | Publisher                                        | 3.39          |
| `WithServiceSubcategory(subcategory)`                    | ServiceSubcategory                               | 3.43          |
| `WithSkuDetails(meter, priceDetails)`                    | SkuMeter, SkuPriceDetails                        | 3.46, 3.48    |

**Other Conditional Fields**:

| Method                                       | Fields Set                                  | FOCUS Section |
| -------------------------------------------- | ------------------------------------------- | ------------- |
| `WithSubAccount(id, name)`                   | SubAccountId, SubAccountName                | 2.1           |
| `WithLocation(regionID, regionName, az)`     | RegionId, RegionName, AvailabilityZone      | 2.9           |
| `WithResource(id, name, type)`               | ResourceId, ResourceName, ResourceType      | 2.7           |
| `WithSKU(skuID, skuPriceID)`                 | SkuId, SkuPriceId                           | 2.8           |
| `WithPricing(quantity, unit, listUnitPrice)` | PricingQuantity, PricingUnit, ListUnitPrice | 2.5           |
| `WithUsage(quantity, unit)`                  | ConsumedQuantity, ConsumedUnit              | 2.11          |
| `WithCommitmentDiscount(category, id, name)` | CommitmentDiscount\*                        | 2.12          |
| `WithInvoice(id, issuer)`                    | InvoiceId, InvoiceIssuer                    | 2.13          |
| `WithTag(key, value)` / `WithTags(map)`      | Tags                                        | 2.14          |
| `WithExtension(key, value)`                  | ExtendedColumns                             | 2.14          |

### Validation

The builder validates mandatory fields and business rules on `Build()`:

```go
record, err := builder.Build()
if err != nil {
    // Handle validation error
    // e.g., "billing_account_id is required"
}
```

**Validated Rules**:

- BillingAccountId is required
- ChargePeriod (start/end) is required
- ServiceCategory must be specified (not UNSPECIFIED)
- ChargeCategory must be specified (not UNSPECIFIED)
- BillingCurrency is required
- Usage records must have non-zero ConsumedQuantity

### Migration Guide

If you're updating from an earlier version without FOCUS 1.2 support:

**New Mandatory Field**: `ContractedCost` is now mandatory in FOCUS 1.2. Add it to all records:

```go
builder.WithContractedCost(65.0)
```

**New Builder Methods**: 11 new builder methods for FOCUS 1.2 columns:

- `WithContractedCost`
- `WithBillingAccountType`
- `WithSubAccountType`
- `WithCapacityReservation`
- `WithCommitmentDiscountDetails`
- `WithContractedUnitPrice`
- `WithPricingCurrency`
- `WithPricingCurrencyPrices`
- `WithPublisher`
- `WithServiceSubcategory`
- `WithSkuDetails`

### Troubleshooting

**Common Validation Errors**:

| Error                                          | Cause                                     | Solution                                         |
| ---------------------------------------------- | ----------------------------------------- | ------------------------------------------------ |
| `billing_account_id is required`               | Missing identity                          | Call `WithIdentity()`                            |
| `charge_period (start/end) is required`        | Missing charge period                     | Call `WithChargePeriod()`                        |
| `service_category is required`                 | Unspecified category                      | Call `WithService()` or `WithServiceCategory()`  |
| `charge_category is required`                  | Unspecified category                      | Call `WithChargeDetails()`                       |
| `billing_currency is required`                 | Missing currency                          | Call `WithBillingPeriod()` or `WithFinancials()` |
| `consumed_quantity must be positive for usage` | Zero or negative usage for Usage category | Set positive usage via `WithUsage()`             |

### Complete Example

See `examples/plugins/focus_example.go` for a comprehensive example demonstrating all 57 FOCUS 1.2 columns.

## FOCUS 1.3 Extensions

The SDK extends FOCUS 1.2 support with new FOCUS 1.3 columns for split cost allocation,
provider identification, and contract commitment tracking.

### New FOCUS 1.3 Columns

| Column                 | Builder Method                      | Purpose                                     |
| ---------------------- | ----------------------------------- | ------------------------------------------- |
| AllocatedMethodId      | `WithAllocation(methodId, details)` | Allocation methodology identifier           |
| AllocatedMethodDetails | `WithAllocation(methodId, details)` | Human-readable allocation description       |
| AllocatedResourceId    | `WithAllocatedResource(id, name)`   | Target resource receiving allocated cost    |
| AllocatedResourceName  | `WithAllocatedResource(id, name)`   | Display name of allocated resource          |
| AllocatedTags          | `WithAllocatedTags(tags)`           | Tags for the allocated resource             |
| ServiceProviderName    | `WithServiceProvider(name)`         | Entity providing the service (ISV/reseller) |
| HostProviderName       | `WithHostProvider(name)`            | Entity hosting the resource (cloud vendor)  |
| ContractApplied        | `WithContractApplied(commitmentId)` | Link to ContractCommitment record           |

### Deprecated Fields (FOCUS 1.3)

| Deprecated Field | Replacement             | Migration                                                            |
| ---------------- | ----------------------- | -------------------------------------------------------------------- |
| `provider_name`  | `service_provider_name` | Use `WithServiceProvider()` instead of `WithIdentity()` for provider |
| `publisher`      | `host_provider_name`    | Use `WithHostProvider()` instead of `WithPublisher()`                |

When both deprecated and replacement fields are set, a warning is logged and the FOCUS 1.3
field takes precedence.

### FOCUS 1.3 Usage Examples

**Split Cost Allocation**:

```go
// Allocate shared infrastructure costs to workloads
builder := pluginsdk.NewFocusRecordBuilder()
builder.
    WithIdentity("AWS", "123456789012", "Shared Services").
    WithAllocation("proportional-cpu", "Costs split by CPU utilization percentage").
    WithAllocatedResource("workload-123", "Frontend Application").
    WithAllocatedTags(map[string]string{
        "team":        "frontend",
        "environment": "production",
    })

record, err := builder.Build()
```

**Marketplace Provider Identification**:

```go
// Distinguish ISV from hosting provider in marketplace scenarios
builder := pluginsdk.NewFocusRecordBuilder()
builder.
    WithServiceProvider("Datadog").      // ISV selling via marketplace
    WithHostProvider("AWS").             // Cloud platform hosting the service
    WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_OBSERVABILITY, "Datadog APM")

record, err := builder.Build()
```

**Contract Commitment Link**:

```go
// Link cost record to a contract commitment
builder := pluginsdk.NewFocusRecordBuilder()
builder.
    WithContractApplied("commit-2025-ri-001")  // References ContractCommitment.ContractCommitmentId

record, err := builder.Build()
```

### Validation Rules

- **Allocation**: If `AllocatedMethodId` is set, `AllocatedResourceId` must also be set
- **Deprecation**: Warnings logged when deprecated + replacement fields both present
- **ContractApplied**: Treated as opaque reference (no cross-dataset validation)

## Contract Commitment Dataset (FOCUS 1.3)

FOCUS 1.3 introduces a supplemental dataset for tracking contractual obligations separately
from cost line items. The `ContractCommitmentBuilder` creates these records.

### Quick Start

```go
import (
    "time"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

builder := pluginsdk.NewContractCommitmentBuilder()

commitment, err := builder.
    WithIdentity("commit-2025-ri-001", "contract-ea-2025").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
    WithType("Reserved Instance").
    WithCommitmentPeriod(
        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
    ).
    WithContractPeriod(
        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2027, 12, 31, 23, 59, 59, 0, time.UTC),
    ).
    WithFinancials(10000.00, 0, "", "USD").
    Build()
```

### ContractCommitment Fields

| Field                         | Builder Method                     | Required | Description                               |
| ----------------------------- | ---------------------------------- | -------- | ----------------------------------------- |
| ContractCommitmentId          | `WithIdentity(commitmentId, ...)`  | Yes      | Unique commitment identifier              |
| ContractId                    | `WithIdentity(..., contractId)`    | Yes      | Parent contract identifier                |
| ContractCommitmentCategory    | `WithCategory(category)`           | No       | SPEND or USAGE commitment type            |
| ContractCommitmentType        | `WithType(type)`                   | No       | Provider-specific type (e.g., "RI", "SP") |
| ContractCommitmentPeriodStart | `WithCommitmentPeriod(start, end)` | No       | Commitment period start                   |
| ContractCommitmentPeriodEnd   | `WithCommitmentPeriod(start, end)` | No       | Commitment period end                     |
| ContractPeriodStart           | `WithContractPeriod(start, end)`   | No       | Overall contract period start             |
| ContractPeriodEnd             | `WithContractPeriod(start, end)`   | No       | Overall contract period end               |
| ContractCommitmentCost        | `WithFinancials(cost, ...)`        | No       | Monetary commitment amount (SPEND)        |
| ContractCommitmentQuantity    | `WithFinancials(..., qty, ...)`    | No       | Quantity commitment (USAGE)               |
| ContractCommitmentUnit        | `WithFinancials(..., unit, ...)`   | No       | Unit of measure for quantity              |
| BillingCurrency               | `WithFinancials(..., currency)`    | Yes      | ISO 4217 currency code                    |

### Commitment Categories

| Category | Enum Value                                 | Use Case                                    |
| -------- | ------------------------------------------ | ------------------------------------------- |
| SPEND    | `FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND` | Dollar-based commitments (e.g., $10K/month) |
| USAGE    | `FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE` | Usage-based commitments (e.g., 1000 hours)  |

### Example: SPEND Commitment (Reserved Instance)

```go
commitment, err := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("ri-ec2-2025-001", "aws-ea-2025").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
    WithType("Reserved Instance").
    WithCommitmentPeriod(commitStart, commitEnd).
    WithFinancials(50000.00, 0, "", "USD").  // $50,000 commitment
    Build()
```

### Example: USAGE Commitment (Committed Use Discount)

```go
commitment, err := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("cud-gce-2025-001", "gcp-cud-2025").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
    WithType("Committed Use Discount").
    WithCommitmentPeriod(commitStart, commitEnd).
    WithFinancials(0, 1000, "vCPU-Hours", "USD").  // 1000 vCPU-Hours commitment
    Build()
```

### Linking Cost Records to Commitments

Use `WithContractApplied()` on `FocusRecordBuilder` to link cost records to commitments:

```go
// Create the commitment
commitment, _ := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("commit-001", "contract-2025").
    WithFinancials(10000.00, 0, "", "USD").
    Build()

// Link cost records to the commitment
costRecord, _ := pluginsdk.NewFocusRecordBuilder().
    WithContractApplied(commitment.ContractCommitmentId).  // "commit-001"
    // ... other fields
    Build()
```

### Validation Rules

| Rule                           | Description                                              |
| ------------------------------ | -------------------------------------------------------- |
| Required: ContractCommitmentId | Must be non-empty                                        |
| Required: ContractId           | Must be non-empty                                        |
| Required: BillingCurrency      | Must be valid ISO 4217 code (validated via currency pkg) |
| Period Consistency             | commitment_period_end >= commitment_period_start         |
| Period Consistency             | contract_period_end >= contract_period_start             |
| Non-negative Values            | cost >= 0, quantity >= 0                                 |

### Error Handling

```go
commitment, err := builder.Build()
if err != nil {
    switch {
    case strings.Contains(err.Error(), "contract_commitment_id is required"):
        // Missing commitment ID
    case strings.Contains(err.Error(), "billing_currency"):
        // Invalid or missing currency
    case strings.Contains(err.Error(), "period_end must be >="):
        // Invalid period range
    }
}
```

## Manifest Management

Load and save plugin manifests:

```go
// Load from YAML or JSON
manifest, err := pluginsdk.LoadManifest("plugin-manifest.yaml")

// Save to file (format determined by extension)
err := pluginsdk.SaveManifest("plugin-manifest.json", manifest)
```

## Property Mapping (mapping subpackage)

The `mapping` subpackage provides helper functions for extracting SKU, region, and other
pricing-relevant fields from Pulumi resource properties. This package enables plugin developers
to decouple cloud-specific property extraction from their plugin logic.

### Installation

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"
```

### Provider-Specific Extraction

Each cloud provider has dedicated extraction functions:

**AWS**:

```go
// EC2, RDS, EBS property extraction
props := map[string]string{
    "instanceType":     "t3.medium",
    "availabilityZone": "us-east-1a",
}
sku := mapping.ExtractAWSSKU(props)       // "t3.medium"
region := mapping.ExtractAWSRegion(props) // "us-east-1"

// Direct AZ to region conversion
region := mapping.ExtractAWSRegionFromAZ("us-west-2b") // "us-west-2"
```

**Azure**:

```go
// VM and service property extraction
props := map[string]string{
    "vmSize":   "Standard_D2s_v3",
    "location": "eastus",
}
sku := mapping.ExtractAzureSKU(props)       // "Standard_D2s_v3"
region := mapping.ExtractAzureRegion(props) // "eastus"
```

**GCP**:

```go
// Compute Engine property extraction with region validation
props := map[string]string{
    "machineType": "n1-standard-4",
    "zone":        "us-central1-a",
}
sku := mapping.ExtractGCPSKU(props)       // "n1-standard-4"
region := mapping.ExtractGCPRegion(props) // "us-central1"

// Direct zone to region conversion with validation
region := mapping.ExtractGCPRegionFromZone("europe-west1-b") // "europe-west1"

// Check if a region is valid
if mapping.IsValidGCPRegion("us-central1") {
    // Valid GCP region
}
```

### Generic Extraction

For custom resources or FinOps tools, use generic extractors with custom key lists:

```go
// With default keys (sku, type, tier for SKU; region, location, zone for region)
props := map[string]string{"sku": "custom-sku", "region": "custom-region"}
sku := mapping.ExtractSKU(props)       // "custom-sku"
region := mapping.ExtractRegion(props) // "custom-region"

// With custom keys (checked in order)
props := map[string]string{"customField": "value"}
sku := mapping.ExtractSKU(props, "customField", "fallbackField")
```

### Plugin Integration Example

```go
// convertProperties is a plugin-specific helper that converts Pulumi resource
// properties to map[string]string. Implementation varies by plugin - typically
// iterates resource.Properties and extracts string values.
func convertProperties(props map[string]interface{}) map[string]string {
    result := make(map[string]string)
    for k, v := range props {
        if s, ok := v.(string); ok {
            result[k] = s
        }
    }
    return result
}

func (p *MyPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (
    *pbc.GetProjectedCostResponse, error,
) {
    for _, resource := range req.Resources {
        props := convertProperties(resource.Properties)

        var sku, region string
        switch resource.Provider {
        case "aws":
            sku = mapping.ExtractAWSSKU(props)
            region = mapping.ExtractAWSRegion(props)
        case "azure":
            sku = mapping.ExtractAzureSKU(props)
            region = mapping.ExtractAzureRegion(props)
        case "gcp":
            sku = mapping.ExtractGCPSKU(props)
            region = mapping.ExtractGCPRegion(props)
        default:
            sku = mapping.ExtractSKU(props)
            region = mapping.ExtractRegion(props)
        }

        // Use sku and region for pricing lookup...
    }
    // ...
}
```

### Error Handling

All mapping functions are designed to never panic:

- Returns empty string for `nil` or empty input maps
- Returns empty string when no matching keys are found
- GCP functions validate against known regions list

```go
mapping.ExtractAWSSKU(nil)                  // ""
mapping.ExtractGCPRegionFromZone("invalid") // "" (invalid region)
```

### Migration from finfocus-core

If you have cloud-specific extraction logic in your plugin or core adapter,
you can replace it with the mapping package functions:

```go
// Before: inline extraction logic
var sku string
if v, ok := props["instanceType"]; ok {
    sku = v
} else if v, ok := props["instanceClass"]; ok {
    sku = v
}

// After: use mapping package
sku := mapping.ExtractAWSSKU(props)
```

## Dry Run Mode

The DryRun feature allows hosts to query a plugin for its FOCUS field mapping capabilities
without triggering actual cost data retrieval. This is useful for:

- **Debugging**: Understand which fields a plugin populates for a resource type
- **Validation**: Verify plugin configuration before production deployment
- **Comparison**: Compare capabilities across different plugins

### Quick Start

```go
import (
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Create field mappings for a supported resource type
mappings := pluginsdk.AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)

// Customize specific field statuses
mappings = pluginsdk.SetFieldStatus(mappings, "availability_zone",
    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
    pluginsdk.WithConditionDescription("Only for regional resources"))

// Build the response
resp := pluginsdk.NewDryRunResponse(
    pluginsdk.WithFieldMappings(mappings),
    pluginsdk.WithResourceTypeSupported(true),
    pluginsdk.WithConfigurationValid(true),
)
```

### DryRun RPC

The `DryRun` RPC allows hosts to query a plugin's field mapping capabilities:

```go
// Implement the DryRun RPC on your plugin
func (p *MyPlugin) DryRun(
    ctx context.Context,
    req *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
    resource := req.GetResource()

    // Check if resource type is supported
    if !p.supports(resource.GetType()) {
        return pluginsdk.NewDryRunResponse(
            pluginsdk.WithResourceTypeSupported(false),
        ), nil
    }

    // Return field mappings for supported resource
    mappings := pluginsdk.AllFieldsWithStatus(
        pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
    )

    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithFieldMappings(mappings),
        pluginsdk.WithResourceTypeSupported(true),
        pluginsdk.WithConfigurationValid(true),
    ), nil
}
```

### FieldSupportStatus Enum

Each FOCUS field can have one of the following support statuses:

| Status        | Value | Description                                                               |
| ------------- | ----- | ------------------------------------------------------------------------- |
| `UNSPECIFIED` | 0     | Status not specified (default)                                            |
| `SUPPORTED`   | 1     | Field is always populated by the plugin                                   |
| `UNSUPPORTED` | 2     | Field is never populated by the plugin                                    |
| `CONDITIONAL` | 3     | Field is populated under certain conditions (use `condition_description`) |
| `DYNAMIC`     | 4     | Field population depends on runtime data or simulation parameters         |

### Field Mapping Helpers

**FocusFieldNames()** - Returns all ~66 FOCUS 1.2/1.3 field names:

```go
fields := pluginsdk.FocusFieldNames()
// Returns: ["provider_name", "billing_account_id", "billing_account_name", ...]
```

**NewFieldMapping()** - Creates a single field mapping:

```go
fm := pluginsdk.NewFieldMapping(
    "availability_zone",
    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
    pluginsdk.WithConditionDescription("Only for multi-AZ resources"),
    pluginsdk.WithExpectedType("string"),
)
```

**AllFieldsWithStatus()** - Creates mappings for all FOCUS fields:

```go
// All fields supported
mappings := pluginsdk.AllFieldsWithStatus(
    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
)
```

**SetFieldStatus()** - Updates a specific field's status:

```go
mappings = pluginsdk.SetFieldStatus(
    mappings,
    "commitment_discount_id",
    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_UNSUPPORTED,
)
```

### DryRunResponse Builder Options

| Option                              | Description                                 |
| ----------------------------------- | ------------------------------------------- |
| `WithFieldMappings(mappings)`       | Sets the field_mappings array               |
| `WithResourceTypeSupported(bool)`   | Sets whether the resource type is supported |
| `WithConfigurationValid(bool)`      | Sets whether plugin configuration is valid  |
| `WithConfigurationErrors([]string)` | Sets configuration error messages           |

### dry_run Flag on Cost RPCs

The `dry_run` flag can also be set on `GetActualCostRequest` and `GetProjectedCostRequest`
to get field mapping information embedded in the response:

```go
// Request with dry_run=true
req := &pbc.GetActualCostRequest{
    ResourceId: "i-1234567890abcdef0",
    Start:      timestamppb.New(start),
    End:        timestamppb.New(end),
    DryRun:     true, // Enable dry-run mode
}

resp, err := client.GetActualCost(ctx, req)
if err != nil {
    return err
}

// When dry_run=true, dry_run_result is populated instead of actual costs
dryRunResult := resp.GetDryRunResult()
if dryRunResult != nil {
    for _, mapping := range dryRunResult.GetFieldMappings() {
        fmt.Printf("Field: %s, Status: %v\n",
            mapping.GetFieldName(),
            mapping.GetSupportStatus(),
        )
    }
}
```

### Implementing Configuration Validation

Plugins can optionally implement the `ConfigValidator` interface:

```go
type ConfigValidator interface {
    ValidateConfiguration() []string
}

// Example implementation
func (p *MyPlugin) ValidateConfiguration() []string {
    var errors []string

    if p.apiKey == "" {
        errors = append(errors, "API key is required")
    }

    if p.region == "" {
        errors = append(errors, "Region must be specified")
    }

    return errors
}

// In DryRun handler, call validation
func (p *MyPlugin) DryRun(ctx context.Context, req *pbc.DryRunRequest) (
    *pbc.DryRunResponse, error,
) {
    configErrors := p.ValidateConfiguration()

    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithFieldMappings(p.getFieldMappings(req.GetResource())),
        pluginsdk.WithResourceTypeSupported(true),
        pluginsdk.WithConfigurationValid(len(configErrors) == 0),
        pluginsdk.WithConfigurationErrors(configErrors),
    ), nil
}
```

### Simulation Parameters

The DryRunRequest includes a `simulation_parameters` map for parameterized dry-runs:

```go
// Request with simulation parameters
req := &pbc.DryRunRequest{
    Resource: resource,
    SimulationParameters: map[string]string{
        "pricing_tier": "enterprise",
        "region":       "us-east-1",
    },
}

// In handler, check simulation parameters
func (p *MyPlugin) DryRun(ctx context.Context, req *pbc.DryRunRequest) (
    *pbc.DryRunResponse, error,
) {
    params := req.GetSimulationParameters()

    mappings := pluginsdk.AllFieldsWithStatus(
        pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
    )

    // Adjust field status based on simulation parameters
    if params["pricing_tier"] == "basic" {
        mappings = pluginsdk.SetFieldStatus(
            mappings,
            "commitment_discount_id",
            pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_DYNAMIC,
            pluginsdk.WithConditionDescription("Only available on enterprise tier"),
        )
    }

    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithFieldMappings(mappings),
        pluginsdk.WithResourceTypeSupported(true),
        pluginsdk.WithConfigurationValid(true),
    ), nil
}
```

### Profile-Aware DryRun Responses (FR-007)

Plugins SHOULD document profile-specific behavior through DryRun simulation parameters.
Use the reserved `"usage_profile"` key in `simulation_parameters` to indicate how field
support varies by workload context:

```go
// Request DryRun with profile context
req := &pbc.DryRunRequest{
    Resource: resource,
    SimulationParameters: map[string]string{
        "usage_profile": "prod",
    },
}

// In handler, adjust field mappings based on profile
func (p *MyPlugin) DryRun(ctx context.Context, req *pbc.DryRunRequest) (
    *pbc.DryRunResponse, error,
) {
    params := req.GetSimulationParameters()
    profile, _ := pluginsdk.ParseUsageProfile(params["usage_profile"])

    mappings := pluginsdk.AllFieldsWithStatus(
        pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
    )

    // Profile-dependent field support
    if profile == pbc.UsageProfile_USAGE_PROFILE_DEV {
        mappings = pluginsdk.SetFieldStatus(
            mappings,
            "commitment_discount_id",
            pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_UNSUPPORTED,
            pluginsdk.WithConditionDescription("Commitments not applicable for DEV"),
        )
    }

    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithFieldMappings(mappings),
        pluginsdk.WithResourceTypeSupported(true),
        pluginsdk.WithConfigurationValid(true),
    ), nil
}
```

### Performance Requirements

DryRun operations should be fast and not make external API calls:

| Metric             | Requirement   |
| ------------------ | ------------- |
| Response time      | < 100ms (p99) |
| External API calls | None          |
| Memory allocation  | Minimal       |

### Capability Discovery

Advertise DryRun support via the `Supports` RPC:

```go
func (p *MyPlugin) Supports(
    ctx context.Context,
    req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
    return &pbc.SupportsResponse{
        IsSupported: true,
        Capabilities: map[string]bool{
            "dry_run": true, // Advertise DryRun support
        },
    }, nil
}
```

## Thread Safety

This section documents thread safety guarantees for SDK components.

### Component Thread Safety Summary

| Component              | Thread-Safe | Notes                                                  |
| ---------------------- | ----------- | ------------------------------------------------------ |
| **Client**             | ✅ YES      | Safe for concurrent RPC calls from multiple goroutines |
| **Server**             | ✅ YES      | Assumes Plugin implementation is thread-safe           |
| **WebConfig**          | ✅ YES      | Read-only after construction (value semantics)         |
| **PluginMetrics**      | ✅ YES      | Uses Prometheus internal atomics                       |
| **ResourceMatcher**    | ❌ NO       | Configure before Serve(), then read-only               |
| **FocusRecordBuilder** | ❌ NO       | Single-threaded builder pattern                        |

### Client Thread Safety

The `Client` struct wraps `http.Client` which is explicitly designed for concurrent use.
All methods are stateless request/response operations.

```go
// Create once, use from multiple goroutines
client := pluginsdk.NewConnectClient("http://localhost:8080")
defer client.Close()

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        name, err := client.Name(ctx) // Safe concurrent call
        // ...
    }()
}
wg.Wait()
```

### Server Thread Safety

The `Server` struct is thread-safe for concurrent RPC handling. However, it
delegates all business logic to the `Plugin` interface implementation.

**Plugin Implementation Requirements**:

- Plugin methods (`Name`, `Supports`, `GetActualCost`, etc.) MUST be thread-safe
- The gRPC framework calls plugin methods from multiple goroutines concurrently
- Plugin state (if any) must use appropriate synchronization (mutexes, atomic operations, etc.)

**Best Practice**: Design plugins to be stateless where possible. For stateful plugins,
use `sync.RWMutex` for cache-like state or `sync.Mutex` for mutable configuration.

### ResourceMatcher Thread Safety

**NOT thread-safe** - Must be configured before `Serve()` is called:

```go
// CORRECT: Configure during initialization
matcher := pluginsdk.NewResourceMatcher()
matcher.AddProvider("aws")
matcher.AddResourceType("aws:ec2/instance:Instance")
// ... complete all configuration ...

// After Serve() is called, matcher becomes effectively read-only
pluginsdk.Serve(ctx, config)
```

### FocusRecordBuilder Thread Safety

**NOT thread-safe** - Each goroutine should create its own builder:

```go
// CORRECT: One builder per goroutine
go func() {
    builder := pluginsdk.NewFocusRecordBuilder()
    builder.WithIdentity("AWS", "123456789012", "Production")
    // ...
    record, err := builder.Build()
}()
```

## Rate Limiting

When calling cloud provider APIs, implement rate limiting to avoid throttling.

### Token Bucket Pattern

Use `golang.org/x/time/rate` for efficient rate limiting:

```go
import "golang.org/x/time/rate"

// Create rate limiter: 100 requests per second, burst of 200
limiter := rate.NewLimiter(rate.Limit(100), 200)

func (p *MyPlugin) GetActualCost(
    ctx context.Context,
    req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
    // Check rate limit
    if !limiter.Allow() {
        return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
    }
    // Proceed with request...
}
```

### Cloud Provider Rate Limits (Reference - Verify Current Values)

**Note**: These limits were accurate as of December 2024. Always verify current
limits in provider documentation as they change frequently.

| Provider   | Service         | Default Limit (2024) | Suggested Local Limit    | Source                                                                                                           |
| ---------- | --------------- | -------------------- | ------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| AWS        | Cost Explorer   | 5 req/sec            | 3 req/sec (60% headroom) | [AWS Docs](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/ce-api.html#ce-api-limits)               |
| Azure      | Cost Management | 100 req/5min         | 15 req/min               | [Azure Docs](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/scalability-limits)           |
| GCP        | Billing API     | 1000 req/min         | 800 req/min              | [GCP Docs](https://cloud.google.com/billing/docs/reference/rest/v1/billingAccounts/get#authorization-and-quotas) |
| Kubernetes | Metrics API     | Varies               | 50 req/sec               | [K8s Docs](https://kubernetes.io/docs/reference/using-api/api-concepts/#rate-limiting)                           |

### Backoff Strategies

Implement exponential backoff with jitter for retry logic:

```go
func backoff(attempt int) time.Duration {
    base := 100 * time.Millisecond
    max := 30 * time.Second

    // Exponential: 100ms, 200ms, 400ms, 800ms, ...
    delay := base * time.Duration(1<<attempt)
    if delay > max {
        delay = max
    }

    // Add jitter (±25%)
    jitter := time.Duration((rand.Float64() - 0.5) * 0.5 * float64(delay))
    return delay + jitter
}
```

### gRPC Status Codes for Rate Limiting

| Situation           | Status Code         | Description                     |
| ------------------- | ------------------- | ------------------------------- |
| Local rate limit    | `ResourceExhausted` | Plugin's internal limit reached |
| Upstream throttling | `Unavailable`       | Backend API returned 429        |
| Retry recommended   | `Unavailable`       | Include Retry-After header      |

## Performance Tuning

### Connection Pool Configuration

For high-throughput scenarios, use `HighThroughputClientConfig`:

```go
cfg := pluginsdk.HighThroughputClientConfig("http://localhost:8080")
client := pluginsdk.NewClient(cfg)
defer client.Close()
```

Default connection pool settings:

| Setting               | Default Value | Description                           |
| --------------------- | ------------- | ------------------------------------- |
| `MaxIdleConns`        | 100           | Max idle connections across all hosts |
| `MaxIdleConnsPerHost` | 10            | Max idle connections per host         |
| `IdleConnTimeout`     | 90s           | How long idle connections are kept    |
| `Timeout`             | 30s           | HTTP client timeout                   |

### Server Timeouts

Configure server timeouts for DoS protection:

```go
type ServerTimeouts struct {
    ReadHeaderTimeout time.Duration // Time to read request headers
    ReadTimeout       time.Duration // Time to read entire request
    WriteTimeout      time.Duration // Time to write response
    IdleTimeout       time.Duration // Keep-alive timeout
}
```

Recommended production values:

| Timeout           | Recommended | Rationale                             |
| ----------------- | ----------- | ------------------------------------- |
| ReadHeaderTimeout | 5s          | Prevent slow loris attacks            |
| ReadTimeout       | 30s         | Allow time for large requests         |
| WriteTimeout      | 60s         | Allow time for cost calculations      |
| IdleTimeout       | 120s        | Balance connection reuse vs resources |

### Protocol Performance Trade-offs

| Protocol     | Transport   | Performance            | Use Case                         |
| ------------ | ----------- | ---------------------- | -------------------------------- |
| **gRPC**     | HTTP/2 only | Best (binary protobuf) | Server-to-server, native clients |
| **Connect**  | HTTP/1.1+   | Good (JSON)            | Web dashboards, REST clients     |
| **gRPC-Web** | HTTP/1.1+   | Good (binary)          | Browser clients needing binary   |

### Benchmark Reference Values

Response time baselines from conformance tests.

**Note**: These are proposed performance targets. Some conformance levels currently
use shared thresholds (e.g., 100ms for simple RPCs) while these more granular
targets are being phased into the testing suite.

| Method             | Standard   | Advanced    |
| ------------------ | ---------- | ----------- |
| Name()             | < 100ms    | < 50ms      |
| Supports()         | < 50ms     | < 25ms      |
| GetProjectedCost() | < 200ms    | < 100ms     |
| GetPricingSpec()   | < 200ms    | < 100ms     |
| GetActualCost()    | < 2s (24h) | < 10s (30d) |

## CORS Configuration

When enabling web support (`Web.Enabled = true`), configure CORS appropriately for your deployment scenario.

### Deployment Scenarios

#### 1. Local Development

```go
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"http://localhost:3000", "http://127.0.0.1:3000"}).
    WithHealthEndpoint(true)
```

- Multiple localhost aliases handled
- No credentials needed for local CORS
- Health endpoint useful for testing

#### 2. Single-Origin Production

```go
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithAllowCredentials(true)
```

- Specific origin prevents unauthorized access
- HTTPS required before sending credentials
- Most restrictive and secure option

#### 3. Multi-Origin (Trusted Partners)

```go
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{
        "https://app.example.com",
        "https://dashboard.example.com",
        "https://partner.trusted.com",
    }).
    WithAllowCredentials(true)
```

- Explicit whitelist for partner integrations
- If origin list > 10, consider API gateway pattern

**Security Warning**: With `AllowCredentials(true)`, every origin in the list can
send authenticated requests. Carefully audit all origins - a compromised or malicious
origin can access user credentials. Consider these alternatives:

1. **Single-Origin per deployment**: Deploy separate plugin instances per origin
2. **Dynamic origin validation**: Implement custom middleware to validate origins
   against a database rather than static config
3. **API Gateway pattern**: Handle CORS at the gateway level (see scenario #4)

#### 4. API Gateway Pattern

When plugins run behind an API gateway:

- Plugins don't need CORS (gateway handles it)
- Gateway provides centralized authentication
- Better observability and rate limiting
- Recommended for production deployments

#### 5. Multi-Tenant SaaS

For multi-tenant deployments:

- Each tenant gets own plugin server instance
- Complete isolation between tenant origins
- Per-tenant rate limiting possible

### Custom CORS Headers

The SDK provides configurable CORS headers for security compliance and observability requirements.

#### Default Headers

When `AllowedHeaders` and `ExposedHeaders` are `nil`, sensible defaults are used:

| Header Type      | Default Value                                                                                                                                                                              |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `AllowedHeaders` | Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token, X-Requested-With, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent |
| `ExposedHeaders` | Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin, Connect-Content-Encoding, Connect-Content-Type                                                                                         |

#### Customizing Headers

Use the builder methods to customize which headers are allowed or exposed:

```go
// Minimal security footprint (omit Authorization if using different auth)
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithAllowedHeaders([]string{
        "Accept",
        "Content-Type",
        "Connect-Protocol-Version",
        "Connect-Timeout-Ms",
        "Grpc-Timeout",
        "X-Grpc-Web",
    })
```

```go
// Add observability headers for tracing
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithAllowedHeaders([]string{
        "Content-Type",
        "Authorization",
        "X-Request-ID",       // Custom tracing header
        "X-Correlation-ID",   // Custom tracing header
        "Connect-Protocol-Version",
    }).
    WithExposedHeaders([]string{
        "Grpc-Status",
        "Grpc-Message",
        "X-Request-ID",       // Expose for client correlation
        "X-Trace-ID",         // Expose for client tracing
    })
```

```go
// Strict compliance: only CORS-safelisted headers
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithAllowedHeaders([]string{}).  // Empty = simple headers only
    WithExposedHeaders([]string{})   // Empty = no custom headers exposed
```

#### Header Configuration Semantics

| Config Value         | Behavior                                                |
| -------------------- | ------------------------------------------------------- |
| `nil` (default)      | Use default headers for Connect/gRPC-Web compatibility  |
| `[]string{}` (empty) | Set empty header (only CORS-safelisted headers allowed) |
| `[]string{"A", "B"}` | Use exactly these headers, joined by ", "               |

#### Accessing Default Header Constants

The default header values are available as package constants:

```go
// Use in documentation or comparisons
fmt.Println(pluginsdk.DefaultAllowedHeaders)
fmt.Println(pluginsdk.DefaultExposedHeaders)
fmt.Println(pluginsdk.DefaultMaxAge) // 86400 seconds (24 hours)
```

### Preflight Cache (Max-Age)

The `MaxAge` setting controls how long browsers cache CORS preflight responses. This affects performance
(fewer OPTIONS requests) versus security (faster policy propagation).

```go
// Default: 24 hours (maximum for most browsers)
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"})
// MaxAge is nil, uses DefaultMaxAge (86400 seconds)

// Custom: 1 hour for faster policy updates
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"https://app.example.com"}).
    WithMaxAge(3600)

// Disable caching: useful for development or high-security environments
cfg := pluginsdk.DefaultWebConfig().
    WithWebEnabled(true).
    WithAllowedOrigins([]string{"http://localhost:3000"}).
    WithMaxAge(0)
```

| Value           | Use Case                                                           |
| --------------- | ------------------------------------------------------------------ |
| `nil` (default) | Production - uses 86400s (24 hours)                                |
| `3600`          | Balanced - policy updates within 1 hour                            |
| `0`             | Development/High-security - no caching, preflight on every request |

**Note**: Most browsers cap max-age at 24 hours regardless of the value set.

### Security Guidelines

**When to Avoid Wildcard Origin (`*`)**:

- Cannot send credentials (browser blocks it)
- No protection against CSRF-like attacks
- Only legitimate for public APIs with no sensitive data

**Credentials Handling**:

- HTTPS required (browser enforces)
- Enable credentials ONLY for trusted origins
- Prefer Authorization headers over cookies for cross-origin

### Debugging CORS Issues

1. **Browser DevTools**: Check Network tab for CORS headers
2. **curl Testing**: Simulate browser with Origin header:

   ```bash
   curl -X OPTIONS http://localhost:8080/finfocus.v1.CostSourceService/Name \
     -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: POST" \
     -v
   ```

3. **Server-side Logging**: Log origin, method, allowed status
4. **Integration Tests**: Test preflight and actual requests

## Migration from finfocus-core

If you're migrating from `finfocus-core`, update your imports:

```go
// Before
import "github.com/rshade/finfocus-core/pluginsdk"

// After
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
```

Key changes:

1. **Logging**: Now uses zerolog instead of slog
2. **Server.logger**: Changed from `*slog.Logger` to `zerolog.Logger`
3. **HoursPerMonth**: Now exported (was `hoursPerMonth`)
4. **ServeConfig.Logger**: New field for custom logger injection
5. **TracingUnaryServerInterceptor**: Now automatically integrated in `Serve()`

### Proto Package Path Change (v0.5.0+)

The proto `go_package` path has changed to support connect-go code generation:

```diff
// Before
-option go_package = "github.com/rshade/finfocus-spec/sdk/go/proto;pbc";

// After
+option go_package = "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1;pbc";
```

**Impact**: If you import proto types directly, update your imports:

```go
// Before
import pbc "github.com/rshade/finfocus-spec/sdk/go/proto"

// After
import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
```

The `pbc` alias remains the same, so no other code changes are required.

## API Reference

### Types

| Type                | Description                                  |
| ------------------- | -------------------------------------------- |
| `Plugin`            | Core interface for plugin implementations    |
| `SupportsProvider`  | Optional interface for Supports() capability |
| `BasePlugin`        | Scaffold with default implementations        |
| `ResourceMatcher`   | Resource filtering by provider/type          |
| `CostCalculator`    | Cost calculation utilities                   |
| `Server`            | gRPC server wrapper                          |
| `ServeConfig`       | Configuration for Serve()                    |
| `TestServer`        | Testing server with cleanup                  |
| `TestPlugin`        | High-level testing utilities                 |
| `ValidationErrors`  | Multiple validation errors                   |
| `ConformanceResult` | Result of conformance suite execution        |
| `ConformanceLevel`  | Conformance certification level              |
| `ResultSummary`     | Aggregate test counts                        |

### Functions

| Function                                               | Description                         |
| ------------------------------------------------------ | ----------------------------------- |
| `NewServer(plugin)`                                    | Create server with default registry |
| `NewServerWithRegistry(plugin, registry)`              | Create server with custom registry  |
| `NewServerWithOptions(plugin, registry, logger, info)` | Create server with all options      |
| `Serve(ctx, config)`                                   | Start gRPC server                   |
| `NewLogWriter()`                                       | Get log writer respecting env var   |
| `NewPluginLogger(name, version, level, writer)`        | Create configured logger            |
| `TracingUnaryServerInterceptor()`                      | gRPC interceptor for trace IDs      |
| `TraceIDFromContext(ctx)`                              | Extract trace ID from context       |
| `ContextWithTraceID(ctx, traceID)`                     | Inject trace ID into context        |
| `GenerateTraceID()`                                    | Generate new trace ID               |
| `LogOperation(logger, operation)`                      | Log operation with timing           |
| `NotSupportedError(resource)`                          | Create not-supported error          |
| `NoDataError(resourceID)`                              | Create no-data error                |
| `LoadManifest(path)`                                   | Load manifest from file             |
| `SaveManifest(path, manifest)`                         | Save manifest to file               |
| `NewTestServer(t, plugin)`                             | Create test server                  |
| `NewTestPlugin(t, plugin)`                             | Create test plugin helper           |
| `CreateTestResource(provider, type, props)`            | Create test resource                |
| `RunBasicConformance(plugin)`                          | Run basic conformance tests         |
| `RunStandardConformance(plugin)`                       | Run standard conformance tests      |
| `RunAdvancedConformance(plugin)`                       | Run advanced conformance tests      |
| `PrintConformanceReport(t, result)`                    | Print formatted report to test log  |
| `PrintConformanceReportTo(result, writer)`             | Print formatted report to io.Writer |
