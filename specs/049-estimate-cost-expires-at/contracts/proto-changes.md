# Proto Contract Changes: EstimateCost expires_at

**Source**: `proto/finfocus/v1/costsource.proto`

## EstimateCostResponse - Add expires_at (field 5)

```proto
// EstimateCostResponse contains estimated cost information for a resource.
message EstimateCostResponse {
  // currency is the ISO 4217 currency code for the cost estimate
  string currency = 1;
  // cost_monthly is the estimated monthly cost
  double cost_monthly = 2;
  // pricing_category indicates the pricing model (Standard/Committed/Dynamic)
  FocusPricingCategory pricing_category = 3;
  // spot_interruption_risk_score is the probability of spot instance interruption
  double spot_interruption_risk_score = 4;
  // expires_at is a caching hint indicating when this cost estimate should be
  // considered stale. Callers MAY use this to manage local caches and avoid
  // re-requesting estimates before the expiration time.
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
  //   - Fixed pricing: Set far future for resources with stable pricing
  //   - Volatile/spot pricing: Set short TTL or leave unset
  //   - Rate-limited APIs: Set to avoid redundant upstream calls
  google.protobuf.Timestamp expires_at = 5;
}
```

## Wire Format Compatibility

- Addition is backward-compatible (new field with default nil value)
- No existing field numbers changed
- No reserved fields affected
- `buf breaking` check will pass (additive-only change)
- Existing plugins compiled against older spec continue to work (field defaults to nil)
- Newer callers reading responses from older plugins see nil (no caching guidance)
