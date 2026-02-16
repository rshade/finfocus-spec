# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **finfocus-spec**, a repository that provides the canonical protocol and schemas for FinFocus plugins. It defines:

- gRPC service definitions for cost source plugins
- JSON schemas for pricing specifications
- Go SDK with generated protobuf code and helper types

## Build Commands

### Core Build Commands

- `make generate` - Generate Go code from protobuf definitions (installs buf locally in bin/)
- `make tidy` - Run `go mod tidy` to clean up dependencies
- `make test` - Run all Go tests including integration tests
- `make validate` - Run tests, linting, and npm validations together
- `make clean` - Remove generated proto files
- `make clean-all` - Remove generated files and local tools (bin/)
- `go build ./...` - Build all Go packages
- `go test -bench=. -benchmem ./sdk/go/testing/` - Run performance benchmarks

### Linting Commands

- `make lint` - Run all linting (Go, buf, markdown, and YAML)
- `make lint-go` - Run Go linting (golangci-lint and buf lint)
- `make lint-markdown` - Run markdown linting with markdownlint-cli2
- `make lint-markdown-fix` - Auto-fix markdown linting issues
- `make lint-yaml` - Run YAML linting on GitHub workflows
- `make lint-yaml-fix` - Auto-fix YAML linting issues

### NPM/Schema Validation Commands

- `make validate-schema` - Validate JSON schema syntax
- `make validate-examples` - Validate example files against schema
- `make validate-npm` - Run all npm validations (schema + examples)
- `npm run lint:markdown` - Direct npm markdown linting
- `npm run validate` - Direct npm validation command

## Architecture

### Core Components

**Proto Definition (`proto/finfocus/v1/costsource.proto`)**

- Defines `CostSource` gRPC service with RPCs for: Name, Supports, GetActualCost, GetProjectedCost, GetPricingSpec
- Contains message definitions for requests/responses
- Uses Google protobuf types (Empty, Timestamp)

**JSON Schema (`schemas/pricing_spec.schema.json`)**

- Validates PricingSpec documents
- Defines required fields: provider, resource_type, billing_mode, rate_per_unit, currency
- Enforces billing_mode enum values and data types

**Go SDK (`sdk/go/`)**

- `sdk/go/proto/` - Generated protobuf Go code (do not edit manually)
- `sdk/go/registry/` - Plugin registry domain types with optimized zero-allocation validation (8 enum types)
- `sdk/go/pricing/domain.go` - BillingMode enum constants and validation helpers
- `sdk/go/pricing/validate.go` - JSON schema validation for PricingSpec documents
- `sdk/go/testing/` - Comprehensive plugin testing framework

**Performance Optimization** (Registry Package):

The registry package implements **zero-allocation enum validation** using package-level slice variables:

- **Performance**: 5-12 ns/op, 0 allocs/op across all 8 enum types
- **Pattern**: Package-level variables instead of function-returned slices
- **Memory**: ~608 bytes total for all enums (vs ~3.5 KB for map-based alternatives)
- **Speed**: 2x faster than map-based validation for small enums (4-14 values)

See `specs/001-domain-enum-optimization/` for complete documentation and performance analysis.

**Plugin Metadata (GetPluginInfo RPC):**

The SDK implements `GetPluginInfo` RPC for retrieving plugin metadata:

- **GetPluginInfo**: RPC returning Name, Version, SpecVersion, and Providers
- **PluginInfo**: Struct for configuring metadata in `ServeConfig`
- **PluginInfoProvider**: Optional interface for dynamic metadata
- **SpecVersion**: Constant defining the compiled SDK spec version (validated at init)

### Capability Discovery Pattern

The SDK supports a dual-mode capability discovery system:

1. **Auto-Discovery (Default)**: Capabilities are automatically detected based on the interfaces
   implemented by the plugin struct.
   - `DryRunHandler` -> `PLUGIN_CAPABILITY_DRY_RUN`
   - `RecommendationsProvider` -> `PLUGIN_CAPABILITY_RECOMMENDATIONS`
   - `BudgetsProvider` -> `PLUGIN_CAPABILITY_BUDGETS`
   - `DismissProvider` -> `PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS`

```go
// Plugin with DryRunHandler implementation
type MyPlugin struct {
    proto.UnimplementedCostSourceServiceServer
}

func (p *MyPlugin) HandleDryRun(req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
    return pluginsdk.NewDryRunResponse(
        pluginsdk.WithResourceTypeSupported(true),
    ), nil
}

// Capability auto-discovered: PLUGIN_CAPABILITY_DRY_RUN
info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0")
```

