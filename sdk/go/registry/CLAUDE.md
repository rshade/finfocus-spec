# CLAUDE.md - Registry Package

**Package**: `github.com/rshade/finfocus-spec/sdk/go/registry`
**Purpose**: Domain types and validation for FinFocus plugin registry management

## Overview

The registry package provides enum types, validation functions, and domain models for plugin discovery, installation,
and lifecycle management in the FinFocus ecosystem.

## Core Components

### Enum Types (8 Total)

The package defines 8 enum types with optimized zero-allocation validation:

1. **Provider** (5 values): `aws`, `azure`, `gcp`, `kubernetes`, `custom`
2. **DiscoverySource** (4 values): `filesystem`, `registry`, `url`, `git`
3. **PluginStatus** (6 values): `available`, `installed`, `active`, `inactive`, `error`, `updating`
4. **SecurityLevel** (4 values): `untrusted`, `community`, `verified`, `official`
5. **InstallationMethod** (4 values): `binary`, `container`, `script`, `package`
6. **PluginCapability** (32 values): the 14 manifest strings (`cost_retrieval`, `cost_projection`, ...) plus
   the 18 `pbc.PluginCapability` names built at init (`dry_run`, `estimate_cost`, ...)
7. **SystemPermission** (9 values): `network_access`, `filesystem_read`, `filesystem_write`, etc.
8. **AuthMethod** (6 values): `none`, `api_key`, `jwt`, `oauth2`, `mtls`, `basic_auth`

### Validation Pattern (Optimized ✅)

**Status**: Fully optimized as of 2025-11-17

The package uses **optimized slice-based validation** with package-level variables for zero-allocation performance:

```go
// Package-level slice allocated once at initialization
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allProviders = []Provider{
    ProviderAWS, ProviderAzure, ProviderGCP, ProviderKubernetes, ProviderCustom,
}

// Accessor returns reference to package variable (zero allocation)
func AllProviders() []Provider {
    return allProviders
}

// Validation uses direct slice iteration (zero allocation)
func IsValidProvider(p string) bool {
    provider := Provider(p)
    for _, validProvider := range allProviders {
        if provider == validProvider {
            return true
        }
    }
    return false
}
```

**Performance Characteristics**:

- **Zero allocation**: 0 B/op, 0 allocs/op across all enum types
- **High performance**: 5-12 ns/op (Provider: 7 ns/op, PluginCapability: 10.5 ns/op)
- **Memory efficient**: 608 bytes total for all 8 enums
- **2x faster**: Compared to map-based alternatives (7 ns/op vs 15 ns/op)

See [validation-pattern.md](../../../specs/001-domain-enum-optimization/validation-pattern.md) for complete pattern documentation.

## Build Commands

### Testing

```bash
# From this directory
go test
go test -v
go test -bench=. -benchmem  # Run benchmarks with memory profiling

# From repository root (recommended)
cd ../../../ && make test
cd ../../../ && go test ./sdk/go/registry/
```

### Benchmarking

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark categories
go test -bench=BenchmarkIsValid -benchmem          # All validation benchmarks
go test -bench=BenchmarkValidation_ -benchmem      # Scalability benchmarks
go test -bench=MapBased -benchmem                  # Map comparison benchmarks
```

### Development

```bash
# Build package
go build

# Format and tidy
go fmt
go mod tidy

# Full validation from root
cd ../../../ && make lint && make validate
```

## Usage Patterns

### Basic Validation

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
```

**Performance**: All validations complete in < 12 ns with zero allocation.

### Iterating Enum Values

```go
// Get all valid providers
for _, provider := range registry.AllProviders() {
    fmt.Printf("Provider: %s\n", provider)
}

// Get all valid capabilities
for _, capability := range registry.AllPluginCapabilities() {
    fmt.Printf("Capability: %s\n", capability)
}
```

**Performance**: Zero allocation when retrieving enum lists.

### High-Frequency Validation

```go
// Validate thousands of plugin manifests efficiently
for _, manifest := range pluginManifests {
    if !registry.IsValidProvider(manifest.Provider) {
        return fmt.Errorf("invalid provider: %s", manifest.Provider)
    }

    for _, capability := range manifest.Capabilities {
        if !registry.IsValidPluginCapability(capability) {
            return fmt.Errorf("invalid capability: %s", capability)
        }
    }
}
```

**Performance**: Zero allocation even when validating thousands of manifests.

## Standard Domain Enum Pattern

All 8 enum types in this package follow a consistent pattern. Use this as a template for new domain types:

### Pattern Components

1. **Type Definition** - String-backed type for type safety
2. **Constants** - Exported constants with descriptive comments
3. **Package-Level Slice** - Pre-allocated for zero-allocation validation
4. **AllXxx() Function** - Returns all valid values
5. **String() Method** - Returns string representation
6. **IsValidXxx() Function** - Validates string input

### Complete Pattern Example

```go
// TypeName represents [description of what it represents].
type TypeName string

const (
    // TypeNameValue1 indicates [description].
    TypeNameValue1 TypeName = "value1"
    // TypeNameValue2 indicates [description].
    TypeNameValue2 TypeName = "value2"
)

// allTypeNames is a package-level slice containing all valid TypeName values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allTypeNames = []TypeName{
    TypeNameValue1, TypeNameValue2,
}

// AllTypeNames returns all valid type names.
func AllTypeNames() []TypeName {
    return allTypeNames
}

// String returns the type name as a lowercase string value.
func (t TypeName) String() string {
    return string(t)
}

// IsValidTypeName checks if a type name is valid.
func IsValidTypeName(name string) bool {
    typeName := TypeName(name)
    for _, validName := range allTypeNames {
        if typeName == validName {
            return true
        }
    }
    return false
}
```

