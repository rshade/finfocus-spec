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
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// capabilityRecordingPlugin records the capability of each Supports request and can
// be told to misbehave for one capability.
type capabilityRecordingPlugin struct {
	*plugintesting.MockPlugin

	declared []pbc.PluginCapability
	infoErr  error
	errorOn  []pbc.PluginCapability
	noReason []pbc.PluginCapability

	mu   sync.Mutex
	seen []pbc.PluginCapability
}

func newCapabilityRecordingPlugin(declared ...pbc.PluginCapability) *capabilityRecordingPlugin {
	mock := plugintesting.NewMockPlugin()
	mock.SupportedProviders = []string{customProvider}
	mock.SupportedResources = map[string][]string{customProvider: {"instance"}}
	return &capabilityRecordingPlugin{MockPlugin: mock, declared: declared}
}

func (p *capabilityRecordingPlugin) Supports(
	ctx context.Context, req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	capability := req.GetCapability()
	p.mu.Lock()
	p.seen = append(p.seen, capability)
	p.mu.Unlock()
	if slices.Contains(p.errorOn, capability) {
		return nil, status.Error(codes.Internal, "backend down")
	}
	if slices.Contains(p.noReason, capability) {
		return &pbc.SupportsResponse{Supported: false}, nil
	}
	return p.MockPlugin.Supports(ctx, req)
}

func (p *capabilityRecordingPlugin) GetPluginInfo(
	ctx context.Context, req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	if p.infoErr != nil {
		return nil, p.infoErr
	}
	resp, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Capabilities = p.declared
	return resp, nil
}

func (p *capabilityRecordingPlugin) recorded() []pbc.PluginCapability {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]pbc.PluginCapability(nil), p.seen...)
}

// TestRPCCorrectness_SupportsCapability checks that the capability-specific Supports
// check passes plugins that ignore or honor the field, asks the right capabilities,
// and fails plugins that error or answer invalidly, naming the capability.
func TestRPCCorrectness_SupportsCapability(t *testing.T) {
	var check plugintesting.ConformanceSuiteTest
	for _, candidate := range plugintesting.RPCCorrectnessTests() {
		if candidate.Name == "RPCCorrectness_SupportsCapability" {
			check = candidate
		}
	}
	require.NotNil(t, check.TestFunc, "RPCCorrectness_SupportsCapability is not registered")
	assert.Equal(t, plugintesting.ConformanceLevelStandard, check.MinLevel)
	assert.Equal(t, plugintesting.CategoryRPCCorrectness, check.Category)

	unspecified := pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED
	projected := pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS
	actual := pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS
	carbon := pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON
	legacyQueries := []pbc.PluginCapability{unspecified, projected, actual}

	perCapability := plugintesting.NewMockPlugin()
	perCapability.UnsupportedCapabilities = []pbc.PluginCapability{projected}

	cases := []struct {
		name        string
		plugin      pbc.CostSourceServiceServer
		wantSuccess bool
		wantInError string
		wantQueries []pbc.PluginCapability
	}{
		{name: "default mock ignoring the field passes", plugin: plugintesting.NewMockPlugin(), wantSuccess: true},
		{name: "mock answering per capability passes", plugin: perCapability, wantSuccess: true},
		{
			name: "legacy plugin is asked projected and actual cost", plugin: newCapabilityRecordingPlugin(),
			wantSuccess: true, wantQueries: legacyQueries,
		},
		{
			name:        "declared resource-scoped capabilities are asked, others are not",
			plugin:      newCapabilityRecordingPlugin(carbon, pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS),
			wantSuccess: true, wantQueries: []pbc.PluginCapability{unspecified, carbon},
		},
		{
			name:        "plugin declaring only plugin-level capabilities gets the unspecified request",
			plugin:      newCapabilityRecordingPlugin(pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN),
			wantSuccess: true, wantQueries: []pbc.PluginCapability{unspecified},
		},
		{
			name: "error on a capability-specific request fails and names it",
			plugin: func() pbc.CostSourceServiceServer {
				p := newCapabilityRecordingPlugin()
				p.errorOn = []pbc.PluginCapability{projected}
				return p
			}(),
			wantInError: "PLUGIN_CAPABILITY_PROJECTED_COSTS",
		},
		{
			name: "unsupported answer without a reason fails and names it",
			plugin: func() pbc.CostSourceServiceServer {
				p := newCapabilityRecordingPlugin()
				p.noReason = []pbc.PluginCapability{actual}
				return p
			}(),
			wantInError: "PLUGIN_CAPABILITY_ACTUAL_COSTS",
		},
		{
			name: "capability discovery failure fails",
			plugin: func() pbc.CostSourceServiceServer {
				p := newCapabilityRecordingPlugin()
				p.infoErr = status.Error(codes.Internal, "info down")
				return p
			}(),
			wantInError: "discover capabilities",
		},
		{
			name: "unimplemented capability discovery is treated as legacy",
			plugin: func() pbc.CostSourceServiceServer {
				p := newCapabilityRecordingPlugin()
				p.infoErr = status.Error(codes.Unimplemented, "no info")
				return p
			}(),
			wantSuccess: true, wantQueries: legacyQueries,
		},
		{
			name: "error on the unspecified request fails",
			plugin: func() pbc.CostSourceServiceServer {
				p := newCapabilityRecordingPlugin()
				p.errorOn = []pbc.PluginCapability{unspecified}
				return p
			}(),
			wantInError: "PLUGIN_CAPABILITY_UNSPECIFIED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			harness := plugintesting.NewTestHarness(tc.plugin)
			harness.Start(t)
			defer harness.Stop()
			plugintesting.SetSampleResourceForTest(harness, customSample())

			result := check.TestFunc(harness)
			require.Equal(t, tc.wantSuccess, result.Success, "error: %v, details: %s", result.Error, result.Details)
			if tc.wantInError != "" {
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantInError)
			}
			if recorder, ok := tc.plugin.(*capabilityRecordingPlugin); ok && tc.wantQueries != nil {
				assert.Equal(t, tc.wantQueries, recorder.recorded())
			}
		})
	}
}
