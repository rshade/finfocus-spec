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
	futureTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluginsdk.IsProjectedCostExpired(tt.resp, tt.now)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestProjectedCostExpiresAt(t *testing.T) {
	futureTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

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
