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
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/internal/refalloc"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const allocCallTimeout = 5 * time.Second

// allocTestPlugin is a cost-source plugin that also serves AllocatorService
// by delegating to inner.
type allocTestPlugin struct {
	*pluginsdk.BasePlugin

	inner pluginsdk.AllocatorProvider
}

func newAllocTestPlugin(inner pluginsdk.AllocatorProvider) *allocTestPlugin {
	return &allocTestPlugin{BasePlugin: pluginsdk.NewBasePlugin("alloc-test"), inner: inner}
}

func (p *allocTestPlugin) Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	return p.inner.Allocate(ctx, req)
}

// allocatorFunc adapts a function to pluginsdk.AllocatorProvider.
type allocatorFunc func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)

func (f allocatorFunc) Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	return f(ctx, req)
}

//nolint:gochecknoglobals // Test table shared by the transport-parity tests.
var allocTransports = []struct {
	name string
	web  bool
}{
	{name: "grpc", web: false},
	{name: "connect", web: true},
}

func allocNodeUsage(node string, cpu, mem float64) []*pbc.UsageRow {
	subject := map[string]string{pluginsdk.SubjectKind: pluginsdk.KindNode, pluginsdk.SubjectNode: node}
	return []*pbc.UsageRow{
		{Subject: subject, Metric: pluginsdk.MetricCPUAllocatable, Amount: cpu, Unit: pluginsdk.UnitCore},
		{Subject: subject, Metric: pluginsdk.MetricMemAllocatable, Amount: mem, Unit: pluginsdk.UnitGiB},
	}
}

func allocWorkloadUsage(node, namespace, pod string, cpu, mem float64) []*pbc.UsageRow {
	subject := map[string]string{
		pluginsdk.SubjectKind:      pluginsdk.KindWorkload,
		pluginsdk.SubjectNamespace: namespace,
		pluginsdk.SubjectPod:       pod,
		pluginsdk.SubjectNode:      node,
	}
	return []*pbc.UsageRow{
		{Subject: subject, Metric: pluginsdk.MetricCPURequest, Amount: cpu, Unit: pluginsdk.UnitCore},
		{Subject: subject, Metric: pluginsdk.MetricMemRequest, Amount: mem, Unit: pluginsdk.UnitGiB},
	}
}

func allocPriced(kind, id string, cost float64, priced bool, note string) *pbc.PricedResource {
	currency := "USD"
	if !priced {
		currency = ""
	}
	return &pbc.PricedResource{
		Resource: &pbc.ResourceDescriptor{
			Id: id, Provider: "kubernetes", Tags: map[string]string{pluginsdk.SubjectKind: kind},
		},
		Cost:     cost,
		Currency: currency,
		Priced:   priced,
		Note:     note,
	}
}

// fixtureAllocateRequest is a two-node run-rate cluster (n1 costs 10, n2 costs
// 6) with two workloads per node, plus an unpriced node n3 running one workload.
func fixtureAllocateRequest() *pbc.AllocateRequest {
	var usage []*pbc.UsageRow
	for _, node := range []string{"n1", "n2"} {
		usage = append(usage, allocNodeUsage(node, 4, 16)...)
		usage = append(usage, allocWorkloadUsage(node, "payments", "api-"+node, 1, 4)...)
		usage = append(usage, allocWorkloadUsage(node, "web", "frontend-"+node, 0.5, 2)...)
	}
	usage = append(usage, allocNodeUsage("n3", 2, 8)...)
	usage = append(usage, allocWorkloadUsage("n3", "web", "batch", 1, 1)...)

	return &pbc.AllocateRequest{
		Usage: usage,
		Priced: []*pbc.PricedResource{
			allocPriced(pluginsdk.KindNode, "n1", 10, true, ""),
			allocPriced(pluginsdk.KindNode, "n2", 6, true, ""),
			allocPriced(pluginsdk.KindNode, "n3", 0, false, "no price"),
		},
		Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
	}
}

type allocateCall func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)

// startAllocServer serves plugin through pluginsdk.Serve on an injected
// loopback listener and returns its address and an Allocate client for the
// selected transport.
func startAllocServer(
	t *testing.T, plugin pluginsdk.Plugin, web bool, opts ...func(*pluginsdk.ServeConfig),
) (string, allocateCall) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()

	config := pluginsdk.ServeConfig{Plugin: plugin, Listener: listener, Web: pluginsdk.WebConfig{Enabled: web}}
	for _, opt := range opts {
		opt(&config)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- pluginsdk.Serve(ctx, config) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(allocCallTimeout):
			t.Error("server did not shut down in time")
		}
	})

	if web {
		client := pbcconnect.NewAllocatorServiceClient(http.DefaultClient, "http://"+addr)
		return addr, func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
			resp, callErr := client.Allocate(ctx, connect.NewRequest(req))
			if callErr != nil {
				return nil, callErr
			}
			return resp.Msg, nil
		}
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewAllocatorServiceClient(conn)
	return addr, func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		return client.Allocate(ctx, req)
	}
}

