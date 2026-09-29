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

//nolint:testpackage // Exercises the unexported contract commitment adapters through Serve
package pluginsdk

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
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
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

type getCommitmentsFunc func(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error)

// commitmentTestPlugin is a cost plugin that also serves contract commitments
// from the reference producer, or fails with err when it is set.
type commitmentTestPlugin struct {
	*BasePlugin

	source *plugintesting.MockContractCommitmentSource
	err    error
}

func newCommitmentTestPlugin(t testing.TB) *commitmentTestPlugin {
	t.Helper()
	source, err := plugintesting.NewMockContractCommitmentSource(fixtureCommitments())
	require.NoError(t, err)
	return &commitmentTestPlugin{BasePlugin: NewBasePlugin("commitments"), source: source}
}

func (p *commitmentTestPlugin) GetContractCommitments(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.source.GetContractCommitments(ctx, req)
}

func fixtureCommitments() []*pbc.ContractCommitment {
	out := make([]*pbc.ContractCommitment, 0, 5)
	for i := range 5 {
		c, err := NewContractCommitmentBuilder().
			WithIdentity(fmt.Sprintf("cc-%d", i), "contract-1").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
			WithCommitmentPeriod(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
			WithFinancials(1200, 0, "", "USD").
			WithBaselineTerms(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build()
		if err != nil {
			panic(err)
		}
		out = append(out, c)
	}
	return out
}

// startCommitmentServer serves plugin on an injected loopback listener and
// returns its address and a GetContractCommitments client for the transport.
func startCommitmentServer(
	t *testing.T, plugin Plugin, web bool, opts ...func(*ServeConfig),
) (string, getCommitmentsFunc) {
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
		client := pbcconnect.NewSupplementalDatasetServiceClient(http.DefaultClient, "http://"+addr)
		return addr, func(
			ctx context.Context, req *pbc.GetContractCommitmentsRequest,
		) (*pbc.GetContractCommitmentsResponse, error) {
			resp, callErr := client.GetContractCommitments(ctx, connect.NewRequest(req))
			if callErr != nil {
				return nil, callErr
			}
			return resp.Msg, nil
		}
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewSupplementalDatasetServiceClient(conn)
	return addr, func(
		ctx context.Context, req *pbc.GetContractCommitmentsRequest,
	) (*pbc.GetContractCommitmentsResponse, error) {
		return client.GetContractCommitments(ctx, req)
	}
}

func callCommitments(
	t *testing.T, call getCommitmentsFunc, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
	defer cancel()
	return call(ctx, req)
}

func TestContractCommitmentsServeTransportParity(t *testing.T) {
	req := &pbc.GetContractCommitmentsRequest{PageSize: 2}
	responses := make(map[string]*pbc.GetContractCommitmentsResponse, len(usageTransports))
	for _, tr := range usageTransports {
		_, call := startCommitmentServer(t, newCommitmentTestPlugin(t), tr.web)

		resp, err := callCommitments(t, call, req)
		require.NoError(t, err, tr.name)
		require.NoError(t, ValidateGetContractCommitmentsResponse(req, resp), tr.name)
		assert.Len(t, resp.GetCommitments(), 2, tr.name)
		assert.Equal(t, int32(5), resp.GetTotalCount(), tr.name)
		assert.NotEmpty(t, resp.GetNextPageToken(), tr.name)
		responses[tr.name] = resp
	}
	assert.True(t, proto.Equal(responses["grpc"], responses["connect"]), "gRPC and Connect responses differ")
}

func TestContractCommitmentsErrorParity(t *testing.T) {
	start := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		failure  error
		req      *pbc.GetContractCommitmentsRequest
		wantCode codes.Code
	}{
		{name: "only start", req: &pbc.GetContractCommitmentsRequest{Start: timestamppb.New(start)},
			wantCode: codes.InvalidArgument},
		{name: "inverted window", req: &pbc.GetContractCommitmentsRequest{
			Start: timestamppb.New(start), End: timestamppb.New(start.Add(-time.Hour)),
		}, wantCode: codes.InvalidArgument},
		{name: "negative page size", req: &pbc.GetContractCommitmentsRequest{PageSize: -1},
			wantCode: codes.InvalidArgument},
		{name: "malformed token", req: &pbc.GetContractCommitmentsRequest{PageToken: "!not-a-token!"},
			wantCode: codes.InvalidArgument},
		{name: "permission denied", failure: status.Error(codes.PermissionDenied, "cannot read reservations"),
			req: &pbc.GetContractCommitmentsRequest{}, wantCode: codes.PermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotCodes [2]codes.Code
			var gotMessages [2]string
			for i, tr := range usageTransports {
				plugin := newCommitmentTestPlugin(t)
				plugin.err = tt.failure
				_, call := startCommitmentServer(t, plugin, tr.web)

				_, err := callCommitments(t, call, tt.req)
				gotCodes[i], gotMessages[i] = wireCode(t, err, tr.web)
				assert.Equal(t, tt.wantCode, gotCodes[i], tr.name)
				assert.NotContains(t, gotMessages[i], "rpc error:", tr.name)
			}
			assert.Equal(t, gotCodes[0], gotCodes[1], "codes differ between transports")
			assert.Equal(t, gotMessages[0], gotMessages[1], "messages differ between transports")
		})
	}
}

