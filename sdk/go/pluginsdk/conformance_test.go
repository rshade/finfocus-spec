//nolint:testpackage // Testing internal conformance adapter functions with mocks
package pluginsdk

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// conformanceMockPlugin implements the Plugin interface for conformance testing.
// It provides realistic responses that will pass conformance tests.
type conformanceMockPlugin struct {
	name string
}

// Name returns the plugin name.
func (m *conformanceMockPlugin) Name() string { return m.name }

// GetProjectedCost returns a valid projected cost response.
func (m *conformanceMockPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{
		UnitPrice:    0.10,
		Currency:     "USD",
		CostPerMonth: 72.0, // 0.10 * 24 * 30
	}, nil
}

// GetActualCost returns a valid actual cost response.
func (m *conformanceMockPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{},
	}, nil
}

// GetPricingSpec returns a valid pricing spec response.
func (m *conformanceMockPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{
		Spec: &pbc.PricingSpec{
			Provider:     "aws",
			ResourceType: "ec2",
			BillingMode:  "per_hour",
			RatePerUnit:  0.10,
			Currency:     "USD",
			Unit:         "hour",
		},
	}, nil
}

// EstimateCost returns a valid estimate cost response.
func (m *conformanceMockPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{
		Currency:    "USD",
		CostMonthly: 72.0,
	}, nil
}

// =============================================================================
// Phase 2: Foundational (Nil Plugin Validation) Tests - T006, T007
// =============================================================================

// TestValidatePluginNil verifies that validatePlugin returns an error for nil input.
// This test MUST pass before any adapter functions can safely rely on validation.
func TestValidatePluginNil(t *testing.T) {
	err := validatePlugin(nil)

	require.Error(t, err)
	assert.Equal(t, ErrNilPlugin, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestValidatePluginValid verifies that validatePlugin passes for non-nil plugins.
func TestValidatePluginValid(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "test-plugin"}

	err := validatePlugin(plugin)

	require.NoError(t, err)
}

// =============================================================================
// Phase 3: User Story 1 - RunBasicConformance Tests - T009, T010
// =============================================================================

// TestRunBasicConformanceNilPlugin verifies that RunBasicConformance returns
// an error when passed a nil plugin.
func TestRunBasicConformanceNilPlugin(t *testing.T) {
	result, err := RunBasicConformance(nil)

	require.Error(t, err)
	assert.Equal(t, ErrNilPlugin, err)
	assert.Nil(t, result)
}

// TestRunBasicConformanceValidPlugin verifies that RunBasicConformance returns
// a ConformanceResult when passed a valid plugin.
func TestRunBasicConformanceValidPlugin(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "test-basic-plugin"}

	result, err := RunBasicConformance(plugin)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test-basic-plugin", result.PluginName)
	assert.NotZero(t, result.Summary.Total)
}

// =============================================================================
// Phase 4: User Story 2 - RunStandardConformance Tests - T015, T016
// =============================================================================

// TestRunStandardConformanceNilPlugin verifies that RunStandardConformance returns
// an error when passed a nil plugin.
func TestRunStandardConformanceNilPlugin(t *testing.T) {
	result, err := RunStandardConformance(nil)

	require.Error(t, err)
	assert.Equal(t, ErrNilPlugin, err)
	assert.Nil(t, result)
}

// TestRunStandardConformanceValidPlugin verifies that RunStandardConformance returns
// a ConformanceResult when passed a valid plugin.
func TestRunStandardConformanceValidPlugin(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "test-standard-plugin"}

	result, err := RunStandardConformance(plugin)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test-standard-plugin", result.PluginName)
	assert.NotZero(t, result.Summary.Total)
	// Verify at least one test was executed
	assert.GreaterOrEqual(t, result.Summary.Total, 1)
}

// =============================================================================
// Phase 5: User Story 3 - RunAdvancedConformance Tests - T021, T022
// =============================================================================

// TestRunAdvancedConformanceNilPlugin verifies that RunAdvancedConformance returns
// an error when passed a nil plugin.
func TestRunAdvancedConformanceNilPlugin(t *testing.T) {
	result, err := RunAdvancedConformance(nil)

	require.Error(t, err)
	assert.Equal(t, ErrNilPlugin, err)
	assert.Nil(t, result)
}

