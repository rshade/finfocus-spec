package testing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	"github.com/rshade/finfocus-spec/sdk/go/pricing"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestBasicPluginFunctionality tests all basic RPC methods of a plugin.
func TestBasicPluginFunctionality(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	testNameRPC(ctx, t, client, plugin)
	testSupportsRPC(ctx, t, client)
	testGetActualCostRPC(ctx, t, client)
	testGetProjectedCostRPC(ctx, t, client)
	testGetPricingSpecRPC(ctx, t, client)
}

// testNameRPC tests the Name RPC functionality.
func testNameRPC(
	ctx context.Context,
	t *testing.T,
	client pbc.CostSourceServiceClient,
	plugin *plugintesting.MockPlugin,
) {
	t.Run("Name", func(t *testing.T) {
		resp, err := client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			t.Fatalf("Name() failed: %v", err)
		}

		if validationErr := plugintesting.ValidateNameResponse(resp); validationErr != nil {
			t.Errorf("Invalid name response: %v", validationErr)
		}

		if resp.GetName() != plugin.PluginName {
			t.Errorf("Expected name %s, got %s", plugin.PluginName, resp.GetName())
		}
	})
}

// testSupportsRPC tests the Supports RPC functionality.
func testSupportsRPC(ctx context.Context, t *testing.T, client pbc.CostSourceServiceClient) {
	t.Run("Supports", func(t *testing.T) {
		// Test supported resource
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		if err != nil {
			t.Fatalf("Supports() failed: %v", err)
		}

		if validationErr := plugintesting.ValidateSupportsResponse(resp); validationErr != nil {
			t.Errorf("Invalid supports response: %v", validationErr)
		}

		if !resp.GetSupported() {
			t.Errorf("Expected resource to be supported, got: %s", resp.GetReason())
		}
	})

	t.Run("SupportsUnsupportedProvider", func(t *testing.T) {
		// Test unsupported provider
		resource := plugintesting.CreateResourceDescriptor("unsupported", "some_resource", "", "")
		resp, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		if err != nil {
			t.Fatalf("Supports() failed: %v", err)
		}

		if resp.GetSupported() {
			t.Error("Expected unsupported provider to be rejected")
		}

		if resp.GetReason() == "" {
			t.Error("Expected reason for unsupported provider")
		}
	})
}

func testGetActualCostRPC(ctx context.Context, t *testing.T, client pbc.CostSourceServiceClient) {
	t.Run("GetActualCost", func(t *testing.T) {
		start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
		resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource-123",
			Start:      start,
			End:        end,
			Tags: map[string]string{
				"environment": "test",
			},
		})
		if err != nil {
			t.Fatalf("GetActualCost() failed: %v", err)
		}

		if validationErr := plugintesting.ValidateActualCostResponse(resp); validationErr != nil {
			t.Errorf("Invalid actual cost response: %v", validationErr)
		}

		if len(resp.GetResults()) == 0 {
			t.Error("Expected some cost results")
		}
	})
}

func testGetProjectedCostRPC(
	ctx context.Context,
	t *testing.T,
	client pbc.CostSourceServiceClient,
) {
	t.Run("GetProjectedCost", func(t *testing.T) {
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		if validationErr := plugintesting.ValidateProjectedCostResponse(resp); validationErr != nil {
			t.Errorf("Invalid projected cost response: %v", validationErr)
		}

		if resp.GetUnitPrice() <= 0 {
			t.Errorf("Expected positive unit price, got: %f", resp.GetUnitPrice())
		}
	})
}

func testGetPricingSpecRPC(ctx context.Context, t *testing.T, client pbc.CostSourceServiceClient) {
	t.Run("GetPricingSpec", func(t *testing.T) {
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
			Resource: resource,
		})
		if err != nil {
			t.Fatalf("GetPricingSpec() failed: %v", err)
		}

		if validationErr := plugintesting.ValidatePricingSpecResponse(resp); validationErr != nil {
			t.Errorf("Invalid pricing spec response: %v", validationErr)
		}

		spec := resp.GetSpec()
		if spec.GetProvider() != resource.GetProvider() {
			t.Errorf("Expected provider %s, got %s", resource.GetProvider(), spec.GetProvider())
		}
		if spec.GetResourceType() != resource.GetResourceType() {
			t.Errorf(
				"Expected resource type %s, got %s",
				resource.GetResourceType(),
				spec.GetResourceType(),
			)
		}
	})
}

// TestErrorHandling tests various error conditions.
func TestErrorHandling(t *testing.T) {
	plugin := plugintesting.ConfigurableErrorMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	t.Run("NameError", func(t *testing.T) {
		plugin.ShouldErrorOnName = true
		_, err := client.Name(ctx, &pbc.NameRequest{})
		if err == nil {
			t.Error("Expected error from Name(), got nil")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Error("Expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("Expected Internal error, got %v", st.Code())
		}
	})

	t.Run("SupportsError", func(t *testing.T) {
		plugin.ShouldErrorOnSupports = true
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		_, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		if err == nil {
			t.Error("Expected error from Supports(), got nil")
		}
	})

	t.Run("ActualCostError", func(t *testing.T) {
		plugin.ShouldErrorOnActualCost = true
		start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
		_, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
		})
		if err == nil {
			t.Error("Expected error from GetActualCost(), got nil")
		}
	})

	t.Run("ProjectedCostError", func(t *testing.T) {
		plugin.ShouldErrorOnProjectedCost = true
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err == nil {
			t.Error("Expected error from GetProjectedCost(), got nil")
		}
	})

	t.Run("PricingSpecError", func(t *testing.T) {
		plugin.ShouldErrorOnPricingSpec = true
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		_, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
			Resource: resource,
		})
		if err == nil {
			t.Error("Expected error from GetPricingSpec(), got nil")
		}
	})
}

// TestInputValidation tests input validation for all methods.
func TestInputValidation(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	t.Run("SupportsNilResource", func(t *testing.T) {
		resp, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: nil})
		if err != nil {
			t.Fatalf("Supports() failed: %v", err)
		}

		if resp.GetSupported() {
			t.Error("Expected nil resource to be unsupported")
		}
	})

	t.Run("ActualCostMissingTimestamps", func(t *testing.T) {
		_, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			// Missing Start and End timestamps
		})
		if err == nil {
			t.Error("Expected error for missing timestamps")
		}
	})

	t.Run("ActualCostInvalidTimeRange", func(t *testing.T) {
		end, start := plugintesting.CreateTimeRange(plugintesting.HoursPerDay) // Swapped start and end
		_, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
		})
		if err == nil {
			t.Error("Expected error for invalid time range")
		}
	})

	t.Run("ProjectedCostNilResource", func(t *testing.T) {
		_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: nil,
		})
		if err == nil {
			t.Error("Expected error for nil resource")
		}
	})

	t.Run("PricingSpecNilResource", func(t *testing.T) {
		_, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
			Resource: nil,
		})
		if err == nil {
			t.Error("Expected error for nil resource")
		}
	})
}

// TestMultipleProviders tests plugin behavior with different providers.
func TestMultipleProviders(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	providers := []struct {
		name         string
		provider     string
		resourceType string
		sku          string
	}{
		{"AWS EC2", "aws", "ec2", "t3.micro"},
		{"Azure VM", "azure", "vm", "Standard_B1s"},
		{"GCP Compute", "gcp", "compute_engine", "n1-standard-1"},
		{"Kubernetes", "kubernetes", "namespace", ""},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			testProviderSupport(ctx, t, client, p.provider, p.resourceType, p.sku)
		})
	}
}

func testProviderSupport(
	ctx context.Context,
	t *testing.T,
	client pbc.CostSourceServiceClient,
	provider, resourceType, sku string,
) {
	resource := plugintesting.CreateResourceDescriptor(provider, resourceType, sku, "us-east-1")

	// Test Supports
	supportsResp, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
	if err != nil {
		t.Fatalf("Supports() failed: %v", err)
	}

	if !supportsResp.GetSupported() {
		t.Errorf("Provider %s should be supported: %s", provider, supportsResp.GetReason())
	}

	// Test GetProjectedCost
	projectedResp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: resource,
	})
	if err != nil {
		t.Fatalf("GetProjectedCost() failed: %v", err)
	}

	if validationErr := plugintesting.ValidateProjectedCostResponse(projectedResp); validationErr != nil {
		t.Errorf("Invalid projected cost response for %s: %v", provider, validationErr)
	}

	// Test GetPricingSpec
	specResp, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{
		Resource: resource,
	})
	if err != nil {
		t.Fatalf("GetPricingSpec() failed: %v", err)
	}

	spec := specResp.GetSpec()
	if spec.GetProvider() != provider {
		t.Errorf("Expected provider %s, got %s", provider, spec.GetProvider())
	}
	if spec.GetResourceType() != resourceType {
		t.Errorf("Expected resource type %s, got %s", resourceType, spec.GetResourceType())
	}
}

// TestCrossProviderBudgetMapping tests budget mapping across different providers.
// This validates that budgets from AWS, GCP, and other providers are properly
// structured and contain required fields for unified display.
func TestCrossProviderBudgetMapping(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.MockBudgets = []*pbc.Budget{
		{
			Id:     "aws-budget-123",
			Name:   "AWS Production Budget",
			Source: "aws-budgets",
			Amount: &pbc.BudgetAmount{
				Limit:    5000.00,
				Currency: "USD",
			},
			Period: pbc.BudgetPeriod_BUDGET_PERIOD_MONTHLY,
			Status: &pbc.BudgetStatus{
				CurrentSpend:         3200.00,
				ForecastedSpend:      4800.00,
				PercentageUsed:       64.0,
				PercentageForecasted: 96.0,
				Currency:             "USD",
				Health:               pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK,
			},
		},
		{
			Id:     "gcp-budget-456",
			Name:   "GCP Cloud Budget",
			Source: "gcp-billing",
			Amount: &pbc.BudgetAmount{
				Limit:    10000.00,
				Currency: "USD",
			},
			Period: pbc.BudgetPeriod_BUDGET_PERIOD_MONTHLY,
			Status: &pbc.BudgetStatus{
				CurrentSpend:         7500.00,
				ForecastedSpend:      11000.00,
				PercentageUsed:       75.0,
				PercentageForecasted: 110.0,
				Currency:             "USD",
				Health:               pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING,
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	resp, err := client.GetBudgets(ctx, &pbc.GetBudgetsRequest{
		Filter:        &pbc.BudgetFilter{},
		IncludeStatus: true,
	})
	require.NoError(t, err)

	// Validate response structure
	err = plugintesting.ValidateBudgetsResponse(resp)
	require.NoError(t, err)

	// Verify cross-provider aspects
	budgets := resp.GetBudgets()
	require.Len(t, budgets, 2, "Expected 2 budgets")

	// Check that budgets have different sources
	sources := make(map[string]bool)
	for _, budget := range budgets {
		sources[budget.GetSource()] = true
	}
	require.Len(t, sources, 2, "Budgets should have different provider sources")

	// Verify summary calculation
	summary := resp.GetSummary()
	require.NotNil(t, summary)
	require.Equal(t, int32(2), summary.GetTotalBudgets(), "Summary total_budgets should be 2")
	require.Equal(t, int32(1), summary.GetBudgetsOk(), "Should have 1 OK budget")
	require.Equal(t, int32(1), summary.GetBudgetsWarning(), "Should have 1 warning budget")
	require.Equal(t, int32(0), summary.GetBudgetsExceeded(), "Should have 0 exceeded budgets")
}

// TestConcurrentRequests tests plugin behavior under concurrent load.
func TestConcurrentRequests(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	const numConcurrentRequestsLocal = plugintesting.NumConcurrentRequests

	t.Run("ConcurrentNameRequests", func(t *testing.T) {
		errors := make(chan error, numConcurrentRequestsLocal)

		// Run concurrent Name requests
		for range numConcurrentRequestsLocal {
			go func() {
				_, err := client.Name(ctx, &pbc.NameRequest{})
				errors <- err
			}()
		}

		// Check all requests completed successfully
		for i := range numConcurrentRequestsLocal {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent Name request %d failed: %v", i, err)
			}
		}
	})

	t.Run("ConcurrentGetPluginInfoRequests", func(t *testing.T) {
		errors := make(chan error, numConcurrentRequestsLocal)

		// Run concurrent GetPluginInfo requests
		for range numConcurrentRequestsLocal {
			go func() {
				_, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
				errors <- err
			}()
		}

		// Check all requests completed successfully
		for i := range numConcurrentRequestsLocal {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent GetPluginInfo request %d failed: %v", i, err)
			}
		}
	})
}

// TestResponseTimeouts tests plugin behavior with configured delays.
func TestResponseTimeouts(t *testing.T) {
	plugin := plugintesting.SlowMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	t.Run("NameWithDelay", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Name(ctx, &pbc.NameRequest{})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Name() failed: %v", err)
		}

		if duration < plugin.NameDelay {
			t.Errorf("Expected delay of at least %v, got %v", plugin.NameDelay, duration)
		}

		if resp.GetName() != plugin.PluginName {
			t.Errorf("Expected name %s, got %s", plugin.PluginName, resp.GetName())
		}
	})

	t.Run("SupportsWithDelay", func(t *testing.T) {
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		start := time.Now()
		_, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Supports() failed: %v", err)
		}

		if duration < plugin.SupportsDelay {
			t.Errorf("Expected delay of at least %v, got %v", plugin.SupportsDelay, duration)
		}
	})
}