1. **Manual Override (Advanced)**: Capabilities can be explicitly defined using `WithCapabilities()`.
   This **completely replaces** auto-discovery. Use this to dynamically enable/disable features or
   when proxying.

```go
info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0",
    pluginsdk.WithCapabilities(
        pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
    ),
)
```

**Backward Compatibility**:
The SDK automatically populates both the modern `capabilities` enum list and the legacy `metadata`
map (e.g., `"supports_dry_run": "true"`) to ensure compatibility with older hosts.

**GetPluginInfo Configuration Patterns:**

```go
// Pattern 1: Static configuration via ServeConfig (recommended for most plugins)
info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0",
    pluginsdk.WithProviders("aws", "azure"),
    pluginsdk.WithMetadata("build_date", "2024-01-15"),
)
pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
    Plugin:     &MyPlugin{},
    PluginInfo: info,
})

// Pattern 2: Dynamic metadata via PluginInfoProvider interface
// Use when metadata must be computed at runtime
type MyDynamicPlugin struct {
    // ...
}

func (p *MyDynamicPlugin) GetPluginInfo(ctx context.Context, req *pbc.GetPluginInfoRequest) (
    *pbc.GetPluginInfoResponse, error) {
    return &pbc.GetPluginInfoResponse{
        Name:        "my-plugin",
        Version:     p.computeVersion(), // Dynamic
        SpecVersion: pluginsdk.SpecVersion,
        Providers:   p.discoverProviders(), // Dynamic
    }, nil
}

// Client-side error handling for legacy plugins
resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
if err != nil {
    if status.Code(err) == codes.Unimplemented {
        // Legacy plugin - use fallback values
        return &PluginMetadata{Name: "unknown", Version: "unknown"}
    }
    return nil, fmt.Errorf("GetPluginInfo failed: %w", err)
}
```

**Testing Framework (`sdk/go/testing/`)**

- `harness.go` - In-memory gRPC test harness with bufconn
- `mock_plugin.go` - Configurable mock plugin implementation with DryRun support
- `integration_test.go` - Comprehensive integration tests for all RPC methods
- `benchmark_test.go` - Performance benchmarks with memory profiling (including DryRun)
- `conformance_test.go` - Multi-level plugin conformance testing (Basic/Standard/Advanced)
- `dry_run_conformance_test.go` - DryRun capability conformance tests
- `focus13_conformance_test.go` - FOCUS 1.3 backward compatibility and feature tests
- `README.md` - Complete testing guide for plugin developers

**FOCUS 1.3 Support (`sdk/go/pluginsdk/`)**

The pluginsdk implements FOCUS 1.3 FinOps specification extensions:

- **New Columns (8 fields)**: AllocatedMethodId, AllocatedMethodDetails, AllocatedResourceId,
  AllocatedResourceName, AllocatedTags, ServiceProviderName, HostProviderName, ContractApplied
- **ContractCommitment Dataset**: Supplemental dataset for tracking contractual obligations
- **Deprecated Fields**: `provider_name` → `service_provider_name`, `publisher` → `host_provider_name`

Key files:

- `focus_builder.go` - FocusRecordBuilder with FOCUS 1.3 methods (WithAllocation, WithServiceProvider, etc.)
- `contract_commitment_builder.go` - ContractCommitmentBuilder for commitment records
- `focus_conformance.go` - Validation rules including allocation consistency

Performance (FOCUS 1.3 builder methods):

- Simple setters: < 1 ns/op, 0 allocs/op
- Allocation methods: 1.5-1.8 ns/op, 0 allocs/op
- Tag operations: ~130 ns/op (map copy overhead)

**Forecasting Primitives (`sdk/go/pricing/growth.go`)**

The pricing package provides growth projection helpers for cost forecasting:

- **GrowthType Enum**: NONE, LINEAR, EXPONENTIAL (UNSPECIFIED treated as NONE)
- **Growth Formulas**:
  - Linear: `cost = baseCost * (1 + rate * periods)`
  - Exponential: `cost = baseCost * (1 + rate)^periods`
- **Validation**: `ValidateGrowthParams()` validates growth type and rate combinations
- **Warnings**: `CheckGrowthWarnings()` detects unrealistic assumptions

Key files:

- `growth.go` - Growth calculation and validation functions
- `growth_test.go` - Comprehensive tests including overflow edge cases

Constants:

- `HighGrowthRateThreshold = 1.0` (100% per period triggers warning)
- `LongProjectionThreshold = 36` (months for exponential projection warning)
- `MinValidGrowthRate = -1.0` (minimum allowed rate)

Usage patterns:

