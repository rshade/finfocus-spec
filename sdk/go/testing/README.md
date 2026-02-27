# FinFocus Plugin Testing Framework

This package provides a comprehensive testing framework for FinFocus plugin
implementations, including integration tests, performance benchmarks, and conformance validation.

## Overview

The testing framework consists of several key components:

- **Test Harness**: In-memory gRPC testing environment using bufconn
- **Mock Plugin**: Configurable mock implementation for testing
- **Conformance Suite**: Four-category test suite with three conformance levels
- **Performance Benchmarks**: Standardized performance testing with latency baselines
- **Concurrency Tests**: Thread safety and parallel request handling validation

### Test Categories

The conformance suite organizes tests into four categories:

| Category            | Description                               | Example Tests                                               |
| ------------------- | ----------------------------------------- | ----------------------------------------------------------- |
| **Spec Validation** | JSON schema and data format compliance    | Billing mode enum, currency format, required fields         |
| **RPC Correctness** | Protocol behavior and response validation | Name response format, error handling, time range validation |
| **Performance**     | Latency thresholds and response time      | Method latency, baseline variance (< 10%)                   |
| **Concurrency**     | Thread safety and parallel execution      | Parallel requests, response consistency                     |

### Conformance Levels

| Level        | Description              | Requirements                                          |
| ------------ | ------------------------ | ----------------------------------------------------- |
| **Basic**    | Required for all plugins | Core functionality, spec compliance                   |
| **Standard** | Production-ready plugins | Reliability, consistency, 10 parallel requests        |
| **Advanced** | High-performance plugins | Strict latency, 50+ parallel requests, variance < 10% |

## Quick Start

### Basic Plugin Testing

```go
package main

import (
    "testing"

    plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Assume you have a plugin implementation
func TestMyPlugin(t *testing.T) {
    plugin := &MyPluginImpl{} // Your implementation

    // Run basic integration tests
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(t)
    defer harness.Stop()

    // Test individual methods
    client := harness.Client()
    ctx := context.Background()

    resp, err := client.Name(ctx, &pbc.NameRequest{})
    if err != nil {
        t.Fatalf("Name() failed: %v", err)
    }

    if err := plugintesting.ValidateNameResponse(resp); err != nil {
        t.Errorf("Invalid response: %v", err)
    }
}
```

### Running Conformance Tests

```go
func TestPluginConformance(t *testing.T) {
    plugin := &MyPluginImpl{}

    // Create conformance suite
    suite := plugintesting.NewConformanceSuite()

    // Register all test categories
    plugintesting.RegisterSpecValidationTests(suite)
    plugintesting.RegisterRPCCorrectnessTests(suite)
    plugintesting.RegisterPerformanceTests(suite)
    plugintesting.RegisterConcurrencyTests(suite)

    // Run tests at desired conformance level
    result := suite.Run(plugin, plugintesting.ConformanceLevelStandard)

    // Print detailed report
    plugintesting.PrintReportTo(result, os.Stdout)

    if !result.Passed() {
        t.Errorf("Plugin failed conformance tests")
    }
}

// Or use convenience functions
func TestPluginConformanceSimple(t *testing.T) {
    plugin := &MyPluginImpl{}

    // Run standard conformance (includes Basic + Standard tests)
    result, err := plugintesting.RunStandardConformance(plugin)
    if err != nil {
        t.Fatalf("conformance tests failed to run: %v", err)
    }
    plugintesting.PrintReportTo(result, os.Stdout)

    if !result.Passed() {
        t.Errorf("Plugin failed conformance: %d/%d tests failed",
            result.Summary.Failed, result.Summary.Total)
    }
}
```

### Performance Testing

```go
func TestPluginPerformance(t *testing.T) {
    plugin := &MyPluginImpl{}
    suite := plugintesting.NewPerformanceTestSuite(plugin)
    suite.RunPerformanceTests(t)
}

func BenchmarkMyPlugin(b *testing.B) {
    plugin := &MyPluginImpl{}
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(&testing.T{})
    defer harness.Stop()

    client := harness.Client()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.Name(ctx, &pbc.NameRequest{})
        if err != nil {
            b.Fatalf("Name() failed: %v", err)
        }
    }
}
```

## Components

### Conformance Suite

The `ConformanceSuite` is the main entry point for plugin validation:

```go
// Create and configure suite
suite := plugintesting.NewConformanceSuite()

// Configure options
suite.WithConfig(plugintesting.SuiteConfig{
    Timeout:          60 * time.Second,
    ParallelRequests: 10,
})

// Register test categories
plugintesting.RegisterSpecValidationTests(suite)
plugintesting.RegisterRPCCorrectnessTests(suite)
plugintesting.RegisterPerformanceTests(suite)
plugintesting.RegisterConcurrencyTests(suite)

// Run at desired level
result := suite.Run(plugin, plugintesting.ConformanceLevelStandard)
```

