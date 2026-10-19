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

package testing

import (
	"errors"
	"fmt"
	"math"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

var (
	// ErrOnDemandEquivalentInvalidValue reports a non-finite or negative present price.
	ErrOnDemandEquivalentInvalidValue = errors.New("on_demand_equivalent_monthly_cost must be finite and non-negative")
	// ErrOnDemandEquivalentInvalidCategory reports a present price outside DYNAMIC pricing.
	ErrOnDemandEquivalentInvalidCategory = errors.New("on_demand_equivalent_monthly_cost requires DYNAMIC pricing")
)

// ValidateOnDemandEquivalent validates a presence-tracked on-demand monthly price.
// An absent value is valid for every category, including DYNAMIC. The valid path
// does not allocate.
func ValidateOnDemandEquivalent(value *float64, category pbc.FocusPricingCategory) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return fmt.Errorf("%w: got %v", ErrOnDemandEquivalentInvalidValue, *value)
	}
	if category != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC {
		return fmt.Errorf("%w: got value=%v with category=%s",
			ErrOnDemandEquivalentInvalidCategory, *value, category.String())
	}
	return nil
}