func callAllocate(t *testing.T, call allocateCall, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
	defer cancel()
	return call(ctx, req)
}

// allocWireCode returns the status code and message of an error from either transport.
func allocWireCode(t *testing.T, err error, web bool) (codes.Code, string) {
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

func allocRowsOfKind(resp *pbc.AllocateResponse, kind string) []*pbc.AllocationRow {
	var rows []*pbc.AllocationRow
	for _, row := range resp.GetRows() {
		if row.GetSubject()[pluginsdk.SubjectKind] == kind {
			rows = append(rows, row)
		}
	}
	return rows
}

func allocTotal(resp *pbc.AllocateResponse) float64 {
	total := 0.0
	for _, row := range resp.GetRows() {
		total += row.GetTotalCost()
	}
	return total
}

func TestAllocator_TransportParity(t *testing.T) {
	cases := []struct {
		name string
		req  *pbc.AllocateRequest
	}{
		{name: "fixture", req: fixtureAllocateRequest()},
		{name: "policy override", req: func() *pbc.AllocateRequest {
			req := fixtureAllocateRequest()
			req.PolicyJson = []byte(`{"node_split":{"cpu_weight":0.25}}`)
			return req
		}()},
		{name: "empty", req: &pbc.AllocateRequest{}},
	}

	calls := make(map[string]allocateCall, len(allocTransports))
	for _, tr := range allocTransports {
		_, calls[tr.name] = startAllocServer(t, newAllocTestPlugin(refalloc.New()), tr.web)
	}

	results := make(map[string]*pbc.AllocateResponse, len(cases))
	for _, tc := range cases {
		grpcResp, err := callAllocate(t, calls["grpc"], tc.req)
		require.NoError(t, err, tc.name)
		connectResp, err := callAllocate(t, calls["connect"], tc.req)
		require.NoError(t, err, tc.name)
		assert.True(t, proto.Equal(grpcResp, connectResp), "%s: gRPC and Connect responses differ", tc.name)
		results[tc.name] = grpcResp
	}

	fixture := results["fixture"]
	idleNodes := map[string]int{}
	for _, row := range allocRowsOfKind(fixture, pluginsdk.KindIdle) {
		idleNodes[row.GetSubject()[pluginsdk.SubjectNode]]++
	}
	assert.Equal(t, map[string]int{"n1": 1, "n2": 1}, idleNodes)

	var n3Rows []*pbc.AllocationRow
	for _, row := range allocRowsOfKind(fixture, pluginsdk.KindWorkload) {
		if row.GetSubject()[pluginsdk.SubjectNode] == "n3" {
			n3Rows = append(n3Rows, row)
		}
	}
	require.Len(t, n3Rows, 1)
	assert.Zero(t, n3Rows[0].GetTotalCost())
	assert.NotEmpty(t, n3Rows[0].GetNote())
	assert.InDelta(t, 16, allocTotal(fixture), 1e-9)

	assert.Contains(t, string(results["policy override"].GetEffectivePolicyJson()), `"cpu_weight":0.25`)
	assert.NotEqual(t, fixture.GetPolicyDigest(), results["policy override"].GetPolicyDigest())
	assert.Empty(t, results["empty"].GetRows())
	assert.NotEmpty(t, results["empty"].GetPolicyDigest())
}

func TestAllocator_ErrorParity(t *testing.T) {
	withPolicy := func(doc string) *pbc.AllocateRequest {
		req := fixtureAllocateRequest()
		req.PolicyJson = []byte(doc)
		return req
	}
	mixed := fixtureAllocateRequest()
	mixed.Priced[1].Currency = "EUR"

	failing := allocatorFunc(func(context.Context, *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		return nil, status.Error(codes.FailedPrecondition, "boom")
	})

	tests := []struct {
		name         string
		inner        pluginsdk.AllocatorProvider
		req          *pbc.AllocateRequest
		wantCode     codes.Code
		wantContains string
	}{
		{
			name: "unknown policy field", inner: refalloc.New(), req: withPolicy(`{"node_split":{"cpu":1}}`),
			wantCode: codes.InvalidArgument, wantContains: "node_split.cpu",
		},
		{
			name: "unknown policy version", inner: refalloc.New(), req: withPolicy(`{"version":99}`),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unpriced with cost", inner: refalloc.New(),
			req: &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
				allocPriced(pluginsdk.KindNode, "n1", 1, false, ""),
			}},
			wantCode: codes.InvalidArgument,
		},
		{name: "mixed currencies", inner: refalloc.New(), req: mixed, wantCode: codes.InvalidArgument},
		{
			name: "allocator failure", inner: failing, req: fixtureAllocateRequest(),
			wantCode: codes.FailedPrecondition, wantContains: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCodes [2]codes.Code
			var gotMessages [2]string
			for i, tr := range allocTransports {
				_, call := startAllocServer(t, newAllocTestPlugin(tt.inner), tr.web)
				_, err := callAllocate(t, call, tt.req)
				gotCodes[i], gotMessages[i] = allocWireCode(t, err, tr.web)
				assert.Equal(t, tt.wantCode, gotCodes[i], tr.name)
			}
			assert.Equal(t, gotMessages[0], gotMessages[1], "messages differ between transports")
			assert.NotContains(t, gotMessages[0], "rpc error:")
			if tt.wantContains != "" {
				assert.Contains(t, gotMessages[0], tt.wantContains)
			}
		})
	}
}

