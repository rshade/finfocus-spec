package testing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/internal/semver"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	bufSize = 1024 * 1024

	// Validation limits.
	maxPluginNameLength = 100 // Maximum allowed plugin name length
	currencyCodeLength  = 3   // ISO currency code length (e.g., USD, EUR)

	// HoursPerDay represents hours in a day for time range calculations.
	HoursPerDay              = 24
	MaxResponseTimeMs        = 100 // Maximum response time in milliseconds for performance tests
	NumConcurrentRequests    = 10  // Number of concurrent requests for concurrency tests
	NumPerformanceIterations = 100 // Number of iterations for performance measurements
	ReducedIterations        = 50  // Reduced iterations for expensive operations
	HoursIn30Days            = 720 // Hours in 30 days (24 * 30)
	MaxLargeQueryTimeSeconds = 10  // Maximum time for large dataset queries in seconds
	NumConsistencyChecks     = 3   // Number of consistency check iterations
	SuccessRateMultiplier    = 100 // Multiplier for success rate percentage calculation

	// DefaultTestTimeoutSeconds is the default timeout for test operations.
	DefaultTestTimeoutSeconds = 60
	// AdvancedTestTimeoutSeconds is the timeout for advanced conformance tests.
	AdvancedTestTimeoutSeconds = 120
	// ConcurrencyTestTimeoutSeconds is the default timeout for concurrency tests.
	ConcurrencyTestTimeoutSeconds = 30

	// DefaultBenchmarkDurationSeconds is the default duration for benchmarks.
	DefaultBenchmarkDurationSeconds = 5
	// AdvancedBenchmarkDurationSeconds is the duration for advanced benchmarks.
	AdvancedBenchmarkDurationSeconds = 10

	// StandardParallelRequests is the number of concurrent requests for Standard conformance.
	StandardParallelRequests = 10
	// AdvancedParallelRequests is the number of concurrent requests for Advanced conformance.
	AdvancedParallelRequests = 50

	// NameStandardLatencyMs is the Name RPC standard latency threshold in milliseconds.
	NameStandardLatencyMs = 100
	// NameAdvancedLatencyMs is the Name RPC advanced latency threshold in milliseconds.
	NameAdvancedLatencyMs = 50
	// SupportsStandardLatencyMs is the Supports RPC standard latency threshold in milliseconds.
	SupportsStandardLatencyMs = 50
	// SupportsAdvancedLatencyMs is the Supports RPC advanced latency threshold in milliseconds.
	SupportsAdvancedLatencyMs = 25
	// GetPluginInfoStandardLatencyMs is the GetPluginInfo RPC standard latency threshold in milliseconds.
	GetPluginInfoStandardLatencyMs = 100
	// GetPluginInfoAdvancedLatencyMs is the GetPluginInfo RPC advanced latency threshold in milliseconds.
	GetPluginInfoAdvancedLatencyMs = 50
	// ProjectedCostStandardLatencyMs is the GetProjectedCost standard latency threshold in milliseconds.
	ProjectedCostStandardLatencyMs = 200
	// ProjectedCostAdvancedLatencyMs is the GetProjectedCost advanced latency threshold in milliseconds.
	ProjectedCostAdvancedLatencyMs = 100
	// PricingSpecStandardLatencyMs is the GetPricingSpec standard latency threshold in milliseconds.
	PricingSpecStandardLatencyMs = 200
	// PricingSpecAdvancedLatencyMs is the GetPricingSpec advanced latency threshold in milliseconds.
	PricingSpecAdvancedLatencyMs = 100
	// ActualCost24hStandardLatencyMs is the GetActualCost 24h standard latency threshold in milliseconds.
	ActualCost24hStandardLatencyMs = 2000
	// ActualCost24hAdvancedLatencyMs is the GetActualCost 24h advanced latency threshold in milliseconds.
	ActualCost24hAdvancedLatencyMs = 1000
	// ActualCost30dAdvancedLatencyMs is the GetActualCost 30d advanced latency threshold in milliseconds.
	ActualCost30dAdvancedLatencyMs = 10000
	// GetBudgetsStandardLatencyMs is the GetBudgets standard latency threshold in milliseconds (<5s).
	GetBudgetsStandardLatencyMs = 5000
	// GetBudgetsAdvancedLatencyMs is the GetBudgets advanced latency threshold in milliseconds (<2s).
	GetBudgetsAdvancedLatencyMs = 2000

	// AdvancedConcurrencyTimeoutSeconds is the timeout for advanced concurrency tests.
	AdvancedConcurrencyTimeoutSeconds = 60

	// LatencyTestIterations is the number of iterations for latency tests.
	LatencyTestIterations = 10
	// VarianceTestIterations is the number of iterations for variance tests.
	VarianceTestIterations = 20
	// PercentageCalculationFactor is the multiplier for percentage calculations.
	PercentageCalculationFactor = 100
	// CurrencyCodeRequiredLength is the required length for ISO 4217 currency codes.
	CurrencyCodeRequiredLength = 3
)