// TestRunAdvancedConformanceValidPlugin verifies that RunAdvancedConformance returns
// a ConformanceResult when passed a valid plugin.
func TestRunAdvancedConformanceValidPlugin(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "test-advanced-plugin"}

	result, err := RunAdvancedConformance(plugin)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test-advanced-plugin", result.PluginName)
	assert.NotZero(t, result.Summary.Total)
}

// =============================================================================
// Phase 6: User Story 4 - PrintConformanceReport Tests - T027, T028
// =============================================================================

// TestPrintConformanceReportNilResult verifies that PrintConformanceReport does not
// panic when passed a nil result.
func TestPrintConformanceReportNilResult(t *testing.T) {
	// This should not panic - it should log a warning and return
	assert.NotPanics(t, func() {
		PrintConformanceReport(t, nil)
	})
}

// TestPrintConformanceReportValidResult verifies that PrintConformanceReport outputs
// formatted content with expected sections.
func TestPrintConformanceReportValidResult(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "test-report-plugin"}

	result, err := RunBasicConformance(plugin)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Use PrintConformanceReportTo with a buffer to capture output
	var buf bytes.Buffer
	PrintConformanceReportTo(result, &buf)

	output := buf.String()

	// Verify output contains expected sections
	assert.Contains(t, output, "test-report-plugin", "output should contain plugin name")
	assert.Contains(t, output, "Summary", "output should contain summary section")
	assert.Contains(t, output, "Total", "output should contain total count")
	assert.Contains(t, output, "Passed", "output should contain passed count")
}

// TestPrintConformanceReportToNilResult verifies that PrintConformanceReportTo
// returns immediately when passed a nil result without panicking.
func TestPrintConformanceReportToNilResult(t *testing.T) {
	var buf bytes.Buffer

	assert.NotPanics(t, func() {
		PrintConformanceReportTo(nil, &buf)
	})

	// Buffer should be empty since nil result returns early
	assert.Empty(t, buf.String())
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

// TestTypeAliasesExist verifies that type aliases are accessible and work correctly.
func TestTypeAliasesExist(t *testing.T) {
	// Verify ConformanceLevel constants are accessible
	assert.Equal(t, ConformanceLevelBasic, ConformanceLevel(0))
	assert.Equal(t, ConformanceLevelStandard, ConformanceLevel(1))
	assert.Equal(t, ConformanceLevelAdvanced, ConformanceLevel(2))
}

// TestConformanceResultPassed verifies the Passed() method works through the alias.
func TestConformanceResultPassed(t *testing.T) {
	plugin := &conformanceMockPlugin{name: "passed-test-plugin"}

	result, err := RunBasicConformance(plugin)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify Passed() method is accessible through the type alias and returns a boolean.
	// The actual result depends on the mock implementation's completeness.
	// We verify it doesn't panic and returns a deterministic value.
	passed := result.Passed()
	t.Logf("Conformance result: Passed=%v, Total=%d, Passed=%d, Failed=%d",
		passed, result.Summary.Total, result.Summary.Passed, result.Summary.Failed)

	// Verify the result is consistent on repeated calls (deterministic behavior)
	assert.Equal(t, passed, result.Passed(), "Passed() should return consistent results")
}

// TestGetPluginInfoCapabilitiesDiscovery verifies that GetPluginInfo auto-discovers
// capabilities based on implemented interfaces.
func TestGetPluginInfoCapabilitiesDiscovery(t *testing.T) {
	// Create a test plugin that implements specific interfaces
	testPlugin := &capabilityTestPlugin{
		name: "capability-test-plugin",
	}

	// Create server with basic plugin info to enable GetPluginInfo
	pluginInfo := NewPluginInfo("capability-test-plugin", "v1.0.0")
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify auto-discovered capabilities - includes all optional interfaces + base interface capabilities
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
	}

	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilities(),
		"GetPluginInfo should auto-discover capabilities based on implemented interfaces")
}

