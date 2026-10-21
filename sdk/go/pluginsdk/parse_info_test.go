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

package pluginsdk_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

type costParseInfoPlugin struct {
	*pluginsdk.BasePlugin
}

var _ pluginsdk.CostParseProvider = (*costParseInfoPlugin)(nil)

func (*costParseInfoPlugin) Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return &pbc.ParseResponse{}, nil
}

func TestCostParseInfoMetadataGate(t *testing.T) {
	valid := map[string]string{
		"parse_formats": "fixture", "parse_input_modes": "content,path", "parse_max_input_bytes": "1024",
	}
	type metadataCase struct {
		name       string
		metadata   map[string]string
		advertised bool
		valid      bool
	}
	cases := []metadataCase{
		{"existing plugin without limits", nil, false, true},
		{"advertised parser without limits", nil, true, false},
		{"advertised complete limits", valid, true, true},
		{"unadvertised complete limits", valid, false, true},
		{"unrelated metadata", map[string]string{"build": "test"}, false, true},
	}
	for _, key := range []string{"parse_formats", "parse_input_modes", "parse_max_input_bytes"} {
		missing := make(map[string]string, len(valid))
		for k, v := range valid {
			if k != key {
				missing[k] = v
			}
		}
		cases = append(cases,
			metadataCase{
				"advertised missing " + key, missing, true, false,
			},
			metadataCase{
				"unadvertised missing " + key, missing, false, false,
			},
		)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []pluginsdk.PluginInfoOption{pluginsdk.WithMetadataMap(tc.metadata)}
			if tc.advertised {
				opts = append(opts, pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE))
			}
			info := pluginsdk.NewPluginInfo("parser", "v1.0.0", opts...)
			err := info.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, plugintesting.ErrInvalidParseLimits)
			}
		})
	}
}

func TestCostParseInfoWithParseLimits(t *testing.T) {
	info := pluginsdk.NewPluginInfo("parser", "v1.0.0",
		pluginsdk.WithMetadata("build", "test"),
		pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE),
		pluginsdk.WithParseLimits([]string{"fixture", "fixture-v2"}, []string{"content", "path"}, 1024),
	)
	require.NoError(t, info.Validate())
	require.Equal(t, map[string]string{
		"build": "test", "parse_formats": "fixture,fixture-v2",
		"parse_input_modes": "content,path", "parse_max_input_bytes": "1024",
	}, info.Metadata)
	limits, err := pluginsdk.ParseParseLimits(info.Metadata)
	require.NoError(t, err)
	require.Equal(t, &pluginsdk.ParseLimits{
		Formats: []string{"fixture", "fixture-v2"}, InputModes: []string{"content", "path"}, MaxInputBytes: 1024,
	}, limits)
}

func TestCostParseInfoInvalidLimitsAreNotRepaired(t *testing.T) {
	cases := []struct {
		name     string
		formats  []string
		modes    []string
		maxBytes int64
	}{
		{"uppercase format", []string{"Fixture"}, []string{"content"}, 1024},
		{"duplicate formats", []string{"fixture", "fixture"}, []string{"content"}, 1024},
		{"empty formats", nil, []string{"content"}, 1024},
		{"invalid mode", []string{"fixture"}, []string{"root_path"}, 1024},
		{"duplicate modes", []string{"fixture"}, []string{"path", "path"}, 1024},
		{"empty modes", []string{"fixture"}, nil, 1024},
		{"zero size", []string{"fixture"}, []string{"content"}, 0},
		{"negative size", []string{"fixture"}, []string{"content"}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Metadata remains a configuration gate even without a declared parse capability.
			info := pluginsdk.NewPluginInfo("parser", "v1.0.0",
				pluginsdk.WithParseLimits(tc.formats, tc.modes, tc.maxBytes))
			require.ErrorIs(t, info.Validate(), plugintesting.ErrInvalidParseLimits)
		})
	}
}

func TestCostParseInfoInferenceAndOverride(t *testing.T) {
	base := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	}
	cases := []struct {
		name     string
		plugin   pluginsdk.Plugin
		explicit []pbc.PluginCapability
		want     []pbc.PluginCapability
	}{
		{"existing pricing plugin", pluginsdk.NewBasePlugin("pricing"), nil, base},
		{"mixed plugin", &costParseInfoPlugin{pluginsdk.NewBasePlugin("mixed")}, nil,
			append(append([]pbc.PluginCapability(nil), base...), pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE)},
		{"parser only override", &costParseInfoPlugin{pluginsdk.NewBasePlugin("parser")},
			[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE},
			[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE}},
		{"explicit pricing override", &costParseInfoPlugin{pluginsdk.NewBasePlugin("pricing")},
			[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS},
			[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []pluginsdk.PluginInfoOption{
				pluginsdk.WithParseLimits([]string{"fixture"}, []string{"content", "path"}, 1024),
			}
			if tc.explicit != nil {
				opts = append(opts, pluginsdk.WithCapabilities(tc.explicit...))
			}
			info := pluginsdk.NewPluginInfo("parser", "v1.0.0", opts...)
			require.NoError(t, info.Validate())
			server := pluginsdk.NewServerWithOptions(tc.plugin, nil, nil, info)
			require.ElementsMatch(t, tc.want, server.GetGlobalCapabilities())
			resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, resp.GetCapabilities())
			if hasCapability(tc.want, pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE) {
				require.Equal(t, "true", resp.GetMetadata()["supports_cost_parse"])
			} else {
				require.NotContains(t, resp.GetMetadata(), "supports_cost_parse")
			}
			require.Equal(t, "fixture", resp.GetMetadata()["parse_formats"])
			require.Equal(t, "content,path", resp.GetMetadata()["parse_input_modes"])
			require.Equal(t, "1024", resp.GetMetadata()["parse_max_input_bytes"])
		})
	}
}

func TestCostParseCapabilityBoundsAndLegacy(t *testing.T) {
	require.Equal(t, int32(19), int32(pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE))
	for _, tc := range []struct {
		capability pbc.PluginCapability
		valid      bool
	}{
		{pbc.PluginCapability(-1), false},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, false},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS, true},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING, true},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE, true},
		{pbc.PluginCapability(21), false},
	} {
		t.Run(tc.capability.String(), func(t *testing.T) {
			require.Equal(t, tc.valid, pluginsdk.IsValidCapability(tc.capability))
		})
	}
	require.Equal(t, "supports_cost_parse",
		pluginsdk.CapabilityToLegacyName(pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE))
	metadata, warnings := pluginsdk.CapabilitiesToLegacyMetadataWithWarnings([]pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
	})
	require.Empty(t, warnings)
	require.Equal(t, map[string]string{"supports_cost_parse": "true", "supports_projected_costs": "true"}, metadata)
}
