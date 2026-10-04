// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements RPC correctness validation for the Plugin Conformance Test Suite.
package testing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// testNameRPC tests the Name RPC method.
func testNameRPC(harness *TestHarness) TestResult {
	start := time.Now()
	resp, err := harness.Client().Name(context.Background(), &pbc.NameRequest{})
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Method:   MethodName,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodName + " RPC failed",
		}
	}

	if valErr := ValidateNameResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodName,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodName,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Plugin name: %s", resp.GetName()),
	}
}

// testSupportsRPC tests the Supports RPC method with valid input.
func testSupportsRPC(harness *TestHarness) TestResult {
	start := time.Now()
	resource := harness.SampleResource()
	resp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
		Resource: resource,
	})
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Method:   MethodSupports,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodSupports + " RPC failed",
		}
	}

	if valErr := ValidateSupportsResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodSupports,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodSupports,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Supported: %v", resp.GetSupported()),
	}
}

// testGetActualCostRPC tests the GetActualCost RPC method.
func testGetActualCostRPC(harness *TestHarness) TestResult {
	start := time.Now()
	timeStart, timeEnd := CreateTimeRange(HoursPerDay)
	resp, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		ResourceId: testResourceID,
		Start:      timeStart,
		End:        timeEnd,
		Resource:   harness.SampleResource(),
	})
	duration := time.Since(start)

	if err != nil {
		// Some errors are acceptable (e.g., no data available)
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.NotFound || st.Code() == codes.Unavailable) {
			return TestResult{
				Method:   MethodGetActualCost,
				Category: CategoryRPCCorrectness,
				Success:  true,
				Duration: duration,
				Details:  detailsNoDataAvailable,
			}
		}

		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetActualCost + " RPC failed",
		}
	}

	if valErr := ValidateActualCostResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodGetActualCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Returned %d cost data points", len(resp.GetResults())),
	}
}

// detailsNoDataAvailable is the result detail for a plugin that reports no data.
const detailsNoDataAvailable = "Correctly indicated no data available"

// conformanceTeamTagKey and conformanceTeamTag are the tag the descriptor conformance tests send.
const (
	conformanceTeamTagKey = "team"
	conformanceTeamTag    = "platform"
)

// testBillingAccountID is the id the billing account conformance test sends on GetActualCost.
const testBillingAccountID = "conformance-billing-account"

// testGetActualCostBillingAccountRPC checks that FOCUS records echo the request's billing_account_id.
func testGetActualCostBillingAccountRPC(harness *TestHarness) TestResult {
	start := time.Now()
	timeStart, timeEnd := CreateTimeRange(HoursPerDay)
	req := &pbc.GetActualCostRequest{
		ResourceId:       testResourceID,
		Start:            timeStart,
		End:              timeEnd,
		BillingAccountId: testBillingAccountID,
		Resource:         harness.SampleResource(),
	}
	resp, err := harness.Client().GetActualCost(context.Background(), req)
	duration := time.Since(start)

	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.NotFound || st.Code() == codes.Unavailable) {
			return TestResult{
				Method:   MethodGetActualCost,
				Category: CategoryRPCCorrectness,
				Success:  true,
				Duration: duration,
				Details:  detailsNoDataAvailable,
			}
		}

		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetActualCost + " RPC failed",
		}
	}

	if valErr := ValidateActualCostBillingAccount(req, resp); valErr != nil {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  "FOCUS record does not echo the request billing_account_id",
		}
	}

	return TestResult{
		Method:   MethodGetActualCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Checked %d cost data points", len(resp.GetResults())),
	}
}

