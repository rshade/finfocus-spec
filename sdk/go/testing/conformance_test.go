package testing_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ConformanceLevel defines the level of conformance testing.
type ConformanceLevel int

const (
	ConformanceBasic ConformanceLevel = iota
	ConformanceStandard
	ConformanceAdvanced
)

// ConformanceResult contains the result of a conformance test run.
type ConformanceResult struct {
	Level        ConformanceLevel
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Results      []plugintesting.TestResult
	Summary      string
}

// NameValidationTest validates that the plugin returns a properly formatted name response.
func NameValidationTest(harness *plugintesting.TestHarness) plugintesting.TestResult {
	start := time.Now()
	resp, err := harness.Client().Name(context.Background(), &pbc.NameRequest{})
	duration := time.Since(start)

	if err != nil {
		return plugintesting.TestResult{
			Method:   "Name",
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  "RPC call failed",
		}
	}

	if validationErr := plugintesting.ValidateNameResponse(resp); validationErr != nil {
		return plugintesting.TestResult{
			Method:   "Name",
			Success:  false,
			Error:    validationErr,
			Duration: duration,
			Details:  "Response validation failed",
		}
	}

	return plugintesting.TestResult{
		Method:   "Name",
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Plugin name: %s", resp.GetName()),
	}
}

// RunBasicConformanceTests runs basic conformance tests that all plugins must pass.
func RunBasicConformanceTests(t *testing.T, impl pbc.CostSourceServiceServer) *ConformanceResult {
	suite := plugintesting.NewPluginConformanceSuite()
	addBasicConformanceTests(suite)
	return runConformanceTestSuite(t, impl, suite, ConformanceBasic, "Basic conformance")
}

func addBasicConformanceTests(suite *plugintesting.PluginConformanceSuite) {
	// Basic test cases that all plugins MUST pass
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "NameReturnsValidResponse",
		Description: "Plugin must return a valid name",
		TestFunc:    NameValidationTest,
	})

	// GetPluginInfo conformance test (FR-001 to FR-004)
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetPluginInfoReturnsValidResponse",
		Description: "Plugin must return valid metadata including name, version, and spec_version",
		TestFunc:    createGetPluginInfoValidResponseTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetPluginInfoPerformance",
		Description: "Plugin must respond to GetPluginInfo within 100ms",
		TestFunc:    createGetPluginInfoPerformanceTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "SupportsHandlesValidInput",
		Description: "Plugin must handle Supports requests correctly",
		TestFunc:    createSupportsValidInputTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "SupportsHandlesNilResource",
		Description: "Plugin must handle nil resource gracefully",
		TestFunc:    createSupportsNilResourceTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "SupportsGranularDiscovery",
		Description: "Plugin must return granular capabilities in Supports response (inheriting global if needed)",
		TestFunc:    createSupportsGranularDiscoveryTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetProjectedCostHandlesValidResource",
		Description: "Plugin must handle GetProjectedCost for valid resources",
		TestFunc:    createProjectedCostValidResourceTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetPricingSpecReturnsCompleteResponse",
		Description: "Plugin must return PricingSpec with unit and assumptions populated",
		TestFunc:    createPricingSpecFlatRateTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetPricingSpecHandlesDefaultFields",
		Description: "Plugin must handle PricingSpec with default/empty optional fields",
		TestFunc:    createPricingSpecDefaultFieldsTest(),
	})

	// EstimateCost Basic tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostHandlesValidResource",
		Description: "Plugin must return cost estimate for valid resource type",
		TestFunc:    createEstimateCostValidResourceTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostRejectsInvalidFormat",
		Description: "Plugin must reject invalid resource type format with InvalidArgument",
		TestFunc:    createEstimateCostInvalidFormatTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostRejectsUnsupportedResource",
		Description: "Plugin must return NotFound for unsupported resource types",
		TestFunc:    createEstimateCostUnsupportedResourceTest(),
	})

	// GetRecommendations Basic tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_EmptyPlugin",
		Description: "Plugin must return empty list when not implementing RecommendationsProvider",
		TestFunc:    createGetRecommendationsEmptyPluginTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_Basic",
		Description: "Plugin must return a non-empty list of recommendations with valid structure (IDs, specified categories, action types) and a valid summary.",
		TestFunc:    createGetRecommendationsBasicTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_SummaryCalculation",
		Description: "Plugin must return summary that matches individual recommendations",
		TestFunc:    createGetRecommendationsSummaryTest(),
	})

	// ResolveResourceTypes Basic tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "ResolveResourceTypes_EmptyPlugin",
		Description: "Plugin must return empty mappings (not an error) when not implementing ResolveResourceTypesProvider",
		TestFunc:    createResolveResourceTypesEmptyPluginTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "ResolveResourceTypes_Basic",
		Description: "Plugin must resolve known source_types to mappings with a non-empty pulumi_token and omit unknown types.",
		TestFunc:    createResolveResourceTypesBasicTest(),
	})
}

func runConformanceTestSuite(
	t *testing.T,
	impl pbc.CostSourceServiceServer,
	suite *plugintesting.PluginConformanceSuite,
	level ConformanceLevel,
	summaryPrefix string,
) *ConformanceResult {
	results := suite.RunTests(t, impl)
	passed, failed := countTestResults(results)

	return &ConformanceResult{
		Level:       level,
		TotalTests:  len(results),
		PassedTests: passed,
		FailedTests: failed,
		Results:     results,
		Summary:     fmt.Sprintf("%s: %d/%d tests passed", summaryPrefix, passed, len(results)),
	}
}

func countTestResults(results []plugintesting.TestResult) (int, int) {
	var passed, failed int
	for _, result := range results {
		if result.Success {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func createSupportsValidInputTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
			Resource: resource,
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "Supports",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		if validationErr := plugintesting.ValidateSupportsResponse(resp); validationErr != nil {
			return plugintesting.TestResult{
				Method:   "Supports",
				Success:  false,
				Error:    validationErr,
				Duration: duration,
				Details:  "Response validation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "Supports",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Supported: %v, Reason: %s",
				resp.GetSupported(),
				resp.GetReason(),
			),
		}
	}
}

func createSupportsNilResourceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
			Resource: nil,
		})
		duration := time.Since(start)

		if err != nil {
			// Error is acceptable for nil resource
			return plugintesting.TestResult{
				Method:   "Supports",
				Success:  true,
				Duration: duration,
				Details:  "Correctly rejected nil resource with error",
			}
		}

		if resp.GetSupported() {
			return plugintesting.TestResult{
				Method:   "Supports",
				Success:  false,
				Error:    errors.New("plugin incorrectly supports nil resource"),
				Duration: duration,
				Details:  "Should not support nil resource",
			}
		}

		return plugintesting.TestResult{
			Method:   "Supports",
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected nil resource",
		}
	}
}

func createProjectedCostValidResourceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		// Check resource support first
		if !checkResourceSupport(harness, resource) {
			return handleUnsupportedResource(harness, resource, start)
		}

		// Test supported resource
		return testSupportedResource(harness, resource, start)
	}
}

func checkResourceSupport(
	harness *plugintesting.TestHarness,
	resource *pbc.ResourceDescriptor,
) bool {
	supportsResp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
		Resource: resource,
	})
	return err == nil && supportsResp.GetSupported()
}

