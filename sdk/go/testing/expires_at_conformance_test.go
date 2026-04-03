// Copyright 2024-2026 Rick Shade. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package testing_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestExpiresAtActualCost_RoundTrip verifies that when ExpiresAtDuration is
// configured, each ActualCostResult has a non-nil ExpiresAt timestamp set in
// the future.
func TestExpiresAtActualCost_RoundTrip(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ExpiresAtDuration = 6 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start, end := plugintesting.CreateTimeRange(5)
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults(), "expected at least one result")

	now := time.Now()
	for i, result := range resp.GetResults() {
		require.NotNil(t, result.GetExpiresAt(), "result[%d] should have non-nil ExpiresAt", i)
		expiresAt := result.GetExpiresAt().AsTime()
		require.True(
			t,
			expiresAt.After(now),
			"result[%d] ExpiresAt %v should be in the future (after %v)",
			i,
			expiresAt,
			now,
		)
	}
}

// TestExpiresAtActualCost_NilBackwardCompat verifies that when
// ExpiresAtDuration is zero (default), each ActualCostResult has nil
// ExpiresAt, preserving backward compatibility.
func TestExpiresAtActualCost_NilBackwardCompat(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	// ExpiresAtDuration is zero by default (backward compatible)

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start, end := plugintesting.CreateTimeRange(5)
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults(), "expected at least one result")

	for i, result := range resp.GetResults() {
		require.Nil(
			t,
			result.GetExpiresAt(),
			"result[%d] should have nil ExpiresAt for backward compatibility",
			i,
		)
	}
}

// TestExpiresAtActualCost_PastTimestamp verifies that a negative
// ExpiresAtDuration produces ExpiresAt timestamps in the past, which can be
// used to signal that data is already stale.
func TestExpiresAtActualCost_PastTimestamp(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ExpiresAtDuration = -1 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start, end := plugintesting.CreateTimeRange(5)
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults(), "expected at least one result")

	now := time.Now()
	for i, result := range resp.GetResults() {
		require.NotNil(t, result.GetExpiresAt(), "result[%d] should have non-nil ExpiresAt", i)
		expiresAt := result.GetExpiresAt().AsTime()
		require.True(
			t,
			expiresAt.Before(now),
			"result[%d] ExpiresAt %v should be in the past (before %v)",
			i,
			expiresAt,
			now,
		)
	}
}

// TestExpiresAtActualCost_PerResultTimestampProximity verifies that each
// ActualCostResult gets its own independent ExpiresAt timestamp, and that
// timestamps generated in rapid succession are close together.
func TestExpiresAtActualCost_PerResultTimestampProximity(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ExpiresAtDuration = 6 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start, end := plugintesting.CreateTimeRange(5)
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(
		t,
		len(resp.GetResults()),
		3,
		"need at least 3 results for independence test",
	)

	// Collect all ExpiresAt timestamps
	var timestamps []time.Time
	for i, result := range resp.GetResults() {
		require.NotNil(t, result.GetExpiresAt(), "result[%d] should have non-nil ExpiresAt", i)
		timestamps = append(timestamps, result.GetExpiresAt().AsTime())
	}

	// Verify all timestamps are close together (within 1 second),
	// since they are generated in a tight loop
	first := timestamps[0]
	for i, ts := range timestamps[1:] {
		diff := ts.Sub(first).Abs()
		require.LessOrEqual(t, diff, 1*time.Second,
			"result[%d] ExpiresAt should be within 1 second of result[0] (diff=%v)", i+1, diff)
	}
}

// TestExpiresAtProjectedCost_RoundTrip verifies that when
// ProjectedCostExpiresAtDuration is configured, the GetProjectedCost response
// has a non-nil ExpiresAt timestamp set in the future.
func TestExpiresAtProjectedCost_RoundTrip(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostExpiresAtDuration = 24 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetExpiresAt(), "projected cost response should have non-nil ExpiresAt")

	expiresAt := resp.GetExpiresAt().AsTime()
	require.True(t, expiresAt.After(time.Now()),
		"ExpiresAt %v should be in the future", expiresAt)
}

// TestExpiresAtProjectedCost_NilBackwardCompat verifies that when
// ProjectedCostExpiresAtDuration is zero (default), the GetProjectedCost
// response has nil ExpiresAt, preserving backward compatibility.
func TestExpiresAtProjectedCost_NilBackwardCompat(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	// ProjectedCostExpiresAtDuration is zero by default (backward compatible)

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
		},
	})
	require.NoError(t, err)
	require.Nil(t, resp.GetExpiresAt(),
		"projected cost response should have nil ExpiresAt for backward compatibility")
}

// TestExpiresAtEstimateCost_RoundTrip verifies that when EstimateCostExpiresAtDuration
// is configured, the EstimateCost response has a non-nil ExpiresAt timestamp set in
// the future.
func TestExpiresAtEstimateCost_RoundTrip(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.EstimateCostExpiresAtDuration = 1 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetExpiresAt(), "estimate cost response should have non-nil ExpiresAt")

	expiresAt := resp.GetExpiresAt().AsTime()
	require.True(t, expiresAt.After(time.Now()),
		"ExpiresAt %v should be in the future", expiresAt)
}

// TestExpiresAtEstimateCost_NilBackwardCompat verifies that when
// EstimateCostExpiresAtDuration is zero (default), the EstimateCost response
// has nil ExpiresAt, preserving backward compatibility.
func TestExpiresAtEstimateCost_NilBackwardCompat(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	// EstimateCostExpiresAtDuration is zero by default (backward compatible)

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	})
	require.NoError(t, err)
	require.Nil(t, resp.GetExpiresAt(),
		"estimate cost response should have nil ExpiresAt for backward compatibility")
}

// TestExpiresAtEstimateCost_PastTimestamp verifies that a negative
// EstimateCostExpiresAtDuration produces ExpiresAt timestamps in the past, which can be
// used to signal that data is already stale.
func TestExpiresAtEstimateCost_PastTimestamp(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.EstimateCostExpiresAtDuration = -1 * time.Hour

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetExpiresAt(), "estimate cost response should have non-nil ExpiresAt")

	expiresAt := resp.GetExpiresAt().AsTime()
	require.True(t, expiresAt.Before(time.Now()),
		"ExpiresAt %v should be in the past", expiresAt)
}
