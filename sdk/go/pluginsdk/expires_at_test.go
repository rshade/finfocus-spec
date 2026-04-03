// Copyright 2024-2026 Rick Shade. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package pluginsdk_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestIsActualCostExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		result   *pbc.ActualCostResult
		now      time.Time
		expected bool
	}{
		{
			name:     "nil result",
			result:   nil,
			now:      now,
			expected: false,
		},
		{
			name:     "nil expires_at",
			result:   &pbc.ActualCostResult{},
			now:      now,
			expected: false,
		},
		{
			name: "future timestamp",
			result: &pbc.ActualCostResult{
				ExpiresAt: timestamppb.New(now.Add(time.Hour)),
			},
			now:      now,
			expected: false,
		},
		{
			name: "past timestamp",
			result: &pbc.ActualCostResult{
				ExpiresAt: timestamppb.New(now.Add(-time.Hour)),
			},
			now:      now,
			expected: true,
		},
		{
			name: "exact boundary",
			result: &pbc.ActualCostResult{
				ExpiresAt: timestamppb.New(now),
			},
			now:      now,
			expected: false, // Before is strict: now.Before(now) == false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluginsdk.IsActualCostExpired(tt.result, tt.now)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestActualCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		result     *pbc.ActualCostResult
		expectedOk bool
	}{
		{
			name:       "nil result",
			result:     nil,
			expectedOk: false,
		},
		{
			name:       "nil expires_at",
			result:     &pbc.ActualCostResult{},
			expectedOk: false,
		},
		{
			name: "has expires_at",
			result: &pbc.ActualCostResult{
				ExpiresAt: timestamppb.New(futureTime),
			},
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pluginsdk.ActualCostExpiresAt(tt.result)
			require.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				require.Equal(t, futureTime, got)
			}
		})
	}
}

func TestIsProjectedCostExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		resp     *pbc.GetProjectedCostResponse
		now      time.Time
		expected bool
	}{
		{
			name:     "nil response",
			resp:     nil,
			now:      now,
			expected: false,
		},
		{
			name:     "nil expires_at",
			resp:     &pbc.GetProjectedCostResponse{},
			now:      now,
			expected: false,
		},
		{
			name: "future timestamp",
			resp: &pbc.GetProjectedCostResponse{
				ExpiresAt: timestamppb.New(now.Add(time.Hour)),
			},
			now:      now,
			expected: false,
		},
		{
			name: "past timestamp",
			resp: &pbc.GetProjectedCostResponse{
				ExpiresAt: timestamppb.New(now.Add(-time.Hour)),
			},
			now:      now,
			expected: true,
		},
		{
			name: "exact boundary",
			resp: &pbc.GetProjectedCostResponse{
				ExpiresAt: timestamppb.New(now),
			},
			now:      now,
			expected: false, // Before is strict: now.Before(now) == false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluginsdk.IsProjectedCostExpired(tt.resp, tt.now)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestProjectedCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		resp       *pbc.GetProjectedCostResponse
		expectedOk bool
	}{
		{
			name:       "nil response",
			resp:       nil,
			expectedOk: false,
		},
		{
			name:       "nil expires_at",
			resp:       &pbc.GetProjectedCostResponse{},
			expectedOk: false,
		},
		{
			name: "has expires_at",
			resp: &pbc.GetProjectedCostResponse{
				ExpiresAt: timestamppb.New(futureTime),
			},
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pluginsdk.ProjectedCostExpiresAt(tt.resp)
			require.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				require.Equal(t, futureTime, got)
			}
		})
	}
}

func TestWithActualCostResultExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		result    *pbc.ActualCostResult
		expiresAt time.Time
		expectNil bool
	}{
		{
			name:      "nil result is silent no-op",
			result:    nil,
			expiresAt: futureTime,
			expectNil: true, // nil result cannot be inspected
		},
		{
			name:      "zero time sets nil",
			result:    &pbc.ActualCostResult{Cost: 10.0, Source: "test"},
			expiresAt: time.Time{},
			expectNil: true,
		},
		{
			name:      "non-zero time sets timestamp",
			result:    &pbc.ActualCostResult{Cost: 10.0, Source: "test"},
			expiresAt: futureTime,
			expectNil: false,
		},
		{
			name: "overwrite existing timestamp",
			result: &pbc.ActualCostResult{
				Cost:      10.0,
				Source:    "test",
				ExpiresAt: timestamppb.New(time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			expiresAt: futureTime,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				pluginsdk.ApplyActualCostResultOptions(tt.result,
					pluginsdk.WithActualCostResultExpiresAt(tt.expiresAt),
				)
			})
			if tt.result == nil {
				return // cannot inspect nil result
			}
			if tt.expectNil {
				require.Nil(t, tt.result.GetExpiresAt())
			} else {
				require.NotNil(t, tt.result.GetExpiresAt())
				actualTime := tt.result.GetExpiresAt().AsTime()
				require.True(t, actualTime.Equal(tt.expiresAt),
					"expected %v, got %v", tt.expiresAt, actualTime)
			}
		})
	}
}

