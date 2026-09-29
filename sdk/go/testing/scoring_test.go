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

package testing_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func scoreRequest(ids ...string) *pbc.ScoreRecommendationsRequest {
	req := &pbc.ScoreRecommendationsRequest{}
	for _, id := range ids {
		req.Recommendations = append(req.Recommendations, &pbc.Recommendation{Id: id})
	}
	return req
}

func scoredResult(id string, scores *pbc.RecommendationScores) *pbc.RecommendationScoreResult {
	return &pbc.RecommendationScoreResult{
		RecommendationId: id,
		Result:           &pbc.RecommendationScoreResult_Scores{Scores: scores},
	}
}

func failedResult(id string, code int32) *pbc.RecommendationScoreResult {
	return &pbc.RecommendationScoreResult{
		RecommendationId: id,
		Result:           &pbc.RecommendationScoreResult_Error{Error: &pbc.ResourceError{Code: code, Message: "no"}},
	}
}

func allSignals() []pbc.ScoreSignal {
	return []pbc.ScoreSignal{
		pbc.ScoreSignal_SCORE_SIGNAL_RISK,
		pbc.ScoreSignal_SCORE_SIGNAL_FALSE_POSITIVE,
		pbc.ScoreSignal_SCORE_SIGNAL_WORTH_ACTING,
		pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY,
		pbc.ScoreSignal_SCORE_SIGNAL_INSUFFICIENT_EVIDENCE,
		pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP,
	}
}

func scoreResponse(results ...*pbc.RecommendationScoreResult) *pbc.ScoreRecommendationsResponse {
	return &pbc.ScoreRecommendationsResponse{
		Results:          results,
		MaxBatchSize:     25,
		SupportedSignals: allSignals(),
	}
}

