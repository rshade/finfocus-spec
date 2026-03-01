package pluginsdk_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// BenchmarkIsValidCostQueryType measures validation of CostQueryType values.
func BenchmarkIsValidCostQueryType(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.IsValidCostQueryType(pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE)
	}
}

// BenchmarkNormalizeCostQueryType measures normalization of CostQueryType values.
func BenchmarkNormalizeCostQueryType(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.NormalizeCostQueryType(pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED)
	}
}

// BenchmarkValidateBatchCostRequest_Empty measures validation of an empty batch.
func BenchmarkValidateBatchCostRequest_Empty(b *testing.B) {
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{},
	}

	// Validate once before benchmark loop to fail fast on regressions
	_, err := pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	}
}

// BenchmarkValidateBatchCostRequest_Valid measures validation of a valid non-empty batch.
func BenchmarkValidateBatchCostRequest_Valid(b *testing.B) {
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "azure", ResourceType: "vm"},
			{Provider: "gcp", ResourceType: "compute_engine"},
		},
	}

	// Validate once before benchmark loop to fail fast on regressions
	_, err := pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	}
}

// BenchmarkValidateBatchCostRequest_Actual measures validation of an ACTUAL query with timestamps.
func BenchmarkValidateBatchCostRequest_Actual(b *testing.B) {
	now := timestamppb.Now()
	later := timestamppb.New(now.AsTime().Add(24 * 60 * 60 * 1e9)) // +24h
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     now,
		End:       later,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	}

	// Validate once before benchmark loop to fail fast on regressions
	_, err := pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = pluginsdk.ValidateBatchCostRequest(req, pluginsdk.DefaultMaxBatchSize)
	}
}

// BenchmarkNewResourceError measures construction of ResourceError values.
func BenchmarkNewResourceError(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.NewResourceError(codes.NotFound, "resource not found", false)
	}
}

// BenchmarkNewBatchCostResponse measures construction of BatchCostResponse with options.
func BenchmarkNewBatchCostResponse(b *testing.B) {
	results := []*pbc.ResourceCostResult{
		{Resource: &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"}},
		{Resource: &pbc.ResourceDescriptor{Provider: "azure", ResourceType: "vm"}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = pluginsdk.NewBatchCostResponse(
			pluginsdk.WithBatchResults(results),
			pluginsdk.WithMaxBatchSize(pluginsdk.DefaultMaxBatchSize),
		)
	}
}

// BenchmarkBatchCostFallback_Small measures fallback processing for a small batch.
// req is shared across iterations and must remain read-only (fallback path skips proto.Clone).
func BenchmarkBatchCostFallback_Small(b *testing.B) {
	plugin := &benchmarkPlugin{}
	server := pluginsdk.NewServer(plugin)

	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "azure", ResourceType: "vm"},
			{Provider: "gcp", ResourceType: "compute_engine"},
		},
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostFallback_Large measures fallback processing for a larger batch.
// req is shared across iterations and must remain read-only (fallback path skips proto.Clone).
func BenchmarkBatchCostFallback_Large(b *testing.B) {
	plugin := &benchmarkPlugin{}
	server := pluginsdk.NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, 50)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"}
	}
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostFallback_100Resources measures fallback processing for 100 resources.
// req is shared across iterations and must remain read-only (fallback path skips proto.Clone).
func BenchmarkBatchCostFallback_100Resources(b *testing.B) {
	plugin := &benchmarkPlugin{}
	server := pluginsdk.NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, 100)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"}
	}
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostFallback_50ResourcesWithTags measures fallback with tags.
// req is shared across iterations and must remain read-only (fallback path skips proto.Clone).
func BenchmarkBatchCostFallback_50ResourcesWithTags(b *testing.B) {
	plugin := &benchmarkPlugin{}
	server := pluginsdk.NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, 50)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Tags: map[string]string{
				"env":     "production",
				"team":    "platform",
				"service": "api",
			},
		}
	}
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostFallback_100ResourcesWithTags measures fallback with tags.
// req is shared across iterations and must remain read-only (fallback path skips proto.Clone).
func BenchmarkBatchCostFallback_100ResourcesWithTags(b *testing.B) {
	plugin := &benchmarkPlugin{}
	server := pluginsdk.NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, 100)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Tags: map[string]string{
				"env":     "production",
				"team":    "platform",
				"service": "api",
			},
		}
	}
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostCustomHandler_Small measures the custom handler path (with proto.Clone).
// Compare with BenchmarkBatchCostFallback_Small to see the clone/no-clone delta.
func BenchmarkBatchCostCustomHandler_Small(b *testing.B) {
	plugin := &benchmarkCustomPlugin{}
	server := pluginsdk.NewServer(plugin)

	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "azure", ResourceType: "vm"},
			{Provider: "gcp", ResourceType: "compute_engine"},
		},
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// BenchmarkBatchCostCustomHandler_Large measures the custom handler path (with proto.Clone) for 50 resources.
// Compare with BenchmarkBatchCostFallback_Large to see the clone/no-clone delta.
func BenchmarkBatchCostCustomHandler_Large(b *testing.B) {
	plugin := &benchmarkCustomPlugin{}
	server := pluginsdk.NewServer(plugin)

	resources := make([]*pbc.ResourceDescriptor, 50)
	for i := range resources {
		resources[i] = &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"}
	}
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: resources,
	}

	ctx := context.Background()

	// Validate once before benchmark loop to fail fast on regressions.
	resp, err := server.BatchCost(ctx, req)
	if err != nil {
		b.Fatalf("pre-check failed: %v", err)
	}
	if resp == nil {
		b.Fatal("pre-check returned nil response")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = server.BatchCost(ctx, req)
	}
}

// benchmarkPlugin is a minimal plugin for benchmark tests (no BatchCostHandler → triggers fallback).
type benchmarkPlugin struct{}

func (p *benchmarkPlugin) Name() string { return "benchmark-plugin" }

func (p *benchmarkPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{UnitPrice: 0.1, Currency: "USD", CostPerMonth: 73}, nil
}

func (p *benchmarkPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{{Cost: 1, UsageAmount: 1, UsageUnit: "hour"}},
	}, nil
}

func (p *benchmarkPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *benchmarkPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 42}, nil
}

// benchmarkCustomPlugin implements BatchCostHandler to exercise the custom handler path
// (which calls proto.Clone). Embedding benchmarkPlugin provides the base Plugin interface.
type benchmarkCustomPlugin struct {
	benchmarkPlugin
}

func (p *benchmarkCustomPlugin) BatchCost(
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
							CostMonthly: 42,
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