func TestGetPluginInfoCapabilitiesOverride(t *testing.T) {
	testPlugin := &capabilityTestPlugin{
		name: "override-test-plugin",
	}

	pluginInfo := NewPluginInfo("override-test-plugin", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS))
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.ElementsMatch(t,
		[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS},
		resp.GetCapabilities(),
		"GetPluginInfo should honor explicit capabilities override")
}

// capabilityTestPlugin implements multiple interfaces for testing capability discovery.
type capabilityTestPlugin struct {
	name string
}

func (p *capabilityTestPlugin) Name() string { return p.name }

// Implements Plugin interface.
func (p *capabilityTestPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (p *capabilityTestPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (p *capabilityTestPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *capabilityTestPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

// Implements RecommendationsProvider.
func (p *capabilityTestPlugin) GetRecommendations(
	_ context.Context,
	_ *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return &pbc.GetRecommendationsResponse{}, nil
}

// Implements BudgetsProvider.
func (p *capabilityTestPlugin) GetBudgets(
	_ context.Context,
	_ *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	return &pbc.GetBudgetsResponse{}, nil
}

// Implements DismissProvider.
func (p *capabilityTestPlugin) DismissRecommendation(
	_ context.Context,
	_ *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	return &pbc.DismissRecommendationResponse{}, nil
}

// Implements DryRunHandler.
func (p *capabilityTestPlugin) HandleDryRun(
	_ context.Context,
	_ *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	return &pbc.DryRunResponse{}, nil
}

// consistencyTestRegistry is a mock registry for testing that accepts any provider/region.
type consistencyTestRegistry struct{}

func (r *consistencyTestRegistry) FindPlugin(_, _ string) string {
	// Always return a plugin name to indicate the provider/region combo is supported
	return "test-plugin"
}

// TestGetPluginInfoCapabilitiesEmptyOverride verifies that empty capability override
// falls back to auto-discovered capabilities from implemented interfaces.
func TestGetPluginInfoCapabilitiesEmptyOverride(t *testing.T) {
	testPlugin := &capabilityTestPlugin{
		name: "empty-override-test-plugin",
	}

	// Override with empty slice - should fall back to auto-discovery
	pluginInfo := NewPluginInfo("empty-override-test-plugin", "v1.0.0",
		WithCapabilities()) // Empty override
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Empty override should fall back to globalCapabilities from auto-discovery
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
	}

	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilities(),
		"Empty override should fall back to auto-discovered capabilities")
}

// TestGetPluginInfoCapabilitiesUnimplementedOverride verifies that capability override
// is honored even for capabilities not implemented by the plugin.
func TestGetPluginInfoCapabilitiesUnimplementedOverride(t *testing.T) {
	// Create a minimal plugin with no optional interfaces
	testPlugin := &conformanceMockPlugin{
		name: "minimal-override-test-plugin",
	}

	// Override with CARBON capability, even though plugin doesn't implement CarbonProvider
	pluginInfo := NewPluginInfo("minimal-override-test-plugin", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON))
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.ElementsMatch(t,
		[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON},
		resp.GetCapabilities(),
		"Override with unimplemented capability should be honored")
}

