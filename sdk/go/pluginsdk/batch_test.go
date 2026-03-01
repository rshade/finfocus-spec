//nolint:testpackage // Tests need access to unexported server tuning fields.
package pluginsdk

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestValidateBatchCostRequestActualRequiresTimeRange(t *testing.T) {
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	}

	_, err := ValidateBatchCostRequest(req, DefaultMaxBatchSize)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "start and end are required")
}

func TestValidateBatchCostRequestActualStartBeforeEnd(t *testing.T) {
	now := time.Now()
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     timestamppb.New(now),
		End:       timestamppb.New(now),
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	}

	_, err := ValidateBatchCostRequest(req, DefaultMaxBatchSize)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "start must be before end")
}

func TestValidateBatchCostRequestNonActualIgnoresTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		queryType pbc.CostQueryType
	}{
		{
			name:      "estimate ignores time range",
			queryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		},
		{
			name:      "projected ignores time range",
			queryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
		},
		{
			name:      "unspecified defaults and ignores time range",
			queryType: pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &pbc.BatchCostRequest{
				QueryType: tc.queryType,
				Resources: []*pbc.ResourceDescriptor{
					{Provider: "aws", ResourceType: "ec2"},
				},
			}
			emptyResp, err := ValidateBatchCostRequest(req, DefaultMaxBatchSize)
			require.NoError(t, err)
			assert.Nil(t, emptyResp)
		})
	}
}

func TestNewResourceError(t *testing.T) {
	err := NewResourceError(codes.NotFound, "resource missing", false)
	assert.Equal(t, int32(codes.NotFound), err.GetCode())
	assert.Equal(t, "resource missing", err.GetMessage())
	assert.False(t, err.GetResourceTypeUnsupported())
}

func TestResourceErrorUnsupported(t *testing.T) {
	err := resourceErrorUnsupported("ec2")
	assert.Equal(t, int32(codes.Unimplemented), err.GetCode())
	assert.Contains(t, err.GetMessage(), "ec2")
	assert.True(t, err.GetResourceTypeUnsupported())
}

func TestResourceErrorFromGRPCError(t *testing.T) {
	err := status.Error(codes.Unavailable, "temporary outage")
	resourceErr := resourceErrorFromGRPCError(err)
	assert.Equal(t, int32(codes.Unavailable), resourceErr.GetCode())
	assert.Equal(t, "temporary outage", resourceErr.GetMessage())
}

func BenchmarkIsValidCostQueryType_ZeroAllocs(b *testing.B) {
	b.ReportAllocs()
	queryType := pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL

	b.ResetTimer()
	for range b.N {
		if !IsValidCostQueryType(queryType) {
			b.Fatal("expected ACTUAL to be valid")
		}
	}
}

func BenchmarkValidateResourceDescriptor_ZeroAllocs(b *testing.B) {
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Id:           "i-1234567890abcdef0",
		Arn:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		Sku:          "t3.micro",
		Region:       "us-east-1",
		Tags: map[string]string{
			"env":     "production",
			"team":    "platform",
			"service": "api",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := ValidateResourceDescriptor(resource); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestServerBatchCostEnforcesDefaultMaxBatchSize(t *testing.T) {
	plugin := &batchServerTestPlugin{}
	server := NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, DefaultMaxBatchSize+1)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
		}
	}

	_, err := server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServerBatchCostCustomMaxBatchSize(t *testing.T) {
	plugin := &batchServerTestPlugin{}
	server := NewServer(plugin)
	server.maxBatchSize = 200

	buildResources := func(count int) []*pbc.ResourceDescriptor {
		resources := make([]*pbc.ResourceDescriptor, count)
		for i := range resources {
			resources[i] = &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
			}
		}
		return resources
	}

	resp, err := server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: buildResources(150),
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 150)
	assert.Equal(t, int32(200), resp.GetMaxBatchSize())

	resp, err = server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: buildResources(200),
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 200)

	_, err = server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: buildResources(201),
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServerBatchCostFallbackWorkerPool(t *testing.T) {
	makeRequest := func() *pbc.BatchCostRequest {
		return &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			Resources: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: "ec2"},
			},
		}
	}

	sequentialPlugin := &batchServerTestPlugin{
		estimateDelay: 20 * time.Millisecond,
	}
	sequentialServer := NewServer(sequentialPlugin)
	sequentialServer.batchWorkers = 1
	_, err := sequentialServer.BatchCost(context.Background(), makeRequest())
	require.NoError(t, err)
	assert.Equal(t, int64(1), sequentialPlugin.maxConcurrent.Load())

	concurrentPlugin := &batchServerTestPlugin{
		estimateDelay: 20 * time.Millisecond,
	}
	concurrentServer := NewServer(concurrentPlugin)
	concurrentServer.batchWorkers = 10
	_, err = concurrentServer.BatchCost(context.Background(), makeRequest())
	require.NoError(t, err)
	assert.Greater(t, concurrentPlugin.maxConcurrent.Load(), int64(1))
}

