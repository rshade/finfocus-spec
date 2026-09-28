package testing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
