# FinFocus Documentation Index

This directory contains comprehensive documentation for the FinFocus specification
and SDK. Use this guide to navigate to the appropriate documentation for your needs.

## Quick Links

| Document | Description |
|----------|-------------|
| [Main README](../README.md) | Project overview, quick start, and core concepts |
| [Plugin Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md) | Complete guide to building cost source plugins |
| [Observability Guide](../OBSERVABILITY_GUIDE.md) | Telemetry, health checks, and metrics implementation |
| [SDK Documentation](../sdk/go/pluginsdk/README.md) | Go SDK reference and API documentation |

## Documentation by Topic

### Getting Started

- **[Main README](../README.md)** - Project overview, installation, and quick start guide
- **[Plugin Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md)** - Step-by-step plugin development tutorial
- **[SDK README](../sdk/go/pluginsdk/README.md)** - SDK overview and usage patterns

### Plugin Development

- **[Plugin Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md)** - Complete plugin implementation guide
  - gRPC service interface (8 RPC methods)
  - Request/response message formats
  - Error handling and validation
  - Packaging and distribution
  - Testing and validation

- **[Plugin Startup Protocol](PLUGIN_STARTUP_PROTOCOL.md)** - Plugin lifecycle and startup sequence
  - Port resolution and binding
  - Environment variable handling
  - Health check implementation
  - Graceful shutdown

- **[Plugin Migration Guide](PLUGIN_MIGRATION_GUIDE.md)** - Breaking change migrations
  - Version upgrade paths
  - Backwards compatibility
  - Deprecation timelines

### FOCUS 1.2 Compliance

- **[FOCUS Columns Reference](focus-columns.md)** - FinOps FOCUS 1.2 column mapping
  - Required column definitions
  - Data type specifications
  - Mapping from cloud providers

### SDK Reference

- **[SDK Documentation](../sdk/go/pluginsdk/README.md)** - Go SDK comprehensive guide
  - `Serve()` function and ServeConfig
  - Environment variable handling
  - Structured logging with zerolog
  - Prometheus metrics integration
  - FOCUS 1.2 cost record builder
  - Response validation helpers

- **[Property Mapping](PROPERTY_MAPPING.md)** - Cloud provider property extraction
  - AWS property mapping (ARN, region, SKU)
  - Azure property mapping
  - GCP property mapping

- **[Testing Framework](../sdk/go/testing/README.md)** - Plugin testing reference
  - Test harness with bufconn
  - Mock plugin implementation
  - Conformance testing (Basic/Standard/Advanced)
  - Performance benchmarks

### Observability

- **[Observability Guide](../OBSERVABILITY_GUIDE.md)** - Comprehensive observability implementation
  - Health check endpoints
  - Metrics collection
  - Service Level Indicators (SLIs)
  - Distributed tracing with OpenTelemetry
  - Structured logging
  - Dashboard templates

### Plugin Registry

- **[Plugin Registry Specification](plugin-registry-specification.md)** - Registry design and implementation
  - Plugin discovery and versioning
  - Security and validation
  - Registration workflow

### Examples

- **[Examples Directory](../examples/README.md)** - Reference implementations
  - PricingSpec JSON examples (10 cross-provider examples)
  - gRPC request samples
  - Billing model demonstrations

## SDK Package Reference

The Go SDK consists of these packages:

| Package | Description | Documentation |
|---------|-------------|---------------|
| `sdk/go/proto` | Generated protobuf code | [Go Reference](https://pkg.go.dev/github.com/rshade/finfocus-spec/sdk/go/proto) |
| `sdk/go/pluginsdk` | Plugin development SDK | [README](../sdk/go/pluginsdk/README.md) |
| `sdk/go/pluginsdk/mapping` | Property extraction helpers | [Property Mapping](PROPERTY_MAPPING.md) |
| `sdk/go/pricing` | Domain types and validation | [SDK CLAUDE.md](../sdk/go/CLAUDE.md) |
| `sdk/go/currency` | ISO 4217 currency validation | [SDK CLAUDE.md](../sdk/go/CLAUDE.md) |
| `sdk/go/registry` | Plugin registry domain types | [SDK CLAUDE.md](../sdk/go/CLAUDE.md) |
| `sdk/go/testing` | Testing framework | [README](../sdk/go/testing/README.md) |

## gRPC Service Reference

The `CostSourceService` provides 8 RPC methods:

| RPC Method | Description | Documentation |
|------------|-------------|---------------|
| `Name` | Plugin identification | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#name-rpc) |
| `Supports` | Resource support check | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#supports-rpc) |
| `GetActualCost` | Historical cost data (FOCUS 1.2) | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#getactualcost-rpc) |
| `GetProjectedCost` | Cost projections | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#getprojectedcost-rpc) |
| `GetPricingSpec` | Pricing specifications | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#getpricingspec-rpc) |
| `EstimateCost` | Pre-deployment cost estimation | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#estimatecost-rpc) |
| `GetRecommendations` | Cost optimization advice | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#getrecommendations-rpc) |
| `GetBudgets` | Budget tracking and alerts | [Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#getbudgets-rpc) |

## Contributing to Documentation

When adding or updating documentation:

1. Follow the existing structure and formatting
2. Use proper markdown syntax
3. Include code examples where appropriate
4. Run `make lint-markdown` to validate formatting
5. Update this index when adding new documents

## Version Information

- **Specification Version**: v0.4.7
- **Go SDK**: go 1.27.1
- **Node.js**: v24.15.0 (see mise.toml)
- **FOCUS Version**: 1.2
