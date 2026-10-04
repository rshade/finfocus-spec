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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestIncludeDismissedFieldNumber(t *testing.T) {
	t.Parallel()

	field := (&pbc.GetRecommendationsRequest{}).ProtoReflect().Descriptor().Fields().ByName("include_dismissed")
	require.NotNil(t, field)
	assert.Equal(t, protoreflect.FieldNumber(8), field.Number())
	assert.False(t, (&pbc.GetRecommendationsRequest{}).GetIncludeDismissed())
}

func TestApplyRecommendationVisibility(t *testing.T) {
	t.Parallel()

	rec1 := &pbc.Recommendation{Id: "rec-1"}
	rec2 := &pbc.Recommendation{Id: "rec-2"}
	recBlank := &pbc.Recommendation{}
	input := []*pbc.Recommendation{rec1, nil, rec2, recBlank}

	t.Run("empty lists return the same slice", func(t *testing.T) {
		t.Parallel()
		got := pluginsdk.ApplyRecommendationVisibility(input, pluginsdk.RecommendationVisibility{})
		require.Len(t, got, len(input))
		assert.Same(t, &input[0], &got[0], "empty id lists must not copy the slice")
	})

	t.Run("dismissed ids are omitted by default", func(t *testing.T) {
		t.Parallel()
		got := pluginsdk.ApplyRecommendationVisibility(input, pluginsdk.RecommendationVisibility{
			DismissedIDs: []string{"rec-2", ""},
		})
		require.Len(t, got, 3)
		assert.Same(t, rec1, got[0])
		assert.Nil(t, got[1])
		assert.Same(t, recBlank, got[2])
	})

	t.Run("exclusion omits a live id when the flag is false", func(t *testing.T) {
		t.Parallel()
		got := pluginsdk.ApplyRecommendationVisibility(input, pluginsdk.RecommendationVisibility{
			DismissedIDs: []string{"rec-2"},
			ExcludedIDs:  []string{"rec-1"},
		})
		require.Len(t, got, 2)
		assert.Nil(t, got[0])
		assert.Same(t, recBlank, got[1])
	})

	t.Run("include dismissed keeps them and exclusion still wins", func(t *testing.T) {
		t.Parallel()
		got := pluginsdk.ApplyRecommendationVisibility(input, pluginsdk.RecommendationVisibility{
			DismissedIDs:     []string{"rec-2"},
			IncludeDismissed: true,
			ExcludedIDs:      []string{"rec-1"},
		})
		require.Len(t, got, 3)
		assert.Nil(t, got[0])
		assert.Same(t, rec2, got[1])
		assert.Same(t, recBlank, got[2])
	})
}
