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

//nolint:testpackage // Uses the unexported capability bounds and the unknown-value set.
package pluginsdk

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

const supportsCapabilityTimeout = 5 * time.Second

// perCapabilityPlugin serves actual cost but cannot price projected cost.
type perCapabilityPlugin struct {
	*BasePlugin
}

func (p *perCapabilityPlugin) Supports(_ context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	if req.GetCapability() == pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS {
		return &pbc.SupportsResponse{Supported: false, Reason: "projected cost not priced"}, nil
	}
	return &pbc.SupportsResponse{Supported: true}, nil
}

type supportsCall func(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error)

func startSupportsServer(t *testing.T, plugin Plugin, web bool) supportsCall {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, ServeConfig{Plugin: plugin, Listener: listener, Web: WebConfig{Enabled: web}})
	}()
	t.Cleanup(func() {
		// Shutdown waits 5s on a pooled connection that never sent a request.
		http.DefaultClient.CloseIdleConnections()
		cancel()
		select {
		case <-errCh:
		case <-time.After(supportsCapabilityTimeout):
			t.Error("server did not shut down in time")
		}
	})

	if web {
		client := pbcconnect.NewCostSourceServiceClient(http.DefaultClient, "http://"+addr)
		return func(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
			resp, callErr := client.Supports(ctx, connect.NewRequest(req))
			if callErr != nil {
				return nil, callErr
			}
			return resp.Msg, nil
		}
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewCostSourceServiceClient(conn)
	return func(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
		return client.Supports(ctx, req)
	}
}

// TestSupportsCapabilityEndToEnd shows a plugin answering differently for the same
// resource depending on the capability the host asks about, over both transports.
func TestSupportsCapabilityEndToEnd(t *testing.T) {
	resource := &pbc.ResourceDescriptor{Provider: "custom", ResourceType: "custom:compute:Instance"}
	cases := []struct {
		capability pbc.PluginCapability
		supported  bool
	}{
		{pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS, true},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS, false},
		{pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, true},
	}
	for _, tr := range []struct {
		name string
		web  bool
	}{{"grpc", false}, {"connect", true}} {
		t.Run(tr.name, func(t *testing.T) {
			call := startSupportsServer(t, &perCapabilityPlugin{BasePlugin: NewBasePlugin("per-capability")}, tr.web)
			for _, tc := range cases {
				ctx, cancel := context.WithTimeout(context.Background(), supportsCapabilityTimeout)
				resp, err := call(ctx, &pbc.SupportsRequest{Resource: resource, Capability: tc.capability})
				cancel()
				require.NoError(t, err, tc.capability)
				assert.Equal(t, tc.supported, resp.GetSupported(), tc.capability)
				if !tc.supported {
					assert.Equal(t, "projected cost not priced", resp.GetReason())
				}
			}
		})
	}
}

func TestRequestedCapability(t *testing.T) {
	assert.Equal(t, pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, RequestedCapability(nil))

	unspecified := &pbc.SupportsRequest{}
	assert.Equal(t, pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, RequestedCapability(unspecified))

	for capability := minValidCapability; capability <= maxValidCapability; capability++ {
		req := &pbc.SupportsRequest{Capability: capability}
		assert.Equal(t, capability, RequestedCapability(req), "known value %d", capability)
	}

	for _, raw := range []int32{999, -1, int32(maxValidCapability) + 1} {
		req := &pbc.SupportsRequest{Capability: pbc.PluginCapability(raw)}
		assert.Equal(t, pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, RequestedCapability(req), "unknown %d", raw)
	}
}

func TestRequestedCapabilityWarnsOncePerValue(t *testing.T) {
	unknownCapabilitiesSeen.Clear()
	var buf bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&buf) //nolint:reassign // zerolog's global logger has no setter; restored below.
	t.Cleanup(func() {
		log.Logger = previous //nolint:reassign // restores the global logger replaced above.
		unknownCapabilitiesSeen.Clear()
	})

	for _, raw := range []int32{997, 997, 998} {
		RequestedCapability(&pbc.SupportsRequest{Capability: pbc.PluginCapability(raw)})
	}

	var warnings []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "Unknown Supports capability") {
			warnings = append(warnings, line)
		}
	}
	require.Len(t, warnings, 2, buf.String())
	assert.Contains(t, warnings[0], `"capability":997`)
	assert.Contains(t, warnings[1], `"capability":998`)
	assert.Contains(t, warnings[0], `"level":"warn"`)
}

func BenchmarkRequestedCapability(b *testing.B) {
	for _, bc := range []struct {
		name string
		req  *pbc.SupportsRequest
	}{
		{"known", &pbc.SupportsRequest{Capability: pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS}},
		{"unspecified", &pbc.SupportsRequest{}},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = RequestedCapability(bc.req)
			}
		})
	}
}

type failingSupportsPlugin struct {
	*BasePlugin
}

func (p *failingSupportsPlugin) Supports(context.Context, *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	return nil, errors.New("boom")
}

func TestSupportsHandlerErrorLogsCapability(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	server := NewServerWithOptions(&failingSupportsPlugin{BasePlugin: NewBasePlugin("failing")}, nil, &logger, nil)

	_, err := server.Supports(context.Background(), &pbc.SupportsRequest{
		Resource:   &pbc.ResourceDescriptor{Provider: "custom", ResourceType: "custom:compute:Instance"},
		Capability: pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
	})
	require.Error(t, err)
	assert.Contains(t, buf.String(), "Supports handler error")
	assert.Contains(t, buf.String(), `"capability":"PLUGIN_CAPABILITY_PROJECTED_COSTS"`)
}
