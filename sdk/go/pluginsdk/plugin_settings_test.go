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

package pluginsdk //nolint:testpackage // Uses the unexported mockPlugin and Server wiring.

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func declaredSettings() (*pbc.PluginSetting, *pbc.PluginSetting) {
	apiKey := &pbc.PluginSetting{
		Key:      "api_key",
		Type:     pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
		Required: true,
		Secret:   true,
		Env:      "TYPESAFE_API_KEY",
	}
	timeout := &pbc.PluginSetting{
		Key:          "timeout",
		Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION,
		DefaultValue: "30s",
		Env:          "JEV_TIMEOUT",
	}
	return apiKey, timeout
}

func TestPluginSettings_StaticOrder(t *testing.T) {
	apiKey, timeout := declaredSettings()
	original := []*pbc.PluginSetting{apiKey, timeout}
	info := NewPluginInfo("jev", "v1.0.0", WithProviders("custom"), WithSettings(original...))
	apiKey.Key = "changed"
	original[0] = &pbc.PluginSetting{Key: "replaced"}

	server := NewServerWithOptions(&mockPlugin{name: "jev"}, nil, nil, info)
	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetSettings(), 2)

	gotKey := resp.GetSettings()[0]
	assert.Equal(t, "api_key", gotKey.GetKey())
	assert.Equal(t, pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING, gotKey.GetType())
	assert.True(t, gotKey.GetRequired())
	assert.True(t, gotKey.GetSecret())
	assert.Equal(t, "TYPESAFE_API_KEY", gotKey.GetEnv())

	gotTimeout := resp.GetSettings()[1]
	assert.Equal(t, "timeout", gotTimeout.GetKey())
	assert.Equal(t, pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION, gotTimeout.GetType())
	assert.Equal(t, "30s", gotTimeout.GetDefaultValue())
	assert.Equal(t, "JEV_TIMEOUT", gotTimeout.GetEnv())

	for key, value := range resp.GetMetadata() {
		assert.NotEqual(t, "api_key", key)
		assert.NotEqual(t, "TYPESAFE_API_KEY", key)
		assert.NotContains(t, value, "30s")
	}
}

func TestPluginSettings_EmptyList(t *testing.T) {
	plain := NewPluginInfo("jev", "v1.0.0", WithProviders("custom"))
	empty := NewPluginInfo("jev", "v1.0.0", WithProviders("custom"), WithSettings())
	plugin := &mockPlugin{name: "jev"}

	plainResp := mustPluginInfo(t, plugin, plain)
	emptyResp := mustPluginInfo(t, plugin, empty)
	assert.Empty(t, plainResp.GetSettings())
	assert.Empty(t, emptyResp.GetSettings())
	assert.Equal(t, plainResp.GetName(), emptyResp.GetName())
	assert.Equal(t, plainResp.GetVersion(), emptyResp.GetVersion())
	assert.Equal(t, plainResp.GetSpecVersion(), emptyResp.GetSpecVersion())
	assert.Equal(t, plainResp.GetProviders(), emptyResp.GetProviders())
	assert.Equal(t, plainResp.GetCapabilities(), emptyResp.GetCapabilities())

	provider := &settingsProvider{mockPlugin: mockPlugin{name: "jev"}, resp: &pbc.GetPluginInfoResponse{
		Name: "jev", Version: "v1.0.0", SpecVersion: SpecVersion,
	}}
	resp, err := NewServer(provider).GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetSettings())
}

func TestPluginSettings_RejectStatic(t *testing.T) {
	info := NewPluginInfo("jev", "v1.0.0", WithSettings(
		&pbc.PluginSetting{
			Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
			Env: "TYPESAFE_API_KEY", DefaultValue: "s3cret-value",
		},
		&pbc.PluginSetting{
			Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
			Env: "OTHER_API_KEY",
		},
	))
	err := info.Validate()
	require.ErrorIs(t, err, ErrInvalidPluginSettings)
	require.NotContains(t, err.Error(), "s3cret-value")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	err = Serve(context.Background(), ServeConfig{
		Plugin:     &mockPlugin{name: "jev"},
		Listener:   ln,
		PluginInfo: info,
	})
	require.ErrorIs(t, err, ErrInvalidPluginSettings)
	require.NotContains(t, err.Error(), "s3cret-value")
}

func TestPluginSettings_RejectProvider(t *testing.T) {
	bad := &settingsProvider{
		mockPlugin: mockPlugin{name: "jev"},
		resp: &pbc.GetPluginInfoResponse{
			Name: "jev", Version: "v1.0.0", SpecVersion: SpecVersion,
			Settings: []*pbc.PluginSetting{
				{
					Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
					Env: "TYPESAFE_API_KEY", DefaultValue: "s3cret-value",
				},
				{
					Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
					Env: "OTHER_API_KEY",
				},
			},
		},
	}
	resp, err := NewServer(bad).GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "plugin settings are invalid", st.Message())
	assert.NotContains(t, st.Message(), "s3cret-value")

	good := &settingsProvider{
		mockPlugin: mockPlugin{name: "jev"},
		resp: &pbc.GetPluginInfoResponse{
			Name: "jev", Version: "v1.0.0", SpecVersion: SpecVersion,
			Settings: []*pbc.PluginSetting{{
				Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
				Secret: true, Env: "TYPESAFE_API_KEY",
			}},
		},
	}
	resp, err = NewServer(good).GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetSettings(), 1)
	assert.Equal(t, "api_key", resp.GetSettings()[0].GetKey())
	assert.True(t, resp.GetSettings()[0].GetSecret())
	assert.Equal(t, "TYPESAFE_API_KEY", resp.GetSettings()[0].GetEnv())
}

type settingsProvider struct {
	mockPlugin

	resp *pbc.GetPluginInfoResponse
}

func (p *settingsProvider) GetPluginInfo(
	context.Context, *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return p.resp, nil
}

func mustPluginInfo(t *testing.T, plugin Plugin, info *PluginInfo) *pbc.GetPluginInfoResponse {
	t.Helper()
	resp, err := NewServerWithOptions(plugin, nil, nil, info).GetPluginInfo(
		context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	return resp
}
