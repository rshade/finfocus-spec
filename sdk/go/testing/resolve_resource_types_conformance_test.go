// Copyright 2024-2026 Richard Shade
// Licensed under the Apache License, Version 2.0

package testing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// Note: this file validates the Server-level fallback path directly via TestHarness.
// It is orthogonal to (and does not duplicate) the ResolveResourceTypes_EmptyPlugin /
// ResolveResourceTypes_Basic tests registered into the Basic conformance tier in
// conformance_test.go, which validate the conformance-suite contract instead.

// resolveConformancePlugin implements pluginsdk.Plugin without ResolveResourceTypesProvider.
type resolveConformancePlugin struct{}

func (p *resolveConformancePlugin) Name() string { return "resolve-conformance" }

func (p *resolveConformancePlugin) GetProjectedCost(
	_ context.Context, _ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (p *resolveConformancePlugin) GetActualCost(
	_ context.Context, _ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (p *resolveConformancePlugin) GetPricingSpec(
	_ context.Context, _ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *resolveConformancePlugin) EstimateCost(
	_ context.Context, _ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

func TestResolveResourceTypesConformance_UnimplementedPlugin(t *testing.T) {
	plugin := &resolveConformancePlugin{}
	server := pluginsdk.NewServer(plugin)
	harness := plugintesting.NewTestHarness(server)
	harness.Start(t)
	defer harness.Stop()

	ctx := context.Background()
	client := harness.Client()

	t.Run("returns empty response for unimplemented plugin", func(t *testing.T) {
		resp, err := client.ResolveResourceTypes(ctx, &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "aws_s3_bucket"},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetMappings())
	})

	t.Run("returns empty response for empty source_types", func(t *testing.T) {
		resp, err := client.ResolveResourceTypes(ctx, &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetMappings())
	})

	t.Run("returns empty response for UNSPECIFIED format", func(t *testing.T) {
		resp, err := client.ResolveResourceTypes(ctx, &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
			SourceTypes:  []string{"aws_instance"},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetMappings())
	})
}

// TestResolveResourceTypesConformance_MockPluginImplemented validates the
// implemented-plugin happy path using the SDK's MockPlugin fixture, which
// implements pluginsdk.ResolveResourceTypesProvider with a seeded default mapping.
func TestResolveResourceTypesConformance_MockPluginImplemented(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	ctx := context.Background()
	client := harness.Client()

	t.Run("resolves known type and omits unknown type", func(t *testing.T) {
		resp, err := client.ResolveResourceTypes(ctx, &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "definitely_unknown_type_xyz"},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetMappings(), 1)
		assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
		assert.True(t, resp.GetMappings()["aws_instance"].GetSupported())
		_, exists := resp.GetMappings()["definitely_unknown_type_xyz"]
		assert.False(t, exists)
	})

	t.Run("returns configured error", func(t *testing.T) {
		errPlugin := plugintesting.NewMockPlugin()
		errPlugin.SetResolveResourceTypesConfig(plugintesting.ResolveResourceTypesConfig{
			ShouldError:  true,
			ErrorMessage: "type resolution unavailable",
		})
		errHarness := plugintesting.NewTestHarness(errPlugin)
		errHarness.Start(t)
		defer errHarness.Stop()

		_, err := errHarness.Client().ResolveResourceTypes(ctx, &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance"},
		})
		require.Error(t, err)
	})
}
