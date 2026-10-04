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

//nolint:testpackage // Exercises unexported Server state (TypeRegistry) and test plugins.
package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// errorHandlerPlugin implements every optional provider interface that pluginsdk.Server
// wraps, and each one returns err.
type errorHandlerPlugin struct {
	mockPlugin

	err error
}

func (p *errorHandlerPlugin) HandleDryRun(context.Context, *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) Supports(context.Context, *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) GetRecommendations(
	context.Context, *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) GetBudgets(context.Context, *pbc.GetBudgetsRequest) (*pbc.GetBudgetsResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) DismissRecommendation(
	context.Context, *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) BatchCost(context.Context, *pbc.BatchCostRequest) (*pbc.BatchCostResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) ResolveResourceTypes(
	context.Context, *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	return nil, p.err
}

func (p *errorHandlerPlugin) GetPluginInfo(
	context.Context, *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return nil, p.err
}

// stubEmbeddingPlugin embeds the generated stub, the usual forward-compatibility
// pattern, and implements only the required Plugin methods.
type stubEmbeddingPlugin struct {
	pbc.UnimplementedCostSourceServiceServer

	base mockPlugin
}

func (p *stubEmbeddingPlugin) Name() string { return p.base.Name() }

func (p *stubEmbeddingPlugin) GetProjectedCost(
	ctx context.Context, req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return p.base.GetProjectedCost(ctx, req)
}

func (p *stubEmbeddingPlugin) GetActualCost(
	ctx context.Context, req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return p.base.GetActualCost(ctx, req)
}

func (p *stubEmbeddingPlugin) GetPricingSpec(
	ctx context.Context, req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return p.base.GetPricingSpec(ctx, req)
}

func (p *stubEmbeddingPlugin) EstimateCost(
	ctx context.Context, req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return p.base.EstimateCost(ctx, req)
}

// grpcStatusError is a custom error type that carries a gRPC status.
type grpcStatusError struct{ msg string }

func (e grpcStatusError) Error() string { return e.msg }

func (e grpcStatusError) GRPCStatus() *status.Status {
	return status.New(codes.FailedPrecondition, e.msg)
}

func neutralResource() *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{Provider: "custom", ResourceType: "instance", Sku: "standard", Region: "region-1"}
}

// wrappedRPC calls one RPC that pluginsdk.Server wraps, over gRPC-style method calls and
// over the Connect handler.
type wrappedRPC struct {
	name        string
	internalMsg string
	call        func(context.Context, *Server) (proto.Message, error)
	connect     func(context.Context, *ConnectHandler) error
}

