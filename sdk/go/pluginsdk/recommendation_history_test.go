package pluginsdk_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

type historyTestPlugin struct {
	connectTestPlugin

	response *pbc.GetRecommendationHistoryResponse
	err      error
}

func (p *historyTestPlugin) GetRecommendationHistory(
	_ context.Context, req *pbc.GetRecommendationHistoryRequest,
) (*pbc.GetRecommendationHistoryResponse, error) {
	if req.GetRecommendationId() != "rec-123" {
		return nil, status.Error(codes.NotFound, "recommendation not found")
	}
	return p.response, p.err
}

func TestGetRecommendationHistory_Transports(t *testing.T) {
	event := &pbc.LifecycleEvent{
		Action: "dismissed", Reason: "business_constraint", CustomReason: "Migration scheduled",
		Timestamp: timestamppb.New(time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)),
		ExpiresAt: timestamppb.New(time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)),
		Actor:     "alice@example.com/workstation-1",
	}
	tests := []struct {
		name        string
		plugin      pluginsdk.Plugin
		id          string
		wantCode    connect.Code
		wantMessage string
		wantEvents  []*pbc.LifecycleEvent
	}{
		{name: "rich history", plugin: &historyTestPlugin{response: &pbc.GetRecommendationHistoryResponse{
			Events: []*pbc.LifecycleEvent{event},
		}}, id: "rec-123", wantEvents: []*pbc.LifecycleEvent{event}},
		{
			name:   "empty history",
			plugin: &historyTestPlugin{response: &pbc.GetRecommendationHistoryResponse{}},
			id:     "rec-123",
		},
		{name: "unsupported", plugin: &connectTestPlugin{}, id: "rec-123", wantCode: connect.CodeUnimplemented},
		{
			name:        "stub fallback",
			plugin:      &historyTestPlugin{err: status.Error(codes.Unimplemented, "stub")},
			id:          "rec-123",
			wantCode:    connect.CodeUnimplemented,
			wantMessage: "plugin does not support GetRecommendationHistory",
		},
		{name: "missing ID", plugin: &historyTestPlugin{}, wantCode: connect.CodeInvalidArgument},
		{name: "unknown ID", plugin: &historyTestPlugin{}, id: "missing", wantCode: connect.CodeNotFound},
		{name: "backend unavailable", plugin: &historyTestPlugin{err: status.Error(codes.Unavailable, "try later")},
			id: "rec-123", wantCode: connect.CodeUnavailable, wantMessage: "try later"},
		{name: "wrapped status", plugin: &historyTestPlugin{err: fmt.Errorf("private context: %w",
			status.Error(codes.PermissionDenied, "access denied"))},
			id: "rec-123", wantCode: connect.CodePermissionDenied, wantMessage: "access denied"},
		{
			name:        "internal error sanitized",
			plugin:      &historyTestPlugin{err: errors.New("private credentials")},
			id:          "rec-123",
			wantCode:    connect.CodeInternal,
			wantMessage: "plugin failed to execute GetRecommendationHistory",
		},
		{name: "nil response", plugin: &historyTestPlugin{}, id: "rec-123", wantCode: connect.CodeInternal},
	}
	protocols := []struct {
		name    string
		options []connect.ClientOption
	}{
		{name: "Connect"},
		{name: "gRPC", options: []connect.ClientOption{connect.WithGRPC()}},
		{name: "gRPCWeb", options: []connect.ClientOption{connect.WithGRPCWeb()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, handler := pbcconnect.NewCostSourceServiceHandler(
				pluginsdk.NewConnectHandler(pluginsdk.NewServer(tt.plugin)),
			)
			require.Equal(t, "/finfocus.v1.CostSourceService/", path)
			server := httptest.NewUnstartedServer(handler)
			server.EnableHTTP2 = true
			server.StartTLS()
			defer server.Close()
			for _, protocol := range protocols {
				t.Run(protocol.name, func(t *testing.T) {
					client := pluginsdk.NewClient(pluginsdk.ClientConfig{
						BaseURL: server.URL, HTTPClient: server.Client(), ConnectOptions: protocol.options,
					})
					response, err := client.GetRecommendationHistory(context.Background(),
						&pbc.GetRecommendationHistoryRequest{RecommendationId: tt.id},
					)
					if tt.wantCode != 0 {
						require.Error(t, err)
						assert.Equal(t, tt.wantCode, connect.CodeOf(err))
						if tt.wantMessage != "" {
							var connectErr *connect.Error
							require.ErrorAs(t, err, &connectErr)
							assert.Equal(t, tt.wantMessage, connectErr.Message())
						}
						return
					}
					require.NoError(t, err)
					assert.Empty(t, cmp.Diff(tt.wantEvents, response.GetEvents(), protocmp.Transform()))
				})
			}
		})
	}
}

func TestGetRecommendationHistory_NilRequest(t *testing.T) {
	server := pluginsdk.NewServer(&historyTestPlugin{})
	_, err := server.GetRecommendationHistory(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestClient_GetRecommendationHistory_NilRequest(t *testing.T) {
	client := pluginsdk.NewConnectClient("http://localhost")
	defer client.Close()
	_, err := client.GetRecommendationHistory(context.Background(), nil)
	require.ErrorContains(t, err, "request cannot be nil")
}