**Convenience Functions:**

```go
// Run all tests at specific level
result, err := plugintesting.RunBasicConformance(plugin)
result, err := plugintesting.RunStandardConformance(plugin)
result, err := plugintesting.RunAdvancedConformance(plugin)
if err != nil {
    t.Fatalf("conformance tests failed to run: %v", err)
}
```

### Test Harness

The `TestHarness` provides an in-memory gRPC testing environment:

```go
type TestHarness struct {
    server   *grpc.Server
    listener *bufconn.Listener
    client   pbc.CostSourceServiceClient
    conn     *grpc.ClientConn
}
```

**Key Methods:**

- `NewTestHarness(impl)`: Create harness for plugin implementation
- `Start(t)`: Initialize client connection
- `Stop()`: Clean up resources
- `Client()`: Get gRPC client for testing

### Mock Plugin

The `MockPlugin` provides a configurable test implementation:

```go
// Basic mock plugin
plugin := plugintesting.NewMockPlugin()

// Configure supported providers/resources
plugin.SupportedProviders = []string{"aws", "azure"}
plugin.SupportedResources["aws"] = []string{"ec2", "s3"}

// Configure error behavior
plugin.ShouldErrorOnName = true

// Configure response delays
plugin.NameDelay = 100 * time.Millisecond
```

**Specialized Mock Plugins:**

- `ConfigurableErrorMockPlugin()`: For error testing
- `SlowMockPlugin()`: For timeout/performance testing

### Batch Cost Testing

`MockPlugin` includes batch RPC controls for conformance and integration tests:

- `ShouldErrorOnBatchCost`: force top-level BatchCost RPC failure
- `BatchCostDelay`: add latency to simulate slower backends
- `UnsupportedBatchResourceTypes`: per-resource partial failure injection (`resource_type_unsupported=true`)

Example:

```go
plugin := plugintesting.NewMockPlugin()
plugin.UnsupportedBatchResourceTypes["unsupported_resource"] = true

harness := plugintesting.NewTestHarness(plugin)
harness.Start(t)
defer harness.Stop()

ctx := context.Background()
resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
    QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
    Resources: []*pbc.ResourceDescriptor{
        plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
        plugintesting.CreateResourceDescriptor("aws", "unsupported_resource", "", "us-east-1"),
    },
})
require.NoError(t, err)
require.Len(t, resp.GetResults(), 2)
```

Batch conformance coverage includes:

- Query type behavior (`ESTIMATE`, `ACTUAL`, `PROJECTED`, `DRY_RUN`, `UNSPECIFIED`)
- Positional ordering (`results[i]` matches `resources[i]`)
- Partial failure semantics (mixed success/error in one response)
- Fallback path behavior for plugins without `BatchCostHandler`

### Validation Functions

The framework provides comprehensive response validation:

```go
// Validate individual responses
err := plugintesting.ValidateNameResponse(nameResp)
err := plugintesting.ValidateSupportsResponse(supportsResp)
err := plugintesting.ValidateActualCostResponse(actualCostResp)
err := plugintesting.ValidateProjectedCostResponse(projectedResp)
err := plugintesting.ValidatePricingSpecResponse(specResp)

// Validate protobuf messages
err := plugintesting.ValidatePricingSpec(spec)
err := plugintesting.ValidateActualCostResult(result)
```

### FOCUS Record Validation (Contextual FinOps)

The `pluginsdk` package provides comprehensive FOCUS 1.2/1.3 validation for cost records:

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Basic validation (fail-fast mode)
err := pluginsdk.ValidateFocusRecord(record)

// Aggregate mode - collect all validation errors
opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)
for _, err := range errs {
    log.Printf("Validation error: %v", err)
}
```

**Validation Rules Enforced:**

| Rule   | Severity | Description                                                   | Sentinel Error                         |
| ------ | -------- | ------------------------------------------------------------- | -------------------------------------- |
| FR-001 | Critical | EffectiveCost must not exceed BilledCost                      | `ErrEffectiveCostExceedsBilledCost`    |
| FR-002 | Critical | ListCost must be >= EffectiveCost                             | `ErrListCostLessThanEffectiveCost`     |
| FR-003 | Major    | CommitmentDiscountStatus required when ID set + Usage charges | `ErrCommitmentStatusMissing`           |
| FR-004 | Major    | CommitmentDiscountId required when Status is set              | `ErrCommitmentIDMissingForStatus`      |
| FR-005 | Major    | CapacityReservationStatus required when ID set + Usage        | `ErrCapacityReservationStatusMissing`  |
| FR-005 | Major    | CapacityReservationId required when Status is set             | `ErrCapacityReservationIDMissing`      |
| FR-006 | Minor    | PricingUnit required when PricingQuantity > 0                 | `ErrPricingUnitMissing`                |

**Error Checking with Sentinel Errors:**

```go
import "errors"

