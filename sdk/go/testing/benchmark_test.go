package testing_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// BenchmarkName benchmarks the Name RPC method.
func BenchmarkName(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{}) // Dummy testing.T for harness
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, err := client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			b.Fatalf("Name() failed: %v", err)
		}
	}
}

// BenchmarkSupports benchmarks the Supports RPC method.
func BenchmarkSupports(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ResetTimer()
	for range b.N {
		_, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		if err != nil {
			b.Fatalf("Supports() failed: %v", err)
		}
	}
}

// BenchmarkGetActualCost benchmarks the GetActualCost RPC method.
func BenchmarkGetActualCost(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)

	b.ResetTimer()
	for range b.N {
		_, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
		})
		if err != nil {
			b.Fatalf("GetActualCost() failed: %v", err)
		}
	}
}

// BenchmarkGetProjectedCost benchmarks the GetProjectedCost RPC method.
func BenchmarkGetProjectedCost(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ResetTimer()
	for range b.N {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			b.Fatalf("GetProjectedCost() failed: %v", err)
		}
	}
}

// BenchmarkGetPricingSpec benchmarks the GetPricingSpec RPC method.
func BenchmarkGetPricingSpec(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ResetTimer()
	for range b.N {
		_, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
			Resource: resource,
		})
		if err != nil {
			b.Fatalf("GetPricingSpec() failed: %v", err)
		}
	}
}

// BenchmarkEstimateCost benchmarks the EstimateCost RPC method.
func BenchmarkEstimateCost(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   nil,
		})
		if err != nil {
			b.Fatalf("EstimateCost() failed: %v", err)
		}
	}
}

// BenchmarkAllMethods benchmarks all RPC methods in sequence.
func BenchmarkAllMethods(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)

	b.ResetTimer()
	for range b.N {
		// Name
		_, err := client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			b.Fatalf("Name() failed: %v", err)
		}

		// Supports
		_, err = client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		if err != nil {
			b.Fatalf("Supports() failed: %v", err)
		}

		// GetActualCost
		_, err = client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
		})
		if err != nil {
			b.Fatalf("GetActualCost() failed: %v", err)
		}

		// GetProjectedCost
		_, err = client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			b.Fatalf("GetProjectedCost() failed: %v", err)
		}

		// GetPricingSpec
		_, err = client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
			Resource: resource,
		})
		if err != nil {
			b.Fatalf("GetPricingSpec() failed: %v", err)
		}

		// EstimateCost
		_, err = client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   nil,
		})
		if err != nil {
			b.Fatalf("EstimateCost() failed: %v", err)
		}
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent RPC requests.
func BenchmarkConcurrentRequests(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Name(ctx, &pbc.NameRequest{})
			if err != nil {
				b.Fatalf("Name() failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentEstimateCost benchmarks concurrent EstimateCost requests.
// This is a key benchmark for T044 - validates 50+ concurrent requests under load.
func BenchmarkConcurrentEstimateCost(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   nil,
			})
			if err != nil {
				b.Fatalf("EstimateCost() failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentEstimateCost50 benchmarks exactly 50 concurrent EstimateCost requests.
// This benchmark validates Advanced conformance requirements per T044.
func BenchmarkConcurrentEstimateCost50(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		var wg sync.WaitGroup
		errChan := make(chan error, plugintesting.AdvancedParallelRequests)

		for range plugintesting.AdvancedParallelRequests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
					ResourceType: "aws:ec2/instance:Instance",
					Attributes:   nil,
				})
				if err != nil {
					errChan <- err
				}
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			b.Fatalf("EstimateCost() failed: %v", err)
		}
	}
}