### Cross-Package Reference

The `pricing` package (`sdk/go/pricing/`) uses similar enum patterns for `BillingMode` and `Provider`. When adding
new enum types, ensure consistency with both packages.

## Adding New Enum Values

### Step 1: Add Constant

```go
const (
    ProviderNewCloud Provider = "newcloud"  // Add new provider constant
)
```

### Step 2: Update Package-Level Variable

```go
var allProviders = []Provider{
    ProviderAWS, ProviderAzure, ProviderGCP, ProviderKubernetes, ProviderCustom,
    ProviderNewCloud,  // Add to package-level slice
}
```

### Step 3: Update Tests

```go
func TestAllProvidersCompleteness(t *testing.T) {
    expected := 6  // Update expected count (was 5)
    if len(registry.AllProviders()) != expected {
        t.Errorf("Expected %d providers, got %d", expected, len(registry.AllProviders()))
    }
}
```

### Step 4: Add Validation Test

```go
func TestIsValidProvider_NewCloud(t *testing.T) {
    if !registry.IsValidProvider("newcloud") {
        t.Error("newcloud should be valid provider")
    }
}
```

## Adding a New Provider

When adding support for a new cloud provider:

1. **Add to registry package** (`sdk/go/registry/domain.go`):
   - Add `ProviderXxx Provider = "xxx"` constant
   - Add to `allProviders` slice
   - Update tests with new expected count

2. **Consider pricing package** (`sdk/go/pricing/domain.go`):
   - Check if Provider enum there also needs updating
   - Ensure consistency between packages

3. **Add example spec** (`examples/specs/`):
   - Create example JSON demonstrating the new provider's pricing patterns

4. **Update documentation**:
   - Update this CLAUDE.md with new provider count
   - Document provider-specific considerations

## Performance Optimization Details

### Benchmark Results

From `specs/001-domain-enum-optimization/performance-results.md`:

```text
BenchmarkIsValidProvider-4                     151766487          7.034 ns/op        0 B/op        0 allocs/op
BenchmarkIsValidDiscoverySource-4              235204474          5.001 ns/op        0 B/op        0 allocs/op
BenchmarkIsValidPluginStatus-4                 167060607          7.333 ns/op        0 B/op        0 allocs/op
BenchmarkIsValidPluginCapability-4             100000000         10.52 ns/op        0 B/op        0 allocs/op
```

**Comparison with Map-Based Validation**:

```text
BenchmarkIsValidProvider_MapBased-4            88503589         15.08 ns/op        0 B/op        0 allocs/op
BenchmarkIsValidPluginCapability_MapBased-4    89727180         16.06 ns/op        0 B/op        0 allocs/op
```

**Conclusion**: Slice-based validation is 2x faster for small enums (4-14 values).

### Memory Footprint

Total memory for all 8 package-level enum slices:

```text
allProviders:            40 bytes (5 values)
allDiscoverySources:     32 bytes (4 values)
allPluginStatuses:       48 bytes (6 values)
allSecurityLevels:       32 bytes (4 values)
allInstallationMethods:  32 bytes (4 values)
allPluginCapabilities:  112 bytes (14 values)
allSystemPermissions:    72 bytes (9 values)
allAuthMethods:          48 bytes (6 values)
────────────────────────────────────────────
Total:                  ~608 bytes
```

**Map-based alternative**: ~3.5 KB (6x larger)

## Pattern Guidelines

### When to Use This Pattern

✅ **Use for enums with < 20 values** (optimal performance range)

This pattern provides:

- Zero allocation (0 B/op, 0 allocs/op)
- Fast validation (5-12 ns/op)
- Simple implementation
- Easy maintenance

### When to Consider Alternatives

⚠️ **Consider map-based validation if:**

- Enum has > 40 values (approaching crossover point)
- Constant-time lookup required regardless of size
- Memory footprint is not a concern

### Validation Pattern Reference

Complete pattern documentation: `specs/001-domain-enum-optimization/validation-pattern.md`

## Common Issues

### Issue: Linter Complains About Global Variables

**Symptom**: `gochecknoglobals` linter warning

**Solution**: Add nolint directive with explanation:

```go
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allProviders = []Provider{...}
```

### Issue: New Enum Value Not Validated

**Symptom**: `IsValidXxx()` returns false for new value

**Checklist**:

1. ✓ Added constant definition
2. ✓ Added to package-level slice (`allXxx`)
3. ✓ Updated completeness test expected count
4. ✓ Added validation test case

## Related Documentation

- **Validation Pattern**: `specs/001-domain-enum-optimization/validation-pattern.md`
- **Performance Results**: `specs/001-domain-enum-optimization/performance-results.md`
- **Research Analysis**: `specs/001-domain-enum-optimization/research.md`
- **Quickstart Guide**: `specs/001-domain-enum-optimization/quickstart.md`
- **Parent CLAUDE.md**: `sdk/go/CLAUDE.md`