err := pluginsdk.ValidateFocusRecord(record)
if errors.Is(err, pluginsdk.ErrEffectiveCostExceedsBilledCost) {
    // Handle cost hierarchy violation
}
if errors.Is(err, pluginsdk.ErrCommitmentStatusMissing) {
    // Handle commitment discount consistency issue
}
```

**Exemptions:**

- ChargeClass `CORRECTION` is exempt from cost hierarchy rules (FR-001, FR-002) and contracted cost validation
- Negative costs (credits/refunds) are exempt from hierarchy validation (FR-001, FR-002)
- Zero costs (free tier) pass validation without error
- ChargeCategory `PURCHASE` does not require CommitmentDiscountStatus (FR-003) or CapacityReservationStatus (FR-005)

**Performance:**

- Zero allocations on valid records (sentinel error pattern)
- Benchmarks: ~1000 ns/op for full validation, 0 B/op, 0 allocs/op

### Conformance Result

The `ConformanceResult` contains detailed test execution information:

```go
type ConformanceResult struct {
    Version          string                           // Report schema version
    Timestamp        time.Time                        // When suite was executed
    PluginName       string                           // Name from plugin's Name() RPC
    LevelAchieved    ConformanceLevel                 // Highest level passed
    Summary          ResultSummary                    // Aggregate test counts
    Categories       map[TestCategory]*CategoryResult // Results by category
    Duration         time.Duration                    // Total execution time
}

type ResultSummary struct {
    Total   int // Total tests executed
    Passed  int // Tests that passed
    Failed  int // Tests that failed
    Skipped int // Tests skipped
}

type TestResult struct {
    Method   string        // RPC method tested
    Category TestCategory  // Test category
    Success  bool          // Pass/fail status
    Error    error         // Error if failed
    Duration time.Duration // Test execution time
    Details  string        // Additional information
}
```

**Reporting:**

```go
// Print to stdout
plugintesting.PrintReportTo(result, os.Stdout)

// Get JSON report
jsonReport := result.ToJSON()
```

### Helper Functions

```go
// Create test data
resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
start, end := plugintesting.CreateTimeRange(24) // 24 hours ago to now