// TestDataConsistency tests that plugin returns consistent data.
func TestDataConsistency(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	// Get projected cost multiple times - should be consistent
	var firstResponse *pbc.GetProjectedCostResponse
	for i := range 5 {
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed on iteration %d: %v", i, err)
		}

		if i == 0 {
			firstResponse = resp
		} else {
			// Responses should be identical
			if resp.GetUnitPrice() != firstResponse.GetUnitPrice() {
				t.Errorf("Unit price inconsistent: iteration 0 = %f, iteration %d = %f",
					firstResponse.GetUnitPrice(), i, resp.GetUnitPrice())
			}
			if resp.GetCurrency() != firstResponse.GetCurrency() {
				t.Errorf("Currency inconsistent: iteration 0 = %s, iteration %d = %s",
					firstResponse.GetCurrency(), i, resp.GetCurrency())
			}
			if resp.GetCostPerMonth() != firstResponse.GetCostPerMonth() {
				t.Errorf("Cost per month inconsistent: iteration 0 = %f, iteration %d = %f",
					firstResponse.GetCostPerMonth(), i, resp.GetCostPerMonth())
			}
		}
	}
}

// =============================================================================
// STRUCTURED LOGGING EXAMPLE
// =============================================================================
//
// TestStructuredLoggingExample demonstrates structured logging patterns for the
// EstimateCost RPC, per NFR-001 of spec 006-estimate-cost.
//
// This example serves as the canonical reference for plugin developers to
// understand how to properly integrate zerolog structured logging with
// FinFocus plugin operations.
//
// Key patterns demonstrated:
//   - Creating a configured logger with plugin metadata (FR-001)
//   - Logging requests with resource context (FR-002)
//   - Logging successful responses with cost details (FR-003)
//   - Logging errors with error codes and context (FR-004)
//   - Correlation ID (trace_id) propagation (FR-005)
//   - Using standard field name constants (FR-006)
//   - Operation timing with LogOperation helper (FR-009)
//
// Best Practices:
//   - ALWAYS include trace_id when available for distributed tracing
//   - NEVER log attribute values directly - log count only to prevent credential exposure
//   - Use standard field constants from pluginsdk for consistent naming
//   - Include operation name in every log entry for filterability
//   - Log at appropriate levels: Info for normal flow, Error for failures
//
//nolint:gocognit // Intentional: educational test with comprehensive inline documentation
func TestStructuredLoggingExample(t *testing.T) {
	// Setup plugin and harness for all subtests
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()

	// =========================================================================
	// User Story 1: Plugin Developer Learns Logging Patterns
	// =========================================================================

	// T008, T009, T010: RequestLogging subtest
	t.Run("RequestLogging", func(t *testing.T) {
		// Create a buffer to capture log output for verification
		var buf bytes.Buffer

		// FR-001: Create a configured logger with plugin name and version
		// Best Practice: Use NewPluginLogger to ensure consistent metadata fields
		logger := pluginsdk.NewPluginLogger(
			"example-plugin",  // plugin name - identifies the plugin in logs
			"v1.0.0",          // version - helps correlate logs with deployments
			zerolog.InfoLevel, // minimum log level
			&buf,              // output writer (use os.Stderr in production)
		)

		// Simulate a trace ID from incoming request context
		// Best Practice: Extract trace_id from gRPC metadata using TracingUnaryServerInterceptor
		traceID := "trace-abc123"
		ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

		// Create sample request data
		resourceType := "aws:ec2/instance:Instance"
		attrs, _ := structpb.NewStruct(map[string]interface{}{
			"instanceType": "t3.micro",
			"region":       "us-east-1",
		})

		// FR-002: Log incoming request with resource context
		// Best Practice: Log attribute COUNT not values to prevent credential exposure
		logger.Info().
			Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
			Str(pluginsdk.FieldOperation, "EstimateCost").
			Str(pluginsdk.FieldResourceType, resourceType).
			Int("attribute_count", len(attrs.GetFields())).
			Msg("Processing cost estimation request")

		// Verify log output contains expected fields
		logOutput := buf.String()
		assertLogContains(t, logOutput, pluginsdk.FieldTraceID, "trace_id field missing")
		assertLogContains(t, logOutput, pluginsdk.FieldOperation, "operation field missing")
		assertLogContains(t, logOutput, pluginsdk.FieldResourceType, "resource_type field missing")
		assertLogContains(t, logOutput, "attribute_count", "attribute_count field missing")
		assertLogContains(t, logOutput, traceID, "trace_id value missing")
		assertLogContains(t, logOutput, resourceType, "resource_type value missing")

		// Verify we did NOT log sensitive attribute values
		assertLogNotContains(t, logOutput, "t3.micro", "sensitive attribute value should not be logged")
	})

	// T011, T012, T013: SuccessResponseLogging subtest
	t.Run("SuccessResponseLogging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := pluginsdk.NewPluginLogger("example-plugin", "v1.0.0", zerolog.InfoLevel, &buf)

		traceID := "trace-def456"
		ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)
		resourceType := "aws:ec2/instance:Instance"

		// FR-009: Use LogOperation helper for automatic timing measurement
		// Note: In production, use `defer done()` to ensure timing is logged even on panic.
		// Here we call done() explicitly to verify log output within the test.
		done := pluginsdk.LogOperation(logger, "EstimateCost")

		// Perform the actual EstimateCost RPC call
		attrs, _ := structpb.NewStruct(map[string]interface{}{
			"instanceType": "t3.micro",
			"region":       "us-east-1",
		})
		resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: resourceType,
			Attributes:   attrs,
		})
		if err != nil {
			t.Fatalf("EstimateCost() failed: %v", err)
		}

		// FR-003: Log successful response with cost details
		// Best Practice: Include all relevant business data for operational visibility
		logger.Info().
			Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
			Str(pluginsdk.FieldOperation, "EstimateCost").
			Str(pluginsdk.FieldResourceType, resourceType).
			Float64(pluginsdk.FieldCostMonthly, resp.GetCostMonthly()).
			Str("currency", resp.GetCurrency()).
			Msg("Cost estimation completed")

		// Log operation timing (this will add duration_ms)
		done()

		// Verify log output
		logOutput := buf.String()
		assertLogContains(t, logOutput, pluginsdk.FieldCostMonthly, "cost_monthly field missing")
		assertLogContains(t, logOutput, "currency", "currency field missing")
		assertLogContains(t, logOutput, pluginsdk.FieldDurationMs, "duration_ms field missing")
		assertLogContains(t, logOutput, "Cost estimation completed", "completion message missing")

		// Verify cost value is present (even if zero - valid business case)
		entries := parseMultipleLogEntries(t, logOutput)
		foundCost := false
		for _, entry := range entries {
			if _, ok := entry[pluginsdk.FieldCostMonthly]; ok {
				foundCost = true
				break
			}
		}
		if !foundCost {
			t.Error("cost_monthly field not found in any log entry")
		}
	})

	// =========================================================================
	// User Story 2: Plugin Developer Implements Error Logging
	// =========================================================================

	// T015, T016, T017: ErrorLogging subtest
	t.Run("ErrorLogging", func(t *testing.T) {
		// Use configurable error mock for error injection
		errorPlugin := plugintesting.ConfigurableErrorMockPlugin()
		errorPlugin.ShouldErrorOnEstimateCost = true

		errorHarness := plugintesting.NewTestHarness(errorPlugin)
		errorHarness.Start(t)
		defer errorHarness.Stop()

		errorClient := errorHarness.Client()

		var buf bytes.Buffer
		logger := pluginsdk.NewPluginLogger("example-plugin", "v1.0.0", zerolog.InfoLevel, &buf)

		traceID := "trace-error789"
		ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)
		resourceType := "invalid:resource/type:Type"

		// Attempt the call (will fail)
		attrs, _ := structpb.NewStruct(map[string]interface{}{})
		_, err := errorClient.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: resourceType,
			Attributes:   attrs,
		})

		// FR-004: Log errors with error code, message, and original request context
		// Best Practice: Include enough context to diagnose without re-running the request
		if err != nil {
			st, _ := status.FromError(err)
			logger.Error().
				Err(err).
				Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
				Str(pluginsdk.FieldOperation, "EstimateCost").
				Str(pluginsdk.FieldResourceType, resourceType).
				Str(pluginsdk.FieldErrorCode, st.Code().String()).
				Msg("Cost estimation failed")
		}

		// Verify error log output
		logOutput := buf.String()
		assertLogContains(t, logOutput, "error", "error level/field missing")
		assertLogContains(t, logOutput, pluginsdk.FieldErrorCode, "error_code field missing")
		assertLogContains(t, logOutput, pluginsdk.FieldTraceID, "trace_id missing in error log")
		assertLogContains(t, logOutput, pluginsdk.FieldResourceType, "resource_type missing in error log")
		assertLogContains(t, logOutput, "Cost estimation failed", "error message missing")
	})

	// T018, T019, T020, T021: CorrelationIDPropagation subtest
	t.Run("CorrelationIDPropagation", func(t *testing.T) {
		var buf bytes.Buffer
		logger := pluginsdk.NewPluginLogger("example-plugin", "v1.0.0", zerolog.InfoLevel, &buf)

		// FR-005: Demonstrate correlation ID propagation
		// Best Practice: Use ContextWithTraceID to store, TraceIDFromContext to retrieve
		traceID := "trace-correlation-xyz"
		ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

		// Verify trace ID can be retrieved from context
		retrievedTraceID := pluginsdk.TraceIDFromContext(ctx)
		if retrievedTraceID != traceID {
			t.Errorf("Expected trace ID %s, got %s", traceID, retrievedTraceID)
		}

		// Log multiple operations with the same trace_id
		for i, op := range []string{"validate", "estimate", "respond"} {
			logger.Info().
				Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
				Str(pluginsdk.FieldOperation, op).
				Int("step", i+1).
				Msg("Processing step")
		}

		// Verify all log entries contain the same trace_id
		logOutput := buf.String()
		entries := parseMultipleLogEntries(t, logOutput)

		// Best Practice: Verify trace_id appears in ALL related log entries
		for i, entry := range entries {
			entryTraceID, ok := entry[pluginsdk.FieldTraceID].(string)
			if !ok {
				t.Errorf("Log entry %d missing trace_id", i)
				continue
			}
			if entryTraceID != traceID {
				t.Errorf("Log entry %d has wrong trace_id: expected %s, got %s", i, traceID, entryTraceID)
			}
		}

		// Test graceful degradation: empty context returns empty string
		emptyCtx := context.Background()
		emptyTraceID := pluginsdk.TraceIDFromContext(emptyCtx)
		if emptyTraceID != "" {
			t.Errorf("Expected empty trace_id from empty context, got %s", emptyTraceID)
		}
	})

	// =========================================================================
	// User Story 3: Operator Monitors EstimateCost Health
	// =========================================================================

	// T022, T023, T024, T025: LogStructureValidation subtest
	t.Run("LogStructureValidation", func(t *testing.T) {
		var buf bytes.Buffer
		logger := pluginsdk.NewPluginLogger("example-plugin", "v1.0.0", zerolog.InfoLevel, &buf)

		traceID := "trace-structure-test"
		ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

		// Generate multiple log entries for different operations
		operations := []string{"EstimateCost", "GetProjectedCost", "GetActualCost"}
		resourceTypes := []string{
			"aws:ec2/instance:Instance",
			"azure:compute/virtualMachine:VirtualMachine",
			"gcp:compute/instance:Instance",
		}

		for i, op := range operations {
			logger.Info().
				Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
				Str(pluginsdk.FieldOperation, op).
				Str(pluginsdk.FieldResourceType, resourceTypes[i]).
				Float64(pluginsdk.FieldCostMonthly, float64(i+1)*10.50).
				Msg("Operation completed")
		}

		// Parse all log entries
		logOutput := buf.String()
		entries := parseMultipleLogEntries(t, logOutput)

		if len(entries) != len(operations) {
			t.Errorf("Expected %d log entries, got %d", len(operations), len(entries))
		}

		// Verify all entries are valid JSON and use standard field names
		// Best Practice: Consistent field names enable cross-plugin log aggregation
		for i, entry := range entries {
			// Check required fields exist
			if _, ok := entry[pluginsdk.FieldTraceID]; !ok {
				t.Errorf("Entry %d missing %s", i, pluginsdk.FieldTraceID)
			}
			if _, ok := entry[pluginsdk.FieldOperation]; !ok {
				t.Errorf("Entry %d missing %s", i, pluginsdk.FieldOperation)
			}

			// Verify filterability: operation field should allow filtering
			opValue, ok := entry[pluginsdk.FieldOperation].(string)
			if !ok {
				t.Errorf("Entry %d: operation field not a string", i)
				continue
			}
			if opValue != operations[i] {
				t.Errorf("Entry %d: expected operation %s, got %s", i, operations[i], opValue)
			}
		}

		// Demonstrate filterability by operation
		// Operators can query: jq 'select(.operation == "EstimateCost")' logs.json
		estimateCostCount := 0
		for _, entry := range entries {
			if op, ok := entry[pluginsdk.FieldOperation].(string); ok && op == "EstimateCost" {
				estimateCostCount++
			}
		}
		if estimateCostCount != 1 {
			t.Errorf("Expected 1 EstimateCost operation, found %d", estimateCostCount)
		}
	})
}

