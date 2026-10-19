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
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

type costParseFunc func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error)

func (f costParseFunc) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return f(ctx, req)
}

type costParseTestPlugin struct {
	*pluginsdk.BasePlugin

	inner pluginsdk.CostParseProvider
}

func newCostParseTestPlugin(inner pluginsdk.CostParseProvider) *costParseTestPlugin {
	return &costParseTestPlugin{BasePlugin: pluginsdk.NewBasePlugin("parse-test"), inner: inner}
}

func (p *costParseTestPlugin) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return p.inner.Parse(ctx, req)
}

// Embedding the generated stub still implements the optional provider interface.
type costParseStubPlugin struct {
	*pluginsdk.BasePlugin
	pbc.UnimplementedCostParseServiceServer
}

type costParseCall func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error)

// startCostParseServer reuses the injected-listener Serve harness. The address is
// also available for raw codec probes and discovery/health calls in later tests.
func startCostParseServer(
	t *testing.T, plugin pluginsdk.Plugin, web bool, opts ...func(*pluginsdk.ServeConfig),
) (string, costParseCall) {
	t.Helper()
	configure := []func(*pluginsdk.ServeConfig){func(config *pluginsdk.ServeConfig) {
		config.PluginInfo = pluginsdk.NewPluginInfo("parse-test", "1.0.0",
			pluginsdk.WithParseLimits([]string{"fixture"}, []string{"content", "path"}, 1024))
	}}
	configure = append(configure, opts...)
	addr, _ := startAllocServer(t, plugin, web, configure...)
	if web {
		client := pbcconnect.NewCostParseServiceClient(http.DefaultClient, "http://"+addr)
		return addr, func(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
			resp, err := client.Parse(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return resp.Msg, nil
		}
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewCostParseServiceClient(conn)
	return addr, func(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
		return client.Parse(ctx, req)
	}
}

func callCostParse(t *testing.T, call costParseCall, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
	defer cancel()
	return call(ctx, req)
}

func costParseFixtureRequest() *pbc.ParseRequest {
	return &pbc.ParseRequest{Format: "fixture", Input: &pbc.ParseRequest_Files{Files: &pbc.ParseFileSet{
		Files: []*pbc.ParseFile{{Path: "main.fixture", Content: []byte("instance")}},
	}}}
}

func costParseFixtureResponse() *pbc.ParseResponse {
	return &pbc.ParseResponse{Resources: []*pbc.ParsedResource{{
		SourceId: "Instance", Resource: &pbc.ResourceDescriptor{
			Provider: "custom", ResourceType: "fixture:compute:Instance", Region: "us-east-1",
		},
	}}}
}

func TestCostParseServe_ValidInputsAndDiscovery(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			for _, mode := range []string{"content", "path"} {
				t.Run(mode, func(t *testing.T) {
					req := costParseFixtureRequest()
					if mode == "path" {
						req.Input = &pbc.ParseRequest_RootPath{RootPath: t.TempDir()}
					}
					provider := costParseFunc(
						func(_ context.Context, got *pbc.ParseRequest) (*pbc.ParseResponse, error) {
							assert.True(t, proto.Equal(req, got), "Serve changed parser input")
							return costParseFixtureResponse(), nil
						},
					)
					addr, call := startCostParseServer(t, newCostParseTestPlugin(provider), tr.web)
					resp, err := callCostParse(t, call, req)
					require.NoError(t, err)
					require.NoError(t, plugintesting.ValidateParseResponse(req, resp))
					assert.True(t, proto.Equal(costParseFixtureResponse(), resp))

					ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
					defer cancel()
					var info *pbc.GetPluginInfoResponse
					if tr.web {
						client := pbcconnect.NewCostSourceServiceClient(http.DefaultClient, "http://"+addr)
						result, infoErr := client.GetPluginInfo(ctx, connect.NewRequest(&pbc.GetPluginInfoRequest{}))
						require.NoError(t, infoErr)
						info = result.Msg
					} else {
						conn, connErr := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
						require.NoError(t, connErr)
						t.Cleanup(func() { _ = conn.Close() })
						info, err = pbc.NewCostSourceServiceClient(conn).GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
						require.NoError(t, err)
					}
					assert.Contains(t, info.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE)
					assert.Equal(t, "true", info.GetMetadata()["supports_cost_parse"])
					assert.Equal(t, "fixture", info.GetMetadata()["parse_formats"])
					assert.Equal(t, "content,path", info.GetMetadata()["parse_input_modes"])
					assert.Equal(t, "1024", info.GetMetadata()["parse_max_input_bytes"])
				})
			}
		})
	}
}