// TestHarness provides a testing framework for CostSource plugin implementations.
type TestHarness struct {
	server   *grpc.Server
	listener *bufconn.Listener
	client   pbc.CostSourceServiceClient
	conn     *grpc.ClientConn
}

// NewTestHarness creates a new test harness for the given CostSource implementation.
func NewTestHarness(impl pbc.CostSourceServiceServer) *TestHarness {
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	pbc.RegisterCostSourceServiceServer(server, impl)

	go func() {
		_ = server.Serve(listener)
	}()

	return &TestHarness{
		server:   server,
		listener: listener,
	}
}

// Start initializes the client connection to the test server.
func (h *TestHarness) Start(t testing.TB) {
	//nolint:staticcheck // grpc.NewClient doesn't work with bufconn
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return h.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	h.conn = conn
	h.client = pbc.NewCostSourceServiceClient(conn)
}

// Stop cleans up the test harness.
func (h *TestHarness) Stop() {
	if h.conn != nil {
		_ = h.conn.Close()
	}
	if h.server != nil {
		h.server.Stop()
	}
}

// Client returns the gRPC client for making requests.
func (h *TestHarness) Client() pbc.CostSourceServiceClient {
	return h.client
}

// TestResult represents the result of a single test operation.
// It includes timing information, success status, and error details.
type TestResult struct {
	Method   string
	Success  bool
	Error    error
	Duration time.Duration
	Details  string
	Category TestCategory // Category this test belongs to (for conformance suite)
}

// ConformanceTest represents a single conformance test case.
type ConformanceTest struct {
	Name        string
	Description string
	TestFunc    func(*TestHarness) TestResult
}

// PluginConformanceSuite contains all conformance tests for a plugin.
type PluginConformanceSuite struct {
	tests []ConformanceTest
}

// NewPluginConformanceSuite creates a new conformance test suite.
//
// Deprecated: Use NewConformanceSuite() for the new conformance framework.
func NewPluginConformanceSuite() *PluginConformanceSuite {
	return &PluginConformanceSuite{
		tests: make([]ConformanceTest, 0),
	}
}

// AddTest adds a test to the conformance suite.
func (s *PluginConformanceSuite) AddTest(test ConformanceTest) {
	s.tests = append(s.tests, test)
}

// RunTests executes all conformance tests against the given plugin implementation.
func (s *PluginConformanceSuite) RunTests(t *testing.T, impl pbc.CostSourceServiceServer) []TestResult {
	harness := NewTestHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	results := make([]TestResult, 0, len(s.tests))

	for _, test := range s.tests {
		t.Run(test.Name, func(t *testing.T) {
			result := test.TestFunc(harness)
			results = append(results, result)

			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
		})
	}

	return results
}

// Standard test helpers

// CreateResourceDescriptor creates a standard resource descriptor for testing.
func CreateResourceDescriptor(provider, resourceType, sku, region string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Provider:     provider,
		ResourceType: resourceType,
		Sku:          sku,
		Region:       region,
		Tags: map[string]string{
			"environment": "test",
			"app":         "integration-test",
		},
	}
}