// Performance measurement
metrics, err := plugintesting.MeasurePerformance("TestName", 100, func() error {
    _, err := client.Name(ctx, &pbc.NameRequest{})
    return err
})
```

## Testing Levels

### 1. Basic Integration Tests

Tests core functionality of all RPC methods:

```go
func TestBasicPluginFunctionality(t *testing.T)
func TestErrorHandling(t *testing.T)
func TestInputValidation(t *testing.T)
func TestMultipleProviders(t *testing.T)
func TestConcurrentRequests(t *testing.T)
func TestDataConsistency(t *testing.T)
```

**Run with:**

```bash
go test -v -run TestBasicPluginFunctionality
```

### 2. Performance Benchmarks

Standardized performance tests for all RPC methods:

```go
func BenchmarkName(b *testing.B)
func BenchmarkSupports(b *testing.B)
func BenchmarkGetActualCost(b *testing.B)
func BenchmarkGetProjectedCost(b *testing.B)
func BenchmarkGetPricingSpec(b *testing.B)
func BenchmarkGetRecommendations_LargeResultSet(b *testing.B)
func BenchmarkGetRecommendations_LargeResultSetPagination(b *testing.B)
func BenchmarkAllMethods(b *testing.B)
func BenchmarkConcurrentRequests(b *testing.B)
```

**Run with:**

```bash
go test -bench=. -benchmem
```

### 3. Conformance Tests

The conformance suite provides multi-level validation across four test categories.

#### Test Categories

**Spec Validation Tests:**

- `SpecValidation_ValidPricingSpec` - Schema-compliant response validation
- `SpecValidation_BillingModeEnum` - Valid billing mode enumeration
- `SpecValidation_RequiredFields` - Required field presence

**RPC Correctness Tests:**

- `RPCCorrectness_NameResponse` - Valid Name response format
- `RPCCorrectness_SupportsValidation` - Supports validation behavior
- `RPCCorrectness_ErrorHandling` - Proper gRPC error codes
- `RPCCorrectness_TimeRangeValidation` - Time range validation
- `RPCCorrectness_ConsistentResponses` - Response consistency
- `RPCCorrectness_GetRecommendations_Pagination` - Pagination token handling
- `RPCCorrectness_GetRecommendations_Filtering` - Filter criteria validation
- `RPCCorrectness_GetRecommendations_ActionDetails` - Action type details

**Performance Tests:**

- `Performance_NameLatency` - Name RPC latency threshold
- `Performance_SupportsLatency` - Supports RPC latency threshold
- `Performance_GetProjectedCostLatency` - GetProjectedCost latency
- `Performance_GetPricingSpecLatency` - GetPricingSpec latency
- `Performance_GetRecommendationsLatency` - GetRecommendations latency
- `Performance_GetRecommendations_LargeResultSet` - Large result set performance
- `Performance_BaselineVariance` - Variance within 10% (SC-003)

**Concurrency Tests:**

- `Concurrency_ParallelRequests` - Thread-safe parallel execution
- `Concurrency_ConsistentUnderLoad` - Consistent responses under load
- `Concurrency_NoRaceConditions` - Race condition detection

#### Conformance Levels

**Basic Conformance** (Required for all plugins):

- All Spec Validation tests
- Core RPC Correctness tests

**Standard Conformance** (Production-ready):

- All Basic tests
- All RPC Correctness tests
- Standard Performance tests (10 parallel requests)

**Advanced Conformance** (High-performance):

- All Standard tests
- All Performance tests (strict latency thresholds)
- All Concurrency tests (50+ parallel requests)

**Run conformance tests:**

```go
func TestConformance(t *testing.T) {
    plugin := &MyPluginImpl{}

    // Choose conformance level
    result, err := plugintesting.RunBasicConformance(plugin)
    // result, err := plugintesting.RunStandardConformance(plugin)
    // result, err := plugintesting.RunAdvancedConformance(plugin)
    if err != nil {
        t.Fatalf("conformance tests failed to run: %v", err)
    }

    plugintesting.PrintReportTo(result, os.Stdout)

    if !result.Passed() {
        t.Fatalf("Plugin failed conformance: %d/%d tests failed",
            result.Summary.Failed, result.Summary.Total)
    }
}
```

**Custom test selection:**

```go
func TestCustomConformance(t *testing.T) {
    plugin := &MyPluginImpl{}

    suite := plugintesting.NewConformanceSuite()

    // Register only specific categories
    plugintesting.RegisterSpecValidationTests(suite)
    plugintesting.RegisterRPCCorrectnessTests(suite)
    // Skip performance and concurrency tests

    result := suite.Run(plugin, plugintesting.ConformanceLevelBasic)
    plugintesting.PrintReportTo(result, os.Stdout)
}
```

## Test Requirements by RPC Method

### Name()

**Basic Requirements:**

- MUST return non-empty plugin name
- Name MUST be ≤ 100 characters
- MUST respond within 100ms (advanced conformance)

**Validation:**

```go
func ValidateNameResponse(response *pbc.NameResponse) error
```

### Supports()

**Basic Requirements:**

- MUST handle nil resource gracefully (return false or error)
- MUST provide reason when resource not supported
- MUST be consistent (same input → same output)

**Standard Requirements:**

- SHOULD respond within 50ms for supported providers
- MUST validate all resource descriptor fields

### GetActualCost()

**Basic Requirements:**

- MUST validate time range (end > start)
- MUST return valid cost data structure
- Cost values MUST be non-negative
- MUST include source in results

**Standard Requirements:**

- SHOULD handle missing data gracefully (return empty or appropriate error)
- SHOULD support reasonable time ranges (up to 30 days)
- MUST provide meaningful error messages

**Advanced Requirements:**

- SHOULD handle large datasets efficiently (30+ days)
- MUST respond within 10 seconds for large queries

### GetProjectedCost()

**Basic Requirements:**

- MUST return valid pricing data
- Unit price MUST be non-negative
- Currency MUST be valid 3-character ISO code
- MUST reject unsupported resources

**Standard Requirements:**

- SHOULD be consistent across calls
- SHOULD provide cost per month calculation
- MUST include billing detail information

### GetPricingSpec()

**Basic Requirements:**

- MUST return complete pricing specification
- MUST include all required fields (provider, resource_type, billing_mode, rate_per_unit, currency)
- Rate per unit MUST be non-negative
- Currency MUST be valid 3-character ISO code

**Standard Requirements:**

- SHOULD include metric hints for cost calculation
- SHOULD be consistent across calls
- SHOULD provide meaningful descriptions

### GetRecommendations()

**Basic Requirements:**

- MUST return empty list (not error) if plugin doesn't support recommendations
- Recommendations MUST include valid category, action_type, and resource info
- Impact MUST include estimated_savings and currency
- Summary MUST accurately reflect aggregated statistics

**Standard Requirements:**

- SHOULD support pagination with page_size and page_token
- SHOULD support filtering by provider, category, and action_type
- MUST respond within 500ms for typical result sets (< 100 recommendations)

**Advanced Requirements:**

- MUST handle large result sets efficiently (1000+ recommendations)
- Pagination MUST be stable and consistent across pages
- SHOULD respond within 100ms for paginated requests

**Usage Examples:**

```go
// Basic GetRecommendations call
func TestGetRecommendations(t *testing.T) {
    plugin := &MyPluginImpl{}
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(t)
    defer harness.Stop()

    client := harness.Client()
    ctx := context.Background()

    // Get all recommendations
    resp, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{})
    if err != nil {
        t.Fatalf("GetRecommendations() failed: %v", err)
    }
    if resp == nil {
        t.Fatal("GetRecommendations() returned nil response")
    }

    // Validate response structure
    if resp.GetSummary() == nil {
        t.Error("Summary should not be nil")
    }
    if resp.GetSummary().GetTotalRecommendations() != int32(len(resp.GetRecommendations())) {
        t.Error("Summary count mismatch")
    }
}