// BenchmarkConcurrentEstimateCostLatency measures per-request latency under concurrent load.
// Validates <500ms response time requirement under 50+ concurrent requests.
func BenchmarkConcurrentEstimateCostLatency(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		var wg sync.WaitGroup
		latencies := make(chan time.Duration, plugintesting.AdvancedParallelRequests)
		errChan := make(chan error, plugintesting.AdvancedParallelRequests)

		for range plugintesting.AdvancedParallelRequests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
					ResourceType: "aws:ec2/instance:Instance",
					Attributes:   nil,
				})
				latencies <- time.Since(start)
				if err != nil {
					errChan <- err
				}
			}()
		}

		wg.Wait()
		close(latencies)
		close(errChan)

		// Check for errors
		for err := range errChan {
			b.Fatalf("EstimateCost() failed: %v", err)
		}

		// Verify all latencies are under 500ms
		for latency := range latencies {
			if latency > 500*time.Millisecond {
				b.Fatalf("Latency %v exceeds 500ms threshold", latency)
			}
		}
	}
}

// BenchmarkActualCostDataSizes benchmarks GetActualCost with different data sizes.
func BenchmarkActualCostDataSizes(b *testing.B) {
	testCases := []struct {
		name       string
		hours      int
		dataPoints int
	}{
		{"1Hour", 1, 1},
		{"24Hours", plugintesting.HoursPerDay, plugintesting.HoursPerDay},
		{"7Days", 168, 168},
		{"30Days", plugintesting.HoursIn30Days, plugintesting.HoursIn30Days},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			plugin.SetActualCostDataPoints(tc.dataPoints)

			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(&testing.T{})
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()
			start, end := plugintesting.CreateTimeRange(tc.hours)

			b.ResetTimer()
			for range b.N {
				_, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
					ResourceId: "test-resource",
					Start:      start,
					End:        end,
				})
				if err != nil {
					b.Fatalf("GetActualCost() failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkDifferentProviders benchmarks requests across different providers.
func BenchmarkDifferentProviders(b *testing.B) {
	providers := []struct {
		name         string
		provider     string
		resourceType string
		sku          string
	}{
		{"AWS", "aws", "ec2", "t3.micro"},
		{"Azure", "azure", "vm", "Standard_B1s"},
		{"GCP", "gcp", "compute_engine", "n1-standard-1"},
		{"Kubernetes", "kubernetes", "namespace", ""},
	}

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(&testing.T{})
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()
			resource := plugintesting.CreateResourceDescriptor(
				p.provider,
				p.resourceType,
				p.sku,
				"us-east-1",
			)

			b.ResetTimer()
			for range b.N {
				_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
					Resource: resource,
				})
				if err != nil {
					b.Fatalf("GetProjectedCost() failed: %v", err)
				}
			}
		})
	}
}

// PerformanceTestSuite provides standardized performance testing.
type PerformanceTestSuite struct {
	impl pbc.CostSourceServiceServer
}

// NewPerformanceTestSuite creates a new performance test suite.
func NewPerformanceTestSuite(impl pbc.CostSourceServiceServer) *PerformanceTestSuite {
	return &PerformanceTestSuite{impl: impl}
}

