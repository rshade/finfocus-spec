# Quickstart: EstimateCost expires_at Cache-Hint Parity

**Feature Branch**: `049-estimate-cost-expires-at`

## For Plugin Developers

### Setting expires_at on Estimate Cost Responses

```go
import (
    "time"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func (p *MyPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (
    *pbc.EstimateCostResponse, error) {

    // Fixed pricing: cache for 24 hours (pricing rarely changes)
    resp := pluginsdk.NewEstimateCostResponse(
        pluginsdk.WithEstimateCost("USD", 36.50),
        pluginsdk.WithEstimateCostExpiresAt(time.Now().Add(24 * time.Hour)),
    )

    return resp, nil
}
```

### Volatile Pricing (Short TTL)

```go
func (p *MyPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (
    *pbc.EstimateCostResponse, error) {

    // Spot pricing: short TTL because prices change frequently
    resp := pluginsdk.NewEstimateCostResponse(
        pluginsdk.WithEstimateCost("USD", 12.80),
        pluginsdk.WithPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
        pluginsdk.WithSpotRisk(0.7),
        pluginsdk.WithEstimateCostExpiresAt(time.Now().Add(5 * time.Minute)),
    )

    return resp, nil
}
```

### No Caching Guidance

```go
func (p *MyPlugin) EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (
    *pbc.EstimateCostResponse, error) {

    // Real-time pricing: no caching hint (callers should always re-fetch)
    resp := pluginsdk.NewEstimateCostResponse(
        pluginsdk.WithEstimateCost("USD", 42.00),
    )
    // expires_at is nil by default - no caching guidance

    return resp, nil
}
```

## For Callers (CLI / Host Applications)

### Uniform Caching Across All Cost RPCs

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Same caching pattern works for all three cost response types
func cacheEstimate(resp *pbc.EstimateCostResponse) {
    if pluginsdk.IsEstimateCostExpired(resp, time.Now()) {
        // Estimate is stale, re-fetch from plugin
        return
    }

    // Get TTL for cache management
    if expiresAt, ok := pluginsdk.EstimateCostExpiresAt(resp); ok {
        ttl := time.Until(expiresAt)
        cache.SetWithTTL(cacheKey, resp, ttl)
    }
}
```

### Handling Missing expires_at

```go
// No expires_at set = always re-fetch (no caching assumption)
if expiresAt, ok := pluginsdk.EstimateCostExpiresAt(resp); !ok {
    // Plugin didn't set expires_at - fetch fresh data every time
    refreshEstimate(resp)
} else if expiresAt.Before(time.Now()) {
    // Plugin set expires_at but it's in the past - stale
    refreshEstimate(resp)
} else {
    // Estimate is still valid
    useCachedEstimate(resp)
}
```

### Nil-Safe Usage

```go
// All helpers handle nil inputs gracefully
var resp *pbc.EstimateCostResponse = nil

// Safe: returns false (not expired because there's nothing to check)
expired := pluginsdk.IsEstimateCostExpired(resp, time.Now())

// Safe: returns zero time and false
expiresAt, ok := pluginsdk.EstimateCostExpiresAt(resp)
```

## Testing

### Using Mock Plugin with Estimate Caching Hints

```go
mock := plugintesting.NewMockPlugin()
mock.EstimateCostExpiresAtDuration = 1 * time.Hour // Estimates expire in 1 hour

harness := plugintesting.NewTestHarness(mock)
harness.Start(t)
defer harness.Stop()

// Verify expires_at is set on estimate response
resp, err := harness.Client().EstimateCost(ctx, &pbc.EstimateCostRequest{
    ResourceType: "aws_instance",
})
require.NoError(t, err)
require.NotNil(t, resp.ExpiresAt, "expires_at should be set")

expiresAt := resp.ExpiresAt.AsTime()
require.True(t, expiresAt.After(time.Now()), "expires_at should be in the future")
```

### Testing Stale Estimates

```go
mock := plugintesting.NewMockPlugin()
mock.EstimateCostExpiresAtDuration = -1 * time.Hour // Already expired

harness := plugintesting.NewTestHarness(mock)
harness.Start(t)
defer harness.Stop()

resp, err := harness.Client().EstimateCost(ctx, req)
require.NoError(t, err)
require.True(t, pluginsdk.IsEstimateCostExpired(resp, time.Now()))
```

### Testing No Caching Guidance (Backward Compatible)

```go
mock := plugintesting.NewMockPlugin()
// EstimateCostExpiresAtDuration defaults to 0 (no expires_at)

harness := plugintesting.NewTestHarness(mock)
harness.Start(t)
defer harness.Stop()

resp, err := harness.Client().EstimateCost(ctx, req)
require.NoError(t, err)
require.Nil(t, resp.ExpiresAt, "expires_at should be nil when duration is zero")
```