// Test with filtering
func TestGetRecommendationsFiltered(t *testing.T) {
    plugin := plugintesting.NewMockPlugin()
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(t)
    defer harness.Stop()

    client := harness.Client()
    ctx := context.Background()

    // Filter by provider and category
    resp, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            Provider: "aws",
            Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
        },
    })

    // The following examples demonstrate various filter configurations.
    // Error handling shown once at the end for brevity.

    // P0: Filter by priority and minimum savings threshold
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            Priority:            pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_HIGH,
            MinEstimatedSavings: 100.0, // Only recommendations saving $100+
            SortBy:              pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
            SortOrder:           pbc.SortOrder_SORT_ORDER_DESC,
        },
    })

    // P0: Filter by source for multi-source environments
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            Source:   "kubecost",
            Provider: "kubernetes",
        },
    })

    // P1: Filter by account for enterprise multi-account setups
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            AccountId: "123456789012",
            Provider:  "aws",
            SortBy:    pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_PRIORITY,
        },
    })

    // P2: Filter for automation pipelines - high confidence, recent only
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            MinConfidenceScore: 0.8,
            MaxAgeDays:         7, // Recommendations from last 7 days
        },
    })

    // P2: Get recommendations for a specific resource
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            ResourceId: "i-0abc123def456789",
        },
    })

    // Combined: SKU-based filtering with savings threshold
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        Filter: &pbc.RecommendationFilter{
            Provider:            "aws",
            ResourceType:        "ec2",
            Sku:                 "t2.medium",
            Tags:                map[string]string{"env": "production"},
            MinEstimatedSavings: 50.0,
            Category:            pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
        },
    })
    if err != nil {
        t.Fatalf("GetRecommendations() failed: %v", err)
    }
    if resp == nil {
        t.Fatal("GetRecommendations() returned nil response")
    }

    // Verify all recommendations match filter
    for _, rec := range resp.GetRecommendations() {
        if rec == nil {
            t.Error("Recommendation should not be nil")
            continue
        }
        if rec.GetResource() == nil {
            t.Error("Resource should not be nil")
            continue
        }
        if rec.GetResource().GetProvider() != "aws" {
            t.Errorf("Expected aws provider, got %s", rec.GetResource().GetProvider())
        }
        if rec.GetCategory() != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST {
            t.Errorf("Expected COST category, got %s", rec.GetCategory())
        }
    }
}

// Test pagination
func TestGetRecommendationsPagination(t *testing.T) {
    client := harness.Client()
    ctx := context.Background()

    // First page
    page1, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        PageSize: 5,
    })
    if err != nil {
        t.Fatalf("GetRecommendations() page 1 failed: %v", err)
    }

    if len(page1.GetRecommendations()) > 5 {
        t.Errorf("Expected max 5 recommendations, got %d", len(page1.GetRecommendations()))
    }

    // Get next page if available
    if page1.GetNextPageToken() != "" {
        page2, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
            PageSize:  5,
            PageToken: page1.GetNextPageToken(),
        })
        if err != nil {
            t.Fatalf("GetRecommendations() page 2 failed: %v", err)
        }

        // Verify no duplicate IDs between pages
        page1IDs := make(map[string]bool)
        for _, rec := range page1.GetRecommendations() {
            page1IDs[rec.GetId()] = true
        }
        for _, rec := range page2.GetRecommendations() {
            if page1IDs[rec.GetId()] {
                t.Errorf("Duplicate recommendation ID across pages: %s", rec.GetId())
            }
        }
    }
}

