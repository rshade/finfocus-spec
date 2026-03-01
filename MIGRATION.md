# Migration Guide: PulumiCost to FinFocus

## Overview

The project has been renamed from **PulumiCost** to **FinFocus** to align with the
industry-standard FinOps FOCUS specification. This change affects environment variables,
configuration paths, and plugin discovery. This guide provides step-by-step instructions
for migrating existing deployments.

## Environment Variables

All environment variables have been renamed from `PULUMICOST_*` to `FINFOCUS_*`. The SDK provides backwards
compatibility with fallback chains for all variables.

| Old Variable (Deprecated) | New Variable           | Description                                  | Fallback Chain |
| ------------------------- | ---------------------- | -------------------------------------------- | --------------- |
| `PULUMICOST_PLUGIN_PORT` | `FINFOCUS_PLUGIN_PORT` | Port for the plugin gRPC server              | `FINFOCUS_PLUGIN_PORT` → `PULUMICOST_PLUGIN_PORT` |
| `PULUMICOST_LOG_LEVEL`   | `FINFOCUS_LOG_LEVEL`   | Logging verbosity (debug, info, warn, error) | `FINFOCUS_LOG_LEVEL` → `PULUMICOST_LOG_LEVEL` → `LOG_LEVEL` |
| `PULUMICOST_LOG_FILE`    | `FINFOCUS_LOG_FILE`    | Path to log file (redirects from stderr)     | `FINFOCUS_LOG_FILE` → `PULUMICOST_LOG_FILE` |
| `PULUMICOST_LOG_FORMAT`  | `FINFOCUS_LOG_FORMAT`  | Log output format (json, text)               | `FINFOCUS_LOG_FORMAT` → `PULUMICOST_LOG_FORMAT` |
| `PULUMICOST_TRACE_ID`    | `FINFOCUS_TRACE_ID`    | Distributed tracing correlation ID           | `FINFOCUS_TRACE_ID` → `PULUMICOST_TRACE_ID` |
| `PULUMICOST_TEST_MODE`   | `FINFOCUS_TEST_MODE`   | Enable test mode features                    | `FINFOCUS_TEST_MODE` → `PULUMICOST_TEST_MODE` |

## Plugin Discovery Paths

The default directory for plugin discovery has changed.

| Old Path                 | New Path               |
| ------------------------ | ---------------------- |
| `~/.pulumicost/plugins/` | `~/.finfocus/plugins/` |

## Migration Steps

### 1. Update Deployment Configurations

Update your shell profiles (`.bashrc`, `.zshrc`), Kubernetes manifests, systemd service
files, and Dockerfiles to use the new environment variable names.

**Example (Bash):**

```bash
# Old
export PULUMICOST_LOG_LEVEL=debug
export PULUMICOST_PLUGIN_PORT=50051

# New
export FINFOCUS_LOG_LEVEL=debug
export FINFOCUS_PLUGIN_PORT=50051
```

### 2. Move Plugins

Move your existing plugins to the new discovery directory.

```bash
mkdir -p ~/.finfocus/plugins
mv ~/.pulumicost/plugins/* ~/.finfocus/plugins/ 2>/dev/null || true
rmdir ~/.pulumicost/plugins 2>/dev/null || true
echo "Plugins migrated successfully"
```

**Example Output:**

```text
Plugins migrated successfully
```

**Troubleshooting:**

- If the old directory doesn't exist, no error is thrown (safe for new installations)
- If plugins exist and are successfully moved, no additional output is shown
- To verify the migration: `ls -la ~/.finfocus/plugins/`

### 3. Update Plugin Configuration Files

If you have custom plugin configuration files that reference `pulumicost` paths or
variables, update them to use `finfocus` equivalents.

## SDK Changes (Go)

If you are developing plugins using the Go SDK, **you must** update your imports. The module path has changed.

### Import Updates Required

**Old Import:**

```go
import "github.com/rshade/pulumicost/sdk/go/pluginsdk"
```

**New Import (Required):**

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
```

### Complete Migration Workflow for Go Plugins

#### Step 1: Update go.mod

```bash
# Remove old module reference
go mod edit -droprequire github.com/rshade/pulumicost/sdk/go/pluginsdk

# Add new module (replace vX.X.X with desired version)
go get github.com/rshade/finfocus-spec/sdk/go/pluginsdk@vX.X.X

# Clean up dependencies
go mod tidy
```

#### Step 2: Update Source Code Imports

Replace all instances of:

```bash
sed -i 's|github.com/rshade/pulumicost/sdk/go/pluginsdk|github.com/rshade/finfocus-spec/sdk/go/pluginsdk|g' $(find . -name '*.go')
```

#### Step 3: Verify

```bash
# Build to verify imports are correct
go build ./...

# Run tests
go test ./...
```

### Breaking Changes in SDK Interfaces

#### DryRunHandler.HandleDryRun Interface Change (v0.5.7+)

The `DryRunHandler` interface signature has been updated to include a `context.Context` parameter
for proper cancellation propagation in batch operations. This is a **compile-time breaking change**
that affects all plugins implementing the DryRunHandler interface.

##### Interface Signature Change

**Before (pre-v0.5.7):**

```go
type DryRunHandler interface {
    HandleDryRun(req *pbc.DryRunRequest) (*pbc.DryRunResponse, error)
}
```

**After (v0.5.7+):**

```go
type DryRunHandler interface {
    HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error)
}
```

##### Migration Steps for Plugin Developers

If your plugin implements the `DryRunHandler` interface, you must update your implementation:

**Add context.Context parameter:**

```go
// Old implementation (will not compile with v0.5.7+)
func (p *MyPlugin) HandleDryRun(req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
    // Implementation
}

