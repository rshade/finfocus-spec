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
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestMockOnDemandEquivalent(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	zero := 0.0
	plugin.OnDemandEquivalentMonthlyCost = &zero
	resource := &pbc.ResourceDescriptor{
		Provider: "aws", ResourceType: "aws:ec2/instance:Instance", Sku: "t3.micro", Region: "us-east-1",
	}
	projectedRequest := &pbc.GetProjectedCostRequest{Resource: resource}
	estimateRequest := &pbc.EstimateCostRequest{ResourceType: "aws:ec2/instance:Instance"}

	projected, err := plugin.GetProjectedCost(context.Background(), projectedRequest)
	require.NoError(t, err)
	estimate, err := plugin.EstimateCost(context.Background(), estimateRequest)
	require.NoError(t, err)
	assert.Nil(t, projected.OnDemandEquivalentMonthlyCost)
	assert.Nil(t, estimate.OnDemandEquivalentMonthlyCost)
	require.NoError(t, plugintesting.ValidateProjectedCostResponse(projected))
	require.NoError(t, plugintesting.ValidateEstimateCostResponse(estimate))

	plugin.SetPricingCategoryForResourceType("ec2", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
	projected, err = plugin.GetProjectedCost(context.Background(), projectedRequest)
	require.NoError(t, err)
	estimate, err = plugin.EstimateCost(context.Background(), estimateRequest)
	require.NoError(t, err)
	require.NotNil(t, projected.OnDemandEquivalentMonthlyCost)
	require.NotNil(t, estimate.OnDemandEquivalentMonthlyCost)
	assert.Zero(t, projected.GetOnDemandEquivalentMonthlyCost())
	assert.Zero(t, estimate.GetOnDemandEquivalentMonthlyCost())
	require.NoError(t, plugintesting.ValidateProjectedCostResponse(projected))
	require.NoError(t, plugintesting.ValidateEstimateCostResponse(estimate))

	plugin.SetPricingCategoryForResourceType("ec2", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED)
	projected, err = plugin.GetProjectedCost(context.Background(), projectedRequest)
	require.NoError(t, err)
	estimate, err = plugin.EstimateCost(context.Background(), estimateRequest)
	require.NoError(t, err)
	assert.Nil(t, projected.OnDemandEquivalentMonthlyCost)
	assert.Nil(t, estimate.OnDemandEquivalentMonthlyCost)

	plugin.SetPricingCategoryForResourceType("ec2", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
	projectedRequest.DryRun = true
	projected, err = plugin.GetProjectedCost(context.Background(), projectedRequest)
	require.NoError(t, err)
	assert.Nil(t, projected.OnDemandEquivalentMonthlyCost)
}

func TestLegacyMockLeavesOnDemandEquivalentAbsent(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockLegacyPlugin("legacy"))
	harness.Start(t)
	defer harness.Stop()
	projected, err := harness.Client().GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{})
	require.NoError(t, err)
	estimate, err := harness.Client().EstimateCost(t.Context(), &pbc.EstimateCostRequest{})
	require.NoError(t, err)
	assert.Nil(t, projected.OnDemandEquivalentMonthlyCost)
	assert.Nil(t, estimate.OnDemandEquivalentMonthlyCost)
	require.NoError(t, plugintesting.ValidateOnDemandEquivalent(
		projected.OnDemandEquivalentMonthlyCost, projected.GetPricingCategory(),
	))
	require.NoError(t, plugintesting.ValidateOnDemandEquivalent(
		estimate.OnDemandEquivalentMonthlyCost, estimate.GetPricingCategory(),
	))
}

func TestValidateOnDemandEquivalentZeroAllocs(t *testing.T) {
	value := 0.0
	dynamic := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC
	assert.Zero(t, testing.AllocsPerRun(1000, func() {
		if err := plugintesting.ValidateOnDemandEquivalent(&value, dynamic); err != nil {
			panic(err)
		}
	}))
	assert.Zero(t, testing.AllocsPerRun(1000, func() {
		if err := plugintesting.ValidateOnDemandEquivalent(nil, dynamic); err != nil {
			panic(err)
		}
	}))
}

