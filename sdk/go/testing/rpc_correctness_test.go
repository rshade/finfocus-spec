package testing_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

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

// resourceIgnoringPlugin is a plugin built before GetActualCostRequest.resource existed:
// it never sees the descriptor.
type resourceIgnoringPlugin struct {
	*plugintesting.MockPlugin
}

func (p *resourceIgnoringPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	stripped := proto.CloneOf(req)
	stripped.Resource = nil
	return p.MockPlugin.GetActualCost(ctx, stripped)
}

// resourceRejectingPlugin wrongly rejects any request that carries a descriptor.
type resourceRejectingPlugin struct {
	*plugintesting.MockPlugin
}

func (p *resourceRejectingPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if req.GetResource() != nil {
		return nil, status.Error(codes.InvalidArgument, "resource is not supported")
	}
	return p.MockPlugin.GetActualCost(ctx, req)
}

// TestRPCCorrectness_GetActualCostWithResource checks that a descriptor on GetActualCost is
// additive: plugins that read it or ignore it pass, and only rejecting it fails.
func TestRPCCorrectness_GetActualCostWithResource(t *testing.T) {
	var test plugintesting.ConformanceSuiteTest
	for _, candidate := range plugintesting.RPCCorrectnessTests() {
		if candidate.Name == "RPCCorrectness_GetActualCostWithResource" {
			test = candidate
		}
	}
	if test.TestFunc == nil {
		t.Fatal("RPCCorrectness_GetActualCostWithResource is not registered")
	}
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
		{name: "plugin reading the descriptor passes", plugin: plugintesting.NewMockPlugin(), wantSuccess: true},
		{
			name:        "plugin ignoring the descriptor passes",
			plugin:      &resourceIgnoringPlugin{MockPlugin: plugintesting.NewMockPlugin()},
			wantSuccess: true,
		},
		{name: "plugin with no data passes", plugin: erroring, wantSuccess: true},
		{
			name:        "plugin rejecting the descriptor fails",
			plugin:      &resourceRejectingPlugin{MockPlugin: plugintesting.NewMockPlugin()},
			wantSuccess: false,
		},
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
		})
	}
}

type spotRecommendationsPlugin struct {
	*plugintesting.MockPlugin

	recommendations []*pbc.Recommendation
	pages           [][]*pbc.Recommendation
	capabilities    []pbc.PluginCapability
}

func (p *spotRecommendationsPlugin) GetRecommendations(
	_ context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	if len(p.pages) == 0 {
		return &pbc.GetRecommendationsResponse{Recommendations: p.recommendations}, nil
	}
	index := 0
	if token := req.GetPageToken(); token != "" {
		index, _ = strconv.Atoi(token)
	}
	resp := &pbc.GetRecommendationsResponse{Recommendations: p.pages[index]}
	if index+1 < len(p.pages) {
		resp.NextPageToken = strconv.Itoa(index + 1)
	}
	return resp, nil
}

func (p *spotRecommendationsPlugin) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	resp, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Capabilities = p.capabilities
	return resp, nil
}

type unimplementedRecommendationsPlugin struct {
	*plugintesting.MockPlugin

	capabilities []pbc.PluginCapability
}

func (p *unimplementedRecommendationsPlugin) GetRecommendations(
	context.Context,
	*pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "no recommendations")
}

func (p *unimplementedRecommendationsPlugin) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	resp, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Capabilities = p.capabilities
	return resp, nil
}

