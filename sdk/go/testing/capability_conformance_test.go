package testing_test

import (
	"context"
	"slices"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// capabilityPlugin exposes a declared surface independently from the RPCs it implements.
// The suite must reject both false advertisements and incorrect unsupported statuses.
type capabilityPlugin struct {
	*plugintesting.MockPlugin

	capabilities      []pbc.PluginCapability
	implemented       []pbc.PluginCapability
	failureCapability pbc.PluginCapability
	failureCode       codes.Code
	infoError         codes.Code
}

func (p *capabilityPlugin) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	if p.infoError != codes.OK {
		return nil, status.Error(p.infoError, "discovery failure")
	}
	info, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if info != nil {
		info.Capabilities = p.capabilities
	}
	return info, err
}

func (p *capabilityPlugin) rpcError(capability pbc.PluginCapability) error {
	if p.failureCapability == capability && p.failureCode != codes.OK {
		return status.Error(p.failureCode, "RPC failure")
	}
	if !slices.Contains(p.implemented, capability) {
		return status.Error(codes.Unimplemented, "unsupported capability")
	}
	return nil
}

func (p *capabilityPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if err := p.rpcError(pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetActualCost(ctx, req)
}

func (p *capabilityPlugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if err := p.rpcError(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetProjectedCost(ctx, req)
}

func (p *capabilityPlugin) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if err := p.rpcError(pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetPricingSpec(ctx, req)
}

func (p *capabilityPlugin) GetBudgets(
	ctx context.Context,
	req *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	if err := p.rpcError(pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetBudgets(ctx, req)
}

func TestConformanceDeclaredCapabilities(t *testing.T) {
	capabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
	}
	for _, capability := range capabilities {
		t.Run(capability.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
				capabilities: []pbc.PluginCapability{capability},
				implemented:  []pbc.PluginCapability{capability}}
			checkCapabilityConformance(t, plugin, true)
		})
		t.Run("false advertisement/"+capability.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
				capabilities: capabilities, implemented: capabilitiesExcept(capabilities, capability)}
			checkCapabilityConformance(t, plugin, false)
		})
		t.Run("unsupported error/"+capability.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
				capabilities:      capabilitiesExcept(capabilities, capability),
				implemented:       capabilitiesExcept(capabilities, capability),
				failureCapability: capability, failureCode: codes.Internal}
			checkCapabilityConformance(t, plugin, false)
		})
		t.Run("undeclared success/"+capability.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
				capabilities: capabilitiesExcept(capabilities, capability),
				implemented:  capabilitiesExcept(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED)}
			checkCapabilityConformance(t, plugin, false)
		})
	}
}

func TestConformanceLegacyCapabilities(t *testing.T) {
	for _, infoError := range []codes.Code{codes.OK, codes.Unimplemented} {
		t.Run(infoError.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(), infoError: infoError,
				implemented: []pbc.PluginCapability{
					pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
					pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
					pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
				}}
			result, err := plugintesting.RunStandardConformance(plugin)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Passed() {
				t.Fatalf("legacy plugin failed: %s", plugintesting.FormatCategoryResults(result.Categories))
			}
			plugin.implemented = capabilitiesExcept(
				plugin.implemented,
				pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			)
			result, err = plugintesting.RunStandardConformance(plugin)
			if err != nil {
				t.Fatal(err)
			}
			if result.Passed() {
				t.Fatal("missing capability metadata suppressed legacy mandatory pricing checks")
			}
		})
	}
}

func TestConformanceCapabilityDiscoveryFailure(t *testing.T) {
	plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(), infoError: codes.Unavailable,
		implemented: []pbc.PluginCapability{
			pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		}}
	result, err := plugintesting.RunStandardConformance(plugin)
	if err == nil && result.Passed() {
		t.Fatal("discovery failure was treated as legacy capability metadata")
	}
}

func capabilitiesExcept(all []pbc.PluginCapability, except pbc.PluginCapability) []pbc.PluginCapability {
	var result []pbc.PluginCapability
	for _, capability := range all {
		if capability != except {
			result = append(result, capability)
		}
	}
	return result
}

func checkCapabilityConformance(t *testing.T, plugin *capabilityPlugin, wantPass bool) {
	t.Helper()
	result, err := plugintesting.RunStandardConformance(plugin)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() != wantPass {
		t.Fatalf("conformance passed=%v, want %v: %s", result.Passed(), wantPass,
			plugintesting.FormatCategoryResults(result.Categories))
	}
	// Only the two Advanced checks may be skipped. Unsupported methods
	// must be checked for Unimplemented, not excluded from the suite.
	if result.Summary.Skipped != 2 {
		t.Fatalf("unexpected skipped checks: %+v", result.Summary)
	}
}

func TestBasicConformanceRejectsFalseActualCapability(t *testing.T) {
	plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
		capabilities: []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS}}
	result, err := plugintesting.RunBasicConformance(plugin)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() {
		t.Fatal("Basic accepted advertised actual costs returning Unimplemented")
	}
}

func TestCapabilityPerformanceRejectsRPCFailures(t *testing.T) {
	for _, capability := range []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
	} {
		t.Run(capability.String(), func(t *testing.T) {
			plugin := &capabilityPlugin{MockPlugin: plugintesting.NewMockPlugin(),
				capabilities:      []pbc.PluginCapability{capability},
				implemented:       []pbc.PluginCapability{capability},
				failureCapability: capability, failureCode: codes.Internal}
			suite := plugintesting.NewConformanceSuite()
			plugintesting.RegisterPerformanceTests(suite)
			result, err := suite.RunCategory(plugin, plugintesting.CategoryPerformance)
			if err != nil {
				t.Fatal(err)
			}
			if result.Failed == 0 {
				t.Fatal("a fast failing advertised RPC passed its latency check")
			}
		})
	}
}