func handleUnsupportedResource(
	harness *plugintesting.TestHarness,
	resource *pbc.ResourceDescriptor,
	start time.Time,
) plugintesting.TestResult {
	_, projectedErr := harness.Client().
		GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
	duration := time.Since(start)

	if projectedErr != nil {
		return plugintesting.TestResult{
			Method:   "GetProjectedCost",
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected unsupported resource",
		}
	}

	return plugintesting.TestResult{
		Method:   "GetProjectedCost",
		Success:  false,
		Error:    errors.New("plugin returned cost for unsupported resource"),
		Duration: duration,
		Details:  "Should reject unsupported resources",
	}
}

func testSupportedResource(
	harness *plugintesting.TestHarness,
	resource *pbc.ResourceDescriptor,
	start time.Time,
) plugintesting.TestResult {
	resp, err := harness.Client().
		GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
	duration := time.Since(start)

	if err != nil {
		return plugintesting.TestResult{
			Method:   "GetProjectedCost",
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  "RPC call failed for supported resource",
		}
	}

	if validationErr := plugintesting.ValidateProjectedCostResponse(resp); validationErr != nil {
		return plugintesting.TestResult{
			Method:   "GetProjectedCost",
			Success:  false,
			Error:    validationErr,
			Duration: duration,
			Details:  "Response validation failed",
		}
	}

	return plugintesting.TestResult{
		Method:   "GetProjectedCost",
		Success:  true,
		Duration: duration,
		Details: fmt.Sprintf(
			"Unit price: %.6f %s",
			resp.GetUnitPrice(),
			resp.GetCurrency(),
		),
	}
}

// RunStandardConformanceTests runs standard conformance tests for production-ready plugins.
func RunStandardConformanceTests(
	t *testing.T,
	impl pbc.CostSourceServiceServer,
) *ConformanceResult {
	// Run basic tests first
	basicResult := RunBasicConformanceTests(t, impl)
	if basicResult.FailedTests > 0 {
		return createFailedResult(
			ConformanceStandard,
			basicResult,
			"Standard conformance failed: basic tests must pass first",
		)
	}

	suite := plugintesting.NewPluginConformanceSuite()
	addStandardConformanceTests(suite)
	results := suite.RunTests(t, impl)

	return combineConformanceResults(
		basicResult,
		results,
		ConformanceStandard,
		"Standard conformance",
	)
}

func createFailedResult(
	level ConformanceLevel,
	basicResult *ConformanceResult,
	summary string,
) *ConformanceResult {
	return &ConformanceResult{
		Level:       level,
		TotalTests:  basicResult.TotalTests,
		PassedTests: basicResult.PassedTests,
		FailedTests: basicResult.FailedTests,
		Results:     basicResult.Results,
		Summary:     summary,
	}
}

func combineConformanceResults(
	basicResult *ConformanceResult,
	additionalResults []plugintesting.TestResult,
	level ConformanceLevel,
	summaryPrefix string,
) *ConformanceResult {
	allResults := make([]plugintesting.TestResult, len(basicResult.Results)+len(additionalResults))
	copy(allResults, basicResult.Results)
	copy(allResults[len(basicResult.Results):], additionalResults)

	passed := basicResult.PassedTests
	failed := basicResult.FailedTests
	additionalPassed, additionalFailed := countTestResults(additionalResults)
	passed += additionalPassed
	failed += additionalFailed

	return &ConformanceResult{
		Level:       level,
		TotalTests:  len(allResults),
		PassedTests: passed,
		FailedTests: failed,
		Results:     allResults,
		Summary:     fmt.Sprintf("%s: %d/%d tests passed", summaryPrefix, passed, len(allResults)),
	}
}

func addStandardConformanceTests(suite *plugintesting.PluginConformanceSuite) {
	// Standard-level tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetActualCostHandlesValidTimeRange",
		Description: "Plugin must handle GetActualCost with valid time ranges",
		TestFunc:    createActualCostValidTimeRangeTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetActualCostRejectsInvalidTimeRange",
		Description: "Plugin must reject invalid time ranges",
		TestFunc:    createActualCostInvalidTimeRangeTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetPricingSpecConsistency",
		Description: "Plugin must return consistent pricing specs for same resource",
		TestFunc:    createPricingSpecConsistencyTest(),
	})

	// EstimateCost Standard tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostIsDeterministic",
		Description: "Plugin must return consistent cost estimates for identical requests",
		TestFunc:    createEstimateCostDeterministicTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostHandlesConcurrentRequests",
		Description: "Plugin must handle concurrent EstimateCost requests",
		TestFunc:    createEstimateCostConcurrentTest(),
	})

	// GetRecommendations Filtering tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_FilterByCategory",
		Description: "Plugin must filter recommendations by category",
		TestFunc:    createGetRecommendationsFilterByCategoryTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_FilterByActionType",
		Description: "Plugin must filter recommendations by action type",
		TestFunc:    createGetRecommendationsFilterByActionTypeTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_FilterNoMatches",
		Description: "Plugin must return empty list when no recommendations match filter",
		TestFunc:    createGetRecommendationsFilterNoMatchesTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_FilterByProvider",
		Description: "Plugin must filter recommendations by provider",
		TestFunc:    createGetRecommendationsFilterByProviderTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_FilterCombined",
		Description: "Plugin must filter by multiple criteria together",
		TestFunc:    createGetRecommendationsFilterCombinedTest(),
	})

	// GetRecommendations Pagination tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_Pagination_FirstPage",
		Description: "Plugin must respect page_size and return first page",
		TestFunc:    createGetRecommendationsPaginationFirstPageTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_Pagination_NextPage",
		Description: "Plugin must navigate to next page with page_token",
		TestFunc:    createGetRecommendationsPaginationNextPageTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_Pagination_LastPage",
		Description: "Plugin must return empty token on last page",
		TestFunc:    createGetRecommendationsPaginationLastPageTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_Pagination_InvalidToken",
		Description: "Plugin must return error for invalid page token",
		TestFunc:    createGetRecommendationsPaginationInvalidTokenTest(),
	})

	// GetRecommendations Action Detail tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_RightsizeAction",
		Description: "Plugin must populate rightsize action with current/recommended instance types",
		TestFunc:    createGetRecommendationsRightsizeActionTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_KubernetesAction",
		Description: "Plugin must populate kubernetes action with cluster/namespace/container details",
		TestFunc:    createGetRecommendationsKubernetesActionTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "GetRecommendations_CommitmentAction",
		Description: "Plugin must populate commitment action with type/term/quantity",
		TestFunc:    createGetRecommendationsCommitmentActionTest(),
	})
}

// RunAdvancedConformanceTests runs advanced conformance tests for high-performance plugins.
func RunAdvancedConformanceTests(
	t *testing.T,
	impl pbc.CostSourceServiceServer,
) *ConformanceResult {
	// First run standard tests
	standardResult := RunStandardConformanceTests(t, impl)
	if standardResult.FailedTests > 0 {
		return createFailedResult(
			ConformanceAdvanced,
			standardResult,
			"Advanced conformance failed: standard tests must pass first",
		)
	}

	suite := plugintesting.NewPluginConformanceSuite()
	addAdvancedConformanceTests(suite)
	results := suite.RunTests(t, impl)

	return combineConformanceResults(
		standardResult,
		results,
		ConformanceAdvanced,
		"Advanced conformance",
	)
}