func wrappedRPCs() []wrappedRPC {
	resource := neutralResource()
	dryRun := &pbc.DryRunRequest{Resource: resource}
	supports := &pbc.SupportsRequest{Resource: resource}
	recs := &pbc.GetRecommendationsRequest{ProjectionPeriod: "monthly"}
	budgets := &pbc.GetBudgetsRequest{}
	dismiss := &pbc.DismissRecommendationRequest{RecommendationId: "rec-1"}
	batch := &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
		Resources: []*pbc.ResourceDescriptor{resource},
	}
	resolve := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"custom_instance"},
	}
	info := &pbc.GetPluginInfoRequest{}

	return []wrappedRPC{
		{
			name: "DryRun", internalMsg: "plugin failed to execute DryRun",
			call: func(ctx context.Context, s *Server) (proto.Message, error) { return s.DryRun(ctx, dryRun) },
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.DryRun(ctx, connect.NewRequest(dryRun))
				return err
			},
		},
		{
			name: "Supports", internalMsg: "plugin failed to execute",
			call: func(ctx context.Context, s *Server) (proto.Message, error) { return s.Supports(ctx, supports) },
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.Supports(ctx, connect.NewRequest(supports))
				return err
			},
		},
		{
			name: "GetRecommendations", internalMsg: "plugin failed to execute GetRecommendations",
			call: func(ctx context.Context, s *Server) (proto.Message, error) {
				return s.GetRecommendations(ctx, recs)
			},
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.GetRecommendations(ctx, connect.NewRequest(recs))
				return err
			},
		},
		{
			name: "GetBudgets", internalMsg: "plugin failed to execute GetBudgets",
			call: func(ctx context.Context, s *Server) (proto.Message, error) { return s.GetBudgets(ctx, budgets) },
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.GetBudgets(ctx, connect.NewRequest(budgets))
				return err
			},
		},
		{
			name: "DismissRecommendation", internalMsg: "plugin failed to execute DismissRecommendation",
			call: func(ctx context.Context, s *Server) (proto.Message, error) {
				return s.DismissRecommendation(ctx, dismiss)
			},
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.DismissRecommendation(ctx, connect.NewRequest(dismiss))
				return err
			},
		},
		{
			name: "BatchCost", internalMsg: "plugin failed to execute BatchCost",
			call: func(ctx context.Context, s *Server) (proto.Message, error) { return s.BatchCost(ctx, batch) },
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.BatchCost(ctx, connect.NewRequest(batch))
				return err
			},
		},
		{
			name: "ResolveResourceTypes", internalMsg: "plugin failed to execute ResolveResourceTypes",
			call: func(ctx context.Context, s *Server) (proto.Message, error) {
				return s.ResolveResourceTypes(ctx, resolve)
			},
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.ResolveResourceTypes(ctx, connect.NewRequest(resolve))
				return err
			},
		},
		{
			name: "GetPluginInfo", internalMsg: "plugin failed to retrieve metadata",
			call: func(ctx context.Context, s *Server) (proto.Message, error) { return s.GetPluginInfo(ctx, info) },
			connect: func(ctx context.Context, h *ConnectHandler) error {
				_, err := h.GetPluginInfo(ctx, connect.NewRequest(info))
				return err
			},
		},
	}
}

func errorServer(err error) *Server {
	return NewServer(&errorHandlerPlugin{mockPlugin: mockPlugin{name: "neutral"}, err: err})
}

func TestHandlerStatusPassThrough(t *testing.T) {
	handlerErrors := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			"invalid argument",
			status.Error(codes.InvalidArgument, "sku is required"),
			codes.InvalidArgument,
			"sku is required",
		},
		{"not found", status.Error(codes.NotFound, "no such resource"), codes.NotFound, "no such resource"},
		{
			"unavailable",
			status.Error(codes.Unavailable, "pricing source down"),
			codes.Unavailable,
			"pricing source down",
		},
		{
			"wrapped status",
			fmt.Errorf("lookup: %w", status.Error(codes.InvalidArgument, "bad region")),
			codes.InvalidArgument, "bad region",
		},
		{"GRPCStatus type", grpcStatusError{msg: "not configured"}, codes.FailedPrecondition, "not configured"},
	}

	for _, rpc := range wrappedRPCs() {
		for _, he := range handlerErrors {
			t.Run(rpc.name+"/"+he.name, func(t *testing.T) {
				_, err := rpc.call(context.Background(), errorServer(he.err))
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, he.wantCode, st.Code())
				assert.Contains(t, st.Message(), he.wantMsg)
			})
		}
	}
}

func TestHandlerStatusInternal(t *testing.T) {
	handlerErrors := []struct {
		name   string
		err    error
		secret string
	}{
		{"plain error", errors.New("db password is hunter2"), "hunter2"},
		{"context canceled", context.Canceled, "canceled"},
		{"unknown status", status.Error(codes.Unknown, "stack trace at line 42"), "line 42"},
	}

	for _, rpc := range wrappedRPCs() {
		for _, he := range handlerErrors {
			t.Run(rpc.name+"/"+he.name, func(t *testing.T) {
				_, err := rpc.call(context.Background(), errorServer(he.err))
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Internal, st.Code())
				assert.Equal(t, rpc.internalMsg, st.Message())
				assert.NotContains(t, st.Message(), he.secret)
			})
		}
	}
}

