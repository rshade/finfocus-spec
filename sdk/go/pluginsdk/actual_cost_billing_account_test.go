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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestActualCostBillingAccount shows that a billing account id passed on the request
// yields FOCUS records that pass ValidateFocusRecord, with no id invented by the plugin.
func TestActualCostBillingAccount(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	harness.Start(t)
	defer harness.Stop()

	start, end := plugintesting.CreateTimeRange(plugintesting.HoursPerDay)
	resp, err := harness.Client().GetActualCost(context.Background(), &pbc.GetActualCostRequest{
		ResourceId:       "i-billing-account",
		Start:            start,
		End:              end,
		BillingAccountId: "ba-123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults())

	for i, result := range resp.GetResults() {
		record := result.GetFocusRecord()
		require.NotNil(t, record, "result %d", i)
		assert.Equal(t, "ba-123", record.GetBillingAccountId(), "result %d", i)
		assert.NoError(t, pluginsdk.ValidateFocusRecord(record), "result %d", i)
	}
}