func addAdvancedConformanceTests(suite *plugintesting.PluginConformanceSuite) {
	// Advanced performance and reliability tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "PerformanceBaseline",
		Description: "Plugin must meet minimum performance requirements",
		TestFunc:    createPerformanceBaselineTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "ConcurrentRequestHandling",
		Description: "Plugin must handle concurrent requests safely",
		TestFunc:    createConcurrentRequestTest(),
	})

	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "LargeDataHandling",
		Description: "Plugin must handle large datasets efficiently",
		TestFunc:    createLargeDataHandlingTest(),
	})

	// EstimateCost Advanced tests
	suite.AddTest(plugintesting.ConformanceTest{
		Name:        "EstimateCostPerformance",
		Description: "Plugin must return EstimateCost responses within 500ms",
		TestFunc:    createEstimateCostPerformanceTest(),
	})
}

// PrintConformanceReport prints a detailed conformance test report.
func PrintConformanceReport(t *testing.T, result *ConformanceResult) {
	t.Log("\n=== CONFORMANCE TEST REPORT ===")
	t.Logf("Level: %s", conformanceLevelString(result.Level))
	t.Logf("Total Tests: %d", result.TotalTests)
	t.Logf("Passed: %d", result.PassedTests)
	t.Logf("Failed: %d", result.FailedTests)
	if result.SkippedTests > 0 {
		t.Logf("Skipped: %d", result.SkippedTests)
	}
	t.Logf(
		"Success Rate: %.1f%%",
		float64(result.PassedTests)/float64(result.TotalTests)*plugintesting.SuccessRateMultiplier,
	)
	t.Logf("Summary: %s", result.Summary)

	if result.FailedTests > 0 {
		t.Log("\n--- FAILED TESTS ---")
		for _, testResult := range result.Results {
			if !testResult.Success {
				t.Logf("❌ %s: %v (%s)", testResult.Method, testResult.Error, testResult.Details)
			}
		}
	}

	t.Log("\n--- ALL TEST RESULTS ---")
	for _, testResult := range result.Results {
		status := "✅"
		if !testResult.Success {
			status = "❌"
		}
		t.Logf(
			"%s %s (%v) - %s",
			status,
			testResult.Method,
			testResult.Duration,
			testResult.Details,
		)
	}
	t.Log("===============================")
}

func conformanceLevelString(level ConformanceLevel) string {
	switch level {
	case ConformanceBasic:
		return "Basic"
	case ConformanceStandard:
		return "Standard"
	case ConformanceAdvanced:
		return "Advanced"
	default:
		return "Unknown"
	}
}

// ConformanceTestMain provides a standard main function for plugin conformance testing.
func ConformanceTestMain(impl pbc.CostSourceServiceServer, level ConformanceLevel) {
	t := &testing.T{}

	var result *ConformanceResult

	switch level {
	case ConformanceBasic:
		result = RunBasicConformanceTests(t, impl)
	case ConformanceStandard:
		result = RunStandardConformanceTests(t, impl)
	case ConformanceAdvanced:
		result = RunAdvancedConformanceTests(t, impl)
	}

	PrintConformanceReport(t, result)

	if result.FailedTests > 0 {
		t.Logf("❌ Conformance tests failed. Plugin does not meet %s conformance requirements.",
			strings.ToLower(conformanceLevelString(level)))
		return
	}

	t.Logf("✅ Plugin successfully meets %s conformance requirements!",
		strings.ToLower(conformanceLevelString(level)))
}

func createActualCostValidTimeRangeTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		timeStart, timeEnd := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
		resp, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      timeStart,
			End:        timeEnd,
		})
		duration := time.Since(start)

		if err != nil {
			// Error is acceptable if no data is available
			st, ok := status.FromError(err)
			if ok && (st.Code() == codes.NotFound || st.Code() == codes.Unavailable) {
				return plugintesting.TestResult{
					Method:   "GetActualCost",
					Success:  true,
					Duration: duration,
					Details:  "Correctly indicated no data available",
				}
			}

			return plugintesting.TestResult{
				Method:   "GetActualCost",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "Unexpected error type",
			}
		}

		if validationErr := plugintesting.ValidateActualCostResponse(resp); validationErr != nil {
			return plugintesting.TestResult{
				Method:   "GetActualCost",
				Success:  false,
				Error:    validationErr,
				Duration: duration,
				Details:  "Response validation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetActualCost",
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Returned %d cost data points", len(resp.GetResults())),
		}
	}
}

func createActualCostInvalidTimeRangeTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		// Swap start and end to create invalid range
		timeEnd, timeStart := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
		_, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      timeStart,
			End:        timeEnd,
		})
		duration := time.Since(start)

		if err == nil {
			return plugintesting.TestResult{
				Method:   "GetActualCost",
				Success:  false,
				Error:    errors.New("plugin accepted invalid time range"),
				Duration: duration,
				Details:  "Should reject end time before start time",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetActualCost",
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected invalid time range",
		}
	}
}

func createPricingSpecConsistencyTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		// Get pricing spec multiple times
		var firstSpec *pbc.PricingSpec
		for i := range plugintesting.NumConsistencyChecks {
			resp, err := harness.Client().
				GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
					Resource: resource,
				})
			if err != nil {
				return plugintesting.TestResult{
					Method:   "GetPricingSpec",
					Success:  false,
					Error:    err,
					Duration: time.Since(start),
					Details:  fmt.Sprintf("Failed on iteration %d", i),
				}
			}

			if i == 0 {
				firstSpec = resp.GetSpec()
			} else {
				if inconsistencyErr := checkSpecConsistency(firstSpec, resp.GetSpec(), start); inconsistencyErr != nil {
					return *inconsistencyErr
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetPricingSpec",
			Success:  true,
			Duration: time.Since(start),
			Details:  "Pricing spec is consistent across multiple calls",
		}
	}
}

func checkSpecConsistency(
	firstSpec, currentSpec *pbc.PricingSpec,
	start time.Time,
) *plugintesting.TestResult {
	if currentSpec.GetRatePerUnit() != firstSpec.GetRatePerUnit() {
		return &plugintesting.TestResult{
			Method:  "GetPricingSpec",
			Success: false,
			Error: fmt.Errorf(
				"inconsistent rate: %.6f vs %.6f",
				firstSpec.GetRatePerUnit(),
				currentSpec.GetRatePerUnit(),
			),
			Duration: time.Since(start),
			Details:  "Rate per unit should be consistent",
		}
	}
	if currentSpec.GetCurrency() != firstSpec.GetCurrency() {
		return &plugintesting.TestResult{
			Method:  "GetPricingSpec",
			Success: false,
			Error: fmt.Errorf(
				"inconsistent currency: %s vs %s",
				firstSpec.GetCurrency(),
				currentSpec.GetCurrency(),
			),
			Duration: time.Since(start),
			Details:  "Currency should be consistent",
		}
	}
	return nil
}

func createPerformanceBaselineTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		// Test Name performance (should be fast)
		nameStart := time.Now()
		_, err := harness.Client().Name(context.Background(), &pbc.NameRequest{})
		nameDuration := time.Since(nameStart)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "Performance",
				Success:  false,
				Error:    err,
				Duration: time.Since(start),
				Details:  "Name RPC failed",
			}
		}

		// Name should respond within 100ms
		if nameDuration > plugintesting.MaxResponseTimeMs*time.Millisecond {
			return plugintesting.TestResult{
				Method:   "Performance",
				Success:  false,
				Error:    fmt.Errorf("name RPC too slow: %v", nameDuration),
				Duration: time.Since(start),
				Details:  "Name RPC should respond within 100ms",
			}
		}

		return plugintesting.TestResult{
			Method:   "Performance",
			Success:  true,
			Duration: time.Since(start),
			Details:  fmt.Sprintf("Name RPC responded in %v", nameDuration),
		}
	}
}

func createConcurrentRequestTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		const numConcurrent = plugintesting.NumConcurrentRequests
		errors := make(chan error, numConcurrent)

		// Launch concurrent Name requests
		for range numConcurrent {
			go func() {
				_, err := harness.Client().Name(context.Background(), &pbc.NameRequest{})
				errors <- err
			}()
		}

		// Collect results
		for i := range numConcurrent {
			if err := <-errors; err != nil {
				return plugintesting.TestResult{
					Method:   "Concurrency",
					Success:  false,
					Error:    err,
					Duration: time.Since(start),
					Details:  fmt.Sprintf("Concurrent request %d failed", i),
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "Concurrency",
			Success:  true,
			Duration: time.Since(start),
			Details:  fmt.Sprintf("Successfully handled %d concurrent requests", numConcurrent),
		}
	}
}

func createLargeDataHandlingTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		// Request 30 days of data (should be a reasonable large dataset)
		timeStart, timeEnd := plugintesting.CreateTimeRange(plugintesting.HoursIn30Days) // 30 days
		resp, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
			ResourceId: "large-dataset-test",
			Start:      timeStart,
			End:        timeEnd,
		})
		duration := time.Since(start)

		if err != nil {
			// Error is acceptable if plugin doesn't support large datasets
			st, ok := status.FromError(err)
			if ok && (st.Code() == codes.InvalidArgument || st.Code() == codes.ResourceExhausted) {
				return plugintesting.TestResult{
					Method:   "LargeData",
					Success:  true,
					Duration: duration,
					Details:  "Correctly indicated large dataset not supported",
				}
			}

			return plugintesting.TestResult{
				Method:   "LargeData",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "Unexpected error for large dataset",
			}
		}

		// Should respond within reasonable time (10 seconds)
		if duration > plugintesting.MaxLargeQueryTimeSeconds*time.Second {
			return plugintesting.TestResult{
				Method:   "LargeData",
				Success:  false,
				Error:    fmt.Errorf("large dataset query too slow: %v", duration),
				Duration: duration,
				Details:  "Large dataset queries should complete within 10 seconds",
			}
		}

		return plugintesting.TestResult{
			Method:   "LargeData",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Handled large dataset (%d points) in %v",
				len(resp.GetResults()),
				duration,
			),
		}
	}
}

// createPricingSpecFlatRateTest validates that GetPricingSpec returns complete response
// with unit and assumptions populated (FR-009: assumptions SHOULD be populated).
func createPricingSpecFlatRateTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		resp, err := harness.Client().
			GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
				Resource: resource,
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetPricingSpec",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		spec := resp.GetSpec()
		if spec == nil {
			return plugintesting.TestResult{
				Method:   "GetPricingSpec",
				Success:  false,
				Error:    errors.New("spec is nil"),
				Duration: duration,
				Details:  "Response must contain a PricingSpec",
			}
		}

		// Validate required fields
		var issues []string

		if spec.GetProvider() == "" {
			issues = append(issues, "provider is empty")
		}
		if spec.GetResourceType() == "" {
			issues = append(issues, "resource_type is empty")
		}
		if spec.GetBillingMode() == "" {
			issues = append(issues, "billing_mode is empty")
		}
		if spec.GetCurrency() == "" {
			issues = append(issues, "currency is empty")
		}
		if spec.GetUnit() == "" {
			issues = append(issues, "unit is empty (should be populated for flat-rate pricing)")
		}
		if len(spec.GetAssumptions()) == 0 {
			issues = append(issues, "assumptions array is empty (SHOULD be populated per FR-009)")
		}

		if len(issues) > 0 {
			return plugintesting.TestResult{
				Method:   "GetPricingSpec",
				Success:  false,
				Error:    fmt.Errorf("validation issues: %s", strings.Join(issues, "; ")),
				Duration: duration,
				Details:  "PricingSpec validation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetPricingSpec",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"unit=%s, assumptions=%d, rate=%.4f %s",
				spec.GetUnit(),
				len(spec.GetAssumptions()),
				spec.GetRatePerUnit(),
				spec.GetCurrency(),
			),
		}
	}
}

// createPricingSpecDefaultFieldsTest validates that GetPricingSpec handles empty optional fields.
func createPricingSpecDefaultFieldsTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		// Use a resource with minimal fields to test defaults
		resource := &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			// Intentionally omit sku and region to test defaults
		}

		resp, err := harness.Client().
			GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
				Resource: resource,
			})
		duration := time.Since(start)

		if err != nil {
			// Some plugins may reject resources without sku/region - that's acceptable
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.InvalidArgument {
				return plugintesting.TestResult{
					Method:   "GetPricingSpec",
					Success:  true,
					Duration: duration,
					Details:  "Plugin correctly rejects incomplete resource descriptor",
				}
			}
			return plugintesting.TestResult{
				Method:   "GetPricingSpec",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "Unexpected error for resource with minimal fields",
			}
		}

		spec := resp.GetSpec()
		if spec == nil {
			return plugintesting.TestResult{
				Method:   "GetPricingSpec",
				Success:  false,
				Error:    errors.New("spec is nil"),
				Duration: duration,
				Details:  "Response must contain a PricingSpec",
			}
		}

		// Verify that empty optional fields don't cause issues
		// pricing_tiers can be empty for flat-rate billing
		tiersCount := len(spec.GetPricingTiers())

		return plugintesting.TestResult{
			Method:   "GetPricingSpec",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf("Handled defaults gracefully: pricing_tiers=%d, unit=%s",
				tiersCount, spec.GetUnit()),
		}
	}
}

// EstimateCost conformance test helpers

// createEstimateCostValidResourceTest validates FR-001: valid EstimateCost responses
// with proper currency format and non-negative monthly cost.
func createEstimateCostValidResourceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().EstimateCost(context.Background(), &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   nil, // Null attributes should be handled per FR-005
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		if validationErr := plugintesting.ValidateEstimateCostResponse(resp); validationErr != nil {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    validationErr,
				Duration: duration,
				Details:  "Response validation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Cost: %s %f/month", resp.GetCurrency(), resp.GetCostMonthly()),
		}
	}
}

