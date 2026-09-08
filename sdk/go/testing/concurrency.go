// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements concurrency testing for the Plugin Conformance Test Suite.
package testing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// RPC method name constants used across the conformance test suite.
const (
	MethodName             = "Name"
	MethodSupports         = "Supports"
	MethodGetActualCost    = "GetActualCost"
	MethodGetProjectedCost = "GetProjectedCost"
	MethodGetPricingSpec   = "GetPricingSpec"
	MethodGetBudgets       = "GetBudgets"
	MethodGetPluginInfo    = "GetPluginInfo"
	MethodConcurrency      = "Concurrency"
	MethodPerformance      = "Performance"
)

// Shared test fixture and result-detail constants used across the conformance test suite.
const (
	// testResourceID is the ResourceId used by RPC correctness/error-handling tests
	// that don't require a specific resource identity.
	testResourceID = "test-resource"

	// detailsResponseValidationFailed is the TestResult.Details message used when
	// an RPC response fails schema/field validation.
	detailsResponseValidationFailed = "Response validation failed"

	// detailsRPCCallFailed is the TestResult.Details message used when an RPC call itself errors.
	detailsRPCCallFailed = "RPC call failed"
)

// ConcurrencyConfig configures concurrency test execution.
type ConcurrencyConfig struct {
	// ParallelRequests is the number of concurrent requests to run.
	ParallelRequests int

	// Timeout is the maximum time to wait for all requests.
	Timeout time.Duration

	// Method is the RPC method to test.
	Method string
}

// DefaultConcurrencyConfig returns the default concurrency configuration.
func DefaultConcurrencyConfig() ConcurrencyConfig {
	return ConcurrencyConfig{
		ParallelRequests: NumConcurrentRequests,
		Timeout:          ConcurrencyTestTimeoutSeconds * time.Second,
		Method:           MethodName,
	}
}

// runParallelRequests executes multiple requests in parallel.
func runParallelRequests(harness *TestHarness, config ConcurrencyConfig) ([]TestResult, error) {
	var wg sync.WaitGroup
	results := make(chan TestResult, config.ParallelRequests)
	errChan := make(chan error, config.ParallelRequests)

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	for i := range config.ParallelRequests {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			start := time.Now()
			var err error

			switch config.Method {
			case MethodName:
				_, err = harness.Client().Name(ctx, &pbc.NameRequest{})
			case MethodSupports:
				resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
				_, err = harness.Client().Supports(ctx, &pbc.SupportsRequest{Resource: resource})
			case MethodGetProjectedCost:
				resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
				_, err = harness.Client().GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: resource})
			case MethodGetPricingSpec:
				resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
				_, err = harness.Client().GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
			default:
				err = fmt.Errorf("unsupported method: %s", config.Method)
			}

			duration := time.Since(start)

			if err != nil {
				errChan <- fmt.Errorf("request %d failed: %w", reqNum, err)
				results <- TestResult{
					Method:   config.Method,
					Category: CategoryConcurrency,
					Success:  false,
					Error:    err,
					Duration: duration,
					Details:  fmt.Sprintf("Request %d failed", reqNum),
				}
			} else {
				results <- TestResult{
					Method:   config.Method,
					Category: CategoryConcurrency,
					Success:  true,
					Duration: duration,
					Details:  fmt.Sprintf("Request %d completed", reqNum),
				}
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(results)
	close(errChan)

	// Collect results
	var testResults []TestResult
	for result := range results {
		testResults = append(testResults, result)
	}

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return testResults, fmt.Errorf("%d/%d requests failed", len(errs), config.ParallelRequests)
	}

	return testResults, nil
}

// validateConsistentResponses validates that concurrent responses are consistent.
func validateConsistentResponses(harness *TestHarness, numRequests int) (bool, error) {
	var wg sync.WaitGroup
	names := make(chan string, numRequests)
	errChan := make(chan error, numRequests)

	ctx, cancel := context.WithTimeout(context.Background(), ConcurrencyTestTimeoutSeconds*time.Second)
	defer cancel()

	for range numRequests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := harness.Client().Name(ctx, &pbc.NameRequest{})
			if err != nil {
				errChan <- err
				return
			}
			names <- resp.GetName()
		}()
	}

	wg.Wait()
	close(names)
	close(errChan)

	// Check for errors
	for err := range errChan {
		return false, err
	}

	// Verify all names are the same
	var firstName string
	first := true
	for name := range names {
		if first {
			firstName = name
			first = false
		} else if name != firstName {
			return false, fmt.Errorf("inconsistent responses: %s vs %s", firstName, name)
		}
	}

	return true, nil
}

