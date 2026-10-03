# FinFocus Registry Package

Domain types and validation for FinFocus plugin registry management. This package provides
enum types with optimized zero-allocation validation for plugin discovery, installation, and
lifecycle management.

## Overview

The registry package defines 8 enum types with high-performance validation:

| Enum Type | Values | Description |
|-----------|--------|-------------|
| `Provider` | 5 | Cloud providers (aws, azure, gcp, kubernetes, custom) |
| `DiscoverySource` | 4 | Plugin discovery sources (filesystem, registry, url, git) |
| `PluginStatus` | 6 | Plugin lifecycle states (available, installed, active, etc.) |
| `SecurityLevel` | 4 | Trust levels (untrusted, community, verified, official) |
| `InstallationMethod` | 4 | Install methods (binary, container, script, package) |
| `PluginCapability` | 14 | Plugin features (cost_retrieval, cost_projection, etc.) |
| `SystemPermission` | 9 | Required permissions (network_access, filesystem_read, etc.) |
| `AuthMethod` | 6 | Authentication methods (none, api_key, jwt, oauth2, etc.) |

## Quick Start

```go
import "github.com/rshade/finfocus-spec/sdk/go/registry"

// Validate provider
if !registry.IsValidProvider("aws") {
    return errors.New("invalid provider")
}

// Validate plugin status
if !registry.IsValidPluginStatus("active") {
    return errors.New("invalid status")
}

// Validate capabilities
if !registry.IsValidPluginCapability("cost_retrieval") {
    return errors.New("invalid capability")
}

// Get all valid providers
for _, provider := range registry.AllProviders() {
    fmt.Printf("Provider: %s\n", provider)
}
```

## Enum Types

### Provider

Supported cloud providers:

```go
registry.ProviderAWS        // "aws"
registry.ProviderAzure      // "azure"
registry.ProviderGCP        // "gcp"
registry.ProviderKubernetes // "kubernetes"
registry.ProviderCustom     // "custom"
```

### DiscoverySource

Plugin discovery mechanisms:

```go
registry.DiscoverySourceFilesystem // "filesystem"
registry.DiscoverySourceRegistry   // "registry"
registry.DiscoverySourceURL        // "url"
registry.DiscoverySourceGit        // "git"
```

### PluginStatus

Plugin lifecycle states:

```go
registry.PluginStatusAvailable // "available"
registry.PluginStatusInstalled // "installed"
registry.PluginStatusActive    // "active"
registry.PluginStatusInactive  // "inactive"
registry.PluginStatusError     // "error"
registry.PluginStatusUpdating  // "updating"
```

### SecurityLevel

Trust levels for plugins:

```go
registry.SecurityLevelUntrusted // "untrusted"
registry.SecurityLevelCommunity // "community"
registry.SecurityLevelVerified  // "verified"
registry.SecurityLevelOfficial  // "official"
```

### InstallationMethod

Plugin installation methods:

```go
registry.InstallationMethodBinary    // "binary"
registry.InstallationMethodContainer // "container"
registry.InstallationMethodScript    // "script"
registry.InstallationMethodPackage   // "package"
```

### PluginCapability

Plugin feature capabilities. A manifest's `specification.capabilities` accepts two kinds of value.

The 14 manifest capability strings have constants:

```go
registry.PluginCapabilityCostRetrieval   // "cost_retrieval"
registry.PluginCapabilityCostProjection  // "cost_projection"
registry.PluginCapabilityPricingSpecs    // "pricing_specs"
registry.PluginCapabilityHistoricalData  // "historical_data"
registry.PluginCapabilityRealTimeData    // "real_time_data"
registry.PluginCapabilityBatchProcessing // "batch_processing"
registry.PluginCapabilityRateLimiting    // "rate_limiting"
registry.PluginCapabilityCaching         // "caching"
registry.PluginCapabilityEncryption      // "encryption"
registry.PluginCapabilityCompression     // "compression"
registry.PluginCapabilityFiltering       // "filtering"
registry.PluginCapabilityAggregation     // "aggregation"
registry.PluginCapabilityMultiTenancy    // "multi_tenancy"
registry.PluginCapabilityAuditLogging    // "audit_logging"
```