// testGetActualCostWithResourceRPC checks that GetActualCost accepts a request carrying a
// ResourceDescriptor with tags and nested attributes. It never checks whether the plugin
// used the descriptor, so a plugin that ignores it passes.
func testGetActualCostWithResourceRPC(harness *TestHarness) TestResult {
	attrs, err := conformanceAttributes()
	if err != nil {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Details:  "failed to build attributes fixture",
		}
	}
	resource := harness.SampleResource()
	if resource.Tags == nil {
		resource.Tags = map[string]string{}
	}
	resource.Tags[conformanceTeamTagKey] = conformanceTeamTag
	resource.Attributes = attrs

	start := time.Now()
	timeStart, timeEnd := CreateTimeRange(HoursPerDay)
	resp, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		ResourceId: testResourceID,
		Start:      timeStart,
		End:        timeEnd,
		Resource:   resource,
	})
	duration := time.Since(start)

	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.NotFound || st.Code() == codes.Unavailable) {
			return TestResult{
				Method:   MethodGetActualCost,
				Category: CategoryRPCCorrectness,
				Success:  true,
				Duration: duration,
				Details:  detailsNoDataAvailable,
			}
		}

		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetActualCost + " RPC failed with a resource descriptor",
		}
	}

	if valErr := ValidateActualCostResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  "Response validation failed",
		}
	}

	return TestResult{
		Method:   MethodGetActualCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Accepted a resource descriptor; returned %d cost data points", len(resp.GetResults())),
	}
}

// testGetProjectedCostRPC tests the GetProjectedCost RPC method.
func testGetProjectedCostRPC(harness *TestHarness) TestResult {
	start := time.Now()
	resource := harness.SampleResource()
	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: resource,
	})
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Method:   MethodGetProjectedCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetProjectedCost + " RPC failed",
		}
	}

	if valErr := ValidateProjectedCostResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetProjectedCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodGetProjectedCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Unit price: %.6f %s", resp.GetUnitPrice(), resp.GetCurrency()),
	}
}

// conformanceAttributes is a nested attributes value ten segments deep, the depth
// of a Kubernetes CronJob container's CPU request. Plugins are not expected to
// read it; the test proves a descriptor carrying it is accepted.
func conformanceAttributes() (*structpb.Struct, error) {
	const spec = "spec"
	return structpb.NewStruct(map[string]any{
		spec: map[string]any{"jobTemplate": map[string]any{spec: map[string]any{
			"template": map[string]any{spec: map[string]any{"containers": []any{
				map[string]any{"resources": map[string]any{"requests": map[string]any{"cpu": "250m"}}},
			}}},
		}}},
		"tags": map[string]any{conformanceTeamTagKey: conformanceTeamTag},
	})
}

// testGetProjectedCostWithAttributesRPC tests that GetProjectedCost accepts a
// ResourceDescriptor carrying nested attributes alongside tags. A plugin that
// ignores attributes passes.
func testGetProjectedCostWithAttributesRPC(harness *TestHarness) TestResult {
	attrs, err := conformanceAttributes()
	if err != nil {
		return TestResult{
			Method:   MethodGetProjectedCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Details:  "failed to build attributes fixture",
		}
	}
	resource := harness.SampleResource()
	if resource.Tags == nil {
		resource.Tags = map[string]string{}
	}
	resource.Tags[conformanceTeamTagKey] = conformanceTeamTag
	resource.Attributes = attrs

	start := time.Now()
	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: resource,
	})
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Method:   MethodGetProjectedCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetProjectedCost + " RPC failed for a descriptor with attributes",
		}
	}

	if valErr := ValidateProjectedCostResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetProjectedCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodGetProjectedCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  "Descriptor with nested attributes accepted",
	}
}

// testGetPricingSpecRPC tests the GetPricingSpec RPC method.
func testGetPricingSpecRPC(harness *TestHarness) TestResult {
	start := time.Now()
	resource := harness.SampleResource()
	resp, err := harness.Client().GetPricingSpec(context.Background(), &pbc.GetPricingSpecRequest{
		Resource: resource,
	})
	duration := time.Since(start)

	if err != nil {
		return TestResult{
			Method:   MethodGetPricingSpec,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetPricingSpec + " RPC failed",
		}
	}

	if valErr := ValidatePricingSpecResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetPricingSpec,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  errResponseValidationFailed,
		}
	}

	return TestResult{
		Method:   MethodGetPricingSpec,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Billing mode: %s", resp.GetSpec().GetBillingMode()),
	}
}

