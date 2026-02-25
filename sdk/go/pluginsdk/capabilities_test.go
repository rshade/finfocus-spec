//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// mockCapabilityPlugin implements Plugin and optional interfaces.
type mockCapabilityPlugin struct {
	// Satisfy Plugin interface
}

func (m *mockCapabilityPlugin) Name() string { return "capability-test" }

func (m *mockCapabilityPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (m *mockCapabilityPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (m *mockCapabilityPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (m *mockCapabilityPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

// Implement RecommendationsProvider.
func (m *mockCapabilityPlugin) GetRecommendations(
	_ context.Context,
	_ *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return &pbc.GetRecommendationsResponse{}, nil
}

// Implement BudgetsProvider.
func (m *mockCapabilityPlugin) GetBudgets(
	_ context.Context,
	_ *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	return &pbc.GetBudgetsResponse{}, nil
}

// Implement DismissProvider.
func (m *mockCapabilityPlugin) DismissRecommendation(
	_ context.Context,
	_ *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	return &pbc.DismissRecommendationResponse{}, nil
}

// Implement DryRunHandler.
func (m *mockCapabilityPlugin) HandleDryRun(
	_ *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	return &pbc.DryRunResponse{}, nil
}

// Implement BatchCostHandler.
func (m *mockCapabilityPlugin) BatchCost(
	_ context.Context,
	_ *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	return &pbc.BatchCostResponse{}, nil
}

func TestGetPluginInfo_AutoDiscovery(t *testing.T) {
	plugin := &mockCapabilityPlugin{}
	// Create server with minimal PluginInfo to trigger default logic
	info := &PluginInfo{Name: "test", Version: "1.0.0", SpecVersion: "1.0.0"}
	server := NewServerWithOptions(plugin, nil, nil, info)

	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)

	// Verify Capabilities Enum - all optional interfaces
	caps := resp.GetCapabilities()
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS)
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS)
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
	// Core capabilities
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS)
	assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS)

	// Verify Legacy Metadata - optional interfaces
	meta := resp.GetMetadata()
	assert.Equal(t, "true", meta["supports_recommendations"])
	assert.Equal(t, "true", meta["supports_budgets"])
	assert.Equal(t, "true", meta["supports_dismiss_recommendations"])
	assert.Equal(t, "true", meta["supports_dry_run"])
	assert.Equal(t, "true", meta["supports_batch_cost"])
	assert.Equal(t, "100", meta["max_batch_size"])
}
