# FinFocus Specification v0.5.4

## Focusing your finances left

FinFocus (formerly PulumiCost) provides a comprehensive, standardized protocol for cost source
plugins to integrate with cost management platforms. (See [Migration Guide](MIGRATION.md) for upgrade instructions).
This specification enables consistent cost data retrieval across AWS, Azure, GCP, Kubernetes, and custom providers.

[![CI](https://github.com/rshade/finfocus-spec/actions/workflows/ci.yml/badge.svg)](https://github.com/rshade/finfocus-spec/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/rshade/finfocus-spec/branch/main/graph/badge.svg)](https://codecov.io/gh/rshade/finfocus-spec)
[![Go Reference](https://pkg.go.dev/badge/github.com/rshade/finfocus-spec/sdk/go.svg)](https://pkg.go.dev/github.com/rshade/finfocus-spec/sdk/go)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![FOCUS 1.2](https://img.shields.io/badge/FinOps-FOCUS_1.2-green)](https://focus.finops.org)

## Overview

FinFocus Specification v0.5.4 is a complete, enterprise-ready protocol for
standardizing cloud cost data retrieval. It provides:

### Core Features

- **Universal Protocol**: Standardized gRPC interface for cost plugins
- **FinOps FOCUS 1.2**: Aligned with industry standard cost data schema
- **Comprehensive Schema**: JSON Schema supporting 44+ billing models across all major cloud providers
- **Multi-Provider Support**: Native support for AWS, Azure, GCP, Kubernetes, and custom providers
- **Production SDK**: Complete Go SDK with automatic protobuf generation
- **Enterprise Testing**: Multi-level conformance testing framework (Basic, Standard, Advanced)
- **CI/CD Ready**: Complete GitHub Actions pipeline with validation and benchmarks

### Supported Billing Models

- **Time-based**: per_hour, per_minute, per_second, per_cpu_hour, per_memory_gb_hour
- **Storage**: per_gb_month, per_gb_hour, per_gb_day, per_iops, per_provisioned_iops
- **Usage-based**: per_request, per_invocation, per_operation, per_transaction
- **Commitment**: reserved, spot, preemptible, savings_plan, committed_use
- **Database**: per_rcu, per_wcu, per_dtu, per_ru
- **Network**: per_data_transfer_gb, per_bandwidth_gb, per_api_call
- **And more**: [See complete list](schemas/pricing_spec.schema.json)

### Key Capabilities

- **Tiered Pricing**: Volume-based pricing tiers with automatic discounts
- **Time Aggregation**: Flexible cost aggregation rules (calendar, billing, continuous)
- **Commitment Terms**: Reserved instance and spot pricing with discount tracking
- **Rich Metadata**: Resource tags, plugin metadata, and metric hints
- **Enterprise Tags**: Billing centers, cost centers, environments, projects

## Repository Structure

```text
finfocus-spec/
├─ proto/finfocus/v1/           # gRPC service definitions
│  └─ costsource.proto            # Complete CostSource service specification
├─ schemas/                       # JSON schema validation
│  └─ pricing_spec.schema.json    # Comprehensive pricing schema (44+ billing modes)
├─ sdk/go/                        # Production Go SDK
│  ├─ proto/                      # Generated protobuf bindings (auto-generated)
│  ├─ pluginsdk/                  # Plugin development SDK with Serve(), logging, metrics
│  │  └─ mapping/                 # Property extraction helpers (AWS, Azure, GCP)
│  ├─ pricing/                    # Domain types, billing modes, and validation
│  ├─ currency/                   # ISO 4217 currency validation (180+ currencies)
│  ├─ registry/                   # Plugin registry domain types (8 enum types)
│  └─ testing/                    # Complete testing framework with conformance
├─ examples/                      # Cross-vendor examples
│  ├─ specs/                      # 9 comprehensive pricing examples
│  └─ requests/                   # Sample gRPC request payloads
├─ docs/                          # Comprehensive documentation
│  ├─ PLUGIN_MIGRATION_GUIDE.md   # Migration guide for breaking changes
│  ├─ PLUGIN_STARTUP_PROTOCOL.md  # Plugin startup and lifecycle
│  ├─ PROPERTY_MAPPING.md         # Property extraction documentation
│  └─ focus-columns.md            # FOCUS 1.2 column mapping
├─ .github/workflows/             # Enterprise CI/CD pipeline
│  └─ ci.yml                      # Complete validation and testing
├─ OBSERVABILITY_GUIDE.md         # Structured logging and metrics guide
└─ PLUGIN_DEVELOPER_GUIDE.md      # Complete plugin development guide
```

### Core Components

- **[gRPC Service](proto/finfocus/v1/costsource.proto)**: CostSourceService with 11 RPC methods
- **[JSON Schema](schemas/pricing_spec.schema.json)**: Comprehensive validation supporting all major cloud providers
- **[Go SDK](sdk/go/)**: Production-ready SDK with automatic protobuf generation
- **[Plugin SDK](sdk/go/pluginsdk/)**: Serve(), environment handling, logging, metrics, FOCUS builder
- **[Pricing Package](sdk/go/pricing/)**: 44+ billing modes, domain types, schema validation
- **[Currency Package](sdk/go/currency/)**: ISO 4217 validation with zero-allocation performance
- **[Registry Package](sdk/go/registry/)**: Plugin registry types with optimized enum validation
- **[Testing Framework](sdk/go/testing/)**: Multi-level conformance testing (Basic, Standard, Advanced)
- **[Examples](examples/)**: Cross-vendor examples demonstrating all major billing models
- **[Plugin Developer Guide](PLUGIN_DEVELOPER_GUIDE.md)**: Complete guide to building plugins
- **[Observability Guide](OBSERVABILITY_GUIDE.md)**: Structured logging and Prometheus metrics

## Quick Start

### Installation

```bash
# Add SDK to your Go project
go get github.com/rshade/finfocus-spec/sdk/go/proto      # Generated protobuf code
go get github.com/rshade/finfocus-spec/sdk/go/pluginsdk  # Plugin development SDK
go get github.com/rshade/finfocus-spec/sdk/go/pricing    # Domain types and validation
go get github.com/rshade/finfocus-spec/sdk/go/currency   # ISO 4217 currency validation
go get github.com/rshade/finfocus-spec/sdk/go/registry   # Plugin registry types
```

### Development Setup

```bash
# Clone repository
git clone https://github.com/rshade/finfocus-spec.git
cd finfocus-spec

# Generate Go SDK (installs buf automatically)
make generate

# Run complete test suite
make test

# Validate implementation
make validate
```

### Build System

- `make generate` - Generate Go SDK from protobuf (auto-installs buf CLI)
- `make test` - Run comprehensive test suite
- `make validate` - Run validation (tests + linting + schema validation)
- `make clean` - Clean generated files
- `make tidy` - Tidy Go dependencies

## SDK Client Timeout Configuration

The FinFocus Go SDK supports configurable per-client timeouts for plugin RPC calls.
This prevents indefinite blocking on slow servers.

### Basic Usage

```go
// Configure client with 5-second timeout
cfg := pluginsdk.DefaultClientConfig("http://localhost:8080")
cfg = cfg.WithTimeout(5 * time.Second)
client := pluginsdk.NewClient(cfg)
```

### Timeout Precedence Rules

When making RPC calls, timeout is resolved in this order (highest to lowest priority):

1. **Context Deadline** (if set via `context.WithTimeout()`)
2. **Custom HTTPClient.Timeout** (if `HTTPClient` provided)
3. **ClientConfig.Timeout** (if > 0)
4. **DefaultClientTimeout** (30 seconds)

### Context Deadline Override

```go
cfg := pluginsdk.DefaultClientConfig("http://localhost:8080")
cfg = cfg.WithTimeout(30 * time.Second)
client := pluginsdk.NewClient(cfg)

// Context deadline (1 second) takes precedence over client timeout (30 seconds)
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()
resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
if err != nil {
    // Times out after 1 second (not 30 seconds)
}
```

## User-Friendly GetPluginInfo Error Messages

The SDK returns user-friendly error messages for GetPluginInfo RPC failures,
while logging detailed information server-side for debugging.

### Error Messages

| Error Condition      | Client Receives                                    | Server Logs                                   |
| -------------------- | -------------------------------------------------- | --------------------------------------------- |
| Plugin returns nil   | "unable to retrieve plugin metadata"               | "GetPluginInfo returned nil response"         |
| Incomplete metadata  | "plugin metadata is incomplete"                    | "GetPluginInfo returned incomplete response"  |
| Invalid spec version | "plugin reported an invalid specification version" | "GetPluginInfo returned invalid spec_version" |

## GetPluginInfo Performance Conformance

The SDK includes performance conformance tests for GetPluginInfo RPC calls.

### Thresholds

- **Standard Conformance**: ≤100ms average latency
- **Advanced Conformance**: ≤50ms average latency

### Running Tests

```bash
# Run performance conformance test
go test -v ./sdk/go/testing -run Performance_GetPluginInfoLatency
```

### Legacy Plugin Support

Plugins without GetPluginInfo implementation are handled gracefully (Unimplemented error does not
fail performance tests).

## Plugin Development

### Creating a Cost Source Plugin

Implement the `CostSourceServiceServer` interface:

#### Complete Plugin Implementation

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/protobuf/types/known/timestamppb"

    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

type kubecostPlugin struct {
    pbc.UnimplementedCostSourceServiceServer
}

// Name returns the plugin name
func (k *kubecostPlugin) Name(ctx context.Context, req *pbc.NameRequest) (*pbc.NameResponse, error) {
    return &pbc.NameResponse{
        Name: "kubecost",
    }, nil
}

// Supports checks if a resource type is supported
func (k *kubecostPlugin) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
    resource := req.GetResource()

    // Example: Support Kubernetes namespaces
    if resource.GetProvider() == "kubernetes" && resource.GetResourceType() == "namespace" {
        return &pbc.SupportsResponse{
            Supported: true,
            Reason:    "",
        }, nil
    }

    return &pbc.SupportsResponse{
        Supported: false,
        Reason:    "Only Kubernetes namespaces are supported",
    }, nil
}

// GetActualCost retrieves historical cost data
func (k *kubecostPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
    // Example: Return mock cost data for demonstration
    results := []*pbc.ActualCostResult{
        {
            Timestamp:   timestamppb.New(time.Now().Add(-24 * time.Hour)),
            Cost:        45.67,
            UsageAmount: 12.5,
            UsageUnit:   "cpu-hours",
            Source:      "kubecost",
        },
        {
            Timestamp:   timestamppb.New(time.Now().Add(-12 * time.Hour)),
            Cost:        52.34,
            UsageAmount: 14.2,
            UsageUnit:   "cpu-hours",
            Source:      "kubecost",
        },
    }

    // Use pluginsdk helper for consistent response construction
    return pluginsdk.NewActualCostResponse(
        pluginsdk.WithResults(results),
        pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
    ), nil
}

// GetProjectedCost calculates projected costs
func (k *kubecostPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
    return &pbc.GetProjectedCostResponse{
        UnitPrice:     0.03,
        Currency:      "USD",
        CostPerMonth:  216.00, // 30 days * 24 hours * 0.03
        BillingDetail: "kubecost-hourly-avg",
    }, nil
}

// GetPricingSpec returns detailed pricing specification
func (k *kubecostPlugin) GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error) {
    spec := &pbc.PricingSpec{
        Provider:     "kubernetes",
        ResourceType: "namespace",
        BillingMode:  "per_cpu_hour",
        RatePerUnit:  0.03,
        Currency:     "USD",
        Description:  "Kubernetes namespace CPU pricing via Kubecost",
        MetricHints: []*pbc.UsageMetricHint{
            {
                Metric: "cpu_cores",
                Unit:   "hour",
            },
            {
                Metric: "memory_gb",
                Unit:   "hour",
            },
        },
        PluginMetadata: map[string]string{
            "cluster_name":     "production-cluster",
            "namespace_labels": "app=web,tier=frontend",
            "kubecost_version": "1.98.0",
        },
        Source: "kubecost",
    }

    return &pbc.GetPricingSpecResponse{
        Spec: spec,
    }, nil
}

// EstimateCost estimates the monthly cost for a resource before deployment
func (k *kubecostPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error) {
    // Example: Estimate cost for a Kubernetes workload
    // In production, this would analyze the attributes to calculate actual cost
    return &pbc.EstimateCostResponse{
        Currency:    "USD",
        CostMonthly: 216.00, // 30 days * 24 hours * 0.03 per hour
    }, nil
}

func main() {
    // Create gRPC server
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    server := grpc.NewServer()
    plugin := &kubecostPlugin{}

    pbc.RegisterCostSourceServiceServer(server, plugin)

    fmt.Println("Kubecost plugin server listening on :50051")
    if err := server.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

#### Consuming Cost Source Plugins

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func main() {
    // Connect to cost source plugin
    conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pbc.NewCostSourceServiceClient(conn)
    ctx := context.Background()

    // Get plugin name
    nameResp, err := client.Name(ctx, &pbc.NameRequest{})
    if err != nil {
        log.Fatalf("Failed to get name: %v", err)
    }
    fmt.Printf("Connected to plugin: %s\n", nameResp.GetName())

    // Check if plugin supports a resource
    resource := &pbc.ResourceDescriptor{
        Provider:     "kubernetes",
        ResourceType: "namespace",
        Region:       "us-east-1",
        Tags: map[string]string{
            "app":  "web",
            "tier": "frontend",
        },
    }

    supportsResp, err := client.Supports(ctx, &pbc.SupportsRequest{
        Resource: resource,
    })
    if err != nil {
        log.Fatalf("Failed to check support: %v", err)
    }

    if !supportsResp.GetSupported() {
        fmt.Printf("Resource not supported: %s\n", supportsResp.GetReason())
        return
    }

    // Get actual cost data
    actualCostResp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
        ResourceId: "namespace/production",
        Start:      timestamppb.New(time.Now().Add(-48 * time.Hour)),
        End:        timestamppb.New(time.Now()),
        Tags: map[string]string{
            "app": "web",
        },
    })
    if err != nil {
        log.Fatalf("Failed to get actual cost: %v", err)
    }

    fmt.Printf("\nActual cost data:\n")
    for _, result := range actualCostResp.GetResults() {
        fmt.Printf("  Time: %s, Cost: $%.2f, Usage: %.2f %s\n",
            result.GetTimestamp().AsTime().Format(time.RFC3339),
            result.GetCost(),
            result.GetUsageAmount(),
            result.GetUsageUnit())
    }

    // Get projected cost
    projectedResp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
        Resource: resource,
    })
    if err != nil {
        log.Fatalf("Failed to get projected cost: %v", err)
    }

    fmt.Printf("\nProjected cost:\n")
    fmt.Printf("  Unit price: $%.4f %s\n", projectedResp.GetUnitPrice(), projectedResp.GetCurrency())
    fmt.Printf("  Monthly cost: $%.2f\n", projectedResp.GetCostPerMonth())
    fmt.Printf("  Billing detail: %s\n", projectedResp.GetBillingDetail())

    // Get pricing specification
    specResp, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
        Resource: resource,
    })
    if err != nil {
        log.Fatalf("Failed to get pricing spec: %v", err)
    }

    spec := specResp.GetSpec()
    fmt.Printf("\nPricing specification:\n")
    fmt.Printf("  Provider: %s\n", spec.GetProvider())
    fmt.Printf("  Resource Type: %s\n", spec.GetResourceType())
    fmt.Printf("  Billing Mode: %s\n", spec.GetBillingMode())
    fmt.Printf("  Rate per unit: $%.4f %s\n", spec.GetRatePerUnit(), spec.GetCurrency())
    fmt.Printf("  Description: %s\n", spec.GetDescription())

    if len(spec.GetMetricHints()) > 0 {
        fmt.Printf("  Metric hints:\n")
        for _, hint := range spec.GetMetricHints() {
            fmt.Printf("    - %s (%s)\n", hint.GetMetric(), hint.GetUnit())
        }
    }

    // Estimate cost for a resource before deployment ("what-if" analysis)
    // Use structpb.NewStruct for ergonomic attribute construction
    attrs, _ := structpb.NewStruct(map[string]interface{}{
        "cpu_limit":    "2",
        "memory_limit": "4Gi",
    })
    estimateResp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
        ResourceType: "kubernetes:core/v1:Namespace",
        Attributes:   attrs,
    })
    if err != nil {
        log.Fatalf("Failed to estimate cost: %v", err)
    }

    fmt.Printf("\nEstimated cost (before deployment):\n")
    fmt.Printf("  Monthly cost: $%.2f %s\n", estimateResp.GetCostMonthly(), estimateResp.GetCurrency())
}
```

### Schema Validation

#### Validating PricingSpec Documents

```go
package main