// RunPerformanceTests runs a standardized set of performance tests.
func (pts *PerformanceTestSuite) RunPerformanceTests(t *testing.T) {
	harness := plugintesting.NewTestHarness(pts.impl)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Test Name performance
	nameMetrics, err := plugintesting.MeasurePerformance(
		"Name",
		plugintesting.NumPerformanceIterations,
		func() error {
			_, err := client.Name(ctx, &pbc.NameRequest{})
			return err
		},
	)
	if err != nil {
		t.Fatalf("Name performance test failed: %v", err)
	}
	t.Logf("Name Performance: avg=%v, min=%v, max=%v",
		nameMetrics.AvgDuration, nameMetrics.MinDuration, nameMetrics.MaxDuration)

	// Test Supports performance
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
	supportsMetrics, err := plugintesting.MeasurePerformance(
		"Supports",
		plugintesting.NumPerformanceIterations,
		func() error {
			_, callErr := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
			return callErr
		},
	)
	if err != nil {
		t.Fatalf("Supports performance test failed: %v", err)
	}
	t.Logf("Supports Performance: avg=%v, min=%v, max=%v",
		supportsMetrics.AvgDuration, supportsMetrics.MinDuration, supportsMetrics.MaxDuration)

	// Test GetProjectedCost performance
	projectedMetrics, err := plugintesting.MeasurePerformance(
		"GetProjectedCost",
		plugintesting.NumPerformanceIterations,
		func() error {
			_, callErr := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
				Resource: resource,
			})
			return callErr
		},
	)
	if err != nil {
		t.Fatalf("GetProjectedCost performance test failed: %v", err)
	}
	t.Logf("GetProjectedCost Performance: avg=%v, min=%v, max=%v",
		projectedMetrics.AvgDuration, projectedMetrics.MinDuration, projectedMetrics.MaxDuration)

	// Test GetActualCost performance
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	actualMetrics, err := plugintesting.MeasurePerformance(
		"GetActualCost",
		plugintesting.ReducedIterations,
		func() error {
			_, callErr := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
				ResourceId: "test-resource",
				Start:      start,
				End:        end,
			})
			return callErr
		},
	)
	if err != nil {
		t.Fatalf("GetActualCost performance test failed: %v", err)
	}
	t.Logf("GetActualCost Performance: avg=%v, min=%v, max=%v",
		actualMetrics.AvgDuration, actualMetrics.MinDuration, actualMetrics.MaxDuration)

	// Test GetPricingSpec performance
	specMetrics, err := plugintesting.MeasurePerformance(
		"GetPricingSpec",
		plugintesting.NumPerformanceIterations,
		func() error {
			_, callErr := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
				Resource: resource,
			})
			return callErr
		},
	)
	if err != nil {
		t.Fatalf("GetPricingSpec performance test failed: %v", err)
	}
	t.Logf("GetPricingSpec Performance: avg=%v, min=%v, max=%v",
		specMetrics.AvgDuration, specMetrics.MinDuration, specMetrics.MaxDuration)
}

// largeResultSetRecommendationCount is the number of recommendations for large result set testing.
const largeResultSetRecommendationCount = 10000

// BenchmarkGetRecommendations_LargeResultSet benchmarks GetRecommendations with 10,000 recommendations
// returned in a single response. Per SC-005, this should complete in <500ms.
func BenchmarkGetRecommendations_LargeResultSet(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	// Generate 10,000 recommendations
	plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: plugintesting.GenerateSampleRecommendations(largeResultSetRecommendationCount),
	})

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		// PageSize=0 returns all results without pagination for SC-005 large result set testing.
		_, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{})
		if err != nil {
			b.Fatalf("GetRecommendations() failed: %v", err)
		}
	}
}

// BenchmarkGetRecommendations_LargeResultSetPagination benchmarks paginating through 10,000 recommendations.
func BenchmarkGetRecommendations_LargeResultSetPagination(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: plugintesting.GenerateSampleRecommendations(largeResultSetRecommendationCount),
	})

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var token string
		for {
			resp, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
				PageSize:  100,
				PageToken: token,
			})
			if err != nil {
				b.Fatalf("GetRecommendations() failed: %v", err)
			}
			token = resp.GetNextPageToken()
			if token == "" {
				break
			}
		}
	}
}

// TestGetRecommendations_LargeResultSetLatency validates <500ms requirement for large result sets.
// This tests fetching all 10k recommendations in a single response per SC-005.
func TestGetRecommendations_LargeResultSetLatency(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: plugintesting.GenerateSampleRecommendations(largeResultSetRecommendationCount),
	})

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Measure single request latency for the full result set
	// PageSize=0 returns all results without pagination for SC-005 large result set testing.
	start := time.Now()
	_, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GetRecommendations() failed: %v", err)
	}

	maxLatency := 500 * time.Millisecond
	if duration > maxLatency {
		t.Errorf("GetRecommendations latency %v exceeds %v requirement", duration, maxLatency)
	}
	t.Logf("GetRecommendations latency for 10k recommendations: %v", duration)
}

