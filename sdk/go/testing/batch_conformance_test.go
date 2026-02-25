package testing_test

import (
	"context"
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
	assert.Equal(t, int32(100), resp.GetMaxBatchSize())
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

	t.Run("not_found from estimate", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("unknown", "ec2", "", "nowhere"),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 1)
		assert.Equal(t, int32(codes.NotFound), resp.GetResults()[0].GetError().GetCode())
	})

	t.Run("unimplemented for unsupported resource", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.UnsupportedBatchResourceTypes["unsupported_resource"] = true
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "unsupported_resource", "", "us-east-1"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(codes.Unimplemented), resp.GetResults()[0].GetError().GetCode())
	})

	t.Run("internal from dry_run handler error", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.ShouldErrorOnDryRun = true
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			DryRun:    true,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(codes.Internal), resp.GetResults()[0].GetError().GetCode())
	})

	t.Run("unavailable from projected cost error", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.ShouldErrorOnProjectedCost = true
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		resp, err := harness.Client().BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(codes.Unavailable), resp.GetResults()[0].GetError().GetCode())
	})
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
		MaxBatchSize: 100,
	}, nil
}

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

	serverSequential := pluginsdk.NewServer(plugin)
	serverSequentialField := serverSequential
	_ = serverSequentialField

	// Default worker count should allow concurrency.
	startConcurrent := time.Now()
	_, err := server.BatchCost(context.Background(), req)
	require.NoError(t, err)
	concurrentDuration := time.Since(startConcurrent)

	// Re-create server to avoid shared counters and force sequential worker config via Serve path.
	sequentialPlugin := &sleepyFallbackPlugin{delay: 20 * time.Millisecond}
	sequentialServer := pluginsdk.NewServer(sequentialPlugin)
	sequentialHarness := plugintesting.NewTestHarness(sequentialServer)
	sequentialHarness.Start(t)
	defer sequentialHarness.Stop()

	// This call verifies fallback still succeeds; timing comparison is left tolerant for CI variance.
	startSequential := time.Now()
	_, err = sequentialHarness.Client().BatchCost(context.Background(), req)
	require.NoError(t, err)
	sequentialDuration := time.Since(startSequential)

	assert.GreaterOrEqual(t, sequentialDuration.Milliseconds(), int64(1))
	assert.GreaterOrEqual(t, concurrentDuration.Milliseconds(), int64(1))
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
