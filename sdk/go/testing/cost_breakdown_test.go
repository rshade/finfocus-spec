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
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const costBreakdownFieldNumber = 15

// TestGetProjectedCostBreakdown verifies that a mock configured with breakdown
// weights returns components scaled to cost_per_month over gRPC.
func TestGetProjectedCostBreakdown(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostBreakdown = map[string]float64{"compute": 3, "root_volume": 1}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
	})
	require.NoError(t, err)
	require.Positive(t, resp.GetCostPerMonth())

	breakdown := resp.GetCostBreakdown()
	require.Len(t, breakdown, 2)
	require.Contains(t, breakdown, "compute")
	require.Contains(t, breakdown, "root_volume")
	assert.InDelta(t, 3.0, breakdown["compute"]/breakdown["root_volume"], 1e-9)
	assert.InDelta(t, resp.GetCostPerMonth(), breakdown["compute"]+breakdown["root_volume"], 1e-9)
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostBreakdown_Nil verifies that a mock without breakdown
// weights returns an empty breakdown ("no breakdown available").
func TestGetProjectedCostBreakdown_Nil(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
	})
	require.NoError(t, err)
	assert.Empty(t, resp.GetCostBreakdown())
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostBreakdown_DryRun verifies that dry-run responses never
// carry a breakdown, even when the mock is configured with one.
func TestGetProjectedCostBreakdown_DryRun(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostBreakdown = map[string]float64{"compute": 1}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
		DryRun:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetDryRunResult())
	assert.Empty(t, resp.GetCostBreakdown())
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostBreakdown_ZeroCost verifies that weights on a zero-cost
// resource scale to all-zero components, which validate against a zero total.
func TestGetProjectedCostBreakdown_ZeroCost(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.BaseHourlyRate = 0
	plugin.ProjectedCostBreakdown = map[string]float64{"compute": 3, "root_volume": 1}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]float64{"compute": 0, "root_volume": 0}, resp.GetCostBreakdown())
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostBreakdown_InvalidWeights verifies that weights no scaling
// can make valid are reported as a FailedPrecondition configuration error.
func TestGetProjectedCostBreakdown_InvalidWeights(t *testing.T) {
	tests := []struct {
		name    string
		weights map[string]float64
	}{
		{"zero_sum", map[string]float64{"compute": 0, "storage": 0}},
		{"negative", map[string]float64{"compute": 2, "credit": -1}},
		{"nan", map[string]float64{"compute": math.NaN()}},
		{"inf", map[string]float64{"compute": math.Inf(1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := plugintesting.NewMockPlugin()
			plugin.ProjectedCostBreakdown = tt.weights

			harness := plugintesting.NewTestHarness(plugin)
			harness.Start(t)
			defer harness.Stop()

			_, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{
				Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
			})
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err))
			assert.Contains(t, err.Error(), "ProjectedCostBreakdown")
		})
	}
}

// TestBatchProjectedCostBreakdown verifies that the batch path carries the
// breakdown inside CostData.projected_cost.
func TestBatchProjectedCostBreakdown(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostBreakdown = map[string]float64{"compute": 3, "root_volume": 1}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
		Resources: []*pbc.ResourceDescriptor{
			plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)

	projected := resp.GetResults()[0].GetCostData().GetProjectedCost()
	require.NotNil(t, projected)
	require.Len(t, projected.GetCostBreakdown(), 2)
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(projected))
}

// TestCostBreakdownWireCompat verifies that an empty breakdown adds no field 15
// bytes, so the encoding matches a pre-feature response, and that a populated
// breakdown round-trips through proto marshaling.
func TestCostBreakdownWireCompat(t *testing.T) {
	t.Run("EmptyBreakdownOmitsField", func(t *testing.T) {
		for _, breakdown := range []map[string]float64{nil, {}} {
			data, err := proto.Marshal(&pbc.GetProjectedCostResponse{
				Currency:      "USD",
				CostPerMonth:  8.0,
				CostBreakdown: breakdown,
			})
			require.NoError(t, err)

			for b := data; len(b) > 0; {
				num, _, n := protowire.ConsumeField(b)
				require.GreaterOrEqual(t, n, 0, "malformed wire data")
				assert.NotEqual(t, protowire.Number(costBreakdownFieldNumber), num)
				b = b[n:]
			}
		}
	})

	t.Run("PopulatedBreakdownRoundTrips", func(t *testing.T) {
		original := &pbc.GetProjectedCostResponse{
			Currency:      "USD",
			CostPerMonth:  8.392,
			CostBreakdown: map[string]float64{"compute": 7.592, "root_volume": 0.80},
		}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		decoded := &pbc.GetProjectedCostResponse{}
		require.NoError(t, proto.Unmarshal(data, decoded))
		assert.True(t, proto.Equal(original, decoded))
	})
}

// BenchmarkMockGetProjectedCost_CostBreakdown measures the mock's breakdown
// weight scaling by calling GetProjectedCost directly, without gRPC overhead.
func BenchmarkMockGetProjectedCost_CostBreakdown(b *testing.B) {
	full := make(map[string]float64, 32)
	for i := range 32 {
		full[fmt.Sprintf("component_%02d", i)] = float64(i + 1)
	}
	cases := []struct {
		name    string
		weights map[string]float64
	}{
		{"none", nil},
		{"2_entries", map[string]float64{"compute": 3, "root_volume": 1}},
		{"32_entries", full},
	}

	req := &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			plugin := plugintesting.NewMockPlugin()
			plugin.ProjectedCostBreakdown = tc.weights
			ctx := context.Background()
			b.ReportAllocs()
			for range b.N {
				if _, err := plugin.GetProjectedCost(ctx, req); err != nil {
					b.Fatalf("GetProjectedCost() failed: %v", err)
				}
			}
		})
	}
}