// =============================================================================
// FallbackHint Performance Benchmarks
// =============================================================================

// BenchmarkFallbackHintResponseCreation benchmarks creating responses with different FallbackHint values.
// Measures overhead of the functional options pattern for response construction.
func BenchmarkFallbackHintResponseCreation(b *testing.B) {
	b.Run("WithoutHint", func(b *testing.B) {
		results := []*pbc.ActualCostResult{
			{Cost: 10.0, Source: "aws-ce"},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			resp := &pbc.GetActualCostResponse{Results: results}
			_ = resp.GetResults() // Use the result to avoid unused write warning
		}
	})

	b.Run("WithFallbackHintNone", func(b *testing.B) {
		results := []*pbc.ActualCostResult{
			{Cost: 10.0, Source: "aws-ce"},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = plugintesting.NewActualCostResponseWithHint(
				results,
				pbc.FallbackHint_FALLBACK_HINT_NONE,
			)
		}
	})

	b.Run("WithFallbackHintRecommended", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = plugintesting.NewActualCostResponseWithHint(
				nil,
				pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED,
			)
		}
	})

	b.Run("WithFallbackHintRequired", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = plugintesting.NewActualCostResponseWithHint(
				nil,
				pbc.FallbackHint_FALLBACK_HINT_REQUIRED,
			)
		}
	})
}

// BenchmarkGetActualCostWithFallbackHint benchmarks GetActualCost RPC with different FallbackHint values.
// Validates that FallbackHint adds minimal overhead to RPC performance.
func BenchmarkGetActualCostWithFallbackHint(b *testing.B) {
	testCases := []struct {
		name string
		hint pbc.FallbackHint
	}{
		{"Unspecified", pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED},
		{"None", pbc.FallbackHint_FALLBACK_HINT_NONE},
		{"Recommended", pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED},
		{"Required", pbc.FallbackHint_FALLBACK_HINT_REQUIRED},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			plugin.SetFallbackHint(tc.hint)

			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(&testing.T{})
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()
			start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
					ResourceId: "test-resource",
					Start:      start,
					End:        end,
				})
				if err != nil {
					b.Fatalf("GetActualCost() failed: %v", err)
				}
				if resp.GetFallbackHint() != tc.hint {
					b.Fatalf("Expected hint %v, got %v", tc.hint, resp.GetFallbackHint())
				}
			}
		})
	}
}

// BenchmarkValidateActualCostResponse benchmarks response validation with various data sizes.
func BenchmarkValidateActualCostResponse(b *testing.B) {
	testCases := []struct {
		name        string
		resultCount int
	}{
		{"Empty", 0},
		{"1Result", 1},
		{"10Results", 10},
		{"100Results", 100},
		{"1000Results", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			results := make([]*pbc.ActualCostResult, tc.resultCount)
			for i := range tc.resultCount {
				results[i] = &pbc.ActualCostResult{
					Cost:   float64(i) * 1.5,
					Source: "aws-ce",
				}
			}
			resp := &pbc.GetActualCostResponse{
				Results:      results,
				FallbackHint: pbc.FallbackHint_FALLBACK_HINT_NONE,
			}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = pluginsdk.ValidateActualCostResponse(resp)
			}
		})
	}
}

// =============================================================================
// GetBudgets Performance Benchmarks - T040
// =============================================================================

// BenchmarkGetBudgets benchmarks the GetBudgets RPC method.
func BenchmarkGetBudgets(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := client.GetBudgets(ctx, &pbc.GetBudgetsRequest{
			Filter:        &pbc.BudgetFilter{},
			IncludeStatus: true,
		})
		if err != nil {
			b.Fatalf("GetBudgets() failed: %v", err)
		}
	}
}

