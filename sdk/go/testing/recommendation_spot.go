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

package testing

import (
	"errors"
	"fmt"
	"math"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// recommendationSpotEpsilon is the tolerance around zero for Recommendation.spot_interruption_risk_score.
// It equals the tolerance applied to the same field on the cost responses.
const recommendationSpotEpsilon = 1e-9

// spotRecommendationSavings is the monthly saving carried by NewSpotRecommendation.
const spotRecommendationSavings = 62

var (
	// ErrRecommendationSpotInvalidValue reports a risk score that is NaN, infinite, negative, or above 1.0.
	ErrRecommendationSpotInvalidValue = errors.New(
		"recommendation.spot_interruption_risk_score must be finite and between 0.0 and 1.0")
	// ErrRecommendationSpotInvalidCategory reports a non-zero risk score without DYNAMIC pricing.
	ErrRecommendationSpotInvalidCategory = errors.New(
		"recommendation.spot_interruption_risk_score requires pricing_category DYNAMIC")
)

// ValidateRecommendationSpot validates the pricing category and spot interruption risk of a
// Recommendation. A score within 1e-9 of zero is valid for every category. Any other score must be
// finite, at most 1.0, and carried with DYNAMIC pricing. The valid path does not allocate.
func ValidateRecommendationSpot(category pbc.FocusPricingCategory, score float64) error {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return fmt.Errorf("%w: got %v", ErrRecommendationSpotInvalidValue, score)
	}
	if math.Abs(score) < recommendationSpotEpsilon {
		return nil
	}
	if score < -recommendationSpotEpsilon || score > 1.0 {
		return fmt.Errorf("%w: got %v", ErrRecommendationSpotInvalidValue, score)
	}
	if category != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC {
		return fmt.Errorf("%w: got score=%v with category=%s",
			ErrRecommendationSpotInvalidCategory, score, category.String())
	}
	return nil
}

// NewSpotRecommendation returns a valid recommendation that moves a resource to spot capacity.
// The category is DYNAMIC and the action type is MODIFY. Savings are carried in Impact, never in
// the spot fields. A risk of 0 means the interruption risk is unknown.
func NewSpotRecommendation(id string, risk float64) *pbc.Recommendation {
	return &pbc.Recommendation{
		Id:         id,
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Resource: &pbc.ResourceRecommendationInfo{
			Id:       "resource-1",
			Provider: mockParseProvider,
		},
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings: spotRecommendationSavings,
			Currency:         defaultAllocationCurrency,
		},
		Priority:                  pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
		PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
		SpotInterruptionRiskScore: risk,
	}
}