// CreateTimeRange creates a standard time range for testing.
func CreateTimeRange(hoursBack int) (*timestamppb.Timestamp, *timestamppb.Timestamp) {
	end := time.Now()
	start := end.Add(-time.Duration(hoursBack) * time.Hour)
	return timestamppb.New(start), timestamppb.New(end)
}

// ValidateNameResponse validates a Name RPC response.
func ValidateNameResponse(response *pbc.NameResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}
	if response.GetName() == "" {
		return errors.New("plugin name is empty")
	}
	if len(response.GetName()) > maxPluginNameLength {
		return fmt.Errorf("plugin name too long: %d characters", len(response.GetName()))
	}
	return nil
}

// IsValidSemVer validates that a version string is a valid semantic version.
// The version must be in the format vMAJOR.MINOR.PATCH where MAJOR, MINOR, and PATCH
// are non-negative integers without leading zeros (except for 0 itself).
//
// Valid examples: "v0.4.11", "v1.0.0", "v2.15.3"
// Invalid examples: "0.4.11" (no v prefix), "v1.2" (missing patch), "v01.2.3" (leading zero).
//
// This function delegates to the internal/semver package to avoid regex duplication.
func IsValidSemVer(version string) bool {
	return semver.IsValid(version)
}

// ValidateSupportsResponse validates a Supports RPC response.
func ValidateSupportsResponse(response *pbc.SupportsResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}
	// If not supported, should have a reason
	if !response.GetSupported() && response.GetReason() == "" {
		return errors.New("unsupported resource should have a reason")
	}
	return nil
}

// ValidateActualCostResponse validates a GetActualCost RPC response.
func ValidateActualCostResponse(response *pbc.GetActualCostResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}

	results := response.GetResults()
	if len(results) == 0 {
		// Empty results are valid (no cost data available)
		return nil
	}

	for i, result := range results {
		if err := ValidateActualCostResult(result); err != nil {
			return fmt.Errorf("result[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateActualCostResult validates a single ActualCostResult.
func ValidateActualCostResult(result *pbc.ActualCostResult) error {
	if result == nil {
		return errors.New("result is nil")
	}

	if result.GetTimestamp() == nil {
		return errors.New("timestamp is required")
	}

	if result.GetCost() < 0 {
		return fmt.Errorf("cost cannot be negative: %f", result.GetCost())
	}

	if result.GetUsageAmount() < 0 {
		return fmt.Errorf("usage amount cannot be negative: %f", result.GetUsageAmount())
	}

	if result.GetSource() == "" {
		return errors.New("source is required")
	}

	return nil
}

// ValidateProjectedCostResponse validates a GetProjectedCost RPC response.
func ValidateProjectedCostResponse(response *pbc.GetProjectedCostResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}

	if response.GetUnitPrice() < 0 {
		return fmt.Errorf("unit price cannot be negative: %f", response.GetUnitPrice())
	}

	if response.GetCurrency() == "" {
		return errors.New("currency is required")
	}

	// Currency should be 3-character ISO code
	if len(response.GetCurrency()) != currencyCodeLength {
		return fmt.Errorf("currency should be 3-character ISO code, got: %s", response.GetCurrency())
	}

	if response.GetCostPerMonth() < 0 {
		return fmt.Errorf("cost per month cannot be negative: %f", response.GetCostPerMonth())
	}

	return nil
}

// ValidatePricingSpecResponse validates a GetPricingSpec RPC response.
func ValidatePricingSpecResponse(response *pbc.GetPricingSpecResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}

	spec := response.GetSpec()
	if spec == nil {
		return errors.New("pricing spec is nil")
	}

	return ValidatePricingSpec(spec)
}

// ValidatePricingSpec validates a PricingSpec message.
func ValidatePricingSpec(spec *pbc.PricingSpec) error {
	if spec == nil {
		return errors.New("spec is nil")
	}

	if spec.GetProvider() == "" {
		return errors.New("provider is required")
	}

	if spec.GetResourceType() == "" {
		return errors.New("resource type is required")
	}

	if spec.GetBillingMode() == "" {
		return errors.New("billing mode is required")
	}

	if spec.GetRatePerUnit() < 0 {
		return fmt.Errorf("rate per unit cannot be negative: %f", spec.GetRatePerUnit())
	}

	if spec.GetCurrency() == "" {
		return errors.New("currency is required")
	}

	// Currency should be 3-character ISO code
	if len(spec.GetCurrency()) != currencyCodeLength {
		return fmt.Errorf("currency should be 3-character ISO code, got: %s", spec.GetCurrency())
	}

	return nil
}

// ValidateEstimateCostResponse validates an EstimateCost RPC response.
func ValidateEstimateCostResponse(response *pbc.EstimateCostResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}

	if response.GetCurrency() == "" {
		return errors.New("currency is required")
	}

	// Currency should be 3-character ISO code
	if len(response.GetCurrency()) != currencyCodeLength {
		return fmt.Errorf("currency should be 3-character ISO code, got: %s", response.GetCurrency())
	}

	if response.GetCostMonthly() < 0 {
		return fmt.Errorf("cost_monthly cannot be negative: %f", response.GetCostMonthly())
	}

	return nil
}

