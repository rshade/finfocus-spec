// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements the Plugin Conformance Test Suite for validating plugin implementations.
package testing

import (
	"context"
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ConformanceLevel defines the certification level for plugin validation.
// Higher levels include all tests from lower levels.
type ConformanceLevel int

const (
	// ConformanceLevelBasic - Core functionality, required for all plugins.
	// Tests: Name validation, Supports handling, basic GetProjectedCost/GetPricingSpec.
	ConformanceLevelBasic ConformanceLevel = iota

	// ConformanceLevelStandard - Production readiness, recommended for deployment.
	// Includes Basic tests plus: error handling, consistency, 24h data, 10 concurrent.
	ConformanceLevelStandard

	// ConformanceLevelAdvanced - High performance, for demanding environments.
	// Includes Standard tests plus: latency thresholds, 50 concurrent, 30-day data.
	ConformanceLevelAdvanced
)

// String returns the human-readable name for the conformance level.
func (l ConformanceLevel) String() string {
	switch l {
	case ConformanceLevelBasic:
		return "Basic"
	case ConformanceLevelStandard:
		return "Standard"
	case ConformanceLevelAdvanced:
		return "Advanced"
	default:
		return "Unknown"
	}
}

// TestCategory groups related conformance tests.
type TestCategory string

const (
	// CategorySpecValidation validates PricingSpec schema compliance.
	CategorySpecValidation TestCategory = "spec_validation"

	// CategoryRPCCorrectness validates RPC method behavior with valid/invalid inputs.
	CategoryRPCCorrectness TestCategory = "rpc_correctness"

	// CategoryPerformance validates latency and allocation benchmarks.
	CategoryPerformance TestCategory = "performance"

	// CategoryConcurrency validates parallel request handling.
	CategoryConcurrency TestCategory = "concurrency"
)

// String returns the human-readable name for the test category.
func (c TestCategory) String() string {
	switch c {
	case CategorySpecValidation:
		return "Spec Validation"
	case CategoryRPCCorrectness:
		return "RPC Correctness"
	case CategoryPerformance:
		return MethodPerformance
	case CategoryConcurrency:
		return "Concurrency"
	default:
		return string(c)
	}
}

// SuiteConfig configures conformance suite execution.
type SuiteConfig struct {
	// TargetLevel is the conformance level to validate against.
	// Default: ConformanceLevelStandard
	TargetLevel ConformanceLevel

	// Timeout is the maximum duration for each individual test.
	// Default: 60 * time.Second
	Timeout time.Duration

	// ParallelRequests is the number of concurrent requests for concurrency tests.
	// Default: 10 (Standard), 50 (Advanced)
	ParallelRequests int

	// EnableBenchmarks controls whether performance benchmarks are run.
	// Default: true
	EnableBenchmarks bool

	// BenchmarkDuration is how long to run each benchmark.
	// Default: 5 * time.Second
	BenchmarkDuration time.Duration
}

// DefaultSuiteConfig returns the default suite configuration.
func DefaultSuiteConfig() SuiteConfig {
	return SuiteConfig{
		TargetLevel:       ConformanceLevelStandard,
		Timeout:           DefaultTestTimeoutSeconds * time.Second,
		ParallelRequests:  StandardParallelRequests,
		EnableBenchmarks:  true,
		BenchmarkDuration: DefaultBenchmarkDurationSeconds * time.Second,
	}
}

// ValidationError provides field-level error details for spec validation failures.
// Implements the error interface for seamless integration with Go error handling.
type ValidationError struct {
	// Field is the field name that failed validation.
	Field string `json:"field"`

	// Value is the actual value received.
	Value interface{} `json:"value"`

	// Expected describes what value or constraint was expected.
	Expected string `json:"expected"`

	// Message is a human-readable error message.
	Message string `json:"message"`
}

// NewValidationError creates a new ValidationError with the given details.
func NewValidationError(field string, value interface{}, expected, message string) ValidationError {
	return ValidationError{
		Field:    field,
		Value:    value,
		Expected: expected,
		Message:  message,
	}
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed: field=%s value=%v expected=%s: %s",
		e.Field, e.Value, e.Expected, e.Message)
}

// CategoryResult contains results for a single test category.
type CategoryResult struct {
	// Name is the category identifier.
	Name TestCategory `json:"name"`

	// Passed is the number of tests that passed.
	Passed int `json:"passed"`

	// Failed is the number of tests that failed.
	Failed int `json:"failed"`

	// Skipped is the number of tests skipped (due to level).
	Skipped int `json:"skipped"`

	// Results contains individual test results.
	Results []TestResult `json:"-"` // Excluded from JSON for brevity
}