func TestCostParseServe_HealthIncludesParser(t *testing.T) {
	for _, stub := range []bool{false, true} {
		name := "implemented"
		var plugin pluginsdk.Plugin = newCostParseTestPlugin(costParseFunc(
			func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
				t.Error("health must not invoke Parse")
				return costParseFixtureResponse(), nil
			}))
		if stub {
			name = "generated_stub"
			plugin = &costParseStubPlugin{BasePlugin: pluginsdk.NewBasePlugin("stub")}
		}
		t.Run(name, func(t *testing.T) {
			addr, _ := startCostParseServer(t, plugin, true)
			resp, err := checkServiceHealth(t, addr, pbcconnect.CostParseServiceName)
			require.NoError(t, err)
			assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
		})
	}
}

func TestCostParseServe_Unimplemented(t *testing.T) {
	for _, tr := range allocTransports {
		for _, stub := range []bool{false, true} {
			name := "absent"
			var plugin pluginsdk.Plugin = pluginsdk.NewBasePlugin("plain")
			if stub {
				name = "generated_stub"
				plugin = &costParseStubPlugin{BasePlugin: pluginsdk.NewBasePlugin("stub")}
			}
			t.Run(tr.name+"/"+name, func(t *testing.T) {
				addr, call := startCostParseServer(t, plugin, tr.web)
				req := costParseFixtureRequest()
				if !stub {
					// Missing providers take precedence even over malformed requests.
					req = &pbc.ParseRequest{}
				}
				_, err := callCostParse(t, call, req)
				code, _ := allocWireCode(t, err, tr.web)
				assert.Equal(t, codes.Unimplemented, code)
				if tr.web && !stub {
					_, healthErr := checkServiceHealth(t, addr, pbcconnect.CostParseServiceName)
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(healthErr))
				}
			})
		}
	}
}

func TestCostParseServe_InvalidSuccessIsInternal(t *testing.T) {
	for _, tr := range allocTransports {
		for name, resp := range map[string]*pbc.ParseResponse{
			"nil":                nil,
			"missing_descriptor": {Resources: []*pbc.ParsedResource{{SourceId: "Instance"}}},
			"missing_source": {Resources: []*pbc.ParsedResource{{Resource: &pbc.ResourceDescriptor{
				Provider: "custom", ResourceType: "fixture:compute:Instance",
			}}}},
		} {
			t.Run(tr.name+"/"+name, func(t *testing.T) {
				plugin := newCostParseTestPlugin(costParseFunc(
					func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) { return resp, nil }))
				_, call := startCostParseServer(t, plugin, tr.web)
				_, err := callCostParse(t, call, costParseFixtureRequest())
				code, _ := allocWireCode(t, err, tr.web)
				assert.Equal(t, codes.Internal, code)
			})
		}
	}
}

func TestCostParseServe_MissingLimitsRejectsStartup(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			plugin := newCostParseTestPlugin(costParseFunc(
				func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
					t.Error("invalid configuration must not invoke Parse")
					return costParseFixtureResponse(), nil
				}))
			ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
			defer cancel()
			err = pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
				Plugin: plugin, Listener: listener, Web: pluginsdk.WebConfig{Enabled: tr.web},
				PluginInfo: pluginsdk.NewPluginInfo("parse-test", "1.0.0"),
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, plugintesting.ErrInvalidParseLimits)
		})
	}
}