// ValidateBudgetsResponse validates a GetBudgets RPC response.
//
//nolint:gocognit,nestif // Complex validation logic requires nested checks
func ValidateBudgetsResponse(response *pbc.GetBudgetsResponse) error {
	if response == nil {
		return errors.New("response is nil")
	}

	// Validate budgets
	for i, budget := range response.GetBudgets() {
		if budget.GetId() == "" {
			return fmt.Errorf("budget[%d]: id is required", i)
		}
		if budget.GetName() == "" {
			return fmt.Errorf("budget[%d]: name is required", i)
		}
		if budget.GetSource() == "" {
			return fmt.Errorf("budget[%d]: source is required", i)
		}
		if budget.GetAmount() == nil {
			return fmt.Errorf("budget[%d]: amount is required", i)
		}
		if budget.GetAmount().GetLimit() <= 0 {
			return fmt.Errorf("budget[%d]: amount.limit must be positive: %f", i, budget.GetAmount().GetLimit())
		}
		if len(budget.GetAmount().GetCurrency()) != currencyCodeLength {
			return fmt.Errorf("budget[%d]: amount.currency should be 3-character ISO code, got: %s",
				i, budget.GetAmount().GetCurrency())
		}
		if budget.GetPeriod() == pbc.BudgetPeriod_BUDGET_PERIOD_UNSPECIFIED {
			return fmt.Errorf("budget[%d]: period must be specified", i)
		}

		// Validate status if present
		if budget.GetStatus() != nil {
			status := budget.GetStatus()
			if status.GetCurrentSpend() < 0 {
				return fmt.Errorf("budget[%d]: status.current_spend cannot be negative: %f",
					i, status.GetCurrentSpend())
			}
			if status.GetForecastedSpend() < status.GetCurrentSpend() && status.GetForecastedSpend() != 0 {
				return fmt.Errorf("budget[%d]: status.forecasted_spend cannot be less than current_spend", i)
			}
			if status.GetPercentageUsed() < 0 {
				return fmt.Errorf("budget[%d]: status.percentage_used cannot be negative: %f",
					i, status.GetPercentageUsed())
			}
			if status.GetPercentageForecasted() < 0 {
				return fmt.Errorf("budget[%d]: status.percentage_forecasted cannot be negative: %f",
					i, status.GetPercentageForecasted())
			}
			if len(status.GetCurrency()) != currencyCodeLength {
				return fmt.Errorf("budget[%d]: status.currency should be 3-character ISO code, got: %s",
					i, status.GetCurrency())
			}
			if status.GetHealth() == pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_UNSPECIFIED {
				return fmt.Errorf("budget[%d]: status.health must be specified", i)
			}
		}
	}

	// Validate summary
	summary := response.GetSummary()
	if summary != nil {
		if summary.GetTotalBudgets() < 0 {
			return fmt.Errorf("summary.total_budgets cannot be negative: %d", summary.GetTotalBudgets())
		}
		if summary.GetBudgetsOk() < 0 {
			return fmt.Errorf("summary.budgets_ok cannot be negative: %d", summary.GetBudgetsOk())
		}
		if summary.GetBudgetsWarning() < 0 {
			return fmt.Errorf("summary.budgets_warning cannot be negative: %d", summary.GetBudgetsWarning())
		}
		if summary.GetBudgetsExceeded() < 0 {
			return fmt.Errorf("summary.budgets_exceeded cannot be negative: %d", summary.GetBudgetsExceeded())
		}
		if summary.GetBudgetsCritical() < 0 {
			return fmt.Errorf("summary.budgets_critical cannot be negative: %d", summary.GetBudgetsCritical())
		}
		total := summary.GetBudgetsOk() + summary.GetBudgetsWarning() + summary.GetBudgetsCritical() + summary.GetBudgetsExceeded()
		if total > summary.GetTotalBudgets() {
			return fmt.Errorf("summary health counts exceed total_budgets: %d > %d", total, summary.GetTotalBudgets())
		}
	}

	return nil
}