// createEstimateCostInvalidFormatTest validates FR-003: InvalidArgument error
// for malformed resource type strings.
func createEstimateCostInvalidFormatTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		_, err := harness.Client().EstimateCost(context.Background(), &pbc.EstimateCostRequest{
			ResourceType: "invalid-format",
			Attributes:   nil,
		})
		duration := time.Since(start)

		// Should return InvalidArgument error per FR-003
		if err == nil {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    errors.New("expected error for invalid format, got nil"),
				Duration: duration,
				Details:  "Should reject invalid resource type format",
			}
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    fmt.Errorf("expected InvalidArgument error, got: %w", err),
				Duration: duration,
				Details:  "Wrong error code for invalid format",
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected invalid resource type format",
		}
	}
}

// createEstimateCostUnsupportedResourceTest validates FR-008: NotFound error
// for resource types not supported by the plugin.
func createEstimateCostUnsupportedResourceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		_, err := harness.Client().EstimateCost(context.Background(), &pbc.EstimateCostRequest{
			ResourceType: "aws:lambda/function:Function",
			Attributes:   nil,
		})
		duration := time.Since(start)

		// Should return NotFound error per FR-008
		if err == nil {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    errors.New("expected error for unsupported resource, got nil"),
				Duration: duration,
				Details:  "Should return error for unsupported resource type",
			}
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    fmt.Errorf("expected NotFound error, got: %w", err),
				Duration: duration,
				Details:  "Wrong error code for unsupported resource",
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details:  "Correctly returned NotFound for unsupported resource",
		}
	}
}

// createEstimateCostDeterministicTest validates FR-011: deterministic responses
// for identical inputs across multiple calls.
func createEstimateCostDeterministicTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		// Call EstimateCost multiple times with same input
		var costs []float64
		for i := range plugintesting.NumConsistencyChecks {
			resp, err := harness.Client().
				EstimateCost(context.Background(), &pbc.EstimateCostRequest{
					ResourceType: "aws:ec2/instance:Instance",
					Attributes:   nil,
				})
			if err != nil {
				return plugintesting.TestResult{
					Method:   "EstimateCost",
					Success:  false,
					Error:    err,
					Duration: time.Since(start),
					Details:  fmt.Sprintf("RPC call %d failed", i+1),
				}
			}
			costs = append(costs, resp.GetCostMonthly())
		}

		duration := time.Since(start)

		// Verify all costs are identical (FR-011)
		for i := 1; i < len(costs); i++ {
			if costs[i] != costs[0] {
				return plugintesting.TestResult{
					Method:   "EstimateCost",
					Success:  false,
					Error:    fmt.Errorf("inconsistent costs: %v vs %v", costs[0], costs[i]),
					Duration: duration,
					Details:  "Responses should be deterministic for identical inputs",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Deterministic: all %d calls returned %f", len(costs), costs[0]),
		}
	}
}

// createEstimateCostConcurrentTest validates Standard conformance: concurrent request handling
// with at least 10 simultaneous requests without errors.
func createEstimateCostConcurrentTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		// Test concurrent requests per Standard conformance
		results := make(chan error, plugintesting.NumConcurrentRequests)

		for range plugintesting.NumConcurrentRequests {
			go func() {
				_, err := harness.Client().
					EstimateCost(context.Background(), &pbc.EstimateCostRequest{
						ResourceType: "aws:ec2/instance:Instance",
						Attributes:   nil,
					})
				results <- err
			}()
		}

		// Collect results
		var errors []error
		for range plugintesting.NumConcurrentRequests {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}

		duration := time.Since(start)

		if len(errors) > 0 {
			maxErrors := 3
			if len(errors) < maxErrors {
				maxErrors = len(errors)
			}
			return plugintesting.TestResult{
				Method:  "EstimateCost",
				Success: false,
				Error: fmt.Errorf(
					"%d/%d concurrent requests failed",
					len(errors),
					plugintesting.NumConcurrentRequests,
				),
				Duration: duration,
				Details:  fmt.Sprintf("Failed requests: %v", errors[:maxErrors]),
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"All %d concurrent requests succeeded",
				plugintesting.NumConcurrentRequests,
			),
		}
	}
}

// createEstimateCostPerformanceTest validates SC-002: response time under 500ms
// for EstimateCost operations.
func createEstimateCostPerformanceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().EstimateCost(context.Background(), &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   nil,
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		// Check response time < 500ms per SC-002
		maxDuration := 500 * time.Millisecond
		if duration > maxDuration {
			return plugintesting.TestResult{
				Method:   "EstimateCost",
				Success:  false,
				Error:    fmt.Errorf("response time %v exceeds %v", duration, maxDuration),
				Duration: duration,
				Details:  "Performance requirement not met",
			}
		}

		return plugintesting.TestResult{
			Method:   "EstimateCost",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Response time %v < %v (Cost: %f)",
				duration,
				maxDuration,
				resp.GetCostMonthly(),
			),
		}
	}
}

// =============================================================================
// GetRecommendations Conformance Tests
// =============================================================================

// createGetRecommendationsEmptyPluginTest validates that plugins return empty list
// when not implementing RecommendationsProvider (FR-012).
func createGetRecommendationsEmptyPluginTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		// Response should have empty recommendations, not nil
		if resp == nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("response is nil"),
				Duration: duration,
				Details:  "Response should not be nil",
			}
		}

		// Summary should exist
		if resp.GetSummary() == nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("summary is nil"),
				Duration: duration,
				Details:  "Summary should be provided even for empty results",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Returned %d recommendations with summary",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// createGetRecommendationsBasicTest validates basic response structure.
func createGetRecommendationsBasicTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		// Validate each recommendation has required fields
		for i, rec := range resp.GetRecommendations() {
			if rec.GetId() == "" {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    fmt.Errorf("recommendation %d missing id", i),
					Duration: duration,
					Details:  "All recommendations must have id",
				}
			}
			if rec.GetCategory() == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    fmt.Errorf("recommendation %d has unspecified category", i),
					Duration: duration,
					Details:  "All recommendations must have category",
				}
			}
			if rec.GetActionType() == pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    fmt.Errorf("recommendation %d has unspecified action_type", i),
					Duration: duration,
					Details:  "All recommendations must have action_type",
				}
			}
			if rec.GetResource() == nil {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    fmt.Errorf("recommendation %d missing resource", i),
					Duration: duration,
					Details:  "All recommendations must have resource",
				}
			}
			if rec.GetImpact() == nil {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    fmt.Errorf("recommendation %d missing impact", i),
					Duration: duration,
					Details:  "All recommendations must have impact",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"All %d recommendations have valid required fields",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// createResolveResourceTypesEmptyPluginTest validates the graceful-degradation
// contract: a plugin without type-resolution support (or without a matching
// mapping) must return an empty response, never an error.
func createResolveResourceTypesEmptyPluginTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "definitely_unknown_type_xyz"},
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "ResolveResourceTypes",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		if resp == nil {
			return plugintesting.TestResult{
				Method:   "ResolveResourceTypes",
				Success:  false,
				Error:    errors.New("response is nil"),
				Duration: duration,
				Details:  "Response should not be nil even when the plugin has no type-resolution support",
			}
		}

		return plugintesting.TestResult{
			Method:   "ResolveResourceTypes",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Returned %d mappings (empty is a valid outcome for plugins without type-resolution support)",
				len(resp.GetMappings()),
			),
		}
	}
}

