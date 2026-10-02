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
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const (
	priceOptionsProjectedFieldNumber = 16
	priceOptionsEstimateFieldNumber  = 6
)

// mockPriceOptions returns an on-demand, a 1-year reservation, and a 3-year
// savings plan option. The mock returns them as configured.
func mockPriceOptions() []*pbc.PriceOption {
	return []*pbc.PriceOption{
		{
			Category:    pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			Model:       "Consumption",
			UnitPrice:   0.0104,
			MonthlyCost: 7.592,
		},
		{
			Category:        pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			Model:           "Reservation",
			Term:            "1 Year",
			UnitPrice:       0.0065,
			MonthlyCost:     4.745,
			UpfrontCost:     56.94,
			SavingsFraction: 0.375,
		},
		{
			Category:        pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			Model:           "SavingsPlan",
			Term:            "3 Years",
			UnitPrice:       0.0052,
			MonthlyCost:     3.796,
			SavingsFraction: 0.5,
		},
	}
}

func projectedRequest() *pbc.GetProjectedCostRequest {
	return &pbc.GetProjectedCostRequest{
		Resource: plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1"),
	}
}

func estimateRequest() *pbc.EstimateCostRequest {
	return &pbc.EstimateCostRequest{ResourceType: "aws:ec2/instance:Instance"}
}

// assertPriceOptionsEqual compares option lists with proto.Equal per entry.
func assertPriceOptionsEqual(t *testing.T, want, got []*pbc.PriceOption) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], got[i]), "price_options[%d]", i)
	}
}

// TestGetProjectedCostPriceOptions verifies that configured options reach the
// client over gRPC and leave cost_per_month unchanged.
func TestGetProjectedCostPriceOptions(t *testing.T) {
	plain := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	plain.Start(t)
	defer plain.Stop()
	baseline, err := plain.Client().GetProjectedCost(context.Background(), projectedRequest())
	require.NoError(t, err)

	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostPriceOptions = mockPriceOptions()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), projectedRequest())
	require.NoError(t, err)
	assertPriceOptionsEqual(t, mockPriceOptions(), resp.GetPriceOptions())
	assert.InDelta(t, baseline.GetCostPerMonth(), resp.GetCostPerMonth(), 0)
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostPriceOptions_Nil verifies that a mock without options
// returns an empty list.
func TestGetProjectedCostPriceOptions_Nil(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().GetProjectedCost(context.Background(), projectedRequest())
	require.NoError(t, err)
	assert.Empty(t, resp.GetPriceOptions())
}

// TestGetProjectedCostPriceOptions_DryRun verifies that dry-run responses never
// carry options, even when the mock is configured with them.
func TestGetProjectedCostPriceOptions_DryRun(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostPriceOptions = mockPriceOptions()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	req := projectedRequest()
	req.DryRun = true
	resp, err := harness.Client().GetProjectedCost(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.GetDryRunResult())
	assert.Empty(t, resp.GetPriceOptions())
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
}

// TestGetProjectedCostPriceOptions_Isolation calls the mock directly, because
// gRPC serialization would hide shared pointers, and checks that changing one
// response changes neither the next response nor the mock's configuration.
func TestGetProjectedCostPriceOptions_Isolation(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostPriceOptions = mockPriceOptions()

	first, err := plugin.GetProjectedCost(context.Background(), projectedRequest())
	require.NoError(t, err)
	first.GetPriceOptions()[0].UnitPrice = 99

	second, err := plugin.GetProjectedCost(context.Background(), projectedRequest())
	require.NoError(t, err)
	assert.InDelta(t, 0.0104, second.GetPriceOptions()[0].GetUnitPrice(), 0)
	assert.InDelta(t, 0.0104, plugin.ProjectedCostPriceOptions[0].GetUnitPrice(), 0)
}

