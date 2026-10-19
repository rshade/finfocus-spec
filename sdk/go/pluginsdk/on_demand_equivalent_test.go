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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestOnDemandEquivalentSDK(t *testing.T) {
	dynamic := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC
	standard := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
	committed := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED
	unspecified := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED
	pricePtr := func(v float64) *float64 { return &v }
	tests := []struct {
		name     string
		value    *float64
		category pbc.FocusPricingCategory
		want     error
	}{
		{"dynamic value", pricePtr(100), dynamic, nil},
		{"dynamic unset", nil, dynamic, nil},
		{"dynamic zero", pricePtr(0), dynamic, nil},
		{"dynamic spot cost exceeds equivalent", pricePtr(50), dynamic, nil},
		{"standard value", pricePtr(100), standard, pluginsdk.ErrOnDemandEquivalentInvalidCategory},
		{"standard zero", pricePtr(0), standard, pluginsdk.ErrOnDemandEquivalentInvalidCategory},
		{"committed value", pricePtr(100), committed, pluginsdk.ErrOnDemandEquivalentInvalidCategory},
		{"unspecified value", pricePtr(100), unspecified, pluginsdk.ErrOnDemandEquivalentInvalidCategory},
		{"legacy unspecified", nil, unspecified, nil},
		{"NaN", pricePtr(math.NaN()), dynamic, pluginsdk.ErrOnDemandEquivalentInvalidValue},
		{"positive infinity", pricePtr(math.Inf(1)), dynamic, pluginsdk.ErrOnDemandEquivalentInvalidValue},
		{"negative infinity", pricePtr(math.Inf(-1)), dynamic, pluginsdk.ErrOnDemandEquivalentInvalidValue},
		{"negative", pricePtr(-0.01), dynamic, pluginsdk.ErrOnDemandEquivalentInvalidValue},
		{"value before category", pricePtr(math.NaN()), standard, pluginsdk.ErrOnDemandEquivalentInvalidValue},
		{"dry run presence", pricePtr(0), unspecified, pluginsdk.ErrOnDemandEquivalentInvalidCategory},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projected := &pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 80,
				PricingCategory: tc.category, OnDemandEquivalentMonthlyCost: tc.value}
			estimate := &pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 80,
				PricingCategory: tc.category, OnDemandEquivalentMonthlyCost: tc.value}
			for _, validate := range []func() error{
				func() error { return pluginsdk.ValidateGetProjectedCostResponse(projected) },
				func() error { return pluginsdk.ValidateEstimateCostResponse(estimate) },
			} {
				err := validate()
				if tc.want == nil {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, tc.want)
				}
			}
		})
	}
}

func TestOnDemandEquivalentSentinelsShared(t *testing.T) {
	require.ErrorIs(t, pluginsdk.ErrOnDemandEquivalentInvalidValue, plugintesting.ErrOnDemandEquivalentInvalidValue)
	require.ErrorIs(t, pluginsdk.ErrOnDemandEquivalentInvalidCategory,
		plugintesting.ErrOnDemandEquivalentInvalidCategory)
}

func TestOnDemandEquivalentDryRunPresence(t *testing.T) {
	zero := 0.0
	projected := &pbc.GetProjectedCostResponse{
		DryRunResult: &pbc.DryRunResponse{}, OnDemandEquivalentMonthlyCost: &zero,
	}
	require.ErrorIs(t, pluginsdk.ValidateGetProjectedCostResponse(projected),
		pluginsdk.ErrOnDemandEquivalentInvalidCategory)
}

func TestCheckOnDemandEquivalentConsistency(t *testing.T) {
	dynamic := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC
	standard := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
	value := 0.0
	assert.Empty(t, pluginsdk.CheckOnDemandEquivalentConsistency(dynamic, &value))
	assert.Empty(t, pluginsdk.CheckOnDemandEquivalentConsistency(standard, nil))
	assert.NotEmpty(t, pluginsdk.CheckOnDemandEquivalentConsistency(dynamic, nil))
	assert.NotEmpty(t, pluginsdk.CheckOnDemandEquivalentConsistency(standard, &value))
}

func TestOnDemandEquivalentOptions(t *testing.T) {
	value := 0.0
	projectedOption := pluginsdk.WithProjectedCostOnDemandEquivalent(value)
	estimateOption := pluginsdk.WithOnDemandEquivalent(value)
	projected := pluginsdk.NewGetProjectedCostResponse(
		pluginsdk.WithProjectedCostPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
		projectedOption,
	)
	estimate := pluginsdk.NewEstimateCostResponse(
		pluginsdk.WithPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
		estimateOption,
	)
	require.NotNil(t, projected.OnDemandEquivalentMonthlyCost)
	require.NotNil(t, estimate.OnDemandEquivalentMonthlyCost)
	assert.Zero(t, projected.GetOnDemandEquivalentMonthlyCost())
	assert.Zero(t, estimate.GetOnDemandEquivalentMonthlyCost())
	secondProjected := pluginsdk.NewGetProjectedCostResponse(projectedOption)
	secondEstimate := pluginsdk.NewEstimateCostResponse(estimateOption)
	*projected.OnDemandEquivalentMonthlyCost = 12
	*estimate.OnDemandEquivalentMonthlyCost = 12
	assert.Zero(t, secondProjected.GetOnDemandEquivalentMonthlyCost())
	assert.Zero(t, secondEstimate.GetOnDemandEquivalentMonthlyCost())
	assert.NotPanics(t, func() {
		invalid := pluginsdk.NewEstimateCostResponse(pluginsdk.WithOnDemandEquivalent(math.NaN()))
		assert.ErrorIs(t, pluginsdk.ValidateEstimateCostResponse(invalid), pluginsdk.ErrOnDemandEquivalentInvalidValue)
	})
	assert.NotPanics(t, func() {
		invalid := pluginsdk.NewGetProjectedCostResponse(pluginsdk.WithProjectedCostOnDemandEquivalent(-1))
		assert.ErrorIs(t, pluginsdk.ValidateGetProjectedCostResponse(invalid),
			pluginsdk.ErrOnDemandEquivalentInvalidValue)
	})
}
