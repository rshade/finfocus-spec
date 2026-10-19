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

package testing_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func costParseMockRequest() *pbc.ParseRequest {
	return &pbc.ParseRequest{
		Format: "fixture",
		Input: &pbc.ParseRequest_Files{Files: &pbc.ParseFileSet{Files: []*pbc.ParseFile{{
			Path: "input.fixture", Content: []byte("resource First\nresource Second\n"),
		}}}},
	}
}

func TestCostParseMockFixtures(t *testing.T) {
	ctx := context.Background()
	parser := pt.NewMockCostParser()
	info, err := parser.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	limits, err := pt.ParseParseLimits(info.GetMetadata())
	require.NoError(t, err, "parser discovery limits")
	require.NotNil(t, limits)
	require.Contains(t, limits.Formats, "fixture")
	require.Contains(t, limits.InputModes, "content")
	require.Contains(t, limits.InputModes, "path")
	require.Positive(t, limits.MaxInputBytes)

	root := t.TempDir()
	content := costParseMockRequest().GetFiles().GetFiles()[0].GetContent()
	require.NoError(t, os.WriteFile(filepath.Join(root, "input.fixture"), content, 0o600))
	var baseline *pbc.ParseResponse
	for _, mode := range []string{"content", "path"} {
		t.Run(mode, func(t *testing.T) {
			req := costParseMockRequest()
			if mode == "path" {
				req.Input = &pbc.ParseRequest_RootPath{RootPath: root}
			}
			before := proto.Clone(req)
			for attempt := range 100 {
				resp, parseErr := parser.Parse(ctx, req)
				require.NoError(t, parseErr, "attempt %d", attempt)
				require.NoError(t, pt.ValidateParseResponse(req, resp), "invalid parse response")
				require.Len(t, resp.GetResources(), 2)
				require.Empty(t, resp.GetWarnings())
				for i, source := range []string{"First", "Second"} {
					parsed := resp.GetResources()[i]
					require.Equal(t, source, parsed.GetSourceId())
					require.Equal(t, "custom", parsed.GetResource().GetProvider())
					require.Equal(t, "custom:compute:Instance", parsed.GetResource().GetResourceType())
				}
				// Freshness may change independently of resource and warning determinism.
				stable := proto.CloneOf(resp)
				stable.ExpiresAt = nil
				if baseline == nil {
					baseline = stable
				}
				require.True(t, proto.Equal(baseline, stable), "fixture result changed across calls or modes")
			}
			require.True(t, proto.Equal(req, before), "Parse mutated caller input")
		})
	}
}

func TestCostParseMockClonesResource(t *testing.T) {
	attributes := &structpb.Struct{Fields: map[string]*structpb.Value{
		"Sizing": structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"Cpu": structpb.NewNumberValue(2),
		}}),
	}}
	resource := &pbc.ResourceDescriptor{
		Provider: "custom", ResourceType: "custom:compute:Instance",
		Tags: map[string]string{"environment": "test"}, Attributes: attributes,
	}
	want := proto.CloneOf(resource)
	parser := pt.NewMockCostParser(pt.WithParseResource(resource))
	resource.Tags["environment"] = "changed"
	resource.Attributes.Fields["Sizing"].GetStructValue().Fields["Cpu"] = structpb.NewNumberValue(99)
	resp, err := parser.Parse(context.Background(), costParseMockRequest())
	require.NoError(t, err)
	if len(resp.GetResources()) != 2 {
		t.Fatalf("expected two fixture resources: %v", resp)
	}
	for _, parsed := range resp.GetResources() {
		if !proto.Equal(parsed.GetResource(), want) {
			t.Fatalf("parser retained caller-owned resource state: %v", parsed)
		}
	}
	resp.Resources[0].Resource.Tags["environment"] = "response mutation"
	resp.Resources[0].Resource.Attributes.Fields["Sizing"].GetStructValue().Fields["Cpu"] = structpb.NewNumberValue(100)
	require.True(
		t,
		proto.Equal(resp.GetResources()[1].GetResource(), want),
		"resources within a response share mutable state",
	)
	again, err := parser.Parse(context.Background(), costParseMockRequest())
	require.NoError(t, err)
	require.True(
		t,
		proto.Equal(again.GetResources()[0].GetResource(), want),
		"response mutation changed subsequent fixture results",
	)
}

// costParsePricingProbe observes accidental host-side type resolution while
// retaining the existing pricing mock's implementation.
type costParsePricingProbe struct {
	*pt.MockPlugin

	resolveCalls int
}

func (m *costParsePricingProbe) Name() string { return "pricing-probe" }

func (m *costParsePricingProbe) ResolveResourceTypes(
	ctx context.Context, req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	m.resolveCalls++
	return m.MockPlugin.ResolveResourceTypes(ctx, req)
}

func TestCostParseMockDirectPricingCompatibility(t *testing.T) {
	ctx := context.Background()
	pricing := &costParsePricingProbe{MockPlugin: pt.NewMockPlugin()}
	parser := pt.NewMockCostParser()
	resp, err := parser.Parse(ctx, costParseMockRequest())
	require.NoError(t, err)
	if len(resp.GetResources()) == 0 {
		t.Fatal("fixture returned no descriptors to price")
	}
	for _, parsed := range resp.GetResources() {
		cost, priceErr := pricing.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: parsed.GetResource()})
		if priceErr != nil || cost == nil || cost.GetCostPerMonth() <= 0 {
			t.Fatalf("source %q is not directly priceable: cost=%v err=%v", parsed.GetSourceId(), cost, priceErr)
		}
	}
	if pricing.resolveCalls != 0 {
		t.Fatalf("direct pricing made %d type-resolution calls", pricing.resolveCalls)
	}
	if _, ok := any(pricing.MockPlugin).(interface {
		Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error)
	}); ok {
		t.Fatal("existing MockPlugin unexpectedly implements parsing")
	}
	info, err := pluginsdk.NewServer(pricing).GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	want := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES,
	}
	if !slices.Equal(info.GetCapabilities(), want) {
		t.Fatalf("existing MockPlugin capabilities changed: got %v want %v", info.GetCapabilities(), want)
	}
}