func checkCommitmentHealth(t *testing.T, addr string) (*healthpb.HealthCheckResponse, error) {
	t.Helper()
	client := connect.NewClient[healthpb.HealthCheckRequest, healthpb.HealthCheckResponse](
		http.DefaultClient, "http://"+addr+"/grpc.health.v1.Health/Check")
	ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
	defer cancel()
	resp, err := client.CallUnary(ctx, connect.NewRequest(&healthpb.HealthCheckRequest{
		Service: pbcconnect.SupplementalDatasetServiceName,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func TestContractCommitmentsHealth(t *testing.T) {
	addr, _ := startCommitmentServer(t, newCommitmentTestPlugin(t), true)

	resp, err := checkCommitmentHealth(t, addr)
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
}

func TestContractCommitmentsNotRegistered(t *testing.T) {
	for _, tr := range usageTransports {
		t.Run(tr.name, func(t *testing.T) {
			addr, call := startCommitmentServer(t, NewBasePlugin("cost-only"), tr.web)

			_, err := callCommitments(t, call, &pbc.GetContractCommitmentsRequest{})
			code, _ := wireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)

			if tr.web {
				_, healthErr := checkCommitmentHealth(t, addr)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(healthErr))
			}
		})
	}
}

func TestContractCommitmentsNoStartupWarning(t *testing.T) {
	var logs syncBuffer
	logger := zerolog.New(&logs)
	_, call := startCommitmentServer(t, newCommitmentTestPlugin(t), false, func(c *ServeConfig) {
		c.Logger = &logger
	})

	_, err := callCommitments(t, call, &pbc.GetContractCommitmentsRequest{})
	require.NoError(t, err)
	assert.NotContains(t, logs.String(), "PluginInfo.Capabilities")
}

func TestInferCapabilities_ContractCommitments(t *testing.T) {
	server := NewServerWithOptions(newCommitmentTestPlugin(t), nil, nil,
		&PluginInfo{Name: "commitments", Version: "1.0.0", SpecVersion: "1.0.0"})

	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS)
	assert.Equal(t, "true", resp.GetMetadata()["supports_contract_commitments"])
}

func TestInferCapabilities_NoContractCommitments(t *testing.T) {
	caps := inferCapabilities(&mockPlugin{name: "basic-plugin"})
	assert.NotContains(t, caps, pbc.PluginCapability_PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS)
	metadata, _ := CapabilitiesToLegacyMetadataWithWarnings(caps)
	assert.NotContains(t, metadata, "supports_contract_commitments")
}

func TestExplicitCapabilities_OverrideContractCommitments(t *testing.T) {
	info := NewPluginInfo("commitments", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS))
	server := NewServerWithOptions(newCommitmentTestPlugin(t), nil, nil, info)

	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t,
		[]pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS},
		resp.GetCapabilities())
}