```go
// Apply growth projection
cost := pricing.ApplyGrowth(baseCost, pbc.GrowthType_GROWTH_TYPE_LINEAR, &rate, periods)

// Validate parameters
err := pricing.ValidateGrowthParams(growthType, &rate)

// Check for warnings
warnings := pricing.CheckGrowthWarnings(growthType, &rate, periods)
```

See `specs/030-forecasting-primitives/` for complete specification and `sdk/go/pricing/README.md`
for detailed documentation.

**DryRun Capability (`sdk/go/pluginsdk/dry_run.go`)**

The DryRun feature enables hosts to query plugin field mapping capabilities without cost data retrieval:

- **DryRun RPC**: Standalone RPC for capability introspection
- **dry_run flag**: Optional flag on GetActualCost/GetProjectedCost RPCs
- **FieldSupportStatus enum**: SUPPORTED, UNSUPPORTED, CONDITIONAL, DYNAMIC
- **Performance**: <100ms p99 latency requirement (no external API calls)

Key helpers:

- `FocusFieldNames()` - Returns all ~66 FOCUS 1.2/1.3 field names
- `NewFieldMapping()` - Creates field mapping with status and optional description
- `AllFieldsWithStatus()` - Creates mappings for all fields with given status
- `SetFieldStatus()` - Updates specific field status in mapping slice
- `NewDryRunResponse()` - Builder for DryRunResponse with functional options

Usage patterns:

```go
// Create field mappings for supported resource type
mappings := pluginsdk.AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)

// Customize specific fields
mappings = pluginsdk.SetFieldStatus(mappings, "availability_zone",
    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
    pluginsdk.WithConditionDescription("Only for multi-AZ resources"))

// Build response
resp := pluginsdk.NewDryRunResponse(
    pluginsdk.WithFieldMappings(mappings),
    pluginsdk.WithResourceTypeSupported(true),
    pluginsdk.WithConfigurationValid(true),
)
```

See `specs/032-plugin-dry-run/` for complete specification and `sdk/go/pluginsdk/README.md`
for implementation guide.

**Examples (`examples/`)**

- `examples/specs/` - 8 comprehensive cross-vendor JSON examples
- `examples/README.md` - Documentation of all billing models and examples

### Generated Code

The `sdk/go/proto/` directory contains generated Go protobuf code. To regenerate:

1. Run `make generate` (automatically installs buf v1.32.1 locally in bin/)
2. Generated code is automatically validated in CI

### Code Generation Dependencies

- **buf** - Protocol buffer toolchain (installed locally in bin/ via make generate)
- **google.golang.org/protobuf** - Go protobuf runtime
- **google.golang.org/grpc** - gRPC Go implementation
- **golangci-lint** - Go linting (installed via make depend)
- **Node.js >=22** - Required for npm commands and markdown linting
- **markdownlint-cli2** - Markdown linting tool
- **ajv** - JSON schema validation
- **yamllint** - YAML linting tool (install with `pip install yamllint` or `brew install yamllint`)

### Local Tool Management

The project uses local tool installation to avoid version conflicts:

- `bin/buf` - buf CLI v1.32.1 installed automatically
- Tools are excluded from git via `.gitignore`
- `make clean-all` removes all local tools

## Development Workflow

1. **Setup**: Ensure Node.js >=22 and npm >=10 are installed, then run `npm install`
2. **Modify Proto**: Edit `proto/finfocus/costsource.proto`
3. **Update Schema**: Edit `schemas/pricing_spec.schema.json` if PricingSpec message changes
4. **Regenerate**: Run `make generate` to update Go bindings
5. **Update Types**: Modify helper code in `sdk/go/pricing/` as needed
6. **Test**: Run `make validate` to run tests, linting, and npm validations
7. **Verify**: Run integration tests with `go test -v ./sdk/go/testing/`
8. **Format**: Use `make lint-markdown-fix` to auto-fix markdown issues

## Testing Workflow

### Integration Testing

- Use `TestHarness` for in-memory gRPC testing with bufconn
- Create mock plugins with `NewMockPlugin()` for configurable behavior
- Run comprehensive tests: `go test -v ./sdk/go/testing/`

### Performance Testing

- Run benchmarks: `go test -bench=. -benchmem ./sdk/go/testing/`
- Measure all RPC methods with memory profiling
- Test different data sizes and concurrent requests

### Conformance Testing

- **Basic**: Required for all plugins - core functionality
- **Standard**: Recommended for production - reliability and consistency
- **Advanced**: High-performance requirements - scalability and performance

### Example Usage

