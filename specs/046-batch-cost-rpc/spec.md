# Feature Specification: Batch Cost RPC

**Feature Branch**: `046-batch-cost-rpc`
**Created**: 2026-02-24
**Status**: Draft
**Input**: GitHub Issue #221 - Add dedicated batch RPC for efficient multi-resource cost queries

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Batch Cost Query for Resource Stacks (Priority: P1)

As a host application operator managing cloud infrastructure, I want to query cost data
for multiple resources in a single request so that I can reduce network overhead and get
faster results when aggregating costs across an entire resource stack.

**Why this priority**: This is the core value proposition. Without efficient multi-resource
querying, operators must make N individual requests for N resources, creating O(N) network
round-trips and preventing plugin-side query optimization.

**Independent Test**: Can be fully tested by sending a batch request with 5-10 resources
of mixed types and verifying that cost results are returned for each resource in a single
response, delivering immediate latency and throughput improvements.

**Acceptance Scenarios**:

1. **Given** a host with 10 resources across 2 providers, **When** a batch cost request
   is sent with all 10 resource descriptors, **Then** cost results are returned for all
   10 resources in a single response.
2. **Given** a batch request with a supported query type (actual, projected, or estimate),
   **When** the plugin processes the batch, **Then** each result corresponds to the
   requested cost type.
3. **Given** a batch request with resources the plugin supports, **When** the plugin has
   optimized batch API access to the cloud provider, **Then** the total query time is less
   than making equivalent individual requests.

---

### User Story 2 - Partial Failure Handling (Priority: P1)

As a host application, I want batch responses to include per-resource error information
so that I can handle failures for individual resources without losing results for
successful ones.

**Why this priority**: Partial failure semantics are critical for production reliability.
A single unsupported resource must not cause the entire batch to fail, as this would make
the feature unusable for heterogeneous resource sets.

**Independent Test**: Can be fully tested by including a mix of supported and unsupported
resource types in a batch request and verifying that supported resources return cost data
while unsupported ones return structured error information.

**Acceptance Scenarios**:

1. **Given** a batch of 5 resources where 3 are supported and 2 are unsupported,
   **When** the batch is processed, **Then** the response contains 3 cost results and
   2 resource errors with appropriate error codes.
2. **Given** a batch where one resource causes a transient error, **When** the batch is
   processed, **Then** the error response for that resource includes a meaningful error
   code and message while other resources succeed.
3. **Given** a batch where all resources fail, **When** the batch is processed, **Then**
   the response contains error entries for all resources (not a top-level gRPC error).

---

### User Story 3 - Plugin Automatic Batch Fallback (Priority: P2)

As a plugin developer, I want the option to either implement optimized batch processing
or have the SDK automatically fall back to concurrent processing with bounded parallelism
so that I can adopt the batch RPC without rewriting my plugin logic.

**Why this priority**: Lowering the adoption barrier ensures plugins can immediately
support batch requests. Optimized implementations can be added incrementally.

**Independent Test**: Can be fully tested by implementing a plugin without a batch
handler and verifying that batch requests are still served correctly via automatic
sequential fallback.

**Acceptance Scenarios**:

1. **Given** a plugin that does not implement a batch handler, **When** a batch request
   arrives, **Then** the SDK processes resources concurrently using existing RPCs with
   bounded parallelism and assembles the batch response.
2. **Given** a plugin that implements a custom batch handler, **When** a batch request
   arrives, **Then** the custom handler is invoked instead of the sequential fallback.

---

### User Story 4 - Batch Size Limits and Capability Discovery (Priority: P2)

As a host application, I want to discover the maximum batch size a plugin supports so
that I can split large requests appropriately and avoid overwhelming plugins.

**Why this priority**: Without batch size limits, hosts risk sending requests that exceed
plugin capacity, causing timeouts or resource exhaustion. Capability discovery enables
intelligent request planning.

**Independent Test**: Can be fully tested by querying plugin capabilities and verifying
that the batch size limit is reported, then sending a request exceeding that limit and
verifying the appropriate error.

**Acceptance Scenarios**:

1. **Given** a plugin that supports batch operations, **When** the host queries
   capabilities, **Then** the response includes the maximum supported batch size.
2. **Given** a batch request exceeding the plugin's maximum batch size, **When** the
   request is received, **Then** the plugin returns an appropriate error indicating the
   batch size limit was exceeded.
3. **Given** a plugin that does not specify a batch size limit, **When** the host queries
   capabilities, **Then** a sensible default limit is assumed.

---

### User Story 5 - Batch DryRun Support (Priority: P3)

As a host application, I want to perform a batch dry-run query to discover field mapping
capabilities for multiple resource types in a single request, without incurring the cost
of actual data retrieval.

**Why this priority**: Extends the existing DryRun pattern to batch operations. Lower
priority because DryRun is already an optional capability and batch DryRun is additive.

**Independent Test**: Can be fully tested by sending a batch request with the dry-run
flag set and verifying that field mapping information is returned for each resource type
without actual cost data.

**Acceptance Scenarios**:

1. **Given** a batch request with the dry-run flag enabled, **When** the plugin processes
   the batch, **Then** field mapping results are returned per resource type without
   querying external cost APIs.

---

### Edge Cases

- What happens when a batch request contains zero resources? The system returns an empty
  response, not an error.
- What happens when the same resource appears multiple times in a batch? Each occurrence
  is processed independently; deduplication is the host's responsibility.
- What happens when a batch request is sent to a plugin that does not support the batch
  capability? The system returns an `Unimplemented` status code, consistent with existing
  optional RPC patterns like DryRun and GetPluginInfo.
