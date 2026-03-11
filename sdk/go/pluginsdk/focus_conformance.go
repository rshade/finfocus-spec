package pluginsdk

import (
	"errors"
	"fmt"
	"math"

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// contractedCostTolerance defines the relative tolerance for ContractedCost validation.
//
// The tolerance is set to 0.0001 (1 basis point, or 0.01%) to account for
// IEEE 754 floating-point precision limitations in cost calculations.
//
// Why 1 basis point?
//   - Floating-point arithmetic can introduce small errors (e.g., 0.1 + 0.2 ≠ 0.3)
//   - 1 basis point (0.01%) is the standard financial tolerance for rounding
//   - Example: $1,000,000 contracted cost allows $100 variance (0.01%)
//
// Usage in validation:
//
//	expected := contractedUnitPrice * pricingQuantity
//	diff := abs(contractedCost - expected)
//	valid := diff <= max(abs(contractedCost), abs(expected)) * contractedCostTolerance
//
// Reference: FOCUS 1.2 Section 3.20 (ContractedCost).
const contractedCostTolerance = 0.0001

// Sentinel errors for contextual FinOps validation rules.
// These are package-level variables to enable zero-allocation validation on the error path.
var (
	// ErrEffectiveCostExceedsBilledCost indicates EffectiveCost > BilledCost violation.
	ErrEffectiveCostExceedsBilledCost = errors.New("effective_cost must not exceed billed_cost")

	// ErrListCostLessThanEffectiveCost indicates ListCost < EffectiveCost violation.
	ErrListCostLessThanEffectiveCost = errors.New("list_cost must be >= effective_cost")

	// ErrCommitmentStatusMissing indicates CommitmentDiscountStatus is required.
	ErrCommitmentStatusMissing = errors.New(
		"commitment_discount_status required when commitment_discount_id set for usage charges",
	)

	// ErrCommitmentIDMissingForStatus indicates CommitmentDiscountId is required when status is set.
	ErrCommitmentIDMissingForStatus = errors.New(
		"commitment_discount_id required when commitment_discount_status is set",
	)

	// ErrCapacityReservationStatusMissing indicates CapacityReservationStatus is required.
	ErrCapacityReservationStatusMissing = errors.New(
		"capacity_reservation_status required when capacity_reservation_id set for usage charges",
	)

	// ErrCapacityReservationIDMissing indicates CapacityReservationId is required when status is set.
	ErrCapacityReservationIDMissing = errors.New(
		"capacity_reservation_id required when capacity_reservation_status is set",
	)

	// ErrPricingUnitMissing indicates PricingUnit is required when PricingQuantity > 0.
	ErrPricingUnitMissing = errors.New("pricing_unit required when pricing_quantity > 0")
)

// ValidateFocusRecord checks if a record complies with FOCUS 1.2 mandatory fields and business rules.
// This is a convenience wrapper that uses fail-fast validation mode.
// For aggregate mode or other options, use ValidateFocusRecordWithOptions.
// Reference: https://focus.finops.org
func ValidateFocusRecord(r *pbc.FocusCostRecord) error {
	errs := ValidateFocusRecordWithOptions(r, ValidationOptions{Mode: ValidationModeFailFast})
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// ValidateFocusRecordWithOptions validates a FocusCostRecord according to FOCUS 1.2/1.3
// and contextual FinOps rules using the provided ValidationOptions.
// In FailFast mode, validation stops on the first error. In Aggregate mode, all errors
// are collected and returned. Returns an empty slice when the record is valid.
// If r is nil, returns a single ValidationError with FieldName "record".
func ValidateFocusRecordWithOptions(r *pbc.FocusCostRecord, opts ValidationOptions) []error {
	var errs []error

	if r == nil {
		return []error{NewValidationError("record", "must not be nil", "nil", "non-nil record")}
	}

	// Validate cost values (check for Inf/NaN).
	costErrs := validateCostValues(r, opts)
	if len(costErrs) > 0 {
		if opts.Mode == ValidationModeFailFast {
			return []error{costErrs[0]}
		}
		errs = append(errs, costErrs...)
	}

	// Validate mandatory fields (FOCUS 1.2).
	if err := validateMandatoryFields(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Validate currency codes (ISO 4217).
	if err := validateCurrencyFields(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Validate business rules including contextual FinOps validation.
	businessErrs := validateBusinessRulesWithOptions(r, opts)
	if len(businessErrs) > 0 {
		if opts.Mode == ValidationModeFailFast {
			return []error{businessErrs[0]}
		}
		errs = append(errs, businessErrs...)
	}

	return errs
}

// validateCostValues checks the numeric cost fields on the provided FocusCostRecord for invalid floating-point values.
// It verifies billed_cost, effective_cost, list_cost, contracted_cost, contracted_unit_price, list_unit_price, pricing_quantity, and consumed_quantity.
// If any of these fields is +Inf, -Inf, or NaN, validateCostValues returns an error naming the offending field; otherwise it returns nil.
func validateCostValues(r *pbc.FocusCostRecord, opts ValidationOptions) []error {
	var errs []error

	// Build slice of name/value pairs directly to avoid heap allocations from method value capture.
	// Calling the getters directly and storing their float64 results is more efficient.
	costFields := []struct {
		name string
		val  float64
	}{
		{"billed_cost", r.GetBilledCost()},
		{"effective_cost", r.GetEffectiveCost()},
		{"list_cost", r.GetListCost()},
		{"contracted_cost", r.GetContractedCost()},
		{"contracted_unit_price", r.GetContractedUnitPrice()},
		{"list_unit_price", r.GetListUnitPrice()},
		{"pricing_quantity", r.GetPricingQuantity()},
		{"consumed_quantity", r.GetConsumedQuantity()},
	}

	for _, f := range costFields {
		if err := checkCostValue(f.val, f.name); err != nil {
			if opts.Mode == ValidationModeFailFast {
				return []error{err}
			}
			errs = append(errs, err)
		}
	}

	return errs
}

// checkCostValue validates a single cost field for Inf/NaN values.
func checkCostValue(val float64, name string) error {
	if math.IsInf(val, 0) {
		return NewValidationError(name, "must be finite", "infinity", "finite number")
	}
	if math.IsNaN(val) {
		return NewValidationError(name, "must be a number", "NaN", "finite number")
	}
	return nil
}

// validateMandatoryFields verifies the 14 mandatory FOCUS 1.2 fields are present in r.
// It returns an error describing the first missing or unspecified required field.
// This validation requires provider_name for FOCUS 1.2 (provider_name is deprecated in FOCUS 1.3).
// BilledCost and ContractedCost are considered mandatory but may be zero or negative; billing_account_name
// validation is intentionally relaxed to accommodate providers that omit account names.
func validateMandatoryFields(r *pbc.FocusCostRecord) error {
	// Identity fields (FOCUS 1.2 Section 2.1).
	// Note: provider_name is deprecated in FOCUS 1.3 but still required for FOCUS 1.2 conformance.
	//nolint:staticcheck // SA1019: Checking deprecated field for FOCUS 1.2 backward compatibility
	if r.GetProviderName() == "" {
		return NewValidationError("provider_name", "required", "", "non-empty string")
	}
	if r.GetBillingAccountId() == "" {
		return NewValidationError("billing_account_id", "required", "", "non-empty string")
	}

	// Billing period (FOCUS 1.2 Section 2.2).
	if r.GetBillingPeriodStart() == nil {
		return NewValidationError("billing_period.start", "required", "", "non-nil timestamp")
	}
	if r.GetBillingPeriodEnd() == nil {
		return NewValidationError("billing_period.end", "required", "", "non-nil timestamp")
	}
	if r.GetBillingCurrency() == "" {
		return NewValidationError("billing_currency", "required", "", "non-empty string")
	}

	// Charge period (FOCUS 1.2 Section 2.3).
	if r.GetChargePeriodStart() == nil {
		return NewValidationError("charge_period.start", "required", "", "non-nil timestamp")
	}
	if r.GetChargePeriodEnd() == nil {
		return NewValidationError("charge_period.end", "required", "", "non-nil timestamp")
	}

	// Charge details (FOCUS 1.2 Section 2.4).
	if r.GetChargeCategory() == pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_UNSPECIFIED {
		return NewValidationError("charge_category", "required", "UNSPECIFIED", "valid charge category")
	}
	if r.GetChargeClass() == pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_UNSPECIFIED {
		return NewValidationError("charge_class", "required", "UNSPECIFIED", "valid charge class")
	}
	if r.GetChargeDescription() == "" {
		return NewValidationError("charge_description", "required", "", "non-empty string")
	}

	// Service details (FOCUS 1.2 Section 2.6).
	if r.GetServiceCategory() == pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_UNSPECIFIED {
		return NewValidationError("service_category", "required", "UNSPECIFIED", "valid service category")
	}
	if r.GetServiceName() == "" {
		return NewValidationError("service_name", "required", "", "non-empty string")
	}

	// Note: BilledCost and ContractedCost are mandatory but can be zero or negative
	// (e.g., credits, refunds per FOCUS 1.2 spec).
	//
	// Note: BillingAccountName is mandatory per FOCUS 1.2 but validation is intentionally
	// relaxed to accommodate cloud providers that may not include account names in all
	// billing data exports. Plugins should populate this field when available.
	return nil
}

// validateCurrencyFields validates ISO 4217 currency codes.
func validateCurrencyFields(r *pbc.FocusCostRecord) error {
	if err := validateCurrency(r.GetBillingCurrency(), "billing_currency"); err != nil {
		return err
	}

	// PricingCurrency is optional, but if present, must be valid ISO 4217.
	if r.GetPricingCurrency() != "" {
		if err := validateCurrency(r.GetPricingCurrency(), "pricing_currency"); err != nil {
			return err
		}
	}
	return nil
}

// validateBusinessRulesWithOptions validates FOCUS 1.2/1.3 business rules with options support.
func validateBusinessRulesWithOptions(r *pbc.FocusCostRecord, opts ValidationOptions) []error {
	var errs []error

	// Rule: Usage records must have positive consumed quantity.
	// FOCUS 1.2: If ChargeCategory=Usage, ConsumedQuantity must be > 0.
	if r.GetChargeCategory() == pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE {
		if r.GetConsumedQuantity() <= 0 {
			err := NewValidationError(
				"consumed_quantity", "must be positive for usage charges",
				fmt.Sprintf("%g", r.GetConsumedQuantity()), "> 0",
			)
			if opts.Mode == ValidationModeFailFast {
				return []error{err}
			}
			errs = append(errs, err)
		}
	}

	// Rule: ContractedCost must equal ContractedUnitPrice × PricingQuantity.
	// FOCUS 1.2 Section 3.20: When ContractedUnitPrice and PricingQuantity are present
	// and ChargeClass is not "Correction", this relationship must hold.
	if err := validateContractedCostRule(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Validate FOCUS 1.3 specific rules (split cost allocation, etc.)
	if err := validateFocus13Rules(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Validate contextual FinOps rules (cost hierarchy, commitment discounts, etc.)
	contextualErrs := validateContextualFinOpsRules(r, opts)
	if len(contextualErrs) > 0 {
		if opts.Mode == ValidationModeFailFast {
			return []error{contextualErrs[0]}
		}
		errs = append(errs, contextualErrs...)
	}

	return errs
}

// validateContextualFinOpsRules validates contextual business logic for FinOps cost records.
// This includes cost hierarchy validation, commitment discount consistency,
// capacity reservation consistency, and pricing model validation.
func validateContextualFinOpsRules(r *pbc.FocusCostRecord, opts ValidationOptions) []error {
	var errs []error

	// Cost hierarchy validation (FR-001, FR-002)
	if err := validateCostHierarchy(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Commitment discount consistency (FR-003, FR-004)
	commitmentErrs := validateCommitmentDiscountConsistency(r, opts)
	if len(commitmentErrs) > 0 {
		if opts.Mode == ValidationModeFailFast {
			return []error{commitmentErrs[0]}
		}
		errs = append(errs, commitmentErrs...)
	}

	// Pricing consistency (FR-006)
	if err := validatePricingConsistency(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	// Capacity reservation consistency (FR-005)
	if err := validateCapacityReservationConsistency(r); err != nil {
		if opts.Mode == ValidationModeFailFast {
			return []error{err}
		}
		errs = append(errs, err)
	}

	return errs
}

// =============================================================================
// FOCUS 1.3 Validation Rules
// =============================================================================

// validateFocus13Rules validates all FOCUS 1.3 specific business rules.
// This includes split cost allocation constraints and provider field consistency.
func validateFocus13Rules(r *pbc.FocusCostRecord) error {
	// Rule: AllocatedMethodId requires AllocatedResourceId.
	// FOCUS 1.3: If allocated_method_id is set, allocated_resource_id MUST also be set.
	// This ensures allocation methodology is always tied to a target resource.
	if err := validateAllocationRule(r); err != nil {
		return err
	}

	// Future FOCUS 1.3 rules can be added here:
	// - validateProviderConsistency(r) - ensure service/host provider logic
	// - validateContractAppliedReference(r) - validate commitment references if needed

	return nil
}

// validateAllocationRule verifies FOCUS 1.3 allocation field dependencies.
// If allocated_method_id is set, allocated_resource_id MUST also be set.
func validateAllocationRule(r *pbc.FocusCostRecord) error {
	if r.GetAllocatedMethodId() != "" && r.GetAllocatedResourceId() == "" {
		return NewValidationError(
			"allocated_resource_id", "required when allocated_method_id set",
			"", "non-empty string",
		)
	}
	return nil
}

// validateCurrency checks if a currency code is a valid ISO 4217 code.
func validateCurrency(code string, fieldName string) error {
	if !currency.IsValid(code) {
		return NewValidationError(
			fieldName, "must be valid ISO 4217 currency code",
			code, "valid ISO 4217 code",
		)
	}
	return nil
}

// validateContractedCostRule verifies ContractedCost = ContractedUnitPrice × PricingQuantity.
// This rule applies when both ContractedUnitPrice and PricingQuantity are present (non-zero)
// and ChargeClass is not Correction.
func validateContractedCostRule(r *pbc.FocusCostRecord) error {
	// Skip validation for correction charges.
	if r.GetChargeClass() == pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION {
		return nil
	}

	contractedUnitPrice := r.GetContractedUnitPrice()
	pricingQuantity := r.GetPricingQuantity()

	// Only validate when both values are present (non-zero).
	if contractedUnitPrice == 0 || pricingQuantity == 0 {
		return nil
	}

	contractedCost := r.GetContractedCost()
	expectedCost := contractedUnitPrice * pricingQuantity

	// Use relative tolerance for floating-point comparison.
	if !floatEquals(contractedCost, expectedCost, contractedCostTolerance) {
		return NewValidationError(
			"contracted_cost", "must equal unit_price x quantity",
			fmt.Sprintf("%g", contractedCost),
			fmt.Sprintf("%g (unit_price=%g × quantity=%g)", expectedCost, contractedUnitPrice, pricingQuantity),
		)
	}

	return nil
}

// floatEquals compares two floats with relative tolerance.
func floatEquals(a, b, tolerance float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	largest := math.Max(math.Abs(a), math.Abs(b))
	return diff <= largest*tolerance
}

// =============================================================================
// Contextual FinOps Validation Rules
// =============================================================================

// validateCostHierarchy validates the cost relationship: ListCost >= BilledCost >= EffectiveCost.
// FR-001: EffectiveCost must not exceed BilledCost (when both positive, non-correction).
// FR-002: ListCost must be >= EffectiveCost (when both positive, non-correction).
//
// Exemptions:
// - ChargeClass CORRECTION is exempt from all cost hierarchy rules.
// - Negative costs (credits/refunds) are exempt from hierarchy validation.
// - Zero costs (free tier) pass validation without error.
func validateCostHierarchy(r *pbc.FocusCostRecord) error {
	// Skip validation for correction charges (per FOCUS spec).
	if r.GetChargeClass() == pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION {
		return nil
	}

	billedCost := r.GetBilledCost()
	effectiveCost := r.GetEffectiveCost()
	listCost := r.GetListCost()

	// FR-001: EffectiveCost must not exceed BilledCost.
	// Only validate when both are positive (excludes credits/refunds and free tier).
	if billedCost > 0 && effectiveCost > 0 {
		if effectiveCost > billedCost && !floatEquals(effectiveCost, billedCost, contractedCostTolerance) {
			return NewValidationErrorWithCause(
				"effective_cost", "must not exceed billed_cost",
				fmt.Sprintf("%g", effectiveCost), fmt.Sprintf("<= %g", billedCost),
				ErrEffectiveCostExceedsBilledCost,
			)
		}
	}

	// FR-002: ListCost must be >= EffectiveCost.
	// Only validate when both are positive.
	if listCost > 0 && effectiveCost > 0 {
		if listCost < effectiveCost && !floatEquals(listCost, effectiveCost, contractedCostTolerance) {
			return NewValidationErrorWithCause(
				"list_cost", "must be >= effective_cost",
				fmt.Sprintf("%g", listCost), fmt.Sprintf(">= %g", effectiveCost),
				ErrListCostLessThanEffectiveCost,
			)
		}
	}

	return nil
}

// validateCommitmentDiscountConsistency validates commitment discount field dependencies.
// FR-003: CommitmentDiscountStatus required when CommitmentDiscountId set + ChargeCategory=Usage.
// FR-004: CommitmentDiscountId required when CommitmentDiscountStatus is set (non-UNSPECIFIED).
func validateCommitmentDiscountConsistency(
	r *pbc.FocusCostRecord,
	opts ValidationOptions,
) []error {
	var errs []error

	commitmentID := r.GetCommitmentDiscountId()
	commitmentStatus := r.GetCommitmentDiscountStatus()
	isUsageCharge := r.GetChargeCategory() == pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE

	// FR-003: If CommitmentDiscountId is set AND ChargeCategory is Usage,
	// CommitmentDiscountStatus must be set (not UNSPECIFIED).
	if commitmentID != "" && isUsageCharge {
		if commitmentStatus == pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED {
			err := NewValidationErrorWithCause(
				"commitment_discount_status", "required when commitment_discount_id set for usage charges",
				"UNSPECIFIED", "USED or UNUSED",
				ErrCommitmentStatusMissing,
			)
			if opts.Mode == ValidationModeFailFast {
				return []error{err}
			}
			errs = append(errs, err)
		}
	}

	// FR-004: If CommitmentDiscountStatus is set (not UNSPECIFIED),
	// CommitmentDiscountId must also be set.
	if commitmentStatus != pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED {
		if commitmentID == "" {
			err := NewValidationErrorWithCause(
				"commitment_discount_id", "required when commitment_discount_status is set",
				"", "non-empty string",
				ErrCommitmentIDMissingForStatus,
			)
			if opts.Mode == ValidationModeFailFast {
				return []error{err}
			}
			errs = append(errs, err)
		}
	}

	return errs
}

// validatePricingConsistency validates pricing field dependencies.
// FR-006: PricingUnit is required when PricingQuantity > 0.
func validatePricingConsistency(r *pbc.FocusCostRecord) error {
	pricingQuantity := r.GetPricingQuantity()
	pricingUnit := r.GetPricingUnit()

	// If PricingQuantity > 0, PricingUnit must be specified.
	if pricingQuantity > 0 && pricingUnit == "" {
		return NewValidationErrorWithCause(
			"pricing_unit", "required when pricing_quantity > 0",
			"", "non-empty string",
			ErrPricingUnitMissing,
		)
	}

	return nil
}

// validateCapacityReservationConsistency validates capacity reservation field dependencies.
// FR-005: CapacityReservationStatus required when CapacityReservationId set + ChargeCategory=Usage.
// FR-005 (bidirectional): CapacityReservationId required when CapacityReservationStatus is set.
func validateCapacityReservationConsistency(r *pbc.FocusCostRecord) error {
	capacityID := r.GetCapacityReservationId()
	capacityStatus := r.GetCapacityReservationStatus()
	isUsageCharge := r.GetChargeCategory() == pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE

	// If CapacityReservationId is set AND ChargeCategory is Usage,
	// CapacityReservationStatus must be set (not UNSPECIFIED).
	if capacityID != "" && isUsageCharge {
		if capacityStatus == pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED {
			return NewValidationErrorWithCause(
				"capacity_reservation_status", "required when capacity_reservation_id set for usage charges",
				"UNSPECIFIED", "USED or UNUSED",
				ErrCapacityReservationStatusMissing,
			)
		}
	}

	// FR-005 (bidirectional): If CapacityReservationStatus is set (not UNSPECIFIED),
	// CapacityReservationId must also be set.
	if capacityStatus != pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED {
		if capacityID == "" {
			return NewValidationErrorWithCause(
				"capacity_reservation_id", "required when capacity_reservation_status is set",
				"", "non-empty string",
				ErrCapacityReservationIDMissing,
			)
		}
	}

	return nil
}
