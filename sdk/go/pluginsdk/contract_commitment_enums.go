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

package pluginsdk

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// Package-level slices make enum membership a zero-allocation scan.
// UNSPECIFIED is a known value; callers that require a set column reject it separately.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var (
	allBenefitCategories = []pbc.FocusContractCommitmentBenefitCategory{
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_UNSPECIFIED,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_ENTITLEMENT,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_AVAILABILITY,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_OTHER,
	}
	allFulfillmentIntervals = []pbc.FocusContractCommitmentFulfillmentInterval{
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_UNSPECIFIED,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_HOURLY,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_DAILY,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_WEEKLY,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_MONTHLY,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_QUARTERLY,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_SEMI_ANNUAL,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_ANNUAL,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_FULL_PERIOD,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_TRANSACTIONAL,
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_CUSTOM,
	}
	allLifecycleStatuses = []pbc.FocusContractCommitmentLifecycleStatus{
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_UNSPECIFIED,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_PROPOSED,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_PENDING,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_ACTIVE,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_EXHAUSTED,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_EXPIRED,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_CANCELED,
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_SUPERSEDED,
	}
	allCommitmentModels = []pbc.FocusContractCommitmentModel{
		pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_UNSPECIFIED,
		pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_CONTINUOUS,
		pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_DISCONTINUOUS,
	}
	allOfferCategories = []pbc.FocusContractCommitmentOfferCategory{
		pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_UNSPECIFIED,
		pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_PUBLIC,
		pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_NEGOTIATED,
	}
	allPaymentIntervals = []pbc.FocusContractCommitmentPaymentInterval{
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_UNSPECIFIED,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_ONE_TIME,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_MONTHLY,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_QUARTERLY,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_SEMI_ANNUAL,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_ANNUAL,
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_CUSTOM,
	}
	allPaymentModels = []pbc.FocusContractCommitmentPaymentModel{
		pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_UNSPECIFIED,
		pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_NO_UPFRONT,
		pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_PARTIAL_UPFRONT,
		pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_ALL_UPFRONT,
	}
)

func knownEnum[T ~int32](value T, all []T) bool {
	for _, candidate := range all {
		if candidate == value {
			return true
		}
	}
	return false
}

// IsValidContractCommitmentBenefitCategory reports whether value is a defined benefit category,
// including UNSPECIFIED. Unknown numeric values return false. The check does not allocate.
func IsValidContractCommitmentBenefitCategory(value pbc.FocusContractCommitmentBenefitCategory) bool {
	return knownEnum(value, allBenefitCategories)
}

// IsValidContractCommitmentFulfillmentInterval reports whether value is a defined fulfillment
// interval, including UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentFulfillmentInterval(value pbc.FocusContractCommitmentFulfillmentInterval) bool {
	return knownEnum(value, allFulfillmentIntervals)
}

// IsValidContractCommitmentLifecycleStatus reports whether value is a defined lifecycle status,
// including UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentLifecycleStatus(value pbc.FocusContractCommitmentLifecycleStatus) bool {
	return knownEnum(value, allLifecycleStatuses)
}

// IsValidContractCommitmentModel reports whether value is a defined commitment model, including
// UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentModel(value pbc.FocusContractCommitmentModel) bool {
	return knownEnum(value, allCommitmentModels)
}

// IsValidContractCommitmentOfferCategory reports whether value is a defined offer category,
// including UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentOfferCategory(value pbc.FocusContractCommitmentOfferCategory) bool {
	return knownEnum(value, allOfferCategories)
}

// IsValidContractCommitmentPaymentInterval reports whether value is a defined payment interval,
// including UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentPaymentInterval(value pbc.FocusContractCommitmentPaymentInterval) bool {
	return knownEnum(value, allPaymentIntervals)
}

// IsValidContractCommitmentPaymentModel reports whether value is a defined payment model,
// including UNSPECIFIED. The check does not allocate.
func IsValidContractCommitmentPaymentModel(value pbc.FocusContractCommitmentPaymentModel) bool {
	return knownEnum(value, allPaymentModels)
}