func TestServerBatchCostCustomHandlerPreferred(t *testing.T) {
	plugin := &customBatchServerPlugin{}
	server := NewServer(plugin)

	resp, err := server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	assert.Equal(t, int64(1), plugin.batchCalls.Load())
	assert.Equal(t, int64(0), plugin.estimateCalls.Load())
}

func TestServerBatchCostStructuredLogging(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := zerolog.New(&logBuffer).With().Timestamp().Logger()

	plugin := &batchServerTestPlugin{}
	server := NewServerWithOptions(plugin, nil, &logger, nil)

	_, err := server.BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2", Id: "resource-success"},
			{Provider: "aws", ResourceType: "unsupported-ec2", Id: "resource-error"},
		},
	})
	require.NoError(t, err)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "BatchCost request received")
	assert.Contains(t, logOutput, "\"resource_count\":2")
	assert.Contains(t, logOutput, "\"query_type\":\"COST_QUERY_TYPE_ESTIMATE\"")
	assert.Contains(t, logOutput, "\"dry_run\":false")
	assert.Contains(t, logOutput, "BatchCost resource failed")
	assert.Contains(t, logOutput, "\"error_code\":5")
	assert.Contains(t, logOutput, "\"resource_id\":\"resource-error\"")
	assert.Contains(t, logOutput, "BatchCost completed")
	assert.Contains(t, logOutput, "\"result_count\":2")
	assert.Contains(t, logOutput, "\"error_count\":1")
	assert.Contains(t, logOutput, "\"duration_ms\":")
}

func TestBatchCostFallbackDoesNotMutateRequest(t *testing.T) {
	plugin := &batchServerTestPlugin{}
	server := NewServer(plugin)

	originalQueryType := pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED
	req := &pbc.BatchCostRequest{
		QueryType: originalQueryType,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	}

	_, err := server.BatchCost(context.Background(), req)
	require.NoError(t, err)

	// The fallback path normalizes query_type into a local variable;
	// it must not write the normalized value back into the original request.
	assert.Equal(t, originalQueryType, req.GetQueryType(),
		"BatchCost fallback must not mutate the original request's QueryType")
}

type batchServerTestPlugin struct {
	estimateDelay  time.Duration
	currentRunning atomic.Int64
	maxConcurrent  atomic.Int64
}

func (p *batchServerTestPlugin) Name() string {
	return "batch-server-test"
}

func (p *batchServerTestPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{
		UnitPrice:    0.1,
		Currency:     "USD",
		CostPerMonth: 73,
	}, nil
}

func (p *batchServerTestPlugin) GetActualCost(
	_ context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if req.GetStart() == nil || req.GetEnd() == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end are required")
	}
	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{
			{Cost: 1, UsageAmount: 1, UsageUnit: "hour", Source: "test"},
		},
	}, nil
}

func (p *batchServerTestPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *batchServerTestPlugin) EstimateCost(
	_ context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	current := p.currentRunning.Add(1)
	updateMaxConcurrent(&p.maxConcurrent, current)
	defer p.currentRunning.Add(-1)

	if p.estimateDelay > 0 {
		time.Sleep(p.estimateDelay)
	}

	if strings.Contains(req.GetResourceType(), "unsupported") {
		return nil, status.Error(codes.NotFound, "resource type unsupported")
	}

	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: 10,
	}, nil
}

func updateMaxConcurrent(currentMax *atomic.Int64, value int64) {
	for {
		current := currentMax.Load()
		if value <= current {
			return
		}
		if currentMax.CompareAndSwap(current, value) {
			return
		}
	}
}

type customBatchServerPlugin struct {
	batchServerTestPlugin

	batchCalls    atomic.Int64
	estimateCalls atomic.Int64
}

func (p *customBatchServerPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	p.estimateCalls.Add(1)
	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: 1,
	}, nil
}

func (p *customBatchServerPlugin) BatchCost(
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
							CostMonthly: 88,
						},
					},
				},
			},
		}
	}
	return &pbc.BatchCostResponse{
		Results:      results,
		MaxBatchSize: DefaultMaxBatchSize,
	}, nil
}