```go
// Create test harness
plugin := &MyPluginImpl{}
harness := plugintesting.NewTestHarness(plugin)
harness.Start(t)
defer harness.Stop()

// Run conformance tests
result, err := plugintesting.RunStandardConformance(plugin)
if err != nil {
    t.Fatalf("conformance tests failed to run: %v", err)
}
if !result.Passed() {
    t.Errorf("Plugin failed conformance: %d/%d tests failed",
        result.Summary.Failed, result.Summary.Total)
}
```

## Package Structure

```text
github.com/rshade/finfocus-spec/sdk/go/proto   # Generated protobuf code
github.com/rshade/finfocus-spec/sdk/go/pricing # Domain types and validation (formerly 'types')
github.com/rshade/finfocus-spec/sdk/go/testing # Plugin testing framework
```

## Schema Validation

The pricing package embeds the JSON schema and provides `ValidatePricingSpec(doc []byte) error`
for validating PricingSpec JSON documents against the schema.

## Versioning

Follow semantic versioning for proto changes:

- MAJOR: Breaking proto changes
- MINOR: Backward-compatible additions
- PATCH: Bug fixes, documentation

Tag releases as `v0.1.0`, `v1.0.0`, etc.

## Changelog Management

**IMPORTANT**: The `CHANGELOG.md` is automatically generated by **release-please**. Do NOT manually edit it.

### How It Works

- release-please automatically generates changelog entries from conventional commit messages
- PRs are created automatically when commits are pushed to main
- Version bumps and changelog updates happen through the release-please PR

Configuration is defined in `release-please-config.json` at the repository root, and automation runs via `.github/workflows/release-please.yml`.

### Commit Message Format

Use conventional commits to ensure proper changelog generation:

- `feat:` - New features (generates "Features" section)
- `fix:` - Bug fixes (generates "Bug Fixes" section)
- `perf:` - Performance improvements
- `docs:` - Documentation changes
- `chore:` - Maintenance tasks (not included in changelog)

### What NOT to Do

- Do NOT manually edit CHANGELOG.md - edits will be overwritten by release-please
- Do NOT add entries to `## [Unreleased]` section manually
- Do NOT modify version numbers or release dates

### Changelog Commands

```bash
# Validate changelog format
npm run lint:changelog

# Full markdown validation (includes changelog)
npm run lint:markdown
```

- Project board: <https://github.com/users/rshade/projects/3>

## Commit Message Validation

This project uses **Lefthook** with **commitlint** to enforce Conventional Commits.

```bash
make install-lefthook    # Install git hooks
make commitlint          # Validate last commit
make validate-commit     # Validate PR_MESSAGE.md or last commit
```

Configuration: `lefthook.yml`, `commitlint.config.js`

## Common Issues & Solutions

### YAML Linting Configuration

- Issue: yamllint errors with default configuration
- Solution: Created `.yamllint` configuration file with sensible defaults
- Configuration disables document-start rule and sets line length to 120
- Use `make lint-yaml` to check YAML files and `make lint-yaml-fix` to auto-fix issues

### Dot Import Linting Issues

- Issue: golangci-lint flags dot imports (`. "package"`) as style violations
- Solution: Replace with explicit imports and use package prefixes
- Pattern: Change `import . "pkg"` to `import "pkg"` and update all function calls to use `pkg.Function()`
- Special case: When importing custom package with same name as stdlib (e.g., `testing`),
  use import alias: `import plugintesting "custom/testing"`

### Package Naming Conflicts

- Issue: Import name conflicts between stdlib and custom packages (e.g., `testing` vs custom `testing` package)
- Solution: Use import aliases to disambiguate: `import plugintesting "github.com/repo/testing"`
- Pattern: Rename one of the imports with a descriptive alias, typically the custom package

### Package Renaming Process

- Issue: Need to rename package for better naming conventions
- Solution: Systematic approach to avoid breaking changes:
  1. `mv old_package new_package` (rename directory)
  2. Update `package` declarations in all `.go` files
  3. Update import paths in all files
  4. Update package references in code (e.g., `old.Function()` to `new.Function()`)
  5. Update test package names (`package old_test` → `package new_test`)
- Verification: Run `go build ./...`, `make test`, and `make lint` to ensure no breakage

### Mock Plugin Implementation

- Issue: gRPC method name conflicts in mock plugins
- Solution: Use `PluginName` field instead of `Name` to avoid conflicts with RPC method names
- Pattern: Separate data fields from method names in struct design

### Integration Testing Setup

- Issue: Network-based testing complexity and flakiness
- Solution: Use `bufconn` for in-memory gRPC testing in `TestHarness`
- Pattern: Always prefer in-memory testing for unit/integration tests

### Tool Management Issues

