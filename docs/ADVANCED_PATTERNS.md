# Advanced Implementation Patterns

This guide covers complex plugin scenarios including tiered pricing models, multi-provider
mapping, and advanced usage of the SDK packages.

## Table of Contents

- [Tiered Pricing Implementation](#tiered-pricing-implementation)
- [Multi-Provider Plugin Architecture](#multi-provider-plugin-architecture)
- [Advanced Mapping Package Usage](#advanced-mapping-package-usage)
- [Plugin Orchestration with FallbackHint](#plugin-orchestration-with-fallbackhint)
- [Cost Aggregation Strategies](#cost-aggregation-strategies)
- [Performance Optimization Patterns](#performance-optimization-patterns)

## Tiered Pricing Implementation

Cloud providers often use tiered pricing where rates decrease as usage increases. The
FinFocus schema supports this via the `pricing_tiers` field.

### Understanding the Tier Structure

```json
{
  "pricing_tiers": [
    {
      "min_quantity": 0,
      "max_quantity": 50000,
      "rate_per_unit": 0.023,
      "description": "First 50 TB / Month"
    },
    {
      "min_quantity": 50000,
      "max_quantity": 450000,
      "rate_per_unit": 0.022,
      "description": "Next 400 TB / Month"
    },
    {
      "min_quantity": 450000,
      "max_quantity": 0,
      "rate_per_unit": 0.021,
      "description": "Over 450 TB / Month"
    }
  ]
}
```

**Key rules:**

- Tiers are evaluated in order from lowest to highest `min_quantity`
- `max_quantity: 0` indicates unlimited (no upper bound)
- Usage in each tier is calculated as `min(usage, max) - min`

### Go Implementation: Tiered Cost Calculator

```go
package pricing

// PricingTier represents a single pricing tier.
type PricingTier struct {
    MinQuantity float64
    MaxQuantity float64 // 0 means unlimited
    RatePerUnit float64
    Description string
}

// TieredPricingCalculator calculates costs across volume-based pricing tiers.
type TieredPricingCalculator struct {
    tiers    []PricingTier
    currency string
}

// NewTieredPricingCalculator creates a calculator with the given tiers.
// Tiers should be provided in ascending order by MinQuantity.
func NewTieredPricingCalculator(tiers []PricingTier, currency string) *TieredPricingCalculator {
    return &TieredPricingCalculator{
        tiers:    tiers,
        currency: currency,
    }
}

// CalculateCost computes the total cost for a given usage quantity.
// Returns the total cost and a breakdown by tier.
func (c *TieredPricingCalculator) CalculateCost(usage float64) (float64, []TierBreakdown) {
    var totalCost float64
    var breakdown []TierBreakdown
    remainingUsage := usage

    for _, tier := range c.tiers {
        if remainingUsage <= 0 {
            break
        }

        // Calculate usage within this tier
        tierMin := tier.MinQuantity
        tierMax := tier.MaxQuantity
        if tierMax == 0 {
            tierMax = usage + 1 // Unlimited tier
        }

        // Skip if usage hasn't reached this tier yet
        if usage <= tierMin {
            continue
        }

        // Calculate quantity in this tier
        usageInTier := min(usage, tierMax) - tierMin
        if usageInTier <= 0 {
            continue
        }

        tierCost := usageInTier * tier.RatePerUnit
        totalCost += tierCost
        breakdown = append(breakdown, TierBreakdown{
            Tier:        tier,
            UsageInTier: usageInTier,
            TierCost:    tierCost,
        })

        remainingUsage -= usageInTier
    }

    return totalCost, breakdown
}

// TierBreakdown provides details about cost in each tier.
type TierBreakdown struct {
    Tier        PricingTier
    UsageInTier float64
    TierCost    float64
}

// Example usage:
//
//   tiers := []PricingTier{
//       {MinQuantity: 0, MaxQuantity: 50000, RatePerUnit: 0.023},
//       {MinQuantity: 50000, MaxQuantity: 450000, RatePerUnit: 0.022},
//       {MinQuantity: 450000, MaxQuantity: 0, RatePerUnit: 0.021},
//   }
//   calc := NewTieredPricingCalculator(tiers, "USD")
//   cost, breakdown := calc.CalculateCost(100000) // 100 TB
//   // cost = (50000 * 0.023) + (50000 * 0.022) = $2,250
```

### Tiered Pricing in GetPricingSpec

```go
func (p *S3Plugin) GetPricingSpec(
    ctx context.Context,
    req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
    // Check for context cancellation early
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // Build tiered pricing spec
    spec := map[string]interface{}{
        "provider":      "aws",
        "resource_type": "s3",
        "billing_mode":  "tiered",
        "rate_per_unit": 0.023, // Base rate (first tier)
        "currency":      "USD",
        "pricing_tiers": []map[string]interface{}{
            {
                "min_quantity":  0,
                "max_quantity":  50000,
                "rate_per_unit": 0.023,
                "description":   "First 50 TB / Month",
            },
            {
                "min_quantity":  50000,
                "max_quantity":  450000,
                "rate_per_unit": 0.022,
                "description":   "Next 400 TB / Month",
            },
            {
                "min_quantity":  450000,
                "max_quantity":  0,
                "rate_per_unit": 0.021,
                "description":   "Over 450 TB / Month",
            },
        },
    }

    specJSON, err := json.Marshal(spec)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to marshal pricing spec: %v", err)
    }
    return &pbc.GetPricingSpecResponse{
        Spec: string(specJSON),
    }, nil
}
```

## Multi-Provider Plugin Architecture

Some plugins need to support multiple cloud providers. This section covers patterns
for cleanly handling provider-specific logic.

### Strategy Pattern for Provider Handling

```go
package multiprovider

import (
    "context"
    "fmt"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ProviderHandler defines the interface for provider-specific cost calculations.
type ProviderHandler interface {
    // ExtractResourceInfo extracts SKU, region from resource properties.
    ExtractResourceInfo(properties map[string]string) (sku, region string)

    // GetProjectedCost calculates projected cost for this provider.
    GetProjectedCost(ctx context.Context, resource *pbc.ResourceDescriptor) (float64, error)

    // GetActualCost retrieves actual cost from provider's billing API.
    GetActualCost(ctx context.Context, resourceID string, timeRange TimeRange) (float64, error)

    // SupportedResourceTypes returns the resource types this handler supports.
    SupportedResourceTypes() []string
}

// AWSHandler implements ProviderHandler for AWS resources.
type AWSHandler struct {
    pricingClient AWSPricingClient
}

func (h *AWSHandler) ExtractResourceInfo(props map[string]string) (sku, region string) {
    return mapping.ExtractAWSSKU(props), mapping.ExtractAWSRegion(props)
}

func (h *AWSHandler) SupportedResourceTypes() []string {
    return []string{
        "aws:ec2/instance:Instance",
        "aws:rds/instance:Instance",
        "aws:s3/bucket:Bucket",
        "aws:lambda/function:Function",
    }
}

// AzureHandler implements ProviderHandler for Azure resources.
type AzureHandler struct {
    pricingClient AzurePricingClient
}

func (h *AzureHandler) ExtractResourceInfo(props map[string]string) (sku, region string) {
    return mapping.ExtractAzureSKU(props), mapping.ExtractAzureRegion(props)
}

func (h *AzureHandler) SupportedResourceTypes() []string {
    return []string{
        "azure:compute/virtualMachine:VirtualMachine",
        "azure:sql/database:Database",
        "azure:storage/account:Account",
    }
}

// GCPHandler implements ProviderHandler for GCP resources.
type GCPHandler struct {
    pricingClient GCPPricingClient
}

func (h *GCPHandler) ExtractResourceInfo(props map[string]string) (sku, region string) {
    return mapping.ExtractGCPSKU(props), mapping.ExtractGCPRegion(props)
}

func (h *GCPHandler) SupportedResourceTypes() []string {
    return []string{
        "gcp:compute/instance:Instance",
        "gcp:sql/databaseInstance:DatabaseInstance",
        "gcp:storage/bucket:Bucket",
    }
}

// MultiProviderPlugin coordinates cost calculations across providers.
type MultiProviderPlugin struct {
    pbc.UnimplementedCostSourceServiceServer
    handlers map[string]ProviderHandler // provider name -> handler
    matcher  *pluginsdk.ResourceMatcher
}

// NewMultiProviderPlugin creates a plugin supporting multiple cloud providers.
func NewMultiProviderPlugin() *MultiProviderPlugin {
    p := &MultiProviderPlugin{
        handlers: make(map[string]ProviderHandler),
        matcher:  pluginsdk.NewResourceMatcher(),
    }

    // Register provider handlers
    p.RegisterHandler("aws", &AWSHandler{})
    p.RegisterHandler("azure", &AzureHandler{})
    p.RegisterHandler("gcp", &GCPHandler{})

    return p
}

// RegisterHandler adds a provider handler and updates the resource matcher.
func (p *MultiProviderPlugin) RegisterHandler(provider string, handler ProviderHandler) {
    p.handlers[provider] = handler
    p.matcher.AddProvider(provider)
    for _, rt := range handler.SupportedResourceTypes() {
        p.matcher.AddResourceType(rt)
    }
}

// Supports checks if this plugin handles the given resource.
func (p *MultiProviderPlugin) Supports(
    ctx context.Context,
    req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
    return &pbc.SupportsResponse{
        Supported: p.matcher.Supports(req.GetResource()),
    }, nil
}

// GetProjectedCost routes to the appropriate provider handler.
func (p *MultiProviderPlugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    resource := req.GetResource()
    handler, ok := p.handlers[resource.GetProvider()]
    if !ok {
        return nil, pluginsdk.NotSupportedError(resource)
    }

    cost, err := handler.GetProjectedCost(ctx, resource)
    if err != nil {
        return nil, err
    }

    calc := pluginsdk.NewCostCalculator()
    return calc.CreateProjectedCostResponse("USD", cost, "Multi-provider cost"), nil
}
```

### Provider-Specific Property Extraction

When dealing with resources from multiple providers, use the mapping package's
provider-specific extractors:

```go
func (p *MultiProviderPlugin) extractResourceDetails(
    resource *pbc.ResourceDescriptor,
) (sku, region string, err error) {
    props := resource.GetProperties()
    provider := resource.GetProvider()

    switch provider {
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
        // Use generic extractors for custom providers
        sku = mapping.ExtractSKU(props, "sku", "type", "size")
        region = mapping.ExtractRegion(props, "region", "location", "zone")
    }

    if sku == "" && region == "" {
        return "", "", fmt.Errorf("could not extract SKU or region for %s resource", provider)
    }

    return sku, region, nil
}
```

## Advanced Mapping Package Usage

The `mapping` package provides provider-specific property extraction. This section
covers advanced patterns beyond basic usage.

### Custom Property Key Priority

Use the generic extractors with custom key priorities for non-standard resources:

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"

// For a custom resource with non-standard property names
props := map[string]string{
    "machineSize":     "large",       // Custom key name
    "deploymentZone":  "us-west-2a",  // Custom key name
}

// Define custom key priority (checked in order)
sku := mapping.ExtractSKU(props,
    "machineSize",      // Custom key (highest priority)
    "instanceType",     // AWS-style fallback
    "vmSize",           // Azure-style fallback
    "machineType",      // GCP-style fallback
)

region := mapping.ExtractRegion(props,
    "deploymentZone",   // Custom key
    "region",           // Standard fallback
    "location",         // Azure fallback
)
```

### Composing Extractors for Complex Resources

Some resources require multiple extraction strategies:

```go
// ExtractComputeDetails extracts all pricing-relevant details from compute resources.
func ExtractComputeDetails(provider string, props map[string]string) ComputeDetails {
    var details ComputeDetails

    switch provider {
    case "aws":
        details.SKU = mapping.ExtractAWSSKU(props)
        details.Region = mapping.ExtractAWSRegion(props)
        details.InstanceFamily = extractInstanceFamily(details.SKU) // Custom logic
        details.IsSpot = props["instanceLifecycle"] == "spot"

    case "azure":
        details.SKU = mapping.ExtractAzureSKU(props)
        details.Region = mapping.ExtractAzureRegion(props)
        details.IsSpot = props["priority"] == "Spot"

    case "gcp":
        details.SKU = mapping.ExtractGCPSKU(props)
        details.Region = mapping.ExtractGCPRegion(props)
        details.IsPreemptible = props["scheduling.preemptible"] == "true"
    }

    return details
}

type ComputeDetails struct {
    SKU            string
    Region         string
    InstanceFamily string
    IsSpot         bool
    IsPreemptible  bool
}

func extractInstanceFamily(sku string) string {
    // AWS instance types: t3.micro -> t3
    // Azure VM sizes: Standard_D2s_v3 -> D
    // GCP machine types: n1-standard-4 -> n1
    if sku == "" {
        return ""
    }

    // Simple extraction for AWS (before first dot)
    if idx := strings.Index(sku, "."); idx > 0 {
        return sku[:idx]
    }

    // Simple extraction for GCP (before first dash)
    if idx := strings.Index(sku, "-"); idx > 0 {
        return sku[:idx]
    }

    return sku
}
```

### Region Normalization

Different providers use different region naming conventions. Here's how to normalize:

```go
// RegionNormalizer maps provider-specific regions to canonical names.
type RegionNormalizer struct {
    mappings map[string]map[string]string // provider -> localName -> canonical
}

func NewRegionNormalizer() *RegionNormalizer {
    return &RegionNormalizer{
        mappings: map[string]map[string]string{
            "aws": {
                "us-east-1":      "us-east",
                "us-west-2":      "us-west",
                "eu-west-1":      "europe-west",
                "ap-northeast-1": "asia-northeast",
            },
            "azure": {
                "eastus":       "us-east",
                "westus2":      "us-west",
                "westeurope":   "europe-west",
                "japaneast":    "asia-northeast",
            },
            "gcp": {
                "us-east1":           "us-east",
                "us-west1":           "us-west",
                "europe-west1":       "europe-west",
                "asia-northeast1":    "asia-northeast",
            },
        },
    }
}

func (n *RegionNormalizer) Normalize(provider, region string) string {
    if providerMappings, ok := n.mappings[provider]; ok {
        if canonical, ok := providerMappings[region]; ok {
            return canonical
        }
    }
    return region // Return as-is if no mapping found
}
```

## Plugin Orchestration with FallbackHint

The `FallbackHint` mechanism enables sophisticated plugin orchestration where
multiple plugins can contribute to cost calculations.

### Understanding FallbackHint Values

| Value                       | When to Use                                      |
|-----------------------------|--------------------------------------------------|
| `FALLBACK_HINT_UNSPECIFIED` | Default; backwards compatible "no fallback"      |
| `FALLBACK_HINT_NONE`        | Plugin has authoritative data                    |
| `FALLBACK_HINT_RECOMMENDED` | Plugin has no data; other plugins might          |
| `FALLBACK_HINT_REQUIRED`    | Plugin cannot handle; core MUST try alternatives |

### Implementing Graceful Fallback

```go
func (p *AWSCostPlugin) GetActualCost(
    ctx context.Context,
    req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
    resourceID := req.GetResourceId()

    // Check if this is a resource we can handle
    if !strings.HasPrefix(resourceID, "arn:aws:") {
        // Not an AWS resource - require fallback to another plugin
        return pluginsdk.NewActualCostResponse(
            pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_REQUIRED),
        ), nil
    }

    // Try to get cost data from AWS Cost Explorer
    costs, err := p.costExplorer.GetResourceCosts(ctx, resourceID, req.GetTimeRange())
    if err != nil {
        // API error - return gRPC error, not fallback
        return nil, status.Errorf(codes.Unavailable, "AWS API error: %v", err)
    }

    if len(costs) == 0 {
        // No billing data yet - recommend fallback to estimation plugin
        return pluginsdk.NewActualCostResponse(
            pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
        ), nil
    }

    // We have data - signal no fallback needed
    results := convertToResults(costs)
    return pluginsdk.NewActualCostResponse(
        pluginsdk.WithResults(results),
        pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
    ), nil
}
```

### Plugin Chain Pattern

For scenarios requiring multiple plugins in sequence:

```go
// PluginChain orchestrates multiple plugins with fallback handling.
type PluginChain struct {
    plugins []pbc.CostSourceServiceClient
}

func (c *PluginChain) GetActualCost(
    ctx context.Context,
    req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
    for i, plugin := range c.plugins {
        resp, err := plugin.GetActualCost(ctx, req)
        if err != nil {
            // Log and try next plugin
            log.Printf("Plugin %d failed: %v", i, err)
            continue
        }

        switch resp.GetFallbackHint() {
        case pbc.FallbackHint_FALLBACK_HINT_NONE:
            // Plugin has authoritative data
            return resp, nil

        case pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED:
            // Plugin suggests trying others, but we can use this if nothing better
            if len(resp.GetResults()) > 0 {
                // Save as fallback and continue
                continue
            }

        case pbc.FallbackHint_FALLBACK_HINT_REQUIRED:
            // Plugin explicitly cannot handle - must try next
            continue
        }
    }

    return nil, status.Error(codes.NotFound, "no plugin could provide cost data")
}
```

## Cost Diff Patterns: Comparing Resource States

The **cost diff** feature enables showing cost impact before applying infrastructure changes by
calling `GetProjectedCost` twice - once with old state properties and once with new state
properties. This section covers advanced patterns for handling this use case.

### Understanding the Two-Call Pattern

```go
// Baseline cost: Properties before the change (OldState.Inputs)
baselineReq := &pbc.GetProjectedCostRequest{
    Resource: &pbc.ResourceDescriptor{
        Provider:     "aws",
        ResourceType: "ec2",
        Tags: map[string]string{
            "instanceType": "t3.medium", // From old state
            // availabilityZone missing (was computed)
            // monitoring missing (was defaulted)
        },
    },
}
baselineResp, err := plugin.GetProjectedCost(ctx, baselineReq)
if err != nil {
    return fmt.Errorf("baseline cost lookup failed: %w", err)
}

// Modified cost: Properties after the change (NewState.Inputs)
modifiedReq := &pbc.GetProjectedCostRequest{
    Resource: &pbc.ResourceDescriptor{
        Provider:     "aws",
        ResourceType: "ec2",
        Tags: map[string]string{
            "instanceType":     "t3.large",  // Updated
            "availabilityZone": "us-east-1a", // Now present
            "monitoring":       "detailed",   // Explicitly set
        },
    },
}
modifiedResp, err := plugin.GetProjectedCost(ctx, modifiedReq)
if err != nil {
    return fmt.Errorf("modified cost lookup failed: %w", err)
}

// Validate responses before use
if baselineResp == nil || modifiedResp == nil {
    return fmt.Errorf("received nil response from GetProjectedCost")
}

// Cost impact
costDiff := modifiedResp.CostPerMonth - baselineResp.CostPerMonth
```

### Pattern 1: Graceful Degradation with Partial Properties

When critical properties are missing, signal the limitation rather than returning misleading results:

```go
func (p *Plugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    // Extract using mapping helpers
    sku := mapping.ExtractAWSSKU(req.Resource.Tags)
    region := mapping.ExtractAWSRegion(req.Resource.Tags)

    // Check for sparse property scenario
    if sku == "" {
        // Cannot estimate without SKU - signal explicitly
        return pluginsdk.NewProjectedCostResponse(
            pluginsdk.WithCostPerMonth(0),
            pluginsdk.WithCurrency("USD"),
            pluginsdk.WithBillingDetail("SPARSE_PROPERTIES:instanceType|Cannot estimate without instance type"),
        ), nil
    }

    // If region is missing, we can try a default but should document it
    regionDefaulted := false
    if region == "" {
        region = "us-east-1"
        regionDefaulted = true
    }

    // Proceed with available properties
    cost, err := p.pricingAPI.GetCost(ctx, sku, region)
    if err != nil {
        return nil, status.Errorf(codes.Unavailable, "pricing lookup failed: %v", err)
    }

    var billingDetail strings.Builder
    billingDetail.WriteString(fmt.Sprintf("SKU: %s, Region: %s", sku, region))
    if regionDefaulted {
        billingDetail.WriteString("; region defaulted to us-east-1")
    }

    return pluginsdk.NewProjectedCostResponse(
        pluginsdk.WithCostPerMonth(cost),
        pluginsdk.WithCurrency("USD"),
        pluginsdk.WithBillingDetail(billingDetail.String()),
    ), nil
}
```

### Pattern 2: Using Most-Specific Available Properties

Prefer more-specific properties when available, with graceful fallback:

```go
func extractStorageSize(tags map[string]string) (int, bool, []string) {
    var defaults []string

    // Try explicit storage specification
    if size, ok := tags["allocatedStorage"]; ok {
        return parseSize(size), true, defaults
    }

    // Try volume configuration
    if size, ok := tags["volumeSize"]; ok {
        return parseSize(size), true, defaults
    }

    // No explicit storage - check if this is a sparse property scenario
    // by verifying expected keys are present alongside instanceType
    if _, hasInstanceType := tags["instanceType"]; hasInstanceType {
        _, hasAMI := tags["ami"]
        _, hasRegion := tags["region"]
        if !hasAMI && !hasRegion {
            // Missing expected keys indicates sparse old state - return 0 with flag
            return 0, false, defaults
        }
    }

    // Use provider default
    defaultSize := 20 // GB
    defaults = append(defaults, fmt.Sprintf("allocatedStorage=%dGB", defaultSize))
    return defaultSize, true, defaults
}

func (p *Plugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    tags := req.Resource.Tags

    sku := mapping.ExtractAWSSKU(tags)
    if sku == "" {
        return signalSparseProperties("instanceType")
    }

    storageGB, hasStorage, defaults := extractStorageSize(tags)
    if !hasStorage {
        // Storage property missing and cannot default safely
        return signalSparseProperties("allocatedStorage")
    }

    // Include defaults in response
    var billingDetail strings.Builder
    billingDetail.WriteString(fmt.Sprintf("Instance: %s, Storage: %dGB", sku, storageGB))
    if len(defaults) > 0 {
        billingDetail.WriteString(fmt.Sprintf("; defaults: %s", strings.Join(defaults, ", ")))
    }

    cost := calculateCost(sku, storageGB)
    return pluginsdk.NewProjectedCostResponse(
        pluginsdk.WithCostPerMonth(cost),
        pluginsdk.WithCurrency("USD"),
        pluginsdk.WithBillingDetail(billingDetail.String()),
    ), nil
}
```

### Pattern 3: Differentiating Usage Properties from Configuration Properties

Handle missing **usage properties** (requests, data transfer) differently from **configuration
properties** (instance type, region):

```go
func (p *Plugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    tags := req.Resource.Tags

    // Configuration properties - MUST be present
    functionMemory := tags["memory"]
    if functionMemory == "" {
        return signalSparseProperties("memory")
    }

    // Usage properties - can estimate with reasonable defaults
    monthlyRequests := 1000000 // Default 1M requests
    var defaults []string

    if r, ok := tags["monthlyRequests"]; ok {
        monthlyRequests = parseInt(r)
    } else {
        defaults = append(defaults, "monthlyRequests=1000000")
    }

    avgDuration := 100.0 // Default 100ms
    if d, ok := tags["averageDuration"]; ok {
        avgDuration = parseFloat(d)
    } else {
        defaults = append(defaults, "averageDuration=100ms")
    }

    // Calculate with documented assumptions
    cost := calculateLambdaCost(functionMemory, monthlyRequests, avgDuration)

    var billingDetail strings.Builder
    billingDetail.WriteString(fmt.Sprintf(
        "Memory: %sMB, Requests: %d/mo, Duration: %.0fms",
        functionMemory, monthlyRequests, avgDuration,
    ))
    if len(defaults) > 0 {
        billingDetail.WriteString(fmt.Sprintf("; assumed: %s", strings.Join(defaults, ", ")))
    }

    return pluginsdk.NewProjectedCostResponse(
        pluginsdk.WithCostPerMonth(cost),
        pluginsdk.WithCurrency("USD"),
        pluginsdk.WithBillingDetail(billingDetail.String()),
    ), nil
}
```

### Anti-Pattern: Silent Defaulting

**DO NOT** silently apply defaults that produce misleading cost comparisons:

```go
// WRONG: Silent defaults make cost diffs meaningless
func (p *Plugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    tags := req.Resource.Tags

    // Silently default everything
    sku := tags["instanceType"]
    if sku == "" {
        sku = "t3.micro" // USER DOESN'T KNOW THIS HAPPENED
    }

    region := tags["region"]
    if region == "" {
        region = "us-east-1" // USER DOESN'T KNOW THIS HAPPENED
    }

    storageGB := parseInt(tags["storage"])
    if storageGB == 0 {
        storageGB = 20 // USER DOESN'T KNOW THIS HAPPENED
    }

    // This cost may be completely wrong but user sees a number
    cost := calculateCost(sku, region, storageGB)
    return pluginsdk.NewProjectedCostResponse(
        pluginsdk.WithCostPerMonth(cost),
        pluginsdk.WithCurrency("USD"),
    ), nil
}

// CORRECT: Document defaults or signal missing properties
func (p *Plugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    tags := req.Resource.Tags
    var defaults []string

    // Critical property missing - cannot estimate
    sku := tags["instanceType"]
    if sku == "" {
        return signalSparseProperties("instanceType")
    }

    // Optional property - use default but document it
    region := tags["region"]
    if region == "" {
        region = "us-east-1"
        defaults = append(defaults, "region=us-east-1")
    }

    storageGB := parseInt(tags["storage"])
    if storageGB == 0 {
        storageGB = 20
        defaults = append(defaults, "storage=20GB")
    }

    cost := calculateCost(sku, region, storageGB)

    var billingDetail strings.Builder
    billingDetail.WriteString(fmt.Sprintf("SKU: %s, Region: %s, Storage: %dGB", sku, region, storageGB))
    if len(defaults) > 0 {
        billingDetail.WriteString(fmt.Sprintf("; defaults: %s", strings.Join(defaults, ", ")))
    }

    return pluginsdk.NewProjectedCostResponse(
        pluginsdk.WithCostPerMonth(cost),
        pluginsdk.WithCurrency("USD"),
        pluginsdk.WithBillingDetail(billingDetail.String()),
    ), nil
}
```

### When NOT to Use FallbackHint for Sparse Properties

`FallbackHint` signals **"resource type not supported"** or **"no data available"**, not **"properties incomplete"**:

| Scenario | Correct Approach | Wrong Approach |
|---|---|---|
| SKU missing from old state | Return response with `SPARSE_PROPERTIES` billing detail | `FALLBACK_HINT_REQUIRED` |
| Region missing from old state | Use default region, document in billing detail | `FALLBACK_HINT_RECOMMENDED` |
| Usage property missing | Use reasonable default, document in billing detail | `FALLBACK_HINT_RECOMMENDED` |
| Unsupported resource type | `FALLBACK_HINT_REQUIRED` | Error or $0 response |
| No billing data available | `FALLBACK_HINT_RECOMMENDED` | Error or $0 response |

**Correct pattern for sparse properties:**

```go
if sku == "" {
    // NOT fallback - just incomplete properties
    return &pbc.GetProjectedCostResponse{
        CostPerMonth:  0,
        Currency:      "USD",
        BillingDetail: "SPARSE_PROPERTIES:instanceType|Cannot estimate without SKU",
        // FallbackHint is UNSPECIFIED - this is the right response
    }, nil
}
```

### Testing Cost Diff Scenarios

Comprehensive tests should cover realistic old/new state combinations:

```go
func TestCostDiff_RealWorldScenarios(t *testing.T) {
    tests := []struct {
        name         string
        oldTags      map[string]string
        newTags      map[string]string
        wantBaseline float64
        wantModified float64
        wantDiff     float64
    }{
        {
            name: "instance type change with sparse old state",
            oldTags: map[string]string{
                "instanceType": "t3.medium",
                // monitoring was defaulted (missing)
                // availabilityZone was computed (missing)
            },
            newTags: map[string]string{
                "instanceType": "t3.large",
                "monitoring":   "detailed", // Now explicit
            },
            wantBaseline: 30.37,  // t3.medium with basic monitoring (default)
            wantModified: 80.65,  // t3.large with detailed monitoring
            wantDiff:     50.28,
        },
        {
            name: "storage increase with complete state",
            oldTags: map[string]string{
                "instanceType": "t3.medium",
                "storage":      "20",
            },
            newTags: map[string]string{
                "instanceType": "t3.medium",
                "storage":      "100",
            },
            wantBaseline: 38.37,  // t3.medium + 20GB
            wantModified: 46.37,  // t3.medium + 100GB
            wantDiff:     8.00,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test baseline
            baselineReq := &pbc.GetProjectedCostRequest{
                Resource: &pbc.ResourceDescriptor{
                    Provider:     "aws",
                    ResourceType: "ec2",
                    Tags:         tt.oldTags,
                },
            }
            baselineResp, err := plugin.GetProjectedCost(ctx, baselineReq)
            require.NoError(t, err)
            assert.InDelta(t, tt.wantBaseline, baselineResp.CostPerMonth, 0.01)

            // Test modified
            modifiedReq := &pbc.GetProjectedCostRequest{
                Resource: &pbc.ResourceDescriptor{
                    Provider:     "aws",
                    ResourceType: "ec2",
                    Tags:         tt.newTags,
                },
            }
            modifiedResp, err := plugin.GetProjectedCost(ctx, modifiedReq)
            require.NoError(t, err)
            assert.InDelta(t, tt.wantModified, modifiedResp.CostPerMonth, 0.01)

            // Verify cost diff
            diff := modifiedResp.CostPerMonth - baselineResp.CostPerMonth
            assert.InDelta(t, tt.wantDiff, diff, 0.01)
        })
    }
}
```

## Cost Aggregation Strategies

### Time-Based Aggregation

```go
// TimeAggregator aggregates costs over time windows.
type TimeAggregator struct {
    window    time.Duration
    method    AggregationMethod
    alignment AlignmentType
}

type AggregationMethod string

const (
    AggregationSum AggregationMethod = "sum"
    AggregationAvg AggregationMethod = "avg"
    AggregationMax AggregationMethod = "max"
    AggregationMin AggregationMethod = "min"
)

type AlignmentType string

const (
    AlignmentBilling    AlignmentType = "billing"    // Align to billing period
    AlignmentCalendar   AlignmentType = "calendar"   // Align to calendar boundaries
    AlignmentContinuous AlignmentType = "continuous" // Rolling window
)

func (a *TimeAggregator) Aggregate(dataPoints []CostDataPoint) float64 {
    if len(dataPoints) == 0 {
        return 0
    }

    switch a.method {
    case AggregationSum:
        var total float64
        for _, dp := range dataPoints {
            total += dp.Cost
        }
        return total

    case AggregationAvg:
        var total float64
        for _, dp := range dataPoints {
            total += dp.Cost
        }
        return total / float64(len(dataPoints))

    case AggregationMax:
        maxCost := dataPoints[0].Cost
        for _, dp := range dataPoints[1:] {
            if dp.Cost > maxCost {
                maxCost = dp.Cost
            }
        }
        return maxCost

    case AggregationMin:
        minCost := dataPoints[0].Cost
        for _, dp := range dataPoints[1:] {
            if dp.Cost < minCost {
                minCost = dp.Cost
            }
        }
        return minCost

    default:
        return 0
    }
}

type CostDataPoint struct {
    Timestamp time.Time
    Cost      float64
    Currency  string
}
```

### Cross-Resource Cost Rollup

```go
// ResourceCostRollup aggregates costs across multiple resources.
type ResourceCostRollup struct {
    costs map[string]float64 // resource ID -> cost
}

// AddResourceCost adds a cost entry for a resource.
func (r *ResourceCostRollup) AddResourceCost(resourceID string, cost float64) {
    if r.costs == nil {
        r.costs = make(map[string]float64)
    }
    r.costs[resourceID] += cost
}

// TotalCost returns the sum of all resource costs.
func (r *ResourceCostRollup) TotalCost() float64 {
    var total float64
    for _, cost := range r.costs {
        total += cost
    }
    return total
}

// GroupByTag groups costs by a specific tag.
func (r *ResourceCostRollup) GroupByTag(
    resources map[string]*pbc.ResourceDescriptor,
    tagKey string,
) map[string]float64 {
    grouped := make(map[string]float64)

    for resourceID, cost := range r.costs {
        resource, ok := resources[resourceID]
        if !ok {
            grouped["untagged"] += cost
            continue
        }

        tagValue := resource.GetTags()[tagKey]
        if tagValue == "" {
            tagValue = "untagged"
        }
        grouped[tagValue] += cost
    }

    return grouped
}
```

## Performance Optimization Patterns

### Caching Pricing Data

```go
import (
    "sync"
    "time"
)

// PricingCache caches pricing lookups with TTL expiration.
type PricingCache struct {
    mu      sync.RWMutex
    entries map[string]cacheEntry
    ttl     time.Duration
}

type cacheEntry struct {
    price     float64
    expiresAt time.Time
}

func NewPricingCache(ttl time.Duration) *PricingCache {
    return &PricingCache{
        entries: make(map[string]cacheEntry),
        ttl:     ttl,
    }
}

func (c *PricingCache) Get(key string) (float64, bool) {
    c.mu.RLock()
    entry, ok := c.entries[key]
    c.mu.RUnlock()

    if !ok {
        return 0, false
    }

    // Check expiration and clean up if needed
    if time.Now().After(entry.expiresAt) {
        c.mu.Lock()
        delete(c.entries, key)
        c.mu.Unlock()
        return 0, false
    }

    return entry.price, true
}

func (c *PricingCache) Set(key string, price float64) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.entries[key] = cacheEntry{
        price:     price,
        expiresAt: time.Now().Add(c.ttl),
    }
}

// StartCleanup runs periodic cleanup of expired entries.
// Call this in a goroutine: go cache.StartCleanup(ctx, time.Minute)
func (c *PricingCache) StartCleanup(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.cleanup()
        }
    }
}

func (c *PricingCache) cleanup() {
    c.mu.Lock()
    defer c.mu.Unlock()

    now := time.Now()
    for key, entry := range c.entries {
        if now.After(entry.expiresAt) {
            delete(c.entries, key)
        }
    }
}

// Usage in plugin:
func (p *CachingPlugin) GetProjectedCost(
    ctx context.Context,
    req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
    resource := req.GetResource()
    cacheKey := fmt.Sprintf("%s:%s:%s",
        resource.GetProvider(),
        resource.GetSku(),
        resource.GetRegion(),
    )

    // Check cache first
    if price, ok := p.cache.Get(cacheKey); ok {
        return p.calc.CreateProjectedCostResponse("USD", price, "cached"), nil
    }

    // Fetch from pricing API
    price, err := p.pricingAPI.GetPrice(ctx, resource)
    if err != nil {
        return nil, err
    }

    // Cache for future requests
    p.cache.Set(cacheKey, price)

    return p.calc.CreateProjectedCostResponse("USD", price, "live"), nil
}
```

### Batch Request Processing

```go
// BatchProcessor handles multiple cost requests efficiently.
type BatchProcessor struct {
    maxBatchSize int
    timeout      time.Duration
}

func (b *BatchProcessor) ProcessBatch(
    ctx context.Context,
    requests []*pbc.GetActualCostRequest,
    handler func(context.Context, *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error),
) ([]*pbc.GetActualCostResponse, error) {
    if len(requests) == 0 {
        return nil, nil
    }

    // Process in parallel with bounded concurrency
    results := make([]*pbc.GetActualCostResponse, len(requests))
    errors := make([]error, len(requests))

    sem := make(chan struct{}, b.maxBatchSize)
    var wg sync.WaitGroup

    for i, req := range requests {
        wg.Add(1)
        go func(idx int, r *pbc.GetActualCostRequest) {
            defer wg.Done()

            // Non-blocking semaphore acquire with context
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
            case <-ctx.Done():
                errors[idx] = ctx.Err()
                return
            }

            // Check context before calling handler
            if err := ctx.Err(); err != nil {
                errors[idx] = err
                return
            }

            resp, err := handler(ctx, r)
            results[idx] = resp
            errors[idx] = err
        }(i, req)
    }

    // Wait with context awareness
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // All goroutines completed
    case <-ctx.Done():
        return results, fmt.Errorf("batch cancelled: %w", ctx.Err())
    }

    // Check for any errors
    for i, err := range errors {
        if err != nil {
            return results, fmt.Errorf("request %d failed: %w", i, err)
        }
    }

    return results, nil
}
```

### Memory-Efficient Large Dataset Handling

```go
// StreamingCostProcessor processes large cost datasets without loading all into memory.
type StreamingCostProcessor struct {
    batchSize int
}

