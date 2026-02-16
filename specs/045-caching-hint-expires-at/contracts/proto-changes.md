# Proto Contract Changes: expires_at Caching Hint

**Source**: `proto/finfocus/v1/costsource.proto`

## ActualCostResult - Add expires_at (field 8)

```proto
// ActualCostResult represents a single cost data point.
message ActualCostResult {
  // timestamp indicates the point-in-time or bucket start for this cost data
  google.protobuf.Timestamp timestamp = 1;
  // cost is the total cost in the specified currency for the period/bucket
  double cost = 2;
  // usage_amount is the optional usage amount aligned with BillingMode
  double usage_amount = 3;
  // usage_unit specifies the unit of usage (e.g., "hour", "GB", "request")
  string usage_unit = 4;
  // source identifies the data source (e.g., "kubecost", "flexera")
  string source = 5;
  // focus_record provides the cost data in FOCUS 1.2 format.
  // This field is optional and will eventually replace the legacy fields.
  FocusCostRecord focus_record = 6;
  // impact_metrics contains sustainability metrics (Carbon, Energy, etc.)
  repeated ImpactMetric impact_metrics = 7;
  // expires_at is a caching hint indicating when this cost data should be
  // considered stale. Callers MAY use this to manage local caches and avoid
  // re-requesting data before the expiration time.
  //
  // Semantics:
  //   - nil/unset: No caching guidance. Caller should re-fetch on each request.
  //   - Past timestamp: Data is immediately stale (equivalent to no hint).
  //   - Future timestamp: Data is valid until this time.
  //
  // This field is advisory. Callers MAY apply their own maximum TTL policy
  // regardless of the value set by the plugin.
  //
  // Use cases:
  //   - Rate-limited APIs: Set 6-12 hours to prevent excessive upstream calls
  //   - Billing cycle alignment: Set to end of current billing period
  //   - Real-time sources: Leave unset (no caching)
  google.protobuf.Timestamp expires_at = 8;
}
```

## GetProjectedCostResponse - Add expires_at (field 13)

```proto
// GetProjectedCostResponse contains projected cost information.
message GetProjectedCostResponse {
  // ... existing fields 1-12 ...

  // expires_at is a caching hint indicating when this projected cost data
  // should be considered stale. Callers MAY use this to manage local caches
  // and avoid re-requesting projections before the expiration time.
  //
  // Semantics:
  //   - nil/unset: No caching guidance. Caller should re-fetch on each request.
  //   - Past timestamp: Data is immediately stale (equivalent to no hint).
  //   - Future timestamp: Data is valid until this time.
  //
  // This field is advisory. Callers MAY apply their own maximum TTL policy
  // regardless of the value set by the plugin.
  //
  // Use cases:
  //   - Pricing cycle boundaries: Set to end of billing cycle (e.g., month-end)
  //   - Stable pricing: Set far future for resources with fixed pricing
  //   - Volatile pricing: Leave unset or set short TTL
  google.protobuf.Timestamp expires_at = 13;
}
```

## Wire Format Compatibility

- Both additions are backward-compatible (new fields with default nil value)
- No existing field numbers changed
- No reserved fields affected
- `buf breaking` check will pass (additive-only change)