- How does pagination interact with batch requests? Each resource's cost data within the
  batch may be paginated independently via existing pagination mechanisms.
- What happens when the batch request exceeds the maximum batch size? The system returns
  an appropriate error indicating the limit was exceeded.
- What happens when the query type is UNSPECIFIED? Defaults to estimate cost, matching
  the most common use case for batch operations.
- What happens when `dry_run=true` AND `query_type=ACTUAL` with time range parameters?
  Time range validation is skipped because dry-run does not perform actual cost retrieval;
  the query type is used only to determine which fields to map.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a batch cost query operation that accepts multiple
  resource descriptors and returns cost results for each resource in a single
  request-response cycle.
- **FR-002**: System MUST support specifying the cost query type (estimate, actual, or
  projected) for the entire batch.
- **FR-002a**: Batch requests for actual cost queries MUST include shared time range
  parameters (start/end) that apply uniformly to all resources in the batch.
- **FR-003**: System MUST return per-resource results, where each result is either cost
  data or a structured error, enabling partial failure handling.
- **FR-004**: System MUST return results in the same order as the request's resource
  descriptors, enabling index-based correlation between request and response.
- **FR-005**: Per-resource errors MUST include an error code, a human-readable message,
  and an indication of whether the resource type is unsupported by the plugin.
- **FR-006**: System MUST support a configurable maximum batch size per plugin, with a
  sensible default for plugins that do not specify a limit.
- **FR-007**: System MUST reject batch requests that exceed the maximum batch size with a
  clear error.
- **FR-008**: System MUST provide a fallback mechanism that allows plugins without custom
  batch implementations to serve batch requests via concurrent processing of individual
  resources with bounded parallelism (configurable worker pool).
- **FR-009**: System MUST expose batch capability through the existing capability
  discovery mechanism so that hosts can determine batch support and limits before
  sending requests.
- **FR-010**: System MUST handle empty batch requests gracefully by returning an empty
  response.
- **FR-011**: System MUST support the dry-run flag on batch requests, returning field
  mapping information per resource type without actual cost data retrieval.
- **FR-012**: System MUST maintain backward compatibility such that plugins not
  implementing the batch RPC continue to function without modification.

### Key Entities

- **BatchCostRequest**: A collection of resource descriptors with a shared cost query
  type and shared time range parameters (start/end), representing the input to the batch
  operation. The time range applies uniformly to all resources in the batch.
- **BatchCostResponse**: A positionally-ordered collection of per-resource results where
  results[i] corresponds to request resources[i], each containing either cost data or
  error information.
- **ResourceCostResult**: A single resource's outcome within a batch, pairing the
  resource descriptor with its cost data or error.
- **ResourceError**: Structured error information for a failed resource within a batch,
  including error code, message, and unsupported-resource indicator.
- **CostQueryType**: Enumeration specifying which cost operation to perform across the
  batch (estimate, actual, or projected).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A batch request for 50 resources completes in less time than 50 equivalent
  individual requests made sequentially.
- **SC-002**: Partial failures do not prevent successful resources from returning results;
  a batch with 80% supported resources returns cost data for at least 80% of resources.
- **SC-003**: Plugins without custom batch handlers can serve batch requests without code
  changes via the SDK's automatic sequential fallback.
- **SC-004**: Batch capability and size limits are discoverable by hosts before sending
  batch requests, enabling intelligent request planning.
- **SC-005**: All existing plugin implementations continue to function without
  modification after the batch RPC is added.
- **SC-006**: Batch requests with the dry-run flag return field mapping information for
  each resource type within <100ms p99 latency per resource (matching the individual
  DryRun RPC requirement from spec 032-plugin-dry-run).

## Clarifications

### Session 2026-02-24

- Q: Should time range parameters (start/end) be shared across the batch or
  per-resource? → A: Shared time range fields on the batch request; start/end
  apply uniformly to all resources.
- Q: Should SDK fallback processing be sequential or concurrent? → A: Concurrent
  with bounded parallelism (configurable worker pool).
- Q: Should batch results maintain request ordering for correlation? → A: Yes,
  results MUST be in the same order as request resource descriptors (index-based
  correlation).

## Assumptions

- Batch requests share a single cost query type across all resources; mixed-type queries
  (e.g., some actual, some projected) are out of scope for this iteration.
- The default maximum batch size is 100 resources, which covers the majority of
  production use cases (typical Pulumi stacks have 10-200 resources).
- Streaming responses for very large batches are deferred to a future enhancement; this
  iteration uses unary request-response.
- Resource deduplication within a batch is the host's responsibility; the plugin processes
  each entry independently.
- Batch requests reuse the existing `ResourceDescriptor` message rather than defining a
  new resource identification format.
- The batch RPC follows the same optional-capability pattern as DryRun and
  GetPluginInfo, returning `Unimplemented` for plugins that do not support it.
- Time range parameters (start/end) for actual cost queries apply uniformly across all
  resources in the batch.

## Dependencies

- Existing `ResourceDescriptor` message type in `costsource.proto`.
- Existing `PluginCapability` enum for capability discovery.
- Existing SDK fallback and capability inference patterns.
- Existing mock plugin and test harness infrastructure for testing.

## Out of Scope

- Streaming batch responses for very large batches (future enhancement).
- Mixed cost query types within a single batch request.
- Server-side resource deduplication.
- Batch operations for non-cost RPCs (e.g., batch recommendations).
- Cross-plugin batch routing (host-side orchestration).
