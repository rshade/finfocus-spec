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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// recordingSupportsPlugin records the capability of each Supports request it receives.
type recordingSupportsPlugin struct {
	*pluginsdk.BasePlugin

	mu    sync.Mutex
	calls []pbc.PluginCapability
}

func (p *recordingSupportsPlugin) Supports(
	_ context.Context, req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, req.GetCapability())
	return &pbc.SupportsResponse{Supported: true}, nil
}

func (p *recordingSupportsPlugin) recorded() []pbc.PluginCapability {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]pbc.PluginCapability(nil), p.calls...)
}

func TestClientSupportsFor(t *testing.T) {
	resource := &pbc.ResourceDescriptor{Provider: "custom", ResourceType: "custom:compute:Instance"}
	for _, protocol := range []struct {
		name      string
		newClient func(string) *pluginsdk.Client
	}{
		{"connect", pluginsdk.NewConnectClient},
		{"grpc", pluginsdk.NewGRPCClient},
	} {
		t.Run(protocol.name, func(t *testing.T) {
			plugin := &recordingSupportsPlugin{BasePlugin: pluginsdk.NewBasePlugin("recording")}
			addr := serveForClient(t, plugin)
			client := protocol.newClient("http://" + addr)
			t.Cleanup(client.Close)
			ctx := context.Background()

			_, err := client.SupportsFor(ctx, resource, pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS)
			require.NoError(t, err)
			_, err = client.Supports(ctx, resource)
			require.NoError(t, err)
			_, err = client.SupportsResourceType(ctx, resource.GetResourceType())
			require.NoError(t, err)
			_, err = client.SupportsFor(ctx, resource, pbc.PluginCapability(999))
			require.NoError(t, err)

			assert.Equal(t, []pbc.PluginCapability{
				pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
				pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED,
				pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED,
				pbc.PluginCapability(999),
			}, plugin.recorded())
		})
	}
}

func TestClientSupportsForRejectsBadInput(t *testing.T) {
	client := pluginsdk.NewConnectClient("http://127.0.0.1:1")
	t.Cleanup(client.Close)
	ctx := context.Background()
	projected := pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS

	_, err := client.SupportsFor(ctx, nil, projected)
	require.EqualError(t, err, "resource descriptor cannot be nil")
	_, err = client.SupportsFor(ctx, &pbc.ResourceDescriptor{Provider: "custom"}, projected)
	require.EqualError(t, err, "resource type is required")
}

// serveForClient serves plugin in web mode on an injected listener for SDK client tests.
func serveForClient(t *testing.T, plugin pluginsdk.Plugin) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin: plugin, Listener: listener, Web: pluginsdk.WebConfig{Enabled: true},
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
			t.Error("server did not shut down in time")
		}
	})
	addr := listener.Addr().String()
	waitForServer(t, addr)
	return addr
}