// testGetBudgetsRPC tests the GetBudgets RPC method.
// This tests the optional RPC - if not implemented, Unimplemented error is expected.
func testGetBudgetsRPC(harness *TestHarness) TestResult {
	start := time.Now()
	resp, err := harness.Client().GetBudgets(context.Background(), &pbc.GetBudgetsRequest{
		Filter:        &pbc.BudgetFilter{},
		IncludeStatus: false,
	})
	duration := time.Since(start)

	if err != nil {
		// Check if it's the expected Unimplemented error for plugins that don't support budgets
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unimplemented {
			return TestResult{
				Method:   MethodGetBudgets,
				Category: CategoryRPCCorrectness,
				Success:  true,
				Duration: duration,
				Details:  "Plugin correctly returns Unimplemented for unsupported GetBudgets RPC",
			}
		}
		// Unexpected error
		return TestResult{
			Method:   MethodGetBudgets,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    err,
			Duration: duration,
			Details:  MethodGetBudgets + " RPC failed with unexpected error",
		}
	}

	// Plugin supports budgets - validate response
	if valErr := ValidateBudgetsResponse(resp); valErr != nil {
		return TestResult{
			Method:   MethodGetBudgets,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    valErr,
			Duration: duration,
			Details:  "Response validation failed",
		}
	}

	return TestResult{
		Method:   MethodGetBudgets,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  fmt.Sprintf("Returned %d budgets", len(resp.GetBudgets())),
	}
}

// testCrossProviderBudgetMapping is implemented as an integration test
// due to the need for custom mock plugin configuration.
// See TestCrossProviderBudgetMapping in integration_test.go

// testNilResourceHandling tests that the plugin handles nil resources gracefully.
func testNilResourceHandling(harness *TestHarness) TestResult {
	start := time.Now()
	resp, err := harness.Client().Supports(context.Background(), &pbc.SupportsRequest{
		Resource: nil,
	})
	duration := time.Since(start)

	if err != nil {
		// Error is expected for nil resource
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.InvalidArgument {
			return TestResult{
				Method:   MethodSupports,
				Category: CategoryRPCCorrectness,
				Success:  true,
				Duration: duration,
				Details:  "Correctly rejected nil resource with InvalidArgument",
			}
		}
		// Any error is acceptable for nil resource
		return TestResult{
			Method:   MethodSupports,
			Category: CategoryRPCCorrectness,
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected nil resource with error",
		}
	}

	// If no error, it should at least indicate not supported
	if resp.GetSupported() {
		return TestResult{
			Method:   "Supports",
			Category: CategoryRPCCorrectness,
			Success:  false,
			Duration: duration,
			Details:  "Plugin returned Supported: true for nil resource",
		}
	}

	return TestResult{
		Method:   "Supports",
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  "Handled nil resource without error (returned Supported: false)",
	}
}

// testInvalidTimeRangeHandling tests that the plugin handles invalid time ranges.
func testInvalidTimeRangeHandling(harness *TestHarness) TestResult {
	startTest := time.Now()
	// Create invalid time range (end before start)
	start, end := CreateTimeRange(HoursPerDay)
	_, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      end,   // Swap start/end to create invalid range
		End:        start, // Swap start/end to create invalid range
		Resource:   harness.SampleResource(),
	})
	duration := time.Since(startTest)

	if err == nil {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  false,
			Error:    errors.New("plugin accepted invalid time range"),
			Duration: duration,
			Details:  "Should reject end time before start time",
		}
	}

	st, ok := status.FromError(err)
	if ok && st.Code() == codes.InvalidArgument {
		return TestResult{
			Method:   MethodGetActualCost,
			Category: CategoryRPCCorrectness,
			Success:  true,
			Duration: duration,
			Details:  "Correctly rejected invalid time range with InvalidArgument",
		}
	}

	return TestResult{
		Method:   MethodGetActualCost,
		Category: CategoryRPCCorrectness,
		Success:  true,
		Duration: duration,
		Details:  "Correctly rejected invalid time range",
	}
}

