# Quickstart: Caching Hint (expires_at) for Cost Results

**Feature Branch**: `045-caching-hint-expires-at`

## For Plugin Developers

### Setting expires_at on Actual Cost Results

```go
import (
    "time"

    "google.golang.org/protobuf/types/known/timestamppb"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func (p *MyPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (
    *pbc.GetActualCostResponse, error) {

    // Fetch cost data from upstream API...
    results := fetchCosts(req)

    // Set caching hint: data is valid for 6 hours
    expiresAt := time.Now().Add(6 * time.Hour)
    for _, r := range results {
        r.ExpiresAt = timestamppb.New(expiresAt)
    }

    return &pbc.GetActualCostResponse{Results: results}, nil
}
```

### Setting expires_at on Projected Cost Responses

```go
func (p *MyPlugin) GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (
    *pbc.GetProjectedCostResponse, error) {

    resp := pluginsdk.NewGetProjectedCostResponse(
        pluginsdk.WithProjectedCostExpiresAt(endOfBillingCycle()),
    )
    resp.UnitPrice = 0.05
    resp.Currency = "USD"
    resp.CostPerMonth = 36.50

    return resp, nil
}
```

## For Callers (CLI / Host Applications)

### Checking Expiration

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// Check if a cost result has expired
for _, result := range resp.Results {
    if pluginsdk.IsActualCostExpired(result, time.Now()) {
        // Data is stale, re-fetch from plugin
        continue
    }

    // Data is fresh, use cached version
    processResult(result)
}

// Get TTL for cache management
if expiresAt, ok := pluginsdk.ActualCostExpiresAt(result); ok {
    ttl := time.Until(expiresAt)
    cache.SetWithTTL(cacheKey, result, ttl)
}
```

### Handling Missing expires_at

```go
// No expires_at set = always re-fetch (no caching assumption)
if expiresAt, ok := pluginsdk.ActualCostExpiresAt(result); !ok {
    // Plugin didn't set expires_at - fetch fresh data every time
    refreshData(result)
} else if expiresAt.Before(time.Now()) {
    // Plugin set expires_at but it's in the past - stale
    refreshData(result)
} else {
    // Data is still valid
    useCachedData(result)
}
```

## Testing

### Using Mock Plugin with Caching Hints

```go
mock := plugintesting.NewMockPlugin()
mock.ExpiresAtDuration = 6 * time.Hour // Results expire in 6 hours

harness := plugintesting.NewTestHarness(mock)
harness.Start(t)
defer harness.Stop()

// Verify expires_at is set on results
resp, err := harness.Client().GetActualCost(ctx, req)
require.NoError(t, err)

for _, result := range resp.Results {
    require.NotNil(t, result.ExpiresAt, "expires_at should be set")
    expiresAt := result.ExpiresAt.AsTime()
    require.True(t, expiresAt.After(time.Now()), "expires_at should be in the future")
}
```
