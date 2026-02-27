# Proto Contract Changes: Batch Cost RPC

**Feature**: 046-batch-cost-rpc | **Date**: 2026-02-24

## Changes to `proto/finfocus/v1/enums.proto`

### New CostQueryType Enum

```protobuf
+ // CostQueryType specifies which cost operation to perform in a batch query.
+ // Used by BatchCostRequest to select the underlying cost RPC for all resources.
+ enum CostQueryType {
+   // Default value; treated as ESTIMATE for backward compatibility.
+   COST_QUERY_TYPE_UNSPECIFIED = 0;
+   // Use EstimateCost logic for each resource (pre-deployment estimates).
+   COST_QUERY_TYPE_ESTIMATE = 1;
+   // Use GetActualCost logic for each resource (historical cost data).
+   // Requires start/end timestamps in BatchCostRequest.
+   COST_QUERY_TYPE_ACTUAL = 2;
+   // Use GetProjectedCost logic for each resource (cost projections).
+   COST_QUERY_TYPE_PROJECTED = 3;
+ }
```

### New PluginCapability Value

```protobuf
  enum PluginCapability {
    // ... existing values 0-11 ...
    PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS = 11;
+   // Plugin implements BatchCost RPC for efficient multi-resource cost queries
+   PLUGIN_CAPABILITY_BATCH_COST = 12;
  }
```

## Changes to `proto/finfocus/v1/costsource.proto`

### New RPC in CostSourceService

```protobuf
  service CostSourceService {
    // ... existing RPCs ...
    rpc DryRun(DryRunRequest) returns (DryRunResponse);
+
+   // BatchCost queries cost data for multiple resources in a single request.
+   // Supports estimate, actual, and projected cost queries via the query_type
+   // field. Returns per-resource results with partial failure semantics -
+   // individual resource failures do not cause the entire batch to fail.
+   //
+   // This is an optional RPC. Plugins that do not implement BatchCostHandler
+   // will have requests served via the SDK's automatic sequential fallback
+   // with bounded parallelism.
+   //
+   // Returns UNIMPLEMENTED only if the SDK does not support batch operations.
+   // Returns INVALID_ARGUMENT if the batch exceeds the plugin's maximum size.
+   rpc BatchCost(BatchCostRequest) returns (BatchCostResponse);
  }
```

### New Messages

```protobuf
+ // BatchCostRequest contains multiple resource descriptors to query costs for
+ // in a single request-response cycle. All resources share the same query type
+ // and time range parameters.
+ message BatchCostRequest {
+   // Resources to query costs for. Maximum count is plugin-dependent
+   // (reported via max_batch_size in BatchCostResponse and GetPluginInfo
+   // metadata). Empty list returns an empty response.
+   repeated ResourceDescriptor resources = 1;
+
+   // The cost operation to perform for all resources in the batch.
+   // UNSPECIFIED is treated as ESTIMATE.
+   CostQueryType query_type = 2;
+
+   // Start of the time range for actual cost queries.
+   // Required when query_type is COST_QUERY_TYPE_ACTUAL.
+   // Ignored for ESTIMATE and PROJECTED queries.
+   google.protobuf.Timestamp start = 3;
+
+   // End of the time range for actual cost queries.
+   // Required when query_type is COST_QUERY_TYPE_ACTUAL. Must be after start.
+   // Ignored for ESTIMATE and PROJECTED queries.
+   google.protobuf.Timestamp end = 4;
+
+   // When true, return field mapping information per resource type without
+   // querying external cost APIs. Reuses the existing DryRunResult message.
+   bool dry_run = 5;
+ }
+
+ // BatchCostResponse contains per-resource results in the same order as the
+ // request's resource descriptors. results[i] corresponds to
+ // request.resources[i].
+ message BatchCostResponse {
+   // Per-resource results. Each entry contains either cost data or error
+   // information. Length equals the request's resources length.
+   repeated ResourceCostResult results = 1;
+
+   // The maximum batch size this plugin supports. Hosts can use this to
+   // plan future batch requests.
+   int32 max_batch_size = 2;
+ }
+
+ // ResourceCostResult represents a single resource's outcome within a batch
+ // response. Contains either cost data (success) or error information
+ // (failure).
+ message ResourceCostResult {
+   // The resource descriptor from the request, echoed back for correlation.
+   ResourceDescriptor resource = 1;
+
+   // The result for this resource - either cost data or an error.
+   oneof result {
+     // Cost data for the resource (success path).
+     CostData cost_data = 2;
+     // Error information for the resource (failure path).
+     ResourceError error = 3;
+   }
+ }
+
+ // CostData wraps the query-type-specific cost response for a single resource
+ // within a batch. Exactly one field is populated based on the request's
+ // query_type (or dry_run flag).
+ message CostData {
+   oneof data {
+     // Actual cost results for the resource (query_type=ACTUAL).
+     ActualCostData actual_cost = 1;
+     // Projected cost for the resource (query_type=PROJECTED).
+     CostEstimate projected_cost = 2;
+     // Cost estimate for the resource (query_type=ESTIMATE).
+     CostEstimate estimate = 3;
+     // Field mapping information (dry_run=true).
+     DryRunResult dry_run_result = 4;
+   }
+ }
+
+ // ActualCostData wraps actual cost results for a single resource within a
+ // batch response. Contains the same fields as GetActualCostResponse but
+ // scoped to one resource.
+ message ActualCostData {
+   // Cost data points for the requested time range.
+   repeated ActualCostResult results = 1;
+
+   // Hint for the host on whether to try other plugins for this resource.
+   FallbackHint fallback_hint = 2;
+
+   // Pagination token for retrieving additional results for this resource.
+   // If empty, all results for this resource have been returned.
+   string next_page_token = 3;
+
+   // Total number of cost data points available for this resource.
+   int32 total_count = 4;
+ }
+
+ // ResourceError contains structured error information for a single resource
+ // that failed within a batch. Per-resource errors do not cause the entire
+ // batch RPC to fail.
+ message ResourceError {
+   // gRPC-compatible status code (e.g., NOT_FOUND=5, INTERNAL=13,
+   // UNAVAILABLE=14, UNIMPLEMENTED=12).
+   int32 code = 1;
+
+   // Human-readable error description.
+   string message = 2;
+
+   // True if the error is specifically because the plugin does not support
+   // the resource's type. Enables hosts to route to alternative plugins.
+   bool resource_type_unsupported = 3;
+ }
```

## Backward Compatibility Analysis

| Change | Type | Breaking? | Notes |
|--------|------|-----------|-------|
| New `BatchCost` RPC | Additive | No | Optional RPC; existing plugins return Unimplemented or use SDK fallback |
| New `CostQueryType` enum | Additive | No | New enum type; no existing fields affected |
| New `PLUGIN_CAPABILITY_BATCH_COST = 12` | Additive | No | New enum value; existing values unchanged |
| New messages (6 total) | Additive | No | New message types; no existing messages modified |

**buf breaking check**: All changes are additive (new RPC, new messages, new enum values).
No existing field numbers, types, or names are modified. Expected result: PASS.

## Field Number Allocation

| Message | Field Numbers Used | Notes |
|---------|-------------------|-------|
| BatchCostRequest | 1-5 | New message |
| BatchCostResponse | 1-2 | New message |
| ResourceCostResult | 1-3 | New message; 2-3 are oneof |
| CostData | 1-4 | New message; all oneof |
| ActualCostData | 1-4 | New message |
| ResourceError | 1-3 | New message |
| PluginCapability | 12 | Next available after DISMISS_RECOMMENDATIONS=11 |
| CostQueryType | 0-3 | New enum |