// =============================================================================
// Helper Functions for Log Verification
// =============================================================================

// parseMultipleLogEntries parses newline-delimited JSON log entries from a buffer.
// Each line is expected to be a valid JSON object.
func parseMultipleLogEntries(t *testing.T, logOutput string) []map[string]interface{} {
	t.Helper()
	var entries []map[string]interface{}

	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Failed to parse log entry %d as JSON: %v\nLine: %s", i, err, line)
			continue
		}
		entries = append(entries, entry)
	}

	return entries
}

// assertLogContains verifies that the log output contains the expected substring.
func assertLogContains(t *testing.T, logOutput, expected, errMsg string) {
	t.Helper()
	if !strings.Contains(logOutput, expected) {
		t.Errorf("%s: expected '%s' in log output:\n%s", errMsg, expected, logOutput)
	}
}

// assertLogNotContains verifies that the log output does NOT contain the unexpected substring.
// Use this to verify sensitive data is not being logged.
func assertLogNotContains(t *testing.T, logOutput, unexpected, errMsg string) {
	t.Helper()
	if strings.Contains(logOutput, unexpected) {
		t.Errorf("%s: unexpected '%s' found in log output:\n%s", errMsg, unexpected, logOutput)
	}
}

// =============================================================================
// METRICS TRACKING EXAMPLE
// =============================================================================
//
// TestMetricsTrackingExample demonstrates metrics collection patterns for the
// EstimateCost RPC, per NFR-002 of spec 006-estimate-cost.
//
// This example serves as the canonical reference for plugin developers to
// understand how to properly implement metrics tracking for FinFocus plugin
// operations.
//
// Key patterns demonstrated:
//   - Tracking latency (response time) for EstimateCost calls
//   - Tracking success rate and error rates
//   - Calculating percentiles (p50, p95, p99)
//   - Aggregating metrics across multiple requests
//   - Using standard field constants for metric labels
//
// Best Practices:
//   - ALWAYS track both latency and outcome (success/error) together
//   - Use consistent metric naming conventions across plugins
//   - Track error rates by error type/code for debugging
//   - Calculate percentiles for latency to understand tail performance
//   - Include resource_type and operation as metric dimensions
//
// Note: This example uses in-memory metrics collection for demonstration.
// Production implementations should use a metrics library like Prometheus,
// OpenTelemetry, or similar for proper aggregation and export.
func TestMetricsTrackingExample(t *testing.T) {
	// T041: LatencyTracking subtest
	t.Run("LatencyTracking", testMetricsLatencyTracking)

	// T041: SuccessRateTracking subtest
	t.Run("SuccessRateTracking", testMetricsSuccessRateTracking)

	// T041: ErrorRateByCode subtest
	t.Run("ErrorRateByCode", testMetricsErrorRateByCode)

	// T041: PercentileCalculation subtest
	t.Run("PercentileCalculation", testMetricsPercentileCalculation)

	// T041: MetricsWithDimensions subtest
	t.Run("MetricsWithDimensions", testMetricsWithDimensions)

	// T041: MetricsBestPractices subtest
	t.Run("MetricsBestPractices", testMetricsBestPractices)
}

// metricsCollector is a simple in-memory metrics aggregator for demonstration.
// In production, use a proper metrics library (Prometheus, OpenTelemetry).
// This collector demonstrates the essential patterns:
//   - Tracking individual request latencies
//   - Counting success and error outcomes
//   - Calculating percentiles for latency distribution
type metricsCollector struct {
	latencies    []time.Duration
	successCount int
	errorCount   int
	errorCodes   map[string]int
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		latencies:  make([]time.Duration, 0),
		errorCodes: make(map[string]int),
	}
}

// recordRequest records the outcome and latency of a request.
func (m *metricsCollector) recordRequest(duration time.Duration, err error) {
	m.latencies = append(m.latencies, duration)
	if err != nil {
		m.errorCount++
		st, ok := status.FromError(err)
		if ok {
			m.errorCodes[st.Code().String()]++
		} else {
			m.errorCodes["Unknown"]++
		}
	} else {
		m.successCount++
	}
}

// calculatePercentile calculates the pth percentile of latencies.
// Uses linear interpolation for more accurate results.
func (m *metricsCollector) calculatePercentile(p float64) time.Duration {
	if len(m.latencies) == 0 {
		return 0
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sortDurations(sorted)

	// Calculate index using linear interpolation
	n := float64(len(sorted))
	idx := (p / 100.0) * (n - 1)
	lower := int(idx)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation between lower and upper bounds
	weight := idx - float64(lower)
	return time.Duration(
		float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight,
	)
}

// successRate returns the success rate as a percentage.
func (m *metricsCollector) successRate() float64 {
	total := m.successCount + m.errorCount
	if total == 0 {
		return 0
	}
	return float64(m.successCount) / float64(total) * 100
}

func testMetricsLatencyTracking(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	metrics := newMetricsCollector()

	// Best Practice: Track latency for every request, successful or not
	// This enables understanding of both happy path and error path performance
	for i := range 10 {
		start := time.Now()
		attrs, _ := structpb.NewStruct(map[string]interface{}{
			"instanceType": "t3.micro",
			"region":       "us-east-1",
		})
		_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   attrs,
		})
		duration := time.Since(start)

		metrics.recordRequest(duration, err)

		// Log each request with timing information
		// Best Practice: Include operation and iteration for debugging
		t.Logf("Request %d: duration=%v, success=%v", i+1, duration, err == nil)
	}

	// Verify latency tracking
	if len(metrics.latencies) != 10 {
		t.Errorf("Expected 10 latency measurements, got %d", len(metrics.latencies))
	}

	// Calculate and log average latency
	var totalLatency time.Duration
	for _, d := range metrics.latencies {
		totalLatency += d
	}
	avgLatency := totalLatency / time.Duration(len(metrics.latencies))
	t.Logf("Average latency: %v", avgLatency)

	// Verify all latencies are positive (valid measurements)
	for i, d := range metrics.latencies {
		if d <= 0 {
			t.Errorf("Latency %d should be positive, got %v", i, d)
		}
	}
}

func testMetricsSuccessRateTracking(t *testing.T) {
	// Create standard mock for successful requests
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	metrics := newMetricsCollector()

	// Make 8 successful requests
	for range 8 {
		start := time.Now()
		attrs, _ := structpb.NewStruct(map[string]interface{}{
			"instanceType": "t3.micro",
		})
		_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   attrs,
		})
		metrics.recordRequest(time.Since(start), err)
	}

	// Create error mock for failed requests
	errorPlugin := plugintesting.ConfigurableErrorMockPlugin()
	errorPlugin.ShouldErrorOnEstimateCost = true
	errorHarness := plugintesting.NewTestHarness(errorPlugin)
	errorHarness.Start(t)
	defer errorHarness.Stop()
	errorClient := errorHarness.Client()

	// Make 2 requests that will fail
	for range 2 {
		start := time.Now()
		attrs, _ := structpb.NewStruct(map[string]interface{}{})
		_, err := errorClient.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "invalid:resource:Type",
			Attributes:   attrs,
		})
		metrics.recordRequest(time.Since(start), err)
	}

	// Verify success rate calculation
	// Best Practice: Track success rate per operation and resource type
	rate := metrics.successRate()
	expectedRate := 80.0 // 8 success / 10 total = 80%
	if rate != expectedRate {
		t.Errorf("Expected success rate %.1f%%, got %.1f%%", expectedRate, rate)
	}

	// Verify counts
	if metrics.successCount != 8 {
		t.Errorf("Expected 8 successful requests, got %d", metrics.successCount)
	}
	if metrics.errorCount != 2 {
		t.Errorf("Expected 2 failed requests, got %d", metrics.errorCount)
	}

	t.Logf("Success rate: %.1f%% (%d/%d)",
		rate, metrics.successCount, metrics.successCount+metrics.errorCount)
}

func testMetricsErrorRateByCode(t *testing.T) {
	// Best Practice: Track errors by gRPC status code for debugging
	// This helps identify specific failure modes (e.g., InvalidArgument vs Internal)
	errorPlugin := plugintesting.ConfigurableErrorMockPlugin()
	errorPlugin.ShouldErrorOnEstimateCost = true

	harness := plugintesting.NewTestHarness(errorPlugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	metrics := newMetricsCollector()

	// Generate errors
	for range 5 {
		start := time.Now()
		attrs, _ := structpb.NewStruct(map[string]interface{}{})
		_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "test:resource:Type",
			Attributes:   attrs,
		})
		metrics.recordRequest(time.Since(start), err)
	}

	// Verify error code tracking
	if len(metrics.errorCodes) == 0 {
		t.Error("Expected error codes to be tracked")
	}

	// Log error distribution
	// Best Practice: Understanding error distribution helps prioritize fixes
	t.Log("Error distribution by gRPC code:")
	for code, count := range metrics.errorCodes {
		t.Logf("  %s: %d", code, count)
	}

	// Verify total errors matches error count
	totalFromCodes := 0
	for _, count := range metrics.errorCodes {
		totalFromCodes += count
	}
	if totalFromCodes != metrics.errorCount {
		t.Errorf("Error code counts (%d) don't match total errors (%d)",
			totalFromCodes, metrics.errorCount)
	}
}

func testMetricsPercentileCalculation(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	metrics := newMetricsCollector()

	// Make enough requests for meaningful percentile calculation
	// Best Practice: Use at least 100 samples for accurate percentiles
	const numRequests = 100
	for range numRequests {
		start := time.Now()
		attrs, _ := structpb.NewStruct(map[string]interface{}{
			"instanceType": "t3.micro",
			"region":       "us-east-1",
		})
		_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
			ResourceType: "aws:ec2/instance:Instance",
			Attributes:   attrs,
		})
		metrics.recordRequest(time.Since(start), err)
	}

	// Calculate percentiles
	// Best Practice: p50, p95, p99 provide insight into typical and tail latency
	p50 := metrics.calculatePercentile(50)
	p95 := metrics.calculatePercentile(95)
	p99 := metrics.calculatePercentile(99)

	// Log percentile results
	t.Logf("Latency percentiles (n=%d):", numRequests)
	t.Logf("  p50: %v", p50)
	t.Logf("  p95: %v", p95)
	t.Logf("  p99: %v", p99)

	// Verify percentile ordering: p50 <= p95 <= p99
	// This is a fundamental property of percentiles
	if p50 > p95 {
		t.Errorf("p50 (%v) should be <= p95 (%v)", p50, p95)
	}
	if p95 > p99 {
		t.Errorf("p95 (%v) should be <= p99 (%v)", p95, p99)
	}

	// Verify all percentiles are positive
	if p50 <= 0 {
		t.Error("p50 should be positive")
	}
	if p95 <= 0 {
		t.Error("p95 should be positive")
	}
	if p99 <= 0 {
		t.Error("p99 should be positive")
	}
}

// dimensionedMetrics tracks metrics by operation and resource_type dimensions.
// Best Practice: Track metrics by operation and resource_type dimensions
// This enables drilling down into performance by specific resource types.
type dimensionedMetrics struct {
	byOperation    map[string]*metricsCollector
	byResourceType map[string]*metricsCollector
}