// ResultSummary contains aggregate test counts.
type ResultSummary struct {
	// Total is the total number of tests executed.
	Total int `json:"total"`

	// Passed is the number of tests that passed.
	Passed int `json:"passed"`

	// Failed is the number of tests that failed.
	Failed int `json:"failed"`

	// Skipped is the number of tests skipped.
	Skipped int `json:"skipped"`
}

// ConformanceResult is the complete result of suite execution.
// This struct is JSON-serializable for CI/CD integration.
type ConformanceResult struct {
	// Version is the report schema version.
	Version string `json:"version"`

	// Timestamp is when the suite was executed.
	Timestamp time.Time `json:"timestamp"`

	// PluginName is the name returned by the plugin's Name() RPC.
	PluginName string `json:"plugin_name"`

	// LevelAchieved is the highest conformance level passed.
	LevelAchieved ConformanceLevel `json:"-"`

	// LevelAchievedStr is the string representation for JSON.
	LevelAchievedStr string `json:"level_achieved"`

	// Summary contains aggregate test counts.
	Summary ResultSummary `json:"summary"`

	// Categories contains results organized by test category.
	Categories map[TestCategory]*CategoryResult `json:"categories"`

	// Duration is the total execution time.
	Duration time.Duration `json:"-"`

	// DurationStr is the string representation for JSON.
	DurationStr string `json:"duration"`
}

// Passed returns true if all tests passed at the target level.
func (r *ConformanceResult) Passed() bool {
	return r.Summary.Failed == 0
}

// ConformanceSuiteTest represents a single conformance test with level and category.
type ConformanceSuiteTest struct {
	// Name is the unique test identifier.
	Name string

	// Description describes what the test validates.
	Description string

	// Category is the test category (spec_validation, rpc_correctness, etc.).
	Category TestCategory

	// MinLevel is the minimum conformance level required to run this test.
	MinLevel ConformanceLevel

	// TestFunc is the test implementation.
	TestFunc func(*TestHarness) TestResult
}

// ConformanceSuite is the main entry point for running conformance tests.
type ConformanceSuite struct {
	tests      []ConformanceSuiteTest
	config     SuiteConfig
	categories map[TestCategory][]int // Test indices by category
}

// NewConformanceSuite creates a new conformance suite with default configuration.
func NewConformanceSuite() *ConformanceSuite {
	return NewConformanceSuiteWithConfig(DefaultSuiteConfig())
}

// NewConformanceSuiteWithConfig creates a new conformance suite with custom configuration.
func NewConformanceSuiteWithConfig(config SuiteConfig) *ConformanceSuite {
	return &ConformanceSuite{
		tests:      make([]ConformanceSuiteTest, 0),
		config:     config,
		categories: make(map[TestCategory][]int),
	}
}

// SetConfig updates the suite configuration.
func (s *ConformanceSuite) SetConfig(config SuiteConfig) {
	s.config = config
}

// GetConfig returns the current suite configuration.
func (s *ConformanceSuite) GetConfig() SuiteConfig {
	return s.config
}

// AddTest adds a conformance test to the suite.
func (s *ConformanceSuite) AddTest(test ConformanceSuiteTest) {
	idx := len(s.tests)
	s.tests = append(s.tests, test)
	s.categories[test.Category] = append(s.categories[test.Category], idx)
}

// Run executes all conformance tests against the plugin implementation.
func (s *ConformanceSuite) Run(impl pbc.CostSourceServiceServer) (*ConformanceResult, error) {
	harness := NewTestHarness(impl)
	defer harness.Stop()

	conn, err := harness.createClientConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	harness.client = pbc.NewCostSourceServiceClient(conn)

	start := time.Now()

	// Get plugin name
	ctx := context.Background()
	nameResp, err := harness.Client().Name(ctx, &pbc.NameRequest{})
	pluginName := "unknown"
	if err == nil && nameResp != nil {
		pluginName = nameResp.GetName()
	}

	// Initialize result
	result := &ConformanceResult{
		Version:    ReportVersion,
		Timestamp:  time.Now(),
		PluginName: pluginName,
		Categories: make(map[TestCategory]*CategoryResult),
	}

	// Run all tests
	for _, test := range s.tests {
		// Skip tests above target level
		if test.MinLevel > s.config.TargetLevel {
			// Initialize category if needed
			if result.Categories[test.Category] == nil {
				result.Categories[test.Category] = &CategoryResult{Name: test.Category}
			}
			result.Categories[test.Category].Skipped++
			result.Summary.Skipped++
			result.Summary.Total++
			continue
		}

		// Run the test
		testResult := test.TestFunc(harness)
		testResult.Category = test.Category

		// Initialize category if needed
		if result.Categories[test.Category] == nil {
			result.Categories[test.Category] = &CategoryResult{Name: test.Category}
		}

		// Record result
		result.Categories[test.Category].Results = append(
			result.Categories[test.Category].Results, testResult)

		if testResult.Success {
			result.Categories[test.Category].Passed++
			result.Summary.Passed++
		} else {
			result.Categories[test.Category].Failed++
			result.Summary.Failed++
		}
		result.Summary.Total++
	}

	// Determine level achieved
	result.LevelAchieved = determineLevelAchieved(result, s.config.TargetLevel)
	result.LevelAchievedStr = result.LevelAchieved.String()
	result.Duration = time.Since(start)
	result.DurationStr = result.Duration.String()

	return result, nil
}

