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

const testBillingAccountID = "ba-123"

func newBillingAccountRequest(billingAccountID string) *pbc.GetActualCostRequest {
	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	return &pbc.GetActualCostRequest{
		ResourceId:       "i-billing-account",
		Start:            start,
		End:              end,
		BillingAccountId: billingAccountID,
	}
}

func TestMockActualCostBillingAccount(t *testing.T) {
	plugin := plugintesting.NewMockPlugin()
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()
	ctx := context.Background()

	t.Run("populated id is echoed into every FOCUS record", func(t *testing.T) {
		req := newBillingAccountRequest(testBillingAccountID)
		resp, err := harness.Client().GetActualCost(ctx, req)
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())

		for i, result := range resp.GetResults() {
			record := result.GetFocusRecord()
			require.NotNil(t, record, "result %d", i)
			assert.Equal(t, testBillingAccountID, record.GetBillingAccountId(), "result %d", i)
			assert.Equal(t, req.GetResourceId(), record.GetResourceId(), "result %d", i)
			assert.InDelta(t, result.GetCost(), record.GetBilledCost(), 0, "result %d", i)
		}
	})

	t.Run("empty id attaches no FOCUS record", func(t *testing.T) {
		resp, err := harness.Client().GetActualCost(ctx, newBillingAccountRequest(""))
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResults())

		for i, result := range resp.GetResults() {
			assert.Nil(t, result.GetFocusRecord(), "result %d", i)
		}
	})

	t.Run("id does not change cost", func(t *testing.T) {
		without, err := harness.Client().GetActualCost(ctx, newBillingAccountRequest(""))
		require.NoError(t, err)
		with, err := harness.Client().GetActualCost(ctx, newBillingAccountRequest(testBillingAccountID))
		require.NoError(t, err)

		require.Len(t, with.GetResults(), len(without.GetResults()))
		for i := range without.GetResults() {
			assert.InDelta(t, without.GetResults()[i].GetCost(), with.GetResults()[i].GetCost(), 0, "result %d", i)
			assert.InDelta(t, without.GetResults()[i].GetUsageAmount(),
				with.GetResults()[i].GetUsageAmount(), 0, "result %d", i)
		}
	})

	t.Run("id is echoed on every page", func(t *testing.T) {
		plugin.SetActualCostDataPoints(plugintesting.HoursPerDay)
		req := newBillingAccountRequest(testBillingAccountID)
		req.PageSize = 5

		pages := 0
		for {
			resp, err := harness.Client().GetActualCost(ctx, req)
			require.NoError(t, err)
			pages++
			for i, result := range resp.GetResults() {
				assert.Equal(t, testBillingAccountID, result.GetFocusRecord().GetBillingAccountId(),
					"page %d result %d", pages, i)
			}
			if resp.GetNextPageToken() == "" {
				break
			}
			req.PageToken = resp.GetNextPageToken()
		}
		assert.Greater(t, pages, 1)
	})

	t.Run("dry run ignores the id", func(t *testing.T) {
		req := newBillingAccountRequest(testBillingAccountID)
		req.DryRun = true
		resp, err := harness.Client().GetActualCost(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.GetDryRunResult())
		assert.Empty(t, resp.GetResults())
	})
}