func TestAllocator_ThreeNodesAndControlPlane(t *testing.T) {
	var usage []*pbc.UsageRow
	priced := make([]*pbc.PricedResource, 0, 4)
	for i, node := range []string{"n1", "n2", "n3"} {
		usage = append(usage, allocNodeUsage(node, 4, 16)...)
		usage = append(usage, allocWorkloadUsage(node, "payments", "api-"+node, 1, 2)...)
		priced = append(priced, allocPriced(pluginsdk.KindNode, node, float64(4+i), true, ""))
	}
	priced = append(priced, allocPriced("cluster", "control-plane", 3, true, ""))

	_, call := startAllocServer(t, newAllocTestPlugin(refalloc.New()), false)
	resp, err := callAllocate(t, call, &pbc.AllocateRequest{Usage: usage, Priced: priced})
	require.NoError(t, err)

	assert.Len(t, allocRowsOfKind(resp, pluginsdk.KindIdle), 3)
	assert.NotEmpty(t, allocRowsOfKind(resp, pluginsdk.KindCluster))
	assert.InDelta(t, 4+5+6+3, allocTotal(resp), 1e-9)
}

func TestAllocator_HostVerification(t *testing.T) {
	_, call := startAllocServer(t, newAllocTestPlugin(refalloc.New()), false)
	req := fixtureAllocateRequest()
	resp, err := callAllocate(t, call, req)
	require.NoError(t, err)

	require.NoError(t, pluginsdk.CheckConservation(req, resp, pluginsdk.DefaultConservationEpsilon))
	require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))

	tampered := proto.Clone(resp).(*pbc.AllocateResponse)
	tampered.Rows[0].TotalCost++
	var consErr *plugintesting.ConservationError
	require.ErrorAs(t, pluginsdk.CheckConservation(req, tampered, pluginsdk.DefaultConservationEpsilon), &consErr)
	assert.InDelta(t, 1, consErr.Difference, 1e-9)
}

func TestAllocator_WrappersMatchTesting(t *testing.T) {
	requests := []*pbc.AllocateRequest{
		fixtureAllocateRequest(),
		{Priced: []*pbc.PricedResource{allocPriced(pluginsdk.KindNode, "n1", -1, true, "")}},
		{Priced: []*pbc.PricedResource{
			allocPriced(pluginsdk.KindNode, "n1", 1, true, ""),
			{Resource: &pbc.ResourceDescriptor{Id: "n2"}, Cost: 1, Currency: "EUR", Priced: true},
		}},
	}
	for i, req := range requests {
		assert.Equal(t, fmt.Sprint(plugintesting.ValidateAllocateRequest(req)),
			fmt.Sprint(pluginsdk.ValidateAllocateRequest(req)), "ValidateAllocateRequest #%d", i)

		wantCur, wantErr := plugintesting.ResolveCurrency(req.GetPriced())
		gotCur, gotErr := pluginsdk.ResolveCurrency(req.GetPriced())
		assert.Equal(t, wantCur, gotCur, "ResolveCurrency #%d", i)
		assert.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr), "ResolveCurrency #%d", i)
	}

	balanced, err := refalloc.New().Allocate(context.Background(), requests[0])
	require.NoError(t, err)
	conservation := []struct {
		resp    *pbc.AllocateResponse
		epsilon float64
	}{
		{resp: balanced, epsilon: pluginsdk.DefaultConservationEpsilon},
		{resp: &pbc.AllocateResponse{}, epsilon: pluginsdk.DefaultConservationEpsilon},
		{resp: balanced, epsilon: -1},
	}
	for i, tc := range conservation {
		assert.Equal(t, fmt.Sprint(plugintesting.CheckConservation(requests[0], tc.resp, tc.epsilon)),
			fmt.Sprint(pluginsdk.CheckConservation(requests[0], tc.resp, tc.epsilon)), "CheckConservation #%d", i)
	}
	assert.InDelta(t, plugintesting.DefaultConservationEpsilon, pluginsdk.DefaultConservationEpsilon, 0)
}

