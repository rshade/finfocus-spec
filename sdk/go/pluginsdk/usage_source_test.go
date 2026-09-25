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

//nolint:testpackage // Exercises the unexported usage-source adapters through Serve
package pluginsdk

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

const usageCallTimeout = 5 * time.Second

type getStatsFunc func(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)

//nolint:gochecknoglobals // Test table shared by the transport-parity tests.
var usageTransports = []struct {
	name string
	web  bool
}{
	{name: "grpc", web: false},
	{name: "connect", web: true},
}

// startUsageServer serves plugin on an injected loopback listener and returns
// its address and a GetStats client for the selected transport.
func startUsageServer(t *testing.T, plugin Plugin, web bool, opts ...func(*ServeConfig)) (string, getStatsFunc) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()

	config := ServeConfig{Plugin: plugin, Listener: listener, Web: WebConfig{Enabled: web}}
	for _, opt := range opts {
		opt(&config)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- Serve(ctx, config) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(usageCallTimeout):
			t.Error("server did not shut down in time")
		}
	})

	if web {
		client := pbcconnect.NewUsageSourceServiceClient(http.DefaultClient, "http://"+addr)
		return addr, func(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
			resp, callErr := client.GetStats(ctx, connect.NewRequest(req))
			if callErr != nil {
				return nil, callErr
			}
			return resp.Msg, nil
		}
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewUsageSourceServiceClient(conn)
	return addr, func(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
		return client.GetStats(ctx, req)
	}
}

func callGetStats(t *testing.T, call getStatsFunc, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
	defer cancel()
	return call(ctx, req)
}

// wireCode returns the status code and message of an error from either transport.
func wireCode(t *testing.T, err error, web bool) (codes.Code, string) {
	t.Helper()
	require.Error(t, err)
	if web {
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		return codes.Code(connectErr.Code()), connectErr.Message()
	}
	st := status.Convert(err)
	return st.Code(), st.Message()
}

func TestUsageSourceServeTransportParity(t *testing.T) {
	responses := make(map[string]*pbc.GetStatsResponse, len(usageTransports))
	for _, tr := range usageTransports {
		_, call := startUsageServer(t, newUsageTestPlugin(), tr.web)

		resp, err := callGetStats(t, call, &pbc.GetStatsRequest{})
		require.NoError(t, err, tr.name)
		assert.True(t, proto.Equal(fixtureStatsResponse(), resp), "%s response differs from fixture", tr.name)
		assert.Equal(t, pbc.StatsMode_STATS_MODE_RUN_RATE, resp.GetMode())
		responses[tr.name] = resp
	}
	assert.True(t, proto.Equal(responses["grpc"], responses["connect"]), "gRPC and Connect responses differ")
}

func TestUsageSourceErrorParity(t *testing.T) {
	start := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		failure     error
		req         *pbc.GetStatsRequest
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:     "only start set",
			req:      &pbc.GetStatsRequest{Start: timestamppb.New(start)},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "start after end",
			req: &pbc.GetStatsRequest{
				Start: timestamppb.New(start.Add(time.Hour)),
				End:   timestamppb.New(start),
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "historical on run-rate-only source",
			req: &pbc.GetStatsRequest{
				Start: timestamppb.New(start),
				End:   timestamppb.New(start.Add(time.Hour)),
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "rbac failure",
			failure:     status.Error(codes.PermissionDenied, "cannot list pods"),
			req:         &pbc.GetStatsRequest{},
			wantCode:    codes.PermissionDenied,
			wantMessage: "cannot list pods",
		},
		{
			name:     "no credentials",
			failure:  status.Error(codes.Unauthenticated, "no usable credentials"),
			req:      &pbc.GetStatsRequest{},
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCodes [2]codes.Code
			var gotMessages [2]string
			for i, tr := range usageTransports {
				source := newReferenceUsageSource()
				source.failure = tt.failure
				_, call := startUsageServer(t, source, tr.web)

				_, err := callGetStats(t, call, tt.req)
				gotCodes[i], gotMessages[i] = wireCode(t, err, tr.web)
				assert.Equal(t, tt.wantCode, gotCodes[i], tr.name)
				assert.NotEqual(t, codes.Unknown, gotCodes[i], tr.name)
			}
			assert.Equal(t, gotCodes[0], gotCodes[1], "codes differ between transports")
			assert.Equal(t, gotMessages[0], gotMessages[1], "messages differ between transports")
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, gotMessages[0])
			}
		})
	}
}

func TestUsageSourceSelectorNamespace(t *testing.T) {
	_, call := startUsageServer(t, newReferenceUsageSource(), false)

	resp, err := callGetStats(t, call, &pbc.GetStatsRequest{
		Selector: map[string]string{SubjectNamespace: "payments"},
	})
	require.NoError(t, err)

	workloads := 0
	for _, row := range resp.GetRows() {
		if row.GetSubject()[SubjectKind] != KindWorkload {
			continue
		}
		workloads++
		assert.Equal(t, "payments", row.GetSubject()[SubjectNamespace])
	}
	assert.Positive(t, workloads, "expected at least one payments workload row")
}

