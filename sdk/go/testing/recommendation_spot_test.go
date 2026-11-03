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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestNewSpotRecommendation(t *testing.T) {
	rec := plugintesting.NewSpotRecommendation("rec-1", 0.12)

	assert.Equal(t, "rec-1", rec.GetId())
	assert.Equal(t, pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC, rec.GetPricingCategory())
	assert.InDelta(t, 0.12, rec.GetSpotInterruptionRiskScore(), 0)
	assert.Equal(t, pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY, rec.GetActionType())
	require.NoError(t, plugintesting.ValidateRecommendationSpot(
		rec.GetPricingCategory(), rec.GetSpotInterruptionRiskScore()))
}

func TestValidateRecommendationSpotValidAllocationFree(t *testing.T) {
	dynamic := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC

	require.NoError(t, plugintesting.ValidateRecommendationSpot(dynamic, 0))
	require.NoError(t, plugintesting.ValidateRecommendationSpot(dynamic, 0.12))

	allocs := testing.AllocsPerRun(100, func() {
		_ = plugintesting.ValidateRecommendationSpot(dynamic, 0.12)
	})
	assert.Zero(t, allocs, "valid path must not allocate")
}

func TestRecommendationSpotRoundTripOverHarness(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	spot := plugintesting.NewSpotRecommendation("rec-1", 0.12)
	wantSavings := spot.GetImpact().GetEstimatedSavings()
	plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
		Recommendations: []*pbc.Recommendation{spot},
	})

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetRecommendations(context.Background(), &pbc.GetRecommendationsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetRecommendations(), 1)

	got := resp.GetRecommendations()[0]
	assert.Equal(t, pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC, got.GetPricingCategory())
	assert.InDelta(t, 0.12, got.GetSpotInterruptionRiskScore(), 0)
	assert.InDelta(t, wantSavings, got.GetImpact().GetEstimatedSavings(), 0, "savings stay in impact")
}

func TestValidateRecommendationSpot(t *testing.T) {
	const (
		unspecified = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED
		standard    = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
		committed   = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED
		dynamic     = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC
	)
	var (
		valueErr    = plugintesting.ErrRecommendationSpotInvalidValue
		categoryErr = plugintesting.ErrRecommendationSpotInvalidCategory
	)

	tests := []struct {
		name     string
		category pbc.FocusPricingCategory
		score    float64
		wantErr  error
	}{
		{"unspecified zero", unspecified, 0, nil},
		{"standard zero", standard, 0, nil},
		{"committed zero", committed, 0, nil},
		{"dynamic zero", dynamic, 0, nil},
		{"dynamic mid", dynamic, 0.5, nil},
		{"dynamic exactly one", dynamic, 1.0, nil},
		{"unspecified within epsilon", unspecified, 1e-10, nil},
		{"standard within negative epsilon", standard, -1e-10, nil},
		{"unspecified non-zero", unspecified, 0.5, categoryErr},
		{"standard non-zero", standard, 0.5, categoryErr},
		{"committed non-zero", committed, 0.5, categoryErr},
		{"standard just above epsilon", standard, 1.1e-9, categoryErr},
		{"dynamic below negative epsilon", dynamic, -1.1e-9, valueErr},
		{"dynamic negative", dynamic, -0.1, valueErr},
		{"dynamic above one", dynamic, 1.0000001, valueErr},
		{"dynamic NaN", dynamic, math.NaN(), valueErr},
		{"dynamic positive infinity", dynamic, math.Inf(1), valueErr},
		{"dynamic negative infinity", dynamic, math.Inf(-1), valueErr},
		{"standard NaN reports value first", standard, math.NaN(), valueErr},
		{"standard above one reports value first", standard, 2, valueErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateRecommendationSpot(tt.category, tt.score)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), "recommendation.spot_interruption_risk_score")
		})
	}
}

func TestScoreRequestIgnoresSpotPair(t *testing.T) {
	req := scoreRequest("a")
	rec := req.GetRecommendations()[0]
	rec.PricingCategory = pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
	rec.SpotInterruptionRiskScore = 0.5

	require.Error(t, plugintesting.ValidateRecommendationSpot(
		rec.GetPricingCategory(), rec.GetSpotInterruptionRiskScore()))
	require.NoError(t, plugintesting.ValidateScoreRecommendationsRequest(req, 0),
		"the scorer request validator must not apply the spot rule")
}
