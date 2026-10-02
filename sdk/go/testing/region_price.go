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

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

var (
	// ErrInvalidRegionPrice reports a region_prices row that breaks a RegionPrice rule.
	ErrInvalidRegionPrice = errors.New("invalid region price")
	// ErrRegionPricesWithDryRun reports region_prices on a dry-run GetProjectedCostResponse.
	ErrRegionPricesWithDryRun = errors.New("region_prices must be empty for dry-run responses")
)

// ValidateRegionPrices checks the rows of a region_prices list, failing on the first
// bad row. Each row needs a region, a unit_price and monthly_cost that are finite and
// non-negative, and a valid ISO 4217 currency. The list is advisory, so it is never
// summed or compared with the parent response's cost. Errors wrap
// ErrInvalidRegionPrice and name region_prices[i] and, for a present row, the field.
// A valid list returns nil without allocating.
func ValidateRegionPrices(rows []*pbc.RegionPrice) error {
	for i, row := range rows {
		if err := validateRegionPrice(i, row); err != nil {
			return err
		}
	}
	return nil
}

func validateRegionPrice(i int, row *pbc.RegionPrice) error {
	if row == nil {
		return fmt.Errorf("%w: region_prices[%d] is nil", ErrInvalidRegionPrice, i)
	}
	if row.GetRegion() == "" {
		return fmt.Errorf("%w: region_prices[%d].region is required", ErrInvalidRegionPrice, i)
	}
	if !isFiniteNonNegative(row.GetUnitPrice()) {
		return fmt.Errorf("%w: region_prices[%d].unit_price must be finite and non-negative, got %v",
			ErrInvalidRegionPrice, i, row.GetUnitPrice())
	}
	if !isFiniteNonNegative(row.GetMonthlyCost()) {
		return fmt.Errorf("%w: region_prices[%d].monthly_cost must be finite and non-negative, got %v",
			ErrInvalidRegionPrice, i, row.GetMonthlyCost())
	}
	if !currency.IsValid(row.GetCurrency()) {
		return fmt.Errorf("%w: region_prices[%d].currency must be an ISO 4217 code, got %q",
			ErrInvalidRegionPrice, i, row.GetCurrency())
	}
	return nil
}

func isFiniteNonNegative(v float64) bool {
	return v >= 0 && !math.IsInf(v, 1)
}
