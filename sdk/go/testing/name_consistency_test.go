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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// renamedInfoPlugin answers GetPluginInfo with infoName instead of its Name() answer.
type renamedInfoPlugin struct {
	*plugintesting.MockPlugin

	infoName string
}

func (p renamedInfoPlugin) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	resp, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Name = p.infoName
	return resp, nil
}

func namedMock(name string) *plugintesting.MockPlugin {
	mock := plugintesting.NewMockPlugin()
	mock.PluginName = name
	return mock
}

// nameMismatches returns the details of every failed check that reports ErrPluginNameMismatch.
func nameMismatches(result *plugintesting.ConformanceResult) []string {
	var details []string
	for _, category := range result.Categories {
		for _, r := range category.Results {
			if !r.Success && errors.Is(r.Error, plugintesting.ErrPluginNameMismatch) {
				details = append(details, r.Details+": "+r.Error.Error())
			}
		}
	}
	return details
}

func TestConformanceNameChecksAreOptIn(t *testing.T) {
	levels := []plugintesting.ConformanceLevel{
		plugintesting.ConformanceLevelBasic,
		plugintesting.ConformanceLevelStandard,
		plugintesting.ConformanceLevelAdvanced,
	}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			agreeing, err := plugintesting.RunConformance(namedMock("cost-plugin"), level)
			require.NoError(t, err)
			drifting, err := plugintesting.RunConformance(
				renamedInfoPlugin{MockPlugin: namedMock("cost-plugin"), infoName: "other"}, level)
			require.NoError(t, err)
			withName, err := plugintesting.RunConformance(namedMock("cost-plugin"), level,
				plugintesting.WithExpectedName("cost-plugin"))
			require.NoError(t, err)

			assert.Empty(t, nameMismatches(drifting), "name checks must not run without WithExpectedName")
			assert.Equal(t, agreeing.Summary.Total, drifting.Summary.Total)
			assert.Equal(t, agreeing.Summary.Passed, drifting.Summary.Passed)
			assert.Equal(t, agreeing.Summary.Total+2, withName.Summary.Total)
			assert.Equal(t, agreeing.Summary.Passed+2, withName.Summary.Passed)
		})
	}
}

func TestConformanceNameConsistency(t *testing.T) {
	tests := []struct {
		name      string
		plugin    pbc.CostSourceServiceServer
		opts      []plugintesting.ConformanceOption
		wantCount int
		wantText  []string
	}{
		{
			name:   "names agree",
			plugin: namedMock("cost-plugin"),
			opts:   []plugintesting.ConformanceOption{plugintesting.WithExpectedName("cost-plugin")},
		},
		{
			name:      "plugin info name differs from Name",
			plugin:    renamedInfoPlugin{MockPlugin: namedMock("cost-plugin"), infoName: "cost-plugin-v2"},
			opts:      []plugintesting.ConformanceOption{plugintesting.WithExpectedName("cost-plugin")},
			wantCount: 1,
			wantText:  []string{`"cost-plugin"`, `"cost-plugin-v2"`},
		},
		{
			name:   "no plugin info is not compared",
			plugin: plugintesting.NewMockLegacyPlugin("legacy-plugin"),
			opts:   []plugintesting.ConformanceOption{plugintesting.WithExpectedName("legacy-plugin")},
		},
		{
			name:      "expected name differs from Name",
			plugin:    namedMock("y"),
			opts:      []plugintesting.ConformanceOption{plugintesting.WithExpectedName("x")},
			wantCount: 1,
			wantText:  []string{`"x"`, `"y"`},
		},
		{
			name:      "expected name differs and plugin info drifts too",
			plugin:    renamedInfoPlugin{MockPlugin: namedMock("y"), infoName: "z"},
			opts:      []plugintesting.ConformanceOption{plugintesting.WithExpectedName("x")},
			wantCount: 2,
		},
		{
			name:      "expected name is checked without plugin info",
			plugin:    plugintesting.NewMockLegacyPlugin("y"),
			opts:      []plugintesting.ConformanceOption{plugintesting.WithExpectedName("x")},
			wantCount: 1,
		},
		{
			name:   "empty expected name runs no name check",
			plugin: renamedInfoPlugin{MockPlugin: namedMock("y"), infoName: "z"},
			opts:   []plugintesting.ConformanceOption{plugintesting.WithExpectedName("")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plugintesting.RunConformance(tt.plugin, plugintesting.ConformanceLevelBasic, tt.opts...)
			require.NoError(t, err)

			mismatches := nameMismatches(result)
			require.Len(t, mismatches, tt.wantCount, "name mismatches: %v", mismatches)
			for _, text := range tt.wantText {
				assert.Contains(t, mismatches[0], text)
			}
			if tt.wantCount > 0 {
				assert.Positive(t, result.Summary.Failed)
			}
		})
	}
}

func TestConformanceNameChecksInRunCategory(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
		want         int
	}{
		{name: "expected name set", expectedName: "x", want: 2},
		{name: "no expected name", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := plugintesting.DefaultSuiteConfig()
			config.TargetLevel = plugintesting.ConformanceLevelBasic
			config.ExpectedName = tt.expectedName
			suite := plugintesting.NewConformanceSuiteWithConfig(config)
			plugintesting.RegisterRPCCorrectnessTests(suite)

			category, err := suite.RunCategory(
				renamedInfoPlugin{MockPlugin: namedMock("y"), infoName: "z"}, plugintesting.CategoryRPCCorrectness)
			require.NoError(t, err)

			var mismatches int
			for _, r := range category.Results {
				if !r.Success && errors.Is(r.Error, plugintesting.ErrPluginNameMismatch) {
					mismatches++
				}
			}
			assert.Equal(t, tt.want, mismatches)
		})
	}
}
