package testing_test

import (
	"context"
	"errors"
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestRPCCorrectnessNameRPC validates the Name RPC method (T026).
func TestRPCCorrectnessNameRPC(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		if test.Name == "RPCCorrectness_NameRPC" {
			result := test.TestFunc(harness)
			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
			break
		}
	}
}

// TestRPCCorrectnessSupportsRPC validates the Supports RPC method (T027).
func TestRPCCorrectnessSupportsRPC(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		if test.Name == "RPCCorrectness_SupportsRPC" {
			result := test.TestFunc(harness)
			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
			break
		}
	}
}

// TestRPCCorrectnessNilResource validates nil resource handling (T028).
func TestRPCCorrectnessNilResource(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		if test.Name == "RPCCorrectness_NilResource" {
			result := test.TestFunc(harness)
			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
			break
		}
	}
}

// TestRPCCorrectnessInvalidTimeRange validates invalid time range handling (T029).
func TestRPCCorrectnessInvalidTimeRange(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		if test.Name == "RPCCorrectness_InvalidTimeRange" {
			result := test.TestFunc(harness)
			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
			break
		}
	}
}

// TestRPCCorrectnessAllTests runs all RPC correctness tests.
func TestRPCCorrectnessAllTests(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := test.TestFunc(harness)
			if !result.Success {
				t.Errorf("Test %s failed: %v - %s", test.Name, result.Error, result.Details)
			}
		})
	}
}

// TestRegisterRPCCorrectnessTests validates test registration.
func TestRegisterRPCCorrectnessTests(t *testing.T) {
	suite := plugintesting.NewConformanceSuite()
	plugintesting.RegisterRPCCorrectnessTests(suite)

	// Verify tests were registered
	config := suite.GetConfig()
	if config.TargetLevel != plugintesting.ConformanceLevelStandard {
		t.Errorf("Expected default target level Standard, got %v", config.TargetLevel)
	}
}

// TestRPCCorrectnessPanicRecovery validates that the suite recovers from plugin panics (T077).
func TestRPCCorrectnessPanicRecovery(t *testing.T) {
	// Create a plugin that might panic (mock doesn't panic, but we can test the framework)
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	// Run tests - they should complete without panicking the test framework
	tests := plugintesting.RPCCorrectnessTests()
	for _, test := range tests {
		t.Run(test.Name+"_PanicSafe", func(t *testing.T) {
			// Wrap in recovery to test framework stability
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Test framework panicked: %v", r)
				}
			}()

			result := test.TestFunc(harness)
			// We just verify it doesn't panic
			_ = result
		})
	}
}

// rewritingBillingAccountPlugin attaches FOCUS records carrying an id other than the request's.
type rewritingBillingAccountPlugin struct {
	*plugintesting.MockPlugin
}

func (p *rewritingBillingAccountPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	resp, err := p.MockPlugin.GetActualCost(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, result := range resp.GetResults() {
		if result.GetFocusRecord() != nil {
			result.FocusRecord.BillingAccountId = "invented-account"
		}
	}
	return resp, nil
}

func billingAccountRPCTest(t *testing.T) plugintesting.ConformanceSuiteTest {
	t.Helper()
	for _, test := range plugintesting.RPCCorrectnessTests() {
		if test.Name == "RPCCorrectness_GetActualCostBillingAccount" {
			return test
		}
	}
	t.Fatal("RPCCorrectness_GetActualCostBillingAccount is not registered")
	return plugintesting.ConformanceSuiteTest{}
}

// TestRPCCorrectnessGetActualCostBillingAccount checks the billing account echo rule.
func TestRPCCorrectnessGetActualCostBillingAccount(t *testing.T) {
	test := billingAccountRPCTest(t)
	if test.MinLevel != plugintesting.ConformanceLevelStandard {
		t.Errorf("MinLevel = %v, want Standard", test.MinLevel)
	}

	erroring := plugintesting.NewMockPlugin()
	erroring.ShouldErrorOnActualCost = true

	cases := []struct {
		name        string
		plugin      pbc.CostSourceServiceServer
		wantSuccess bool
	}{
		{name: "echoing plugin passes", plugin: plugintesting.NewMockPlugin(), wantSuccess: true},
		{
			name:        "plugin rewriting the id fails",
			plugin:      &rewritingBillingAccountPlugin{MockPlugin: plugintesting.NewMockPlugin()},
			wantSuccess: false,
		},
		{name: "plugin with no data passes", plugin: erroring, wantSuccess: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			harness := plugintesting.NewTestHarness(tc.plugin)
			harness.Start(t)
			defer harness.Stop()

			result := test.TestFunc(harness)
			if result.Success != tc.wantSuccess {
				t.Fatalf("Success = %v, want %v (error: %v, details: %s)",
					result.Success, tc.wantSuccess, result.Error, result.Details)
			}
			if !tc.wantSuccess && !errors.Is(result.Error, plugintesting.ErrBillingAccountIDMismatch) {
				t.Errorf("Error = %v, want ErrBillingAccountIDMismatch", result.Error)
			}
		})
	}
}

// TestRPCCorrectnessGetProjectedCostWithAttributes checks that a plugin which ignores
// ResourceDescriptor.attributes passes the Basic-level attributes test.
func TestRPCCorrectnessGetProjectedCostWithAttributes(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	var found bool
	for _, test := range plugintesting.RPCCorrectnessTests() {
		if test.Name != "RPCCorrectness_GetProjectedCostWithAttributes" {
			continue
		}
		found = true
		if test.MinLevel != plugintesting.ConformanceLevelBasic {
			t.Errorf("MinLevel = %v, want Basic", test.MinLevel)
		}
		result := test.TestFunc(harness)
		if !result.Success {
			t.Errorf("attributes test failed: %s: %v", result.Details, result.Error)
		}
	}
	if !found {
		t.Fatal("RPCCorrectness_GetProjectedCostWithAttributes not registered")
	}
}