// TestBatchProjectedCostPriceOptions verifies that the batch path carries the
// options inside CostData.projected_cost.
func TestBatchProjectedCostPriceOptions(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	plugin.ProjectedCostPriceOptions = mockPriceOptions()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().BatchCost(context.Background(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
		Resources: []*pbc.ResourceDescriptor{projectedRequest().GetResource()},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)

	projected := resp.GetResults()[0].GetCostData().GetProjectedCost()
	require.NotNil(t, projected)
	assertPriceOptionsEqual(t, mockPriceOptions(), projected.GetPriceOptions())
}

// TestEstimateCostPriceOptions verifies that configured options reach the
// client on EstimateCost and leave cost_monthly unchanged.
func TestEstimateCostPriceOptions(t *testing.T) {
	plain := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	plain.Start(t)
	defer plain.Stop()
	baseline, err := plain.Client().EstimateCost(context.Background(), estimateRequest())
	require.NoError(t, err)

	plugin := plugintesting.NewMockPlugin()
	plugin.EstimateCostPriceOptions = mockPriceOptions()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().EstimateCost(context.Background(), estimateRequest())
	require.NoError(t, err)
	assertPriceOptionsEqual(t, mockPriceOptions(), resp.GetPriceOptions())
	assert.InDelta(t, baseline.GetCostMonthly(), resp.GetCostMonthly(), 0)
	require.NoError(t, pluginsdk.ValidateEstimateCostResponse(resp))
}

// TestEstimateCostPriceOptions_Nil verifies that a mock without options
// returns an empty list on EstimateCost.
func TestEstimateCostPriceOptions_Nil(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	harness.Start(t)
	defer harness.Stop()

	resp, err := harness.Client().EstimateCost(context.Background(), estimateRequest())
	require.NoError(t, err)
	assert.Empty(t, resp.GetPriceOptions())
}

// fieldNumbers returns the top-level field numbers present in wire data.
func fieldNumbers(t *testing.T, data []byte) []protowire.Number {
	t.Helper()
	var nums []protowire.Number
	for b := data; len(b) > 0; {
		num, _, n := protowire.ConsumeField(b)
		require.GreaterOrEqual(t, n, 0, "malformed wire data")
		nums = append(nums, num)
		b = b[n:]
	}
	return nums
}

// preFeatureMessage builds a dynamic GetProjectedCostResponse from a copy of
// the costsource.proto descriptor with field 16 removed, which is how a host
// built before price_options sees the message.
func preFeatureMessage(t *testing.T) *dynamicpb.Message {
	t.Helper()
	fdp := protodesc.ToFileDescriptorProto(pbc.File_finfocus_v1_costsource_proto)
	for _, msg := range fdp.GetMessageType() {
		if msg.GetName() != "GetProjectedCostResponse" {
			continue
		}
		kept := make([]*descriptorpb.FieldDescriptorProto, 0, len(msg.GetField()))
		for _, f := range msg.GetField() {
			if f.GetNumber() != priceOptionsProjectedFieldNumber {
				kept = append(kept, f)
			}
		}
		msg.Field = kept
	}
	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	require.NoError(t, err)
	md := fd.Messages().ByName("GetProjectedCostResponse")
	require.NotNil(t, md)
	require.Nil(t, md.Fields().ByNumber(priceOptionsProjectedFieldNumber))
	return dynamicpb.NewMessage(md)
}

// TestPriceOptionsWireCompat verifies that an empty list adds no bytes, that a
// populated list round-trips, and that a pre-feature host decodes it.
func TestPriceOptionsWireCompat(t *testing.T) {
	t.Run("EmptyListOmitsField", func(t *testing.T) {
		for _, options := range [][]*pbc.PriceOption{nil, {}} {
			projected, err := proto.Marshal(&pbc.GetProjectedCostResponse{
				Currency: "USD", CostPerMonth: 8.0, PriceOptions: options,
			})
			require.NoError(t, err)
			assert.NotContains(t, fieldNumbers(t, projected), protowire.Number(priceOptionsProjectedFieldNumber))

			estimate, err := proto.Marshal(&pbc.EstimateCostResponse{
				Currency: "USD", CostMonthly: 8.0, PriceOptions: options,
			})
			require.NoError(t, err)
			assert.NotContains(t, fieldNumbers(t, estimate), protowire.Number(priceOptionsEstimateFieldNumber))
		}
	})

	t.Run("PopulatedListRoundTrips", func(t *testing.T) {
		for _, original := range []proto.Message{
			&pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 7.592, PriceOptions: mockPriceOptions()},
			&pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 7.592, PriceOptions: mockPriceOptions()},
		} {
			data, err := proto.Marshal(original)
			require.NoError(t, err)
			decoded := original.ProtoReflect().New().Interface()
			require.NoError(t, proto.Unmarshal(data, decoded))
			assert.True(t, proto.Equal(original, decoded))
		}
	})

	t.Run("OlderConsumerDecodes", func(t *testing.T) {
		data, err := proto.Marshal(&pbc.GetProjectedCostResponse{
			Currency: "USD", CostPerMonth: 7.592, PriceOptions: mockPriceOptions(),
		})
		require.NoError(t, err)

		old := preFeatureMessage(t)
		require.NoError(t, proto.Unmarshal(data, old))

		costField := old.Descriptor().Fields().ByName(protoreflect.Name("cost_per_month"))
		require.NotNil(t, costField)
		assert.InDelta(t, 7.592, old.Get(costField).Float(), 0)

		unknown := fieldNumbers(t, old.GetUnknown())
		require.Len(t, unknown, len(mockPriceOptions()))
		for _, num := range unknown {
			assert.Equal(t, protowire.Number(priceOptionsProjectedFieldNumber), num)
		}
	})
}