// TestGetPluginInfoConcurrentAccess verifies that concurrent GetPluginInfo calls
// don't cause race conditions when accessing global capabilities.
func TestGetPluginInfoConcurrentAccess(t *testing.T) {
	testPlugin := &capabilityTestPlugin{
		name: "concurrent-test-plugin",
	}

	pluginInfo := NewPluginInfo("concurrent-test-plugin", "v1.0.0")
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	// Launch concurrent requests with goroutine ID tracking for accurate error reporting
	const numGoroutines = 100
	results := make(chan *pbc.GetPluginInfoResponse, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := range numGoroutines {
		go func(goroutineID int) {
			resp, err := server.GetPluginInfo(ctx, req)
			if err != nil {
				// Wrap error with goroutine ID for accurate debugging
				errors <- fmt.Errorf("goroutine %d: %w", goroutineID, err)
			} else {
				results <- resp
			}
		}(i)
	}

	// Collect results - should all succeed without race conditions
	for range numGoroutines {
		select {
		case err := <-errors:
			// Error already includes goroutine ID from the wrapper
			t.Error(err)
		case resp := <-results:
			require.NotNil(t, resp)
			// Verify all concurrent calls return same capabilities
			assert.NotEmpty(t, resp.GetCapabilities())
		}
	}
}

// TestCapabilitiesLegacyMetadataConsistency verifies that GetPluginInfo and Supports
// produce consistent legacy metadata from the same capabilities.
func TestCapabilitiesLegacyMetadataConsistency(t *testing.T) {
	testPlugin := &capabilityTestPlugin{
		name: "legacy-consistency-test-plugin",
	}

	pluginInfo := NewPluginInfo("legacy-consistency-test-plugin", "v1.0.0")
	// Use a simple mock registry that accepts any provider/region
	mockReg := &consistencyTestRegistry{}
	server := NewServerWithOptions(testPlugin, mockReg, nil, pluginInfo)

	ctx := context.Background()

	// Call GetPluginInfo
	getPluginInfoReq := &pbc.GetPluginInfoRequest{}
	getPluginInfoResp, err := server.GetPluginInfo(ctx, getPluginInfoReq)
	require.NoError(t, err)
	require.NotNil(t, getPluginInfoResp)

	// Call Supports
	supportsReq := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			Region:       "us-east-1",
			ResourceType: "ec2",
		},
	}
	supportsResp, err := server.Supports(ctx, supportsReq)
	require.NoError(t, err)
	require.NotNil(t, supportsResp)

	// Verify both have legacy metadata (for backward compatibility)
	assert.NotNil(t, getPluginInfoResp.GetMetadata())
	assert.NotNil(t, supportsResp.GetCapabilities())

	// Verify the legacy metadata keys are consistent
	getPluginInfoMetadataKeys := make(map[string]bool)
	for key := range getPluginInfoResp.GetMetadata() {
		getPluginInfoMetadataKeys[key] = true
	}

	supportsCapabilitiesKeys := make(map[string]bool)
	for key := range supportsResp.GetCapabilities() {
		supportsCapabilitiesKeys[key] = true
	}

	assert.ElementsMatch(t, mapKeys(getPluginInfoMetadataKeys), mapKeys(supportsCapabilitiesKeys),
		"Legacy metadata keys should be consistent between GetPluginInfo and Supports")
}

// TestGetPluginInfoCapabilitiesWithUnspecified verifies that UNSPECIFIED capability
// is filtered out from legacy metadata mapping.
func TestGetPluginInfoCapabilitiesWithUnspecified(t *testing.T) {
	testPlugin := &capabilityTestPlugin{
		name: "unspecified-test-plugin",
	}

	// Override with UNSPECIFIED and a real capability
	pluginInfo := NewPluginInfo("unspecified-test-plugin", "v1.0.0",
		WithCapabilities(
			pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		))
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify UNSPECIFIED is included in enum capabilities (it was explicitly set)
	assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED)

	// Verify UNSPECIFIED is NOT in legacy metadata (should be filtered out)
	metadata := resp.GetMetadata()
	for key := range metadata {
		assert.NotEqual(t, "unspecified", key,
			"UNSPECIFIED should not appear in legacy metadata keys")
	}

	// Verify PROJECTED_COSTS is in legacy metadata
	assert.NotNil(t, metadata["supports_projected_costs"],
		"PROJECTED_COSTS should be in legacy metadata")
}

// TestGetPluginInfoCapabilitiesInvalidEnumValue verifies that invalid enum values
// (undefined capability integers) are handled gracefully without panicking.
// This tests robustness against malformed capability configurations.
func TestGetPluginInfoCapabilitiesInvalidEnumValue(t *testing.T) {
	testPlugin := &conformanceMockPlugin{
		name: "invalid-enum-test-plugin",
	}

	// Create capability with invalid enum value (999 is not a defined PluginCapability)
	invalidCapability := pbc.PluginCapability(999)
	pluginInfo := NewPluginInfo("invalid-enum-test-plugin", "v1.0.0",
		WithCapabilities(
			invalidCapability,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		))
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	// Should not panic - handle gracefully
	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err, "GetPluginInfo should not error on invalid enum values")
	require.NotNil(t, resp)

	// Both capabilities should be in the response (enum slice is passed through)
	caps := resp.GetCapabilities()
	assert.Contains(t, caps, invalidCapability,
		"Invalid enum value should be passed through in capabilities")
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		"Valid capability should be present")

	// Invalid enum should NOT appear in legacy metadata (no mapping exists)
	metadata := resp.GetMetadata()
	assert.Equal(t, "true", metadata["supports_projected_costs"],
		"Valid capability should appear in legacy metadata")
	// The invalid value (999) has no legacy mapping, so it should be silently ignored
	assert.Len(t, metadata, 1,
		"Only valid capabilities should appear in legacy metadata")
}