// RunCategory executes tests for a specific category only.
func (s *ConformanceSuite) RunCategory(
	impl pbc.CostSourceServiceServer,
	category TestCategory,
) (*CategoryResult, error) {
	harness := NewTestHarness(impl)
	defer harness.Stop()

	conn, err := harness.createClientConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	harness.client = pbc.NewCostSourceServiceClient(conn)

	result := &CategoryResult{Name: category}

	// Run tests for the specified category
	indices, ok := s.categories[category]
	if !ok {
		return result, nil // No tests for this category
	}

	for _, idx := range indices {
		test := s.tests[idx]

		// Skip tests above target level
		if test.MinLevel > s.config.TargetLevel {
			result.Skipped++
			continue
		}

		// Run the test
		testResult := test.TestFunc(harness)
		testResult.Category = category
		result.Results = append(result.Results, testResult)

		if testResult.Success {
			result.Passed++
		} else {
			result.Failed++
		}
	}

	return result, nil
}

// determineLevelAchieved determines the highest conformance level passed.
// Since TestResult doesn't store MinLevel, we use a simplified approach:
//   - If no tests failed, return the target level.
//   - If tests failed and target > Basic, return one level below target.
//   - If tests failed at Basic level, return Basic (floor).
func determineLevelAchieved(result *ConformanceResult, targetLevel ConformanceLevel) ConformanceLevel {
	// All tests passed at target level
	if result.Summary.Failed == 0 {
		return targetLevel
	}

	// Some tests failed - return one level below target, with Basic as the floor
	if targetLevel > ConformanceLevelBasic {
		return targetLevel - 1
	}

	// At Basic level with failures - still return Basic as the floor
	// The Summary.Failed > 0 already indicates failure status
	return ConformanceLevelBasic
}

// AggregateResults aggregates results from multiple categories.
// This is a utility function for external use.
func AggregateResults(results map[TestCategory]*CategoryResult) ResultSummary {
	var summary ResultSummary
	for _, catResult := range results {
		summary.Total += catResult.Passed + catResult.Failed + catResult.Skipped
		summary.Passed += catResult.Passed
		summary.Failed += catResult.Failed
		summary.Skipped += catResult.Skipped
	}
	return summary
}

// RunBasicConformance runs basic conformance tests and returns the result.
func RunBasicConformance(impl pbc.CostSourceServiceServer) (*ConformanceResult, error) {
	suite := NewConformanceSuiteWithConfig(SuiteConfig{
		TargetLevel:      ConformanceLevelBasic,
		Timeout:          DefaultTestTimeoutSeconds * time.Second,
		ParallelRequests: StandardParallelRequests,
		EnableBenchmarks: false,
	})

	// Register all test categories
	RegisterSpecValidationTests(suite)
	RegisterRPCCorrectnessTests(suite)

	return suite.Run(impl)
}

// RunStandardConformance runs standard conformance tests and returns the result.
func RunStandardConformance(impl pbc.CostSourceServiceServer) (*ConformanceResult, error) {
	suite := NewConformanceSuiteWithConfig(SuiteConfig{
		TargetLevel:      ConformanceLevelStandard,
		Timeout:          DefaultTestTimeoutSeconds * time.Second,
		ParallelRequests: StandardParallelRequests,
		EnableBenchmarks: true,
	})

	// Register all test categories
	RegisterSpecValidationTests(suite)
	RegisterRPCCorrectnessTests(suite)
	RegisterPerformanceTests(suite)
	RegisterConcurrencyTests(suite)

	return suite.Run(impl)
}

// RunAdvancedConformance runs advanced conformance tests and returns the result.
func RunAdvancedConformance(impl pbc.CostSourceServiceServer) (*ConformanceResult, error) {
	suite := NewConformanceSuiteWithConfig(SuiteConfig{
		TargetLevel:       ConformanceLevelAdvanced,
		Timeout:           AdvancedTestTimeoutSeconds * time.Second,
		ParallelRequests:  AdvancedParallelRequests,
		EnableBenchmarks:  true,
		BenchmarkDuration: AdvancedBenchmarkDurationSeconds * time.Second,
	})

	// Register all test categories
	RegisterSpecValidationTests(suite)
	RegisterRPCCorrectnessTests(suite)
	RegisterPerformanceTests(suite)
	RegisterConcurrencyTests(suite)

	return suite.Run(impl)
}
