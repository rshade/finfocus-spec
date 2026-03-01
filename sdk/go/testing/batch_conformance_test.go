package testing_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestBatchCostConformanceBasicQueryTypes(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	resources := []*pbc.ResourceDescriptor{
		plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
		plugintesting.CreateResourceDescriptor("azure", "vm", "Standard_B1s", "eastus"),
		plugintesting.CreateResourceDescriptor("gcp", "compute_engine", "e2-micro", "us-central1"),
	}

	t.Run("estimate query returns estimate data", func(t *testing.T) {
		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			Resources: resources,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(resources))
		for i, result := range resp.GetResults() {
			assert.Equal(t, resources[i].GetResourceType(), result.GetResource().GetResourceType())
			require.NotNil(t, result.GetCostData())
			assert.NotNil(t, result.GetCostData().GetEstimate())
		}
	})

	t.Run("actual query returns actual cost data", func(t *testing.T) {
		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
			Start:     start,
			End:       end,
			Resources: resources,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(resources))
		for _, result := range resp.GetResults() {
			actual := result.GetCostData().GetActualCost()
			require.NotNil(t, actual)
			assert.NotEmpty(t, actual.GetResults())
		}
	})

	t.Run("projected query returns projected cost data", func(t *testing.T) {
		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
			Resources: resources,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(resources))
		for _, result := range resp.GetResults() {
			assert.NotNil(t, result.GetCostData().GetProjectedCost())
		}
	})

	t.Run("unspecified query defaults to estimate", func(t *testing.T) {
		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED,
			Resources: resources,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(resources))
		for _, result := range resp.GetResults() {
			assert.NotNil(t, result.GetCostData().GetEstimate())
		}
	})
}

func TestBatchCostConformanceEmptyBatch(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Equal(t, int32(pluginsdk.DefaultMaxBatchSize), resp.GetMaxBatchSize())
}

func TestBatchCostConformancePartialFailures(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.UnsupportedBatchResourceTypes["unsupported_a"] = true
	plugin.UnsupportedBatchResourceTypes["unsupported_b"] = true

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resources := []*pbc.ResourceDescriptor{
		plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
		plugintesting.CreateResourceDescriptor("aws", "unsupported_a", "", "us-east-1"),
		plugintesting.CreateResourceDescriptor("azure", "vm", "Standard_B1s", "eastus"),
		plugintesting.CreateResourceDescriptor("gcp", "unsupported_b", "", "us-central1"),
		plugintesting.CreateResourceDescriptor("gcp", "compute_engine", "e2-micro", "us-central1"),
	}

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), len(resources))

	successCount := 0
	errorCount := 0
	for _, result := range resp.GetResults() {
		if result.GetError() != nil {
			errorCount++
			assert.True(t, result.GetError().GetResourceTypeUnsupported())
			assert.Equal(t, int32(codes.Unimplemented), result.GetError().GetCode())
		} else {
			successCount++
			assert.NotNil(t, result.GetCostData().GetEstimate())
		}
	}

	assert.Equal(t, 3, successCount)
	assert.Equal(t, 2, errorCount)
}

func TestBatchCostConformanceAllFailuresStillReturnResponse(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.UnsupportedBatchResourceTypes["unsupported_a"] = true
	plugin.UnsupportedBatchResourceTypes["unsupported_b"] = true

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "unsupported_a", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "unsupported_b", "", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 2)
	for _, result := range resp.GetResults() {
		require.NotNil(t, result.GetError())
		assert.Equal(t, int32(codes.Unimplemented), result.GetError().GetCode())
	}
}