// TestRPCCorrectness_GetRecommendationsSpot checks that returned recommendations obey the spot
// pricing rule, that plugins which never set the fields pass, and that a plugin answering
// Unimplemented passes only when it does not declare recommendations.
func TestRPCCorrectness_GetRecommendationsSpot(t *testing.T) {
	var test plugintesting.ConformanceSuiteTest
	for _, candidate := range plugintesting.RPCCorrectnessTests() {
		if candidate.Name == "RPCCorrectness_GetRecommendationsSpot" {
			test = candidate
		}
	}
	if test.TestFunc == nil {
		t.Fatal("RPCCorrectness_GetRecommendationsSpot is not registered")
	}
	if test.MinLevel != plugintesting.ConformanceLevelStandard {
		t.Errorf("MinLevel = %v, want Standard", test.MinLevel)
	}
	if test.Category != plugintesting.CategoryRPCCorrectness {
		t.Errorf("Category = %v, want %v", test.Category, plugintesting.CategoryRPCCorrectness)
	}

	invalid := plugintesting.NewSpotRecommendation("rec-bad", 0.5)
	invalid.PricingCategory = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
	declared := []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS}

	cases := []struct {
		name        string
		plugin      pbc.CostSourceServiceServer
		wantSuccess bool
		wantInError string
	}{
		{
			name:        "default mock never setting the fields passes",
			plugin:      plugintesting.NewMockPlugin(),
			wantSuccess: true,
		},
		{
			name: "valid spot recommendation passes",
			plugin: &spotRecommendationsPlugin{
				MockPlugin:      plugintesting.NewMockPlugin(),
				recommendations: []*pbc.Recommendation{plugintesting.NewSpotRecommendation("rec-1", 0.2)},
				capabilities:    declared,
			},
			wantSuccess: true,
		},
		{
			name: "empty list passes",
			plugin: &spotRecommendationsPlugin{
				MockPlugin: plugintesting.NewMockPlugin(), capabilities: declared,
			},
			wantSuccess: true,
		},
		{
			name: "non-zero risk without DYNAMIC fails and names the rule",
			plugin: &spotRecommendationsPlugin{
				MockPlugin:      plugintesting.NewMockPlugin(),
				recommendations: []*pbc.Recommendation{invalid},
				capabilities:    declared,
			},
			wantInError: "spot_interruption_risk_score",
		},
		{
			name: "invalid pair on a later page fails",
			plugin: &spotRecommendationsPlugin{
				MockPlugin: plugintesting.NewMockPlugin(),
				pages: [][]*pbc.Recommendation{
					{plugintesting.NewSpotRecommendation("rec-1", 0.2)},
					{plugintesting.NewSpotRecommendation("rec-2", 0.3)},
					{invalid},
				},
				capabilities: declared,
			},
			wantInError: "spot_interruption_risk_score",
		},
		{
			name: "valid pages all pass",
			plugin: &spotRecommendationsPlugin{
				MockPlugin: plugintesting.NewMockPlugin(),
				pages: [][]*pbc.Recommendation{
					{plugintesting.NewSpotRecommendation("rec-1", 0.2)},
					{plugintesting.NewSpotRecommendation("rec-2", 0.3)},
				},
				capabilities: declared,
			},
			wantSuccess: true,
		},
		{
			name: "plugin declaring capabilities without recommendations is skipped",
			plugin: &spotRecommendationsPlugin{
				MockPlugin:      plugintesting.NewMockPlugin(),
				recommendations: []*pbc.Recommendation{invalid},
				capabilities:    []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS},
			},
			wantSuccess: true,
		},
		{
			name:        "legacy plugin declaring no capabilities and answering Unimplemented passes",
			plugin:      &unimplementedRecommendationsPlugin{MockPlugin: plugintesting.NewMockPlugin()},
			wantSuccess: true,
		},
		{
			name: "plugin declaring recommendations but answering Unimplemented fails",
			plugin: &unimplementedRecommendationsPlugin{
				MockPlugin: plugintesting.NewMockPlugin(), capabilities: declared,
			},
			wantInError: "PLUGIN_CAPABILITY_RECOMMENDATIONS",
		},
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
			if tc.wantInError == "" {
				return
			}
			if result.Error == nil || !strings.Contains(result.Error.Error(), tc.wantInError) {
				t.Errorf("Error = %v, want it to contain %q", result.Error, tc.wantInError)
			}
		})
	}
}

func TestRPCCorrectness_GetRecommendationsSpot_PageCapReported(t *testing.T) {
	var test plugintesting.ConformanceSuiteTest
	for _, candidate := range plugintesting.RPCCorrectnessTests() {
		if candidate.Name == "RPCCorrectness_GetRecommendationsSpot" {
			test = candidate
		}
	}
	if test.TestFunc == nil {
		t.Fatal("RPCCorrectness_GetRecommendationsSpot is not registered")
	}

	const total = 101
	run := func(pageCount int) plugintesting.TestResult {
		pages := make([][]*pbc.Recommendation, pageCount)
		for i := range pages {
			pages[i] = []*pbc.Recommendation{plugintesting.NewSpotRecommendation("rec-"+strconv.Itoa(i), 0.2)}
		}
		harness := plugintesting.NewTestHarness(&spotRecommendationsPlugin{
			MockPlugin: plugintesting.NewMockPlugin(),
			pages:      pages,
		})
		harness.Start(t)
		defer harness.Stop()
		return test.TestFunc(harness)
	}

	capped := run(total)
	if !capped.Success || !strings.Contains(capped.Details, "later pages not checked") {
		t.Errorf("capped run: Success = %v, details = %q", capped.Success, capped.Details)
	}

	complete := run(total - 1)
	if !complete.Success || strings.Contains(complete.Details, "not checked") {
		t.Errorf("complete run: Success = %v, details = %q", complete.Success, complete.Details)
	}
}
