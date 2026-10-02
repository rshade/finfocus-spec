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
)

func twoRegionPrices() []*pbc.RegionPrice {
	return []*pbc.RegionPrice{
		{Region: "eastus", UnitPrice: 0.09, MonthlyCost: 65.70, Currency: "USD"},
		{Region: "westeurope", UnitPrice: 0.11, MonthlyCost: 80.30, Currency: "EUR"},
	}
}

func TestValidateResponsesWithRegionPrices(t *testing.T) {
	t.Run("projected cost rows are never summed", func(t *testing.T) {
		resp := pluginsdk.NewGetProjectedCostResponse(
			pluginsdk.WithProjectedCostDetails(0.096, "USD", 70.08, "Standard_B2s in eastus2"),
			pluginsdk.WithProjectedCostRegionPrices(twoRegionPrices()...),
		)
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
		assert.InDelta(t, 70.08, resp.GetCostPerMonth(), 0)
		assert.Len(t, resp.GetRegionPrices(), 2)
	})

	t.Run("estimate rows are never summed", func(t *testing.T) {
		resp := pluginsdk.NewEstimateCostResponse(
			pluginsdk.WithEstimateCost("USD", 70.08),
			pluginsdk.WithEstimateCostRegionPrices(twoRegionPrices()...),
		)
		require.NoError(t, pluginsdk.ValidateEstimateCostResponse(resp))
		assert.InDelta(t, 70.08, resp.GetCostMonthly(), 0)
		assert.Len(t, resp.GetRegionPrices(), 2)
	})

	t.Run("bad row fails both validators", func(t *testing.T) {
		bad := &pbc.RegionPrice{Region: "eastus", UnitPrice: math.Inf(1), MonthlyCost: 65.70, Currency: "USD"}
		projected := &pbc.GetProjectedCostResponse{
			Currency: "USD", CostPerMonth: 70.08, RegionPrices: []*pbc.RegionPrice{bad},
		}
		require.ErrorIs(t, pluginsdk.ValidateGetProjectedCostResponse(projected), pluginsdk.ErrInvalidRegionPrice)
		estimate := &pbc.EstimateCostResponse{
			Currency: "USD", CostMonthly: 70.08, RegionPrices: []*pbc.RegionPrice{bad},
		}
		require.ErrorIs(t, pluginsdk.ValidateEstimateCostResponse(estimate), pluginsdk.ErrInvalidRegionPrice)
	})

	t.Run("rows on a dry-run projected response fail", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			Currency:     "USD",
			RegionPrices: twoRegionPrices(),
			DryRunResult: &pbc.DryRunResponse{},
		}
		require.ErrorIs(t, pluginsdk.ValidateGetProjectedCostResponse(resp), pluginsdk.ErrRegionPricesWithDryRun)
	})

	t.Run("responses without rows still pass", func(t *testing.T) {
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(&pbc.GetProjectedCostResponse{
			Currency: "USD", CostPerMonth: 70.08,
		}))
		require.NoError(t, pluginsdk.ValidateEstimateCostResponse(&pbc.EstimateCostResponse{
			Currency: "USD", CostMonthly: 70.08,
		}))
	})
}

func TestRegionPriceOptionsCopyRows(t *testing.T) {
	rows := twoRegionPrices()
	projected := pluginsdk.NewGetProjectedCostResponse(pluginsdk.WithProjectedCostRegionPrices(rows...))
	estimate := pluginsdk.NewEstimateCostResponse(pluginsdk.WithEstimateCostRegionPrices(rows...))

	rows[0].Region = "changed"
	rows[1] = nil

	require.Len(t, projected.GetRegionPrices(), 2)
	assert.Equal(t, "eastus", projected.GetRegionPrices()[0].GetRegion())
	assert.Equal(t, "westeurope", projected.GetRegionPrices()[1].GetRegion())
	require.Len(t, estimate.GetRegionPrices(), 2)
	assert.Equal(t, "eastus", estimate.GetRegionPrices()[0].GetRegion())

	assert.Nil(t, pluginsdk.NewGetProjectedCostResponse(pluginsdk.WithProjectedCostRegionPrices()).GetRegionPrices())
	assert.Nil(t, pluginsdk.NewEstimateCostResponse(pluginsdk.WithEstimateCostRegionPrices()).GetRegionPrices())
}

func fourRegionPrices() []*pbc.RegionPrice {
	return append(twoRegionPrices(),
		&pbc.RegionPrice{Region: "japaneast", UnitPrice: 0.12, MonthlyCost: 87.60, Currency: "JPY"},
		&pbc.RegionPrice{Region: "free", UnitPrice: 0, MonthlyCost: 0, Currency: "USD"},
	)
}

// BenchmarkValidateGetProjectedCostResponse_WithRegionPrices must stay at 0 allocs/op.
func BenchmarkValidateGetProjectedCostResponse_WithRegionPrices(b *testing.B) {
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:    0.096,
		Currency:     "USD",
		CostPerMonth: 70.08,
		RegionPrices: fourRegionPrices(),
	}
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateEstimateCostResponse_WithRegionPrices must stay at 0 allocs/op.
func BenchmarkValidateEstimateCostResponse_WithRegionPrices(b *testing.B) {
	resp := &pbc.EstimateCostResponse{
		Currency:     "USD",
		CostMonthly:  70.08,
		RegionPrices: fourRegionPrices(),
	}
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateEstimateCostResponse(resp)
	}
}