// BenchmarkGetBudgets_Scale benchmarks GetBudgets with different budget counts (100-1000).
// Validates SC-007: System supports 100-1000 budgets per user/department.
func BenchmarkGetBudgets_Scale(b *testing.B) {
	testCases := []struct {
		name        string
		budgetCount int
	}{
		{"100Budgets", 100},
		{"250Budgets", 250},
		{"500Budgets", 500},
		{"1000Budgets", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			plugin.MockBudgets = generateMockBudgets(tc.budgetCount)

			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(&testing.T{})
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, err := client.GetBudgets(ctx, &pbc.GetBudgetsRequest{
					Filter:        &pbc.BudgetFilter{},
					IncludeStatus: true,
				})
				if err != nil {
					b.Fatalf("GetBudgets() failed: %v", err)
				}
			}
		})
	}
}

// TestGetBudgets_ScaleLatency validates <5s requirement for 100-1000 budgets (SC-001).
func TestGetBudgets_ScaleLatency(t *testing.T) {
	testCases := []struct {
		name        string
		budgetCount int
		maxLatency  time.Duration
	}{
		{"100Budgets", 100, 5 * time.Second},
		{"500Budgets", 500, 5 * time.Second},
		{"1000Budgets", 1000, 5 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := plugintesting.NewMockPlugin()
			plugin.MockBudgets = generateMockBudgets(tc.budgetCount)

			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(t)
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()

			start := time.Now()
			resp, err := client.GetBudgets(ctx, &pbc.GetBudgetsRequest{
				Filter:        &pbc.BudgetFilter{},
				IncludeStatus: true,
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("GetBudgets() failed: %v", err)
			}

			if duration > tc.maxLatency {
				t.Errorf("GetBudgets latency %v exceeds %v requirement for %d budgets",
					duration, tc.maxLatency, tc.budgetCount)
			}

			// Verify correct count returned
			if len(resp.GetBudgets()) != tc.budgetCount {
				t.Errorf("Expected %d budgets, got %d", tc.budgetCount, len(resp.GetBudgets()))
			}

			t.Logf("GetBudgets latency for %d budgets: %v", tc.budgetCount, duration)
		})
	}
}

// BenchmarkGetBudgets_ConcurrentScale benchmarks concurrent GetBudgets with large budget counts.
func BenchmarkGetBudgets_ConcurrentScale(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	plugin.MockBudgets = generateMockBudgets(500)

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.GetBudgets(ctx, &pbc.GetBudgetsRequest{
				Filter:        &pbc.BudgetFilter{},
				IncludeStatus: true,
			})
			if err != nil {
				b.Fatalf("GetBudgets() failed: %v", err)
			}
		}
	})
}

// generateMockBudgets creates n mock budgets with realistic data distribution.
func generateMockBudgets(n int) []*pbc.Budget {
	budgets := make([]*pbc.Budget, n)
	providers := []string{"aws", "gcp", "azure", "kubernetes"}
	healthStatuses := []pbc.BudgetHealthStatus{
		pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK,
		pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING,
		pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL,
		pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED,
	}

	for i := range n {
		provider := providers[i%len(providers)]
		health := healthStatuses[i%len(healthStatuses)]
		limit := float64((i+1)*1000) + 500.0
		spent := limit * (0.5 + float64(i%50)/100.0) // 50-99% utilization

		idSuffix := fmt.Sprintf("%03d", i)
		budgets[i] = &pbc.Budget{
			Id:     "budget-" + provider + "-" + idSuffix,
			Name:   provider + " Budget " + idSuffix[:2],
			Source: provider + "-budgets",
			Amount: &pbc.BudgetAmount{
				Limit:    limit,
				Currency: "USD",
			},
			Period: pbc.BudgetPeriod_BUDGET_PERIOD_MONTHLY,
			Status: &pbc.BudgetStatus{
				CurrentSpend:         spent,
				ForecastedSpend:      spent * 1.1,
				PercentageUsed:       (spent / limit) * 100,
				PercentageForecasted: (spent * 1.1 / limit) * 100,
				Currency:             "USD",
				Health:               health,
			},
		}
	}
	return budgets
}

