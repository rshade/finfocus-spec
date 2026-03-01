# FinFocus Mapping Package

This package provides helper functions for extracting SKU, region, and other
pricing-relevant fields from Pulumi resource properties. It supports AWS, Azure,
and GCP cloud providers with provider-specific extraction functions, plus generic
fallback extractors for custom resource types.

## Overview

| File         | Description                                      |
| ------------ | ------------------------------------------------ |
| `aws.go`     | AWS property extraction (EC2, RDS, EBS)          |
| `azure.go`   | Azure property extraction (VMs, SKUs)            |
| `gcp.go`     | GCP property extraction with region validation   |
| `common.go`  | Generic extractors with configurable keys        |
| `keys.go`    | Property key constants for all providers         |

## Quick Start

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"

// AWS: Extract SKU and region from EC2 properties
props := map[string]string{
    "instanceType":     "t3.medium",
    "availabilityZone": "us-east-1a",
}
sku := mapping.ExtractAWSSKU(props)       // "t3.medium"
region := mapping.ExtractAWSRegion(props) // "us-east-1"

// Azure: Extract from VM properties
props = map[string]string{
    "vmSize":   "Standard_D2s_v3",
    "location": "eastus",
}
sku = mapping.ExtractAzureSKU(props)       // "Standard_D2s_v3"
region = mapping.ExtractAzureRegion(props) // "eastus"

// GCP: Extract from Compute Engine properties
props = map[string]string{
    "machineType": "n1-standard-4",
    "zone":        "us-central1-a",
}
sku = mapping.ExtractGCPSKU(props)       // "n1-standard-4"
region = mapping.ExtractGCPRegion(props) // "us-central1"
```

## Provider-Specific Functions

### AWS

| Function                | Description                                   |
| ----------------------- | --------------------------------------------- |
| `ExtractAWSSKU`         | Extracts from instanceType, instanceClass, type, volumeType |
| `ExtractAWSRegion`      | Extracts from region or availabilityZone      |
| `ExtractAWSRegionFromAZ`| Derives region from AZ (e.g., "us-east-1a" → "us-east-1") |

**Key Priority Order (SKU):**

1. `instanceType` - EC2 instances
2. `instanceClass` - RDS instances
3. `type` - Generic fallback
4. `volumeType` - EBS volumes

### Azure

| Function             | Description                              |
| -------------------- | ---------------------------------------- |
| `ExtractAzureSKU`    | Extracts from vmSize, sku, tier          |
| `ExtractAzureRegion` | Extracts from location or region         |

**Key Priority Order (SKU):**

1. `vmSize` - Virtual machines
2. `sku` - Generic SKU field
3. `tier` - Service tier

### GCP

| Function                | Description                                |
| ----------------------- | ------------------------------------------ |
| `ExtractGCPSKU`         | Extracts from machineType, type, tier      |
| `ExtractGCPRegion`      | Extracts from region or zone               |
| `ExtractGCPRegionFromZone` | Derives region from zone with validation |
| `IsValidGCPRegion`      | Validates against known GCP regions        |
| `AllGCPRegions`         | Returns list of all known GCP regions      |

**Key Priority Order (SKU):**

1. `machineType` - Compute instances
2. `type` - Generic type field
3. `tier` - Service tier

## Generic Functions

For custom resource types or providers not directly supported:

```go
// Custom SKU extraction with specific keys
props := map[string]string{"customSKU": "my-sku"}
sku := mapping.ExtractSKU(props, "customSKU", "fallback") // "my-sku"

// Default keys: "sku", "type", "tier"
sku = mapping.ExtractSKU(props) // Uses defaults

// Custom region extraction
region := mapping.ExtractRegion(props, "customRegion", "zone")

// Default keys: "region", "location", "zone"
region = mapping.ExtractRegion(props) // Uses defaults
```

## Property Key Constants

All property keys are exported as constants for type safety:

```go
// AWS Keys
mapping.AWSKeyInstanceType      // "instanceType"
mapping.AWSKeyInstanceClass     // "instanceClass"
mapping.AWSKeyType              // "type"
mapping.AWSKeyVolumeType        // "volumeType"
mapping.AWSKeyRegion            // "region"
mapping.AWSKeyAvailabilityZone  // "availabilityZone"

// Azure Keys
mapping.AzureKeyVMSize    // "vmSize"
mapping.AzureKeySKU       // "sku"
mapping.AzureKeyTier      // "tier"
mapping.AzureKeyLocation  // "location"
mapping.AzureKeyRegion    // "region"

// GCP Keys
mapping.GCPKeyMachineType // "machineType"
mapping.GCPKeyType        // "type"
mapping.GCPKeyTier        // "tier"
mapping.GCPKeyRegion      // "region"
mapping.GCPKeyZone        // "zone"
```

## Error Handling

All functions are designed to never panic and always return safe default values:

```go
mapping.ExtractAWSSKU(nil)                 // ""
mapping.ExtractAWSSKU(map[string]string{}) // ""
mapping.ExtractAWSRegionFromAZ("")         // ""
mapping.ExtractGCPRegionFromZone("invalid")// "" (fails validation)
```

### Sparse Property Scenarios

When processing cost diff requests, property maps may be incomplete (from Pulumi's
`OldState.Inputs`). Extraction functions return **empty string** for missing properties:

```go
// Sparse property map from old resource state
props := map[string]string{
    "instanceType": "t3.medium",
    // availabilityZone missing (was provider-computed)
    // monitoring missing (was defaulted)
}

sku := mapping.ExtractAWSSKU(props)       // "t3.medium" - present
region := mapping.ExtractAWSRegion(props) // "" - MISSING

// CRITICAL: Always check extraction results before use
if region == "" {
    // Handle sparse property scenario:
    // Option 1: Signal inability to estimate
    return signalSparseProperties("region")

    // Option 2: Apply documented default
    region = "us-east-1"
    billingDetail += "; region defaulted to us-east-1"
}
```

**Best Practice**: Check all extraction results before proceeding with cost calculations. See
[docs/PROPERTY_MAPPING.md](../../../../docs/PROPERTY_MAPPING.md) section "Sparse Property
Scenarios" for comprehensive guidance.

**Related Documentation**:

- [PLUGIN_DEVELOPER_GUIDE.md](../../../../PLUGIN_DEVELOPER_GUIDE.md) - "Cost Diff Pattern: Handling Sparse Properties"
- [docs/ADVANCED_PATTERNS.md](../../../../docs/ADVANCED_PATTERNS.md) - "Cost Diff Patterns: Comparing Resource States"

## Performance

All functions are optimized for minimal allocation:

- **Target**: <50 ns/op, 0 allocs/op
- No regex compilation at runtime
- Simple string operations only
- Package-level constants for key names

Run benchmarks:

```bash
go test -bench=. -benchmem ./sdk/go/pluginsdk/mapping/
```

## Usage in Plugins

Typical usage pattern in a cost source plugin:

```go
func (p *MyPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (
    *pbc.GetActualCostResponse, error) {

    resource := req.GetResource()
    props := resource.GetProperties()

    var sku, region string
    switch resource.GetProvider() {
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
```
