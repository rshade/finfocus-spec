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

package pluginsdk

import (
	"context"

	"connectrpc.com/connect"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// ValidateScoreRecommendationsRequest is identical to the sdk/go/testing
// function of the same name. Scorers call it first: every failure wraps
// plugintesting.ErrInvalidScoreRequest and carries codes.InvalidArgument, so it
// may be returned directly. Pass the scorer's own batch limit as maxBatchSize,
// or zero for no limit.
func ValidateScoreRecommendationsRequest(req *pbc.ScoreRecommendationsRequest, maxBatchSize int32) error {
	return plugintesting.ValidateScoreRecommendationsRequest(req, maxBatchSize)
}

// ValidateScoreRecommendationsResponse is identical to the sdk/go/testing
// function of the same name. Hosts call it on every response before using a
// score: it checks index alignment, ranges, signal support and duplicate
// groups. Every failure wraps plugintesting.ErrInvalidScoreResponse.
func ValidateScoreRecommendationsResponse(
	req *pbc.ScoreRecommendationsRequest, resp *pbc.ScoreRecommendationsResponse,
) error {
	return plugintesting.ValidateScoreRecommendationsResponse(req, resp)
}

// recommendationScorerGRPCServer adapts a RecommendationScorerProvider to the
// generated gRPC server interface, so plugins need not embed the Unimplemented
// server.
type recommendationScorerGRPCServer struct {
	pbc.UnimplementedRecommendationScorerServiceServer

	provider RecommendationScorerProvider
}

func (s *recommendationScorerGRPCServer) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	return s.provider.ScoreRecommendations(ctx, req)
}

// recommendationScorerConnectHandler adapts a RecommendationScorerProvider to
// the generated Connect handler interface, preserving gRPC status codes via
// toConnectError.
type recommendationScorerConnectHandler struct {
	provider RecommendationScorerProvider
}

func (h *recommendationScorerConnectHandler) ScoreRecommendations(
	ctx context.Context, req *connect.Request[pbc.ScoreRecommendationsRequest],
) (*connect.Response[pbc.ScoreRecommendationsResponse], error) {
	resp, err := h.provider.ScoreRecommendations(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}