// RPCCorrectnessTests returns the RPC correctness conformance tests.
func RPCCorrectnessTests() []ConformanceSuiteTest {
	return []ConformanceSuiteTest{
		{
			Name:        "RPCCorrectness_NameRPC",
			Description: "Validates Name RPC returns valid response",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createNameRPCTest(),
		},
		{
			Name:        "RPCCorrectness_SupportsRPC",
			Description: "Validates Supports RPC handles valid input",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createSupportsRPCTest(),
		},
		{
			Name:        "RPCCorrectness_NilResource",
			Description: "Validates plugin handles nil resource correctly",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createNilResourceTest(),
		},
		{
			Name:        "RPCCorrectness_InvalidTimeRange",
			Description: "Validates GetActualCost rejects an end before start for the sample resource (sent as resource)",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createInvalidTimeRangeTest(),
		},
		{
			Name:        "RPCCorrectness_GetActualCostRPC",
			Description: "Validates GetActualCost for the sample resource (sent as resource); requests without one are not checked",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetActualCostRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetActualCostBillingAccount",
			Description: "Validates FOCUS records echo billing_account_id for the sample resource (sent as resource)",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetActualCostBillingAccountRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetActualCostWithResource",
			Description: "Validates GetActualCost accepts a resource descriptor with nested attributes",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createGetActualCostWithResourceRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetProjectedCostRPC",
			Description: "Validates GetProjectedCost RPC returns valid response",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createGetProjectedCostRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetProjectedCostWithAttributes",
			Description: "Validates GetProjectedCost accepts a descriptor with nested attributes",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createGetProjectedCostWithAttributesRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetPricingSpecRPC",
			Description: "Validates GetPricingSpec RPC returns valid response",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createGetPricingSpecRPCTest(),
		},
		{
			Name:        "RPCCorrectness_GetBudgetsRPC",
			Description: "Validates GetBudgets RPC returns valid response or Unimplemented",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createGetBudgetsRPCTest(),
		},
	}
}

func createNameRPCTest() func(*TestHarness) TestResult {
	return testNameRPC
}

func createSupportsRPCTest() func(*TestHarness) TestResult {
	return testSupportsRPC
}

func createNilResourceTest() func(*TestHarness) TestResult {
	return testNilResourceHandling
}

func createInvalidTimeRangeTest() func(*TestHarness) TestResult {
	return testInvalidTimeRangeHandling
}

func createGetActualCostRPCTest() func(*TestHarness) TestResult {
	return testGetActualCostRPC
}

func createGetActualCostBillingAccountRPCTest() func(*TestHarness) TestResult {
	return testGetActualCostBillingAccountRPC
}

func createGetActualCostWithResourceRPCTest() func(*TestHarness) TestResult {
	return testGetActualCostWithResourceRPC
}

func createGetProjectedCostRPCTest() func(*TestHarness) TestResult {
	return testGetProjectedCostRPC
}

func createGetProjectedCostWithAttributesRPCTest() func(*TestHarness) TestResult {
	return testGetProjectedCostWithAttributesRPC
}

func createGetPricingSpecRPCTest() func(*TestHarness) TestResult {
	return testGetPricingSpecRPC
}

func createGetBudgetsRPCTest() func(*TestHarness) TestResult {
	return testGetBudgetsRPC
}

// RegisterRPCCorrectnessTests registers RPC correctness tests with a conformance suite.
func RegisterRPCCorrectnessTests(suite *ConformanceSuite) {
	for _, test := range RPCCorrectnessTests() {
		suite.AddTest(test)
	}
}

// RunRPCCorrectness runs RPC correctness tests against a plugin.
func RunRPCCorrectness(impl pbc.CostSourceServiceServer) ([]TestResult, error) {
	harness := NewTestHarness(impl)

	// Create connection manually
	conn, err := harness.createClientConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	// Set the client on the harness
	harness.client = pbc.NewCostSourceServiceClient(conn)

	var results []TestResult
	for _, test := range RPCCorrectnessTests() {
		result := test.TestFunc(harness)
		results = append(results, result)
	}

	harness.Stop()
	return results, nil
}