func TestBatchCostConformanceErrorCodeMapping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(*plugintesting.MockPlugin)
		request  *pbc.BatchCostRequest
		expected codes.Code
	}{
		{
			name:  "not_found from estimate",
			setup: func(_ *plugintesting.MockPlugin) {},
			request: &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
				Resources: []*pbc.ResourceDescriptor{
					plugintesting.CreateResourceDescriptor("unknown", "ec2", "", "nowhere"),
				},
			},
			expected: codes.NotFound,
		},
		{
			name: "unimplemented for unsupported resource",
			setup: func(p *plugintesting.MockPlugin) {
				p.UnsupportedBatchResourceTypes["unsupported_resource"] = true
			},
			request: &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
				Resources: []*pbc.ResourceDescriptor{
					plugintesting.CreateResourceDescriptor("aws", "unsupported_resource", "", "us-east-1"),
				},
			},
			expected: codes.Unimplemented,
		},
		{
			name: "internal from dry_run handler error",
			setup: func(p *plugintesting.MockPlugin) {
				p.ShouldErrorOnDryRun = true
			},
			request: &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
				DryRun:    true,
				Resources: []*pbc.ResourceDescriptor{
					plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
				},
			},
			expected: codes.Internal,
		},
		{
			name: "unavailable from projected cost error",
			setup: func(p *plugintesting.MockPlugin) {
				p.ShouldErrorOnProjectedCost = true
			},
			request: &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
				Resources: []*pbc.ResourceDescriptor{
					plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
				},
			},
			expected: codes.Unavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := plugintesting.NewMockPlugin()
			tt.setup(plugin)
			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(t)
			defer harness.Stop()

			resp, err := harness.Client().BatchCost(ctx, tt.request)
			require.NoError(t, err)
			require.Len(t, resp.GetResults(), 1)
			assert.Equal(t, int32(tt.expected), resp.GetResults()[0].GetError().GetCode())
		})
	}
}

func TestBatchCostFallbackWhenPluginDoesNotImplementBatchHandler(t *testing.T) {
	plugin := &fallbackBatchPlugin{}
	server := pluginsdk.NewServer(plugin)
	harness := plugintesting.NewTestHarness(server)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "rds", "", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 2)
	assert.Positive(t, plugin.estimateCalls.Load())
	assert.Equal(t, int32(100), resp.GetMaxBatchSize())
}

func TestBatchCostUsesCustomHandlerWhenImplemented(t *testing.T) {
	plugin := &customBatchPlugin{}
	server := pluginsdk.NewServer(plugin)
	harness := plugintesting.NewTestHarness(server)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	assert.Equal(t, int64(1), plugin.batchCalls.Load())
	assert.Equal(t, int64(0), plugin.estimateCalls.Load(), "fallback should not run when custom handler is available")
}

func TestBatchCostCapabilityDiscovery(t *testing.T) {
	info := pluginsdk.NewPluginInfo("batch-plugin", "v1.0.0")

	withBatch := &customBatchPlugin{}
	withBatchServer := pluginsdk.NewServerWithOptions(withBatch, nil, nil, info)
	withBatchHarness := plugintesting.NewTestHarness(withBatchServer)
	withBatchHarness.Start(t)
	defer withBatchHarness.Stop()

	withResp, err := withBatchHarness.Client().GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Contains(t, withResp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
	assert.Equal(t, "true", withResp.GetMetadata()["supports_batch_cost"])
	assert.Equal(t, "100", withResp.GetMetadata()["max_batch_size"])

	withoutBatch := &fallbackBatchPlugin{}
	withoutBatchServer := pluginsdk.NewServerWithOptions(withoutBatch, nil, nil, info)
	withoutBatchHarness := plugintesting.NewTestHarness(withoutBatchServer)
	withoutBatchHarness.Start(t)
	defer withoutBatchHarness.Stop()

	withoutResp, err := withoutBatchHarness.Client().GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.NotContains(t, withoutResp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
	assert.NotContains(t, withoutResp.GetMetadata(), "max_batch_size")
}

func TestBatchCostCapabilityDiscoveryWithPluginInfoProvider(t *testing.T) {
	plugin := &pluginInfoProviderWithBatch{}
	server := pluginsdk.NewServer(plugin)
	harness := plugintesting.NewTestHarness(server)
	harness.Start(t)
	defer harness.Stop()

	ctx := context.Background()

	// Verify GetPluginInfo response includes PLUGIN_CAPABILITY_BATCH_COST
	infoResp, err := harness.Client().GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Contains(t, infoResp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)

	// Verify max_batch_size metadata is present and correct (this is what issue #2 was about)
	assert.Equal(t, "100", infoResp.GetMetadata()["max_batch_size"])

	// Verify custom metadata from plugin is preserved
	assert.Equal(t, "custom_value", infoResp.GetMetadata()["custom_key"])

	// Verify BatchCost RPC works correctly
	batchResp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, batchResp.GetResults(), 1)
	assert.NotNil(t, batchResp.GetResults()[0].GetCostData().GetEstimate())

	// Verify metadata consistency: max_batch_size from GetPluginInfo should match BatchCost response
	assert.Equal(t, infoResp.GetMetadata()["max_batch_size"], strconv.Itoa(int(batchResp.GetMaxBatchSize())))
}

func TestBatchCostDryRunConformance(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.UnsupportedBatchResourceTypes["unsupported_resource"] = true
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		DryRun:    true,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "unsupported_resource", "", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 2)

	firstData := resp.GetResults()[0].GetCostData()
	require.NotNil(t, firstData)
	assert.NotNil(t, firstData.GetDryRunResult())
	assert.Nil(t, firstData.GetEstimate())
	assert.Nil(t, firstData.GetProjectedCost())
	assert.Nil(t, firstData.GetActualCost())

	secondErr := resp.GetResults()[1].GetError()
	require.NotNil(t, secondErr)
	assert.True(t, secondErr.GetResourceTypeUnsupported())
}

