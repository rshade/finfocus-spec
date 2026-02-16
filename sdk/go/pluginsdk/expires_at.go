// Copyright 2024-2026 Rick Shade. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package pluginsdk

import (
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// IsActualCostExpired returns true if the cost result has an expires_at
// timestamp that is before the given reference time.
//
// Returns false if the result is nil or expires_at is nil/unset (no expiration
// guidance means the caller cannot determine expiration status from this field alone).
//
// Usage:
//
//	if pluginsdk.IsActualCostExpired(result, time.Now()) {
//	    // Re-fetch from plugin
//	}
func IsActualCostExpired(result *pbc.ActualCostResult, now time.Time) bool {
	if result == nil || result.GetExpiresAt() == nil {
		return false
	}
	return result.GetExpiresAt().AsTime().Before(now)
}

// ActualCostExpiresAt returns the expiration time for a cost result.
// The second return value is false if the result is nil or expires_at is nil/unset.
//
// Usage:
//
//	if expiresAt, ok := pluginsdk.ActualCostExpiresAt(result); ok {
//	    ttl := time.Until(expiresAt)
//	    cache.SetWithTTL(key, result, ttl)
//	}
func ActualCostExpiresAt(result *pbc.ActualCostResult) (time.Time, bool) {
	if result == nil || result.GetExpiresAt() == nil {
		return time.Time{}, false
	}
	return result.GetExpiresAt().AsTime(), true
}

// IsProjectedCostExpired returns true if the projected cost response has an
// expires_at timestamp that is before the given reference time.
//
// Returns false if the response is nil or expires_at is nil/unset.
func IsProjectedCostExpired(resp *pbc.GetProjectedCostResponse, now time.Time) bool {
	if resp == nil || resp.GetExpiresAt() == nil {
		return false
	}
	return resp.GetExpiresAt().AsTime().Before(now)
}

// ProjectedCostExpiresAt returns the expiration time for a projected cost response.
// The second return value is false if the response is nil or expires_at is nil/unset.
func ProjectedCostExpiresAt(resp *pbc.GetProjectedCostResponse) (time.Time, bool) {
	if resp == nil || resp.GetExpiresAt() == nil {
		return time.Time{}, false
	}
	return resp.GetExpiresAt().AsTime(), true
}
