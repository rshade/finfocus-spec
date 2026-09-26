//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// staticInfoCapabilityPlugin implements every optional capability interface and
// returns a fixed PluginInfoProvider response.
type staticInfoCapabilityPlugin struct {
	mockCapabilityPlugin

	resp *pbc.GetPluginInfoResponse
}

func (p *staticInfoCapabilityPlugin) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return p.resp, nil
}

func baseCapabilityList() []pbc.PluginCapability {
	return []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	}
}

func providerInfoResponse(
	caps []pbc.PluginCapability,
	meta map[string]string,
) *pbc.GetPluginInfoResponse {
	return &pbc.GetPluginInfoResponse{
		Name:         "provider-plugin",
		Version:      "v1.0.0",
		SpecVersion:  SpecVersion,
		Providers:    []string{"aws"},
		Capabilities: caps,
		Metadata:     meta,
	}
}

func TestGetPluginInfo_ProviderCapabilityBackfill(t *testing.T) {
	maxBatch := strconv.Itoa(DefaultMaxBatchSize)
	inferred := inferCapabilities(&mockCapabilityPlugin{})

	tests := []struct {
		name       string
		resp       *pbc.GetPluginInfoResponse
		wantCaps   []pbc.PluginCapability
		wantMeta   map[string]string
		absentKeys []string
	}{
		{
			name:     "empty capabilities inherit the inferred set and legacy keys",
			resp:     providerInfoResponse(nil, map[string]string{"region": "us-east-1", "type": "public"}),
			wantCaps: inferred,
			wantMeta: map[string]string{
				"region":                          "us-east-1",
				"type":                            "public",
				"supports_recommendations":        "true",
				"supports_resolve_resource_types": "true",
				"supports_batch_cost":             "true",
				"max_batch_size":                  maxBatch,
			},
		},
		{
			name:     "nil metadata is allocated for backfilled keys",
			resp:     providerInfoResponse(nil, nil),
			wantCaps: inferred,
			wantMeta: map[string]string{
				"supports_projected_costs": "true",
				"supports_recommendations": "true",
			},
		},
		{
			name:     "explicit list is authoritative with no union",
			resp:     providerInfoResponse(baseCapabilityList(), nil),
			wantCaps: baseCapabilityList(),
			wantMeta: map[string]string{
				"supports_projected_costs": "true",
				"supports_estimate_cost":   "true",
			},
			absentKeys: []string{
				"supports_budgets",
				"supports_dismiss_recommendations",
				"supports_recommendations",
				"max_batch_size",
			},
		},
		{
			name: "provider-set legacy key wins over backfill",
			resp: providerInfoResponse(nil, map[string]string{
				"supports_recommendations": "false",
			}),
			wantCaps: inferred,
			wantMeta: map[string]string{
				"supports_recommendations": "false",
				"supports_budgets":         "true",
			},
		},
		{
			name: "explicit batch cost still gets max_batch_size",
			resp: providerInfoResponse(
				append(baseCapabilityList(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST),
				nil,
			),
			wantCaps: append(baseCapabilityList(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST),
			wantMeta: map[string]string{
				"supports_batch_cost": "true",
				"max_batch_size":      maxBatch,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(&staticInfoCapabilityPlugin{resp: tt.resp})

			resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			require.NoError(t, err)

			assert.Equal(t, tt.wantCaps, resp.GetCapabilities())
			for k, v := range tt.wantMeta {
				assert.Equal(t, v, resp.GetMetadata()[k], "metadata key %q", k)
			}
			for _, k := range tt.absentKeys {
				assert.NotContains(t, resp.GetMetadata(), k)
			}
		})
	}
}

func TestGetPluginInfo_ProviderResponseNotMutated(t *testing.T) {
	shared := providerInfoResponse(nil, map[string]string{"region": "us-east-1"})
	snapshot := proto.Clone(shared)
	server := NewServer(&staticInfoCapabilityPlugin{resp: shared})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			if err != nil {
				t.Error(err)
				return
			}
			assert.NotEmpty(t, resp.GetCapabilities())
		})
	}
	wg.Wait()

	assert.True(t, proto.Equal(snapshot, shared), "provider response must not be mutated")
}