func TestBatchCostConformanceActualCostPaginationPreserved(t *testing.T) {
	plugin := &paginatedBatchFallbackPlugin{}
	server := pluginsdk.NewServer(plugin)
	harness := plugintesting.NewTestHarness(server)
	harness.Start(t)
	defer harness.Stop()

	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "i-batch-page-1", "us-east-1")
	resource.Id = "i-batch-page-1"

	batchResp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     start,
		End:       end,
		Resources: []*pbc.ResourceDescriptor{resource},
	})
	require.NoError(t, err)
	require.Len(t, batchResp.GetResults(), 1)

	actual := batchResp.GetResults()[0].GetCostData().GetActualCost()
	require.NotNil(t, actual)
	require.Len(t, actual.GetResults(), 1)
	assert.Equal(t, "actual-page-2", actual.GetNextPageToken())
	assert.Equal(t, int32(2), actual.GetTotalCount())

	followUpResp, err := harness.Client().GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: resource.GetId(),
		Start:      start,
		End:        end,
		PageSize:   1,
		PageToken:  actual.GetNextPageToken(),
	})
	require.NoError(t, err)
	require.Len(t, followUpResp.GetResults(), 1)
	assert.Empty(t, followUpResp.GetNextPageToken())
	assert.Equal(t, int32(2), followUpResp.GetTotalCount())
	assert.Equal(t, "actual-page-2", followUpResp.GetResults()[0].GetSource())
}

type fallbackBatchPlugin struct {
	estimateCalls  atomic.Int64
	actualCalls    atomic.Int64
	projectedCalls atomic.Int64
}

func (p *fallbackBatchPlugin) Name() string {
	return "fallback-batch-plugin"
}

func (p *fallbackBatchPlugin) GetProjectedCost(
	_ context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	p.projectedCalls.Add(1)
	if req.GetResource() == nil {
		return nil, status.Error(codes.InvalidArgument, "resource is required")
	}
	return &pbc.GetProjectedCostResponse{
		UnitPrice:    0.1,
		Currency:     "USD",
		CostPerMonth: 73,
	}, nil
}

func (p *fallbackBatchPlugin) GetActualCost(
	_ context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	p.actualCalls.Add(1)
	if req.GetStart() == nil || req.GetEnd() == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end are required")
	}
	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{
			{Cost: 1, UsageAmount: 1, UsageUnit: "hour", Source: "fallback"},
		},
	}, nil
}

func (p *fallbackBatchPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *fallbackBatchPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	p.estimateCalls.Add(1)
	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: 42,
	}, nil
}

type customBatchPlugin struct {
	fallbackBatchPlugin

	batchCalls atomic.Int64
}

