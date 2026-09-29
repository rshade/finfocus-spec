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
	"math"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// validateFocus14Commitment checks the FOCUS 1.4 columns and the 1.3 description
// gap's neighbors that this package enforces. Description itself may be empty.
// Cross-row lifecycle rules are out of scope. It does not allocate on valid input.
func validateFocus14Commitment(c *pbc.ContractCommitment) error {
	if !isJSONObject(c.GetContractCommitmentApplicability()) {
		return commitmentError("contract_commitment_applicability must be a JSON object")
	}
	if err := requireEnum("contract_commitment_benefit_category",
		int32(c.GetContractCommitmentBenefitCategory()),
		pbc.FocusContractCommitmentBenefitCategory_name); err != nil {
		return err
	}
	if err := validateDiscount(c); err != nil {
		return err
	}
	if !validDurationType(c.GetContractCommitmentDurationType()) {
		return commitmentError(
			"contract_commitment_duration_type must be a positive integer and a FOCUS unit")
	}
	if err := requireEnum("contract_commitment_fulfillment_interval",
		int32(c.GetContractCommitmentFulfillmentInterval()),
		pbc.FocusContractCommitmentFulfillmentInterval_name); err != nil {
		return err
	}
	if err := validateCommitmentTimestamps(c); err != nil {
		return err
	}
	if err := requireEnum("contract_commitment_lifecycle_status",
		int32(c.GetContractCommitmentLifecycleStatus()),
		pbc.FocusContractCommitmentLifecycleStatus_name); err != nil {
		return err
	}
	if err := requireEnum("contract_commitment_model",
		int32(c.GetContractCommitmentModel()),
		pbc.FocusContractCommitmentModel_name); err != nil {
		return err
	}
	if err := validateModelInterval(c); err != nil {
		return err
	}
	if err := requireEnum("contract_commitment_offer_category",
		int32(c.GetContractCommitmentOfferCategory()),
		pbc.FocusContractCommitmentOfferCategory_name); err != nil {
		return err
	}
	if err := requireEnum("contract_commitment_payment_interval",
		int32(c.GetContractCommitmentPaymentInterval()),
		pbc.FocusContractCommitmentPaymentInterval_name); err != nil {
		return err
	}
	if err := requireEnum("contract_commitment_payment_model",
		int32(c.GetContractCommitmentPaymentModel()),
		pbc.FocusContractCommitmentPaymentModel_name); err != nil {
		return err
	}
	if err := validatePaymentTerms(c); err != nil {
		return err
	}
	if c.GetInvoiceIssuerName() == "" {
		return commitmentError("invoice_issuer_name is required")
	}
	if c.GetServiceProviderName() == "" {
		return commitmentError("service_provider_name is required")
	}
	return validatePricingCurrency(c)
}

func requireEnum(name string, value int32, names map[int32]string) error {
	if value == 0 {
		return commitmentError("%s is required", name)
	}
	if _, ok := names[value]; !ok {
		return commitmentError("%s must be a known FOCUS value, got %d", name, value)
	}
	return nil
}

func validateDiscount(c *pbc.ContractCommitment) error {
	set := c.ContractCommitmentDiscountPercentage != nil
	percentage := c.GetContractCommitmentDiscountPercentage()
	if set && (math.IsNaN(percentage) || percentage < 0 || percentage > 1) {
		return commitmentError("contract_commitment_discount_percentage must be finite and in [0, 1]")
	}
	switch c.GetContractCommitmentBenefitCategory() {
	case pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT:
		if !set {
			return commitmentError(
				"contract_commitment_discount_percentage is required when the benefit category is DISCOUNT")
		}
	case pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_AVAILABILITY:
		if set {
			return commitmentError(
				"contract_commitment_discount_percentage must be null when the benefit category is AVAILABILITY")
		}
	case pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_ENTITLEMENT,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_OTHER,
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_UNSPECIFIED:
		return nil
	}
	return nil
}

func validateCommitmentTimestamps(c *pbc.ContractCommitment) error {
	created := c.GetContractCommitmentCreated()
	updated := c.GetContractCommitmentLastUpdated()
	if created == nil {
		return commitmentError("contract_commitment_created is required")
	}
	if updated == nil {
		return commitmentError("contract_commitment_last_updated is required")
	}
	if updated.AsTime().Before(created.AsTime()) {
		return commitmentError("contract_commitment_last_updated must be >= contract_commitment_created")
	}
	return nil
}

