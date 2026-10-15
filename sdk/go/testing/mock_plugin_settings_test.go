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

func TestMockPluginSettings(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.Settings = []*pbc.PluginSetting{
		{
			Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
			Required: true, Secret: true, Env: "TYPESAFE_API_KEY",
		},
		{
			Key: "timeout", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION,
			DefaultValue: "30s", Env: "JEV_TIMEOUT",
		},
	}
	resp := pluginInfo(t, plugin)
	if len(resp.GetSettings()) != 2 {
		t.Fatalf("settings = %d, want 2", len(resp.GetSettings()))
	}
	if resp.GetSettings()[0].GetKey() != "api_key" || !resp.GetSettings()[0].GetSecret() {
		t.Fatalf("first setting = %+v", resp.GetSettings()[0])
	}
	if resp.GetSettings()[0].GetEnv() != "TYPESAFE_API_KEY" {
		t.Fatalf("env = %q", resp.GetSettings()[0].GetEnv())
	}
	if resp.GetSettings()[1].GetKey() != "timeout" || resp.GetSettings()[1].GetDefaultValue() != "30s" {
		t.Fatalf("second setting = %+v", resp.GetSettings()[1])
	}
}

func TestMockPluginSettings_DefaultEmpty(t *testing.T) {
	resp := pluginInfo(t, plugintesting.NewMockPlugin())
	if resp.GetName() == "" || resp.GetVersion() == "" || resp.GetSpecVersion() == "" {
		t.Fatalf("plugin info = %+v", resp)
	}
	if len(resp.GetSettings()) != 0 {
		t.Fatalf("settings = %d, want 0", len(resp.GetSettings()))
	}
}

func pluginInfo(t *testing.T, plugin *plugintesting.MockPlugin) *pbc.GetPluginInfoResponse {
	t.Helper()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	t.Cleanup(harness.Stop)
	resp, err := harness.Client().GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo() error = %v", err)
	}
	return resp
}