// checkServiceHealth calls grpc.health.v1.Health/Check over Connect for service.
func checkServiceHealth(t *testing.T, addr, service string) (*healthpb.HealthCheckResponse, error) {
	t.Helper()
	client := connect.NewClient[healthpb.HealthCheckRequest, healthpb.HealthCheckResponse](
		http.DefaultClient, "http://"+addr+"/grpc.health.v1.Health/Check")
	ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
	defer cancel()
	resp, err := client.CallUnary(ctx, connect.NewRequest(&healthpb.HealthCheckRequest{Service: service}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func TestAllocatorServe_HealthIncludesAllocator(t *testing.T) {
	addr, _ := startAllocServer(t, newAllocTestPlugin(refalloc.New()), true)

	resp, err := checkServiceHealth(t, addr, pbcconnect.AllocatorServiceName)
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
}

func TestAllocatorServe_NotRegisteredWithoutProvider(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			plugin := pluginsdk.NewBasePlugin("cost-only")
			addr, call := startAllocServer(t, plugin, tr.web)

			_, err := callAllocate(t, call, &pbc.AllocateRequest{})
			code, _ := allocWireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)

			if tr.web {
				resp, healthErr := checkServiceHealth(t, addr, pbcconnect.AllocatorServiceName)
				assert.NotEqual(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(healthErr))
			}
		})
	}
}

// usageAndAllocPlugin serves both UsageSourceService and AllocatorService.
type usageAndAllocPlugin struct {
	*allocTestPlugin
}

func (p *usageAndAllocPlugin) GetStats(context.Context, *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	return &pbc.GetStatsResponse{
		Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
		Rows: allocNodeUsage("n1", 4, 16)[:1],
	}, nil
}

func TestAllocatorServe_WithUsageSource(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			plugin := &usageAndAllocPlugin{allocTestPlugin: newAllocTestPlugin(refalloc.New())}
			addr, call := startAllocServer(t, plugin, tr.web)

			allocResp, err := callAllocate(t, call, fixtureAllocateRequest())
			require.NoError(t, err)
			assert.NotEmpty(t, allocResp.GetRows())

			ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
			defer cancel()
			var stats *pbc.GetStatsResponse
			if tr.web {
				client := pbcconnect.NewUsageSourceServiceClient(http.DefaultClient, "http://"+addr)
				resp, statsErr := client.GetStats(ctx, connect.NewRequest(&pbc.GetStatsRequest{}))
				require.NoError(t, statsErr)
				stats = resp.Msg
			} else {
				conn, dialErr := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				require.NoError(t, dialErr)
				defer conn.Close()
				resp, statsErr := pbc.NewUsageSourceServiceClient(conn).GetStats(ctx, &pbc.GetStatsRequest{})
				require.NoError(t, statsErr)
				stats = resp
			}
			assert.Len(t, stats.GetRows(), 1)

			if tr.web {
				for _, service := range []string{pbcconnect.UsageSourceServiceName, pbcconnect.AllocatorServiceName} {
					resp, healthErr := checkServiceHealth(t, addr, service)
					require.NoError(t, healthErr, service)
					assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus(), service)
				}
			}
		})
	}
}

// Interceptors are gRPC server options, so they apply to every service on the
// gRPC server. In Connect mode ServeConfig.UnaryInterceptors do not apply to
// any service, exactly as for CostSourceService and UsageSourceService.
func TestAllocatorServe_InterceptorsApply(t *testing.T) {
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

	_, call := startAllocServer(t, newAllocTestPlugin(refalloc.New()), false, func(c *pluginsdk.ServeConfig) {
		c.UnaryInterceptors = []grpc.UnaryServerInterceptor{counting}
	})
	_, err := callAllocate(t, call, &pbc.AllocateRequest{})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{pbc.AllocatorService_Allocate_FullMethodName}, methods)
}
