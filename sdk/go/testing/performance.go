// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements performance testing and benchmarking for plugin conformance.
package testing

import (
	"context"
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// PerformanceBaseline defines latency thresholds for performance conformance.
type PerformanceBaseline struct {
	// Method is the RPC method name.
	Method string `json:"method"`

	// StandardLatency is the maximum allowed latency for Standard conformance.
	StandardLatency time.Duration `json:"standard_latency"`

	// AdvancedLatency is the maximum allowed latency for Advanced conformance.
	AdvancedLatency time.Duration `json:"advanced_latency"`

	// MaxAllocBytes is the maximum allowed memory allocation per call (optional).
	MaxAllocBytes int64 `json:"max_alloc_bytes,omitempty"`
}

// DefaultBaselines returns the canonical performance baselines.
// These thresholds are referenced from sdk/go/testing/README.md.
func DefaultBaselines() []PerformanceBaseline {
	return []PerformanceBaseline{
		{
			Method:          MethodName,
			StandardLatency: NameStandardLatencyMs * time.Millisecond,
			AdvancedLatency: NameAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          MethodSupports,
			StandardLatency: SupportsStandardLatencyMs * time.Millisecond,
			AdvancedLatency: SupportsAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          MethodGetPluginInfo,
			StandardLatency: GetPluginInfoStandardLatencyMs * time.Millisecond,
			AdvancedLatency: GetPluginInfoAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          MethodGetProjectedCost,
			StandardLatency: ProjectedCostStandardLatencyMs * time.Millisecond,
			AdvancedLatency: ProjectedCostAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          MethodGetPricingSpec,
			StandardLatency: PricingSpecStandardLatencyMs * time.Millisecond,
			AdvancedLatency: PricingSpecAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          MethodGetBudgets,
			StandardLatency: GetBudgetsStandardLatencyMs * time.Millisecond,
			AdvancedLatency: GetBudgetsAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          "GetActualCost_24h",
			StandardLatency: ActualCost24hStandardLatencyMs * time.Millisecond,
			AdvancedLatency: ActualCost24hAdvancedLatencyMs * time.Millisecond,
		},
		{
			Method:          "GetActualCost_30d",
			StandardLatency: 0, // Not required for Standard
			AdvancedLatency: ActualCost30dAdvancedLatencyMs * time.Millisecond,
		},
	}
}

// GetBaseline returns the performance baseline for a specific method.
// Returns nil if no baseline is defined for the method.
func GetBaseline(method string) *PerformanceBaseline {
	for _, b := range DefaultBaselines() {
		if b.Method == method {
			return &b
		}
	}
	return nil
}

// PerformanceResult contains the results of a performance benchmark.
type PerformanceResult struct {
	// Method is the RPC method benchmarked.
	Method string `json:"method"`

	// Iterations is the number of iterations run.
	Iterations int `json:"iterations"`

	// MinLatency is the minimum observed latency.
	MinLatency time.Duration `json:"min_latency"`

	// AvgLatency is the average observed latency.
	AvgLatency time.Duration `json:"avg_latency"`

	// MaxLatency is the maximum observed latency.
	MaxLatency time.Duration `json:"max_latency"`

	// TotalAllocBytes is the total bytes allocated during the benchmark.
	TotalAllocBytes int64 `json:"total_alloc_bytes,omitempty"`

	// AllocsPerOp is the number of allocations per operation.
	AllocsPerOp int64 `json:"allocs_per_op,omitempty"`

	// PassedStandard indicates if the benchmark passed Standard conformance.
	PassedStandard bool `json:"passed_standard"`

	// PassedAdvanced indicates if the benchmark passed Advanced conformance.
	PassedAdvanced bool `json:"passed_advanced"`

	// VariancePercent is the variance as a percentage of the baseline (for SC-003).
	VariancePercent float64 `json:"variance_percent,omitempty"`
}

// Passed returns true if the benchmark passed the specified conformance level.
func (r *PerformanceResult) Passed(level ConformanceLevel) bool {
	switch level {
	case ConformanceLevelBasic:
		return true // Basic has no performance requirements
	case ConformanceLevelStandard:
		return r.PassedStandard
	case ConformanceLevelAdvanced:
		return r.PassedAdvanced
	default:
		return false
	}
}

// MaxVariancePercent is the maximum allowed variance from baseline (SC-003).
const MaxVariancePercent = 10.0

// measureLatency measures the latency of a function over multiple iterations.
func measureLatency(name string, iterations int, fn func() error) *PerformanceResult {
	result := &PerformanceResult{
		Method:     name,
		Iterations: iterations,
		MinLatency: time.Hour, // Start with high value
	}

	var totalDuration time.Duration
	for range iterations {
		start := time.Now()
		_ = fn()
		duration := time.Since(start)

		totalDuration += duration
		if duration < result.MinLatency {
			result.MinLatency = duration
		}
		if duration > result.MaxLatency {
			result.MaxLatency = duration
		}
	}

	if iterations > 0 {
		result.AvgLatency = totalDuration / time.Duration(iterations)
	}

	return result
}

// compareToBaseline compares performance results against baseline thresholds.
func compareToBaseline(result *PerformanceResult, baseline *PerformanceBaseline) {
	if baseline == nil {
		return
	}

	// Check Standard conformance
	if baseline.StandardLatency > 0 {
		result.PassedStandard = result.AvgLatency <= baseline.StandardLatency

		// Calculate variance percentage for SC-003
		if baseline.StandardLatency > 0 {
			variance := float64(result.AvgLatency-baseline.StandardLatency) /
				float64(baseline.StandardLatency) * PercentageCalculationFactor
			result.VariancePercent = variance
		}
	} else {
		result.PassedStandard = true // No Standard requirement for this method
	}

	// Check Advanced conformance
	if baseline.AdvancedLatency > 0 {
		result.PassedAdvanced = result.AvgLatency <= baseline.AdvancedLatency
	} else {
		result.PassedAdvanced = true // No Advanced requirement for this method
	}
}

// PerformanceTests returns the performance conformance tests.
func PerformanceTests() []ConformanceSuiteTest {
	return []ConformanceSuiteTest{
		{
			Name:        "Performance_NameLatency",
			Description: "Validates Name RPC latency within thresholds",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createNameLatencyTest(),
		},
		{
			Name:        "Performance_SupportsLatency",
			Description: "Validates Supports RPC latency within thresholds",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createSupportsLatencyTest(),
		},
		{
			Name:        "Performance_GetPluginInfoLatency",
			Description: "Validates GetPluginInfo RPC latency within thresholds",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetPluginInfoLatencyTest(),
		},
		{
			Name:        "Performance_GetProjectedCostLatency",
			Description: "Validates GetProjectedCost RPC latency within thresholds",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetProjectedCostLatencyTest(),
		},
		{
			Name:        "Performance_GetPricingSpecLatency",
			Description: "Validates GetPricingSpec RPC latency within thresholds",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetPricingSpecLatencyTest(),
		},
		{
			Name:        "Performance_GetBudgetsLatency",
			Description: "Validates GetBudgets RPC latency within thresholds (<5s)",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetBudgetsLatencyTest(),
		},
		{
			Name:        "Performance_BaselineVariance",
			Description: "Validates performance variance is within 10% of baseline (SC-003)",
			Category:    CategoryPerformance,
			MinLevel:    ConformanceLevelAdvanced,
			TestFunc:    createBaselineVarianceTest(),
		},
	}
}

// buildLatencyTestResult builds a TestResult from performance measurement results.
func buildLatencyTestResult(method string, perfResult *PerformanceResult, baseline *PerformanceBaseline) TestResult {
	if !perfResult.PassedStandard {
		return TestResult{
			Method:   method,
			Category: CategoryPerformance,
			Success:  false,
			Error: fmt.Errorf("latency %.2fms exceeds threshold %.2fms",
				float64(perfResult.AvgLatency.Milliseconds()),
				float64(baseline.StandardLatency.Milliseconds())),
			Duration: perfResult.AvgLatency,
			Details: fmt.Sprintf("Avg: %v, Min: %v, Max: %v",
				perfResult.AvgLatency, perfResult.MinLatency, perfResult.MaxLatency),
		}
	}

	return TestResult{
		Method:   method,
		Category: CategoryPerformance,
		Success:  true,
		Duration: perfResult.AvgLatency,
		Details:  fmt.Sprintf("Avg: %v (threshold: %v)", perfResult.AvgLatency, baseline.StandardLatency),
	}
}

func createNameLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodName)
		result := measureLatency(MethodName, LatencyTestIterations, func() error {
			_, callErr := harness.Client().Name(context.Background(), &pbc.NameRequest{})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodName, result, baseline)
	}
}

func createSupportsLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodSupports)
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		result := measureLatency(MethodSupports, LatencyTestIterations, func() error {
			_, callErr := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{Resource: resource})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodSupports, result, baseline)
	}
}

func createGetPluginInfoLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodGetPluginInfo)
		result := measureLatency(MethodGetPluginInfo, LatencyTestIterations, func() error {
			_, callErr := harness.Client().GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodGetPluginInfo, result, baseline)
	}
}
func createGetProjectedCostLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodGetProjectedCost)
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		result := measureLatency(MethodGetProjectedCost, LatencyTestIterations, func() error {
			_, callErr := harness.Client().GetProjectedCost(context.Background(),
				&pbc.GetProjectedCostRequest{Resource: resource})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodGetProjectedCost, result, baseline)
	}
}

func createGetPricingSpecLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodGetPricingSpec)
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		result := measureLatency(MethodGetPricingSpec, LatencyTestIterations, func() error {
			_, callErr := harness.Client().GetPricingSpec(context.Background(),
				&pbc.GetPricingSpecRequest{Resource: resource})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodGetPricingSpec, result, baseline)
	}
}

func createGetBudgetsLatencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		baseline := GetBaseline(MethodGetBudgets)
		result := measureLatency(MethodGetBudgets, LatencyTestIterations, func() error {
			_, callErr := harness.Client().GetBudgets(context.Background(),
				&pbc.GetBudgetsRequest{
					Filter:        &pbc.BudgetFilter{},
					IncludeStatus: false, // Faster without status
				})
			return callErr
		})
		compareToBaseline(result, baseline)
		return buildLatencyTestResult(MethodGetBudgets, result, baseline)
	}
}