- Issue: buf CLI version conflicts and system installation requirements
- Solution: Install tools locally in `bin/` directory with version pinning
- Pattern: `bin/toolname` with automatic installation in Makefile

### CI Pipeline Structure

- Issue: Missing integration test coverage and performance tracking
- Solution: Separate CI jobs for unit tests, integration tests, and benchmarks
- Pattern: Parallel job execution with artifact collection for benchmarks

## Best Practices Discovered

### Testing Framework Architecture

1. **Harness Pattern**: Use in-memory gRPC with bufconn for fast, reliable testing
2. **Mock Configurability**: Support error injection, delays, and custom behavior
3. **Conformance Levels**: Implement Basic/Standard/Advanced hierarchy for certification
4. **Performance Baselines**: Establish response time and memory usage benchmarks

### SDK Development Patterns

1. **Generated vs Helper Code**: Separate protobuf generation from helper utilities
2. **Validation Integration**: Embed JSON schema for runtime validation
3. **Example Completeness**: Provide comprehensive cross-vendor examples
4. **Documentation Strategy**: Use specialized agents for technical writing

### CI/CD Optimization

1. **Tool Installation**: Local installation avoids version conflicts
2. **Validation Gates**: Generated code must be up-to-date in CI
3. **Test Coverage**: Unit → Integration → Conformance → Performance progression
4. **Artifact Collection**: Store benchmark results for performance tracking

### Protocol Buffer Best Practices

1. **Forward Compatibility**: Use `UnimplementedServer` embedding
2. **Validation Functions**: Create comprehensive validators for all message types
3. **Error Handling**: Use proper gRPC status codes with meaningful messages
4. **Testing Support**: Design messages to support comprehensive testing scenarios

### FOCUS 1.3 Development Patterns

1. **Deprecation Handling**: Log warnings via zerolog when both deprecated and replacement fields set
2. **Backward Compatibility**: New fields have zero/empty defaults that don't affect existing behavior
3. **Builder Pattern**: Fluent API with method chaining (return `*Builder` from all methods)
4. **Validation Rules**: AllocatedMethodId requires AllocatedResourceId (fail-fast at Build())
5. **Cross-Dataset References**: ContractApplied is opaque reference (no validation against commitment dataset)
6. **Variable Naming**: Use `ID` suffix per Go conventions (e.g., `commitmentID` not `commitmentId`)

FOCUS 1.3 Migration:

- `provider_name` → `WithServiceProvider()` (service_provider_name)
- `publisher` → `WithHostProvider()` (host_provider_name)
- New allocation fields via `WithAllocation()`, `WithAllocatedResource()`, `WithAllocatedTags()`
- Contract commitment linking via `WithContractApplied()`

## Development Commands Reference

### Daily Development

```bash
# Setup development environment
npm install            # Install npm dependencies
make generate          # Install buf locally and generate code
make validate          # Run tests, linting, and npm validations

# Testing
go test -v ./sdk/go/testing/                    # Integration tests
go test -bench=. -benchmem ./sdk/go/testing/    # Performance benchmarks
go test -v -run TestConformance ./sdk/go/testing/  # Conformance tests

# Linting
make lint              # Run all linting (Go, buf, markdown, YAML)
make lint-markdown     # Run markdown linting only
make lint-markdown-fix # Auto-fix markdown issues
make lint-yaml         # Run YAML linting
make lint-yaml-fix     # Auto-fix YAML issues

# Schema Validation
make validate-schema   # Validate JSON schema syntax
make validate-examples # Validate example files against schema
make validate-npm      # Run all npm validations

# Cleanup
make test              # Run unit tests only
make clean-all         # Clean all generated files and tools
```

### Cross-Vendor Example Validation

```bash
# Validate all JSON examples against schema
for file in examples/specs/*.json; do
    echo "Validating $file..."
    go run validate_examples.go "$file"
done
```

### Plugin Development Testing

```bash
# Test your plugin implementation
go test -v -run TestBasicPluginFunctionality
go test -v -run TestConformance
go test -bench=BenchmarkAllMethods
```

## Session Learnings and Solutions

### Markdown Linting Configuration

- **Issue**: Markdown linter processing thousands of node_modules files (950+ errors)
- **Solution**: Create `.markdownlintignore` file and update package.json with exclusions
- **Commands**:

  ```bash
  npm run lint:markdown        # Check markdown files
  npm run lint:markdown:fix    # Auto-fix markdown issues
  ```

- **Pattern**: Always exclude `node_modules/`, temporary files in both `.markdownlintignore` and package.json

### JSON Schema Validation Issues

