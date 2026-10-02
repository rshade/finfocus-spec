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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestValidateScoreRecommendationsRequestSessionID(t *testing.T) {
	tests := []struct {
		name    string
		session string
		wantErr bool
	}{
		{"empty", "", false},
		{"printable", "op-2026-10-01.a", false},
		{"max length", strings.Repeat("a", 128), false},
		{"too long", strings.Repeat("a", 129), true},
		{"space", "has space", true},
		{"control", "bad\x01id", true},
		{"non ascii", "café", true},
		{"newline", "a\nb", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := scoreRequest("a")
			req.SessionId = tt.session
			err := plugintesting.ValidateScoreRecommendationsRequest(req, 0)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, plugintesting.ErrInvalidScoreRequest)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.NotContains(t, err.Error(), "rpc error:")
		})
	}
}

func TestValidateScoreRecommendationsResponseSession(t *testing.T) {
	lone := func() *pbc.ScoreRecommendationsResponse {
		return scoreResponse(scoredResult("a", &pbc.RecommendationScores{DuplicateGroupId: "g"}))
	}

	t.Run("lone member allowed with a session", func(t *testing.T) {
		req := scoreRequest("a")
		req.SessionId = "s1"
		resp := lone()
		resp.SessionId = "s1"
		require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp))
	})
	t.Run("lone member rejected without a session", func(t *testing.T) {
		err := plugintesting.ValidateScoreRecommendationsResponse(scoreRequest("a"), lone())
		require.ErrorIs(t, err, plugintesting.ErrInvalidScoreResponse)
		assert.ErrorContains(t, err, "used by only one recommendation")
	})
	t.Run("declined session keeps the two member rule", func(t *testing.T) {
		req := scoreRequest("a")
		req.SessionId = "s1"
		err := plugintesting.ValidateScoreRecommendationsResponse(req, lone())
		require.ErrorIs(t, err, plugintesting.ErrInvalidScoreResponse)
		assert.ErrorContains(t, err, "used by only one recommendation")
	})
	t.Run("declined session with no groups is valid", func(t *testing.T) {
		req := scoreRequest("a")
		req.SessionId = "s1"
		resp := scoreResponse(scoredResult("a", &pbc.RecommendationScores{}))
		require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp))
	})
	t.Run("mismatched echo", func(t *testing.T) {
		req := scoreRequest("a")
		req.SessionId = "s1"
		resp := lone()
		resp.SessionId = "s2"
		err := plugintesting.ValidateScoreRecommendationsResponse(req, resp)
		require.ErrorIs(t, err, plugintesting.ErrInvalidScoreResponse)
		assert.ErrorContains(t, err, "session_id")
	})
	t.Run("echo without a request session", func(t *testing.T) {
		resp := scoreResponse(scoredResult("a", &pbc.RecommendationScores{}))
		resp.SessionId = "s1"
		err := plugintesting.ValidateScoreRecommendationsResponse(scoreRequest("a"), resp)
		require.ErrorIs(t, err, plugintesting.ErrInvalidScoreResponse)
		assert.ErrorContains(t, err, "session_id")
	})
}

func TestScoreValidatorsSessionlessAllocationBound(t *testing.T) {
	req := scoreRequest("a")
	resp := scoreResponse(scoredResult("a", &pbc.RecommendationScores{}))
	requestAllocs := testing.AllocsPerRun(50, func() {
		_ = plugintesting.ValidateScoreRecommendationsRequest(req, 0)
	})
	assert.LessOrEqual(t, requestAllocs, 1.0, "request validation allocates only its id set")
	responseAllocs := testing.AllocsPerRun(50, func() {
		_ = plugintesting.ValidateScoreRecommendationsResponse(req, resp)
	})
	assert.LessOrEqual(t, responseAllocs, 3.0)
}

func TestMockRecommendationScorer_Sessions(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer()
	rightsize := pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE
	score := func(session string, rec *pbc.Recommendation, mode pbc.IdentifierMode) *pbc.ScoreRecommendationsResponse {
		req := &pbc.ScoreRecommendationsRequest{
			SessionId:       session,
			IdentifierMode:  mode,
			Recommendations: []*pbc.Recommendation{rec},
		}
		resp, err := scorer.ScoreRecommendations(context.Background(), req)
		require.NoError(t, err)
		require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp))
		return resp
	}
	pseudonymized := pbc.IdentifierMode_IDENTIFIER_MODE_PSEUDONYMIZED

	t.Run("duplicate pair split across batches shares one id", func(t *testing.T) {
		first := score("s1", mockRec("a", "i-1", rightsize, nil), pseudonymized)
		second := score("s1", mockRec("b", "i-1", rightsize, nil), pseudonymized)
		idA := first.GetResults()[0].GetScores().GetDuplicateGroupId()
		idB := second.GetResults()[0].GetScores().GetDuplicateGroupId()
		assert.NotEmpty(t, idA)
		assert.Equal(t, idA, idB)
		assert.Equal(t, "s1", first.GetSessionId())
	})
	t.Run("different resource gets a different id", func(t *testing.T) {
		first := score("s1", mockRec("a", "i-1", rightsize, nil), pseudonymized)
		other := score("s1", mockRec("c", "i-2", rightsize, nil), pseudonymized)
		assert.NotEqual(t, first.GetResults()[0].GetScores().GetDuplicateGroupId(),
			other.GetResults()[0].GetScores().GetDuplicateGroupId())
	})
	t.Run("sessions are isolated", func(t *testing.T) {
		one := score("s1", mockRec("a", "i-1", rightsize, nil), pseudonymized)
		two := score("s2", mockRec("a", "i-1", rightsize, nil), pseudonymized)
		assert.NotEqual(t, one.GetResults()[0].GetScores().GetDuplicateGroupId(),
			two.GetResults()[0].GetScores().GetDuplicateGroupId())
	})
	t.Run("omitted identifiers produce no id", func(t *testing.T) {
		resp := score("s1", mockRec("a", "i-1", rightsize, nil), pbc.IdentifierMode_IDENTIFIER_MODE_OMITTED)
		assert.Empty(t, resp.GetResults()[0].GetScores().GetDuplicateGroupId())
		assert.Equal(t, "s1", resp.GetSessionId())
	})
	t.Run("sessionless singleton stays cleared and no echo", func(t *testing.T) {
		resp := score("", mockRec("a", "i-1", rightsize, nil), pseudonymized)
		assert.Empty(t, resp.GetResults()[0].GetScores().GetDuplicateGroupId())
		assert.Empty(t, resp.GetSessionId())
	})
	t.Run("invalid session is rejected", func(t *testing.T) {
		req := scoreRequest("a")
		req.SessionId = "bad id"
		_, err := scorer.ScoreRecommendations(context.Background(), req)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