// Test target_resources filtering (resource-scoped recommendations)
func TestGetRecommendationsTargetResources(t *testing.T) {
    client := harness.Client()
    ctx := context.Background()

    // Stack-scoped: Get recommendations for specific Pulumi stack resources
    resp, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        TargetResources: []*pbc.ResourceDescriptor{
            {Provider: "aws", ResourceType: "ec2", Sku: "t3.large", Region: "us-east-1"},
            {Provider: "aws", ResourceType: "rds", Sku: "db.r5.large", Region: "us-east-1"},
        },
    })
    if err != nil {
        t.Fatalf("GetRecommendations() failed: %v", err)
    }
    // Only recommendations matching these resources are returned

    // Pre-deployment: Analyze proposed resources before creation
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        TargetResources: []*pbc.ResourceDescriptor{
            {
                Provider:     "aws",
                ResourceType: "ec2",
                Sku:          "m5.4xlarge",  // SKU from Pulumi preview
                Region:       "us-west-2",
                Tags:         map[string]string{"env": "production"},
            },
        },
    })
    // Returns SKU-specific recommendations (e.g., "consider m5.2xlarge")

    // Batch + Filter: Combined target_resources AND filter (AND logic)
    resp, err = client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
        TargetResources: []*pbc.ResourceDescriptor{
            {Provider: "aws", ResourceType: "ec2"},
            {Provider: "aws", ResourceType: "rds"},
        },
        Filter: &pbc.RecommendationFilter{
            Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
            Priority: pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_HIGH,
        },
    })
    // Returns high-priority COST recommendations for ec2/rds resources only

    _ = resp // Use resp
}
```

**Target Resources Validation:**

```go
// Validate target_resources before request (max 100 resources)
targets := []*pbc.ResourceDescriptor{
    {Provider: "aws", ResourceType: "ec2"},
}
if err := plugintesting.ValidateTargetResources(targets); err != nil {
    t.Fatalf("Invalid target_resources: %v", err)
}
```

**Mock Plugin Configuration for Recommendations:**

```go
// Configure mock with recommendations
mock := plugintesting.NewMockPlugin()

// Default mock includes 12 sample recommendations
// Clear them for empty plugin test
mock.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
    Recommendations: []*pbc.Recommendation{}, // Empty
})

// Or configure custom recommendations
mock.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
    Recommendations: []*pbc.Recommendation{
        {
            Id:         "rec-001",
            Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
            ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
            Resource: &pbc.ResourceRecommendationInfo{
                Id:       "i-abc123",
                Provider: "aws",
            },
            Impact: &pbc.RecommendationImpact{
                EstimatedSavings: 100.0,
                Currency:         "USD",
            },
        },
    },
})

// Configure error behavior
mock.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
    ShouldError:  true,
    ErrorMessage: "mock recommendations error",
})
```

**Validating Recommendation Responses:**

```go
// Validate individual recommendations
if resp == nil {
    t.Fatal("Response cannot be nil")
}
if resp.GetSummary() == nil {
    t.Error("Summary cannot be nil in a valid response")
}
for _, rec := range resp.GetRecommendations() {
    if rec == nil {
        t.Error("Recommendation cannot be nil")
        continue
    }
    // Required fields
    if rec.GetId() == "" {
        t.Error("Recommendation ID is required")
    }
    if rec.GetCategory() == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
        t.Error("Category must be specified")
    }
    if rec.GetResource() == nil {
        t.Error("Resource info is required")
    }

    // Validate impact
    if rec.GetImpact() != nil {
        if rec.GetImpact().GetCurrency() == "" {
            t.Error("Currency is required when impact provided")
        }
        if len(rec.GetImpact().GetCurrency()) != 3 {
            t.Error("Currency must be 3-character ISO code")
        }
    }

    // Validate action details based on type
    switch rec.GetActionType() {
    case pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE:
        if rec.GetRightsize() == nil {
            t.Error("Rightsize action must have rightsize details")
        }
    case pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_ADJUST_REQUESTS:
        if rec.GetKubernetes() == nil {
            t.Error("Adjust requests must have kubernetes details")
        }
    }
}

// Validate summary calculations
var calcSavings float64
for _, rec := range resp.GetRecommendations() {
    if rec == nil {
        continue
    }
    if rec.GetImpact() != nil {
        calcSavings += rec.GetImpact().GetEstimatedSavings()
    }
}
if resp.GetSummary() == nil {
    t.Error("Cannot validate summary calculations without summary")
} else if math.Abs(calcSavings-resp.GetSummary().GetTotalEstimatedSavings()) > 0.01 {
    t.Errorf("Summary savings mismatch: calculated %.2f, reported %.2f",
        calcSavings, resp.GetSummary().GetTotalEstimatedSavings())
}
```

## Error Handling Requirements

### Expected Error Codes

- `InvalidArgument`: Invalid input parameters
- `NotFound`: Resource or data not found
- `Unavailable`: Service temporarily unavailable
- `PermissionDenied`: Access denied to resource/data
- `ResourceExhausted`: Rate limit or quota exceeded
- `Internal`: Internal service error

### Error Testing

```go
// Test error conditions
plugin.ShouldErrorOnName = true
_, err := client.Name(ctx, &pbc.NameRequest{})
if err == nil {
    t.Error("Expected error")
}

