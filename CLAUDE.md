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

func (p *MyPlugin) HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
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
When converting capability enums to legacy metadata manually, prefer
`CapabilitiesToLegacyMetadataWithWarnings`; `CapabilitiesToLegacyMetadata` is deprecated.

**GetPluginInfo Configuration Patterns:**

```go
// Pattern 1: Static configuration via ServeConfig (recommended for most plugins)
info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0",
    pluginsdk.WithProviders("aws", "azure"),
    pluginsdk.WithMetadata("build_date", "2024-01-15"),
)
os.Exit(pluginsdk.Run(pluginsdk.ServeConfig{
    Plugin:     &MyPlugin{},
    PluginInfo: info,
}))

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
- `focus14_conformance_test.go` - FOCUS 1.4 Cost and Usage columns, provider rule, and field-list parity
- `README.md` - Complete testing guide for plugin developers

**FOCUS 1.3 Support (`sdk/go/pluginsdk/`)**

The pluginsdk implements FOCUS 1.3 FinOps specification extensions:

- **New Columns (8 fields)**: AllocatedMethodId, AllocatedMethodDetails, AllocatedResourceId,
  AllocatedResourceName, AllocatedTags, ServiceProviderName, HostProviderName, ContractApplied
- **ContractCommitment Dataset**: Supplemental dataset for tracking contractual obligations
- **Deprecated Fields**: `provider_name` → `service_provider_name`, `publisher` → `host_provider_name`
- **FOCUS 1.4 Cost and Usage**: InvoiceDetailId (`WithInvoiceDetailID`) and
  CommitmentProgramEligibilityDetails (`WithCommitmentProgramEligibilityDetails`); ProviderName and
  PublisherName are removed in 1.4, so validation accepts `service_provider_name` alone

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

- `FocusFieldNames()` - Returns all ~68 FOCUS 1.2-1.4 field names
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

## Spec Kit

Feature specs under `specs/` are driven by GitHub Spec Kit, pinned in `mise.toml` as
`pipx:specify-cli`. Since 1.0 it installs Claude **skills** (`.claude/skills/speckit-*`,
invoked as `/speckit-plan` etc.), not the old `.claude/commands/speckit.*.md` slash commands.

```bash
mise exec pipx:specify-cli -- specify check   # binary is `specify`, not the tool ID
# Refresh templates/scripts/skills after bumping the pin (keeps constitution.md):
mise exec pipx:specify-cli -- specify init --here --force --non-interactive --integration claude --script sh
```

- `init` overwrites but never deletes; remove superseded files by hand and review `git diff`
- Extensions (`specify extension add <id>`) live in `.specify/extensions/` and add skills such as
  `/speckit-bug-assess` → `/speckit-bug-fix` → `/speckit-bug-test` (reports in `.specify/bugs/<slug>/`)
  and the pre-spec idea gate `/speckit-assess-intake` → `-research` → `-define` → `-shape` → `-decide`
  (`.specify/assessments/<slug>/`; a "go" hands off to `/speckit-specify`)
- Vendored spec-kit markdown (`.claude/skills/speckit-*`, `.specify/templates/`, `.specify/extensions/`)
  is excluded in both `.markdownlintignore` and `.markdownlint-cli2.jsonc`; extend both when adding more
- CI jobs pass explicit `install_args` to `jdx/mise-action` so specify-cli is never installed in CI;
  keep that list in sync when adding tools CI actually needs

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

## Pre-Commit Requirements

**MANDATORY**: Before committing any code changes, ALWAYS run:

1. `golangci-lint run ./...` — Verify no lint errors (or `make lint` for full linting)
2. `make test` — Verify all tests pass

Do NOT commit code without running both checks. Fix any issues before committing.

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
- `pluginsdk.Run`: Plugin binary entry point (ax-go CLI). Handshake (`--port` / `serve` / no args)
  calls `Serve()` outside `ax.Execute` so stdout stays `PORT=<n>`. `ServeConfig.Logger` remains
  `*zerolog.Logger`; do not switch it to `ax.Logger`. `ParsePortFlag()` is legacy-only.

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

### Usage Source SDK Pattern (051-usage-source-getstats)

- `UsageSourceProvider` (one method, `GetStats`) is optional. `Serve` registers `UsageSourceService`
  through unexported adapters (`usage_source.go`) only when the plugin implements it, in gRPC and
  Connect mode, and adds it to the Connect health checker. Plugins embed `*BasePlugin` for the
  required cost methods.
- Inference always adds the 4 base pricing capabilities, so usage-only plugins must use
  `WithCapabilities(PLUGIN_CAPABILITY_USAGE_STATS)`; `Serve` warns otherwise (skipped for
  `PluginInfoProvider` plugins).
- Adding a capability enum value: bump `maxValidCapability`, `optionalCapabilities`,
  `legacyCapabilityNames`, and the `IsValidCapability` bounds test (`TestLegacyCapabilityMapCompleteness`
  fails until the legacy name exists).
- `sdk/go/testing` cannot import `pluginsdk` (import cycle), so `testing/usage_source.go` keeps a
  private copy of the subject keys; `pluginsdk/subjects_test.go` guards drift via `KnownSubjectKeys()`.
- `toConnectError` (`connect_errors.go`): connect-go reports any non-`*connect.Error` as `Unknown`,
  so every Connect handler must convert gRPC `status` errors with it. Both the usage adapter and
  `ConnectHandler` (all cost RPCs) do; new Connect handler methods must too.
- `ConnectHandler` implements every `CostSourceService` RPC, including `DryRun` and
  `ResolveResourceTypes`; a new RPC needs its own method there or Connect returns `Unimplemented`.
- `make generate` uses unpinned remote Go plugins; regenerating can reformat doc comments in
  untouched `*.connect.go` files. Restore unrelated generated files with `git checkout`.
- `npm run build` in `sdk/typescript/packages/client` builds cleanly, DTS included
  (`tsconfig.base.json` sets `ignoreDeprecations: "6.0"`).

### TypeScript SDK Workspaces (issue #513)

- `sdk/typescript/package.json` lists workspaces in dependency order (client, middleware,
  framework-plugins) because `--workspaces` runs in list order, not topologically. A new package
  goes after the packages it imports.
- Workspace packages resolve siblings through `node_modules` links to their built `dist/`, so
  build before testing. Do not add tsconfig `references`: the packages are not `composite` (TS6306).
- Depend on the scoped `@rshade/finfocus-client`, never the unscoped name, which would resolve
  from the public registry without the lockfile.
- TypeScript 6 defaults `types` to `[]`; Node packages need `"types": ["node"]`.
- tsup externalizes only `dependencies`/`peerDependencies`. Framework adapters declare express,
  fastify, and NestJS as optional `peerDependencies`, or tsup tries to bundle them.
- `RESTGateway` speaks proto3 JSON (`fromJson`/`toJson` via service descriptors). Never
  `JSON.stringify` a protobuf-es v2 message: int64 and `Timestamp.seconds` are `bigint`.
- CI (`typescript-sdk` job) runs build, test, and `lint` (`tsc --noEmit`) for every workspace.
  middleware and framework-plugins type-check `test/`; the client `tsconfig.json` excludes it.
- `mise.toml`'s `node` pin must bundle an npm that satisfies root `engines.npm`; nothing upgrades npm
  separately. Check `https://nodejs.org/dist/index.json` for the bundled npm before bumping either.
  Renovate does not manage `mise.toml`, so a Renovate `engines.node` bump needs a manual pin bump;
  `engines.npm` bumps are disabled in `renovate.json`.