- **Issue**: JSON Schema validation failing with invalid keywords and format warnings
- **Solution**:
  1. Remove `version` field from schemas (not a valid JSON Schema keyword)
  2. Use `--strict=false` flag for ajv commands
  3. Install and configure ajv-formats dependency
- **Command**: `npm run validate:schema` with `--strict=false` flag

### AJV Compilation Errors

- **Issue**: AJV can't resolve `$schema` references in validation scripts
- **Solution**: Remove `$schema` field before compilation in `validate_examples.js`
- **Pattern**: Clean schema objects before AJV compilation to avoid resolution errors

### CI/CD Debugging

- **Commands**:

  ```bash
  gh run view <run-id> --log-failed     # Get detailed CI failure logs
  gh pr checks <pr-number>              # Quick PR check status overview
  gh run view <run-id> --job <job-id> --log-failed  # Specific job logs
  ```

### Dependency Management

- **Issue**: CI failing due to out-of-sync lock files
- **Solution**: Always sync dependency files before committing
- **Workflow**:

  ```bash
  npm install       # Update package-lock.json
  go mod tidy       # Update go.mod and go.sum
  git add package-lock.json go.mod go.sum
  ```

### Workflow Optimizations

1. **Markdown Fixes**: Run auto-fix first, then manual fixes for remaining issues
2. **CI Debugging**: Use `gh run view --log-failed` for specific error details
3. **Dependency Updates**: Always run both `npm install` and `go mod tidy` together

### Directory-Specific CLAUDE.md Files

- **Multiple CLAUDE.md Strategy**: Use `/init` in each important directory for context-aware guidance
- **Recommended directories for CLAUDE.md**:
  - `sdk/go/pricing/` (domain logic, billing modes, validation)
  - `sdk/go/testing/` (testing framework, harness, mocks)
  - `examples/` (pricing spec patterns, cross-vendor examples)
  - `schemas/` (JSON schema validation patterns)
  - `.claude/agents/` (agent configurations and prompts)
- **Process**: Use `cd <directory> && /init` to create directory-specific guidance
- **Benefits**: Context-aware content, proper tool detection, inheritance + specialization

### Markdown Linting Advanced Configuration

- **MD024 duplicate headings**: Use `"siblings_only": true` in `.markdownlint.json` to allow duplicate headings across
  different sections (needed for Keep a Changelog format)
- **CHANGELOG.md integration**: With proper MD024 configuration, CHANGELOG.md can be included in standard markdown
  linting pipeline
- **Configuration pattern**:

  ```json
  {
    "MD024": {
      "siblings_only": true
    }
  }
  ```

## Directory-Specific CLAUDE.md Documentation

This repository uses a **multi-level CLAUDE.md strategy** with specialized guidance files in key directories to provide
context-aware development assistance. Each directory-specific CLAUDE.md file inherits from this root file and adds
specialized knowledge for its domain.

### Directory Structure

The following directories contain specialized CLAUDE.md files:

- **`.claude/agents/CLAUDE.md`** - Agent system configuration and specialized agent prompts
- **`examples/CLAUDE.md`** - Examples directory with validation architecture and cross-provider patterns
- **`examples/specs/CLAUDE.md`** - Specific PricingSpec JSON examples with billing model coverage
- **`schemas/CLAUDE.md`** - JSON Schema validation patterns and schema evolution strategies
- **`sdk/go/CLAUDE.md`** - Go SDK overview with package structure and development patterns
- **`sdk/go/pricing/CLAUDE.md`** - Pricing package with billing modes, domain types, and validation
- **`sdk/go/testing/CLAUDE.md`** - Testing framework with harness, mocks, conformance, and benchmarks

### Specialized Content Areas

**Agent Configuration (`.claude/agents/`)**:

- Custom agent configurations for FinFocus ecosystem development
- Specialized prompts for technical writing, product management, and senior engineering
- Agent invocation patterns and result expectations

**Schema and Validation (`schemas/`)**:

- JSON Schema architecture with 44+ billing modes and advanced features
- Cross-provider validation patterns and schema evolution strategies
- AJV integration and multi-language validation approaches

**Examples and Documentation (`examples/`)**:

- Cross-provider billing model matrix with AWS, Azure, GCP, and Kubernetes examples
- Metadata patterns, resource tags, and plugin-specific configuration
- Validation integration with CI/CD pipeline and quality standards

**Go SDK Development (`sdk/go/`)**:

- Three-package architecture: `pricing/`, `proto/`, and `testing/`
- Domain type systems with comprehensive billing mode enumerations
- Testing framework architecture with harness, mocks, and conformance levels

### Usage Patterns

**Context-Aware Development**:

Use `/init` commands in specific directories to access specialized guidance:

