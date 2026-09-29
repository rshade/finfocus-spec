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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func mockRec(id, resourceID string, action pbc.RecommendationActionType, tags map[string]string) *pbc.Recommendation {
	return &pbc.Recommendation{
		Id:         id,
		ActionType: action,
		Resource:   &pbc.ResourceRecommendationInfo{Id: resourceID, Tags: tags},
		Impact:     &pbc.RecommendationImpact{EstimatedSavings: 120},
		Priority:   pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_HIGH,
		Reasoning:  []string{"idle"},
	}
}

func TestMockRecommendationScorer_Scores(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer()
	req := &pbc.ScoreRecommendationsRequest{Recommendations: []*pbc.Recommendation{
		mockRec("safe", "i-1", pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE, nil),
		mockRec("risky", "i-2", pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_PURCHASE_COMMITMENT,
			map[string]string{"environment": "production"}),
		mockRec("standby", "i-3", pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
			map[string]string{"purpose": "standby"}),
	}}

	resp, err := scorer.ScoreRecommendations(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp))

	safe, risky, standby := resp.GetResults()[0].GetScores(), resp.GetResults()[1].GetScores(),
		resp.GetResults()[2].GetScores()
	assert.Greater(t, risky.GetRisk(), safe.GetRisk())
	assert.Greater(t, standby.GetFalsePositive(), safe.GetFalsePositive())
	assert.InDelta(t, 3, safe.GetPriority(), 0)
	assert.Equal(t, pbc.ScoreCalibration_SCORE_CALIBRATION_RANKING_ONLY, resp.GetScorer().GetCalibration())
	assert.Equal(t, int32(plugintesting.DefaultMockScorerBatchSize), resp.GetMaxBatchSize())
}

func TestMockRecommendationScorer_DuplicateGroups(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer()
	rightsize := pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE
	newReq := func(mode pbc.IdentifierMode) *pbc.ScoreRecommendationsRequest {
		return &pbc.ScoreRecommendationsRequest{
			IdentifierMode: mode,
			Recommendations: []*pbc.Recommendation{
				mockRec("a", "i-1", rightsize, nil),
				mockRec("b", "i-1", rightsize, nil),
				mockRec("c", "i-2", rightsize, nil),
			},
		}
	}

	pseudonymized := newReq(pbc.IdentifierMode_IDENTIFIER_MODE_PSEUDONYMIZED)
	resp, err := scorer.ScoreRecommendations(context.Background(), pseudonymized)
	require.NoError(t, err)
	a, b, c := resp.GetResults()[0].GetScores(), resp.GetResults()[1].GetScores(), resp.GetResults()[2].GetScores()
	assert.NotEmpty(t, a.GetDuplicateGroupId())
	assert.Equal(t, a.GetDuplicateGroupId(), b.GetDuplicateGroupId())
	assert.Empty(t, c.GetDuplicateGroupId())

	resp, err = scorer.ScoreRecommendations(context.Background(), newReq(pbc.IdentifierMode_IDENTIFIER_MODE_OMITTED))
	require.NoError(t, err)
	for _, result := range resp.GetResults() {
		assert.Empty(t, result.GetScores().GetDuplicateGroupId())
	}
}

func TestMockRecommendationScorer_PerItemError(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer()
	req := &pbc.ScoreRecommendationsRequest{Recommendations: []*pbc.Recommendation{
		{Id: "no-resource"},
		mockRec("ok", "i-1", pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE, nil),
	}}

	resp, err := scorer.ScoreRecommendations(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, plugintesting.ValidateScoreRecommendationsResponse(req, resp))
	assert.Equal(t, int32(codes.InvalidArgument), resp.GetResults()[0].GetError().GetCode())
	assert.NotNil(t, resp.GetResults()[1].GetScores())
}

func TestMockRecommendationScorer_SignalFilter(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer(
		plugintesting.WithScorerSignals(pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY),
	)
	rec := mockRec("a", "i-1", pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE, nil)

	resp, err := scorer.ScoreRecommendations(context.Background(), &pbc.ScoreRecommendationsRequest{
		Recommendations: []*pbc.Recommendation{rec},
	})
	require.NoError(t, err)
	scores := resp.GetResults()[0].GetScores()
	assert.NotNil(t, scores.Risk)
	assert.NotNil(t, scores.Priority)
	assert.Nil(t, scores.FalsePositive)
	assert.Equal(t, []pbc.ScoreSignal{
		pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY,
	}, resp.GetSupportedSignals())

	resp, err = scorer.ScoreRecommendations(context.Background(), &pbc.ScoreRecommendationsRequest{
		Recommendations: []*pbc.Recommendation{rec},
		Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY},
	})
	require.NoError(t, err)
	scores = resp.GetResults()[0].GetScores()
	assert.Nil(t, scores.Risk)
	assert.NotNil(t, scores.Priority)

	_, err = scorer.ScoreRecommendations(context.Background(), &pbc.ScoreRecommendationsRequest{
		Recommendations: []*pbc.Recommendation{rec},
		Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_WORTH_ACTING},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.ErrorIs(t, err, plugintesting.ErrInvalidScoreRequest)
}

func TestMockRecommendationScorer_RejectsInvalidRequests(t *testing.T) {
	scorer := plugintesting.NewMockRecommendationScorer(plugintesting.WithScorerMaxBatchSize(1))

	for name, req := range map[string]*pbc.ScoreRecommendationsRequest{
		"empty":     {},
		"oversize":  scoreRequest("a", "b"),
		"duplicate": scoreRequest("a", "a"),
	} {
		_, err := scorer.ScoreRecommendations(context.Background(), req)
		require.Error(t, err, name)
		assert.Equal(t, codes.InvalidArgument, status.Code(err), name)
	}
}