// ConcurrencyTests returns the concurrency conformance tests.
func ConcurrencyTests() []ConformanceSuiteTest {
	return []ConformanceSuiteTest{
		{
			Name:        "Concurrency_ParallelRequests_Standard",
			Description: "Validates plugin handles 10 concurrent requests",
			Category:    CategoryConcurrency,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createStandardConcurrencyTest(),
		},
		{
			Name:        "Concurrency_ResponseConsistency",
			Description: "Validates concurrent responses are consistent",
			Category:    CategoryConcurrency,
			MinLevel:    ConformanceLevelStandard,
			TestFunc:    createResponseConsistencyTest(),
		},
		{
			Name:        "Concurrency_ParallelRequests_Advanced",
			Description: "Validates plugin handles 50 concurrent requests",
			Category:    CategoryConcurrency,
			MinLevel:    ConformanceLevelAdvanced,
			TestFunc:    createAdvancedConcurrencyTest(),
		},
	}
}

// createStandardConcurrencyTest creates a Standard level concurrency test (10 requests).
func createStandardConcurrencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()

		config := ConcurrencyConfig{
			ParallelRequests: StandardParallelRequests,
			Timeout:          ConcurrencyTestTimeoutSeconds * time.Second,
			Method:           MethodName,
		}

		results, err := runParallelRequests(harness, config)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodConcurrency,
				Category: CategoryConcurrency,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  fmt.Sprintf("Failed: %d requests", config.ParallelRequests),
			}
		}

		// Count successes
		successes := 0
		for _, r := range results {
			if r.Success {
				successes++
			}
		}

		return TestResult{
			Method:   MethodConcurrency,
			Category: CategoryConcurrency,
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Completed %d/%d concurrent requests", successes, config.ParallelRequests),
		}
	}
}

// createResponseConsistencyTest creates a test for response consistency under load.
func createResponseConsistencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()

		consistent, err := validateConsistentResponses(harness, StandardParallelRequests)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodConcurrency,
				Category: CategoryConcurrency,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  "Response inconsistency detected",
			}
		}

		if !consistent {
			return TestResult{
				Method:   MethodConcurrency,
				Category: CategoryConcurrency,
				Success:  false,
				Error:    errors.New("responses are not consistent"),
				Duration: duration,
				Details:  "Response values differ across concurrent requests",
			}
		}

		return TestResult{
			Method:   MethodConcurrency,
			Category: CategoryConcurrency,
			Success:  true,
			Duration: duration,
			Details:  "All concurrent responses are consistent",
		}
	}
}

// createAdvancedConcurrencyTest creates an Advanced level concurrency test (50 requests).
func createAdvancedConcurrencyTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()

		config := ConcurrencyConfig{
			ParallelRequests: AdvancedParallelRequests,
			Timeout:          AdvancedConcurrencyTimeoutSeconds * time.Second,
			Method:           MethodName,
		}

		results, err := runParallelRequests(harness, config)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodConcurrency,
				Category: CategoryConcurrency,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  fmt.Sprintf("Failed: %d requests", config.ParallelRequests),
			}
		}

		// Count successes
		successes := 0
		for _, r := range results {
			if r.Success {
				successes++
			}
		}

		return TestResult{
			Method:   MethodConcurrency,
			Category: CategoryConcurrency,
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Completed %d/%d concurrent requests", successes, config.ParallelRequests),
		}
	}
}

// RegisterConcurrencyTests registers concurrency tests with a conformance suite.
func RegisterConcurrencyTests(suite *ConformanceSuite) {
	for _, test := range ConcurrencyTests() {
		suite.AddTest(test)
	}
}

// RunConcurrencyTests runs concurrency tests against a plugin.
func RunConcurrencyTests(impl pbc.CostSourceServiceServer) ([]TestResult, error) {
	harness := NewTestHarness(impl)
	defer harness.Stop()

	conn, err := harness.createClientConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	harness.client = pbc.NewCostSourceServiceClient(conn)

	var results []TestResult
	for _, test := range ConcurrencyTests() {
		result := test.TestFunc(harness)
		results = append(results, result)
	}

	return results, nil
}