func TestHandlerStatusConnect(t *testing.T) {
	for _, rpc := range wrappedRPCs() {
		t.Run(rpc.name, func(t *testing.T) {
			h := NewConnectHandler(errorServer(status.Error(codes.InvalidArgument, "sku is required")))
			err := rpc.connect(context.Background(), h)
			var connectErr *connect.Error
			require.ErrorAs(t, err, &connectErr)
			assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
			assert.Equal(t, "sku is required", connectErr.Message())

			h = NewConnectHandler(errorServer(errors.New("db password is hunter2")))
			err = rpc.connect(context.Background(), h)
			require.ErrorAs(t, err, &connectErr)
			assert.Equal(t, connect.CodeInternal, connectErr.Code())
			assert.Equal(t, rpc.internalMsg, connectErr.Message())
		})
	}
}

// explicitInfo fixes capabilities so the comparison below does not depend on
// capability inference, which sees the stub's methods.
func explicitInfo() *PluginInfo {
	return NewPluginInfo("neutral", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS))
}

// compareRPC calls rpc on both servers and requires the same code and response.
func compareRPC(t *testing.T, rpc wrappedRPC, want, got *Server) {
	t.Helper()
	wantResp, wantErr := rpc.call(context.Background(), want)
	gotResp, gotErr := rpc.call(context.Background(), got)
	assert.Equal(t, status.Code(wantErr), status.Code(gotErr), "code")
	if wantErr != nil {
		assert.Equal(t, status.Convert(wantErr).Message(), status.Convert(gotErr).Message(), "message")
		return
	}
	require.NoError(t, gotErr)
	assert.True(t, proto.Equal(wantResp, gotResp), "response\nwant: %v\ngot:  %v", wantResp, gotResp)
}

func TestHandlerStatusUnimplementedFallsBack(t *testing.T) {
	unimplemented := status.Error(codes.Unimplemented, "method not implemented")

	for _, withInfo := range []bool{false, true} {
		for _, rpc := range wrappedRPCs() {
			t.Run(fmt.Sprintf("%s/info=%v", rpc.name, withInfo), func(t *testing.T) {
				var info *PluginInfo
				if withInfo {
					info = explicitInfo()
				}
				plain := NewServerWithOptions(&mockPlugin{name: "neutral"}, nil, nil, info)
				erroring := NewServerWithOptions(
					&errorHandlerPlugin{mockPlugin: mockPlugin{name: "neutral"}, err: unimplemented}, nil, nil, info)
				if !withInfo {
					// Without explicit capabilities the two servers infer different sets,
					// which Supports echoes; align them to compare the rest of the response.
					erroring.globalCapabilities = plain.globalCapabilities
				}
				compareRPC(t, rpc, plain, erroring)
			})
		}
	}
}

func TestHandlerStatusUnimplementedUsesTypeRegistry(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMapping(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, "custom_instance", "custom:index:Instance", true)

	server := errorServer(status.Error(codes.Unimplemented, "method not implemented"))
	server.setTypeRegistry(registry, true)

	resp, err := server.ResolveResourceTypes(context.Background(), &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"custom_instance"},
	})
	require.NoError(t, err)
	assert.Contains(t, resp.GetMappings(), "custom_instance")
}

func TestStubEmbeddingMatchesPlainPlugin(t *testing.T) {
	for _, rpc := range wrappedRPCs() {
		if rpc.name == "DryRun" {
			continue // HandleDryRun is not a stub method
		}
		t.Run(rpc.name, func(t *testing.T) {
			plain := NewServerWithOptions(&mockPlugin{name: "neutral"}, nil, nil, explicitInfo())
			stub := NewServerWithOptions(
				&stubEmbeddingPlugin{base: mockPlugin{name: "neutral"}}, nil, nil, explicitInfo())
			compareRPC(t, rpc, plain, stub)
		})
	}
}

func TestStubEmbeddingPassesBudgetsConformance(t *testing.T) {
	result, err := RunStandardConformance(&stubEmbeddingPlugin{base: mockPlugin{name: "neutral"}})
	require.NoError(t, err)

	var budgets int
	for _, category := range result.Categories {
		for _, r := range category.Results {
			if r.Method == plugintesting.MethodGetBudgets && category.Name == plugintesting.CategoryRPCCorrectness {
				budgets++
				assert.True(t, r.Success, "%s: %v", r.Details, r.Error)
			}
		}
	}
	assert.Equal(t, 1, budgets)
}
