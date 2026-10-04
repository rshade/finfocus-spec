# Property Mapping: Pulumi Resources to Plugin Fields

This document provides comprehensive mapping between Pulumi resource properties and
FinFocus plugin fields for all supported cloud providers.

## Overview

FinFocus plugins need to extract pricing-relevant information from Pulumi resource
properties. The `sdk/go/pluginsdk/mapping` package provides helper functions for this
purpose, but understanding the underlying mappings is essential for plugin development.

## ResourceDescriptor Fields

The `ResourceDescriptor` message is the primary data contract between Core and Plugins:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | **Yes** | Cloud provider identifier |
| `resource_type` | string | **Yes** | Type of resource |
| `sku` | string | No | Provider-specific SKU or instance type |
| `region` | string | No | Deployment region |
| `tags` | map | No | Resource labels, and the host's flattened input properties (see [How the Host Hands Properties to a Plugin](#how-the-host-hands-properties-to-a-plugin)). Values are at most 2048 bytes |
| `attributes` | `google.protobuf.Struct` | No | The declared input properties, nested and unflattened (see [Structured Attributes](#structured-attributes)). At most 65536 encoded bytes |

### Provider Is the Cloud, Not the Package

`provider` names the cloud that bills the resource. The IaC package that
declared the resource is not a provider. It stays visible as the
`resource_type` prefix, so a plugin can still tell `azure:` and
`azure-native:` resources apart.

| `resource_type` prefix | `provider` |
|------------------------|------------|
| `aws`, `aws-native`, Terraform `aws_` | `aws` |
| `azure`, `azure-native`, Terraform `azurerm_` | `azure` |
| `gcp`, `google-native`, Terraform `google_` | `gcp` |
| `kubernetes` | `kubernetes` |

Hosts map the prefix to the cloud before they send a descriptor
([rshade/finfocus#1645](https://github.com/rshade/finfocus/issues/1645)).
Plugins list clouds in `GetPluginInfo.providers` and in a manifest's
`supported_providers` and `supported_resources` keys. A plugin should
still accept a package name in `provider`, because older hosts send the raw
prefix.

## How the Host Hands Properties to a Plugin

`ResourceDescriptor.tags` carries the host's flattened input properties in addition to resource labels.
Hosts can also send the same properties unflattened in `ResourceDescriptor.attributes`; see
[Structured Attributes](#structured-attributes). When `attributes` is set, plugins read it first and use
`tags` only as a fallback. This section describes what the reference host, rshade/finfocus core, sends today
(`internal/engine/engine.go`, `ConvertValueToString`, as of core `main`, which pins spec v0.7.0). It is
a description of current behavior, not a guarantee: the collapse rules below are lossy and kept for
compatibility, and a host may change them.

| Input value | What arrives in `tags` |
|-------------|------------------------|
| string | the string |
| number | integer text when whole (`128`), otherwise the shortest decimal (`0.05`) |
| bool | `true` or `false` |
| null | an empty string |
| map | its `value`, else `id`, else `name` entry; else the inner value when it has exactly one key; else Go `fmt` text such as `map[adminUsername:azureuser computerName:lin]` |
| array | an empty string when empty, the element when there is one, otherwise the elements joined with `,` |

Keys that start with `__` (for example `__createBeforeDelete`, or `__defaults` in some Pulumi providers)
are passed through. Only the host's overview display hides them.

A worked example, from a real `pulumi preview --json` of an azure-native 3.28.0 virtual machine with fake
values:

| Pulumi input | `tags` entry |
|--------------|--------------|
| `hardwareProfile: {vmSize: Standard_D4s_v5}` | `hardwareProfile` = `Standard_D4s_v5` (single-key map) |
| `billingProfile: {maxPrice: 0.05}` | `billingProfile` = `0.05` |
| `priority: Spot` | `priority` = `Spot` |
| `zones: ["1"]` | `zones` = `1` |
| `osProfile: {adminUsername, computerName, linuxConfiguration: {...}}` | `osProfile` = `map[adminUsername:azureuser computerName:lin linuxConfiguration:map[disablePasswordAuthentication:false]]` |
| `sku: {name: GP_Gen5, capacity: 4}` (an Azure SQL database) | `sku` = `GP_Gen5`; the capacity is lost |

Consequences for plugin authors:

- Through `tags`, a plugin never receives the nested object, only the collapsed string. Read the key the host
  actually sends, and treat a collapsed value as lossy. Parsing the `map[...]` text is a last resort: it breaks on
  values that contain spaces. Prefer `attributes` when the host sends it.
- The structured form is a `google.protobuf.Struct` that the host builds from the raw properties without
  flattening. It is `ResourceDescriptor.attributes` on every RPC that carries a descriptor (`Supports`,
  `GetProjectedCost`, `GetPricingSpec`, `BatchCost`, and `GetRecommendations` target resources), and
  `EstimateCostRequest.attributes` on `EstimateCost`.
- Tag limits have two different meanings. `pluginsdk.MaxTagsPerResource` (256 entries, key length 128, value
  length 2048 bytes; `sdk/go/pluginsdk/batch.go`) is a denial-of-service guard enforced when `BatchCost` requests
  are validated. The contract bound `MaxTagCount` (50, with the same key and value lengths;
  `sdk/go/testing/contract.go`) is applied by the testing harness. A resource with many properties can exceed the
  second without exceeding the first, which is another reason to prefer `attributes` for nested inputs.
- Strict request validation requires a non-empty `sku` and `region`; `ValidateProjectedCostRequestLenient`
  does not. The SDK server calls neither, so each caller chooses. The reference host calls only the strict
  validator, which means a resource whose SKU the host cannot resolve never reaches the plugin: it becomes a
  validation placeholder instead.

**Recommended convention (not emitted by any host today).** A host should flatten a nested map to dotted keys
in addition to the collapsed value, for example `hardwareProfile.vmSize` = `Standard_D4s_v5` and
`sku.capacity` = `4`, so no information is lost. Until a host does, `hardwareProfile` arrives as the collapsed
value and `hardwareProfile.vmSize` is absent. A plugin may accept both forms. This section will be updated when
the reference host ships it.

## Structured Attributes

`ResourceDescriptor.attributes` (field 12, a `google.protobuf.Struct`) carries the resource's declared input
properties as the IaC program declared them: maps stay maps, lists stay lists, and numbers stay numbers. It
mirrors `EstimateCostRequest.attributes`. A Kubernetes Deployment's CPU request at
`spec.template.spec.containers.0.resources.requests.cpu` (eight segments; a CronJob's is ten) arrives intact,
where `tags` could carry it only as lossy text.

### Rules for Hosts

- **Redact before sending (required).** Omit keys that start with `__`, credential-like keys (a name that
  contains, case-insensitively, `password`, `secret`, `token`, `credential`, `privatekey`, `accesskey`, or
  `connectionstring`), and values that the IaC tool marks secret. For Pulumi, that is any value carrying the
  secret signature `4dabf18193072939515e22adb298388d`. A host may redact more.
- **Keep sending `tags`.** Plugins that predate the field read only `tags`, so `attributes` is additive.
- **Stay within 65536 encoded bytes per resource** (`pluginsdk.MaxAttributesBytes`, the protobuf wire size).
  Both SDK validators reject a larger value: `pluginsdk.ValidateResourceDescriptor` returns a gRPC
  `InvalidArgument` status, and `plugintesting.ValidateResourceDescriptor` returns a `*ContractError` that
  wraps `ErrAttributesTooLarge` (check it with `errors.Is`).
- **Split batches by size, not only by count.** A whole request must fit the transport limit: 1 MB on the
  Connect/HTTP path and 4 MB by default on gRPC. One hundred resources at the per-resource maximum would be about
  6.4 MB, so a host splits a `BatchCost` request until its encoded size fits. The transport rejects an oversized
  request before any plugin code runs, and the SDK does not lower `max_batch_size` for you.
- **Send integers above 2^53 as strings.** Struct numbers are doubles on the wire.

The reference host (rshade/finfocus core) does not populate `attributes` yet; that work is tracked in
[rshade/finfocus#1525](https://github.com/rshade/finfocus/issues/1525). Until it ships, plugins see `tags` only.

### Rules for Plugins

- **Prefer `attributes`, fall back to `tags`.** Unset or empty `attributes` means the host sent none. When the
  same property appears in both with different values, `attributes` wins, because it is the unflattened source.
  Tags with no counterpart in `attributes`, such as resource labels, keep their meaning.
- **Do not log `attributes` verbatim.** Redaction is the host's job, but a plugin's logs are not the place to find
  out a host got it wrong. Log the paths you read, not the values of the whole structure.
- **Do not try to detect redaction.** An omitted secret looks the same as a property the user never set.

### Reading a Path

`pluginsdk.AttributeValue` returns the value at a dot-separated path, so plugins do not each write a walker. A
numeric segment indexes a list. Any miss returns `(nil, false)`, never an error:

```go
attrs := req.GetResource().GetAttributes()

var cpu string
if v, ok := pluginsdk.AttributeValue(attrs, "spec.template.spec.containers.0.resources.requests.cpu"); ok {
    cpu = v.GetStringValue()
} else {
    cpu = req.GetResource().GetTags()["cpu"]
}
```

Decide the fallback on the `ok` result, not on an empty value. A present attribute wins even when it is empty or
`null`, so a conflicting tag never overrides it.

An explicit JSON `null` is found and returned as a `NullValue`. A key that itself contains `.` cannot be
addressed by path; read it from `attrs.GetFields()` directly. The accessor works on any `Struct`, including
`EstimateCostRequest.attributes`.

## AWS Property Mappings

### SKU Extraction

The `ExtractAWSSKU()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `instanceType` | EC2 instances | `t3.micro`, `m5.large` |
| 2 | `instanceClass` | RDS instances | `db.t3.micro`, `db.r5.large` |
| 3 | `type` | Generic fallback | Various |
| 4 | `volumeType` | EBS volumes | `gp3`, `io1`, `st1` |

### Region Extraction

The `ExtractAWSRegion()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `region` | Explicit region | `us-east-1`, `eu-west-1` |
| 2 | `availabilityZone` | Derived region | `us-east-1a` → `us-east-1` |

### Pulumi Resource Type Mappings

| Pulumi Resource Type | SKU Property | Region Property | Notes |
|---------------------|--------------|-----------------|-------|
| `aws:ec2/instance:Instance` | `instanceType` | `availabilityZone` | Region derived from AZ |
| `aws:rds/instance:Instance` | `instanceClass` | `availabilityZone` | Region derived from AZ |
| `aws:ebs/volume:Volume` | `volumeType` | `availabilityZone` | Also consider `size` for cost |
| `aws:s3/bucket:Bucket` | N/A | `region` | Pricing based on storage class |
| `aws:lambda/function:Function` | N/A | `region` | Pricing based on memory/requests |
| `aws:elasticache/cluster:Cluster` | `nodeType` | `availabilityZone` | Use `type` fallback |
| `aws:elasticsearch/domain:Domain` | `instanceType` | `region` | Under `clusterConfig` |

### AWS Examples

```go
// EC2 Instance
props := map[string]string{
    "instanceType":     "t3.medium",
    "availabilityZone": "us-east-1a",
}
sku := mapping.ExtractAWSSKU(props)       // "t3.medium"
region := mapping.ExtractAWSRegion(props) // "us-east-1"

// RDS Instance
props := map[string]string{
    "instanceClass":    "db.t3.micro",
    "availabilityZone": "us-west-2b",
}
sku := mapping.ExtractAWSSKU(props)       // "db.t3.micro"
region := mapping.ExtractAWSRegion(props) // "us-west-2"

// EBS Volume
props := map[string]string{
    "volumeType":       "gp3",
    "availabilityZone": "eu-west-1c",
}
sku := mapping.ExtractAWSSKU(props)       // "gp3"
region := mapping.ExtractAWSRegion(props) // "eu-west-1"
```

## Azure Property Mappings

### SKU Extraction

The `ExtractAzureSKU()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `vmSize` | Virtual machines | `Standard_D2s_v3`, `Standard_B1s` |
| 2 | `sku` | Generic SKU field | Various |
| 3 | `tier` | Service tier | `Basic`, `Standard`, `Premium` |

### Region Extraction

The `ExtractAzureRegion()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `location` | Primary location | `eastus`, `westeurope` |
| 2 | `region` | Alternative field | `eastus`, `westeurope` |

### Pulumi Resource Type Mappings

| Pulumi Resource Type | SKU Property | Region Property | Notes |
|---------------------|--------------|-----------------|-------|
| `azure:compute/virtualMachine:VirtualMachine` | `vmSize` | `location` | Under `hardwareProfile` |
| `azure-native:compute:VirtualMachine` | `vmSize` | `location` | Native package; provider is `azure` |
| `azure:storage/account:Account` | `accountTier` | `location` | Use `tier` fallback |
| `azure:sql/database:Database` | `sku` | `location` | SKU object with name/tier |
| `azure:containerservice/kubernetesCluster:KubernetesCluster` | `vmSize` | `location` | Node pool VM size |
| `azure:appservice/plan:Plan` | `sku` | `location` | SKU with tier/size |

### Azure Examples

```go
// Virtual Machine
props := map[string]string{
    "vmSize":   "Standard_D2s_v3",
    "location": "eastus",
}
sku := mapping.ExtractAzureSKU(props)       // "Standard_D2s_v3"
region := mapping.ExtractAzureRegion(props) // "eastus"

// Storage Account
props := map[string]string{
    "tier":     "Standard",
    "location": "westeurope",
}
sku := mapping.ExtractAzureSKU(props)       // "Standard"
region := mapping.ExtractAzureRegion(props) // "westeurope"

// App Service Plan
props := map[string]string{
    "sku":      "P1v2",
    "location": "centralus",
}
sku := mapping.ExtractAzureSKU(props)       // "P1v2"
region := mapping.ExtractAzureRegion(props) // "centralus"
```

## GCP Property Mappings

### SKU Extraction

The `ExtractGCPSKU()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `machineType` | Compute instances | `e2-micro`, `n1-standard-4` |
| 2 | `type` | Generic type field | Various |
| 3 | `tier` | Service tier | `db-f1-micro`, `BASIC` |

### Region Extraction

The `ExtractGCPRegion()` function checks keys in priority order:

| Priority | Property Key | Use Case | Example Value |
|----------|-------------|----------|---------------|
| 1 | `region` | Explicit region | `us-central1`, `europe-west1` |
| 2 | `zone` | Derived region | `us-central1-a` → `us-central1` |

### Zone-to-Region Derivation

GCP zones follow the pattern `{region}-{zone-letter}`. The mapping package validates
extracted regions against a known list of 40+ GCP regions.

**Supported GCP Regions:**

- **Asia Pacific**: `asia-east1`, `asia-east2`, `asia-northeast1-3`, `asia-south1-2`, `asia-southeast1-2`
- **Australia**: `australia-southeast1-2`
- **Europe**: `europe-central2`, `europe-north1`, `europe-southwest1`, `europe-west1-4`,
  `europe-west6`, `europe-west8-10`, `europe-west12`
- **Middle East**: `me-central1-2`, `me-west1`
- **North America**: `northamerica-northeast1-2`, `us-central1`, `us-east1`, `us-east4-5`, `us-south1`, `us-west1-4`
- **South America**: `southamerica-east1`, `southamerica-west1`

### Pulumi Resource Type Mappings

| Pulumi Resource Type | SKU Property | Region Property | Notes |
|---------------------|--------------|-----------------|-------|
| `gcp:compute/instance:Instance` | `machineType` | `zone` | Region derived from zone |
| `gcp:sql/databaseInstance:DatabaseInstance` | `tier` | `region` | Under `settings` |
| `gcp:storage/bucket:Bucket` | N/A | `location` | Use `region` fallback |
| `gcp:container/cluster:Cluster` | `machineType` | `location` | Node pool machine type |
| `gcp:cloudfunctions/function:Function` | N/A | `region` | Pricing based on invocations |
| `gcp:bigquery/dataset:Dataset` | N/A | `location` | Multi-region or region |

### GCP Examples

```go
// Compute Instance
props := map[string]string{
    "machineType": "n1-standard-4",
    "zone":        "us-central1-a",
}
sku := mapping.ExtractGCPSKU(props)       // "n1-standard-4"
region := mapping.ExtractGCPRegion(props) // "us-central1"

// Cloud SQL Instance
props := map[string]string{
    "tier":   "db-f1-micro",
    "region": "europe-west1",
}
sku := mapping.ExtractGCPSKU(props)       // "db-f1-micro"
region := mapping.ExtractGCPRegion(props) // "europe-west1"

// GKE Cluster Node Pool
props := map[string]string{
    "machineType": "e2-medium",
    "zone":        "asia-east1-b",
}
sku := mapping.ExtractGCPSKU(props)       // "e2-medium"
region := mapping.ExtractGCPRegion(props) // "asia-east1"
```

## Kubernetes Property Mappings

Kubernetes resources typically don't have direct SKU mappings since pricing
is derived from the underlying infrastructure. Use tags for cost allocation.

### Common Kubernetes Tags

| Tag Key | Description | Example Value |
|---------|-------------|---------------|
| `namespace` | Kubernetes namespace | `default`, `production` |
| `app` | Application identifier | `web-frontend`, `api-server` |
| `team` | Team ownership | `platform`, `data-eng` |
| `env` | Environment | `dev`, `staging`, `prod` |
| `cost-center` | Cost allocation | `engineering`, `marketing` |

### ResourceDescriptor for Kubernetes

```go
// Kubernetes Namespace
descriptor := &proto.ResourceDescriptor{
    Provider:     "kubernetes",
    ResourceType: "k8s-namespace",
    Region:       "us-east-1",  // Cluster region
    Tags: map[string]string{
        "namespace":   "production",
        "team":        "platform",
        "cost-center": "engineering",
    },
}

// Kubernetes Deployment
descriptor := &proto.ResourceDescriptor{
    Provider:     "kubernetes",
    ResourceType: "k8s-deployment",
    Tags: map[string]string{
        "namespace":  "default",
        "app":        "web-frontend",
        "controller": "deployment/web-frontend",
    },
}
```

Container requests and replica counts are nested, so they belong in `attributes`
(see [Structured Attributes](#structured-attributes)):

```go
attrs, _ := structpb.NewStruct(map[string]any{
    "spec": map[string]any{
        "replicas": 3,
        "template": map[string]any{"spec": map[string]any{"containers": []any{
            map[string]any{"resources": map[string]any{"requests": map[string]any{"cpu": "250m"}}},
        }}},
    },
})
descriptor.Attributes = attrs

replicas, _ := pluginsdk.AttributeValue(descriptor.GetAttributes(), "spec.replicas")
// replicas.GetNumberValue() == 3
```

## Generic Extraction Functions

For custom resource types or providers not covered by the specific functions,
use the generic extraction functions:

### ExtractSKU

```go
// With custom keys
props := map[string]string{"customSKU": "my-sku-value"}
sku := mapping.ExtractSKU(props, "customSKU", "fallbackKey")  // "my-sku-value"

// With default keys (sku, type, tier)
props := map[string]string{"type": "standard"}
sku := mapping.ExtractSKU(props)  // "standard"
```

### ExtractRegion

```go
// With custom keys
props := map[string]string{"deploymentRegion": "us-west-2"}
region := mapping.ExtractRegion(props, "deploymentRegion")  // "us-west-2"

// With default keys (region, location, zone)
props := map[string]string{"location": "eastus"}
region := mapping.ExtractRegion(props)  // "eastus"
```

## Sparse Property Scenarios: Old vs New State

When `GetProjectedCost` is called as part of a **cost diff operation**, the plugin receives
property maps that may be incomplete. Understanding the difference between old state and new state
properties is critical for accurate cost projections.

### Property Availability by Source

| Property Source | Typical Contents | Completeness |
|---|---|---|
| **OldState.Inputs** | User-declared inputs from previous deployment | **Sparse** - Only explicit declarations |
| **NewState.Inputs** | Current user-declared inputs | **Partial** - Richer than OldState.Inputs but can omit provider-computed/defaulted values |
| **Computed Values** | Provider-calculated values (e.g., auto-generated IDs) | **Never in OldState.Inputs** |

### Per-Provider Property Patterns

#### AWS Properties

| Property | OldState.Inputs | NewState.Inputs | Notes |
|---|---|---|---|
| `instanceType` | ✅ Usually present | ✅ Always present | User-specified |
| `availabilityZone` | ⚠️ Often missing | ✅ Usually present | May be provider-assigned |
| `ebsOptimized` | ❌ Rarely present | ✅ Often present | Defaults to false if omitted |
| `monitoring` | ❌ Rarely present | ⚠️ Sometimes present | Defaults to basic if omitted |
| `volumeSize` | ⚠️ Depends on declaration | ✅ Usually present | May have provider default |

**Extraction Pattern for AWS:**

```go
sku := mapping.ExtractAWSSKU(props)
if sku == "" {
    // Old state likely missing instanceType - cannot estimate
    return signalSparseProperties("instanceType")
}

region := mapping.ExtractAWSRegion(props)
if region == "" {
    // availabilityZone missing - may need to use default region
    region = "us-east-1" // Or signal error
}
```

#### Azure Properties

| Property | OldState.Inputs | NewState.Inputs | Notes |
|---|---|---|---|
| `vmSize` | ✅ Usually present | ✅ Always present | User-specified |
| `location` | ✅ Usually present | ✅ Always present | Required at creation |
| `osDisk.diskSizeGB` | ❌ Often missing | ⚠️ Sometimes present | Provider default varies by SKU |
| `networkInterfaces` | ❌ Rarely present | ⚠️ Sometimes present | Often computed |

**Extraction Pattern for Azure:**

```go
sku := mapping.ExtractAzureSKU(props)
if sku == "" {
    return signalSparseProperties("vmSize")
}

// Location is usually stable across old and new state
region := mapping.ExtractAzureRegion(props)
```

#### GCP Properties

| Property | OldState.Inputs | NewState.Inputs | Notes |
|---|---|---|---|
| `machineType` | ✅ Usually present | ✅ Always present | User-specified |
| `zone` | ⚠️ Sometimes missing | ✅ Usually present | May be project default |
| `diskSizeGb` | ❌ Often missing | ⚠️ Sometimes present | Provider default 10GB |
| `serviceAccount` | ❌ Rarely present | ⚠️ Sometimes present | May use default SA |

**Extraction Pattern for GCP:**

```go
sku := mapping.ExtractGCPSKU(props)
if sku == "" {
    return signalSparseProperties("machineType")
}

region := mapping.ExtractGCPRegion(props)
if region == "" {
    // Zone missing - try project default or signal error
    return signalSparseProperties("zone")
}
```

### Extraction Function Behavior with Sparse Maps

All mapping extraction functions return **empty string** when properties are missing:

```go
// Empty map
props := map[string]string{}
sku := mapping.ExtractAWSSKU(props) // Returns: ""

// Nil map
var props map[string]string
sku = mapping.ExtractAWSSKU(props)  // Returns: ""

// Property present but empty
props = map[string]string{"instanceType": ""}
sku = mapping.ExtractAWSSKU(props)  // Returns: ""
```

**Critical Pattern**: Always check extraction results before proceeding:

```go
// CORRECT: Check before use
sku := mapping.ExtractAWSSKU(req.Resource.Tags)
if sku == "" {
    // Handle sparse property scenario
    return &pbc.GetProjectedCostResponse{
        CostPerMonth:  0,
        Currency:      "USD",
        BillingDetail: "SPARSE_PROPERTIES:instanceType|Cannot estimate without SKU",
    }, nil
}

// WRONG: Assume extraction succeeded
sku := mapping.ExtractAWSSKU(req.Resource.Tags)
price := pricingAPI.GetPrice(sku) // May fail or return wrong result
```

### Testing with Sparse Properties

Plugins should test realistic sparse property scenarios:

```go
func TestSparseOldStateProperties(t *testing.T) {
    tests := []struct {
        name     string
        props    map[string]string
        wantErr  bool
        wantCost float64
    }{
        {
            name: "old state - only instanceType",
            props: map[string]string{
                "instanceType": "t3.medium",
                // No availabilityZone (was provider-assigned)
                // No ebsOptimized (was defaulted)
            },
            wantErr:  false,
            wantCost: 30.37, // Should use default values
        },
        {
            name: "old state - completely sparse",
            props: map[string]string{},
            wantErr:  false,
            wantCost: 0, // Should signal sparse properties
        },
        {
            name: "new state - complete properties",
            props: map[string]string{
                "instanceType":     "t3.large",
                "availabilityZone": "us-east-1a",
                "ebsOptimized":     "true",
            },
            wantErr:  false,
            wantCost: 75.65,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := &pbc.GetProjectedCostRequest{
                Resource: &pbc.ResourceDescriptor{
                    Provider:     "aws",
                    ResourceType: "ec2",
                    Tags:         tt.props,
                },
            }
            resp, err := plugin.GetProjectedCost(ctx, req)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantCost, resp.CostPerMonth)
            }
        })
    }
}
```

## Common Pitfalls

### 1. Nested Struct Properties

Cloud provider properties often contain nested structures, but the mapping functions
expect flat key-value pairs, and the host has already reduced every nested value to a string in
`ResourceDescriptor.tags` (see [How the Host Hands Properties to a Plugin](#how-the-host-hands-properties-to-a-plugin)).

**Problem**: Azure VM sizes are nested under `hardwareProfile`. What the user's Pulumi program declares:

```json
{
    "hardwareProfile": {
        "vmSize": "Standard_D2s_v3"
    },
    "location": "eastus"
}
```

**Solution**: Read `attributes` first. When the host sends it, the nested value is intact and
`pluginsdk.AttributeValue(attrs, "hardwareProfile.vmSize")` returns it. Otherwise read the key the host actually
sends in `tags`. The reference host collapses a single-key map to its value, so
this arrives as the tag `hardwareProfile` = `Standard_D2s_v3`; a host that follows the recommended dotted-key
convention would also send `hardwareProfile.vmSize`. Accept both, and do not rely on the collapse for maps with
more than one key:

```go
var sku string
if v, ok := pluginsdk.AttributeValue(req.GetResource().GetAttributes(), "hardwareProfile.vmSize"); ok {
    sku = v.GetStringValue()
} else {
    props := req.GetResource().GetTags()
    sku = props["hardwareProfile.vmSize"] // recommended convention, not emitted by any host today
    if sku == "" {
        sku = props["hardwareProfile"] // single-key map collapsed to its value
    }
}
```

`mapping.ExtractAzureSKU` reads only the top-level keys `vmSize`, `sku` and `tier`, so pass it a map in which
you have placed the value under one of those names.

### 2. Case Sensitivity in Property Keys

Property keys are **case-sensitive**. Using incorrect casing returns empty strings.

**Problem**:

```go
props := map[string]string{
    "InstanceType": "t3.micro",  // Wrong: capital 'I'
}
sku := mapping.ExtractAWSSKU(props)  // Returns "" (empty)
```

**Solution**:

```go
props := map[string]string{
    "instanceType": "t3.micro",  // Correct: lowercase 'i'
}
sku := mapping.ExtractAWSSKU(props)  // Returns "t3.micro"
```

### 3. Zone vs Region Confusion

GCP and AWS use zones that must be converted to regions for pricing lookups.

**Problem**:

```go
props := map[string]string{
    "region": "us-central1-a",  // This is a zone, not a region!
}
region := mapping.ExtractGCPRegion(props)  // Returns "us-central1-a" (incorrect)
```

**Solution**: Use the correct property key or explicit zone-to-region conversion:

```go
// Option 1: Use "zone" key (automatic conversion)
props := map[string]string{
    "zone": "us-central1-a",
}
region := mapping.ExtractGCPRegion(props)  // Returns "us-central1"

// Option 2: Explicit conversion
region := mapping.ExtractGCPRegionFromZone("us-central1-a")  // Returns "us-central1"
```

### 4. Empty String Returns Are Silent

Mapping functions return empty strings without errors. Always check for empty returns.

**Problem**:

```go
props := map[string]string{}  // Empty properties
sku := mapping.ExtractAWSSKU(props)
// sku is "", but no error is raised
// Cost calculations may fail silently
```

**Solution**: Validate extraction results.

For robust production plugins, use gRPC status errors to clearly communicate missing
requirements back to the Core:

```go
// Extract with validation
sku := mapping.ExtractAWSSKU(props)
if sku == "" {
    return nil, status.Errorf(codes.InvalidArgument,
        "missing required instanceType for EC2 instance")
}
```

### 5. Provider-Specific Key Mismatches

Using the wrong provider's extraction function with another provider's properties.

**Problem**:

```go
// Azure properties
props := map[string]string{
    "vmSize":   "Standard_D2s_v3",
    "location": "eastus",
}
// Wrong function!
sku := mapping.ExtractAWSSKU(props)  // Returns "" (vmSize not checked)
```

**Solution**: Match the extraction function to the resource provider:

```go
sku := mapping.ExtractAzureSKU(props)  // Returns "Standard_D2s_v3"
```

### 6. Availability Zone Edge Cases

AWS availability zones can have non-standard suffixes for Local Zones and Wavelength Zones.

**Problem**:

```go
// AWS Local Zone
props := map[string]string{
    "availabilityZone": "us-east-1-bos-1a",  // Boston Local Zone
}
region := mapping.ExtractAWSRegion(props)
// May not correctly extract "us-east-1"
```

**Solution**: Verify region extraction for non-standard zones:

```go
region := mapping.ExtractAWSRegion(props)
if region == "" || !isKnownAWSRegion(region) {
    // Fall back to explicit region property or query the API
    region = getRegionFromAWSAPI(resourceID)
}
```

## Best Practices

### 1. Use Provider-Specific Functions

Always prefer provider-specific functions over generic ones:

```go
// Good
sku := mapping.ExtractAWSSKU(props)

// Less optimal (may not check all relevant keys)
sku := mapping.ExtractSKU(props)
```

### 2. Handle Empty Returns

All mapping functions return empty strings for missing or invalid data:

```go
sku := mapping.ExtractAWSSKU(props)
if sku == "" {
    // Handle missing SKU - may need to query API or use defaults
}
```

### 3. Validate Regions

For GCP, use the validation function:

```go
region := mapping.ExtractGCPRegion(props)
if region != "" && !mapping.IsValidGCPRegion(region) {
    // Handle invalid region
}
```

### 4. Consider Resource-Specific Needs

Some resources require additional properties beyond SKU and region:

| Resource Type | Additional Properties |
|--------------|----------------------|
| EBS Volume | `size` (GB), `iops`, `throughput` |
| S3 Bucket | `storageClass`, `versioning` |
| Lambda Function | `memorySize`, `timeout` |
| RDS Instance | `allocatedStorage`, `multiAz` |

### 5. Use Tags for Cost Allocation

Tags enable cost attribution across teams, environments, and projects. (A host may also place its flattened
input properties in `tags`; see [How the Host Hands Properties to a Plugin](#how-the-host-hands-properties-to-a-plugin).)

```go
descriptor := &proto.ResourceDescriptor{
    Provider:     "aws",
    ResourceType: "ec2",
    Sku:          "t3.medium",
    Region:       "us-east-1",
    Tags: map[string]string{
        "env":         "production",
        "team":        "platform",
        "cost-center": "eng-123",
        "project":     "api-gateway",
    },
}
```

## API Reference

### Package: `github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping`

| Function | Description |
|----------|-------------|
| `ExtractAWSSKU(props)` | Extract SKU from AWS resource properties |
| `ExtractAWSRegion(props)` | Extract region from AWS resource properties |
| `ExtractAWSRegionFromAZ(az)` | Derive region from AWS availability zone |
| `ExtractAzureSKU(props)` | Extract SKU from Azure resource properties |
| `ExtractAzureRegion(props)` | Extract region from Azure resource properties |
| `ExtractGCPSKU(props)` | Extract SKU from GCP resource properties |
| `ExtractGCPRegion(props)` | Extract region from GCP resource properties |
| `ExtractGCPRegionFromZone(zone)` | Derive region from GCP zone |
| `IsValidGCPRegion(region)` | Validate GCP region name |
| `AllGCPRegions()` | Get list of all known GCP regions |
| `ExtractSKU(props, keys...)` | Generic SKU extraction |
| `ExtractRegion(props, keys...)` | Generic region extraction |

### Property Key Constants

| Constant | Value | Provider |
|----------|-------|----------|
| `AWSKeyInstanceType` | `instanceType` | AWS |
| `AWSKeyInstanceClass` | `instanceClass` | AWS |
| `AWSKeyType` | `type` | AWS |
| `AWSKeyVolumeType` | `volumeType` | AWS |
| `AWSKeyRegion` | `region` | AWS |
| `AWSKeyAvailabilityZone` | `availabilityZone` | AWS |
| `AzureKeyVMSize` | `vmSize` | Azure |
| `AzureKeySKU` | `sku` | Azure |
| `AzureKeyTier` | `tier` | Azure |
| `AzureKeyLocation` | `location` | Azure |
| `AzureKeyRegion` | `region` | Azure |
| `GCPKeyMachineType` | `machineType` | GCP |
| `GCPKeyType` | `type` | GCP |
| `GCPKeyTier` | `tier` | GCP |
| `GCPKeyRegion` | `region` | GCP |
| `GCPKeyZone` | `zone` | GCP |