func TestGetPluginInfo_ProviderCapabilityDriftLoggedOnce(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	plugin := &staticInfoCapabilityPlugin{resp: providerInfoResponse(baseCapabilityList(), nil)}
	server := NewServerWithOptions(plugin, nil, &logger, nil)

	for range 2 {
		_, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, strings.Count(buf.String(), "omit inferred capabilities"))
	assert.Contains(t, buf.String(), "PLUGIN_CAPABILITY_BUDGETS")
}

func TestServer_SetTypeRegistryCapability(t *testing.T) {
	resolve := pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES

	tests := []struct {
		name     string
		plugin   Plugin
		registry *TypeRegistry
		explicit bool
		want     bool
	}{
		{name: "no registry", plugin: &mockPlugin{}, want: false},
		{name: "registry on inferred capabilities", plugin: &mockPlugin{}, registry: NewTypeRegistry(), want: true},
		{
			name:     "registry with explicit capabilities",
			plugin:   &mockPlugin{},
			registry: NewTypeRegistry(),
			explicit: true,
			want:     false,
		},
		{
			name:     "resolver plugin is not duplicated",
			plugin:   &mockCapabilityPlugin{},
			registry: NewTypeRegistry(),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.plugin)
			server.setTypeRegistry(tt.registry, tt.explicit)

			caps := server.GetGlobalCapabilities()
			count := 0
			for _, c := range caps {
				if c == resolve {
					count++
				}
			}
			if tt.want {
				assert.Equal(t, 1, count)
			} else {
				assert.Zero(t, count)
			}
			assert.Same(t, tt.registry, server.typeRegistry)
		})
	}
}

func TestSupports_TypeRegistryCapability(t *testing.T) {
	registry := &mockRegistry{plugins: map[string]string{"aws:us-east-1": "test"}}
	server := NewServerWithRegistry(&mockPlugin{name: "test"}, registry)
	server.setTypeRegistry(NewTypeRegistry(), false)

	resp, err := server.Supports(context.Background(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{Provider: "aws", Region: "us-east-1", ResourceType: "ec2"},
	})
	require.NoError(t, err)
	assert.Contains(t, resp.GetCapabilitiesEnum(), pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
	assert.True(t, resp.GetCapabilities()["supports_resolve_resource_types"])
}

func TestServe_TypeRegistryAdvertisesResolveResourceTypes(t *testing.T) {
	tests := []struct {
		name string
		info *PluginInfo
		want bool
	}{
		{name: "inferred capabilities", info: NewPluginInfo("registry-plugin", "v1.0.0"), want: true},
		{
			name: "explicit capabilities",
			info: NewPluginInfo("registry-plugin", "v1.0.0", WithCapabilities(baseCapabilityList()...)),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			errCh := make(chan error, 1)
			go func() {
				errCh <- Serve(ctx, ServeConfig{
					Plugin:          &mockPlugin{name: "registry-plugin"},
					Listener:        listener,
					PluginInfo:      tt.info,
					TypeRegistry:    NewTypeRegistry(),
					handshakeWriter: io.Discard,
				})
			}()

			conn, err := grpc.NewClient(listener.Addr().String(),
				grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			client := pbc.NewCostSourceServiceClient(conn)
			callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
			defer callCancel()
			resp, err := client.GetPluginInfo(callCtx, &pbc.GetPluginInfoRequest{}, grpc.WaitForReady(true))
			require.NoError(t, err)

			caps := resp.GetCapabilities()
			meta := resp.GetMetadata()
			if tt.want {
				assert.Contains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
				assert.Equal(t, "true", meta["supports_resolve_resource_types"])
			} else {
				assert.NotContains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
				assert.NotContains(t, meta, "supports_resolve_resource_types")
			}

			cancel()
			select {
			case <-errCh:
			case <-time.After(5 * time.Second):
				t.Fatal("server did not shut down in time")
			}
		})
	}
}