```bash
cd sdk/go/pricing && /init     # Domain types and billing validation
cd sdk/go/testing && /init     # Testing framework and conformance
cd examples/specs && /init     # PricingSpec examples and patterns
cd schemas && /init            # JSON Schema validation
```

**Inheritance + Specialization**:

- Each directory CLAUDE.md inherits common patterns from root
- Specialized content focuses on directory-specific architecture and workflows
- Build commands and development patterns remain consistent across directories

### Directory-Specific Benefits

- **Focused Context**: Relevant architecture patterns and command references
- **Specialized Workflows**: Directory-appropriate development and testing approaches
- **Tool Detection**: Context-aware build commands and validation approaches
- **Knowledge Preservation**: Captures domain-specific best practices and solutions

### Common Issues & Solutions (Updated)

- Issue: `make lint` and `make validate` may time out on this project.
  Solution: Run `golangci-lint run` directly for faster Go linting results, or `make test` for unit tests.

### Workflow Optimizations (Updated)

- For CodeRabbit fixes: Always verify `git log` and file content first; reviews may reference older
  commits that have already been fixed by subsequent pushes.

### Project-Specific Patterns (Updated)

- `pluginsdk.Serve`: Tests dealing with `Serve` should prefer injecting a `net.Listener` (via
  `ServeConfig.Listener`) rather than relying on `Port` and `listenOnLoopback` to avoid race
  conditions and ensure predictable port binding.

### Usage Profile SDK Pattern (042-usage-profile-context)

The `pluginsdk` package provides helpers for the `UsageProfile` enum (DEV, PROD, BURST):

- **Zero-allocation validation**: `IsValidUsageProfile()` using package-level slice (follows registry pattern)
- **Forward compatibility**: `NormalizeUsageProfile()` treats unknown values as UNSPECIFIED + logs warning
- **Default hours**: `DefaultMonthlyHours(profile)` returns 730/160/200 by profile
- **Builder integration**: `WithProfileDefaults(profile)` on `FocusRecordBuilder`
- **TypeScript parity**: `sdk/typescript/packages/client/src/utils/usage-profile.ts` mirrors Go helpers

Key files:

- `sdk/go/pluginsdk/usage_profile.go` - Go SDK helpers
- `sdk/go/testing/usage_profile_conformance_test.go` - Conformance tests
- `sdk/typescript/packages/client/src/utils/usage-profile.ts` - TypeScript helpers

Pattern for subtests sharing a gRPC harness: Do NOT use `t.Parallel()` on subtests that
share a `TestHarness` — the deferred `harness.Stop()` can close the connection before
parallel subtests complete.

## Active Technologies

- Go 1.25.7 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (045-caching-hint-expires-at)
- N/A (stateless proto field addition, no persistence) (045-caching-hint-expires-at)

- Go 1.25.7 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (044-actual-cost-pagination)
- N/A (stateless pagination with offset-based tokens) (044-actual-cost-pagination)

- Markdown documentation (no code changes) + N/A (documentation only) (043-docs-drift-audit)

- Go 1.25.7 (per go.mod) + Protocol Buffers v3, buf v1.32.1, google.golang.org/protobuf,
  google.golang.org/grpc, zerolog (042-usage-profile-context)
- N/A (stateless proto definitions and SDK helpers) (042-usage-profile-context)