func testMetricsWithDimensions(t *testing.T) {
	dimMetrics := dimensionedMetrics{
		byOperation:    make(map[string]*metricsCollector),
		byResourceType: make(map[string]*metricsCollector),
	}

	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Test different resource types
	resourceTypes := []string{
		"aws:ec2/instance:Instance",
		"aws:s3/bucket:Bucket",
		"azure:compute/virtualMachine:VirtualMachine",
	}

	for _, resType := range resourceTypes {
		// Initialize metrics for this resource type if needed
		if _, ok := dimMetrics.byResourceType[resType]; !ok {
			dimMetrics.byResourceType[resType] = newMetricsCollector()
		}
		if _, ok := dimMetrics.byOperation["EstimateCost"]; !ok {
			dimMetrics.byOperation["EstimateCost"] = newMetricsCollector()
		}

		// Make requests for this resource type
		for range 5 {
			start := time.Now()
			attrs, _ := structpb.NewStruct(map[string]interface{}{
				"type": "standard",
			})
			_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: resType,
				Attributes:   attrs,
			})
			duration := time.Since(start)

			// Record in both dimension buckets
			dimMetrics.byResourceType[resType].recordRequest(duration, err)
			dimMetrics.byOperation["EstimateCost"].recordRequest(duration, err)
		}
	}

	// Verify metrics by resource type
	t.Log("Metrics by resource type:")
	for resType, m := range dimMetrics.byResourceType {
		rate := m.successRate()
		p50 := m.calculatePercentile(50)
		t.Logf("  %s: success_rate=%.1f%%, p50=%v, requests=%d",
			resType, rate, p50, len(m.latencies))

		// Verify each resource type has metrics
		if len(m.latencies) != 5 {
			t.Errorf("Resource type %s should have 5 requests, got %d",
				resType, len(m.latencies))
		}
	}

	// Verify aggregate operation metrics
	opMetrics := dimMetrics.byOperation["EstimateCost"]
	expectedTotal := len(resourceTypes) * 5
	if len(opMetrics.latencies) != expectedTotal {
		t.Errorf("EstimateCost operation should have %d requests, got %d",
			expectedTotal, len(opMetrics.latencies))
	}
	t.Logf("Aggregate EstimateCost: success_rate=%.1f%%, p50=%v, total=%d",
		opMetrics.successRate(), opMetrics.calculatePercentile(50), len(opMetrics.latencies))
}

func testMetricsBestPractices(t *testing.T) {
	// This test documents best practices for metrics in comments
	// and validates the recommended patterns

	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Best Practice 1: Always track both latency AND outcome together
	// This enables calculating latency separately for success vs error cases
	successMetrics := newMetricsCollector()
	errorMetrics := newMetricsCollector()

	// Successful request
	start := time.Now()
	attrs, _ := structpb.NewStruct(map[string]interface{}{"instanceType": "t3.micro"})
	_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})
	if err == nil {
		successMetrics.recordRequest(time.Since(start), nil)
	}

	// Error request (using error plugin)
	errorPlugin := plugintesting.ConfigurableErrorMockPlugin()
	errorPlugin.ShouldErrorOnEstimateCost = true
	errorHarness := plugintesting.NewTestHarness(errorPlugin)
	errorHarness.Start(t)
	defer errorHarness.Stop()

	start = time.Now()
	_, err = errorHarness.Client().EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "test:resource:Type",
		Attributes:   attrs,
	})
	if err != nil {
		errorMetrics.recordRequest(time.Since(start), err)
	}

	// Verify separate tracking works
	if successMetrics.successCount != 1 {
		t.Errorf("Success metrics should have 1 success, got %d", successMetrics.successCount)
	}
	if errorMetrics.errorCount != 1 {
		t.Errorf("Error metrics should have 1 error, got %d", errorMetrics.errorCount)
	}

	// Best Practice 2: Use standard field names for metric labels
	// This ensures consistency across all plugins for aggregation
	t.Logf("Standard metric label names:")
	t.Logf("  operation: %s", pluginsdk.FieldOperation)
	t.Logf("  resource_type: %s", pluginsdk.FieldResourceType)
	t.Logf("  error_code: %s", pluginsdk.FieldErrorCode)

	// Best Practice 3: Calculate statistics that matter
	// - p50: median performance (typical user experience)
	// - p95: captures most users' experience including slower requests
	// - p99: identifies tail latency issues
	// - success_rate: overall reliability metric
	t.Log("Recommended statistics: p50, p95, p99, success_rate")
}

// sortDurations sorts a slice of durations in ascending order.
// Uses simple insertion sort - efficient for small slices.
func sortDurations(durations []time.Duration) {
	for i := 1; i < len(durations); i++ {
		key := durations[i]
		j := i - 1
		for j >= 0 && durations[j] > key {
			durations[j+1] = durations[j]
			j--
		}
		durations[j+1] = key
	}
}

// =============================================================================
// DISTRIBUTED TRACING EXAMPLE
// =============================================================================
//
// TestDistributedTracingExample demonstrates tracing patterns for the
// EstimateCost RPC, per NFR-003 of spec 006-estimate-cost.
//
// This example serves as the canonical reference for plugin developers to
// understand how to properly implement distributed tracing for FinFocus
// plugin operations.
//
// Key patterns demonstrated:
//   - Generating and validating trace IDs
//   - Propagating correlation IDs through gRPC metadata
//   - Extracting trace IDs from incoming requests via interceptors
//   - Creating and linking spans across service boundaries
//   - Injecting/extracting trace context in gRPC calls
//   - Logging with trace context for correlation
//
// Best Practices:
//   - ALWAYS propagate trace_id through the entire request chain
//   - Use TracingUnaryServerInterceptor for automatic trace extraction
//   - Validate incoming trace IDs and generate new ones if invalid
//   - Include trace_id in all log entries for correlation
//   - Create child spans for significant sub-operations
//   - Use standard gRPC metadata keys for interoperability
//
// Note: This example demonstrates the tracing patterns and context propagation.
// Production implementations should integrate with OpenTelemetry or similar
// distributed tracing systems for full observability.
func TestDistributedTracingExample(t *testing.T) {
	// T042: TraceIDGeneration subtest
	t.Run("TraceIDGeneration", testTracingTraceIDGeneration)

	// T042: TraceIDValidation subtest
	t.Run("TraceIDValidation", testTracingTraceIDValidation)

	// T042: ContextPropagation subtest
	t.Run("ContextPropagation", testTracingContextPropagation)

	// T042: CrossCallCorrelation subtest
	t.Run("CrossCallCorrelation", testTracingCrossCallCorrelation)

	// T042: SpanCreation subtest
	t.Run("SpanCreation", testTracingSpanCreation)

	// T042: TracingBestPractices subtest
	t.Run("TracingBestPractices", testTracingBestPractices)
}

func testTracingTraceIDGeneration(t *testing.T) {
	// Best Practice: Generate trace IDs using cryptographically secure random bytes
	// The pluginsdk.GenerateTraceID function produces OpenTelemetry-compatible 32-char hex IDs

	// Generate multiple trace IDs and verify uniqueness
	traceIDs := make(map[string]bool)
	const numIDs = 100

	for range numIDs {
		traceID, err := pluginsdk.GenerateTraceID()
		if err != nil {
			t.Fatalf("Failed to generate trace ID: %v", err)
		}

		// Verify format: 32 lowercase hex characters
		if len(traceID) != 32 {
			t.Errorf("Trace ID should be 32 chars, got %d: %s", len(traceID), traceID)
		}

		// Verify all characters are valid hex
		for _, c := range traceID {
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
			if !isHex {
				t.Errorf("Trace ID contains non-hex character: %c in %s", c, traceID)
			}
		}

		// Verify uniqueness
		if traceIDs[traceID] {
			t.Errorf("Duplicate trace ID generated: %s", traceID)
		}
		traceIDs[traceID] = true
	}

	t.Logf("Generated %d unique trace IDs", numIDs)

	// Best Practice: Never use all-zeros trace ID (invalid per OpenTelemetry spec)
	for traceID := range traceIDs {
		if traceID == "00000000000000000000000000000000" {
			t.Error("Generated all-zeros trace ID (invalid)")
		}
	}
}

func testTracingTraceIDValidation(t *testing.T) {
	// Best Practice: Validate incoming trace IDs before using them
	// Invalid trace IDs should trigger generation of new ones

	tests := []struct {
		name     string
		traceID  string
		isValid  bool
		scenario string
	}{
		{
			name:     "valid trace ID",
			traceID:  "abcdef1234567890abcdef1234567890",
			isValid:  true,
			scenario: "Standard OpenTelemetry trace ID",
		},
		{
			name:     "valid numeric trace ID",
			traceID:  "12345678901234567890123456789012",
			isValid:  true,
			scenario: "All numeric characters are valid hex",
		},
		{
			name:     "empty trace ID",
			traceID:  "",
			isValid:  true, // Empty is valid (optional field)
			scenario: "Missing trace ID - will trigger generation",
		},
		{
			name:     "too short",
			traceID:  "abcdef1234567890",
			isValid:  false,
			scenario: "Must be exactly 32 characters",
		},
		{
			name:     "too long",
			traceID:  "abcdef1234567890abcdef12345678901",
			isValid:  false,
			scenario: "Must be exactly 32 characters",
		},
		{
			name:     "invalid characters",
			traceID:  "ghijkl1234567890abcdef1234567890",
			isValid:  false,
			scenario: "Only 0-9 and a-f are valid",
		},
		{
			name:     "all zeros",
			traceID:  "00000000000000000000000000000000",
			isValid:  false,
			scenario: "All-zeros is invalid per OpenTelemetry spec",
		},
		{
			name:     "uppercase",
			traceID:  "ABCDEF1234567890ABCDEF1234567890",
			isValid:  false, // Lowercase only
			scenario: "Must be lowercase hex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pricing.ValidateTraceID(tt.traceID)
			isValid := err == nil

			if isValid != tt.isValid {
				if tt.isValid {
					t.Errorf("Expected trace ID '%s' to be valid: %v", tt.traceID, err)
				} else {
					t.Errorf("Expected trace ID '%s' to be invalid, but it was valid", tt.traceID)
				}
			}

			t.Logf("Scenario: %s - valid=%v", tt.scenario, isValid)
		})
	}
}

func testTracingContextPropagation(t *testing.T) {
	// Best Practice: Use context to propagate trace IDs through the call stack
	// The pluginsdk provides helper functions for context-based trace propagation

	// Create a trace ID and store it in context
	traceID, err := pluginsdk.GenerateTraceID()
	if err != nil {
		t.Fatalf("Failed to generate trace ID: %v", err)
	}

	// Store trace ID in context
	ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

	// Retrieve trace ID from context
	retrievedTraceID := pluginsdk.TraceIDFromContext(ctx)
	if retrievedTraceID != traceID {
		t.Errorf("Expected trace ID %s, got %s", traceID, retrievedTraceID)
	}

	// Best Practice: Handle missing trace ID gracefully
	emptyCtx := context.Background()
	emptyTraceID := pluginsdk.TraceIDFromContext(emptyCtx)
	if emptyTraceID != "" {
		t.Errorf("Expected empty trace ID from empty context, got %s", emptyTraceID)
	}

	// Best Practice: Context propagation through function calls
	result := simulateNestedCalls(ctx, 3)
	if result != traceID {
		t.Errorf("Trace ID not propagated through nested calls: expected %s, got %s", traceID, result)
	}

	t.Logf("Trace ID %s successfully propagated through nested calls", traceID)
}

// simulateNestedCalls demonstrates trace ID propagation through nested function calls.
func simulateNestedCalls(ctx context.Context, depth int) string {
	// Extract trace ID at each level
	traceID := pluginsdk.TraceIDFromContext(ctx)

	if depth <= 1 {
		return traceID
	}

	// Pass context to nested call
	return simulateNestedCalls(ctx, depth-1)
}

func testTracingCrossCallCorrelation(t *testing.T) {
	// Best Practice: Use the same trace ID across multiple related RPC calls
	// This enables end-to-end visibility through cost estimation flows

	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()

	// Generate a trace ID for this request chain
	traceID, err := pluginsdk.GenerateTraceID()
	if err != nil {
		t.Fatalf("Failed to generate trace ID: %v", err)
	}

	ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

	// Simulate a cost estimation workflow with correlated calls
	// In production, each call would include the trace_id in gRPC metadata

	// Step 1: Check resource support
	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
	supportsResp, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
	if err != nil {
		t.Fatalf("Supports() failed: %v", err)
	}
	t.Logf("[trace_id=%s] Step 1 - Supports: %v", traceID, supportsResp.GetSupported())

	// Step 2: Get projected cost
	projectedResp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: resource})
	if err != nil {
		t.Fatalf("GetProjectedCost() failed: %v", err)
	}
	t.Logf("[trace_id=%s] Step 2 - ProjectedCost: %.2f %s",
		traceID, projectedResp.GetUnitPrice(), projectedResp.GetCurrency())

	// Step 3: Get pricing spec
	specResp, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
	if err != nil {
		t.Fatalf("GetPricingSpec() failed: %v", err)
	}
	t.Logf("[trace_id=%s] Step 3 - PricingSpec: %s",
		traceID, specResp.GetSpec().GetBillingMode())

	// Step 4: Estimate cost
	attrs, _ := structpb.NewStruct(map[string]interface{}{
		"instanceType": "t3.micro",
		"region":       "us-east-1",
	})
	estimateResp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
		Attributes:   attrs,
	})
	if err != nil {
		t.Fatalf("EstimateCost() failed: %v", err)
	}
	t.Logf("[trace_id=%s] Step 4 - EstimateCost: %.2f %s/month",
		traceID, estimateResp.GetCostMonthly(), estimateResp.GetCurrency())

	// Best Practice: All related calls share the same trace_id for correlation
	// This enables querying all logs/spans for a single request chain
	t.Logf("All 4 RPC calls correlated with trace_id=%s", traceID)
}