### Allocator SDK Pattern (052-allocator-allocate)

- `AllocatorProvider` (one method, `Allocate`) joins `Serve` like `UsageSourceProvider`: `Serve` builds an
  unexported `optionalServices{usage, allocator}` once and passes it to `serveGRPC`/`serveConnect`; add the
  next optional service as a field there, not as another parameter.
- Allocation rules (`ResolveCurrency`, `ValidateAllocateRequest`, `ValidateAllocateResponse`,
  `CheckConservation`) live in `sdk/go/testing/allocation.go`; `pluginsdk/allocator.go` only delegates, so
  hosts get one rule from production code. `DecodePolicy` is native to `pluginsdk/policy.go`.
- Invalid-input errors are plain errors whose type implements `GRPCStatus()` (`InvalidArgument`), so
  `Error()` has no `rpc error:` prefix and both transports report the same code. Do not use
  `status.Error` for them, and do not add `GRPCStatus` to the existing `ContractError`.
- `sdk/go/internal/refalloc` (reference allocator) imports `pluginsdk`, so only external test packages
  (`pluginsdk_test`, `testing_test`) may import it.
- `RunAllocatorConformance` is policy-agnostic: bad policies are derived from the allocator's own effective
  policy. Only the top level is probed for unknown keys: nested objects may be map fields, which accept
  any key, and the effective JSON cannot tell a map from a struct. `testing/export_test.go` exposes
  `runAllocatorScenarios` so broken-allocator tests assert specific scenario failures.
- `make generate` also regenerates the TypeScript bindings; no separate TS buf run was needed.

### Cost Breakdown Pattern (053-projected-cost-breakdown)

- `GetProjectedCostResponse.cost_breakdown` (field 15) sum rule: `|sum - cost_per_month| <=
  max(costBreakdownAbsTolerance 0.01, costBreakdownRelTolerance 0.001 * cost_per_month)`, a hard error.
  At a zero total the 0.01 floor still applies. Entries (key, then value) are checked before the sum, so
  NaN reports `ErrCostBreakdownInvalidValue`, not a mismatch.
