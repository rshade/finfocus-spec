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

func regionPrice(region string, unitPrice, monthlyCost float64, currency string) *pbc.RegionPrice {
	return &pbc.RegionPrice{Region: region, UnitPrice: unitPrice, MonthlyCost: monthlyCost, Currency: currency}
}

func TestValidateRegionPrices(t *testing.T) {
	valid := regionPrice("eastus", 0.09, 65.70, "USD")

	tests := []struct {
		name    string
		rows    []*pbc.RegionPrice
		wantMsg string
	}{
		{name: "nil slice", rows: nil},
		{name: "empty slice", rows: []*pbc.RegionPrice{}},
		{name: "two valid rows", rows: []*pbc.RegionPrice{valid, regionPrice("westeurope", 0.11, 80.30, "EUR")}},
		{name: "zero price is a real price", rows: []*pbc.RegionPrice{regionPrice("free", 0, 0, "USD")}},
		{name: "nil row", rows: []*pbc.RegionPrice{valid, nil}, wantMsg: "region_prices[1]"},
		{
			name:    "empty region",
			rows:    []*pbc.RegionPrice{valid, regionPrice("", 0.09, 65.70, "USD")},
			wantMsg: "region_prices[1].region",
		},
		{
			name:    "NaN unit price",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", math.NaN(), 65.70, "USD")},
			wantMsg: "region_prices[1].unit_price",
		},
		{
			name:    "infinite unit price",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", math.Inf(1), 65.70, "USD")},
			wantMsg: "region_prices[1].unit_price",
		},
		{
			name:    "negative unit price",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", -0.01, 65.70, "USD")},
			wantMsg: "region_prices[1].unit_price",
		},
		{
			name:    "NaN monthly cost",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", 0.09, math.NaN(), "USD")},
			wantMsg: "region_prices[1].monthly_cost",
		},
		{
			name:    "negative monthly cost",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", 0.09, -1, "USD")},
			wantMsg: "region_prices[1].monthly_cost",
		},
		{
			name:    "empty currency",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", 0.09, 65.70, "")},
			wantMsg: "region_prices[1].currency",
		},
		{
			name:    "non-ISO currency",
			rows:    []*pbc.RegionPrice{valid, regionPrice("eastus2", 0.09, 65.70, "XXQ")},
			wantMsg: "region_prices[1].currency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateRegionPrices(tt.rows)
			if tt.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, plugintesting.ErrInvalidRegionPrice)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestMockRegionPrices(t *testing.T) {
	rows := []*pbc.RegionPrice{
		regionPrice("eastus", 0.09, 65.70, "USD"),
		regionPrice("westeurope", 0.11, 80.30, "EUR"),
	}
	ctx := context.Background()
	projectedReq := &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("azure", "vm", "Standard_B2s", "eastus2"),
	}
	estimateReq := &pbc.EstimateCostRequest{ResourceType: "azure:compute/virtualMachine:VirtualMachine"}

	t.Run("configured rows are returned and validate", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.RegionPrices = rows
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		projected, err := harness.Client().GetProjectedCost(ctx, projectedReq)
		require.NoError(t, err)
		require.Len(t, projected.GetRegionPrices(), 2)
		assert.Equal(t, "westeurope", projected.GetRegionPrices()[1].GetRegion())
		require.NoError(t, plugintesting.ValidateProjectedCostResponse(projected))

		estimate, err := harness.Client().EstimateCost(ctx, estimateReq)
		require.NoError(t, err)
		require.Len(t, estimate.GetRegionPrices(), 2)
		require.NoError(t, plugintesting.ValidateEstimateCostResponse(estimate))
	})

	t.Run("dry run returns no rows", func(t *testing.T) {
		plugin := plugintesting.NewMockPlugin()
		plugin.RegionPrices = rows
		harness := plugintesting.NewTestHarness(plugin)
		harness.Start(t)
		defer harness.Stop()

		resp, err := harness.Client().GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: projectedReq.GetResource(),
			DryRun:   true,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetRegionPrices())
	})

	t.Run("unset leaves both responses empty", func(t *testing.T) {
		harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
		harness.Start(t)
		defer harness.Stop()

		projected, err := harness.Client().GetProjectedCost(ctx, projectedReq)
		require.NoError(t, err)
		assert.Empty(t, projected.GetRegionPrices())
		estimate, err := harness.Client().EstimateCost(ctx, estimateReq)
		require.NoError(t, err)
		assert.Empty(t, estimate.GetRegionPrices())
	})
}

func TestHarnessValidatorsRejectBadRegionPrices(t *testing.T) {
	bad := []*pbc.RegionPrice{regionPrice("eastus", math.NaN(), 65.70, "USD")}

	err := plugintesting.ValidateProjectedCostResponse(&pbc.GetProjectedCostResponse{
		Currency: "USD", CostPerMonth: 70.08, RegionPrices: bad,
	})
	require.ErrorIs(t, err, plugintesting.ErrInvalidRegionPrice)

	err = plugintesting.ValidateEstimateCostResponse(&pbc.EstimateCostResponse{
		Currency: "USD", CostMonthly: 70.08, RegionPrices: bad,
	})
	require.ErrorIs(t, err, plugintesting.ErrInvalidRegionPrice)

	err = plugintesting.ValidateProjectedCostResponse(&pbc.GetProjectedCostResponse{
		Currency:     "USD",
		RegionPrices: []*pbc.RegionPrice{regionPrice("eastus", 0.09, 65.70, "USD")},
		DryRunResult: &pbc.DryRunResponse{},
	})
	require.ErrorIs(t, err, plugintesting.ErrRegionPricesWithDryRun)
}