func (s *StreamingCostProcessor) ProcessLargeCostReport(
    ctx context.Context,
    reportReader io.Reader,
    processor func(CostRecord) error,
) error {
    decoder := json.NewDecoder(reportReader)

    // Read opening bracket
    if _, err := decoder.Token(); err != nil {
        return err
    }

    // Process records one at a time
    for decoder.More() {
        var record CostRecord
        if err := decoder.Decode(&record); err != nil {
            return err
        }

        if err := processor(record); err != nil {
            return err
        }

        // Check context cancellation periodically
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }

    return nil
}

type CostRecord struct {
    ResourceID string    `json:"resource_id"`
    Cost       float64   `json:"cost"`
    Currency   string    `json:"currency"`
    Timestamp  time.Time `json:"timestamp"`
}
```

## Related Documentation

- [PLUGIN_DEVELOPER_GUIDE.md](../PLUGIN_DEVELOPER_GUIDE.md) - Core plugin development guide
- [PROPERTY_MAPPING.md](./PROPERTY_MAPPING.md) - Pulumi resource property mapping
- [OBSERVABILITY_GUIDE.md](../OBSERVABILITY_GUIDE.md) - Metrics and logging patterns
- [sdk/go/pluginsdk/README.md](../sdk/go/pluginsdk/README.md) - SDK function reference