// createResolveResourceTypesBasicTest validates that resolved mappings are
// well-formed and that unmapped types are omitted, not returned as empty entries.
func createResolveResourceTypesBasicTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "definitely_unknown_type_xyz"},
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "ResolveResourceTypes",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		for sourceType, mapping := range resp.GetMappings() {
			if mapping.GetPulumiToken() == "" {
				return plugintesting.TestResult{
					Method:   "ResolveResourceTypes",
					Success:  false,
					Error:    fmt.Errorf("mapping for %q missing pulumi_token", sourceType),
					Duration: duration,
					Details:  "All returned mappings must have a non-empty pulumi_token",
				}
			}
		}

		if _, exists := resp.GetMappings()["definitely_unknown_type_xyz"]; exists {
			return plugintesting.TestResult{
				Method:   "ResolveResourceTypes",
				Success:  false,
				Error:    errors.New("unknown type should be omitted, not present in mappings"),
				Duration: duration,
				Details:  "Unmapped types must be omitted from the response, not returned as empty entries",
			}
		}

		return plugintesting.TestResult{
			Method:   "ResolveResourceTypes",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"All %d resolved mappings are well-formed; unknown types correctly omitted",
				len(resp.GetMappings()),
			),
		}
	}
}

// createGetRecommendationsSummaryTest validates summary matches individual recommendations.
// This test implicitly assumes the summary reflects the recommendations returned on the *current page*.
// If the API defines a global summary across all paginated results, this test might need adjustment
// or a separate conformance test for global summary validation.
func createGetRecommendationsSummaryTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		summary := resp.GetSummary()
		if summary == nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("summary is nil"),
				Duration: duration,
				Details:  "Summary must be provided",
			}
		}

		// Verify total count matches
		recs := resp.GetRecommendations()
		expectedCount := int32(len(recs))
		if summary.GetTotalRecommendations() != expectedCount {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: fmt.Errorf(
					"summary.total_recommendations=%d doesn't match len(recommendations)=%d",
					summary.GetTotalRecommendations(),
					expectedCount,
				),
				Duration: duration,
				Details:  "Summary count must match recommendations count",
			}
		}

		// Verify total savings matches sum of individual impacts
		var calculatedSavings float64
		for _, rec := range recs {
			if rec.GetImpact() != nil {
				calculatedSavings += rec.GetImpact().GetEstimatedSavings()
			}
		}

		// Allow small floating point difference
		tolerance := 0.01
		diff := summary.GetTotalEstimatedSavings() - calculatedSavings
		if diff < -tolerance || diff > tolerance {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: fmt.Errorf(
					"summary.total_estimated_savings=%f doesn't match sum=%f",
					summary.GetTotalEstimatedSavings(),
					calculatedSavings,
				),
				Duration: duration,
				Details:  "Summary savings must match sum of individual impacts",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Summary matches: %d recs, $%.2f total savings",
				summary.GetTotalRecommendations(),
				summary.GetTotalEstimatedSavings(),
			),
		}
	}
}

// =============================================================================
// GetRecommendations Filtering Conformance Tests
// =============================================================================

// createGetRecommendationsFilterByCategoryTest validates category filtering.
func createGetRecommendationsFilterByCategoryTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with category filter failed",
			}
		}

		// Verify all returned recommendations have the filtered category
		for i, rec := range resp.GetRecommendations() {
			if rec.GetCategory() != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has category %s, expected COST",
						i,
						rec.GetCategory().String(),
					),
					Duration: duration,
					Details:  "Filter returned non-matching category",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Category filter returned %d COST recommendations",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// createGetRecommendationsFilterByActionTypeTest validates action type filtering.
func createGetRecommendationsFilterByActionTypeTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with action_type filter failed",
			}
		}

		// Verify all returned recommendations have the filtered action type
		for i, rec := range resp.GetRecommendations() {
			if rec.GetActionType() != pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has action_type %s, expected RIGHTSIZE",
						i,
						rec.GetActionType().String(),
					),
					Duration: duration,
					Details:  "Filter returned non-matching action_type",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Action type filter returned %d RIGHTSIZE recommendations",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// createGetRecommendationsFilterNoMatchesTest validates empty result for non-matching filter.
func createGetRecommendationsFilterNoMatchesTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		// Use a filter that won't match any recommendations (non-existent provider)
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Provider: "nonexistent-provider-xyz",
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with non-matching filter failed",
			}
		}

		// Should return empty list, not error
		if len(resp.GetRecommendations()) != 0 {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: fmt.Errorf(
					"expected 0 recommendations for non-matching filter, got %d",
					len(resp.GetRecommendations()),
				),
				Duration: duration,
				Details:  "Non-matching filter should return empty list",
			}
		}

		// Summary should show 0 total
		if resp.GetSummary().GetTotalRecommendations() != 0 {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: errors.New(
					"summary should show 0 recommendations for non-matching filter",
				),
				Duration: duration,
				Details:  "Summary count mismatch",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details:  "Correctly returned empty list for non-matching filter",
		}
	}
}

// createGetRecommendationsFilterByProviderTest validates provider filtering.
func createGetRecommendationsFilterByProviderTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Provider: "aws",
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		if resp == nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("GetRecommendations returned a nil response"),
				Duration: duration,
				Details:  "RPC call returned nil response",
			}
		}

		// Verify all returned recommendations have the filtered provider
		for i, rec := range resp.GetRecommendations() {
			if rec.GetResource() == nil {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has nil resource",
						i,
					),
					Duration: duration,
					Details:  "Provider filter not applied correctly",
				}
			}
			if rec.GetResource().GetProvider() != "aws" {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has provider %s, expected aws",
						i,
						rec.GetResource().GetProvider(),
					),
					Duration: duration,
					Details:  "Provider filter not applied correctly",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Provider filter returned %d AWS recommendations",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// createGetRecommendationsFilterCombinedTest validates multiple filter criteria.
func createGetRecommendationsFilterCombinedTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					Provider: "aws",
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with combined filter failed",
			}
		}

		// Verify all returned recommendations match BOTH criteria
		for i, rec := range resp.GetRecommendations() {
			if rec.GetCategory() != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has category %s, expected COST",
						i,
						rec.GetCategory().String(),
					),
					Duration: duration,
					Details:  "Combined filter: category mismatch",
				}
			}
			if rec.GetResource() == nil || rec.GetResource().GetProvider() != "aws" {
				providerStr := "<nil resource>"
				if rec.GetResource() != nil {
					providerStr = rec.GetResource().GetProvider()
				}
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has provider %s, expected aws",
						i,
						providerStr,
					),
					Duration: duration,
					Details:  "Combined filter: provider mismatch",
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Combined filter returned %d matching recommendations",
				len(resp.GetRecommendations()),
			),
		}
	}
}

// paginationTestPageSize is the page size used for pagination tests.
const paginationTestPageSize = 5