// TestGetPluginInfoCapabilitiesOverrideTakesPrecedence verifies that explicit
// capability override takes precedence over auto-discovered capabilities.
// This tests the scenario where a plugin implements optional interfaces
// but the PluginInfo explicitly declares a subset of capabilities.
func TestGetPluginInfoCapabilitiesOverrideTakesPrecedence(t *testing.T) {
	// Create plugin that implements ALL optional interfaces (DryRun, Recommendations, etc.)
	testPlugin := &capabilityTestPlugin{
		name: "override-precedence-test-plugin",
	}

	// Override with ONLY base capabilities - explicitly excluding optional interfaces
	pluginInfo := NewPluginInfo("override-precedence-test-plugin", "v1.0.0",
		WithCapabilities(
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		))
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Override should take precedence - only explicitly set capabilities
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
	}
	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilities(),
		"Override capabilities should take precedence over auto-discovery")

	// Verify excluded capabilities are NOT present despite being implemented
	assert.NotContains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
		"DRY_RUN should be excluded even though plugin implements DryRunHandler")
	assert.NotContains(t, resp.GetCapabilities(),
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		"RECOMMENDATIONS should be excluded even though plugin implements RecommendationsProvider")

	// Legacy metadata should also reflect only the overridden capabilities
	metadata := resp.GetMetadata()
	assert.Equal(t, "true", metadata["supports_projected_costs"])
	assert.Equal(t, "true", metadata["supports_actual_costs"])
	assert.NotContains(t, metadata, "supports_dry_run",
		"dry_run should not be in legacy metadata")
	assert.NotContains(t, metadata, "supports_recommendations",
		"recommendations should not be in legacy metadata")
}

// TestGetPluginInfoEmptyPluginEmptyOverride verifies that a minimal plugin
// with empty override returns only base capabilities.
func TestGetPluginInfoEmptyPluginEmptyOverride(t *testing.T) {
	// Create plugin with NO optional interfaces implemented
	testPlugin := &conformanceMockPlugin{
		name: "empty-empty-test-plugin",
	}

	// Empty capabilities override on minimal plugin
	pluginInfo := NewPluginInfo("empty-empty-test-plugin", "v1.0.0",
		WithCapabilities())
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Empty override with minimal plugin should return base capabilities only
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	}

	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilities(),
		"Empty override with minimal plugin should return only base capabilities")
}

// mapKeys is a helper to extract keys from a string map for comparison.
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// =============================================================================
// Nil Safety Tests
// =============================================================================

