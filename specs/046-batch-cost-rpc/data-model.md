# Data Model: Batch Cost RPC

**Feature**: 046-batch-cost-rpc | **Date**: 2026-02-24

## Entity Overview

```text
BatchCostRequest
├── resources: [ResourceDescriptor]  ── reuses existing message
├── query_type: CostQueryType        ── new enum
├── start/end: Timestamp             ── optional, for ACTUAL queries
├── dry_run: bool                    ── reuses existing pattern
└── max_batch_size (server config)

BatchCostResponse
├── results: [ResourceCostResult]    ── positionally ordered
└── max_batch_size: int32            ── reports plugin limit

ResourceCostResult
├── resource: ResourceDescriptor     ── echo back for correlation
└── oneof result:
    ├── cost_data: CostData          ── success path
    └── error: ResourceError         ── failure path

CostData
└── oneof data:
    ├── actual_cost: ActualCostData      ── for ACTUAL queries
    ├── projected_cost: CostEstimate     ── for PROJECTED queries
    ├── estimate: CostEstimate           ── for ESTIMATE queries
    └── dry_run_result: DryRunResult     ── for dry_run=true

ResourceError
├── code: int32                      ── gRPC-compatible error code
├── message: string                  ── human-readable description
└── resource_type_unsupported: bool  ── specific unsupported indicator
```

## Entities

### CostQueryType (New Enum)

| Value | Number | Description |
|-------|--------|-------------|
| COST_QUERY_TYPE_UNSPECIFIED | 0 | Default; treated as ESTIMATE |
| COST_QUERY_TYPE_ESTIMATE | 1 | Maps to EstimateCost RPC |
| COST_QUERY_TYPE_ACTUAL | 2 | Maps to GetActualCost RPC |
| COST_QUERY_TYPE_PROJECTED | 3 | Maps to GetProjectedCost RPC |

**Validation**: Zero-allocation validation via package-level slice (follows registry pattern).

### BatchCostRequest (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| resources | repeated ResourceDescriptor | 1 | Resources to query costs for | len <= max_batch_size; empty allowed |
| query_type | CostQueryType | 2 | Cost operation to perform | Required (UNSPECIFIED treated as ESTIMATE) |
| start | google.protobuf.Timestamp | 3 | Start of time range | Required when query_type=ACTUAL |
| end | google.protobuf.Timestamp | 4 | End of time range | Required when query_type=ACTUAL; must be > start |
| dry_run | bool | 5 | Return field mappings only | Optional; false by default |

**Validation Rules**:

- `len(resources) <= MaxBatchSize` (default 100, hard limit 1000)
- When `len(resources) == 0`, return empty response (not an error)
- When `query_type == ACTUAL`: `start` and `end` are required, `start < end`
- When `query_type != ACTUAL`: `start` and `end` are ignored if present
- When `dry_run == true`: `query_type` is still used to determine which fields to map

### BatchCostResponse (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| results | repeated ResourceCostResult | 1 | Per-resource results, positionally ordered | len(results) == len(request.resources) |
| max_batch_size | int32 | 2 | Plugin's maximum supported batch size | > 0 |

**Invariant**: `results[i]` corresponds to `request.resources[i]`.

### ResourceCostResult (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| resource | ResourceDescriptor | 1 | Echo of the input resource descriptor | Always present |
| cost_data | CostData | 2 | Success: cost data for the resource | Present on success (oneof result) |
| error | ResourceError | 3 | Failure: error details for the resource | Present on failure (oneof result) |

**Note**: `cost_data` and `error` are a `oneof result` - exactly one is set per entry.

### CostData (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| actual_cost | ActualCostData | 1 | Actual cost results (for ACTUAL queries) | oneof data |
| projected_cost | CostEstimate | 2 | Projected cost (for PROJECTED queries) | oneof data |
| estimate | CostEstimate | 3 | Cost estimate (for ESTIMATE queries) | oneof data |
| dry_run_result | DryRunResult | 4 | Field mappings (when dry_run=true) | oneof data |

**Note**: The populated field matches the `query_type` in the request (or `dry_run_result`
when `dry_run=true`).

### ActualCostData (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| results | repeated ActualCostResult | 1 | Cost data points for the time range | May be empty |
| fallback_hint | FallbackHint | 2 | Whether host should try other plugins | Optional |
| next_page_token | string | 3 | Pagination token for more results | Optional |
| total_count | int32 | 4 | Total number of results across pages | >= 0 |

**Note**: Wraps the existing `GetActualCostResponse` fields for use within a batch context.
Each resource's actual cost data may be paginated independently.

### ResourceError (New Message)

| Field | Type | Number | Description | Validation |
|-------|------|--------|-------------|------------|
| code | int32 | 1 | gRPC-compatible error code | Required |
| message | string | 2 | Human-readable error description | Required, non-empty |
| resource_type_unsupported | bool | 3 | Whether the resource type is unsupported | Optional |

**Error Codes** (reuse gRPC status codes):

- `5` (NOT_FOUND): Resource not found
- `3` (INVALID_ARGUMENT): Invalid resource descriptor
- `12` (UNIMPLEMENTED): Resource type not supported by plugin
- `13` (INTERNAL): Internal plugin error
- `14` (UNAVAILABLE): Transient error (retryable)

### PluginCapability Extension

| Value | Number | Description |
|-------|--------|-------------|
| PLUGIN_CAPABILITY_BATCH_COST | 12 | Plugin implements BatchCost RPC |

**Auto-Discovery**: Detected via `BatchCostHandler` interface type assertion.
**Legacy Metadata**: `"supports_batch_cost": "true"` in GetPluginInfo metadata map.

## Relationships

```text
BatchCostRequest ──1:N──> ResourceDescriptor    (reuses existing)
BatchCostResponse ──1:N──> ResourceCostResult   (positionally mapped)
ResourceCostResult ──1:1──> ResourceDescriptor  (echo for correlation)
ResourceCostResult ──oneof──> CostData | ResourceError
CostData ──oneof──> ActualCostData | CostEstimate | DryRunResult
ActualCostData ──1:N──> ActualCostResult        (reuses existing)
```

## SDK Constants

| Constant | Value | Description |
|----------|-------|-------------|
| DefaultMaxBatchSize | 100 | Default max resources per batch |
| MaxBatchSize | 1000 | Hard upper limit for batch size |
| DefaultBatchWorkers | 10 | Default concurrent workers for fallback |
| MinBatchWorkers | 1 | Minimum workers (sequential) |
| MaxBatchWorkers | 50 | Maximum workers |

## SDK Interfaces

### BatchCostHandler (New Optional Interface)

```go
// BatchCostHandler is implemented by plugins that provide optimized batch
// cost processing. Plugins that do not implement this interface will have
// batch requests served via concurrent individual RPC calls.
type BatchCostHandler interface {
    BatchCost(ctx context.Context, req *pbc.BatchCostRequest) (*pbc.BatchCostResponse, error)
}
```

### ServeConfig Extension

```go
type ServeConfig struct {
    // ... existing fields ...
    MaxBatchSize int // 0 = DefaultMaxBatchSize (100)
    BatchWorkers int // 0 = DefaultBatchWorkers (10); only used for fallback
}
```

## Data Volume Assumptions

- Typical batch: 10-50 resources
- Maximum batch: 100 resources (default), 1000 (hard limit)
- Per-resource actual cost: 1-1000 data points (depending on time range)
- Per-resource projected/estimate: single CostEstimate message
- Total response size for 100 resources with estimates: ~50KB
- Total response size for 100 resources with 30-day actual costs: ~5MB