func (p *customBatchPlugin) BatchCost(
	_ context.Context,
	req *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	p.batchCalls.Add(1)

	results := make([]*pbc.ResourceCostResult, len(req.GetResources()))
	for i, resource := range req.GetResources() {
		results[i] = &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_CostData{
				CostData: &pbc.CostData{
					Data: &pbc.CostData_Estimate{
						Estimate: &pbc.EstimateCostResponse{
							Currency:    "USD",
							CostMonthly: 99,
						},
					},
				},
			},
		}
	}

	return &pbc.BatchCostResponse{
		Results:      results,
		MaxBatchSize: pluginsdk.DefaultMaxBatchSize,
	}, nil
}

// TestBatchCostFallbackWorkerPoolConfig verifies that the batch fallback path
// succeeds via both the direct Server.BatchCost call and the gRPC transport path.
//
// NOTE: Both servers use NewServer() with the default worker count (10). Without a
// public WithBatchWorkers option there is no way to configure a single-worker server
// for a meaningful sequential-vs-concurrent timing comparison. The assertions here
// are smoke tests that confirm both paths complete without error.
// TODO: Add a meaningful concurrency test once a public WithBatchWorkers option is available.
func TestBatchCostFallbackWorkerPoolConfig(t *testing.T) {
	plugin := &sleepyFallbackPlugin{delay: 20 * time.Millisecond}
	server := pluginsdk.NewServer(plugin)

	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
			plugintesting.CreateResourceDescriptor("aws", "ec2", "", "us-east-1"),
		},
	}

	// Verify direct Server.BatchCost path succeeds with default workers.
	resp, err := server.BatchCost(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), len(req.GetResources()))

	// Verify gRPC transport path also succeeds.
	grpcPlugin := &sleepyFallbackPlugin{delay: 20 * time.Millisecond}
	grpcServer := pluginsdk.NewServer(grpcPlugin)
	grpcHarness := plugintesting.NewTestHarness(grpcServer)
	grpcHarness.Start(t)
	defer grpcHarness.Stop()

	grpcResp, err := grpcHarness.Client().BatchCost(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, grpcResp.GetResults(), len(req.GetResources()))
}

type sleepyFallbackPlugin struct {
	fallbackBatchPlugin

	delay time.Duration
}

func (p *sleepyFallbackPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	time.Sleep(p.delay)
	p.estimateCalls.Add(1)
	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: 1,
	}, nil
}

type paginatedBatchFallbackPlugin struct {
	fallbackBatchPlugin
}

func (p *paginatedBatchFallbackPlugin) GetActualCost(
	_ context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if req.GetStart() == nil || req.GetEnd() == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end are required")
	}

	switch req.GetPageToken() {
	case "":
		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{
				{Cost: 1, UsageAmount: 1, UsageUnit: "hour", Source: "actual-page-1"},
			},
			NextPageToken: "actual-page-2",
			TotalCount:    2,
		}, nil
	case "actual-page-2":
		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{
				{Cost: 2, UsageAmount: 1, UsageUnit: "hour", Source: "actual-page-2"},
			},
			NextPageToken: "",
			TotalCount:    2,
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}
}

type pluginInfoProviderWithBatch struct {
	fallbackBatchPlugin
}

func (p *pluginInfoProviderWithBatch) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        "plugin-info-provider-with-batch",
		Version:     "v1.0.0",
		SpecVersion: pluginsdk.SpecVersion,
		Providers:   []string{"aws"},
		Capabilities: []pbc.PluginCapability{
			pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
		},
		Metadata: map[string]string{
			"custom_key": "custom_value",
		},
	}, nil
}

func (p *pluginInfoProviderWithBatch) BatchCost(
	_ context.Context,
	req *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	results := make([]*pbc.ResourceCostResult, len(req.GetResources()))
	for i, resource := range req.GetResources() {
		results[i] = &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_CostData{
				CostData: &pbc.CostData{
					Data: &pbc.CostData_Estimate{
						Estimate: &pbc.EstimateCostResponse{
							Currency:    "USD",
							CostMonthly: 75,
						},
					},
				},
			},
		}
	}

	return &pbc.BatchCostResponse{
		Results:      results,
		MaxBatchSize: pluginsdk.DefaultMaxBatchSize,
	}, nil
}
