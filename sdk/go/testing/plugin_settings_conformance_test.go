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
	"testing"

	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestPluginSettingsConformance(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assertPluginInfoCheck(t, plugintesting.NewMockPlugin(), true)
	})
	t.Run("valid", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.Settings = []*pbc.PluginSetting{{
			Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
			Secret: true, Env: "TYPESAFE_API_KEY",
		}}
		assertPluginInfoCheck(t, plugin, true)
	})
	t.Run("duplicate key", func(t *testing.T) {
		assertPluginInfoCheck(t, &duplicateSettingsPlugin{MockPlugin: plugintesting.NewMockPlugin()}, false)
	})
}

type duplicateSettingsPlugin struct {
	*plugintesting.MockPlugin
}

func (p *duplicateSettingsPlugin) GetPluginInfo(
	ctx context.Context, req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	resp, err := p.MockPlugin.GetPluginInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Settings = []*pbc.PluginSetting{
		{Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING, Env: "TYPESAFE_API_KEY"},
		{Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING, Env: "OTHER_API_KEY"},
	}
	return resp, nil
}

func assertPluginInfoCheck(t *testing.T, plugin pbc.CostSourceServiceServer, want bool) {
	t.Helper()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	t.Cleanup(harness.Stop)
	result := createGetPluginInfoValidResponseTest()(harness)
	if result.Success != want {
		t.Fatalf("success = %v, want %v, err = %v, details = %s", result.Success, want, result.Error, result.Details)
	}
}