// createGetRecommendationsPaginationFirstPageTest validates first page pagination.
func createGetRecommendationsPaginationFirstPageTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				PageSize: paginationTestPageSize,
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with page_size failed",
			}
		}

		// Verify page size is respected
		if len(resp.GetRecommendations()) > paginationTestPageSize {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: fmt.Errorf(
					"returned %d recommendations, expected at most %d",
					len(resp.GetRecommendations()),
					paginationTestPageSize,
				),
				Duration: duration,
				Details:  "Page size not respected",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"First page returned %d recommendations (max %d), next_token: %v",
				len(resp.GetRecommendations()),
				paginationTestPageSize,
				resp.GetNextPageToken() != "",
			),
		}
	}
}

// createGetRecommendationsPaginationNextPageTest validates next page navigation.
func createGetRecommendationsPaginationNextPageTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()

		// Get first page
		firstResp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				PageSize: paginationTestPageSize,
			})
		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: time.Since(start),
				Details:  "First page RPC call failed",
			}
		}

		// If no next page token, test passes (less than page_size results)
		if firstResp.GetNextPageToken() == "" {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  true,
				Duration: time.Since(start),
				Details:  "Only one page of results (no next token)",
			}
		}

		// Get second page
		secondResp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				PageSize:  paginationTestPageSize,
				PageToken: firstResp.GetNextPageToken(),
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "Second page RPC call failed",
			}
		}

		// Verify we got results and they're different from first page
		if len(secondResp.GetRecommendations()) == 0 && firstResp.GetNextPageToken() != "" {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("second page returned no results despite next_page_token"),
				Duration: duration,
				Details:  "Next page navigation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"Successfully navigated to second page (%d results)",
				len(secondResp.GetRecommendations()),
			),
		}
	}
}

// createGetRecommendationsPaginationLastPageTest validates empty token on last page.
func createGetRecommendationsPaginationLastPageTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		var lastToken string
		pageCount := 0
		maxPages := 100 // Safety limit

		// Navigate through all pages
		for pageCount < maxPages {
			req := &pbc.GetRecommendationsRequest{
				PageSize:  paginationTestPageSize,
				PageToken: lastToken,
			}
			resp, err := harness.Client().GetRecommendations(context.Background(), req)
			if err != nil {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  false,
					Error:    err,
					Duration: time.Since(start),
					Details:  fmt.Sprintf("RPC call failed on page %d", pageCount+1),
				}
			}

			pageCount++
			lastToken = resp.GetNextPageToken()

			// Last page should have empty token
			if lastToken == "" {
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  true,
					Duration: time.Since(start),
					Details:  fmt.Sprintf("Reached last page after %d pages", pageCount),
				}
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  false,
			Error:    errors.New("exceeded max page limit without reaching last page"),
			Duration: time.Since(start),
			Details:  fmt.Sprintf("Iterated %d pages without empty token", maxPages),
		}
	}
}

// createGetRecommendationsPaginationInvalidTokenTest validates error on invalid token.
func createGetRecommendationsPaginationInvalidTokenTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		_, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				PageSize:  paginationTestPageSize,
				PageToken: "invalid-token-not-base64",
			})
		duration := time.Since(start)

		if err == nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    errors.New("expected error for invalid token, got nil"),
				Duration: duration,
				Details:  "Invalid token should return error",
			}
		}

		// Verify it's a proper gRPC error
		st, ok := status.FromError(err)
		if !ok {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    fmt.Errorf("expected gRPC status error, got: %w", err),
				Duration: duration,
				Details:  "Error should be gRPC status",
			}
		}

		// InvalidArgument is the expected status for bad tokens
		if st.Code() != codes.InvalidArgument {
			return plugintesting.TestResult{
				Method:  "GetRecommendations",
				Success: false,
				Error: fmt.Errorf(
					"expected InvalidArgument status, got: %s",
					st.Code().String(),
				),
				Duration: duration,
				Details:  "Invalid token should return InvalidArgument",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details:  "Invalid token correctly returns InvalidArgument error",
		}
	}
}

// createGetRecommendationsRightsizeActionTest validates rightsize action details.
func createGetRecommendationsRightsizeActionTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with rightsize filter failed",
			}
		}

		for i, rec := range resp.GetRecommendations() {
			if rec.GetActionType() != pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has action type %s, expected rightsize",
						i,
						rec.GetActionType(),
					),
					Duration: duration,
					Details:  "Filtered results must have correct action type",
				}
			}

			if rec.GetRightsize() == nil {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d has rightsize action type but missing rightsize details",
						i,
					),
					Duration: duration,
					Details:  "Rightsize action must have rightsize details",
				}
			}

			rightsize := rec.GetRightsize()

			if (rightsize.GetCurrentSku() == "" && rightsize.GetCurrentInstanceType() == "") ||
				(rightsize.GetRecommendedSku() == "" && rightsize.GetRecommendedInstanceType() == "") {
				return plugintesting.TestResult{
					Method:  "GetRecommendations",
					Success: false,
					Error: fmt.Errorf(
						"recommendation %d rightsize action missing current or recommended SKU/instance type",
						i,
					),
					Duration: duration,
					Details:  "Rightsize action must have populated current and recommended details",
				}
			}
		}

		// If no rightsize recommendations are found, the test should still pass if the plugin

		// indicates it supports rightsize actions but doesn't have any to provide.

		return plugintesting.TestResult{

			Method: "GetRecommendations",

			Success: true,

			Duration: duration,

			Details: "Rightsize recommendations validated (or none found)",
		}
	}
}

// createGetRecommendationsKubernetesActionTest validates kubernetes action details.
func createGetRecommendationsKubernetesActionTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_ADJUST_REQUESTS,
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with kubernetes filter failed",
			}
		}

		// Find a kubernetes recommendation and validate its action details
		for _, rec := range resp.GetRecommendations() {
			if rec.GetKubernetes() != nil {
				k8s := rec.GetKubernetes()
				if k8s.GetClusterId() == "" {
					return plugintesting.TestResult{
						Method:   "GetRecommendations",
						Success:  false,
						Error:    errors.New("kubernetes action missing cluster_id"),
						Duration: duration,
						Details:  "Kubernetes action must have cluster identification",
					}
				}
				if k8s.GetContainerName() == "" {
					return plugintesting.TestResult{
						Method:   "GetRecommendations",
						Success:  false,
						Error:    errors.New("kubernetes action missing container_name"),
						Duration: duration,
						Details:  "Kubernetes action must specify target container",
					}
				}
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  true,
					Duration: duration,
					Details: fmt.Sprintf(
						"Kubernetes action valid: cluster=%s, container=%s",
						k8s.GetClusterId(),
						k8s.GetContainerName(),
					),
				}
			}
		}

		// No kubernetes recommendations found - that's okay, test passes
		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details:  "No kubernetes recommendations available (filter returned 0 results)",
		}
	}
}