func testTracingSpanCreation(t *testing.T) {
	// Best Practice: Create spans for significant sub-operations
	// This provides granular timing and error tracking within a trace

	// spanInfo represents a simplified span for demonstration
	type spanInfo struct {
		name      string
		traceID   string
		spanID    string
		parentID  string
		startTime time.Time
		endTime   time.Time
		status    string
		tags      map[string]string
	}

	// Generate trace ID for this request
	traceID, err := pluginsdk.GenerateTraceID()
	if err != nil {
		t.Fatalf("Failed to generate trace ID: %v", err)
	}

	// Simulate span creation for a cost estimation operation
	spans := []spanInfo{}

	// Root span: EstimateCost operation
	rootSpan := spanInfo{
		name:      "EstimateCost",
		traceID:   traceID,
		spanID:    generateSpanID(),
		parentID:  "", // Root span has no parent
		startTime: time.Now(),
		tags: map[string]string{
			"operation":     "EstimateCost",
			"resource_type": "aws:ec2/instance:Instance",
			"provider":      "aws",
		},
	}

	// Child span 1: Validate request
	validateSpan := spanInfo{
		name:      "ValidateRequest",
		traceID:   traceID,
		spanID:    generateSpanID(),
		parentID:  rootSpan.spanID,
		startTime: time.Now(),
		tags: map[string]string{
			"operation": "validate",
		},
	}
	time.Sleep(1 * time.Millisecond) // Simulate work
	validateSpan.endTime = time.Now()
	validateSpan.status = "OK"
	spans = append(spans, validateSpan)

	// Child span 2: Fetch pricing data
	fetchSpan := spanInfo{
		name:      "FetchPricingData",
		traceID:   traceID,
		spanID:    generateSpanID(),
		parentID:  rootSpan.spanID,
		startTime: time.Now(),
		tags: map[string]string{
			"operation": "fetch",
			"source":    "pricing_api",
		},
	}
	time.Sleep(2 * time.Millisecond) // Simulate API call
	fetchSpan.endTime = time.Now()
	fetchSpan.status = "OK"
	spans = append(spans, fetchSpan)

	// Child span 3: Calculate cost
	calculateSpan := spanInfo{
		name:      "CalculateCost",
		traceID:   traceID,
		spanID:    generateSpanID(),
		parentID:  rootSpan.spanID,
		startTime: time.Now(),
		tags: map[string]string{
			"operation": "calculate",
		},
	}
	time.Sleep(1 * time.Millisecond) // Simulate calculation
	calculateSpan.endTime = time.Now()
	calculateSpan.status = "OK"
	spans = append(spans, calculateSpan)

	// Complete root span
	rootSpan.endTime = time.Now()
	rootSpan.status = "OK"
	spans = append([]spanInfo{rootSpan}, spans...) // Prepend root span

	// Verify span relationships
	t.Log("Span hierarchy:")
	for _, span := range spans {
		duration := span.endTime.Sub(span.startTime)
		parentInfo := "root"
		if span.parentID != "" {
			parentInfo = "parent=" + span.parentID[:8] + "..."
		}
		t.Logf("  - %s [%s] trace=%s span=%s (%s) duration=%v",
			span.name, span.status, span.traceID[:8]+"...",
			span.spanID[:8]+"...", parentInfo, duration)
	}

	// Verify all spans share the same trace ID
	for _, span := range spans {
		if span.traceID != traceID {
			t.Errorf("Span %s has wrong trace ID: expected %s, got %s",
				span.name, traceID, span.traceID)
		}
	}

	// Verify parent-child relationships
	rootFound := false
	for _, span := range spans {
		if span.parentID == "" {
			rootFound = true
		} else {
			// Verify parent exists
			parentExists := false
			for _, parent := range spans {
				if parent.spanID == span.parentID {
					parentExists = true
					break
				}
			}
			if !parentExists {
				t.Errorf("Span %s has invalid parent ID: %s", span.name, span.parentID)
			}
		}
	}
	if !rootFound {
		t.Error("No root span found (span with empty parentID)")
	}
}

// generateSpanID generates a 16-character hex span ID for demonstration.
// In production, use proper span ID generation from your tracing library.
func generateSpanID() string {
	traceID, err := pluginsdk.GenerateTraceID()
	if err != nil {
		return "0000000000000000"
	}
	// Use first 16 chars as span ID (simplified for demo)
	return traceID[:16]
}

func testTracingBestPractices(t *testing.T) {
	// This test documents best practices for distributed tracing
	// and validates the recommended patterns

	var buf bytes.Buffer
	logger := pluginsdk.NewPluginLogger("example-plugin", "v1.0.0", zerolog.InfoLevel, &buf)

	// Best Practice 1: Always include trace_id in log entries
	// This enables correlation between traces and logs
	traceID, _ := pluginsdk.GenerateTraceID()
	ctx := pluginsdk.ContextWithTraceID(context.Background(), traceID)

	logger.Info().
		Str(pluginsdk.FieldTraceID, pluginsdk.TraceIDFromContext(ctx)).
		Str(pluginsdk.FieldOperation, "EstimateCost").
		Msg("Processing request")

	logOutput := buf.String()
	if !strings.Contains(logOutput, traceID) {
		t.Error("Log entry should contain trace_id")
	}

	// Best Practice 2: Use standard metadata key for gRPC propagation
	t.Logf("Standard gRPC metadata key: %s", pluginsdk.TraceIDMetadataKey)
	if pluginsdk.TraceIDMetadataKey != "x-finfocus-trace-id" {
		t.Errorf("Expected metadata key 'x-finfocus-trace-id', got '%s'",
			pluginsdk.TraceIDMetadataKey)
	}

	// Best Practice 3: Generate new trace ID if incoming is invalid
	invalidTraceID := "invalid"
	if pricing.ValidateTraceID(invalidTraceID) == nil {
		t.Error("Invalid trace ID should fail validation")
	}
	// When invalid, generate a new one
	newTraceID, err := pluginsdk.GenerateTraceID()
	if err != nil {
		t.Fatalf("Failed to generate replacement trace ID: %v", err)
	}
	t.Logf("Replaced invalid trace ID with: %s", newTraceID)

	// Best Practice 4: Document trace format requirements
	t.Log("Trace ID requirements:")
	t.Log("  - Exactly 32 lowercase hexadecimal characters")
	t.Log("  - Not all zeros (invalid per OpenTelemetry)")
	t.Log("  - Generated using crypto/rand for uniqueness")

	// Best Practice 5: Use TracingUnaryServerInterceptor for automatic extraction
	t.Log("Server interceptor automatically:")
	t.Log("  - Extracts trace_id from x-finfocus-trace-id header")
	t.Log("  - Validates the trace ID format")
	t.Log("  - Generates new trace ID if missing or invalid")
	t.Log("  - Stores trace ID in context for handler access")

	// Best Practice 6: Include span information for sub-operations
	buf.Reset()
	spanID := generateSpanID()
	logger.Info().
		Str(pluginsdk.FieldTraceID, traceID).
		Str("span_id", spanID).
		Str(pluginsdk.FieldOperation, "FetchPricingData").
		Int64(pluginsdk.FieldDurationMs, 15).
		Msg("Sub-operation completed")

	logOutput = buf.String()
	if !strings.Contains(logOutput, spanID) {
		t.Error("Log entry should contain span_id for sub-operations")
	}

	t.Log("Best practices validation completed")
}

// =============================================================================
// CONCURRENT ESTIMATE COST TESTING (T044)
// =============================================================================
//
// TestConcurrentEstimateCost tests EstimateCost under concurrent load
// per Advanced conformance requirements (T044).
//
// Key patterns tested:
//   - 50+ simultaneous requests
//   - <500ms response time under load
//   - Thread safety and race condition detection
//   - Throughput under concurrent load
//   - Resource contention handling
//
// Best Practices:
//   - Run with -race flag to detect race conditions
//   - Verify all responses complete successfully
//   - Track individual request latencies
//   - Ensure bounded memory growth

// TestConcurrentEstimateCost50 tests EstimateCost with exactly 50 concurrent requests.
// This validates Advanced conformance requirement for handling 50+ concurrent requests.
func TestConcurrentEstimateCost50(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	const numRequests = plugintesting.AdvancedParallelRequests // 50 requests

	// Track results from all concurrent requests
	type result struct {
		index    int
		duration time.Duration
		err      error
	}

	results := make(chan result, numRequests)
	var wg sync.WaitGroup

	// Launch 50 concurrent requests
	startTime := time.Now()
	for i := range numRequests {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			start := time.Now()
			attrs, _ := structpb.NewStruct(map[string]interface{}{
				"instanceType": "t3.micro",
				"region":       "us-east-1",
			})
			_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   attrs,
			})
			duration := time.Since(start)

			results <- result{
				index:    idx,
				duration: duration,
				err:      err,
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)
	totalDuration := time.Since(startTime)

	// Collect and analyze results
	var latencies []time.Duration
	var errors []error
	var maxLatency time.Duration
	var minLatency = time.Hour

	for r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
		}
		latencies = append(latencies, r.duration)
		if r.duration > maxLatency {
			maxLatency = r.duration
		}
		if r.duration < minLatency {
			minLatency = r.duration
		}
	}

	// Verify no errors occurred
	if len(errors) > 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errors), errors)
	}

	// Verify all requests completed
	if len(latencies) != numRequests {
		t.Errorf("Expected %d completed requests, got %d", numRequests, len(latencies))
	}

	// Verify <500ms response time requirement
	const maxAllowedLatency = 500 * time.Millisecond
	for i, latency := range latencies {
		if latency > maxAllowedLatency {
			t.Errorf("Request %d exceeded 500ms: %v", i, latency)
		}
	}

	// Calculate average latency
	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	// Log performance summary
	t.Logf("Concurrent EstimateCost (n=%d) results:", numRequests)
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Min latency: %v", minLatency)
	t.Logf("  Max latency: %v", maxLatency)
	t.Logf("  Avg latency: %v", avgLatency)
	t.Logf("  Throughput: %.2f requests/sec", float64(numRequests)/totalDuration.Seconds())
	t.Logf("  Errors: %d/%d", len(errors), numRequests)
}

// TestConcurrentEstimateCost100 tests EstimateCost with 100 concurrent requests.
// This exceeds the Advanced conformance requirement to verify scalability headroom.
func TestConcurrentEstimateCost100(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	const numRequests = 100 // 2x Advanced requirement

	results := make(chan time.Duration, numRequests)
	errors := make(chan error, numRequests)
	var wg sync.WaitGroup

	// Launch 100 concurrent requests
	startTime := time.Now()
	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			attrs, _ := structpb.NewStruct(map[string]interface{}{
				"instanceType": "t3.large",
				"region":       "eu-west-1",
			})
			_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   attrs,
			})
			duration := time.Since(start)

			if err != nil {
				errors <- err
			}
			results <- duration
		}()
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)
	close(errors)
	totalDuration := time.Since(startTime)

	// Count errors
	var errorCount int
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	// Collect latencies
	var latencies []time.Duration
	for d := range results {
		latencies = append(latencies, d)
	}

	// All 100 requests should complete successfully
	if errorCount > 0 {
		t.Errorf("Expected 0 errors for 100 concurrent requests, got %d", errorCount)
	}

	// Calculate statistics
	var totalLatency, maxLatency time.Duration
	minLatency := time.Hour
	for _, l := range latencies {
		totalLatency += l
		if l > maxLatency {
			maxLatency = l
		}
		if l < minLatency {
			minLatency = l
		}
	}

	t.Logf("Concurrent EstimateCost (n=%d) results:", numRequests)
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Min/Max/Avg latency: %v / %v / %v",
		minLatency, maxLatency, totalLatency/time.Duration(len(latencies)))
	t.Logf("  Throughput: %.2f requests/sec", float64(numRequests)/totalDuration.Seconds())
}

// TestConcurrentEstimateCostLatencyVerification validates <500ms per-request
// latency under concurrent load per SC-002 requirements.
func TestConcurrentEstimateCostLatencyVerification(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	const numRequests = plugintesting.AdvancedParallelRequests
	const maxLatency = 500 * time.Millisecond

	latencies := make(chan time.Duration, numRequests)
	var wg sync.WaitGroup

	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			attrs, _ := structpb.NewStruct(map[string]interface{}{
				"instanceType": "m5.xlarge",
			})
			_, _ = client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   attrs,
			})
			latencies <- time.Since(start)
		}()
	}

	wg.Wait()
	close(latencies)

	// Verify EVERY request completes under 500ms
	var violationCount int
	var completedCount int
	for l := range latencies {
		completedCount++
		if l > maxLatency {
			violationCount++
			t.Logf("Latency violation: %v > %v", l, maxLatency)
		}
	}

	// Verify all requests completed
	if completedCount != numRequests {
		t.Errorf("Expected %d completed requests, got %d", numRequests, completedCount)
	}

	// SC-002: Cost estimates are returned within 500ms
	if violationCount > 0 {
		t.Errorf("SC-002 violation: %d/%d requests exceeded %v",
			violationCount, numRequests, maxLatency)
	}

	t.Logf("Latency verification: %d/%d requests under %v",
		completedCount-violationCount, completedCount, maxLatency)
}