import (
    "fmt"
    "log"

    "github.com/rshade/finfocus-spec/sdk/go/pricing"
)

func main() {
    // Example PricingSpec JSON document
    pricingSpecJSON := `{
        "provider": "aws",
        "resource_type": "ec2",
        "sku": "t3.micro",
        "region": "us-east-1",
        "billing_mode": "per_hour",
        "rate_per_unit": 0.0104,
        "currency": "USD",
        "description": "General Purpose t3.micro instance",
        "metric_hints": [
            {
                "metric": "vcpu_hours",
                "unit": "hour",
                "aggregation_method": "sum"
            }
        ],
        "source": "aws"
    }`

    // Validate the JSON document against the schema
    err := pricing.ValidatePricingSpec([]byte(pricingSpecJSON))
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    fmt.Println("PricingSpec JSON is valid!")
}
```

#### Working with Billing Modes

The specification supports 44+ billing models across all major cloud providers:

```go
package main

import (
    "fmt"

    "github.com/rshade/finfocus-spec/sdk/go/pricing"
)

func main() {
    // Check if a billing mode is valid
    validModes := []string{
        "per_hour",
        "per_gb_month",
        "per_request",
        "spot",
        "reserved",
    }

    for _, mode := range validModes {
        if pricing.IsValidBillingMode(mode) {
            fmt.Printf("✓ %s is a valid billing mode\n", mode)
        } else {
            fmt.Printf("✗ %s is not a valid billing mode\n", mode)
        }
    }

    // Get all available billing modes
    fmt.Printf("\nAll available billing modes:\n")
    for _, mode := range pricing.GetAllBillingModes() {
        fmt.Printf("  - %s\n", mode)
    }
}
```

## Testing Framework

FinFocus Specification includes a comprehensive testing framework with multi-level conformance validation.

### Plugin Testing

```go
package main

