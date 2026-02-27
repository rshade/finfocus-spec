// Package pluginsdk provides a development SDK for FinFocus plugins.
package pluginsdk

import (
	"context"

	"connectrpc.com/connect"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

// ConnectHandler adapts a Server to the pbcconnect.CostSourceServiceHandler interface.
// This enables the plugin to be served via connect-go, supporting gRPC, gRPC-Web,
// and Connect protocols simultaneously.
type ConnectHandler struct {
	pbcconnect.UnimplementedCostSourceServiceHandler

	server *Server
}

// NewConnectHandler creates a new ConnectHandler that wraps the given Server.
// Panics if server is nil to fail fast on misconfiguration.
func NewConnectHandler(server *Server) *ConnectHandler {
	if server == nil {
		panic("NewConnectHandler: server cannot be nil")
	}
	return &ConnectHandler{server: server}
}

// Name implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) Name(
	ctx context.Context,
	req *connect.Request[pbc.NameRequest],
) (*connect.Response[pbc.NameResponse], error) {
	resp, err := h.server.Name(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// Supports implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) Supports(
	ctx context.Context,
	req *connect.Request[pbc.SupportsRequest],
) (*connect.Response[pbc.SupportsResponse], error) {
	resp, err := h.server.Supports(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetActualCost implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetActualCost(
	ctx context.Context,
	req *connect.Request[pbc.GetActualCostRequest],
) (*connect.Response[pbc.GetActualCostResponse], error) {
	resp, err := h.server.GetActualCost(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetProjectedCost implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetProjectedCost(
	ctx context.Context,
	req *connect.Request[pbc.GetProjectedCostRequest],
) (*connect.Response[pbc.GetProjectedCostResponse], error) {
	resp, err := h.server.GetProjectedCost(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetPricingSpec implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetPricingSpec(
	ctx context.Context,
	req *connect.Request[pbc.GetPricingSpecRequest],
) (*connect.Response[pbc.GetPricingSpecResponse], error) {
	resp, err := h.server.GetPricingSpec(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// EstimateCost implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) EstimateCost(
	ctx context.Context,
	req *connect.Request[pbc.EstimateCostRequest],
) (*connect.Response[pbc.EstimateCostResponse], error) {
	resp, err := h.server.EstimateCost(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetRecommendations implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetRecommendations(
	ctx context.Context,
	req *connect.Request[pbc.GetRecommendationsRequest],
) (*connect.Response[pbc.GetRecommendationsResponse], error) {
	resp, err := h.server.GetRecommendations(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// DismissRecommendation implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) DismissRecommendation(
	ctx context.Context,
	req *connect.Request[pbc.DismissRecommendationRequest],
) (*connect.Response[pbc.DismissRecommendationResponse], error) {
	resp, err := h.server.DismissRecommendation(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetBudgets implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetBudgets(
	ctx context.Context,
	req *connect.Request[pbc.GetBudgetsRequest],
) (*connect.Response[pbc.GetBudgetsResponse], error) {
	resp, err := h.server.GetBudgets(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// GetPluginInfo implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) GetPluginInfo(
	ctx context.Context,
	req *connect.Request[pbc.GetPluginInfoRequest],
) (*connect.Response[pbc.GetPluginInfoResponse], error) {
	resp, err := h.server.GetPluginInfo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// BatchCost implements pbcconnect.CostSourceServiceHandler.
func (h *ConnectHandler) BatchCost(
	ctx context.Context,
	req *connect.Request[pbc.BatchCostRequest],
) (*connect.Response[pbc.BatchCostResponse], error) {
	resp, err := h.server.BatchCost(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