- `WithProjectedCostBreakdown` copies the map and does NOT validate (the sum depends on another
  option's `cost_per_month`); call `ValidateGetProjectedCostResponse`.
- `ValidateGetProjectedCostResponse` guards the breakdown call with `len() > 0` at the call site: on the
  ~6 ns `_Valid` benchmark a non-inlined call alone costs >10%. Compare benchmarks A/B against a `main`
  worktree with prebuilt test binaries; sequential runs on a loaded box drift 20%+.
- `MockPlugin.ProjectedCostBreakdown` values are weights scaled to the mock's `cost_per_month`, so mock
  responses always validate. Negative, NaN/Inf, or zero-sum weights return `codes.FailedPrecondition`
  rather than an invalid breakdown.
- The conformance suite's `plugintesting.ValidateProjectedCostResponse` does not check the breakdown
  (import cycle, same as #427 metadata); only the `pluginsdk` validator does.
- The TS client `tsconfig.json` excludes `test/`, so `tsc --noEmit` cannot show a failing TS test;
  use `npx vitest run`.
- `TestUsageSourceNotRegistered/connect` (`usage_source_test.go:79`, "server did not shut down in
  time") flakes under full `make test` load (about 1 in 8 runs); it passes in isolation.

### Price Options Pattern (557-price-options)

- `PriceOption` (fields 1-7) is listed by `GetProjectedCostResponse.price_options` (16) and
  `EstimateCostResponse.price_options` (6). The list is advisory: never summed into or compared
  to `cost_per_month`/`cost_monthly` or `cost_breakdown`. Fields 17 and 7 are held for a per-region
  price list by comment only; a `reserved` statement would need a `buf breaking` exception to undo.
- Validators reject a nil entry (`ErrPriceOptionNil`), NaN/Inf/negative `unit_price`,
  `monthly_cost`, `upfront_cost`, and NaN/Inf `savings_fraction` (`ErrPriceOptionInvalidValue`),
  plus options on a dry-run projected response (`ErrPriceOptionsWithDryRun`). Errors name
  `price_options[i].<field>`. `savings_fraction` is never recomputed (Principle III).
- Keep `fmt.Errorf` wraps out of the public validators: an inline wrap behind a never-taken
  `len() > 0` guard grew `ValidateGetProjectedCostResponse`'s frame from 136 to 152 bytes and
  slowed `_Valid` by 33%. The wrap lives in `validateProjectedPriceOptions` /
  `validateEstimatePriceOptions`; check frame sizes with `go build -gcflags=-S` (`TEXT ... $N-8`).
- `WithProjectedCostPriceOptions` / `WithEstimatePriceOptions` deep-copy with `proto.CloneOf` and
  keep nil entries. `MockPlugin.ProjectedCostPriceOptions` / `EstimateCostPriceOptions` return
  deep copies as configured (not validated), never on dry-run. `plugintesting.Validate*` is
  unchanged (import cycle, as in 053).
- A test-first gate that writes every story's tests before the proto change makes stories share
  compiled test packages: no story's tests run until all stories' symbols exist.
- `sdk/typescript/packages/client` tests need `npm ci` in `sdk/typescript` after a dependency bump
  (for example msw 3.0.1); a stale `node_modules` fails with `Cannot find package 'msw/node'`.

### FOCUS 1.4 Cost and Usage Pattern (055-focus-14-cost-usage-columns)

- `FocusCostRecord` fields 67 (`invoice_detail_id`) and 68 (`commitment_program_eligibility_details`)
  are FOCUS 1.4; 69-80 are reserved by comment for later cost-row columns. Fields 1 and 55 stay
  (deprecated) until a v2 package; `invoice_issuer` (40) carries FOCUS 1.4 `InvoiceIssuerName`.
- The mandatory provider rule accepts `service_provider_name` or `provider_name`; when both are empty
  the error keeps `FieldName "provider_name"` so hosts matching on it keep working.
- `validateFocus14Rules` runs after the 1.3 rules. The eligibility JSON check is `json.Valid([]byte(s))`,
  which costs 1 alloc/op only when that field is set; records without it stay 0 allocs/op. A zero-copy
  `unsafe` view was rejected in review to keep `unsafe` out of the SDK.
- `testing/focus14_conformance_test.go` asserts `FocusFieldNames()` equals the `FocusCostRecord`
  descriptor's fields and the mock's default dry-run mappings; a new proto field fails it until both
  `dry_run.go` and `mock_plugin.go` list it.
- `buf.yaml` checks `proto/finfocus/v1/focus.proto`. `breaking.ignore` lists only the old
  pulumicost paths. In a worktree, `buf breaking --against '.git#branch=main'` can fail to read
  the git object; archive origin/main and run `buf breaking` against that directory.

### FOCUS 1.4 Contract Commitment Pattern (545-focus-14-contract-commitment)

- `ContractCommitment` fields 13-29 are the FOCUS 1.4 columns. Field 30 is the 1.3
  `contract_commitment_description` gap. Applicability stays a JSON string.
- `optional double` is used for discount percentage, upfront percentage, and pricing-currency cost.
  Generated Go has no `Has*` method: presence is a non-nil pointer. Zero is present, nil is null.
  The JSON-LD serializer writes that key when the pointer is set, including zero, and omits a nil pointer.
- `billing_currency` is required for SPEND and may be empty for USAGE. `WithBaselineTerms` fills the
  1.4 columns that do not allow nulls so existing builders can opt into a valid record.
- `ContractCommitmentBuilder.Build` enforces only `ValidateContractCommitmentBase` (the pre-1.4 rules), so
  chains written against v0.6.2 still build. `BuildFocus14`, the mock source, and conformance use
  `ValidateContractCommitment`, which adds the 1.4 rules. Do not make `Build` stricter: that broke callers.
- `ValidateContractCommitment` enforces the per-record 1.4 rules and stays 0 allocs/op on valid input.
  The JSON object scan does not call `encoding/json`. Cross-row lifecycle rules are out of scope.
- `WithContractApplied` is deprecated and still stores a bare ID. `FormatContractApplied` emits the
  FOCUS 1.4 object. Keys are `ContractId` and `ContractCommitmentId`. An element needs a cost, or a
  quantity with a unit; zero is present, and both metrics may be set. Seven
  `IsValidContractCommitment*` helpers are 0 allocs/op.
- Spec number 545 follows the highest `specs/` prefix (`544-supplemental-contract-commitments`), not 057.

### FOCUS 1.4 Billing Period and Invoice Detail Pattern (546-focus-14-billing-invoice)

- `BillingPeriod` (fields 1-6) and `InvoiceDetail` (fields 1-23) live in `focus.proto`. Two enums:
  `FocusBillingPeriodStatus` (OPEN, CLOSED) and `FocusInvoiceIssueStatus` (OPEN, ISSUED, VOIDED).
  Charge category reuses `FocusChargeCategory`. REFUND and UNSPECIFIED fail invoice validation.
- Issue 546 added the messages, builders, and per-record validators only. The RPCs are spec 547.
- `payment_currency_billed_cost` is the only `optional double`. Presence is a non-nil pointer.
  Zero is present (`proto.Float64`); nil is absent. The currency and that cost are all-or-nothing.
  The lineage id is separate: when it is set and the cost is non-zero, it must equal
  `invoice_detail_id`. A zero cost does not require that match.
- Grain is `map<string,string>` (the 026 Tags decision). FOCUS keys are the nine property names
  (`ContractId`, `RegionId`, `ResourceId`, `ResourceType`, `ServiceName`, `SkuId`, `SkuMeter`,
  `SkuPriceId`, `SubAccountId`). Custom keys start with `x_` and are longer than `x_`.
  `extended_columns` uses the same prefix. Values are decimal strings, scanned without `strconv`
  so a valid record stays 0 allocs/op. Empty maps mean null.
- `ValidateBillingPeriod` and `ValidateInvoiceDetail` live in `sdk/go/testing/invoice_focus14.go`.
  Builders delegate. Errors wrap `ErrInvalidBillingPeriod` or `ErrInvalidInvoiceDetail` through
  `invalidArgumentError` and name the column. The message is not prefixed with the sentinel text.
  Mandatory timestamps are compared by seconds and nanos, not `AsTime`.
- Simple setters (`WithInvoiceIssuerName`, `WithBilledCost`, `WithChargeCategory`) and
  `IsValidBillingPeriodStatus` / `IsValidInvoiceIssueStatus` are 0 allocs/op. Timestamp setters
  and map copies allocate. The TypeScript builders clone and do not repeat the Go rules.
- JSON-LD writes `billedCost` even when it is 0. `paymentCurrencyBilledCost` is written when the
  pointer is set, including 0, and omitted when nil. Document ids are private hashes.
  `IDGenerator` is unchanged.
- `buf.yaml` checks `proto/finfocus/v1/focus.proto` (`breaking.ignore` lists only the old
  pulumicost paths). In a worktree, `buf breaking --against '.git#branch=main'` can fail to read
  the git object; `git archive origin/main` into a scratch directory and run `buf breaking`
  against that directory.

### Invoice Dataset RPC Pattern (547-invoice-dataset-rpcs)

- `GetBillingPeriods` and `GetInvoiceDetails` are on the existing `SupplementalDatasetService`.
  One provider, `InvoiceDatasetProvider`, implements both. `PLUGIN_CAPABILITY_INVOICE_DATA = 17`,
  legacy `supports_invoice_data`. `Serve` registers the service when `ContractCommitmentProvider`
  or `InvoiceDatasetProvider` is present and lists it once in the Connect health checker. The RPC
  whose provider is nil returns `codes.Unimplemented` on gRPC and Connect (`toConnectError`), with
  no `rpc error:` prefix. The other RPC still serves.
- Request, page, and token rules are shared with commitments through `validateDatasetRequest` and
  `paginateRecords` in `sdk/go/testing/supplemental.go`. Billing-period overlap uses
  `timestampBefore` / `timestampAfter` (seconds, then nanos). A period that ends exactly at the
  window start does not match. A token past the match count is an empty page; a token that does
  not decode to a non-negative offset is `InvalidArgument`. Uniqueness is
  `(invoice_issuer_name, billing_period_start)` for periods (seconds and nanos; a different end is
  still a duplicate) and `invoice_detail_id` for lines. Response validators stay 0 allocs up to 64
  records. The missing-provider message is `method <RpcName> not implemented`.
  `WithCapabilities` replaces discovery only.
- `MockInvoiceDatasetSource` is a separate type, so `MockPlugin` capabilities do not change.
  `RunInvoiceDatasetConformance` covers both RPCs over bufconn. The TypeScript client methods are
  `getBillingPeriods` / `billingPeriods` and `getInvoiceDetails` / `invoiceDetails`. Iterators
  default only a missing or zero page size, and stop on a repeated page token before yielding
  that page.
- Do not add Correction Handling or Delivery Handling fields. Do not change `GetContractCommitments`
  request or response fields. Do not add a second service.

### Supplemental Dataset Pattern (544-supplemental-contract-commitments)

- `SupplementalDatasetService` (`supplemental.proto`) serves FOCUS supplemental datasets.
  `ContractCommitmentProvider` maps to `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16`
  (`supports_contract_commitments`). Invoice RPCs share that service (see 547). The combined
  adapter returns `Unimplemented` for a provider the plugin lacks, and the service registers when
  either provider exists.
- `optionalServices` now has `registerConnect` and `healthServiceNames` (added to keep
  `serveConnect` under funlen); add the next optional service there, not inline.
- Rules live in `sdk/go/testing/supplemental.go`; `pluginsdk/supplemental.go` delegates, and
  `ContractCommitmentBuilder.Build` calls `ValidateContractCommitment`, which keeps the builder's
  unprefixed messages and adds NaN/Inf rejection. Request/response errors use the 052
  `invalidArgumentError` with a sentinel prefix.
- No "return all" mode: `page_size` 0 means 50, above 1000 clamps, negative is InvalidArgument;
  `total_count` is exact. Window matching is overlap on `[start, end)` using the commitment period,
  falling back to the contract period; unset bounds are open.
- `ValidateGetContractCommitmentsResponse` is 0 allocs up to 64 records (pairwise duplicate check)
  and uses a map above; `TestContractCommitmentValidatorsAllocationFree` guards it.
- The reference producer is `MockContractCommitmentSource`, not a `MockPlugin` method, so
  `MockPlugin`'s inferred capabilities do not change. No startup warning for commitment providers
  (they are normally also cost sources).
- In an agent worktree, `buf breaking --against '.git#branch=main'` is blocked; `git archive main
  proto buf.yaml` into a scratch dir and run `buf breaking --against <dir>` instead.

### Recommendation Scorer Pattern (556-recommendation-scoring)

- `RecommendationScorerService` (`scoring.proto`) maps to `PLUGIN_CAPABILITY_RECOMMENDATION_SCORING = 18`
  (legacy `supports_recommendation_scoring`). The issue text said 17, but 17 is `INVOICE_DATA`.
- `RecommendationScorerProvider` is a field on `optionalServices` (`scorer`), with adapters in
  `pluginsdk/scorer.go`. Rules live in `sdk/go/testing/scoring.go`; `pluginsdk` delegates.
- `ValidateScoreRecommendationsRequest(req, maxBatchSize)` (0 means no limit) returns InvalidArgument
  errors; `ValidateScoreRecommendationsResponse` checks alignment, ranges, signal support, and that a
  duplicate group has at least two members. The SDK does not cap `max_batch_size`.
- `MockRecommendationScorer` (fixed rules) and `RunScorerConformance` are model-agnostic; they never
  check score values. Scenario funcs are named `scorerCheck*` to avoid clashing with allocator ones.
- Trust rules and threshold guidance (non-normative, synthetic data) are in `docs/recommendation-scoring.md`.
- Score caching (issue 581) is docs-only: the key hashes the pre-`identifier_mode` record without `id`, plus
  `signals`, `ScorerInfo` name/model/calibration, and plugin version. No `valid_for` proto hint yet.

### Scorer Advertised Limits Pattern (591-scorer-advertised-limits)

- A scorer publishes its limit and signals through `GetPluginInfo` metadata keys `scorer_max_batch_size`
  and `scorer_supported_signals` (lowercase names, no `SCORE_SIGNAL_` prefix), set with
  `pluginsdk.WithScorerLimits`. No proto field was added; response fields stay authoritative.
  `PluginInfo.Validate` rejects a malformed pair via `ParseScorerLimits`.
- The oversize-batch error keeps `InvalidArgument` (existing hosts keep working) and gains a
  `google.rpc.ErrorInfo` detail, reason `BATCH_TOO_LARGE`; test it with `IsBatchTooLarge`. `toConnectError`
  now copies status details onto the `connect.Error`.
- Helpers live in `sdk/go/testing/scorer_limits.go` (import cycle); `pluginsdk/scorer.go` delegates.
  `RunScorerConformance` adds `advertised_limits` only for impls with `AdvertisedScorerMetadata()`.
- `sdk/typescript` needs `npm ci` in a fresh worktree before the client tests run.

### Scorer Session Pattern (592-cross-batch-scoring-groups)

- `ScoreRecommendationsRequest.session_id` (4) and the response echo (5) let the batches of one host
  operation share duplicate group ids and pseudonym tokens. Empty means no session and unchanged rules.
  The scorer derives `duplicate_group_id` from `session_id` and its duplicate key and stores nothing.
- With a session the echo must match, and a response may hold a lone group member; a declined session
  (empty echo) keeps the two-member rule. `isValidSessionID` is 1 to 128 printable ASCII.
- Session conformance scenarios (`session_echo`, `session_across_batches`, `session_isolation`) live in
  `sdk/go/testing/scorer_session.go` and pass for a scorer that declines sessions or does not group the pair.
- Cached items left out of every batch are never grouped; this is documented, not a field.

### Actual Cost Billing Account Pattern (585-actual-cost-billing-account-id)

- `GetActualCostRequest.billing_account_id` (field 9) is caller-supplied. Empty means not supplied:
  plugins must not invent one and may leave `focus_record` unset. When set, any attached FOCUS record
  must carry it exactly. Field 10 is held by comment (not `reserved`) for a later billing account name.
- `MockPlugin.GetActualCost` attaches a FOCUS record per result only when the id is set (built as a
  struct literal, since `sdk/go/testing` cannot import `pluginsdk`). The batch path never sets it.
- `plugintesting.ValidateActualCostBillingAccount` and `RPCCorrectness_GetActualCostBillingAccount`
  (Standard level) check only the echo rule; "do not invent" cannot be tested mechanically.
- In a worktree, `make generate` uses mise's buf, so there is no `bin/buf`; run `buf breaking` directly.

### Region Prices Pattern (586-region-prices)

- `RegionPrice` rows on `GetProjectedCostResponse.region_prices` (17) and
  `EstimateCostResponse.region_prices` (7) are advisory: never summed into or compared with the
  primary cost. Fields 16 and 6 hold `price_options` (557, PR 599).
- Row rules live in `sdk/go/testing/region_price.go` (`ValidateRegionPrices`); `pluginsdk` aliases
  the sentinels and calls it, and so do the harness validators, so conformance checks the rows too
  (unlike 053's `cost_breakdown`, which only `pluginsdk` checks).
- The projected validator keeps only `len(rows) > 0` inline; the rest is in
  `validateProjectedRegionPrices`. A/B on the about-6 ns `_Valid` benchmark: the guard costs about
  0.2 ns, and the change to the binary layout (proto field plus new code) shifts it about 0.45 ns even
  with the check compiled out. Build an `if false` variant to separate layout from code cost.

### Scorer Omitted Fields Pattern (593-omitted-fields)

- `ScoreRecommendationsRequest.omitted_fields` (field 5) names `Recommendation` paths the host cleared by
  policy. Field 4 is `session_id`; `omitted_fields` is field 5 on the request only (the response's field 5 is
  the `session_id` echo). Paths are dot-separated
  proto field names (`resource.tags`, `kubernetes.cluster_id`); the `action_detail` oneof name is accepted;
  map and scalar fields end a path. At most 64 unique entries of 1-128 bytes.
- The rule lives in `validateOmittedFields` (`sdk/go/testing/scoring.go`), guarded by `len() > 0` at the
  call site; it resolves paths with `protoreflect`, so a new `Recommendation` field needs no list update.
- Conformance stays model-agnostic: `omitted_fields_accepted` / `omitted_fields_rejected` check structure
  and `InvalidArgument`, never score values. The mock proves the rule in its own tests.

### Allocation Period and Selector Pattern (588-allocate-period-selector)

- `AllocateRequest` gains `start`/`end` (5, 6; same rule as `GetStatsRequest`, Q7) and `selector` (7);
  `AllocateResponse` echoes `start`/`end` (5, 6). `ValidateAllocateResponse` accepts a missing echo, so
  allocators built before the fields stay valid, but rejects a differing one (P8). Conformance
  (`period_echoed`) is where the echo is mandatory.
- A non-empty selector never relaxes an invariant. `refalloc` adds `PartialSelectionWarning`; hosts omit
  or label idle and cluster rows.
- Test allocators that rebuild `AllocateRequest` field by field drop new fields. Clone with
  `proto.CloneOf(req)` and override only what changes (see `mapFieldAllocator`).

### Shared Harness and Duplicate Scan Pattern (590-shared-supplemental-harness)

- All six bufconn harnesses embed the unexported `bufconnHarness[C]` (`testing/bufconn_harness.go`)
  for `Start` and `Stop`. A new harness embeds it and passes a `register` func and the generated
  `pbc.New*Client`. The one `grpc.DialContext` nolint lives in `dial()`, which spec validation reuses.
- Each harness declares its own `Client()`: `go doc` prints a promoted generic method unsubstituted
  (`Client() C`), so a promoted `Client` would hide the concrete client type.
- Supplemental duplicate checks call `findDuplicate(list, key)` (`supplemental.go`): pairwise up to
  `pairwiseDuplicateLimit` (64), map above; both report the lowest repeat and its earliest match. Pass a
  method expression or top-level func as `key`; a closure can allocate and break the 0-alloc tests.

### Multi-Call Scorer Pattern (589-scorer-multi-call)

- `ScorerInfo.provider_request_ids` (5) and `models` (6) are new; `provider_request_id` (4) is deprecated
  and still set by the mock (with a targeted `//nolint:staticcheck`). `model` must equal `models[0]`, and
  `models` joins the score cache key; request ids never do.
- Scorers must not set `ResourceError.resource_type_unsupported` on per-item errors. The
  `unscorable_item` conformance scenario is the only one that elicits a per-item error, so it is what
  catches that flag.

### Plugin Manifest Pattern (595-manifest-writer-validator)

- `SaveManifest` writes the canonical form (`manifest_codec.go`): protojson `UseProtoNames`, enums shortened
  by the descriptor's zero-value prefix (`INSTALLATION_METHOD_BINARY` → `binary`), re-encoded with
  `encoding/json` for sorted keys and stable bytes. protojson randomizes whitespace on purpose, so never
  ship its raw output as a file format.
- YAML is built from the canonical JSON as a `yaml.Node` tree. Clearing `Style` on a `!!str` node drops
  its quotes, so `blockStyle` keeps quotes whenever `yaml.Marshal` of the value would quote it
  (`"1.0"`, `"no"`, timestamps). Golden files in `pluginsdk/testdata/` pin the bytes (`-update`).
- `LoadManifest` normalizes keys and enums against the message descriptor, so it reads the canonical
  form, protojson camelCase, and yaml.v3's lowercased Go field names (`specversion`). It does not validate.
- `registry` derives `AllServiceMethods` and the protocol half of `AllPluginCapabilities` from descriptors
  at init. `TestSchemaDrift` compares them, `AllManifestBillingModes`, and `MaxResourceTypeLength` (256)
  with the schemas; adding a `CostSourceService` RPC or `PluginCapability` value fails it until the
  manifest schema enum gains the value.
- `IsValidPluginCapability` scans `pluginCapabilitiesByLength` (values grouped by length at init), not all 32
  values: a flat scan doubled the miss cost (3.2 to 6.2 ns) and tripped the CI benchmark alert; grouped it is
  2.4 ns, 0 allocs. The same CI alert also flags untouched packages (currency) at about 2x; A/B against main
  with prebuilt binaries before believing it.
- The registry index schema's `capabilities` enum is a superset (it keeps registry-only values such as
  `tagging`); the drift test checks containment there, equality for the manifest schema.
- `TestExampleManifests` runs all six `examples/plugins/` manifests through the schema and the SDK validator.
  A container image `download_url` needs a scheme (`oci://registry/...`) for the schema's `uri` format.
  Per-resource sustainability metrics are a `SupportsResponse.supported_metrics` concern, not a manifest field;
  a manifest declares `carbon`, `energy`, and `water` capabilities instead.

### Resource Attributes Pattern (596-resource-descriptor-attributes)

- `ResourceDescriptor.attributes` (12) carries declared inputs unflattened on every descriptor RPC; plugins
  prefer it and fall back to `tags`. Host redaction is a contract rule only: the SDK never logs descriptors
  and cannot detect what a host dropped.
- `MaxAttributesBytes` (65536, `proto.Size`) and `MaxTagValueLength` (2048 bytes) are defined in both
  `pluginsdk/batch.go` and `testing/contract.go`; change them together. `TestDescriptorLimitValues` and
  `TestContractDescriptorLimitValues` pin the literals.
- The size check is nil-guarded, so descriptors without attributes stay at 0 allocs/op; with attributes
  `proto.Size` costs one 16 B alloc (map iteration). Batch-vs-transport size is documented, not enforced:
  grpc-go and `payloadLimitMiddleware` reject an oversized request before any validator runs.
- Varint length prefixes make some exact `proto.Size` values unreachable by padding one string; the test
  helpers search a pad range for the exact size.
- `AttributeValue` walks with `strings.Cut` and a digit-only index parser (no `strconv`, so `+0` and
  overflow fail) and holds the current container as a Struct or ListValue pointer, because
  `structpb.NewStructValue` would allocate.
- protobuf-es maps `Struct` to `JsonObject`; the TS builder's `withAttributes` copies with
  `structuredClone` so later caller edits do not leak into built descriptors.

## Active Technologies

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) + google.golang.org/protobuf
  (`proto.Size`, `structpb`), buf v1.32.1; no new dependencies (596-resource-descriptor-attributes)
- N/A (one optional Struct on ResourceDescriptor, two size limits, one read helper)
  (596-resource-descriptor-attributes)

- Go 1.27.1 (per go.mod) + google.golang.org/protobuf (protojson, protoreflect), gopkg.in/yaml.v3,
  santhosh-tekuri/jsonschema/v6 (tests); no new dependencies (595-manifest-writer-validator)
- Files (canonical plugin manifest JSON or YAML) (595-manifest-writer-validator)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) + google.golang.org/protobuf, buf v1.32.1;
  no new dependencies (592-cross-batch-scoring-groups)
- N/A (stateless; group ids derived per call) (592-cross-batch-scoring-groups)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new dependencies (594-allocation-row-provenance)
- N/A (three optional strings on AllocationRow, one validator rule) (594-allocation-row-provenance)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect, buf v1.32.1;
  no new dependencies (588-allocate-period-selector)
- N/A (allocation window and selector on AllocateRequest, echoed window on AllocateResponse)
  (588-allocate-period-selector)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect, buf v1.32.1;
  no new dependencies (589-scorer-multi-call)
- N/A (ScorerInfo list fields and per-item error rules) (589-scorer-multi-call)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf (`proto.CloneOf`), buf v1.32.1; no new dependencies (557-price-options)
- N/A (advisory repeated PriceOption on GetProjectedCostResponse and EstimateCostResponse)
  (557-price-options)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new dependencies
  (586-region-prices)
- N/A (advisory repeated RegionPrice on projected cost and estimate responses) (586-region-prices)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new dependencies
  (585-actual-cost-billing-account-id)
- N/A (one optional string on GetActualCostRequest) (585-actual-cost-billing-account-id)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  buf v1.32.1; no new dependencies (547-invoice-dataset-rpcs)
- N/A (paged RPCs over the existing BillingPeriod and InvoiceDetail messages)
  (547-invoice-dataset-rpcs)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; stdlib only
  (546-focus-14-billing-invoice)
- N/A (stateless BillingPeriod and InvoiceDetail messages, builders, and per-record validation)
  (546-focus-14-billing-invoice)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  buf v1.32.1; no new dependencies (544-supplemental-contract-commitments)
- N/A (stateless paged RPC over the existing ContractCommitment message)
  (544-supplemental-contract-commitments)

- Go 1.27.1 (go.mod) + Existing `google.golang.org/protobuf` (protojson, protocmp in tests). (055-cost-allocation-lineage)
- N/A (wire contract and in-memory builder only) (055-cost-allocation-lineage)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; stdlib encoding/json
  (055-focus-14-cost-usage-columns)
- N/A (stateless FocusCostRecord fields + zero-alloc validation) (055-focus-14-cost-usage-columns)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1 (053-projected-cost-breakdown)
- N/A (stateless map field on GetProjectedCostResponse + zero-alloc validation) (053-projected-cost-breakdown)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  buf v1.32.1; no new Go dependencies (052-allocator-allocate)
- N/A (stateless allocation RPC contract and SDK helpers) (052-allocator-allocate)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  buf v1.32.1 (051-usage-source-getstats)
- N/A (stateless usage-stats RPC; new `usage.proto` + UsageSourceService) (051-usage-source-getstats)

- Go 1.27.1 (per go.mod) + github.com/rshade/ax-go v0.6.0 (Cobra CLI) (494-ax-go-plugin-cli)
- N/A (stateless CLI wrapper around Serve; handshake stdout stays PORT=n) (494-ax-go-plugin-cli)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (050-resolve-resource-types-hardening)
- N/A (stateless in-memory type mappings and metrics counters)
  (050-resolve-resource-types-hardening)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (049-resolve-resource-types)
- N/A (stateless type mappings held in-memory after initialization)
  (049-resolve-resource-types)

- Go 1.25.8 (per go.mod) + `github.com/stretchr/testify`,
  `google.golang.org/protobuf` (existing, unchanged) (048-test-descriptor-helper)
- N/A (test-only refactoring) (048-test-descriptor-helper)

- Go 1.25.8 (per go.mod) + google.golang.org/protobuf, google.golang.org/grpc
  (existing, unchanged) (047-validation-error-integration)

- Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (046-batch-cost-rpc)
- N/A (stateless batch RPC, no data persistence) (046-batch-cost-rpc)

- Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (045-caching-hint-expires-at)
- N/A (stateless proto field addition, no persistence) (045-caching-hint-expires-at)

- Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
  (044-actual-cost-pagination)
- N/A (stateless pagination with offset-based tokens) (044-actual-cost-pagination)

- Markdown documentation (no code changes) + N/A (documentation only) (043-docs-drift-audit)

- Go 1.25.8 (per go.mod) + Protocol Buffers v3, buf v1.32.1, google.golang.org/protobuf,
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

- 596-resource-descriptor-attributes: Added ResourceDescriptor.attributes (field 12, a
  google.protobuf.Struct) with the host redaction rule and the tags fallback, MaxAttributesBytes
  (65536) and ErrAttributesTooLarge in both validators, MaxTagValueLength 256 to 2048 in both,
  pluginsdk.AttributeValue, the TypeScript ResourceDescriptorBuilder.withAttributes, and the Basic
  conformance test RPCCorrectness_GetProjectedCostWithAttributes (issue 617)

- 595-manifest-writer-validator: SaveManifest writes the canonical manifest (snake_case, schema enum
  strings, stable bytes) and LoadManifest reads every earlier form; added MarshalManifestJSON/YAML,
  supported_resources and capabilities validation, descriptor-derived methods and capabilities,
  ManifestCapabilityName, and a 256-character resource type bound (issue 611)

- 594-allocation-row-provenance: Added AllocationRow allocated_method_id (7), allocated_method_details (8),
  allocated_resource_id (9); a method id requires a resource id; field 10 held by comment for a later LineageNode

- 592-cross-batch-scoring-groups: Added scorer `session_id` (request 4, response 5), session-aware
  validators, mock group ids, three session conformance scenarios, and docs for cached items (issue 574)

- 588-allocate-period-selector: Added AllocateRequest start/end/selector and the echoed
  AllocateResponse start/end, the window rule (Q7) and echo rule (P8), the refalloc echo and
  partial-selection warning, and the period_echoed and selector_keeps_invariants scenarios.

- 590-shared-supplemental-harness: The six bufconn harnesses share an unexported generic
  `bufconnHarness`, and the three supplemental duplicate-key checks share `findDuplicate`.
  No exported API, validation rule, or error text changed (issue 561)
- 589-scorer-multi-call: Added ScorerInfo.provider_request_ids (5) and models (6), deprecated
  provider_request_id, validator rules for both lists and resource_type_unsupported, the mock options
  WithScorerModels and WithScorerProviderRequestIDs, and the unscorable_item conformance scenario.

- 557-price-options: Added PriceOption, GetProjectedCostResponse.price_options (16) and
  EstimateCostResponse.price_options (6), three ErrPriceOption* sentinels,
  pluginsdk.WithProjectedCostPriceOptions and WithEstimatePriceOptions, and MockPlugin
  ProjectedCostPriceOptions / EstimateCostPriceOptions; advisory and never summed into the
  selected price; fields 17 and 7 held by comment for a per-region price list (issue 588)
- 586-region-prices: Added RegionPrice and region_prices on GetProjectedCostResponse (17) and
  EstimateCostResponse (7), plugintesting.ValidateRegionPrices with ErrInvalidRegionPrice and
  ErrRegionPricesWithDryRun, pluginsdk region price options, and MockPlugin.RegionPrices.

- 585-actual-cost-billing-account-id: Added GetActualCostRequest.billing_account_id (field 9),
  MockPlugin FOCUS records keyed on it, ValidateActualCostBillingAccount with
  ErrBillingAccountIDMismatch, and RPCCorrectness_GetActualCostBillingAccount. Closes #590.

- 547-invoice-dataset-rpcs: Added SupplementalDatasetService.GetBillingPeriods and
  GetInvoiceDetails, PLUGIN_CAPABILITY_INVOICE_DATA = 17, pluginsdk.InvoiceDatasetProvider,
  shared window and page helpers, MockInvoiceDatasetSource, RunInvoiceDatasetConformance,
  and TypeScript iterators. Closes the FOCUS 1.4 delivery gap on issue 540.

- 546-focus-14-billing-invoice: Added BillingPeriod and InvoiceDetail messages, status enums,
  pluginsdk builders, per-record validators, JSON-LD serializers, and TypeScript builders.
  No invoice RPC.

- 544-supplemental-contract-commitments: Added SupplementalDatasetService.GetContractCommitments
  (supplemental.proto), PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16,
  pluginsdk.ContractCommitmentProvider with validation, window, and pagination helpers,
  MockContractCommitmentSource, RunContractCommitmentConformance, and a TS
  SupplementalDatasetClient

- 055-focus-14-cost-usage-columns: Added FocusCostRecord.invoice_detail_id (67) and
  commitment_program_eligibility_details (68, JSON object string),
  pluginsdk.WithInvoiceDetailID and WithCommitmentProgramEligibilityDetails, two sentinels,
  and TS builder setters; the provider rule now accepts service_provider_name (FOCUS 1.4
  removes ProviderName)

- 055-cost-allocation-lineage: Added Go 1.27.1 (go.mod) +
  `google.golang.org/protobuf` (protojson, protocmp in tests)

- 053-projected-cost-breakdown: Added GetProjectedCostResponse.cost_breakdown
  (map<string,double>, field 15), pluginsdk.WithProjectedCostBreakdown, five
  ErrCostBreakdown* sentinels, and MockPlugin.ProjectedCostBreakdown; validated for
  snake_case keys, non-negative values, and sum within max(0.01, 0.1%) of cost_per_month

- 052-allocator-allocate: Adds AllocatorService.Allocate (allocation.proto),
  PLUGIN_CAPABILITY_ALLOCATION = 15, pluginsdk.AllocatorProvider and DecodePolicy,
  conservation/validation helpers, RunAllocatorConformance, and a TS AllocatorClient

- 051-usage-source-getstats: Added UsageSourceService.GetStats (usage.proto),
  PLUGIN_CAPABILITY_USAGE_STATS = 14, pluginsdk.UsageSourceProvider, and a TS
  UsageSourceClient

- 494-ax-go-plugin-cli: Added pluginsdk.Run() ax-go CLI (handshake-safe Serve, dry-run subcommand)

- 050-resolve-resource-types-hardening: Added Go 1.27.1 (per go.mod) + Protocol
  Buffers v3, TypeScript (SDK) + google.golang.org/protobuf,
  google.golang.org/grpc, buf v1.32.1

- 049-resolve-resource-types: Added Go 1.27.1 (per go.mod) + Protocol Buffers v3,
  TypeScript (SDK) + google.golang.org/protobuf, google.golang.org/grpc,
  buf v1.32.1

- 048-test-descriptor-helper: Added Go 1.25.8 (per go.mod) +
  `github.com/stretchr/testify`, `google.golang.org/protobuf` (existing, unchanged)

- 047-validation-error-integration: Added Go 1.25.8 (per go.mod) +
  google.golang.org/protobuf, google.golang.org/grpc (existing, unchanged)

- 046-batch-cost-rpc: Added Go 1.25.8 (per go.mod) + Protocol Buffers v3,
  TypeScript (SDK) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1

- 045-caching-hint-expires-at: Added Go 1.25.8 (per go.mod) + Protocol Buffers v3,
  TypeScript (SDK) + google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