// TestValidateResourceDescriptor tests field-level validation for ResourceDescriptor.
func TestValidateResourceDescriptor(t *testing.T) {
	tests := []struct {
		name        string
		resource    *pbc.ResourceDescriptor
		expectError bool
		errorText   string
	}{
		{
			name:        "nil resource is rejected",
			resource:    nil,
			expectError: true,
			errorText:   "resource descriptor must not be nil",
		},
		{
			name: "valid resource with all fields",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
				Id:           "test-id",
				Arn:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
				Tags:         map[string]string{"env": "prod"},
			},
			expectError: false,
		},
		{
			name: "provider exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     strings.Repeat("a", MaxProviderLength+1),
				ResourceType: "ec2",
			},
			expectError: true,
			errorText:   "provider length",
		},
		{
			name: "resource_type exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: strings.Repeat("a", MaxResourceTypeLength+1),
			},
			expectError: true,
			errorText:   "resource_type length",
		},
		{
			name: "id exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Id:           strings.Repeat("a", MaxIDLength+1),
			},
			expectError: true,
			errorText:   "id length",
		},
		{
			name: "arn exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Arn:          strings.Repeat("a", MaxARNLength+1),
			},
			expectError: true,
			errorText:   "arn length",
		},
		{
			name: "sku exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          strings.Repeat("a", MaxSKULength+1),
			},
			expectError: true,
			errorText:   "sku length",
		},
		{
			name: "region exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Region:       strings.Repeat("a", MaxRegionLength+1),
			},
			expectError: true,
			errorText:   "region length",
		},
		{
			name: "too many tags",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags:         generateTags(MaxTagsPerResource + 1),
			},
			expectError: true,
			errorText:   "tag count",
		},
		{
			name: "tag key exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags: map[string]string{
					strings.Repeat("k", MaxTagKeyLength+1): "value",
				},
			},
			expectError: true,
			errorText:   "tag key",
		},
		{
			name: "tag value exceeds max length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags: map[string]string{
					"mykey": strings.Repeat("v", MaxTagValueLength+1),
				},
			},
			expectError: true,
			errorText:   `tag value for key "mykey": length`,
		},
		{
			name: "max valid provider length",
			resource: &pbc.ResourceDescriptor{
				Provider:     strings.Repeat("a", MaxProviderLength),
				ResourceType: "ec2",
			},
			expectError: false,
		},
		{
			name: "max valid resource_type length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: strings.Repeat("a", MaxResourceTypeLength),
			},
			expectError: false,
		},
		{
			name: "max valid tags",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags:         generateTags(MaxTagsPerResource),
			},
			expectError: false,
		},
		{
			name: "max valid id length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Id:           strings.Repeat("a", MaxIDLength),
			},
			expectError: false,
		},
		{
			name: "max valid arn length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Arn:          strings.Repeat("a", MaxARNLength),
			},
			expectError: false,
		},
		{
			name: "max valid sku length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          strings.Repeat("a", MaxSKULength),
			},
			expectError: false,
		},
		{
			name: "max valid region length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Region:       strings.Repeat("a", MaxRegionLength),
			},
			expectError: false,
		},
		{
			name: "max valid tag key length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags: map[string]string{
					strings.Repeat("k", MaxTagKeyLength): "value",
				},
			},
			expectError: false,
		},
		{
			name: "max valid tag value length",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags: map[string]string{
					"mykey": strings.Repeat("v", MaxTagValueLength),
				},
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateResourceDescriptor(tc.resource)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorText)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateBatchCostRequestResourceDescriptorValidation tests that
// ValidateBatchCostRequest calls ValidateResourceDescriptor for each resource.
func TestValidateBatchCostRequestResourceDescriptorValidation(t *testing.T) {
	tests := []struct {
		name        string
		resources   []*pbc.ResourceDescriptor
		expectError bool
		errorText   string
	}{
		{
			name: "valid resources pass validation",
			resources: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "azure", ResourceType: "vm"},
			},
			expectError: false,
		},
		{
			name: "first resource invalid",
			resources: []*pbc.ResourceDescriptor{
				{Provider: strings.Repeat("a", MaxProviderLength+1), ResourceType: "ec2"},
				{Provider: "azure", ResourceType: "vm"},
			},
			expectError: true,
			errorText:   "resource at index 0",
		},
		{
			name: "second resource invalid",
			resources: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "azure", ResourceType: strings.Repeat("a", MaxResourceTypeLength+1)},
			},
			expectError: true,
			errorText:   "resource at index 1",
		},
		{
			name: "resource with too many tags",
			resources: []*pbc.ResourceDescriptor{
				{
					Provider:     "aws",
					ResourceType: "ec2",
					Tags:         generateTags(MaxTagsPerResource + 1),
				},
			},
			expectError: true,
			errorText:   "resource at index 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
				Resources: tc.resources,
			}
			_, err := ValidateBatchCostRequest(req, DefaultMaxBatchSize)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorText)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				// Verify clean error wrapping: the status message (desc) should not
				// contain a nested "rpc error: code =" from an inner gRPC status.
				s, ok := status.FromError(err)
				require.True(t, ok, "expected gRPC status error")
				assert.NotContains(t, s.Message(), "rpc error: code =",
					"nested gRPC error should be unwrapped before wrapping")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// generateTags creates a map with the specified number of tags.
func generateTags(count int) map[string]string {
	tags := make(map[string]string, count)
	for i := range count {
		tags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	return tags
}