// =============================================================================
// DryRun Performance Benchmarks - T055
// =============================================================================

// BenchmarkDryRun benchmarks the DryRun RPC method.
// Validates <100ms p99 latency requirement for dry-run introspection.
func BenchmarkDryRun(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := client.DryRun(ctx, &pbc.DryRunRequest{Resource: resource})
		if err != nil {
			b.Fatalf("DryRun() failed: %v", err)
		}
	}
}

// BenchmarkDryRun_WithSimulationParameters benchmarks DryRun with simulation parameters.
func BenchmarkDryRun_WithSimulationParameters(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := client.DryRun(ctx, &pbc.DryRunRequest{
			Resource: resource,
			SimulationParameters: map[string]string{
				"deployment_mode": "multi-az",
				"pricing_tier":    "reserved",
			},
		})
		if err != nil {
			b.Fatalf("DryRun() failed: %v", err)
		}
	}
}

// BenchmarkDryRun_Concurrent benchmarks concurrent DryRun requests.
// Validates thread safety and performance under load.
func BenchmarkDryRun_Concurrent(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.DryRun(ctx, &pbc.DryRunRequest{Resource: resource})
			if err != nil {
				b.Fatalf("DryRun() failed: %v", err)
			}
		}
	})
}

// BenchmarkDryRun_DifferentProviders benchmarks DryRun across different providers.
func BenchmarkDryRun_DifferentProviders(b *testing.B) {
	providers := []struct {
		name         string
		provider     string
		resourceType string
	}{
		{"AWS", "aws", "ec2"},
		{"Azure", "azure", "vm"},
		{"GCP", "gcp", "compute_engine"},
		{"Kubernetes", "kubernetes", "namespace"},
	}

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(&testing.T{})
			defer harness.Stop()

			client := harness.Client()
			ctx := context.Background()
			resource := plugintesting.CreateResourceDescriptor(
				p.provider,
				p.resourceType,
				"",
				"us-east-1",
			)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, err := client.DryRun(ctx, &pbc.DryRunRequest{Resource: resource})
				if err != nil {
					b.Fatalf("DryRun() failed: %v", err)
				}
			}
		})
	}
}

// TestDryRunLatency validates <100ms p99 latency requirement.
func TestDryRunLatency(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	const iterations = 1000
	latencies := make([]time.Duration, iterations)

	for i := range iterations {
		start := time.Now()
		_, err := client.DryRun(ctx, &pbc.DryRunRequest{Resource: resource})
		latencies[i] = time.Since(start)

		if err != nil {
			t.Fatalf("DryRun() failed: %v", err)
		}
	}

	// Calculate p99 latency
	var total time.Duration
	var maxLatency time.Duration
	for _, l := range latencies {
		total += l
		if l > maxLatency {
			maxLatency = l
		}
	}
	avgLatency := total / iterations

	// Verify p99 < 100ms (in practice, in-memory gRPC is much faster)
	const maxP99 = 100 * time.Millisecond
	if maxLatency > maxP99 {
		t.Errorf("DryRun p99 latency %v exceeds %v requirement", maxLatency, maxP99)
	}

	t.Logf("DryRun latency stats: avg=%v, max=%v (iterations=%d)", avgLatency, maxLatency, iterations)
}

// =============================================================================
// GetPluginInfo Performance Benchmarks - T034
// =============================================================================

// BenchmarkGetPluginInfo benchmarks the GetPluginInfo RPC method.
// Measures the overhead of retrieving static metadata.
func BenchmarkGetPluginInfo(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	plugin.PluginName = "benchmark-plugin"
	plugin.PluginVersion = "1.0.0"
	plugin.SpecVersion = pluginsdk.SpecVersion
	plugin.SupportedProviders = []string{"aws", "gcp"}
	plugin.Metadata = map[string]string{
		"build": "release",
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if err != nil {
			b.Fatalf("GetPluginInfo() failed: %v", err)
		}
	}
}

