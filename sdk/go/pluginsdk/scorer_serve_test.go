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
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// scorerTestPlugin is a cost-source plugin that also serves
// RecommendationScorerService by delegating to inner.
type scorerTestPlugin struct {
	*pluginsdk.BasePlugin

	inner pluginsdk.RecommendationScorerProvider
}

func newScorerTestPlugin(inner pluginsdk.RecommendationScorerProvider) *scorerTestPlugin {
	return &scorerTestPlugin{BasePlugin: pluginsdk.NewBasePlugin("scorer-test"), inner: inner}
}

func (p *scorerTestPlugin) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	return p.inner.ScoreRecommendations(ctx, req)
}

type scoreCall func(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error)

// startScorerServer serves plugin through pluginsdk.Serve and returns its
// address and a ScoreRecommendations client for the selected transport.
func startScorerServer(t *testing.T, plugin pluginsdk.Plugin, web bool) (string, scoreCall) {
	t.Helper()
	addr, _ := startAllocServer(t, plugin, web)

	if web {
		client := pbcconnect.NewRecommendationScorerServiceClient(http.DefaultClient, "http://"+addr)
		return addr, func(
			ctx context.Context, req *pbc.ScoreRecommendationsRequest,
		) (*pbc.ScoreRecommendationsResponse, error) {
			resp, err := client.ScoreRecommendations(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return resp.Msg, nil
		}
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewRecommendationScorerServiceClient(conn)
	return addr, func(
		ctx context.Context, req *pbc.ScoreRecommendationsRequest,
	) (*pbc.ScoreRecommendationsResponse, error) {
		return client.ScoreRecommendations(ctx, req)
	}
}

func callScore(
	t *testing.T, call scoreCall, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
	defer cancel()
	return call(ctx, req)
}

func scorerFixtureRequest() *pbc.ScoreRecommendationsRequest {
	rightsize := pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE
	rec := func(id, resource string) *pbc.Recommendation {
		return &pbc.Recommendation{
			Id:         id,
			ActionType: rightsize,
			Priority:   pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
			Reasoning:  []string{"idle"},
			Resource:   &pbc.ResourceRecommendationInfo{Id: resource},
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50},
		}
	}
	return &pbc.ScoreRecommendationsRequest{
		Recommendations: []*pbc.Recommendation{rec("r1", "i-1"), rec("r2", "i-1"), rec("r3", "i-2")},
	}
}

func TestScorer_TransportParity(t *testing.T) {
	req := scorerFixtureRequest()
	responses := make(map[string]*pbc.ScoreRecommendationsResponse, len(allocTransports))
	for _, tr := range allocTransports {
		plugin := newScorerTestPlugin(plugintesting.NewMockRecommendationScorer())
		_, call := startScorerServer(t, plugin, tr.web)
		resp, err := callScore(t, call, req)
		require.NoError(t, err, tr.name)
		require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp), tr.name)
		responses[tr.name] = resp
	}

	assert.True(t, proto.Equal(responses["grpc"], responses["connect"]), "gRPC and Connect responses differ")
	results := responses["grpc"].GetResults()
	require.Len(t, results, 3)
	assert.NotEmpty(t, results[0].GetScores().GetDuplicateGroupId())
	assert.Equal(t, results[0].GetScores().GetDuplicateGroupId(), results[1].GetScores().GetDuplicateGroupId())
}

func TestScorer_ErrorParity(t *testing.T) {
	failing := scorerFunc(func(
		context.Context, *pbc.ScoreRecommendationsRequest,
	) (*pbc.ScoreRecommendationsResponse, error) {
		return nil, status.Error(codes.Unavailable, "backend down")
	})
	tests := []struct {
		name     string
		inner    pluginsdk.RecommendationScorerProvider
		req      *pbc.ScoreRecommendationsRequest
		wantCode codes.Code
	}{
		{
			"empty request", plugintesting.NewMockRecommendationScorer(),
			&pbc.ScoreRecommendationsRequest{}, codes.InvalidArgument,
		},
		{"backend unavailable", failing, scorerFixtureRequest(), codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var messages [2]string
			for i, tr := range allocTransports {
				_, call := startScorerServer(t, newScorerTestPlugin(tt.inner), tr.web)
				_, err := callScore(t, call, tt.req)
				code, message := allocWireCode(t, err, tr.web)
				assert.Equal(t, tt.wantCode, code, tr.name)
				messages[i] = message
			}
			assert.Equal(t, messages[0], messages[1], "messages differ between transports")
			assert.NotContains(t, messages[0], "rpc error:")
		})
	}
}

// scorerFunc adapts a function to pluginsdk.RecommendationScorerProvider.
type scorerFunc func(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error)

func (f scorerFunc) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	return f(ctx, req)
}

func TestScorer_NotRegisteredWithoutProvider(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			addr, call := startScorerServer(t, pluginsdk.NewBasePlugin("cost-only"), tr.web)

			_, err := callScore(t, call, scorerFixtureRequest())
			code, _ := allocWireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)

			if tr.web {
				_, healthErr := checkServiceHealth(t, addr, pbcconnect.RecommendationScorerServiceName)
				assert.Equal(t, connect.CodeNotFound, connect.CodeOf(healthErr))
			}
		})
	}
}

func TestScorer_HealthIncludesScorer(t *testing.T) {
	addr, _ := startScorerServer(t, newScorerTestPlugin(plugintesting.NewMockRecommendationScorer()), true)

	resp, err := checkServiceHealth(t, addr, pbcconnect.RecommendationScorerServiceName)
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
}

func TestScorer_ConformanceOverServe(t *testing.T) {
	plugintesting.RunScorerConformance(t, newScorerTestPlugin(plugintesting.NewMockRecommendationScorer()))
}

func TestScorer_InferredCapability(t *testing.T) {
	server := pluginsdk.NewServerWithOptions(
		newScorerTestPlugin(plugintesting.NewMockRecommendationScorer()), nil, nil,
		&pluginsdk.PluginInfo{Name: "scorer", Version: "1.0.0", SpecVersion: "1.0.0"},
	)
	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING)
	assert.Equal(t, "true", resp.GetMetadata()["supports_recommendation_scoring"])

	plain := pluginsdk.NewServerWithOptions(pluginsdk.NewBasePlugin("plain"), nil, nil,
		&pluginsdk.PluginInfo{Name: "plain", Version: "1.0.0", SpecVersion: "1.0.0"})
	resp, err = plain.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.NotContains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING)
}

func TestScorer_LegacyName(t *testing.T) {
	assert.Equal(t, "supports_recommendation_scoring",
		pluginsdk.CapabilityToLegacyName(pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING))
}

func TestScorer_WrappersDelegate(t *testing.T) {
	bad := &pbc.ScoreRecommendationsRequest{}
	assert.Equal(t, plugintesting.ValidateScoreRecommendationsRequest(bad, 5).Error(),
		pluginsdk.ValidateScoreRecommendationsRequest(bad, 5).Error())

	good := scorerFixtureRequest()
	resp, err := plugintesting.NewMockRecommendationScorer().ScoreRecommendations(context.Background(), good)
	require.NoError(t, err)
	resp.Results = resp.GetResults()[:1]
	assert.Equal(t, plugintesting.ValidateScoreRecommendationsResponse(good, resp).Error(),
		pluginsdk.ValidateScoreRecommendationsResponse(good, resp).Error())
}
