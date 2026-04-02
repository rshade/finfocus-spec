# SDK Helper Contracts: EstimateCost expires_at

## Go SDK - Functional Option

### WithEstimateCostExpiresAt

```go
// WithEstimateCostExpiresAt returns an EstimateCostResponseOption that sets
// the expires_at caching hint on the estimate cost response.
//
// A zero time.Time results in a nil expires_at (no caching guidance).
//
// Usage:
//
//   resp := pluginsdk.NewEstimateCostResponse(
//       pluginsdk.WithEstimateCost("USD", 50.0),
//       pluginsdk.WithEstimateCostExpiresAt(time.Now().Add(24 * time.Hour)),
//   )
func WithEstimateCostExpiresAt(expiresAt time.Time) EstimateCostResponseOption
```

Note: Uses the existing `EstimateCostResponseOption` type already defined in `helpers.go`.
Integrates with the existing `NewEstimateCostResponse()` builder function.

## Go SDK - Expiration Check Helpers

### IsEstimateCostExpired

```go
// IsEstimateCostExpired returns true if the estimate cost response has an
// expires_at timestamp that is before the given reference time.
//
// Returns false if the response is nil or expires_at is nil/unset (no
// expiration guidance means the caller cannot determine expiration status
// from this field alone).
//
// Usage:
//
//   if pluginsdk.IsEstimateCostExpired(resp, time.Now()) {
//       // Re-fetch estimate from plugin
//   }
func IsEstimateCostExpired(resp *pbc.EstimateCostResponse, now time.Time) bool
```

### EstimateCostExpiresAt

```go
// EstimateCostExpiresAt returns the expiration time for an estimate cost
// response. The second return value is false if the response is nil or
// expires_at is nil/unset.
//
// Usage:
//
//   if expiresAt, ok := pluginsdk.EstimateCostExpiresAt(resp); ok {
//       ttl := time.Until(expiresAt)
//       cache.SetWithTTL(key, resp, ttl)
//   }
func EstimateCostExpiresAt(resp *pbc.EstimateCostResponse) (time.Time, bool)
```

## Mock Plugin Configuration

### MockPlugin Fields

```go
// EstimateCostExpiresAtDuration configures the expires_at hint for estimate
// cost responses. When non-zero, EstimateCost will have expires_at set to
// time.Now().Add(EstimateCostExpiresAtDuration).
// When zero (default), expires_at is not set (backward compatible).
// Negative values produce past timestamps (immediately-stale semantics).
EstimateCostExpiresAtDuration time.Duration
```

## Complete expires_at Helper Symmetry

After implementation, all three cost response types have identical helper patterns:

| Helper Pattern | ActualCostResult | GetProjectedCostResponse | EstimateCostResponse |
|----------------|-----------------|-------------------------|---------------------|
| Reader | `ActualCostExpiresAt()` | `ProjectedCostExpiresAt()` | `EstimateCostExpiresAt()` |
| Checker | `IsActualCostExpired()` | `IsProjectedCostExpired()` | `IsEstimateCostExpired()` |
| Option | `WithActualCostResultExpiresAt()` | `WithProjectedCostExpiresAt()` | `WithEstimateCostExpiresAt()` |
| Mock Config | `ExpiresAtDuration` | `ProjectedCostExpiresAtDuration` | `EstimateCostExpiresAtDuration` |