Every `PluginCapability` value from `enums.proto` is also accepted, written as the lowercase value name
without the `PLUGIN_CAPABILITY_` prefix. `ManifestCapabilityName` converts a protocol value, so a plugin
can list the same capabilities it reports through `GetPluginInfo`:

```go
registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)       // "dry_run"
registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST) // "estimate_cost"
registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED)   // ""
```

`pricing_spec` (protocol) and `pricing_specs` (manifest string) are distinct, and both are valid.
`AllPluginCapabilities` returns the 14 strings followed by the protocol names in enum-number order.

### SystemPermission

Required system permissions:

```go
registry.SystemPermissionNetworkAccess    // "network_access"
registry.SystemPermissionFilesystemRead   // "filesystem_read"
registry.SystemPermissionFilesystemWrite  // "filesystem_write"
registry.SystemPermissionEnvRead          // "env_read"
registry.SystemPermissionEnvWrite         // "env_write"
registry.SystemPermissionProcessSpawn     // "process_spawn"
registry.SystemPermissionSocketBind       // "socket_bind"
registry.SystemPermissionSecretsAccess    // "secrets_access"
registry.SystemPermissionCloudCredentials // "cloud_credentials"
```

### AuthMethod

Plugin authentication methods:

```go
registry.AuthMethodNone      // "none"
registry.AuthMethodAPIKey    // "api_key"
registry.AuthMethodJWT       // "jwt"
registry.AuthMethodOAuth2    // "oauth2"
registry.AuthMethodMTLS      // "mtls"
registry.AuthMethodBasicAuth // "basic_auth"
```

## Validation Functions

Each enum type has corresponding validation and accessor functions:

```go
// IsValid functions - check if string is valid enum value
registry.IsValidProvider(s string) bool
registry.IsValidDiscoverySource(s string) bool
registry.IsValidPluginStatus(s string) bool
registry.IsValidSecurityLevel(s string) bool
registry.IsValidInstallationMethod(s string) bool
registry.IsValidPluginCapability(s string) bool
registry.IsValidSystemPermission(s string) bool
registry.IsValidAuthMethod(s string) bool

// All functions - get all valid values
registry.AllProviders() []Provider
registry.AllDiscoverySources() []DiscoverySource
registry.AllPluginStatuses() []PluginStatus
registry.AllSecurityLevels() []SecurityLevel
registry.AllInstallationMethods() []InstallationMethod
registry.AllPluginCapabilities() []PluginCapability
registry.AllSystemPermissions() []SystemPermission
registry.AllAuthMethods() []AuthMethod
```

## Manifest Validation

`ValidatePluginManifest(manifestJSON []byte) error` checks a manifest in the canonical form that
`pluginsdk.SaveManifest` and `pluginsdk.MarshalManifestJSON` write. Beyond metadata, providers, and
installation, it checks:

- `specification.supported_resources`: keys are clouds (`aws`, `azure`, `gcp`, `kubernetes`,
  `custom`); each entry has only `resource_types` (required, non-empty, unique, 1 to
  `MaxResourceTypeLength` = 256 characters), `billing_modes` (unique, from
  `AllManifestBillingModes`), and `regions` (unique, 2 to 30 characters)
- `specification.capabilities`: unique values accepted by `IsValidPluginCapability`
- `specification.service_definition.methods`: unique `CostSourceService` RPC names

The method and capability lists are read from the proto descriptors, and a test compares them with
`schemas/plugin_manifest.schema.json`, so the SDK and the schema give the same verdict.

```go
registry.AllServiceMethods() []string            // CostSourceService RPCs, declaration order
registry.IsValidServiceMethod(s string) bool
registry.AllManifestBillingModes() []string      // manifest billing modes (not pricing.BillingMode)
registry.IsValidManifestBillingMode(s string) bool
registry.ManifestCapabilityName(c pbc.PluginCapability) string
```

A resource type entry is the string the plugin matches against `ResourceDescriptor.resource_type`.
Use the full type token (`azure-native:compute:VirtualMachine`) and list a native package's types
under their cloud.

## Performance

All validation functions are optimized for zero-allocation performance:

- **5-12 ns/op** across all enum types
- **0 B/op, 0 allocs/op** for all validation operations
- **~608 bytes** total memory for all enum slices
- **2x faster** than map-based alternatives

Run benchmarks:

```bash
go test -bench=. -benchmem ./sdk/go/registry/
```