// Check error code
st, ok := status.FromError(err)
if !ok {
    t.Error("Expected gRPC status error")
}
if st.Code() != codes.Internal {
    t.Errorf("Expected Internal error, got %v", st.Code())
}
```

## Performance Requirements

### Latency Baselines

The conformance suite validates against these latency thresholds:

| Method                         | Standard | Advanced |
| ------------------------------ | -------- | -------- |
| Name()                         | 100ms    | 50ms     |
| Supports()                     | 50ms     | 25ms     |
| GetProjectedCost()             | 200ms    | 100ms    |
| GetPricingSpec()               | 200ms    | 100ms    |
| GetActualCost (24h)            | 2000ms   | 1000ms   |
| GetActualCost (30d)            | N/A      | 10000ms  |
| GetRecommendations (<100)      | 500ms    | 200ms    |
| GetRecommendations (paginated) | N/A      | 100ms    |

### Variance Requirements (SC-003)

- Advanced conformance requires benchmark variance ≤ 10% from baseline
- Measured over multiple iterations (default: 50 iterations)
- Helps ensure consistent performance under load

### Concurrency Requirements

| Level    | Parallel Requests | Requirements                    |
| -------- | ----------------- | ------------------------------- |
| Standard | 10                | Thread-safe, no race conditions |
| Advanced | 50                | Consistent responses under load |

### Memory Requirements

- Should not consume excessive memory for normal operations
- Must handle large datasets (30+ days) without memory issues
- Allocations tracked via benchmark tests

## CI/CD Integration

### GitHub Actions

Add to your `.github/workflows/test.yml`:

```yaml
name: Plugin Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Run Integration Tests
        run: go test -v ./...

      - name: Run Benchmarks
        run: go test -bench=. -benchmem

      - name: Run Conformance Tests
        run: go test -v -run TestConformance
```

### Makefile Integration

```makefile
.PHONY: test-integration test-conformance test-performance

test-integration:
 go test -v -run TestBasic

test-conformance:
 go test -v -run TestConformance

test-performance:
 go test -bench=. -benchmem

test-all: test-integration test-conformance test-performance
```

## Resource Identifier Fields

> **Import Note**: Examples in this section use the following import aliases:
>
> ```go
> import (
>     pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
>     "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
> )
> ```

The `ResourceDescriptor` message supports two optional fields for resource identification:

### ID Field (Batch Correlation)

The `id` field enables request/response correlation in batch operations. When clients submit
multiple resources for recommendations, they can assign unique IDs to each resource and use
those IDs to match responses back to their original requests.

```go
// Create descriptors with unique IDs for batch correlation
descriptors := []*pbc.ResourceDescriptor{
    pluginsdk.NewResourceDescriptor(
        "aws", "ec2",
        pluginsdk.WithID("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web"),
        pluginsdk.WithSKU("t3.micro"),
        pluginsdk.WithRegion("us-east-1"),
    ),
    pluginsdk.NewResourceDescriptor(
        "aws", "ec2",
        pluginsdk.WithID("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::api"),
        pluginsdk.WithSKU("t3.small"),
        pluginsdk.WithRegion("us-east-1"),
    ),
}

// Submit batch request
req := &pbc.GetRecommendationsRequest{TargetResources: descriptors}
resp, _ := client.GetRecommendations(ctx, req)

// Correlate responses by ID
for _, rec := range resp.GetRecommendations() {
    resourceID := rec.GetResource().GetId()  // Matches input ID
    // Match back to original request...
}
```

**Key points:**

- IDs are opaque strings - plugins MUST NOT validate or transform them
- Common formats: Pulumi URNs, UUIDs, application-specific identifiers
- Empty ID is valid and maintains backward compatibility

### ARN Field (Exact Matching)

The `arn` field enables exact resource lookup using canonical cloud identifiers.
When provided, plugins can use this for precise matching instead of fuzzy
type/sku/region/tags matching.

```go
// AWS resource with ARN for exact matching
desc := pluginsdk.NewResourceDescriptor(
    "aws", "ec2",
    pluginsdk.WithID("batch-001"),  // For correlation
    pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),  // For exact matching
)

// Azure resource with Resource ID
desc := pluginsdk.NewResourceDescriptor(
    "azure", "virtualMachines",
    pluginsdk.WithARN("/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"),
)

// GCP resource with Full Resource Name
desc := pluginsdk.NewResourceDescriptor(
    "gcp", "compute_engine",
    pluginsdk.WithARN("//compute.googleapis.com/projects/myproj/zones/us-central1-a/instances/vm1"),
)

// Kubernetes resource
desc := pluginsdk.NewResourceDescriptor(
    "kubernetes", "deployment",
    pluginsdk.WithARN("prod-cluster/default/Deployment/nginx"),
)
```

**Supported ARN formats:**

| Provider   | Format                                                           |
| ---------- | ---------------------------------------------------------------- |
| AWS        | `arn:aws:service:region:account:resource`                        |
| Azure      | `/subscriptions/{sub}/resourceGroups/{rg}/providers/...`         |
| GCP        | `//service.googleapis.com/projects/{project}/zones/{zone}/...`   |
| Kubernetes | `{cluster}/{namespace}/{kind}/{name}` or UID                     |
| Cloudflare | `{zone-id}/{resource-type}/{resource-id}`                        |