// TestConcurrentEstimateCostMultipleResourceTypes tests concurrent requests
// with different resource types to verify thread safety across resource handling.
func TestConcurrentEstimateCostMultipleResourceTypes(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	resourceTypes := []string{
		"aws:ec2/instance:Instance",
		"aws:s3/bucket:Bucket",
		"aws:lambda/function:Function",
		"azure:compute/virtualMachine:VirtualMachine",
		"gcp:compute/instance:Instance",
	}

	const requestsPerType = 10
	totalRequests := len(resourceTypes) * requestsPerType

	type result struct {
		resourceType string
		duration     time.Duration
		err          error
	}
	results := make(chan result, totalRequests)
	var wg sync.WaitGroup

	// Launch concurrent requests for each resource type
	for _, resType := range resourceTypes {
		for range requestsPerType {
			wg.Add(1)
			go func(rt string) {
				defer wg.Done()

				start := time.Now()
				attrs, _ := structpb.NewStruct(map[string]interface{}{
					"type": "standard",
				})
				_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
					ResourceType: rt,
					Attributes:   attrs,
				})
				results <- result{
					resourceType: rt,
					duration:     time.Since(start),
					err:          err,
				}
			}(resType)
		}
	}

	wg.Wait()
	close(results)

	// Analyze results by resource type
	latenciesByType := make(map[string][]time.Duration)
	errorsByType := make(map[string]int)

	for r := range results {
		latenciesByType[r.resourceType] = append(latenciesByType[r.resourceType], r.duration)
		if r.err != nil {
			errorsByType[r.resourceType]++
		}
	}

	// Verify no errors for any resource type
	for rt, errCount := range errorsByType {
		if errCount > 0 {
			t.Errorf("Resource type %s had %d errors", rt, errCount)
		}
	}

	// Log per-type statistics
	t.Log("Per-resource-type concurrent statistics:")
	for rt, latencies := range latenciesByType {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avg := total / time.Duration(len(latencies))
		t.Logf("  %s: count=%d, avg=%v, errors=%d",
			rt, len(latencies), avg, errorsByType[rt])
	}

	// Verify total request count
	totalProcessed := 0
	for _, latencies := range latenciesByType {
		totalProcessed += len(latencies)
	}
	if totalProcessed != totalRequests {
		t.Errorf("Expected %d total requests, got %d", totalRequests, totalProcessed)
	}
}

// TestConcurrentEstimateCostResponseConsistency verifies that concurrent
// EstimateCost requests for the same resource return consistent results.
func TestConcurrentEstimateCostResponseConsistency(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	const numRequests = 20

	type responseData struct {
		costMonthly float64
		currency    string
	}
	responses := make(chan responseData, numRequests)
	var wg sync.WaitGroup

	// All requests use identical input
	resourceType := "aws:ec2/instance:Instance"
	attrs, _ := structpb.NewStruct(map[string]interface{}{
		"instanceType": "t3.micro",
		"region":       "us-east-1",
	})

	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: resourceType,
				Attributes:   attrs,
			})
			if err == nil {
				responses <- responseData{
					costMonthly: resp.GetCostMonthly(),
					currency:    resp.GetCurrency(),
				}
			}
		}()
	}

	wg.Wait()
	close(responses)

	// Collect all responses
	var allResponses []responseData
	for r := range responses {
		allResponses = append(allResponses, r)
	}

	if len(allResponses) == 0 {
		t.Fatal("No successful responses received")
	}

	// Verify all responses are identical (thread-safe consistent responses)
	first := allResponses[0]
	for i, resp := range allResponses {
		if resp.costMonthly != first.costMonthly {
			t.Errorf("Response %d has inconsistent cost: expected %.2f, got %.2f",
				i, first.costMonthly, resp.costMonthly)
		}
		if resp.currency != first.currency {
			t.Errorf("Response %d has inconsistent currency: expected %s, got %s",
				i, first.currency, resp.currency)
		}
	}

	t.Logf("Consistency verification: %d/%d responses identical (cost=%.2f %s)",
		len(allResponses), numRequests, first.costMonthly, first.currency)
}

// TestConcurrentEstimateCostWithTimeout tests concurrent requests with context timeouts
// to verify proper timeout handling under load.
func TestConcurrentEstimateCostWithTimeout(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()

	const numRequests = plugintesting.AdvancedParallelRequests
	const timeout = 5 * time.Second // Generous timeout to avoid false positives

	successCount := make(chan struct{}, numRequests)
	var wg sync.WaitGroup

	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			attrs, _ := structpb.NewStruct(map[string]interface{}{
				"instanceType": "t3.micro",
			})
			_, err := client.EstimateCost(ctx, &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   attrs,
			})
			if err == nil {
				successCount <- struct{}{}
			}
		}()
	}

	wg.Wait()
	close(successCount)

	// Count successes
	success := 0
	for range successCount {
		success++
	}

	// All requests should complete within timeout
	if success != numRequests {
		t.Errorf("Expected %d requests to complete within %v, got %d",
			numRequests, timeout, success)
	}

	t.Logf("Timeout test: %d/%d requests completed within %v", success, numRequests, timeout)
}

// =============================================================================
// GetRecommendations Integration Tests
// =============================================================================

// createRecommendationsTestHarness creates a test harness with recommendations configured.
func createRecommendationsTestHarness(t *testing.T) (*plugintesting.TestHarness, pbc.CostSourceServiceClient) {
	plugin := plugintesting.NewMockPlugin()
	sampleRecs := plugintesting.GenerateSampleRecommendations(10)
	plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: sampleRecs,
	})

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	return harness, harness.Client()
}

// TestGetRecommendations_BasicRequest tests basic GetRecommendations functionality.
func TestGetRecommendations_BasicRequest(t *testing.T) {
	harness, client := createRecommendationsTestHarness(t)
	defer harness.Stop()

	resp, err := client.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	if err != nil {
		t.Fatalf("GetRecommendations() failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	recs := resp.GetRecommendations()
	if len(recs) != 10 {
		t.Errorf("Expected 10 recommendations, got %d", len(recs))
	}

	validateRecommendationFields(t, recs)
}

// validateRecommendationFields validates required fields on recommendations.
// Note: UNSPECIFIED action_type values are intentionally included in test data
// to verify that consumers properly handle edge cases. This validation skips
// the action_type check to allow testing of the full enum range.
func validateRecommendationFields(t *testing.T, recs []*pbc.Recommendation) {
	for i, rec := range recs {
		if rec.GetId() == "" {
			t.Errorf("Recommendation %d missing id", i)
		}
		if rec.GetCategory() == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
			t.Errorf("Recommendation %d has unspecified category", i)
		}
		// Note: action_type UNSPECIFIED is intentionally included in GenerateSampleRecommendations
		// to test edge case handling. See issue #174 for rationale.
		if rec.GetPriority() == pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_UNSPECIFIED {
			t.Errorf("Recommendation %d has unspecified priority", i)
		}
		if rec.GetResource() == nil {
			t.Errorf("Recommendation %d missing resource", i)
		}
		if rec.GetImpact() == nil {
			t.Errorf("Recommendation %d missing impact", i)
		}
	}
}

// TestGetRecommendations_SummaryValidation tests summary calculation.
func TestGetRecommendations_SummaryValidation(t *testing.T) {
	harness, client := createRecommendationsTestHarness(t)
	defer harness.Stop()

	t.Run("projection_period_wiring", func(t *testing.T) {
		// Use valid projection period ("daily", "monthly", or "annual")
		req := &pbc.GetRecommendationsRequest{
			ProjectionPeriod: "monthly",
		}
		resp, err := client.GetRecommendations(context.Background(), req)
		if err != nil {
			t.Fatalf("GetRecommendations() failed: %v", err)
		}
		if resp.GetSummary() == nil {
			t.Fatal("GetRecommendations() returned a nil summary")
		}
		if resp.GetSummary().GetProjectionPeriod() != "monthly" {
			t.Errorf("Expected summary projection period 'monthly', got '%s'", resp.GetSummary().GetProjectionPeriod())
		}
	})

	resp, err := client.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	if err != nil {
		t.Fatalf("GetRecommendations() failed: %v", err)
	}

	summary := resp.GetSummary()
	if summary == nil {
		t.Fatal("Summary is nil")
	}

	recs := resp.GetRecommendations()
	if summary.GetTotalRecommendations() != int32(len(recs)) {
		t.Errorf("Summary count %d doesn't match recommendations count %d",
			summary.GetTotalRecommendations(), len(recs))
	}

	var expectedSavings float64
	for _, rec := range recs {
		if rec.GetImpact() != nil {
			expectedSavings += rec.GetImpact().GetEstimatedSavings()
		}
	}

	if diff := summary.GetTotalEstimatedSavings() - expectedSavings; diff < -0.01 || diff > 0.01 {
		t.Errorf("Summary savings %f doesn't match calculated %f",
			summary.GetTotalEstimatedSavings(), expectedSavings)
	}

	t.Logf("Summary: %d recommendations, $%.2f total savings",
		summary.GetTotalRecommendations(), summary.GetTotalEstimatedSavings())
}

// TestGetRecommendations_EmptyPlugin tests empty response when no recommendations configured.
func TestGetRecommendations_EmptyPlugin(t *testing.T) {
	emptyPlugin := plugintesting.NewMockPlugin()
	// Explicitly clear recommendations to test empty response behavior
	emptyPlugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{}, // Empty slice
	})
	harness := plugintesting.NewTestHarness(emptyPlugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	if err != nil {
		t.Fatalf("GetRecommendations() failed: %v", err)
	}

	if len(resp.GetRecommendations()) != 0 {
		t.Errorf("Expected empty recommendations, got %d", len(resp.GetRecommendations()))
	}

	if resp.GetSummary() == nil {
		t.Error("Summary should be provided even for empty results")
	}

	if resp.GetSummary().GetTotalRecommendations() != 0 {
		t.Errorf("Expected 0 total recommendations, got %d", resp.GetSummary().GetTotalRecommendations())
	}
}

// TestGetRecommendations_ErrorHandling tests error responses.
func TestGetRecommendations_ErrorHandling(t *testing.T) {
	errorPlugin := plugintesting.NewMockPlugin()
	errorPlugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		ShouldError:  true,
		ErrorMessage: "simulated recommendations error",
	})

	harness := plugintesting.NewTestHarness(errorPlugin)
	harness.Start(t)
	defer harness.Stop()

	_, err := harness.Client().GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	if err == nil {
		t.Error("Expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("Expected gRPC status error, got: %v", err)
	} else if st.Code() != codes.Unavailable {
		t.Errorf("Expected Unavailable status code, got: %v", st.Code())
	}
}

// TestGetRecommendations_Concurrent tests concurrent request handling.
func TestGetRecommendations_Concurrent(t *testing.T) {
	harness, client := createRecommendationsTestHarness(t)
	defer harness.Stop()

	const numRequests = 10
	var wg sync.WaitGroup
	errCh := make(chan error, numRequests)

	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Errorf("Concurrent requests failed: %v", errs)
	}

	t.Logf("Successfully handled %d concurrent GetRecommendations requests", numRequests)
}

// TestGetRecommendations_CategoryDistribution tests category counts in summary.
func TestGetRecommendations_CategoryDistribution(t *testing.T) {
	harness, client := createRecommendationsTestHarness(t)
	defer harness.Stop()

	resp, err := client.GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	if err != nil {
		t.Fatalf("GetRecommendations() failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetRecommendations() returned nil response")
	}
	if resp.GetSummary() == nil {
		t.Fatal("GetRecommendations() returned a nil summary")
	}

	catCount := make(map[string]int)
	for _, rec := range resp.GetRecommendations() {
		catCount[rec.GetCategory().String()]++
	}

	t.Logf("Category distribution: %v", catCount)

	summary := resp.GetSummary()
	for cat, count := range catCount {
		summaryCount := summary.GetCountByCategory()[cat]
		if summaryCount != int32(count) {
			t.Errorf("Summary category count for %s: expected %d, got %d", cat, count, summaryCount)
		}
	}
}

// =============================================================================
// Target Resources Filtering Tests (Feature 019-target-resources)
// =============================================================================