func TestUsageSourceInterceptors(t *testing.T) {
	var mu sync.Mutex
	var methods []string
	counting := func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		mu.Lock()
		methods = append(methods, info.FullMethod)
		mu.Unlock()
		return handler(ctx, req)
	}

	_, call := startUsageServer(t, newUsageTestPlugin(), false, func(c *ServeConfig) {
		c.UnaryInterceptors = []grpc.UnaryServerInterceptor{counting}
	})

	_, err := callGetStats(t, call, &pbc.GetStatsRequest{})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{pbc.UsageSourceService_GetStats_FullMethodName}, methods)
}

func TestUsageSourceHistoricalParity(t *testing.T) {
	start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	req := &pbc.GetStatsRequest{
		Start: timestamppb.New(start),
		End:   timestamppb.New(start.Add(2 * time.Hour)),
	}

	responses := make(map[string]*pbc.GetStatsResponse, len(usageTransports))
	for _, tr := range usageTransports {
		source := newReferenceUsageSource()
		source.historical = true
		_, call := startUsageServer(t, source, tr.web)

		resp, err := callGetStats(t, call, req)
		require.NoError(t, err, tr.name)
		assert.Equal(t, pbc.StatsMode_STATS_MODE_HISTORICAL, resp.GetMode())
		require.NotEmpty(t, resp.GetRows())
		for _, row := range resp.GetRows() {
			assert.Contains(t, []string{UnitCoreHours, UnitGiBHours}, row.GetUnit(),
				"%s row %s/%s", tr.name, row.GetSubject()[SubjectKind], row.GetMetric())
		}
		responses[tr.name] = resp
	}
	assert.True(t, proto.Equal(responses["grpc"], responses["connect"]), "gRPC and Connect responses differ")
}

func TestUsageSourceUnknownMetric(t *testing.T) {
	_, call := startUsageServer(t, newReferenceUsageSource(), false)

	resp, err := callGetStats(t, call, &pbc.GetStatsRequest{
		Metrics: []string{MetricCPURequest, "gpu_seconds"},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetWarnings(), 1)
	assert.Contains(t, resp.GetWarnings()[0], "gpu_seconds")
}

// checkUsageHealth calls grpc.health.v1.Health/Check over Connect for the
// usage service.
func checkUsageHealth(t *testing.T, addr string) (*healthpb.HealthCheckResponse, error) {
	t.Helper()
	client := connect.NewClient[healthpb.HealthCheckRequest, healthpb.HealthCheckResponse](
		http.DefaultClient, "http://"+addr+"/grpc.health.v1.Health/Check")
	ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
	defer cancel()
	resp, err := client.CallUnary(ctx, connect.NewRequest(&healthpb.HealthCheckRequest{
		Service: pbcconnect.UsageSourceServiceName,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func TestUsageSourceHealth(t *testing.T) {
	addr, _ := startUsageServer(t, newUsageTestPlugin(), true)

	resp, err := checkUsageHealth(t, addr)
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
}

func TestUsageSourceNotRegistered(t *testing.T) {
	for _, tr := range usageTransports {
		t.Run(tr.name, func(t *testing.T) {
			addr, call := startUsageServer(t, NewBasePlugin("cost-only"), tr.web)

			_, err := callGetStats(t, call, &pbc.GetStatsRequest{})
			code, _ := wireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)

			if tr.web {
				_, healthErr := checkUsageHealth(t, addr)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(healthErr))
			}
		})
	}
}

// usageInfoPlugin is a usage source that also supplies its own plugin info.
type usageInfoPlugin struct {
	*usageTestPlugin
}

func (p *usageInfoPlugin) GetPluginInfo(
	_ context.Context, _ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:         "usage-dynamic",
		Version:      "v1.0.0",
		SpecVersion:  SpecVersion,
		Capabilities: []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS},
	}, nil
}

func TestUsageSourceWarning(t *testing.T) {
	explicit := NewPluginInfo("usage", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS))

	tests := []struct {
		name     string
		plugin   Plugin
		info     *PluginInfo
		wantWarn int
	}{
		{name: "inferred capabilities", plugin: newUsageTestPlugin(), wantWarn: 1},
		{name: "explicit capabilities", plugin: newUsageTestPlugin(), info: explicit},
		{name: "plugin info provider", plugin: &usageInfoPlugin{usageTestPlugin: newUsageTestPlugin()}},
		{name: "not a usage source", plugin: NewBasePlugin("cost-only")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs syncBuffer
			logger := zerolog.New(&logs)
			_, call := startUsageServer(t, tt.plugin, false, func(c *ServeConfig) {
				c.Logger = &logger
				c.PluginInfo = tt.info
			})

			// A completed RPC proves Serve finished its startup logging.
			_, _ = callGetStats(t, call, &pbc.GetStatsRequest{})

			warnings := 0
			for _, line := range strings.Split(logs.String(), "\n") {
				if strings.Contains(line, `"level":"warn"`) && strings.Contains(line, "PluginInfo.Capabilities") {
					warnings++
				}
			}
			assert.Equal(t, tt.wantWarn, warnings, "log output:\n%s", logs.String())
		})
	}
}