**Matching behavior:**

- ARN provided and valid → Use for exact resource lookup
- ARN empty or invalid → Fall back to type/sku/region/tags matching
- ARN format unrecognized → Log warning, use fallback matching

## Best Practices

### Plugin Implementation

1. **Implement all RPC methods** even if you return "not supported" errors
2. **Validate inputs** and return appropriate error codes
3. **Handle concurrency** safely - avoid shared mutable state
4. **Use timeouts** for external service calls
5. **Implement caching** for frequently requested data
6. **Provide meaningful error messages** to help users debug issues
7. **Preserve ID fields** - Copy `ResourceDescriptor.Id` to response fields for correlation

### Testing Implementation

1. **Test all supported providers and resource types**
2. **Test error conditions** and edge cases
3. **Test with realistic data sizes** (don't just test with 1-2 data points)
4. **Test concurrency** if your plugin will handle concurrent requests
5. **Test performance** under expected load

### Debugging Tips

1. **Use the mock plugin** to understand expected behavior
2. **Check validation errors** - they provide specific failure details
3. **Run conformance tests** to identify compliance issues
4. **Use performance tests** to identify bottlenecks
5. **Enable debug logging** in your plugin during testing

## Example Test Suite

```go
package myplugin_test

import (
    "os"
    "testing"

    "github.com/myplugin/internal"
    plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestMyPlugin(t *testing.T) {
    plugin := internal.NewMyPlugin()

    // Create test harness
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(t)
    defer harness.Stop()

    // Test individual methods
    t.Run("Name", func(t *testing.T) {
        resp, err := harness.Client().Name(context.Background(), &pbc.NameRequest{})
        if err != nil {
            t.Fatalf("Name() failed: %v", err)
        }
        if err := plugintesting.ValidateNameResponse(resp); err != nil {
            t.Errorf("Invalid response: %v", err)
        }
    })
}

func TestMyPluginConformance(t *testing.T) {
    plugin := internal.NewMyPlugin()

    // Run full conformance suite at Standard level
    result, err := plugintesting.RunStandardConformance(plugin)
    if err != nil {
        t.Fatalf("conformance tests failed to run: %v", err)
    }

    // Print detailed report
    plugintesting.PrintReportTo(result, os.Stdout)

    if !result.Passed() {
        t.Errorf("Plugin failed conformance: %d/%d tests failed",
            result.Summary.Failed, result.Summary.Total)
    }
}

func TestMyPluginCustomConformance(t *testing.T) {
    plugin := internal.NewMyPlugin()

    // Create custom suite with specific tests
    suite := plugintesting.NewConformanceSuite()
    plugintesting.RegisterSpecValidationTests(suite)
    plugintesting.RegisterRPCCorrectnessTests(suite)

    result, err := suite.Run(plugin, plugintesting.ConformanceLevelBasic)
    if err != nil {
        t.Fatalf("conformance tests failed to run: %v", err)
    }

    if !result.Passed() {
        t.Errorf("Plugin failed: %d/%d tests failed",
            result.Summary.Failed, result.Summary.Total)
    }
}

func BenchmarkMyPlugin(b *testing.B) {
    plugin := internal.NewMyPlugin()
    harness := plugintesting.NewTestHarness(plugin)
    harness.Start(&testing.T{})
    defer harness.Stop()

    client := harness.Client()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = client.Name(ctx, &pbc.NameRequest{})
    }
}
```

## Conformance Report Example

Running conformance tests produces a detailed report:

```text
================================================================================
                     FinFocus Plugin Conformance Report
================================================================================

Plugin:     my-cost-plugin
Level:      Standard
Duration:   1.234s

--------------------------------------------------------------------------------
                                Test Results
--------------------------------------------------------------------------------

Category: spec_validation
  ✓ SpecValidation_ValidPricingSpec                     [PASS]     12ms
  ✓ SpecValidation_BillingModeEnum                      [PASS]      8ms
  ✓ SpecValidation_RequiredFields                       [PASS]      5ms

Category: rpc_correctness
  ✓ RPCCorrectness_NameResponse                         [PASS]     15ms
  ✓ RPCCorrectness_SupportsValidation                   [PASS]     22ms
  ✓ RPCCorrectness_ErrorHandling                        [PASS]     18ms

Category: performance
  ✓ Performance_NameLatency                             [PASS]     45ms
  ✓ Performance_SupportsLatency                         [PASS]     32ms

--------------------------------------------------------------------------------
                                   Summary
--------------------------------------------------------------------------------

Total:    8 tests
Passed:   8
Failed:   0
Skipped:  0

Result:   PASS - Plugin conforms to Standard level
================================================================================
```

This testing framework ensures your plugin meets the FinFocus specification requirements and
provides a reliable, performant cost source for users.