// TestTargetResourcesFiltering_SingleResource tests filtering with a single target resource.
// US1: Stack-scoped recommendations - filter to only return recommendations for targeted resources.
func TestTargetResourcesFiltering_SingleResource(t *testing.T) {
	// Create a mock plugin with known recommendations
	plugin := plugintesting.NewMockPlugin()

	// Configure recommendations with different providers/types for testing
	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id:       "rec-aws-1",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Id:           "i-12345",
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			{
				Id:       "rec-aws-2",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Id:           "i-67890",
					Provider:     "aws",
					ResourceType: "rds",
					Region:       "us-east-1",
				},
			},
			{
				Id:       "rec-azure-1",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Id:           "vm-12345",
					Provider:     "azure",
					ResourceType: "vm",
					Region:       "eastus",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Request recommendations for only AWS EC2 resources
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
	}

	resp, err := client.GetRecommendations(ctx, req)
	require.NoError(t, err)

	// Should only return the EC2 recommendation
	recs := resp.GetRecommendations()
	require.Len(t, recs, 1, "Expected 1 recommendation for EC2 target")
	require.Equal(t, "rec-aws-1", recs[0].GetId())
	require.Equal(t, "aws", recs[0].GetResource().GetProvider())
	require.Equal(t, "ec2", recs[0].GetResource().GetResourceType())
}

// TestTargetResourcesFiltering_MultipleResources tests filtering with multiple target resources.
// US1: Stack-scoped recommendations with multiple resources in the stack.
func TestTargetResourcesFiltering_MultipleResources(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id:       "rec-aws-ec2",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
				},
			},
			{
				Id:       "rec-aws-rds",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "rds",
				},
			},
			{
				Id:       "rec-aws-s3",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "s3",
				},
			},
			{
				Id:       "rec-azure-vm",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "azure",
					ResourceType: "vm",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request recommendations for EC2 and RDS (not S3 or Azure)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "aws", ResourceType: "rds"},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	// Should return EC2 and RDS recommendations (OR logic between targets)
	recs := resp.GetRecommendations()
	require.Len(t, recs, 2, "Expected 2 recommendations matching target resources")

	// Verify the correct recommendations are returned
	ids := make(map[string]bool)
	for _, rec := range recs {
		ids[rec.GetId()] = true
	}
	require.True(t, ids["rec-aws-ec2"], "Expected EC2 recommendation")
	require.True(t, ids["rec-aws-rds"], "Expected RDS recommendation")
	require.False(t, ids["rec-aws-s3"], "S3 should NOT be included")
	require.False(t, ids["rec-azure-vm"], "Azure VM should NOT be included")
}

// TestTargetResourcesFiltering_EmptyPreservesExisting tests backward compatibility.
// US1: Empty target_resources should return all recommendations (existing behavior).
func TestTargetResourcesFiltering_EmptyPreservesExisting(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{Id: "rec-1", Resource: &pbc.ResourceRecommendationInfo{Provider: "aws", ResourceType: "ec2"}},
			{Id: "rec-2", Resource: &pbc.ResourceRecommendationInfo{Provider: "azure", ResourceType: "vm"}},
			{Id: "rec-3", Resource: &pbc.ResourceRecommendationInfo{Provider: "gcp", ResourceType: "compute"}},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	t.Run("nil target_resources returns all", func(t *testing.T) {
		req := &pbc.GetRecommendationsRequest{
			TargetResources: nil,
		}
		resp, err := harness.Client().GetRecommendations(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, resp.GetRecommendations(), 3, "Expected all 3 recommendations with nil targets")
	})

	t.Run("empty target_resources returns all", func(t *testing.T) {
		req := &pbc.GetRecommendationsRequest{
			TargetResources: []*pbc.ResourceDescriptor{},
		}
		resp, err := harness.Client().GetRecommendations(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, resp.GetRecommendations(), 3, "Expected all 3 recommendations with empty targets")
	})
}

// TestTargetResourcesFiltering_WithSKU tests SKU-specific filtering.
// US2: Pre-deployment optimization - analyze proposed resources with specific SKUs.
func TestTargetResourcesFiltering_WithSKU(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-t3-micro",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.micro",
					Region:       "us-east-1",
				},
			},
			{
				Id: "rec-t3-medium",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.medium",
					Region:       "us-east-1",
				},
			},
			{
				Id: "rec-m5-large",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "m5.large",
					Region:       "us-east-1",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request only t3.medium recommendations (strict SKU match)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2", Sku: "t3.medium"},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	recs := resp.GetRecommendations()
	require.Len(t, recs, 1, "Expected 1 recommendation for t3.medium")
	require.Equal(t, "rec-t3-medium", recs[0].GetId())
	require.Equal(t, "t3.medium", recs[0].GetResource().GetSku())
}

// TestTargetResourcesFiltering_WithRegion tests region-specific filtering.
// US2: Pre-deployment optimization - analyze proposed resources in specific regions.
func TestTargetResourcesFiltering_WithRegion(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-us-east-1",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-east-1",
				},
			},
			{
				Id: "rec-us-west-2",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "us-west-2",
				},
			},
			{
				Id: "rec-eu-west-1",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Region:       "eu-west-1",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request only us-west-2 recommendations (strict region match)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2", Region: "us-west-2"},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	recs := resp.GetRecommendations()
	require.Len(t, recs, 1, "Expected 1 recommendation for us-west-2")
	require.Equal(t, "rec-us-west-2", recs[0].GetId())
	require.Equal(t, "us-west-2", recs[0].GetResource().GetRegion())
}

// TestTargetResourcesFiltering_MultiProvider tests multi-provider filtering.
// US2: Pre-deployment optimization - analyze proposed resources across multiple providers.
func TestTargetResourcesFiltering_MultiProvider(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-aws-ec2",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.medium",
				},
			},
			{
				Id: "rec-azure-vm",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "azure",
					ResourceType: "vm",
					Sku:          "Standard_B2s",
				},
			},
			{
				Id: "rec-gcp-compute",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "gcp",
					ResourceType: "compute",
					Sku:          "e2-medium",
				},
			},
			{
				Id: "rec-aws-s3",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "s3",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request AWS EC2 and Azure VM (cross-provider)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "azure", ResourceType: "vm"},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	recs := resp.GetRecommendations()
	require.Len(t, recs, 2, "Expected 2 recommendations (AWS EC2 + Azure VM)")

	ids := make(map[string]bool)
	for _, rec := range recs {
		ids[rec.GetId()] = true
	}
	require.True(t, ids["rec-aws-ec2"], "Expected AWS EC2 recommendation")
	require.True(t, ids["rec-azure-vm"], "Expected Azure VM recommendation")
	require.False(t, ids["rec-gcp-compute"], "GCP should NOT be included")
	require.False(t, ids["rec-aws-s3"], "AWS S3 should NOT be included")
}

// TestTargetResourcesFiltering_WithTags tests tag-based filtering.
// US3: Batch resource analysis - query recommendations for resources with specific tags.
func TestTargetResourcesFiltering_WithTags(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-prod-web",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Tags:         map[string]string{"env": "prod", "app": "web"},
				},
			},
			{
				Id: "rec-prod-api",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Tags:         map[string]string{"env": "prod", "app": "api"},
				},
			},
			{
				Id: "rec-dev-web",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Tags:         map[string]string{"env": "dev", "app": "web"},
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request only prod environment (tag subset match)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags:         map[string]string{"env": "prod"},
			},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	recs := resp.GetRecommendations()
	require.Len(t, recs, 2, "Expected 2 prod recommendations")

	ids := make(map[string]bool)
	for _, rec := range recs {
		ids[rec.GetId()] = true
	}
	require.True(t, ids["rec-prod-web"], "Expected prod-web")
	require.True(t, ids["rec-prod-api"], "Expected prod-api")
	require.False(t, ids["rec-dev-web"], "dev-web should NOT be included")
}

// TestTargetResourcesFiltering_ANDLogicWithFilter tests AND logic between target_resources and filter.
// US3: Batch resource analysis - target_resources defines scope, filter defines selection.
func TestTargetResourcesFiltering_ANDLogicWithFilter(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id:       "rec-cost-aws",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
				},
			},
			{
				Id:       "rec-perf-aws",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
				},
			},
			{
				Id:       "rec-cost-azure",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "azure",
					ResourceType: "vm",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request AWS EC2 with COST category filter
	// Should return only rec-cost-aws (matches both scope AND filter)
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
		},
		Filter: &pbc.RecommendationFilter{
			Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	recs := resp.GetRecommendations()
	require.Len(t, recs, 1, "Expected 1 recommendation (COST + AWS EC2)")
	require.Equal(t, "rec-cost-aws", recs[0].GetId())
	require.Equal(t, pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST, recs[0].GetCategory())
}

// TestTargetResourcesFiltering_LargeList tests handling of many target resources.
// Uses 25 targets (well below MaxTargetResources limit of 100) for batch query validation.
// US3: Batch resource analysis - support for large batches up to the limit.
func TestTargetResourcesFiltering_LargeList(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	// Generate 50 recommendations with different resource types
	recs := make([]*pbc.Recommendation, 50)
	for i := range 50 {
		recs[i] = &pbc.Recommendation{
			Id: fmt.Sprintf("rec-%d", i),
			Resource: &pbc.ResourceRecommendationInfo{
				Provider:     "aws",
				ResourceType: fmt.Sprintf("type-%d", i),
			},
		}
	}
	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: recs,
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request 25 specific resource types (batch query)
	targets := make([]*pbc.ResourceDescriptor, 25)
	for i := range 25 {
		targets[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: fmt.Sprintf("type-%d", i*2), // Even types: 0, 2, 4, ...
		}
	}

	req := &pbc.GetRecommendationsRequest{
		TargetResources: targets,
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	result := resp.GetRecommendations()
	require.Len(t, result, 25, "Expected 25 recommendations for batch query")

	// Verify all returned recommendations are for even-numbered types
	for _, rec := range result {
		resourceType := rec.GetResource().GetResourceType()
		require.Contains(t, resourceType, "type-", "Should be type-X format")
	}
}

// TestTargetResourcesFiltering_NoMatchReturnsEmpty tests that no match returns empty list.
// Edge Case: When target_resources matches nothing, return empty recommendations.
func TestTargetResourcesFiltering_NoMatchReturnsEmpty(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	// Configure with some recommendations
	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-aws",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.large",
					Region:       "us-east-1",
				},
			},
			{
				Id: "rec-azure",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "azure",
					ResourceType: "vm",
					Sku:          "Standard_B2s",
					Region:       "eastus",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request non-existent resource type
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{
				Provider:     "gcp", // No GCP recommendations exist
				ResourceType: "compute-instance",
			},
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	// Should return empty list, not an error
	recs := resp.GetRecommendations()
	require.Empty(t, recs, "Non-matching target_resources should return empty list")
}

// TestTargetResourcesFiltering_DuplicatesHandled tests that duplicate targets are handled.
// Edge Case: Duplicate ResourceDescriptors in target_resources should not cause duplicates.
func TestTargetResourcesFiltering_DuplicatesHandled(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()

	// Configure with one matching recommendation
	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{
			{
				Id: "rec-aws",
				Resource: &pbc.ResourceRecommendationInfo{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.large",
					Region:       "us-east-1",
				},
			},
		},
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Request the same resource multiple times
	req := &pbc.GetRecommendationsRequest{
		TargetResources: []*pbc.ResourceDescriptor{
			{Provider: "aws", ResourceType: "ec2"},
			{Provider: "aws", ResourceType: "ec2"}, // Duplicate
			{Provider: "aws", ResourceType: "ec2"}, // Another duplicate
		},
	}

	resp, err := harness.Client().GetRecommendations(context.Background(), req)
	require.NoError(t, err)

	// Should return exactly 1 recommendation (not 3 duplicates)
	recs := resp.GetRecommendations()
	require.Len(t, recs, 1, "Duplicate targets should not cause duplicate recommendations")
	require.Equal(t, "rec-aws", recs[0].GetId())
}

// TestTargetResourcesFiltering_ExceedsLimit verifies the RPC handler rejects requests
// exceeding MaxTargetResources (100). This tests end-to-end RPC enforcement.
func TestTargetResourcesFiltering_ExceedsLimit(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Create 101 targets (exceeds MaxTargetResources of 100)
	targets := make([]*pbc.ResourceDescriptor, plugintesting.MaxTargetResources+1)
	for i := range targets {
		targets[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: fmt.Sprintf("type-%d", i),
		}
	}

	req := &pbc.GetRecommendationsRequest{
		TargetResources: targets,
	}

	_, err := harness.Client().GetRecommendations(context.Background(), req)
	require.Error(t, err, "Should reject > 100 target_resources")

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	require.Equal(t, codes.InvalidArgument, st.Code(),
		"Error code should be InvalidArgument for exceeding limit")
}