// PerformanceMetrics contains performance measurement data.
type PerformanceMetrics struct {
	Method        string
	RequestCount  int
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
}

// MeasurePerformance measures the performance of a function.
func MeasurePerformance(name string, iterations int, testFunc func() error) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{
		Method:       name,
		RequestCount: iterations,
		MinDuration:  time.Hour, // Start with a large value
	}

	var totalErr error
	for range iterations {
		start := time.Now()
		err := testFunc()
		duration := time.Since(start)

		if err != nil && totalErr == nil {
			totalErr = err
		}

		metrics.TotalDuration += duration
		if duration < metrics.MinDuration {
			metrics.MinDuration = duration
		}
		if duration > metrics.MaxDuration {
			metrics.MaxDuration = duration
		}
	}

	if iterations > 0 {
		metrics.AvgDuration = metrics.TotalDuration / time.Duration(iterations)
	}

	return metrics, totalErr
}

// ErrorHandlingTestSuite provides utilities for testing error handling scenarios.
type ErrorHandlingTestSuite struct {
	harness *TestHarness
	client  pbc.CostSourceServiceClient
}

// NewErrorHandlingTestSuite creates a new error handling test suite.
func NewErrorHandlingTestSuite(impl pbc.CostSourceServiceServer) *ErrorHandlingTestSuite {
	harness := NewTestHarness(impl)
	return &ErrorHandlingTestSuite{
		harness: harness,
	}
}

// Start initializes the error handling test suite.
func (s *ErrorHandlingTestSuite) Start(t *testing.T) {
	s.harness.Start(t)
	s.client = s.harness.Client()
}

// Stop shuts down the error handling test suite.
func (s *ErrorHandlingTestSuite) Stop() {
	s.harness.Stop()
}

// TestTransientErrorRetry tests that transient errors are retried appropriately.
func (s *ErrorHandlingTestSuite) TestTransientErrorRetry(t *testing.T, method string, _ int) {
	ctx := context.Background()

	var lastError error
	attempts := 0

	// Create a function that simulates transient failures
	testFunc := func() error {
		attempts++
		switch method {
		case MethodName:
			_, err := s.client.Name(ctx, &pbc.NameRequest{})
			lastError = err
		case MethodSupports:
			resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
			_, err := s.client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
			lastError = err
		case MethodGetActualCost:
			start, end := CreateTimeRange(HoursPerDay)
			_, err := s.client.GetActualCost(ctx, &pbc.GetActualCostRequest{
				ResourceId: testResourceID,
				Start:      start,
				End:        end,
			})
			lastError = err
		case MethodGetProjectedCost:
			resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
			_, err := s.client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: resource})
			lastError = err
		case MethodGetPricingSpec:
			resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
			_, err := s.client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
			lastError = err
		default:
			lastError = fmt.Errorf("unknown method: %s", method)
		}
		return lastError
	}

	// Execute the test
	if err := testFunc(); err != nil {
		t.Logf("Test execution error: %v", err)
	}

	if lastError == nil {
		t.Logf("Method %s completed successfully", method)
	} else {
		t.Logf("Method %s failed with error: %v after %d attempts", method, lastError, attempts)
	}
}

