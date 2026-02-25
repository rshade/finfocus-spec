# Research: Batch Cost RPC

**Feature**: 046-batch-cost-rpc | **Date**: 2026-02-24

## R1: Batch Message Structure - Single RPC vs Per-Type RPCs

**Task**: Determine whether batch cost queries should use a single `BatchCost` RPC with a
`CostQueryType` discriminator or separate RPCs per cost type (BatchActualCost,
BatchProjectedCost, BatchEstimateCost).

**Decision**: Single `BatchCost` RPC with `CostQueryType` enum discriminator.

**Rationale**: A single RPC reduces proto surface area, simplifies capability discovery
(one capability enum value instead of three), and matches the spec's assumption that "batch
requests share a single cost query type across all resources." The `CostQueryType` enum
(ESTIMATE, ACTUAL, PROJECTED) selects the underlying operation for the entire batch.

**Findings**:

- The existing proto already has separate RPCs for individual cost queries
  (`GetActualCost`, `GetProjectedCost`, `EstimateCost`), each with different
  request/response shapes
- A single batch RPC with a query type enum avoids tripling the batch surface area
- The `GetRecommendations` RPC already demonstrates multi-resource input via
  `target_resources` (max 100) - this is the closest existing pattern
- gRPC best practices favor fewer RPCs with richer messages over many narrow RPCs

**Alternatives Considered**:

- **Three separate batch RPCs**: More type-safe per query type but triples proto surface,
  SDK interfaces, capability enum values, and test coverage. Rejected for complexity.
- **Streaming batch RPC**: Server-streaming could handle very large batches but adds
  complexity for typical 10-100 resource batches. Deferred per spec assumptions.

## R2: Response Structure - Per-Resource Result Union

**Task**: Determine how per-resource results should represent the cost data vs error union,
given that different query types (actual, projected, estimate) return different response shapes.

**Decision**: Use a `oneof` in `ResourceCostResult` that contains either a `CostData`
wrapper (with sub-oneofs for actual/projected/estimate results) or a `ResourceError`.

**Rationale**: The `oneof` pattern cleanly separates success from failure at the per-resource
level. The `CostData` wrapper uses nested oneofs to carry the query-type-specific response
payload, preserving full type information without losing the structured error path.

**Findings**:

- `GetActualCostResponse` contains `repeated ActualCostResult results` (proto line ~247)
- `GetProjectedCostResponse` contains `CostEstimate projected_cost` (proto line ~283)
- `EstimateCostResponse` contains `CostEstimate estimate` (proto line ~348)
- Each response type has different fields; a common wrapper needs to accommodate all three
- The `ResourceDescriptor.id` field (proto field 7) already exists for request/response
  correlation

**Alternatives Considered**:

- **Flatten all into one message**: Loses type safety; callers must inspect query type to
  know which fields are populated. Rejected.
- **Use `google.protobuf.Any`**: Too generic; breaks static typing and requires type
  registration. Rejected.
- **Separate result types per query kind**: More ergonomic per-type but requires callers to
  switch on query type anyway. The oneof approach is equivalent with less proto duplication.

## R3: SDK Fallback Architecture - Bounded Parallelism

**Task**: Design the automatic fallback mechanism for plugins without a custom
`BatchCostHandler` implementation.

**Decision**: Worker pool pattern with configurable concurrency (default 10 workers) that
fans out individual RPC calls and assembles results in request order.

**Rationale**: The spec requires "concurrent processing of individual resources with bounded
parallelism (configurable worker pool)" (FR-008). A fixed worker pool prevents resource
exhaustion while providing parallelism benefits.

**Findings**:

- The existing `Serve()` function in `sdk.go` constructs the gRPC server and wires handlers
  via `ConnectHandler` (connect.go)
- Optional interfaces (DryRunHandler, RecommendationsProvider) are detected via type
  assertions in `inferCapabilities()` (plugin_info.go:234-298)
- When an optional interface is not implemented, the `ConnectHandler` returns
  `codes.Unimplemented`
- For batch fallback, instead of returning Unimplemented, the SDK should dispatch to
  the existing individual RPCs concurrently

**Design**:

```go
// BatchCostHandler is the optional interface for optimized batch processing.
type BatchCostHandler interface {
    BatchCost(ctx context.Context, req *pbc.BatchCostRequest) (*pbc.BatchCostResponse, error)
}

// Default fallback in connect.go:
// 1. Check if plugin implements BatchCostHandler -> call it
// 2. Otherwise, fan out to GetActualCost/GetProjectedCost/EstimateCost per resource
// 3. Use semaphore (chan struct{}) for bounded parallelism
// 4. Collect results in slice preserving request order (index-based)
// 5. Convert individual RPC errors to ResourceError entries (no top-level error)
```

**Alternatives Considered**:

- **Sequential fallback only**: Simpler but does not meet FR-008 requirement for concurrent
  processing. Rejected.
- **Unbounded parallelism**: Risk of overwhelming plugins or upstream APIs. Rejected.
- **errgroup with limit**: Go's `errgroup` with `SetLimit()` is an option but the batch
  should not fail-fast on errors (partial failure semantics require all results). A manual
  semaphore + WaitGroup is more appropriate.

## R4: Capability Discovery Integration

**Task**: Determine how batch cost capability is exposed through the existing capability
discovery mechanism.

**Decision**: Add `PLUGIN_CAPABILITY_BATCH_COST = 12` to the `PluginCapability` enum. The
SDK auto-detects the `BatchCostHandler` interface via type assertion, following the existing
pattern.

**Rationale**: This follows the established pattern used by DryRun (5), Recommendations (4),
Budgets (6), and DismissRecommendations (11). The next available enum value is 12.

**Findings**:

- Current max enum value: `PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS = 11`
- `inferCapabilities()` in plugin_info.go:234-298 uses type assertions for detection
- `maxCapabilities` pre-alloc capacity is 8 (4 base + 4 optional) - needs bump to 9
- `maxValidCapability` constant is 11 - needs update to 12
- Legacy metadata map entry: `"supports_batch_cost": "true"`
- `GetPluginInfoResponse` includes `max_batch_size` reporting via a new field or metadata

**Design**:

```go
// In inferCapabilities():
if _, ok := plugin.(BatchCostHandler); ok {
    capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
}
```

**Alternatives Considered**:

- **Reuse existing capability enum values**: No existing value fits batch semantics. Rejected.
- **Report batch support only via metadata**: Inconsistent with enum-based discovery.
  Metadata is for supplementary info (like max_batch_size). Rejected.

## R5: Batch Size Limits and Validation

**Task**: Determine default batch size limits, how they are configured, and how they are
reported to hosts.

**Decision**: Default max batch size of 100 resources. Configurable per-plugin via
`PluginInfo` metadata or a dedicated `MaxBatchSize` field on `ServeConfig`. Reported to
hosts via `GetPluginInfoResponse` metadata (`max_batch_size` key) and as a field on
`BatchCostResponse`.

**Rationale**: The spec assumes a default of 100 (matching GetRecommendations' target_resources
limit). This covers typical Pulumi stacks (10-200 resources) while preventing resource
exhaustion.

**Findings**:

- `GetRecommendationsRequest.target_resources` already has a documented max of 100 (proto
  line ~1186)
- SDK constants pattern: `DefaultPageSize = 50`, `MaxPageSize = 1000` in helpers.go
- Similar constants needed: `DefaultMaxBatchSize = 100`, `MaxBatchSize = 1000`
- Validation happens server-side before dispatching to handler/fallback
- Error on exceed: `codes.InvalidArgument` with descriptive message

**Design**:

```go
const (
    DefaultMaxBatchSize = 100  // Default if plugin doesn't configure
    MaxBatchSize        = 1000 // Hard upper limit
    DefaultBatchWorkers = 10   // Default concurrent workers for fallback
)
```

**Alternatives Considered**:

- **No limit**: Risks memory/timeout issues for large batches. Rejected per FR-007.
- **Per-request limit negotiation**: Over-engineered for current needs. Rejected.

## R6: CostQueryType Enum Design

**Task**: Design the `CostQueryType` enum that selects which cost operation to perform
across the batch.

**Decision**: New enum `CostQueryType` in enums.proto with values: UNSPECIFIED (0),
ESTIMATE (1), ACTUAL (2), PROJECTED (3).

**Rationale**: Maps directly to the three existing cost RPCs. UNSPECIFIED defaults to
ESTIMATE per the spec's edge case definition ("What happens when the query type is
UNSPECIFIED? Defaults to estimate cost").