// TestWithProjectedCostExpiresAt verifies that the WithProjectedCostExpiresAt
// option function correctly sets and clears ExpiresAt on GetProjectedCostResponse.
func TestWithProjectedCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("sets non-zero time", func(t *testing.T) {
		resp := pluginsdk.NewGetProjectedCostResponse(
			pluginsdk.WithProjectedCostExpiresAt(futureTime),
		)
		require.NotNil(t, resp.GetExpiresAt(), "ExpiresAt should be set for non-zero time")
		actualTime := resp.GetExpiresAt().AsTime()
		diff := actualTime.Sub(futureTime).Abs()
		require.LessOrEqual(t, diff, time.Millisecond,
			"ExpiresAt should match input time (diff=%v)", diff)
	})

	t.Run("zero time sets nil", func(t *testing.T) {
		resp := pluginsdk.NewGetProjectedCostResponse(
			pluginsdk.WithProjectedCostExpiresAt(time.Time{}),
		)
		require.Nil(t, resp.GetExpiresAt(),
			"ExpiresAt should be nil for zero time.Time")
	})
}

func TestIsEstimateCostExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		resp     *pbc.EstimateCostResponse
		now      time.Time
		expected bool
	}{
		{
			name:     "nil response",
			resp:     nil,
			now:      now,
			expected: false,
		},
		{
			name:     "nil expires_at",
			resp:     &pbc.EstimateCostResponse{},
			now:      now,
			expected: false,
		},
		{
			name: "future timestamp",
			resp: &pbc.EstimateCostResponse{
				ExpiresAt: timestamppb.New(now.Add(time.Hour)),
			},
			now:      now,
			expected: false,
		},
		{
			name: "past timestamp",
			resp: &pbc.EstimateCostResponse{
				ExpiresAt: timestamppb.New(now.Add(-time.Hour)),
			},
			now:      now,
			expected: true,
		},
		{
			name: "exact boundary",
			resp: &pbc.EstimateCostResponse{
				ExpiresAt: timestamppb.New(now),
			},
			now:      now,
			expected: false, // Before is strict: now.Before(now) == false
		},
		{
			name: "unix epoch",
			resp: &pbc.EstimateCostResponse{
				ExpiresAt: timestamppb.New(time.Unix(0, 0)),
			},
			now:      now,
			expected: true, // Unix epoch is always in the past
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluginsdk.IsEstimateCostExpired(tt.resp, tt.now)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestEstimateCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		resp       *pbc.EstimateCostResponse
		expectedOk bool
	}{
		{
			name:       "nil response",
			resp:       nil,
			expectedOk: false,
		},
		{
			name:       "nil expires_at",
			resp:       &pbc.EstimateCostResponse{},
			expectedOk: false,
		},
		{
			name: "has expires_at",
			resp: &pbc.EstimateCostResponse{
				ExpiresAt: timestamppb.New(futureTime),
			},
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pluginsdk.EstimateCostExpiresAt(tt.resp)
			require.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				require.Equal(t, futureTime, got)
			}
		})
	}
}

func TestWithEstimateCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("sets non-zero time", func(t *testing.T) {
		resp := pluginsdk.NewEstimateCostResponse(
			pluginsdk.WithEstimateCostExpiresAt(futureTime),
		)
		require.NotNil(t, resp.GetExpiresAt(), "ExpiresAt should be set for non-zero time")
		actualTime := resp.GetExpiresAt().AsTime()
		diff := actualTime.Sub(futureTime).Abs()
		require.LessOrEqual(t, diff, time.Millisecond,
			"ExpiresAt should match input time (diff=%v)", diff)
	})

	t.Run("zero time sets nil", func(t *testing.T) {
		resp := pluginsdk.NewEstimateCostResponse(
			pluginsdk.WithEstimateCostExpiresAt(time.Time{}),
		)
		require.Nil(t, resp.GetExpiresAt(),
			"ExpiresAt should be nil for zero time.Time")
	})
}