func TestValidateScoreRecommendationsRequest(t *testing.T) {
	tests := []struct {
		name         string
		req          *pbc.ScoreRecommendationsRequest
		maxBatchSize int32
		wantErr      string
	}{
		{name: "valid", req: scoreRequest("a", "b"), maxBatchSize: 2},
		{name: "no limit when max is zero", req: scoreRequest("a", "b", "c")},
		{name: "nil request", req: nil, wantErr: "nil"},
		{name: "empty request", req: scoreRequest(), wantErr: "no recommendations"},
		{name: "over max batch size", req: scoreRequest("a", "b", "c"), maxBatchSize: 2, wantErr: "max_batch_size"},
		{name: "duplicate ids", req: scoreRequest("a", "b", "a"), wantErr: `duplicates recommendations[0]`},
		{name: "empty id", req: scoreRequest("a", ""), wantErr: "empty id"},
		{
			name: "nil entry",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: []*pbc.Recommendation{{Id: "a"}, nil},
			},
			wantErr: "recommendations[1] is nil",
		},
		{
			name: "unspecified signal",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED},
			},
			wantErr: "signals[0]",
		},
		{
			name: "unknown signal",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal(99)},
			},
			wantErr: "signals[0]",
		},
		{
			name: "unknown identifier mode",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				IdentifierMode:  pbc.IdentifierMode(99),
			},
			wantErr: "identifier_mode",
		},
		{
			name: "signal subset and mode",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK},
				IdentifierMode:  pbc.IdentifierMode_IDENTIFIER_MODE_OMITTED,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateScoreRecommendationsRequest(tt.req, tt.maxBatchSize)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.ErrorIs(t, err, plugintesting.ErrInvalidScoreRequest)
			require.ErrorContains(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestValidateScoreRecommendationsResponse(t *testing.T) {
	full := &pbc.RecommendationScores{
		Risk:                 proto.Float64(0.3),
		FalsePositive:        proto.Float64(0),
		WorthActing:          proto.Float64(1),
		Priority:             proto.Float64(3),
		InsufficientEvidence: proto.Float64(0.5),
		DuplicateGroupId:     "g1",
	}
	partner := &pbc.RecommendationScores{DuplicateGroupId: "g1"}

	tests := []struct {
		name    string
		req     *pbc.ScoreRecommendationsRequest
		resp    *pbc.ScoreRecommendationsResponse
		wantErr string
	}{
		{
			name: "valid full scores at the range edges",
			req:  scoreRequest("a", "b"),
			resp: scoreResponse(scoredResult("a", full), scoredResult("b", partner)),
		},
		{
			name: "valid per-item error beside scores",
			req:  scoreRequest("a", "b"),
			resp: scoreResponse(
				scoredResult("a", &pbc.RecommendationScores{Risk: proto.Float64(0.1)}),
				failedResult("b", int32(codes.InvalidArgument)),
			),
		},
		{name: "nil response", req: scoreRequest("a"), resp: nil, wantErr: "nil"},
		{
			name:    "too few results",
			req:     scoreRequest("a", "b"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{})),
			wantErr: "1 results for 2 recommendations",
		},
		{
			name: "too many results",
			req:  scoreRequest("a"),
			resp: scoreResponse(
				scoredResult("a", &pbc.RecommendationScores{}), scoredResult("b", &pbc.RecommendationScores{}),
			),
			wantErr: "2 results for 1 recommendations",
		},
		{
			name: "misaligned ids",
			req:  scoreRequest("a", "b"),
			resp: scoreResponse(
				scoredResult("b", &pbc.RecommendationScores{}), scoredResult("a", &pbc.RecommendationScores{}),
			),
			wantErr: `results[0].recommendation_id "b", want "a"`,
		},
		{
			name:    "result oneof unset",
			req:     scoreRequest("a"),
			resp:    scoreResponse(&pbc.RecommendationScoreResult{RecommendationId: "a"}),
			wantErr: "neither scores nor error",
		},
		{
			name:    "nil scores payload",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", nil)),
			wantErr: "neither scores nor error",
		},
		{
			name:    "error code OK",
			req:     scoreRequest("a"),
			resp:    scoreResponse(failedResult("a", int32(codes.OK))),
			wantErr: "code must not be OK",
		},
		{
			name:    "risk above one",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{Risk: proto.Float64(1.01)})),
			wantErr: "risk",
		},
		{
			name:    "false_positive negative",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{FalsePositive: proto.Float64(-0.01)})),
			wantErr: "false_positive",
		},
		{
			name:    "worth_acting above one",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{WorthActing: proto.Float64(2)})),
			wantErr: "worth_acting",
		},
		{
			name:    "priority above three",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{Priority: proto.Float64(3.5)})),
			wantErr: "priority",
		},
		{
			name: "insufficient_evidence NaN",
			req:  scoreRequest("a"),
			resp: scoreResponse(scoredResult("a",
				&pbc.RecommendationScores{InsufficientEvidence: proto.Float64(math.NaN())})),
			wantErr: "insufficient_evidence",
		},
		{
			name:    "risk infinite",
			req:     scoreRequest("a"),
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{Risk: proto.Float64(math.Inf(1))})),
			wantErr: "risk",
		},
		{
			name: "max_batch_size zero",
			req:  scoreRequest("a"),
			resp: &pbc.ScoreRecommendationsResponse{
				Results:          []*pbc.RecommendationScoreResult{scoredResult("a", &pbc.RecommendationScores{})},
				SupportedSignals: allSignals(),
			},
			wantErr: "max_batch_size",
		},
		{
			name: "batch larger than max_batch_size",
			req:  scoreRequest("a", "b"),
			resp: &pbc.ScoreRecommendationsResponse{
				Results: []*pbc.RecommendationScoreResult{
					scoredResult("a", &pbc.RecommendationScores{}), scoredResult("b", &pbc.RecommendationScores{}),
				},
				MaxBatchSize:     1,
				SupportedSignals: allSignals(),
			},
			wantErr: "max_batch_size",
		},
		{
			name: "unspecified supported signal",
			req:  scoreRequest("a"),
			resp: &pbc.ScoreRecommendationsResponse{
				Results:          []*pbc.RecommendationScoreResult{scoredResult("a", &pbc.RecommendationScores{})},
				MaxBatchSize:     5,
				SupportedSignals: []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED},
			},
			wantErr: "supported_signals[0]",
		},
		{
			name: "repeated supported signal",
			req:  scoreRequest("a"),
			resp: &pbc.ScoreRecommendationsResponse{
				Results:      []*pbc.RecommendationScoreResult{scoredResult("a", &pbc.RecommendationScores{})},
				MaxBatchSize: 5,
				SupportedSignals: []pbc.ScoreSignal{
					pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_RISK,
				},
			},
			wantErr: "supported_signals[1]",
		},
		{
			name: "unknown calibration",
			req:  scoreRequest("a"),
			resp: func() *pbc.ScoreRecommendationsResponse {
				r := scoreResponse(scoredResult("a", &pbc.RecommendationScores{}))
				r.Scorer = &pbc.ScorerInfo{Name: "x", Calibration: pbc.ScoreCalibration(9)}
				return r
			}(),
			wantErr: "calibration",
		},
		{
			name: "signal not supported",
			req:  scoreRequest("a"),
			resp: &pbc.ScoreRecommendationsResponse{
				Results: []*pbc.RecommendationScoreResult{
					scoredResult("a", &pbc.RecommendationScores{Risk: proto.Float64(0.1)}),
				},
				MaxBatchSize:     5,
				SupportedSignals: []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY},
			},
			wantErr: "risk is set but SCORE_SIGNAL_RISK is not supported",
		},
		{
			name: "signal not requested",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK},
			},
			resp:    scoreResponse(scoredResult("a", &pbc.RecommendationScores{Priority: proto.Float64(1)})),
			wantErr: "priority is set but SCORE_SIGNAL_PRIORITY was not requested",
		},
		{
			name: "requested signal not supported",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK},
			},
			resp: &pbc.ScoreRecommendationsResponse{
				Results:          []*pbc.RecommendationScoreResult{scoredResult("a", &pbc.RecommendationScores{})},
				MaxBatchSize:     5,
				SupportedSignals: []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY},
			},
			wantErr: "SCORE_SIGNAL_RISK was requested but is not supported",
		},
		{
			name: "duplicate group set but grouping not requested",
			req: &pbc.ScoreRecommendationsRequest{
				Recommendations: scoreRequest("a", "b").GetRecommendations(),
				Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK},
			},
			resp: scoreResponse(
				scoredResult("a", &pbc.RecommendationScores{DuplicateGroupId: "g"}),
				scoredResult("b", &pbc.RecommendationScores{DuplicateGroupId: "g"}),
			),
			wantErr: "duplicate_group_id is set but SCORE_SIGNAL_DUPLICATE_GROUP was not requested",
		},
		{
			name: "group of one",
			req:  scoreRequest("a", "b"),
			resp: scoreResponse(
				scoredResult("a", &pbc.RecommendationScores{DuplicateGroupId: "g"}),
				scoredResult("b", &pbc.RecommendationScores{DuplicateGroupId: "h"}),
			),
			wantErr: `duplicate_group_id "g" is used by only one recommendation`,
		},
		{
			name: "group of two is valid",
			req:  scoreRequest("a", "b", "c"),
			resp: scoreResponse(
				scoredResult("a", &pbc.RecommendationScores{DuplicateGroupId: "g"}),
				scoredResult("b", &pbc.RecommendationScores{DuplicateGroupId: "g"}),
				scoredResult("c", &pbc.RecommendationScores{}),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateScoreRecommendationsResponse(tt.req, tt.resp)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.ErrorIs(t, err, plugintesting.ErrInvalidScoreResponse)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