**Findings**:

- Three existing cost RPCs: `EstimateCost`, `GetActualCost`, `GetProjectedCost`
- Each has distinct request parameters: EstimateCost uses ResourceDescriptor directly;
  GetActualCost needs start/end timestamps; GetProjectedCost needs no time range
- The batch request must carry shared time range for ACTUAL queries
- Zero-allocation validation follows the registry package pattern (package-level slice)

**Design**:

```protobuf
enum CostQueryType {
  COST_QUERY_TYPE_UNSPECIFIED = 0;
  COST_QUERY_TYPE_ESTIMATE = 1;
  COST_QUERY_TYPE_ACTUAL = 2;
  COST_QUERY_TYPE_PROJECTED = 3;
}
```

**Alternatives Considered**:

- **Reuse existing enums**: No existing enum covers this; cost types are implicit in RPC
  names today. Rejected.
- **String-based discriminator**: Inconsistent with enum-first proto patterns. Rejected.

## R7: Time Range Handling for Actual Cost Queries

**Task**: Determine how time range parameters are handled in batch requests, given that
only actual cost queries require start/end timestamps.

**Decision**: Include optional `google.protobuf.Timestamp start` and
`google.protobuf.Timestamp end` fields on `BatchCostRequest`. These are required when
`query_type` is ACTUAL and ignored for ESTIMATE/PROJECTED. Server-side validation enforces
this.

**Rationale**: The spec clarifies that time range is "shared across the batch" (Clarification
Session 2026-02-24). Making the fields optional on the batch request and validating based
on query type keeps the message simple while enforcing correct usage.

**Findings**:

- `GetActualCostRequest` has `start` (field 2) and `end` (field 3) as
  `google.protobuf.Timestamp`
- Validation: start < end, both required for actual cost queries
- For ESTIMATE/PROJECTED, these fields are ignored if present (no error)

**Alternatives Considered**:

- **Per-resource time ranges**: Over-complicated; spec explicitly chose shared time range.
  Rejected.
- **Require time range for all query types**: Unnecessarily restrictive for
  ESTIMATE/PROJECTED. Rejected.

## R8: TypeScript SDK Parity

**Task**: Determine the TypeScript SDK changes needed for batch cost support.

**Decision**: Add `batchCost()` method to `CostSourceClient` and a `batchCostIterator()`
utility if pagination is needed per-resource. Proto bindings are auto-generated by buf.

**Rationale**: The TypeScript SDK follows the same patterns as Go: client wrapper methods
around the generated Connect client, with utility functions for common patterns.

**Findings**:

- TypeScript client at `sdk/typescript/packages/client/src/clients/cost-source.ts`
- Methods wrap `this.client.<method>(req)` with optional validation
- Pagination iterators in `sdk/typescript/packages/client/src/utils/pagination.ts`
- Proto regeneration: `buf generate` produces TypeScript bindings automatically
- Tests use vitest + msw (mock service worker)

**Design**:

```typescript
// In CostSourceClient
async batchCost(request: BatchCostRequest): Promise<BatchCostResponse> {
  return this.client.batchCost(request);
}
```

**Alternatives Considered**:

- **Skip TypeScript SDK**: Violates Constitution XIII (Multi-Language SDK Sync). Rejected.

## R9: DryRun Integration with Batch

**Task**: Determine how the existing DryRun capability integrates with batch requests.

**Decision**: Add an optional `dry_run` boolean field to `BatchCostRequest`. When set, the
batch operation returns `DryRunResult` per resource instead of cost data. This reuses the
existing `DryRunResult` message type.

**Rationale**: The spec (FR-011) requires dry-run support on batch requests. Reusing the
existing `DryRunResult` message maintains consistency with the standalone DryRun RPC.

**Findings**:

- Existing `DryRunResult` message contains field mappings per resource type
- `GetActualCostRequest` already has a `dry_run` field (proto field 6)
- The batch dry-run returns field mappings per resource, not cost data
- Each `ResourceCostResult` in dry-run mode contains a `DryRunResult` in the oneof

**Alternatives Considered**:

- **Separate BatchDryRun RPC**: Additional proto surface for a flag that already exists on
  individual RPCs. Rejected for simplicity.
- **Ignore dry-run in batch**: Violates FR-011. Rejected.