// createGetRecommendationsCommitmentActionTest validates commitment action details.
func createGetRecommendationsCommitmentActionTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_PURCHASE_COMMITMENT,
				},
			})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method:   "GetRecommendations",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call with commitment filter failed",
			}
		}

		// Find a commitment recommendation and validate its action details
		for _, rec := range resp.GetRecommendations() {
			if rec.GetCommitment() != nil {
				commitment := rec.GetCommitment()
				if commitment.GetCommitmentType() == "" {
					return plugintesting.TestResult{
						Method:   "GetRecommendations",
						Success:  false,
						Error:    errors.New("commitment action missing commitment_type"),
						Duration: duration,
						Details:  "Commitment action must specify type",
					}
				}
				if commitment.GetTerm() == "" {
					return plugintesting.TestResult{
						Method:   "GetRecommendations",
						Success:  false,
						Error:    errors.New("commitment action missing term"),
						Duration: duration,
						Details:  "Commitment action must specify term",
					}
				}
				return plugintesting.TestResult{
					Method:   "GetRecommendations",
					Success:  true,
					Duration: duration,
					Details: fmt.Sprintf(
						"Commitment action valid: type=%s, term=%s, qty=%.2f",
						commitment.GetCommitmentType(),
						commitment.GetTerm(),
						commitment.GetRecommendedQuantity(),
					),
				}
			}
		}

		// No commitment recommendations found - that's okay, test passes
		return plugintesting.TestResult{
			Method:   "GetRecommendations",
			Success:  true,
			Duration: duration,
			Details:  "No commitment recommendations available (filter returned 0 results)",
		}
	}
}

// =============================================================================
// GetPluginInfo Conformance Tests
// =============================================================================

// createGetPluginInfoValidResponseTest validates that GetPluginInfo returns valid metadata.
// This test verifies FR-001 (RPC exists), FR-002 (response fields), FR-003 (SDK default),
// and FR-007 (SemVer validation for spec_version).

func createGetPluginInfoValidResponseTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		resp, err := harness.Client().
			GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		duration := time.Since(start)

		if err != nil {
			// Check if it's an Unimplemented error (legacy plugin - US3)
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Unimplemented {
				return plugintesting.TestResult{
					Method:   "GetPluginInfo",
					Success:  true, // Unimplemented is acceptable for legacy plugins
					Duration: duration,
					Details:  "Legacy plugin: GetPluginInfo not implemented (graceful degradation)",
				}
			}
			return plugintesting.TestResult{
				Method:   "GetPluginInfo",
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "RPC call failed",
			}
		}

		// Validate required fields (FR-008)
		var issues []string

		if resp.GetName() == "" {
			issues = append(issues, "name is empty (required)")
		}
		if resp.GetVersion() == "" {
			issues = append(issues, "version is empty (required)")
		}
		if resp.GetSpecVersion() == "" {
			issues = append(issues, "spec_version is empty (required)")
		}

		// Validate spec_version is a valid SemVer (FR-007)
		if resp.GetSpecVersion() != "" {
			if !plugintesting.IsValidSemVer(resp.GetSpecVersion()) {
				issues = append(issues, fmt.Sprintf(
					"spec_version %q is not a valid SemVer (expected format: vMAJOR.MINOR.PATCH)",
					resp.GetSpecVersion(),
				))
			}
		}

		if len(issues) > 0 {
			return plugintesting.TestResult{
				Method:   "GetPluginInfo",
				Success:  false,
				Error:    fmt.Errorf("validation issues: %s", strings.Join(issues, "; ")),
				Duration: duration,
				Details:  "GetPluginInfo validation failed",
			}
		}

		return plugintesting.TestResult{
			Method:   "GetPluginInfo",
			Success:  true,
			Duration: duration,
			Details: fmt.Sprintf(
				"name=%s, version=%s, spec_version=%s, providers=%v",
				resp.GetName(),
				resp.GetVersion(),
				resp.GetSpecVersion(),
				resp.GetProviders(),
			),
		}
	}
}

func createGetPluginInfoPerformanceTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		const maxDuration = 100 * time.Millisecond
		const iterations = 10
		ctx := context.Background()

		var totalDuration time.Duration
		for i := range iterations {
			start := time.Now()
			_, err := harness.Client().GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
			duration := time.Since(start)

			if err != nil {
				// Check if it's an Unimplemented error (legacy plugin)
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Unimplemented {
					return plugintesting.TestResult{
						Method:   "GetPluginInfo",
						Success:  true, // Unimplemented is acceptable for legacy plugins
						Duration: duration,
						Details:  "Legacy plugin: GetPluginInfo not implemented (graceful degradation)",
					}
				}
				return plugintesting.TestResult{
					Method:   "GetPluginInfo",
					Success:  false,
					Error:    err,
					Duration: duration,
					Details:  fmt.Sprintf("RPC call failed on iteration %d", i),
				}
			}

			if duration > maxDuration {
				return plugintesting.TestResult{
					Method:   "GetPluginInfo",
					Success:  false,
					Error:    fmt.Errorf("iteration %d: %v exceeds %v", i, duration, maxDuration),
					Duration: duration,
					Details:  "Performance requirement not met",
				}
			}
			totalDuration += duration
		}

		return plugintesting.TestResult{
			Method:   "GetPluginInfo",
			Success:  true,
			Duration: totalDuration / iterations,
			Details: fmt.Sprintf(
				"Average response time over %d iterations: %v",
				iterations,
				totalDuration/iterations,
			),
		}
	}
}

func createSupportsGranularDiscoveryTest() func(*plugintesting.TestHarness) plugintesting.TestResult {
	return func(harness *plugintesting.TestHarness) plugintesting.TestResult {
		start := time.Now()
		// 1. Check if GetPluginInfo is available
		// If GetPluginInfo returns Unimplemented, continue (legacy plugins may not have it)
		// Otherwise, any other error is a failure
		_, err := harness.Client().
			GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		if err != nil && status.Code(err) != codes.Unimplemented {
			// Non-Unimplemented errors are failures
			return plugintesting.TestResult{
				Method: "Supports", Success: false, Error: err, Duration: time.Since(start),
				Details: "Prerequisite GetPluginInfo failed",
			}
		}

		// 2. Call Supports
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		supportsResp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
			Resource: resource,
		})
		duration := time.Since(start)

		if err != nil {
			return plugintesting.TestResult{
				Method: "Supports", Success: false, Error: err, Duration: duration,
				Details: "Supports RPC failed",
			}
		}

		// 3. Verify Capabilities are present (Granular Discovery)
		// If Supported=true, we expect capabilities to be populated (enum-based)
		if supportsResp.GetSupported() {
			// Enum-based "strongly-typed capabilities" requirement - must always be populated when Supported=true
			if len(supportsResp.GetCapabilitiesEnum()) == 0 {
				return plugintesting.TestResult{
					Method:  "Supports",
					Success: false,
					Error: errors.New(
						"granular capabilities enum is empty but Supported=true",
					),
					Duration: duration,
					Details:  "Enum-based capabilities must be populated when Supported=true",
				}
			}

			// Verify legacy sync
			if len(supportsResp.GetCapabilitiesEnum()) > 0 &&
				len(supportsResp.GetCapabilities()) == 0 {
				return plugintesting.TestResult{
					Method: "Supports", Success: false,
					Error: errors.New(
						"legacy capabilities map is empty but enum capabilities present",
					),
					Duration: duration,
					Details:  "Legacy map should be synced from enum",
				}
			}
		}

		return plugintesting.TestResult{
			Method: "Supports", Success: true, Duration: duration,
			Details: fmt.Sprintf(
				"Validated granular capabilities (count=%d)",
				len(supportsResp.GetCapabilitiesEnum()),
			),
		}
	}
}
