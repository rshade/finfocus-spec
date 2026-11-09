// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"context"
	"fmt"
	"slices"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// resourceScopedCapabilities are the capabilities a host asks Supports about for a
// resource. The others describe plugin-level or dataset services. The list is
// mirrored in the Supports section of PLUGIN_DEVELOPER_GUIDE.md.
//
//nolint:gochecknoglobals // Read-only lookup list shared by the check and its query builder.
var resourceScopedCapabilities = []pbc.PluginCapability{
	pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
	pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
	pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON,
	pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
	pbc.PluginCapability_PLUGIN_CAPABILITY_ENERGY,
	pbc.PluginCapability_PLUGIN_CAPABILITY_WATER,
	pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
	pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
	pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON,
}

// supportsCapabilityTest returns the Standard check for capability-scoped Supports requests.
func supportsCapabilityTest() ConformanceSuiteTest {
	return ConformanceSuiteTest{
		Name: "RPCCorrectness_SupportsCapability",
		Description: "Validates that Supports answers an unspecified request and a request for each " +
			"declared resource-scoped capability with a valid response; answer values are not judged",
		Category: CategoryRPCCorrectness,
		MinLevel: ConformanceLevelStandard,
		TestFunc: testSupportsCapability,
	}
}

// testSupportsCapability sends an unspecified Supports request, then one request per
// declared resource-scoped capability, or projected and actual cost for a plugin that
// declares none. Each answer must be a valid response; a failure names the capability.
func testSupportsCapability(harness *TestHarness) TestResult {
	start := time.Now()
	harness.discoverCapabilities()
	if harness.capabilities.err != nil {
		return TestResult{Method: MethodGetPluginInfo, Category: CategoryRPCCorrectness,
			Error: harness.capabilities.err, Duration: time.Since(start), Details: detailsCapabilityDiscoveryFailed}
	}

	queries := supportsCapabilityQueries(harness.capabilities.values)
	resource := harness.SampleResource()
	for _, capability := range queries {
		if err := askSupportsFor(harness, resource, capability); err != nil {
			return TestResult{Method: MethodSupports, Category: CategoryRPCCorrectness, Error: err,
				Duration: time.Since(start), Details: fmt.Sprintf("Supports failed for %s", capability)}
		}
	}
	return TestResult{Method: MethodSupports, Category: CategoryRPCCorrectness, Success: true,
		Duration: time.Since(start), Details: fmt.Sprintf("Supports answered %d capability requests", len(queries))}
}

// supportsCapabilityQueries returns the capabilities the check asks about, unspecified first.
func supportsCapabilityQueries(declared []pbc.PluginCapability) []pbc.PluginCapability {
	queries := []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED}
	if len(declared) == 0 {
		return append(queries,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS)
	}
	for _, capability := range resourceScopedCapabilities {
		if slices.Contains(declared, capability) {
			queries = append(queries, capability)
		}
	}
	return queries
}

func askSupportsFor(harness *TestHarness, resource *pbc.ResourceDescriptor, capability pbc.PluginCapability) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeoutSeconds*time.Second)
	defer cancel()
	resp, err := harness.Client().Supports(ctx, &pbc.SupportsRequest{Resource: resource, Capability: capability})
	if err != nil {
		return fmt.Errorf("%s: %w", capability, err)
	}
	if valErr := ValidateSupportsResponse(resp); valErr != nil {
		return fmt.Errorf("%s: %w", capability, valErr)
	}
	return nil
}