// createBaselineVarianceTest validates SC-003: benchmark variance within 10%.
func createBaselineVarianceTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()

		// Measure Name latency and check variance
		baseline := GetBaseline(MethodName)
		result := measureLatency(MethodName, VarianceTestIterations, func() error {
			_, callErr := harness.Client().Name(context.Background(), &pbc.NameRequest{})
			return callErr
		})

		compareToBaseline(result, baseline)
		duration := time.Since(start)

		// Check if variance is within 10%
		if result.VariancePercent > MaxVariancePercent {
			return TestResult{
				Method:   performanceLevel,
				Category: CategoryPerformance,
				Success:  false,
				Error: fmt.Errorf(
					"variance %.1f%% exceeds threshold %.1f%%",
					result.VariancePercent,
					MaxVariancePercent,
				),
				Duration: duration,
				Details:  fmt.Sprintf("Avg latency: %v, Baseline: %v", result.AvgLatency, baseline.StandardLatency),
			}
		}

		return TestResult{
			Method:   performanceLevel,
			Category: CategoryPerformance,
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Variance: %.1f%% (threshold: %.1f%%)", result.VariancePercent, MaxVariancePercent),
		}
	}
}

// RegisterPerformanceTests registers performance tests with a conformance suite.
func RegisterPerformanceTests(suite *ConformanceSuite) {
	for _, test := range PerformanceTests() {
		suite.AddTest(test)
	}
}

// RunPerformanceBenchmarks runs performance benchmarks against a plugin.
func RunPerformanceBenchmarks(impl pbc.CostSourceServiceServer) ([]PerformanceResult, error) {
	harness := NewTestHarness(impl)

	conn, err := harness.createClientConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	harness.client = pbc.NewCostSourceServiceClient(conn)

	var results []PerformanceResult

	// Benchmark Name
	nameResult := measureLatency("Name", NumPerformanceIterations, func() error {
		_, callErr := harness.Client().Name(context.Background(), &pbc.NameRequest{})
		return callErr
	})
	compareToBaseline(nameResult, GetBaseline("Name"))
	results = append(results, *nameResult)

	// Benchmark Supports
	resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
	supportsResult := measureLatency("Supports", NumPerformanceIterations, func() error {
		_, callErr := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{Resource: resource})
		return callErr
	})
	compareToBaseline(supportsResult, GetBaseline("Supports"))
	results = append(results, *supportsResult)

	// Benchmark GetProjectedCost
	projectedResult := measureLatency("GetProjectedCost", ReducedIterations, func() error {
		_, callErr := harness.Client().
			GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{Resource: resource})
		return callErr
	})
	compareToBaseline(projectedResult, GetBaseline("GetProjectedCost"))
	results = append(results, *projectedResult)

	// Benchmark GetPricingSpec
	specResult := measureLatency("GetPricingSpec", ReducedIterations, func() error {
		_, callErr := harness.Client().
			GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{Resource: resource})
		return callErr
	})
	compareToBaseline(specResult, GetBaseline("GetPricingSpec"))
	results = append(results, *specResult)

	harness.Stop()
	return results, nil
}
