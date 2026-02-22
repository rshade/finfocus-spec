// Copyright 2024-2026 Rick Shade. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package pluginsdk

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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
	if result == nil {
		return false
	}
	ts := result.GetExpiresAt()
	if ts == nil {
		return false
	}
	return ts.AsTime().Before(now)
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
	if result == nil {
		return time.Time{}, false
	}
	ts := result.GetExpiresAt()
	if ts == nil {
		return time.Time{}, false
	}
	return ts.AsTime(), true
}

// IsProjectedCostExpired returns true if the projected cost response has an
// expires_at timestamp that is before the given reference time.
//
// Returns false if the response is nil or expires_at is nil/unset.
func IsProjectedCostExpired(resp *pbc.GetProjectedCostResponse, now time.Time) bool {
	if resp == nil {
		return false
	}
	ts := resp.GetExpiresAt()
	if ts == nil {
		return false
	}
	return ts.AsTime().Before(now)
}

// ProjectedCostExpiresAt returns the expiration time for a projected cost response.
// The second return value is false if the response is nil or expires_at is nil/unset.
func ProjectedCostExpiresAt(resp *pbc.GetProjectedCostResponse) (time.Time, bool) {
	if resp == nil {
		return time.Time{}, false
	}
	ts := resp.GetExpiresAt()
	if ts == nil {
		return time.Time{}, false
	}
	return ts.AsTime(), true
}

// ActualCostResultOption is a functional option for configuring ActualCostResult.
type ActualCostResultOption func(*pbc.ActualCostResult)

// WithActualCostResultExpiresAt returns an ActualCostResultOption that sets the
// expires_at caching hint on an individual cost result.
//
// A zero time.Time results in a nil expires_at (no caching guidance).
//
// Usage:
//
//	pluginsdk.ApplyActualCostResultOptions(result,
//	    pluginsdk.WithActualCostResultExpiresAt(time.Now().Add(6 * time.Hour)),
//	)
func WithActualCostResultExpiresAt(expiresAt time.Time) ActualCostResultOption {
	return func(result *pbc.ActualCostResult) {
		if expiresAt.IsZero() {
			result.ExpiresAt = nil
			return
		}
		result.ExpiresAt = timestamppb.New(expiresAt)
	}
}

// ApplyActualCostResultOptions applies functional options to an ActualCostResult.
//
// Usage:
//
//	result := &pbc.ActualCostResult{Cost: 10.0, Source: "aws-ce"}
//	pluginsdk.ApplyActualCostResultOptions(result,
//	    pluginsdk.WithActualCostResultExpiresAt(time.Now().Add(6 * time.Hour)),
//	)
func ApplyActualCostResultOptions(result *pbc.ActualCostResult, opts ...ActualCostResultOption) {
	if result == nil {
		return
	}
	for _, opt := range opts {
		opt(result)
	}
}