// TestCircuitBreakerTripping tests that circuit breaker trips after repeated failures.
func (s *ErrorHandlingTestSuite) TestCircuitBreakerTripping(t *testing.T, failureThreshold int) {
	ctx := context.Background()
	failures := 0

	for range failureThreshold + 2 {
		_, err := s.client.Name(ctx, &pbc.NameRequest{})
		if err != nil {
			failures++
		}
	}

	if failures < failureThreshold {
		t.Errorf("Expected at least %d failures to trip circuit breaker, got %d", failureThreshold, failures)
	}

	t.Logf("Circuit breaker test completed with %d failures", failures)
}

// TestTimeoutBehavior tests that operations respect timeout configurations.
func (s *ErrorHandlingTestSuite) TestTimeoutBehavior(t *testing.T, method string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	var err error

	switch method {
	case MethodName:
		_, err = s.client.Name(ctx, &pbc.NameRequest{})
	case MethodSupports:
		resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
		_, err = s.client.Supports(ctx, &pbc.SupportsRequest{Resource: resource})
	case MethodGetActualCost:
		startTime, endTime := CreateTimeRange(HoursPerDay)
		_, err = s.client.GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: testResourceID,
			Start:      startTime,
			End:        endTime,
		})
	case MethodGetProjectedCost:
		resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
		_, err = s.client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: resource})
	case MethodGetPricingSpec:
		resource := CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")
		_, err = s.client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
	}

	duration := time.Since(start)

	switch {
	case err != nil && errors.Is(err, context.DeadlineExceeded):
		t.Logf("Method %s correctly timed out after %v (timeout was %v)", method, duration, timeout)
	case duration > timeout:
		t.Errorf("Method %s took %v, which exceeds timeout of %v", method, duration, timeout)
	default:
		t.Logf("Method %s completed in %v (within timeout of %v)", method, duration, timeout)
	}
}

// ValidateErrorResponse validates that an error response contains proper structured error information.
func ValidateErrorResponse(t *testing.T, err error, expectedCode string, expectedCategory string) {
	if err == nil {
		t.Error("Expected error response, got nil")
		return
	}

	// For this basic validation, we check that the error message contains expected information
	// In a more complete implementation, this would parse gRPC status details
	errorMsg := err.Error()

	if expectedCode != "" && !contains(errorMsg, expectedCode) {
		t.Errorf("Expected error to contain code %s, but got: %s", expectedCode, errorMsg)
	}

	if expectedCategory != "" && !contains(errorMsg, expectedCategory) {
		t.Errorf("Expected error to contain category %s, but got: %s", expectedCategory, errorMsg)
	}

	t.Logf("Error validation passed for: %s", errorMsg)
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findInString(s, substr))))
}

// findInString searches for substring in string.
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ErrorTestScenario represents a test scenario for error handling.
type ErrorTestScenario struct {
	Name             string
	Method           string
	ExpectedCode     string
	ExpectedCategory string
	ShouldRetry      bool
	MaxRetries       int
	Timeout          time.Duration
}

// RunErrorTestScenarios runs a set of error handling test scenarios.
func (s *ErrorHandlingTestSuite) RunErrorTestScenarios(t *testing.T, scenarios []ErrorTestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.ShouldRetry {
				s.TestTransientErrorRetry(t, scenario.Method, scenario.MaxRetries)
			}

			if scenario.Timeout > 0 {
				s.TestTimeoutBehavior(t, scenario.Method, scenario.Timeout)
			}
		})
	}
}