// TestInferCapabilitiesNilPlugin verifies that inferCapabilities handles nil
// plugin input defensively without panicking.
//
// This test ensures robustness in edge cases where a nil plugin might be
// passed during error recovery or testing scenarios.
func TestInferCapabilitiesNilPlugin(t *testing.T) {
	// Should not panic and should return nil
	result := inferCapabilities(nil)
	assert.Nil(t, result, "inferCapabilities should return nil for nil plugin")
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// BenchmarkInferCapabilitiesDirect measures the performance of capability discovery
// by calling inferCapabilities() directly. This verifies that type assertions are
// zero-allocation and slice operations are efficient.
//
// Expected performance for plugins with all optional interfaces:
// - < 100 ns/op: Type assertions and slice operations are fast
// - 1 alloc/op: Single allocation for capability slice
//
// These benchmarks establish performance contracts that can be validated in CI
// to prevent regressions in capability discovery performance.
//
// Note: This differs from BenchmarkInferCapabilitiesRPC in plugin_info_benchmark_test.go
// which measures the full GetPluginInfo RPC flow including server overhead.
func BenchmarkInferCapabilitiesDirect(b *testing.B) {
	plugin := &capabilityTestPlugin{name: "benchmark-plugin"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = inferCapabilities(plugin)
	}
}

// BenchmarkInferCapabilitiesMinimal benchmarks capability discovery for a plugin
// with no optional interfaces (base capabilities only).
func BenchmarkInferCapabilitiesMinimal(b *testing.B) {
	plugin := &conformanceMockPlugin{name: "benchmark-minimal-plugin"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = inferCapabilities(plugin)
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

// TestGetPluginInfoContextCancellation verifies that GetPluginInfo handles
// context cancellation gracefully. Since GetPluginInfo doesn't perform blocking
// operations, it should still succeed even with a cancelled context.
//
// Rationale for succeeding with cancelled context:
//  1. It only reads pre-constructed PluginInfo (no I/O operations)
//  2. Execution time is guaranteed < 1ms (simple field copies)
//  3. Failing fast would prevent legitimate metadata queries during shutdown
//
// This behavior is intentional and differs from RPCs that perform external calls.
func TestGetPluginInfoContextCancellation(t *testing.T) {
	testPlugin := &capabilityTestPlugin{name: "timeout-test-plugin"}
	pluginInfo := NewPluginInfo("timeout-test-plugin", "v1.0.0")
	server := NewServerWithOptions(testPlugin, nil, nil, pluginInfo)

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &pbc.GetPluginInfoRequest{}
	resp, err := server.GetPluginInfo(ctx, req)

	// GetPluginInfo should still succeed even with cancelled context
	// since it doesn't perform blocking operations
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "timeout-test-plugin", resp.GetName())
}

// =============================================================================
// Capability Validation Tests
// =============================================================================

// TestIsValidCapability verifies the IsValidCapability helper function
// correctly identifies valid and invalid PluginCapability enum values.
func TestIsValidCapability(t *testing.T) {
	tests := []struct {
		name       string
		capability pbc.PluginCapability
		wantValid  bool
	}{
		// Valid capabilities (all currently defined values)
		{"PROJECTED_COSTS", pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS, true},
		{"ACTUAL_COSTS", pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS, true},
		{"CARBON", pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON, true},
		{"RECOMMENDATIONS", pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS, true},
		{"DRY_RUN", pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN, true},
		{"BUDGETS", pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS, true},
		{"ENERGY", pbc.PluginCapability_PLUGIN_CAPABILITY_ENERGY, true},
		{"WATER", pbc.PluginCapability_PLUGIN_CAPABILITY_WATER, true},
		{"PRICING_SPEC", pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC, true},
		{"ESTIMATE_COST", pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST, true},
		{"DISMISS_RECOMMENDATIONS", pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS, true},
		{"BATCH_COST", pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST, true},

		// Invalid capabilities
		{"UNSPECIFIED (0)", pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, false},
		{"negative value (-1)", pbc.PluginCapability(-1), false},
		{"out of range (999)", pbc.PluginCapability(999), false},
		{"just above max (13)", pbc.PluginCapability(13), false},
		{"very large value", pbc.PluginCapability(1000000), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidCapability(tt.capability)
			assert.Equal(t, tt.wantValid, got, "IsValidCapability(%d)", int32(tt.capability))
		})
	}
}

// TestPluginInfoValidateDoSProtection verifies that the PluginInfo.Validate
// method rejects capability slices that exceed the DoS protection limit.
func TestPluginInfoValidateDoSProtection(t *testing.T) {
	t.Run("accepts capabilities at max limit", func(t *testing.T) {
		caps := make([]pbc.PluginCapability, MaxConfiguredCapabilities)
		for i := range caps {
			caps[i] = pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS
		}
		info := NewPluginInfo("test-plugin", "v1.0.0", WithCapabilities(caps...))
		err := info.Validate()
		assert.NoError(t, err, "Should accept capabilities at max limit (%d)", MaxConfiguredCapabilities)
	})

	t.Run("rejects capabilities exceeding max limit", func(t *testing.T) {
		caps := make([]pbc.PluginCapability, MaxConfiguredCapabilities+1)
		for i := range caps {
			caps[i] = pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS
		}
		info := NewPluginInfo("test-plugin", "v1.0.0", WithCapabilities(caps...))
		err := info.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many capabilities")
		assert.Contains(t, err.Error(), "test-plugin")
	})

	t.Run("rejects large capability slice (DoS scenario)", func(t *testing.T) {
		// Simulate DoS attack with excessive capabilities
		caps := make([]pbc.PluginCapability, 10000)
		for i := range caps {
			caps[i] = pbc.PluginCapability(999)
		}
		info := NewPluginInfo("malicious-plugin", "v1.0.0", WithCapabilities(caps...))
		err := info.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many capabilities")
	})
}

// TestCapabilitiesToLegacyMetadataWithWarnings verifies that the warning-aware
// conversion function correctly identifies and reports invalid/unmapped capabilities.
func TestCapabilitiesToLegacyMetadataWithWarnings(t *testing.T) {
	t.Run("all valid capabilities - no warnings", func(t *testing.T) {
		caps := []pbc.PluginCapability{
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		}
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings(caps)

		assert.Empty(t, warnings, "Expected no warnings for valid capabilities")
		assert.Len(t, metadata, 3)
		assert.Equal(t, "true", metadata["supports_projected_costs"])
		assert.Equal(t, "true", metadata["supports_actual_costs"])
		assert.Equal(t, "true", metadata["supports_recommendations"])
	})

	t.Run("mixed valid and invalid capabilities", func(t *testing.T) {
		caps := []pbc.PluginCapability{
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability(999), // Invalid
			pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
			pbc.PluginCapability(500), // Invalid
		}
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings(caps)

		assert.Len(t, warnings, 2, "Expected 2 warnings for invalid capabilities")
		assert.Len(t, metadata, 2, "Only valid capabilities should be in metadata")
		assert.Equal(t, "true", metadata["supports_projected_costs"])
		assert.Equal(t, "true", metadata["supports_recommendations"])

		// Verify warning details
		for _, w := range warnings {
			assert.Contains(t, w.Reason, "invalid capability enum value")
		}
	})

	t.Run("all invalid capabilities", func(t *testing.T) {
		caps := []pbc.PluginCapability{
			pbc.PluginCapability(100),
			pbc.PluginCapability(200),
			pbc.PluginCapability(999),
		}
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings(caps)

		assert.Len(t, warnings, 3, "Expected 3 warnings for all invalid capabilities")
		assert.Empty(t, metadata, "No valid capabilities should be in metadata")
	})

	t.Run("UNSPECIFIED is ignored without warning", func(t *testing.T) {
		caps := []pbc.PluginCapability{
			pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		}
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings(caps)

		assert.Empty(t, warnings, "UNSPECIFIED should not generate warnings")
		assert.Len(t, metadata, 1)
		assert.Equal(t, "true", metadata["supports_projected_costs"])
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings([]pbc.PluginCapability{})
		assert.Nil(t, metadata)
		assert.Nil(t, warnings)
	})

	t.Run("nil slice returns nil", func(t *testing.T) {
		metadata, warnings := CapabilitiesToLegacyMetadataWithWarnings(nil)
		assert.Nil(t, metadata)
		assert.Nil(t, warnings)
	})
}

// TestInferCapabilitiesConcurrentNilSafety verifies that inferCapabilities
// handles nil plugin safely even under concurrent access.
func TestInferCapabilitiesConcurrentNilSafety(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Ensure no panics occur during concurrent nil access
	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				result := inferCapabilities(nil)
				assert.Nil(t, result)
			}
		}()
	}

	wg.Wait()
}

// BenchmarkIsValidCapability benchmarks the capability validation function.
// Expected performance: < 5 ns/op, 0 allocs/op (simple comparison).
func BenchmarkIsValidCapability(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = IsValidCapability(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS)
	}
}

// BenchmarkCapabilitiesToLegacyMetadataWithWarnings benchmarks the warning-aware
// conversion function with a mix of valid and invalid capabilities.
func BenchmarkCapabilitiesToLegacyMetadataWithWarnings(b *testing.B) {
	caps := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability(999), // Invalid
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = CapabilitiesToLegacyMetadataWithWarnings(caps)
	}
}