// BenchmarkGetPluginInfo_Concurrent benchmarks concurrent GetPluginInfo requests.
// Ensures metadata retrieval is thread-safe and performant under load.
func BenchmarkGetPluginInfo_Concurrent(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	plugin.PluginName = "benchmark-plugin"
	plugin.PluginVersion = "1.0.0"
	plugin.SpecVersion = pluginsdk.SpecVersion
	plugin.SupportedProviders = []string{"aws", "gcp"}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
			if err != nil {
				b.Fatalf("GetPluginInfo() failed: %v", err)
			}
		}
	})
}

// BenchmarkGetActualCostPaginated benchmarks paginated GetActualCost through TestHarness.
//
//nolint:gocognit // Benchmark pagination loop inherently has nested control flow.
func BenchmarkGetActualCostPaginated(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetActualCostDataPoints(1000)
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(&testing.T{})
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(1000)

	b.Run("SinglePage", func(b *testing.B) {
		req := &pbc.GetActualCostRequest{
			ResourceId: "bench-resource",
			Start:      start,
			End:        end,
			PageSize:   50,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := client.GetActualCost(ctx, req)
			if err != nil {
				b.Fatalf("GetActualCost() failed: %v", err)
			}
		}
	})

	b.Run("FullIteration", func(b *testing.B) {
		const maxIterations = 200 // safety cap to prevent infinite loops
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pageToken := ""
			totalRecords := 0
			iterations := 0
			for {
				resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
					ResourceId: "bench-resource",
					Start:      start,
					End:        end,
					PageSize:   100,
					PageToken:  pageToken,
				})
				if err != nil {
					b.Fatalf("GetActualCost() failed: %v", err)
				}
				totalRecords += len(resp.GetResults())
				iterations++
				if iterations > maxIterations {
					b.Fatalf("exceeded %d iterations; possible infinite pagination loop", maxIterations)
				}
				if resp.GetNextPageToken() == "" {
					break
				}
				pageToken = resp.GetNextPageToken()
			}
			if totalRecords != 1000 {
				b.Fatalf("expected 1000 records, got %d", totalRecords)
			}
		}
	})
}

// BenchmarkExpiresAtHelpers benchmarks the expiration check helper functions.
// Target: < 10 ns/op, 0 allocs/op.
func BenchmarkExpiresAtHelpers(b *testing.B) {
	now := time.Now()
	futureTime := now.Add(6 * time.Hour)

	result := &pbc.ActualCostResult{
		ExpiresAt: timestamppb.New(futureTime),
	}
	projResp := &pbc.GetProjectedCostResponse{
		ExpiresAt: timestamppb.New(futureTime),
	}
	estResp := &pbc.EstimateCostResponse{
		ExpiresAt: timestamppb.New(futureTime),
	}
	nilResult := &pbc.ActualCostResult{}

	b.Run("IsActualCostExpired", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.IsActualCostExpired(result, now)
		}
	})

	b.Run("ActualCostExpiresAt", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.ActualCostExpiresAt(result)
		}
	})

	b.Run("IsProjectedCostExpired", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.IsProjectedCostExpired(projResp, now)
		}
	})

	b.Run("ProjectedCostExpiresAt", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.ProjectedCostExpiresAt(projResp)
		}
	})

	b.Run("IsEstimateCostExpired", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.IsEstimateCostExpired(estResp, now)
		}
	})

	b.Run("EstimateCostExpiresAt", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.EstimateCostExpiresAt(estResp)
		}
	})

	b.Run("WithEstimateCostExpiresAt", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.NewEstimateCostResponse(
				pluginsdk.WithEstimateCostExpiresAt(futureTime),
			)
		}
	})

	b.Run("IsActualCostExpired_Nil", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			pluginsdk.IsActualCostExpired(nilResult, now)
		}
	})
}

