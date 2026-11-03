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

package finfocus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestRecommendationSpotFieldNumbers(t *testing.T) {
	fields := (&pbc.Recommendation{}).ProtoReflect().Descriptor().Fields()

	category := fields.ByName("pricing_category")
	require.NotNil(t, category, "pricing_category is missing from Recommendation")
	assert.Equal(t, protoreflect.FieldNumber(20), category.Number())
	assert.Equal(t, protoreflect.EnumKind, category.Kind())

	risk := fields.ByName("spot_interruption_risk_score")
	require.NotNil(t, risk, "spot_interruption_risk_score is missing from Recommendation")
	assert.Equal(t, protoreflect.FieldNumber(21), risk.Number())
	assert.Equal(t, protoreflect.DoubleKind, risk.Kind())
}

func TestRecommendationSpotRoundTrip(t *testing.T) {
	empty := &pbc.Recommendation{}
	assert.Equal(t, pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED, empty.GetPricingCategory())
	assert.Zero(t, empty.GetSpotInterruptionRiskScore())

	spot := &pbc.Recommendation{
		Id:                        "rec-1",
		PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
		SpotInterruptionRiskScore: 0.12,
	}
	wire, err := proto.Marshal(spot)
	require.NoError(t, err)

	decoded := &pbc.Recommendation{}
	require.NoError(t, proto.Unmarshal(wire, decoded))
	assert.Equal(t, pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC, decoded.GetPricingCategory())
	assert.InDelta(t, 0.12, decoded.GetSpotInterruptionRiskScore(), 0)
}