func TestContractCommitmentWrappersDelegate(t *testing.T) {
	good := fixtureCommitments()[0]
	bad := proto.Clone(good).(*pbc.ContractCommitment)
	bad.ContractCommitmentCost = math.NaN()
	window := &pbc.GetContractCommitmentsRequest{
		Start: timestamppb.New(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
		End:   timestamppb.New(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)),
	}
	badReq := &pbc.GetContractCommitmentsRequest{PageSize: -1}
	resp := &pbc.GetContractCommitmentsResponse{Commitments: []*pbc.ContractCommitment{good, good}, TotalCount: 2}

	assert.Equal(t, plugintesting.ValidateContractCommitment(good), ValidateContractCommitment(good))
	assert.Equal(t, plugintesting.ValidateContractCommitment(bad).Error(), ValidateContractCommitment(bad).Error())
	assert.Equal(t, plugintesting.ValidateGetContractCommitmentsRequest(badReq).Error(),
		ValidateGetContractCommitmentsRequest(badReq).Error())
	assert.Equal(t, plugintesting.ValidateGetContractCommitmentsResponse(window, resp).Error(),
		ValidateGetContractCommitmentsResponse(window, resp).Error())
	assert.Equal(t,
		plugintesting.ContractCommitmentMatchesWindow(good, window.GetStart(), window.GetEnd()),
		ContractCommitmentMatchesWindow(good, window.GetStart(), window.GetEnd()))

	page, next, total, err := PaginateContractCommitments(fixtureCommitments(), 2, "")
	require.NoError(t, err)
	assert.Len(t, page, 2)
	assert.Equal(t, EncodePageToken(2), next, "tokens must match the existing page-token helpers")
	assert.Equal(t, int32(5), total)
}

func TestContractCommitmentPageSizeConstantsMatch(t *testing.T) {
	assert.Equal(t, DefaultPageSize, plugintesting.DefaultPageSize)
	assert.Equal(t, MaxPageSize, plugintesting.MaxPageSize)
}

func TestContractCommitmentBuilderRejectsNonFinite(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1)} {
		_, err := NewContractCommitmentBuilder().
			WithIdentity("cc", "contract").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
			WithQuantity(v, "Hours").
			WithCurrency("USD").
			Build()
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "contract_commitment_quantity must be finite"), err.Error())
	}
}

func BenchmarkPluginsdkValidateContractCommitment(b *testing.B) {
	c := fixtureCommitments()[0]
	b.ReportAllocs()
	for b.Loop() {
		if err := ValidateContractCommitment(c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginsdkValidateGetContractCommitmentsRequest(b *testing.B) {
	req := &pbc.GetContractCommitmentsRequest{PageSize: 50}
	b.ReportAllocs()
	for b.Loop() {
		if err := ValidateGetContractCommitmentsRequest(req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginsdkValidateGetContractCommitmentsResponse(b *testing.B) {
	req := &pbc.GetContractCommitmentsRequest{}
	resp := &pbc.GetContractCommitmentsResponse{Commitments: fixtureCommitments(), TotalCount: 5}
	b.ReportAllocs()
	for b.Loop() {
		if err := ValidateGetContractCommitmentsResponse(req, resp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginsdkContractCommitmentMatchesWindow(b *testing.B) {
	c := fixtureCommitments()[0]
	start := timestamppb.New(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC))
	end := timestamppb.New(time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC))
	b.ReportAllocs()
	for b.Loop() {
		if !ContractCommitmentMatchesWindow(c, start, end) {
			b.Fatal("expected a match")
		}
	}
}

func BenchmarkPluginsdkPaginateContractCommitments(b *testing.B) {
	all := fixtureCommitments()
	b.ReportAllocs()
	for b.Loop() {
		if _, _, _, err := PaginateContractCommitments(all, 2, ""); err != nil {
			b.Fatal(err)
		}
	}
}