- Go 1.25.5 (per go.mod) + Standard library only (`time`, `encoding/json`) (034-validation-bypass-metadata)
- N/A (stateless struct extension; retention is caller's responsibility) (034-validation-bypass-metadata)

- Go 1.25.5 (per go.mod), Protocol Buffers v3, buf v1.32.1 (040-anomaly-detection-recommendations)
- N/A (stateless enum additions) (040-anomaly-detection-recommendations)

- Go 1.25.5 (per go.mod) + net/http (stdlib), strings (stdlib) - no new dependencies (033-cors-headers-config)
- N/A (stateless configuration extension) (033-cors-headers-config)

- Go 1.25.5 (per go.mod) + Protocol Buffers v3 (032-plugin-dry-run)
- N/A (stateless RPC introspection, no data persistence) (032-plugin-dry-run)
- Go 1.25.5 (per go.mod) + encoding/json (stdlib), crypto/sha256 (stdlib), no external JSON-LD (032-jsonld-serialization)
- N/A (stateless serialization library) (032-jsonld-serialization)

- Go 1.25.5 (per go.mod) + markdown documentation + pluginsdk package, testing package, zerolog (logging examples) (031-sdk-docs-consolidation)
- N/A (pure documentation) (031-sdk-docs-consolidation)
- Go 1.25.5 (per go.mod) + google.golang.org/protobuf, google.golang.org/grpc (029-plugin-info-rpc)
- N/A (stateless proto definitions) (029-plugin-info-rpc)
- N/A (stateless proto definitions) (028-resource-id)
- Go 1.25.5 (per go.mod) + google.golang.org/protobuf, google.golang.org/grpc (existing) (027-finops-validation)
- N/A (stateless validation functions) (027-finops-validation)

- Go 1.25.5 (per go.mod) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1 (026-focus-1-3-migration)
- N/A (stateless proto definitions and SDK) (026-focus-1-3-migration)

- Go 1.25.5 (per go.mod) + Protocol Buffers v3 + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1 (019-target-resources)
- Go 1.25.5 (per go.mod) + zerolog v1.34.0+ (already in go.mod), stdlib only for file operations (015-log-file)
- File system (log file) - append mode with 0644 permissions (015-log-file)
- Go 1.25.5 (per go.mod) + zerolog (logging), google.golang.org/grpc (001-pluginsdk-serve-docs)
- Go 1.25.5 (as per go.mod) + Go stdlib only (`os`, `strconv`, `strings`) (013-pluginsdk-env)
- N/A (reads environment variables at runtime) (013-pluginsdk-env)
- Go 1.25.5 (as per go.mod) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1 (013-recommendations-rpc)
- N/A (stateless RPC, recommendations fetched from backend services) (013-recommendations-rpc)
- Go 1.25.5 (as per go.mod) + google.golang.org/grpc, prometheus/client_golang (new) (014-plugin-metrics)
- N/A (in-memory metrics only) (014-plugin-metrics)
- Go 1.25.5 (as per go.mod) + None (stdlib only - no external dependencies required) (013-iso4217-currency)
- N/A (static in-memory data structures) (013-iso4217-currency)
- Go 1.25.5 + `sdk/go/testing` (conformance suite), `sdk/go/pluginsdk` (target package) (012-pluginsdk-conformance)
- N/A (testing utilities only) (012-pluginsdk-conformance)
- sdk/go/testing harness (007-zerolog-logging-example)
- N/A (example code, no data persistence) (007-zerolog-logging-example)
- JSON Schema draft 2020-12 + AJV (validation)(004-plugin-registry-schema)
- Go 1.25.5 (per go.mod) + Go stdlib only (`strings`) (016-pluginsdk-mapping)
- N/A (stateless helper functions, no data persistence) (016-pluginsdk-mapping)

## PulumiCost to FinFocus Migration Documentation

A comprehensive migration guide is available in [MIGRATION.md](./MIGRATION.md) for users migrating from PulumiCost to FinFocus.

**Key Documentation Files**:

- **MIGRATION.md** - Human-readable migration guide with step-by-step instructions, backwards compatibility
  details, rollback procedures, and deprecation timeline
- **llm-migration.json** - Machine-readable migration manifest for automated tooling and AI assistants
- **schemas/migration.schema.json** - JSON Schema defining the migration manifest structure

- **Environment Variables**:
  - **Port**: `--port` flag > `FINFOCUS_PLUGIN_PORT` > `PULUMICOST_PLUGIN_PORT` (deprecated) > ephemeral (0)
  - **Log Level**: `FINFOCUS_LOG_LEVEL` > `PULUMICOST_LOG_LEVEL` (deprecated) > `LOG_LEVEL` (deprecated)
  - **Other Vars**: `FINFOCUS_*` > `PULUMICOST_*` (deprecated)
  - **Deprecation**: Using legacy `PULUMICOST_*` or `LOG_LEVEL` variables triggers
    a warning log. Support will be removed in v1.0.

See [sdk/go/CLAUDE.md](./sdk/go/CLAUDE.md) for detailed environment variable documentation.

## Recent Changes

- 045-caching-hint-expires-at: Added Go 1.25.7 (per go.mod) + Protocol Buffers v3,
  TypeScript (SDK) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1

- 044-actual-cost-pagination: Added Go 1.25.6 (per go.mod) + Protocol Buffers v3,
  TypeScript (SDK) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1

- 043-docs-drift-audit: Added Markdown documentation (no code changes) + N/A (documentation only)

  google.golang.org/protobuf, google.golang.org/grpc, zerolog

  - `RecommendationActionType_RECOMMENDATION_ACTION_TYPE_INVESTIGATE = 12`
  - Enables cost anomaly recommendations alongside optimization recommendations
  - Test coverage: Comprehensive conformance tests in `sdk/go/testing/anomaly_conformance_test.go`
  - MockPlugin updated to generate realistic anomalies with INVESTIGATE action and negative savings
