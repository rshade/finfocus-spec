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

func TestMockPlugin_IncludeDismissed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dismissed []string
		req       *pbc.GetRecommendationsRequest
		want      []string
	}{
		{
			name:      "default omits dismissed",
			dismissed: []string{"rec-2"},
			req:       &pbc.GetRecommendationsRequest{},
			want:      []string{"rec-1", "rec-3"},
		},
		{
			name:      "flag returns dismissed",
			dismissed: []string{"rec-2"},
			req:       &pbc.GetRecommendationsRequest{IncludeDismissed: true},
			want:      []string{"rec-1", "rec-2", "rec-3"},
		},
		{
			name:      "excluded wins over the flag",
			dismissed: []string{"rec-2"},
			req: &pbc.GetRecommendationsRequest{
				IncludeDismissed:          true,
				ExcludedRecommendationIds: []string{"rec-2"},
			},
			want: []string{"rec-1", "rec-3"},
		},
		{
			name:      "excluded live id is omitted when the flag is false",
			dismissed: []string{"rec-2"},
			req: &pbc.GetRecommendationsRequest{
				ExcludedRecommendationIds: []string{"rec-1"},
			},
			want: []string{"rec-3"},
		},
		{
			name:      "dismissal filter runs before pagination",
			dismissed: []string{"rec-1"},
			req:       &pbc.GetRecommendationsRequest{PageSize: 1},
			want:      []string{"rec-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := plugintesting.NewMockPlugin()
			plugin.SetRecommendationsConfig(plugintesting.RecommendationsConfig{
				Recommendations: []*pbc.Recommendation{
					{Id: "rec-1"},
					{Id: "rec-2"},
					{Id: "rec-3"},
				},
				DismissedIDs: tt.dismissed,
			})

			resp, err := plugin.GetRecommendations(context.Background(), tt.req)
			require.NoError(t, err)
			assert.Equal(t, tt.want, recommendationIDs(resp.GetRecommendations()))
		})
	}
}

func recommendationIDs(recs []*pbc.Recommendation) []string {
	ids := make([]string, 0, len(recs))
	for _, rec := range recs {
		ids = append(ids, rec.GetId())
	}
	return ids
}
