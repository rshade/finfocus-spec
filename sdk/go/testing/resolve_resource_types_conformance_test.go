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