// BenchmarkActualCostWithExpiresAt measures GetActualCost RPC overhead
// when expires_at is set vs not set.
func BenchmarkActualCostWithExpiresAt(b *testing.B) {
	start, end := plugintesting.CreateTimeRange(5)
	req := &pbc.GetActualCostRequest{
		Start:      start,
		End:        end,
		ResourceId: "test-resource",
	}
	ctx := context.Background()

	b.Run("WithoutExpiresAt", func(b *testing.B) {
		plugin := plugintesting.NewMockPlugin()
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(b)
		defer harness.Stop()

		client := harness.Client()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := client.GetActualCost(ctx, req)
			if err != nil {
				b.Fatalf("GetActualCost() failed: %v", err)
			}
		}
	})

	b.Run("WithExpiresAt", func(b *testing.B) {
		plugin := plugintesting.NewMockPlugin()
		plugin.ExpiresAtDuration = 6 * time.Hour
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(b)
		defer harness.Stop()

		client := harness.Client()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := client.GetActualCost(ctx, req)
			if err != nil {
				b.Fatalf("GetActualCost() failed: %v", err)
			}
		}
	})
}

func BenchmarkBatchCostEstimate(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(b)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	for _, size := range []int{10, 50, 100} {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			req := &pbc.BatchCostRequest{
				QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
				Resources: benchmarkBatchResources(size),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				resp, err := client.BatchCost(ctx, req)
				if err != nil {
					b.Fatalf("BatchCost() failed: %v", err)
				}
				if len(resp.GetResults()) != size {
					b.Fatalf("expected %d results, got %d", size, len(resp.GetResults()))
				}
			}
		})
	}
}

func BenchmarkBatchCostActual(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(b)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     start,
		End:       end,
		Resources: benchmarkBatchResources(20),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		resp, err := client.BatchCost(ctx, req)
		if err != nil {
			b.Fatalf("BatchCost ACTUAL failed: %v", err)
		}
		if len(resp.GetResults()) != 20 {
			b.Fatalf("expected 20 results, got %d", len(resp.GetResults()))
		}
	}
}

func BenchmarkBatchCostFallback(b *testing.B) {
	ctx := context.Background()
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: benchmarkBatchResources(50),
	}

	b.Run("custom_handler", func(b *testing.B) {
		plugin := plugintesting.NewMockPlugin()
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(b)
		defer harness.Stop()

		client := harness.Client()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			resp, err := client.BatchCost(ctx, req)
			if err != nil {
				b.Fatalf("custom BatchCost failed: %v", err)
			}
			if len(resp.GetResults()) != 50 {
				b.Fatalf("expected 50 results, got %d", len(resp.GetResults()))
			}
		}
	})

	b.Run("fallback_handler", func(b *testing.B) {
		plugin := &fallbackBatchPlugin{}
		server := pluginsdk.NewServer(plugin)
		harness := plugintesting.NewTestHarness(server)
		harness.Start(b)
		defer harness.Stop()

		client := harness.Client()
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			resp, err := client.BatchCost(ctx, req)
			if err != nil {
				b.Fatalf("fallback BatchCost failed: %v", err)
			}
			if len(resp.GetResults()) != 50 {
				b.Fatalf("expected 50 results, got %d", len(resp.GetResults()))
			}
		}
	})
}

func BenchmarkBatchCostConcurrent(b *testing.B) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(b)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	req := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: benchmarkBatchResources(10),
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.BatchCost(ctx, req)
			if err != nil {
				b.Fatalf("BatchCost concurrent failed: %v", err)
			}
			if len(resp.GetResults()) != 10 {
				b.Fatalf("expected 10 results, got %d", len(resp.GetResults()))
			}
		}
	})
}

func benchmarkBatchResources(count int) []*pbc.ResourceDescriptor {
	resources := make([]*pbc.ResourceDescriptor, count)
	for i := range count {
		resources[i] = plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resources[i].Id = fmt.Sprintf("batch-%d", i)
	}
	return resources
}