import (
    "testing"
    plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestMyPlugin(t *testing.T) {
    plugin := &MyPluginImpl{}

    // Run basic integration tests
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(t)
    defer harness.Stop()

    // Test with mock client
    client := harness.Client()
    // ... test your plugin
}
```

### Conformance Testing

Three levels of conformance testing ensure production readiness:

#### Basic Conformance (Required)

- Core functionality validation
- Error handling requirements
- Basic response validation
- **Required for plugin submission**

#### Standard Conformance (Recommended)

- Data consistency validation
- Performance requirements (response times)
- Concurrency testing (10+ requests)
- **Recommended for enterprise deployments**

#### Advanced Conformance (High-Performance)

- Scalability testing (50+ concurrent requests)
- Large dataset handling (30+ days)
- Performance benchmarks
- **Required for high-throughput environments**

```go
func TestPluginConformance(t *testing.T) {
    plugin := &MyPluginImpl{}

    // Choose conformance level
    result := plugintesting.RunStandardConformanceTests(t, plugin)
    plugintesting.PrintConformanceReport(result)

    if result.FailedTests > 0 {
        t.Errorf("Plugin failed conformance: %s", result.Summary)
    }
}
```

### Performance Testing

```bash
# Run complete test suite
make test

# Run performance benchmarks
go test -bench=. -benchmem ./sdk/go/...

# Run conformance tests
go test -v -run TestConformance ./sdk/go/testing/

# Run integration tests
go test -v -tags=integration ./sdk/go/testing/
```

See [Testing Framework Documentation](sdk/go/testing/README.md) for complete details.

## Examples

The specification includes 9 comprehensive examples demonstrating all major cloud providers and billing models:

### Cloud Provider Examples

#### AWS (Amazon Web Services)

- **[EC2 t3.micro](examples/specs/aws-ec2-t3-micro.json)**: On-demand instance pricing
- **[S3 Tiered Storage](examples/specs/aws-s3-tiered-pricing.json)**: Volume-based tiered pricing
- **[Lambda Functions](examples/specs/aws-lambda-per-invocation.json)**: Serverless execution pricing
- **[Lambda@Edge (Not Implemented)](examples/specs/aws-lambda-not-implemented.json)**: Edge case handling

#### Azure (Microsoft Azure)

- **[Virtual Machine](examples/specs/azure-vm-per-second.json)**: Per-second billing with time aggregation
- **[SQL Database](examples/specs/azure-sql-dtu.json)**: DTU-based database pricing

#### GCP (Google Cloud Platform)

- **[Cloud Storage](examples/specs/gcp-storage-standard.json)**: Tiered storage pricing
- **[Preemptible Instances](examples/specs/gcp-preemptible-spot.json)**: Spot pricing with commitment terms

#### Kubernetes

- **[Namespace CPU](examples/specs/kubernetes-namespace-cpu.json)**: CPU-based pricing via Kubecost

### Billing Model Examples

- **Time-based**: Hourly, per-second, CPU hours, memory hours
- **Storage**: GB-month, GB-day, IOPS, provisioned IOPS
- **Usage-based**: Per request, per invocation, per transaction
- **Commitment**: Reserved, spot, preemptible with discount tracking
- **Tiered**: Volume-based pricing with automatic discounts
- **Database**: RCU, WCU, DTU, RU pricing models

See [Examples Documentation](examples/README.md) for detailed explanations.

## Architecture

### gRPC Service Interface

The CostSourceService provides 11 RPC methods for comprehensive cost management:

```protobuf
service CostSourceService {
  // Core Plugin Information
  rpc Name(NameRequest) returns (NameResponse);                              // Plugin identification
  rpc Supports(SupportsRequest) returns (SupportsResponse);                  // Resource support check
  rpc GetPluginInfo(GetPluginInfoRequest) returns (GetPluginInfoResponse);   // Plugin metadata

  // Cost Data Retrieval
  rpc GetActualCost(GetActualCostRequest) returns (GetActualCostResponse);   // Historical costs (FOCUS 1.2)
  rpc GetProjectedCost(GetProjectedCostRequest) returns (GetProjectedCostResponse); // Cost projections
  rpc GetPricingSpec(GetPricingSpecRequest) returns (GetPricingSpecResponse);       // Pricing specifications

  // Pre-Deployment Analysis
  rpc EstimateCost(EstimateCostRequest) returns (EstimateCostResponse);      // "What-if" cost estimation
  rpc DryRun(DryRunRequest) returns (DryRunResponse);                        // Field mapping introspection

  // Cost Optimization
  rpc GetRecommendations(GetRecommendationsRequest) returns (GetRecommendationsResponse); // Cost optimization
  rpc DismissRecommendation(DismissRecommendationRequest) returns (DismissRecommendationResponse);
  rpc GetBudgets(GetBudgetsRequest) returns (GetBudgetsResponse);            // Budget tracking and alerts
}
```

### JSON Schema Validation

The [pricing specification schema](schemas/pricing_spec.schema.json) validates:

- **Required fields**: provider, resource_type, billing_mode, rate_per_unit, currency
- **44+ billing models**: Supporting all major cloud provider patterns
- **Advanced features**: Tiered pricing, time aggregation, commitment terms
- **Rich metadata**: Resource tags, plugin metadata, metric hints

### SDK Architecture

- **[Generated Proto](sdk/go/proto/)**: Auto-generated from protobuf definitions
- **[Plugin SDK](sdk/go/pluginsdk/)**: Serve(), environment handling, logging, metrics, FOCUS builder
  - **[Response Builders](sdk/go/pluginsdk/README.md#response-builders-and-validation)**:
    NewActualCostResponse, FallbackHint, validation helpers
  - **[Mapping](sdk/go/pluginsdk/mapping/)**: Property extraction helpers for AWS, Azure, GCP
- **[Pricing](sdk/go/pricing/)**: Domain types, validation, 44+ billing mode constants
- **[Currency](sdk/go/currency/)**: ISO 4217 validation (180+ currencies, zero-allocation)
- **[Registry](sdk/go/registry/)**: Plugin registry types (8 enum types, zero-allocation)
- **[Testing Framework](sdk/go/testing/)**: Comprehensive plugin testing suite with conformance

## CI/CD Pipeline

Complete GitHub Actions pipeline with:

- **SDK Generation**: Automatic protobuf compilation with up-to-date verification
- **Testing**: Unit tests, integration tests, conformance tests
- **Performance**: Automated performance regression testing (10% threshold) with artifact upload
- **Validation**: Go linting, buf linting, JSON schema validation
- **Breaking Changes**: Automatic detection with buf
- **Coverage**: Code coverage reporting with Codecov

## Production Readiness

### Enterprise Features

- ✅ **Complete specification**: All major cloud providers and billing models
- ✅ **Multi-level testing**: Basic, Standard, Advanced conformance levels
- ✅ **Performance validation**: Response time requirements and benchmarks
- ✅ **Schema validation**: Comprehensive JSON schema with all examples validated
- ✅ **Error handling**: Standardized gRPC error codes and patterns
- ✅ **Concurrency support**: Thread-safe plugin requirements
- ✅ **Documentation**: Complete API reference and developer guides
- ✅ **CI/CD pipeline**: Automated validation and testing

### Plugin Certification

Plugins can be certified at three levels:

1. **Basic**: Core functionality, required for submission
2. **Standard**: Production-ready, recommended for enterprise
3. **Advanced**: High-performance, required for scale

## Contributing

### Prerequisites

This project requires specific Node.js, Go, and linter versions for consistent
builds. All of them are pinned in `mise.toml`:

```bash
# Install mise if not installed
# See: https://mise.jdx.dev/getting-started.html

# Install every pinned tool (go, node, golangci-lint, buf)
mise install

# Verify versions
mise ls        # Each tool should resolve to the mise.toml pin
node --version # 24.15.0
go version     # 1.27.1
```

**Why mise?** `mise.toml` is the single source of truth for tool versions, and
`.github/workflows/ci.yml` provisions the exact same file via `jdx/mise-action`.
That keeps local builds and CI byte-identical, and prevents `package-lock.json`
drift from npm version differences. Bump a version in `mise.toml` and both
follow.

### Development Workflow

```bash
# 0. Ensure correct Node version
nvm use

# 1. Make changes to proto or schema files
vim proto/finfocus/v1/costsource.proto
vim schemas/pricing_spec.schema.json

# 2. Generate SDK and validate
make generate
make validate

# 3. Run complete test suite
make test

# 4. Submit PR
git add .
git commit -m "feat: add new billing model"
git push origin feature-branch
```

### Requirements

- Run `make validate` before submitting PRs
- All examples must pass schema validation
- Breaking changes require buf validation
- New billing models require examples and tests

### Versioning

Semantic versioning for proto changes:

- **MAJOR**: Breaking proto changes
- **MINOR**: Backward-compatible additions
- **PATCH**: Bug fixes, documentation updates

Current version: **v0.5.4** (production-ready)

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Support

### Guides

- **[Plugin Developer Guide](./PLUGIN_DEVELOPER_GUIDE.md)** - Complete guide to building cost source plugins
- **[Observability Guide](./OBSERVABILITY_GUIDE.md)** - Structured logging and Prometheus metrics
- **[Plugin Startup Protocol](./docs/PLUGIN_STARTUP_PROTOCOL.md)** - Plugin lifecycle and initialization
- **[Property Mapping](./docs/PROPERTY_MAPPING.md)** - Extracting properties from cloud resources
- **[FOCUS 1.2 Columns](./docs/focus-columns.md)** - FinOps FOCUS column mapping reference
- **[Plugin Registry Spec](./docs/plugin-registry-specification.md)** - Plugin registration and discovery
- **[Migration Guide](./docs/PLUGIN_MIGRATION_GUIDE.md)** - Upgrading between spec versions
- **[Project Rename Migration](./MIGRATION.md)** - Guide for migrating from PulumiCost to FinFocus

### Community

- **[Issues](https://github.com/rshade/finfocus-spec/issues)** - Bug reports and feature requests
- **[Project Board](https://github.com/users/rshade/projects/3)** - Development roadmap and progress

---

**FinFocus Specification v0.5.4** - Production-ready protocol for cloud cost source plugins
