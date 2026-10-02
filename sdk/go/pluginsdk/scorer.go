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
// or zero for no limit. A non-empty session_id must be at most 128 printable
// ASCII characters.
func ValidateScoreRecommendationsRequest(req *pbc.ScoreRecommendationsRequest, maxBatchSize int32) error {
	return plugintesting.ValidateScoreRecommendationsRequest(req, maxBatchSize)
}

// ValidateScoreRecommendationsResponse is identical to the sdk/go/testing
// function of the same name. Hosts call it on every response before using a
// score: it checks index alignment, ranges, signal support, the echoed
// session_id and duplicate groups (a lone group member is valid in a session). Every failure wraps plugintesting.ErrInvalidScoreResponse.
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

// WithScorerLimits advertises a scorer's max_batch_size and supported signals
// through GetPluginInfo metadata, so a host can plan batches before its first
// scoring call. The response fields of ScoreRecommendations stay authoritative
// for each call. Invalid input (a limit below 1, no signal, an unspecified or
// repeated signal) is reported by PluginInfo.Validate.
func WithScorerLimits(maxBatchSize int32, signals ...pbc.ScoreSignal) PluginInfoOption {
	return func(info *PluginInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]string)
		}
		for k, v := range plugintesting.FormatScorerLimits(maxBatchSize, signals) {
			info.Metadata[k] = v
		}
	}
}

// ScorerLimits are the batch limit and signals a scorer advertised.
type ScorerLimits = plugintesting.ScorerLimits

// ParseScorerLimits reads the limits a scorer advertised in GetPluginInfo
// metadata. It returns nil with a nil error when the plugin advertised nothing;
// hosts then fall back to the response fields.
func ParseScorerLimits(metadata map[string]string) (*ScorerLimits, error) {
	return plugintesting.ParseScorerLimits(metadata)
}

// IsBatchTooLarge reports whether err is the error a scorer returns for a batch
// above its max_batch_size, as opposed to an empty request, duplicate ids or an
// unsupported signal. It works on a gRPC error and on the status-carrying error
// ValidateScoreRecommendationsRequest returns.
func IsBatchTooLarge(err error) bool {
	return plugintesting.IsBatchTooLarge(err)
}