func validateModelInterval(c *pbc.ContractCommitment) error {
	fullPeriod := c.GetContractCommitmentFulfillmentInterval() ==
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_FULL_PERIOD
	continuous := c.GetContractCommitmentModel() ==
		pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_CONTINUOUS
	if fullPeriod && continuous {
		return commitmentError(
			"contract_commitment_model must be DISCONTINUOUS when the fulfillment interval is FULL_PERIOD")
	}
	return nil
}

func validatePaymentTerms(c *pbc.ContractCommitment) error {
	model := c.GetContractCommitmentPaymentModel()
	if model == pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_ALL_UPFRONT &&
		c.GetContractCommitmentPaymentInterval() !=
			pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_ONE_TIME {
		return commitmentError(
			"contract_commitment_payment_interval must be ONE_TIME when the payment model is ALL_UPFRONT")
	}
	set := c.ContractCommitmentPaymentUpfrontPercentage != nil
	percentage := c.GetContractCommitmentPaymentUpfrontPercentage()
	switch model {
	case pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_NO_UPFRONT:
		if !set || percentage != 0 {
			return commitmentError(
				"contract_commitment_payment_upfront_percentage must be 0 when the payment model is NO_UPFRONT")
		}
	case pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_ALL_UPFRONT:
		if !set || percentage != 1 {
			return commitmentError(
				"contract_commitment_payment_upfront_percentage must be 1 when the payment model is ALL_UPFRONT")
		}
	case pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_PARTIAL_UPFRONT:
		if !set || math.IsNaN(percentage) || percentage <= 0 || percentage >= 1 {
			return commitmentError(
				"contract_commitment_payment_upfront_percentage must be strictly between 0 and 1 " +
					"when the payment model is PARTIAL_UPFRONT")
		}
	case pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_UNSPECIFIED:
		return nil
	}
	return nil
}

func validatePricingCurrency(c *pbc.ContractCommitment) error {
	code := c.GetPricingCurrency()
	costSet := c.PricingCurrencyContractCommitmentCost != nil
	if code == "" {
		if costSet {
			return commitmentError(
				"pricing_currency is required when pricing_currency_contract_commitment_cost is set")
		}
		return nil
	}
	if !currency.IsValid(code) {
		return commitmentError("pricing_currency must be a valid ISO 4217 currency code, got %q", code)
	}
	spend := c.GetContractCommitmentCategory() ==
		pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND
	if spend && !costSet {
		return commitmentError(
			"pricing_currency_contract_commitment_cost is required when pricing_currency is set for SPEND")
	}
	if !costSet {
		return nil
	}
	return validateAmount("pricing_currency_contract_commitment_cost",
		c.GetPricingCurrencyContractCommitmentCost())
}

// validDurationType reports whether s is "[positive integer] [FOCUS unit]".
// Units are Minute(s), Hour(s), Day(s), Week(s), Month(s), Quarter(s), and Year(s).
func validDurationType(value string) bool {
	number, unit, ok := strings.Cut(value, " ")
	if !ok || number == "" || number[0] == '0' || strings.Contains(unit, " ") {
		return false
	}
	for _, digit := range number {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	switch unit {
	case "Minute", "Minutes", "Hour", "Hours", "Day", "Days", "Week", "Weeks",
		"Month", "Months", "Quarter", "Quarters", "Year", "Years":
		return true
	default:
		return false
	}
}

// isJSONObject reports whether value is one JSON object, ignoring surrounding whitespace.
// It does not allocate. It accepts any object, including an empty one, and rejects
// arrays, scalars, and truncated input.
func isJSONObject(value string) bool {
	start := skipJSONSpace(value, 0)
	if start >= len(value) || value[start] != '{' {
		return false
	}
	end, ok := scanJSONValue(value, start)
	if !ok || value[end-1] != '}' {
		return false
	}
	return skipJSONSpace(value, end) == len(value)
}

func skipJSONSpace(value string, index int) int {
	for index < len(value) && isJSONSpace(value[index]) {
		index++
	}
	return index
}

// scanJSONValue returns the index just past the value that starts at index.
func scanJSONValue(value string, index int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := index; i < len(value); i++ {
		if inString {
			inString, escaped = consumeStringByte(value[i], escaped)
			continue
		}
		switch value[i] {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1, value[i] == '}'
			}
			if depth < 0 {
				return 0, false
			}
		}
	}
	return 0, false
}

func consumeStringByte(char byte, escaped bool) (bool, bool) {
	if escaped {
		return true, false
	}
	switch char {
	case '\\':
		return true, true
	case '"':
		return false, false
	default:
		return true, false
	}
}

func isJSONSpace(char byte) bool {
	return char == ' ' || char == '\n' || char == '\t' || char == '\r'
}