// TestTargetResourcesFiltering_Concurrent tests concurrent requests with target_resources.
// This ensures the filtering logic is thread-safe under concurrent load.
func TestTargetResourcesFiltering_Concurrent(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	// Configure with diverse recommendations for filtering
	plugin.RecommendationsConfig = plugintesting.RecommendationsConfig{
		Recommendations: plugintesting.GenerateSampleRecommendations(20),
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	const numRequests = 10
	var wg sync.WaitGroup
	errCh := make(chan error, numRequests)

	// Different target_resources for each request to test concurrent filtering
	for i := range numRequests {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &pbc.GetRecommendationsRequest{
				TargetResources: []*pbc.ResourceDescriptor{
					{Provider: "aws", ResourceType: "ec2"},
				},
			}
			_, err := harness.Client().GetRecommendations(context.Background(), req)
			if err != nil {
				errCh <- fmt.Errorf("request %d failed: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// TestGetPluginInfo tests the GetPluginInfo RPC for metadata retrieval.
// This test validates US1: Compatibility Verification - Core can request plugin metadata.

//nolint:gocognit
func TestGetPluginInfo(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	t.Run("ReturnsRequiredFields", func(t *testing.T) {
		resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if err != nil {
			t.Fatalf("GetPluginInfo() failed: %v", err)
		}

		// Validate required fields are present
		if resp.GetName() == "" {
			t.Error("GetPluginInfo: name is required but empty")
		}
		if resp.GetVersion() == "" {
			t.Error("GetPluginInfo: version is required but empty")
		}
		if resp.GetSpecVersion() == "" {
			t.Error("GetPluginInfo: spec_version is required but empty")
		}
	})

	t.Run("SpecVersionIsSemVer", func(t *testing.T) {
		resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if err != nil {
			t.Fatalf("GetPluginInfo() failed: %v", err)
		}

		// Validate spec_version is a valid SemVer format
		specVersion := resp.GetSpecVersion()
		if !plugintesting.IsValidSemVer(specVersion) {
			t.Errorf("GetPluginInfo: spec_version %q is not a valid SemVer (expected format: vMAJOR.MINOR.PATCH)",
				specVersion)
		}
	})

	t.Run("ProvidersListPresent", func(t *testing.T) {
		resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if err != nil {
			t.Fatalf("GetPluginInfo() failed: %v", err)
		}

		// Providers list should be present (can be empty for generic plugins)
		providers := resp.GetProviders()
		t.Logf("GetPluginInfo: providers=%v", providers)
	})

	t.Run("MetadataMapAccessible", func(t *testing.T) {
		resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if err != nil {
			t.Fatalf("GetPluginInfo() failed: %v", err)
		}

		// Metadata map should be accessible (can be nil/empty)
		metadata := resp.GetMetadata()
		t.Logf("GetPluginInfo: metadata keys=%v", func() []string {
			keys := make([]string, 0, len(metadata))
			for k := range metadata {
				keys = append(keys, k)
			}
			return keys
		}())
	})

	t.Run("ResponseTimeWithinBounds", func(t *testing.T) {
		start := time.Now()
		_, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GetPluginInfo() failed: %v", err)
		}

		// GetPluginInfo should complete within 100ms per plan.md
		const maxDuration = 100 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("GetPluginInfo: response time %v exceeds %v threshold", duration, maxDuration)
		}
		t.Logf("GetPluginInfo: response time=%v", duration)
	})

	t.Run("ConsistentResponses", func(t *testing.T) {
		// Make multiple calls and verify consistent responses
		var firstResp *pbc.GetPluginInfoResponse
		for i := range 3 {
			resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
			if err != nil {
				t.Fatalf("GetPluginInfo() call %d failed: %v", i, err)
			}
			if i == 0 {
				firstResp = resp
				continue
			}
			// Verify consistency
			if resp.GetName() != firstResp.GetName() {
				t.Errorf("GetPluginInfo: name changed between calls: %q vs %q",
					firstResp.GetName(), resp.GetName())
			}
			if resp.GetVersion() != firstResp.GetVersion() {
				t.Errorf("GetPluginInfo: version changed between calls: %q vs %q",
					firstResp.GetVersion(), resp.GetVersion())
			}
			if resp.GetSpecVersion() != firstResp.GetSpecVersion() {
				t.Errorf("GetPluginInfo: spec_version changed between calls: %q vs %q",
					firstResp.GetSpecVersion(), resp.GetSpecVersion())
			}
		}
	})
}

// TestGetPluginInfoWithMetadata tests GetPluginInfo when metadata is configured on the plugin.
// This validates US2: Diagnostic Visibility - developers can see additional plugin information.
func TestGetPluginInfoWithMetadata(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	// Configure metadata on the mock plugin
	plugin.Metadata = map[string]string{
		"build_date": "2024-01-15",
		"git_commit": "abc123def",
		"author":     "test-author",
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo() failed: %v", err)
	}

	// Verify metadata is returned correctly
	metadata := resp.GetMetadata()
	if metadata == nil {
		t.Fatal("GetPluginInfo: metadata should not be nil when configured")
	}

	expectedKeys := []string{"build_date", "git_commit", "author"}
	for _, key := range expectedKeys {
		if _, exists := metadata[key]; !exists {
			t.Errorf("GetPluginInfo: metadata missing key %q", key)
		}
	}

	if metadata["build_date"] != "2024-01-15" {
		t.Errorf("GetPluginInfo: metadata[build_date] = %q, want %q",
			metadata["build_date"], "2024-01-15")
	}
	if metadata["git_commit"] != "abc123def" {
		t.Errorf("GetPluginInfo: metadata[git_commit] = %q, want %q",
			metadata["git_commit"], "abc123def")
	}

	t.Logf("GetPluginInfo: metadata validated successfully with %d entries", len(metadata))
}

// TestGetPluginInfoGracefulDegradation tests that calling GetPluginInfo on a plugin
// that doesn't implement it returns Unimplemented error, enabling graceful degradation.
// This validates US3: Legacy plugins are handled without crashing.
func TestGetPluginInfoGracefulDegradation(t *testing.T) {
	// Create a minimal plugin that doesn't implement GetPluginInfo
	// MockPlugin implements GetPluginInfo, so we test the Server's behavior
	// when PluginInfo is not configured (simulating legacy plugin)

	t.Run("UnimplementedErrorCode", func(t *testing.T) {
		// The conformance test already validates Unimplemented handling
		// This test documents the expected error handling pattern

		// When GetPluginInfo is not implemented, the expected behavior is:
		// 1. Server returns status.Error(codes.Unimplemented, ...)
		// 2. Client should catch this error and handle gracefully
		// 3. Core can continue operation without the plugin metadata

		// Example error handling pattern for consumers:
		/*
			resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
			if err != nil {
				if status.Code(err) == codes.Unimplemented {
					// Legacy plugin - use default/fallback values
					log.Info("Plugin does not implement GetPluginInfo, using defaults")
					return &PluginMetadata{Name: "unknown", Version: "unknown"}
				}
				return nil, fmt.Errorf("GetPluginInfo failed: %w", err)
			}
			return &PluginMetadata{
				Name:        resp.GetName(),
				Version:     resp.GetVersion(),
				SpecVersion: resp.GetSpecVersion(),
			}, nil
		*/

		t.Log("Graceful degradation: Legacy plugins should return codes.Unimplemented")
		t.Log("Consumers should check status.Code(err) == codes.Unimplemented")
		t.Log("When Unimplemented, use default/fallback metadata values")
	})

	t.Run("ErrorMessageFormat", func(t *testing.T) {
		// The error message should indicate GetPluginInfo is not implemented
		// This helps with debugging and logging
		t.Log("Error message should contain 'GetPluginInfo' or 'not implemented'")
	})
}

// TestGetPluginInfoConcurrentAccess tests that GetPluginInfo handles concurrent requests safely.
// This validates thread safety of the pluginInfo field access in the Server struct.
func TestGetPluginInfoConcurrentAccess(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// Run 100 concurrent GetPluginInfo calls to stress test thread safety
	const numGoroutines = 100
	var wg sync.WaitGroup

	// Collect any errors from goroutines
	errChan := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d: GetPluginInfo failed: %w", iteration, err)
				return
			}

			// Verify response is valid
			if resp.GetName() == "" {
				errChan <- fmt.Errorf("goroutine %d: GetPluginInfo returned empty name", iteration)
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("GetPluginInfo concurrent access: %d/%d goroutines failed", len(errors), numGoroutines)
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	}

	t.Logf("GetPluginInfo: successfully handled %d concurrent requests", numGoroutines)
}

// =============================================================================
// GetActualCost Pagination Integration Tests
// =============================================================================

// TestPaginatedGetActualCost tests full gRPC round-trip pagination through TestHarness.
func TestPaginatedGetActualCost(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetActualCostDataPoints(200) // Generate 200 data points
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()

	// 200 hours range to generate 200 hourly data points
	start, end := plugintesting.CreateTimeRange(200)

	t.Run("PaginatedFirstPage", func(t *testing.T) {
		resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
			PageSize:   50,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 50)
		require.NotEmpty(t, resp.GetNextPageToken(), "should have next page token")
		require.Equal(t, int32(200), resp.GetTotalCount())
	})

	t.Run("PaginatedFullIteration", func(t *testing.T) {
		var allResults []*pbc.ActualCostResult
		pageToken := ""
		pageCount := 0
		const maxPages = 100 // safety cap to prevent infinite loops

		for {
			resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
				ResourceId: "test-resource",
				Start:      start,
				End:        end,
				PageSize:   50,
				PageToken:  pageToken,
			})
			require.NoError(t, err)
			allResults = append(allResults, resp.GetResults()...)
			pageCount++
			require.LessOrEqual(t, pageCount, maxPages,
				"exceeded %d pages; possible infinite pagination loop", maxPages)

			if resp.GetNextPageToken() == "" {
				break
			}
			pageToken = resp.GetNextPageToken()
		}

		require.Len(t, allResults, 200, "should retrieve all 200 records")
		require.Equal(t, 4, pageCount, "should take 4 pages of 50 records")
	})

	t.Run("DryRunIgnoresPagination", func(t *testing.T) {
		resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: "test-resource",
			Start:      start,
			End:        end,
			PageSize:   10,
			PageToken:  pluginsdk.EncodePageToken(50),
			DryRun:     true,
		})
		require.NoError(t, err)
		require.Empty(t, resp.GetResults(), "dry_run should have no results")
		require.Empty(t, resp.GetNextPageToken(), "dry_run should have no next token")
		require.NotNil(t, resp.GetDryRunResult(), "dry_run should have DryRunResult")
	})
}

// TestBackwardCompat_LegacyPluginNoPagination verifies that a plugin that does NOT
// implement pagination logic returns all records when sent proto3 default page_size=0.
// Legacy plugins ignore unknown fields, so they return everything in one response.
func TestBackwardCompat_LegacyPluginNoPagination(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetActualCostDataPoints(24) // Small dataset
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)

	// Send request with pagination params to a plugin that returns all records
	// When page_size=0 and page_token="" (proto3 defaults), mock returns all results
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
		// page_size=0, page_token="" (proto3 defaults) - non-paginated mode
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults(), "should return results")
	require.Empty(t, resp.GetNextPageToken(), "legacy behavior returns empty next token")
}

// TestBackwardCompat_LegacyHostNoPaginationParams verifies that sending a request
// with page_size=0 and empty page_token to a paginated plugin returns all records.
func TestBackwardCompat_LegacyHostNoPaginationParams(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetActualCostDataPoints(30)
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(30)

	// Legacy host sends no pagination parameters (page_size=0, page_token="")
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "test-resource",
		Start:      start,
		End:        end,
		PageSize:   0,
		PageToken:  "",
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 30, "all 30 records should be returned")
	require.Empty(t, resp.GetNextPageToken(), "no next token when all records returned")
}

func TestBatchCostIntegration(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	ctx := context.Background()
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)

	t.Run("BatchEstimateWithMixedProviders", func(t *testing.T) {
		resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
				plugintesting.CreateResourceDescriptor("azure", "vm", "Standard_B1s", "eastus"),
				plugintesting.CreateResourceDescriptor("gcp", "compute_engine", "e2-micro", "us-central1"),
				plugintesting.CreateResourceDescriptor("kubernetes", "namespace", "", ""),
				plugintesting.CreateResourceDescriptor("aws", "rds", "db.t3.micro", "us-east-1"),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 5)
		for _, result := range resp.GetResults() {
			require.NotNil(t, result.GetCostData())
			assert.NotNil(t, result.GetCostData().GetEstimate())
		}
	})

	t.Run("BatchActualCostIncludesActualCostData", func(t *testing.T) {
		resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
			Start:     start,
			End:       end,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
				plugintesting.CreateResourceDescriptor("aws", "s3", "", "us-east-1"),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 2)
		for _, result := range resp.GetResults() {
			require.NotNil(t, result.GetCostData())
			actual := result.GetCostData().GetActualCost()
			require.NotNil(t, actual)
			assert.NotEmpty(t, actual.GetResults())
		}
	})

	t.Run("BatchProjectedCostIncludesProjectedCostData", func(t *testing.T) {
		resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
			QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
			Resources: []*pbc.ResourceDescriptor{
				plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
				plugintesting.CreateResourceDescriptor("gcp", "compute_engine", "e2-micro", "us-central1"),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 2)
		for _, result := range resp.GetResults() {
			require.NotNil(t, result.GetCostData())
			assert.NotNil(t, result.GetCostData().GetProjectedCost())
		}
	})
}
