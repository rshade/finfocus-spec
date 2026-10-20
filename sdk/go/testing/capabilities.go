package testing

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// capabilityProfile snapshots discovery once per harness. An empty capability
// list or an Unimplemented discovery RPC retains the legacy suite contract.
type capabilityProfile struct {
	once   sync.Once
	values []pbc.PluginCapability
	err    error
}

func (h *TestHarness) discoverCapabilities() {
	h.capabilities.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeoutSeconds*time.Second)
		defer cancel()
		info, err := h.Client().GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		if status.Code(err) == codes.Unimplemented {
			return
		}
		if err != nil {
			h.capabilities.err = fmt.Errorf("discover capabilities: %w", err)
			return
		}
		h.capabilities.values = slices.Clone(info.GetCapabilities())
	})
}

func (p *capabilityProfile) declares(capability pbc.PluginCapability) bool {
	return slices.Contains(p.values, capability)
}

// withCapability keeps each check mandatory: declared capabilities run their
// original validation, and undeclared capabilities must return Unimplemented.
// Empty metadata preserves the pre-discovery requirements for legacy plugins.
func withCapability(
	capability pbc.PluginCapability,
	check func(*TestHarness) TestResult,
) func(*TestHarness) TestResult {
	return func(h *TestHarness) TestResult {
		h.discoverCapabilities()
		if h.capabilities.err != nil {
			return TestResult{Method: MethodGetPluginInfo, Error: h.capabilities.err,
				Details: "Capability discovery failed"}
		}
		if len(h.capabilities.values) == 0 || h.capabilities.declares(capability) {
			return check(h)
		}
		return checkUnsupportedCapability(h, capability)
	}
}

func checkUnsupportedCapability(h *TestHarness, capability pbc.PluginCapability) TestResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeoutSeconds*time.Second)
	defer cancel()
	resource := h.SampleResource()
	var method string
	var err error
	// Only the four capabilities covered by this suite have RPC probes.
	//exhaustive:ignore
	switch capability {
	case pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS:
		method = MethodGetActualCost
		from, to := CreateTimeRange(HoursPerDay)
		_, err = h.Client().GetActualCost(ctx, &pbc.GetActualCostRequest{
			ResourceId: testResourceID, Resource: resource, Start: from, End: to,
		})
	case pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS:
		method = MethodGetProjectedCost
		_, err = h.Client().GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: resource})
	case pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC:
		method = MethodGetPricingSpec
		_, err = h.Client().GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
	case pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS:
		method = MethodGetBudgets
		_, err = h.Client().GetBudgets(ctx, &pbc.GetBudgetsRequest{Filter: &pbc.BudgetFilter{}})
	default:
		return TestResult{Error: fmt.Errorf("no unsupported RPC check for %s", capability)}
	}
	result := TestResult{Method: method, Duration: time.Since(start)}
	if status.Code(err) == codes.Unimplemented {
		result.Success = true
		result.Details = "Undeclared capability correctly returned Unimplemented"
	} else {
		result.Error = fmt.Errorf("undeclared %s returned %s, want Unimplemented", capability, status.Code(err))
		result.Details = "Unsupported capability returned an incorrect protocol status"
	}
	return result
}
