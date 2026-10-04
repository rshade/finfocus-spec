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

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func newActualCostResourceRequest(resource *pbc.ResourceDescriptor) *pbc.GetActualCostRequest {
	req := newBillingAccountRequest(testBillingAccountID)
	req.Resource = resource
	return req
}

func TestMockActualCostResource(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()
	ctx := context.Background()

	resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	t.Run("FOCUS records describe the requested resource", func(t *testing.T) {
		resp, err := harness.Client().GetActualCost(ctx, newActualCostResourceRequest(resource))
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())

		for i, result := range resp.GetResults() {
			record := result.GetFocusRecord()
			require.NotNil(t, record, "result %d", i)
			assert.Equal(t, "ec2", record.GetResourceType(), "result %d", i)
			assert.Equal(t, "us-east-1", record.GetRegionId(), "result %d", i)
			assert.Equal(t, "t3.micro", record.GetSkuId(), "result %d", i)
		}
	})

	t.Run("no descriptor leaves those columns empty", func(t *testing.T) {
		resp, err := harness.Client().GetActualCost(ctx, newActualCostResourceRequest(nil))
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())

		for i, result := range resp.GetResults() {
			record := result.GetFocusRecord()
			require.NotNil(t, record, "result %d", i)
			assert.Empty(t, record.GetResourceType(), "result %d", i)
			assert.Empty(t, record.GetRegionId(), "result %d", i)
			assert.Empty(t, record.GetSkuId(), "result %d", i)
		}
	})

	t.Run("descriptor does not change cost", func(t *testing.T) {
		without, err := harness.Client().GetActualCost(ctx, newActualCostResourceRequest(nil))
		require.NoError(t, err)
		with, err := harness.Client().GetActualCost(ctx, newActualCostResourceRequest(resource))
		require.NoError(t, err)

		require.Len(t, with.GetResults(), len(without.GetResults()))
		for i := range without.GetResults() {
			assert.InDelta(t, without.GetResults()[i].GetCost(), with.GetResults()[i].GetCost(), 0, "result %d", i)
		}
	})
}