// New implementation (required for v0.5.7+)
func (p *MyPlugin) HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
    mappings := pluginsdk.AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)

    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithFieldMappings(mappings),
        pluginsdk.WithResourceTypeSupported(true),
    ), nil
}
```

##### Why This Change Was Necessary

This change enables:

- **Batch operation cancellation**: Hosts can cancel in-flight DryRun requests during batch operations
- **Timeout enforcement**: Proper timeout handling for DryRun requests
- **Request tracing**: Context propagation for distributed tracing in complex plugin chains
- **Resource cleanup**: Graceful handling of request cancellation to prevent resource leaks

##### Impact Assessment

- **Severity**: Breaking change (compile-time error)
- **Affected plugins**: Only plugins that implement the `DryRunHandler` interface
- **Detection**: Compilation will fail with signature mismatch error
- **Mitigation**: Update method signature to include `context.Context` parameter (simple mechanical change)
- **Version**: Introduced in v0.5.7

## Backwards Compatibility

The SDK implements **full backwards compatibility** with multi-layer fallback chains. If a
`FINFOCUS_*` variable is not set, the SDK automatically checks the corresponding `PULUMICOST_*`
variable.

### Fallback Chains by Variable

**Two-Layer Fallback** (most variables):

```text
FINFOCUS_* → PULUMICOST_*
```

Variables with two-layer fallback:

- `FINFOCUS_PLUGIN_PORT` → `PULUMICOST_PLUGIN_PORT`
- `FINFOCUS_LOG_FILE` → `PULUMICOST_LOG_FILE`
- `FINFOCUS_LOG_FORMAT` → `PULUMICOST_LOG_FORMAT`
- `FINFOCUS_TRACE_ID` → `PULUMICOST_TRACE_ID`
- `FINFOCUS_TEST_MODE` → `PULUMICOST_TEST_MODE`

**Three-Layer Fallback** (log level only):

```text
FINFOCUS_LOG_LEVEL → PULUMICOST_LOG_LEVEL → LOG_LEVEL
```

The log level variable has an additional generic fallback to support existing deployments that
only set `LOG_LEVEL`.

### Migration Strategy

You can migrate at your own pace:

1. **Immediate migration**: Update to `FINFOCUS_*` variables now
2. **Gradual migration**: Set both `FINFOCUS_*` and `PULUMICOST_*` temporarily (new variables take precedence)
3. **Deferred migration**: Keep using `PULUMICOST_*` variables (will work until v1.0)

## Deprecation Timeline

The old `PULUMICOST_*` environment variable names are deprecated and will be removed in a future release:

| Timeline       | Action Required                                              |
| -------------- | ------------------------------------------------------------ |
| **Now**        | Update your configurations to use `FINFOCUS_*` variables     |
| **v1.0**       | Support for `PULUMICOST_*` variables will be removed         |
| **After v1.0** | Only `FINFOCUS_*` variables are supported                    |

**Recommended Migration**: Update to `FINFOCUS_*` variables immediately to ensure your deployments
continue to work after the v1.0 release.

## Rollback Instructions

If you encounter issues after migrating, follow these steps to revert to the old configuration:

### Emergency Rollback (Temporary)

If your system is down and you need to restore service immediately:

```bash
# Restore old environment variables
export PULUMICOST_PLUGIN_PORT=$FINFOCUS_PLUGIN_PORT
export PULUMICOST_LOG_LEVEL=$FINFOCUS_LOG_LEVEL
export PULUMICOST_LOG_FILE=$FINFOCUS_LOG_FILE
export PULUMICOST_LOG_FORMAT=$FINFOCUS_LOG_FORMAT
export PULUMICOST_TRACE_ID=$FINFOCUS_TRACE_ID
export PULUMICOST_TEST_MODE=$FINFOCUS_TEST_MODE

# Restart your services with old variable names
systemctl restart your-finfocus-service
```

### Full Rollback (Revert Migration)

If you need to completely revert the migration:

1. **Revert plugin directory** (if you moved them):

   ```bash
   mkdir -p ~/.pulumicost/plugins
   mv ~/.finfocus/plugins/* ~/.pulumicost/plugins/ 2>/dev/null || true
   rmdir ~/.finfocus/plugins 2>/dev/null || true
   ```

2. **Revert environment variables** in all configuration files:
   - Shell profiles (`.bashrc`, `.zshrc`)
   - Kubernetes manifests
   - Systemd service files
   - Docker Compose files
   - Dockerfiles

3. **Revert plugin imports** (for Go plugins):

   ```bash
   # Update go.mod
   go get github.com/rshade/pulumicost/sdk/go/pluginsdk@vX.X.X
   go mod tidy

   # Update source code imports
   sed -i 's|github.com/rshade/finfocus-spec/sdk/go/pluginsdk|github.com/rshade/pulumicost/sdk/go/pluginsdk|g' $(find . -name '*.go')

   # Verify
   go build ./...
   go test ./...
   ```

4. **Restart services** and verify functionality

**Note**: Rollback is possible without code changes to the SDK. The SDK accepts `FINFOCUS_*`
variables and provides limited fallback support (see Backwards Compatibility section).

## Machine-Readable Migration Manifest

For automated migration using tools or LLMs, a JSON manifest is available in
[`llm-migration.json`](./llm-migration.json) to help AI coding assistants automatically
identify and apply these changes in downstream repositories.
