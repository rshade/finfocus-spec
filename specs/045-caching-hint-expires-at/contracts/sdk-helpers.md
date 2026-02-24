# SDK Helper Contracts: expires_at Caching Hint

## Go SDK - Response Option Functions

### Setting expires_at on ActualCostResult (Direct Field Access)

Plugin developers set `expires_at` directly on `ActualCostResult` using the generated
proto field. No SDK wrapper is needed because plugins construct results individually:

```go
// Direct field access (recommended for ActualCostResult)
result := &pbc.ActualCostResult{
    Timestamp: timestamppb.New(dataTime),
    Cost:      42.50,
    ExpiresAt: timestamppb.New(time.Now().Add(6 * time.Hour)),
}
```

### WithProjectedCostExpiresAt

```go
// WithProjectedCostExpiresAt returns a GetProjectedCostResponseOption that sets
// the expires_at caching hint on the projected cost response.
//
// A zero time.Time results in a nil expires_at (no caching guidance).
//
// Usage:
//
//   resp := pluginsdk.NewGetProjectedCostResponse(
//       pluginsdk.WithProjectedCostExpiresAt(time.Now().Add(24 * time.Hour)),
//   )
func WithProjectedCostExpiresAt(expiresAt time.Time) GetProjectedCostResponseOption
```

## Go SDK - Expiration Check Helpers

### IsActualCostExpired

```go
// IsActualCostExpired returns true if the cost result has an expires_at
// timestamp that is before the given reference time.
//
// Returns false if expires_at is nil/unset (no expiration guidance means
// the caller cannot determine expiration status from this field alone).
//
// Usage:
//
//   if pluginsdk.IsActualCostExpired(result, time.Now()) {
//       // Re-fetch from plugin
//   }
func IsActualCostExpired(result *pbc.ActualCostResult, now time.Time) bool
```

### ActualCostExpiresAt

```go
// ActualCostExpiresAt returns the expiration time for a cost result.
// The second return value is false if expires_at is nil/unset.
//
// Usage:
//
//   if expiresAt, ok := pluginsdk.ActualCostExpiresAt(result); ok {
//       ttl := time.Until(expiresAt)
//       cache.SetWithTTL(key, result, ttl)
//   }
func ActualCostExpiresAt(result *pbc.ActualCostResult) (time.Time, bool)
```

### IsProjectedCostExpired

```go
// IsProjectedCostExpired returns true if the projected cost response has an
// expires_at timestamp that is before the given reference time.
//
// Returns false if expires_at is nil/unset.
func IsProjectedCostExpired(resp *pbc.GetProjectedCostResponse, now time.Time) bool
```

### ProjectedCostExpiresAt

```go
// ProjectedCostExpiresAt returns the expiration time for a projected cost response.
// The second return value is false if expires_at is nil/unset.
func ProjectedCostExpiresAt(resp *pbc.GetProjectedCostResponse) (time.Time, bool)
```

## Mock Plugin Configuration

### MockPlugin Fields

```go
// ExpiresAtDuration configures the expires_at hint for generated cost results.
// When non-zero, each ActualCostResult will have expires_at set to
// time.Now().Add(ExpiresAtDuration).
// When zero (default), expires_at is not set (backward compatible).
ExpiresAtDuration time.Duration

// ProjectedCostExpiresAtDuration configures the expires_at hint for projected
// cost responses. Same semantics as ExpiresAtDuration.
ProjectedCostExpiresAtDuration time.Duration
```