func TestOnDemandEquivalentWirePresence(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.SetPricingCategoryForResourceType("ec2", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()
	resource := &pbc.ResourceDescriptor{
		Provider: "aws", ResourceType: "aws:ec2/instance:Instance", Sku: "t3.micro", Region: "us-east-1",
	}
	projectedReq := &pbc.GetProjectedCostRequest{Resource: resource}
	estimateReq := &pbc.EstimateCostRequest{ResourceType: "aws:ec2/instance:Instance"}

	for _, wantPresent := range []bool{false, true} {
		if wantPresent {
			zero := 0.0
			plugin.OnDemandEquivalentMonthlyCost = &zero
		}
		projected, err := harness.Client().GetProjectedCost(t.Context(), projectedReq)
		require.NoError(t, err)
		estimate, err := harness.Client().EstimateCost(t.Context(), estimateReq)
		require.NoError(t, err)
		assert.Equal(t, wantPresent, projected.OnDemandEquivalentMonthlyCost != nil)
		assert.Equal(t, wantPresent, estimate.OnDemandEquivalentMonthlyCost != nil)
		if wantPresent {
			assert.Zero(t, projected.GetOnDemandEquivalentMonthlyCost())
			assert.Zero(t, estimate.GetOnDemandEquivalentMonthlyCost())
		}

		batch := &pbc.BatchCostResponse{Results: []*pbc.ResourceCostResult{
			{Result: &pbc.ResourceCostResult_CostData{CostData: &pbc.CostData{
				Data: &pbc.CostData_ProjectedCost{ProjectedCost: projected}}}},
			{Result: &pbc.ResourceCostResult_CostData{CostData: &pbc.CostData{
				Data: &pbc.CostData_Estimate{Estimate: estimate}}}},
		}}
		wire, err := proto.Marshal(batch)
		require.NoError(t, err)
		var got pbc.BatchCostResponse
		require.NoError(t, proto.Unmarshal(wire, &got))
		gotProjected := got.GetResults()[0].GetCostData().GetProjectedCost()
		gotEstimate := got.GetResults()[1].GetCostData().GetEstimate()
		assert.Equal(t, wantPresent, gotProjected.OnDemandEquivalentMonthlyCost != nil)
		assert.Equal(t, wantPresent, gotEstimate.OnDemandEquivalentMonthlyCost != nil)
	}
}

func TestOnDemandEquivalentValidation(t *testing.T) {
	dynamic := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC
	standard := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD
	committed := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED
	unspecified := pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED
	ptr := func(v float64) *float64 { return &v }
	tests := []struct {
		name     string
		value    *float64
		category pbc.FocusPricingCategory
		want     error
	}{
		{"spot price", ptr(100), dynamic, nil},
		{"spot unset", nil, dynamic, nil},
		{"spot free tier", ptr(0), dynamic, nil},
		{"spot may exceed on demand", ptr(50), dynamic, nil},
		{"standard", ptr(100), standard, plugintesting.ErrOnDemandEquivalentInvalidCategory},
		{"standard zero", ptr(0), standard, plugintesting.ErrOnDemandEquivalentInvalidCategory},
		{"committed", ptr(100), committed, plugintesting.ErrOnDemandEquivalentInvalidCategory},
		{"unspecified", ptr(100), unspecified, plugintesting.ErrOnDemandEquivalentInvalidCategory},
		{"legacy", nil, unspecified, nil},
		{"NaN", ptr(math.NaN()), dynamic, plugintesting.ErrOnDemandEquivalentInvalidValue},
		{"positive infinity", ptr(math.Inf(1)), dynamic, plugintesting.ErrOnDemandEquivalentInvalidValue},
		{"negative infinity", ptr(math.Inf(-1)), dynamic, plugintesting.ErrOnDemandEquivalentInvalidValue},
		{"negative", ptr(-0.01), dynamic, plugintesting.ErrOnDemandEquivalentInvalidValue},
		{"value checked before category", ptr(math.NaN()), standard, plugintesting.ErrOnDemandEquivalentInvalidValue},
		{"dry run presence", ptr(0), unspecified, plugintesting.ErrOnDemandEquivalentInvalidCategory},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projected := &pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 80,
				PricingCategory: tc.category, OnDemandEquivalentMonthlyCost: tc.value}
			estimate := &pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 80,
				PricingCategory: tc.category, OnDemandEquivalentMonthlyCost: tc.value}
			for _, validate := range []func() error{
				func() error { return plugintesting.ValidateProjectedCostResponse(projected) },
				func() error { return plugintesting.ValidateEstimateCostResponse(estimate) },
			} {
				err := validate()
				if tc.want == nil {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, tc.want)
					assert.Contains(t, err.Error(), "on_demand_equivalent_monthly_cost")
				}
			}
		})
	}
}

func TestOnDemandEquivalentDryRunPresence(t *testing.T) {
	zero := 0.0
	response := &pbc.GetProjectedCostResponse{
		DryRunResult: &pbc.DryRunResponse{}, OnDemandEquivalentMonthlyCost: &zero,
	}
	require.ErrorIs(t, plugintesting.ValidateProjectedCostResponse(response),
		plugintesting.ErrOnDemandEquivalentInvalidCategory)
}
