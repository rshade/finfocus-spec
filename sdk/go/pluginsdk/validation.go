// Package pluginsdk provides a development SDK for FinFocus plugins.
//
// # Request Validation
//
// This package provides lightweight request validation for plugin implementations.
// For comprehensive contract testing with detailed error reporting, see sdk/go/testing/contract.go.
//
// Key differences:
//   - pluginsdk: Simple errors, minimal dependencies, optimized for plugin defense-in-depth
//   - testing/contract: Rich error context, comprehensive rules, designed for test suites
//
// # Usage
//
// Validate requests at plugin entry points:
//
//	func (s *MyPlugin) GetProjectedCost(ctx context.Context, req *pb.GetProjectedCostRequest) (*pb.GetProjectedCostResponse, error) {
//	    if err := pluginsdk.ValidateProjectedCostRequest(req); err != nil {
//	        return nil, status.Error(codes.InvalidArgument, err.Error())
//	    }
//	    // Process valid request...
//	}
//
// # Performance
//
// Validation is optimized for performance:
//   - Zero allocations on the happy path (valid request returns nil)
//   - Error paths allocate only for the error message
//   - Target: <100ns execution time for valid requests
package pluginsdk

import (
	"errors"
	"fmt"
	"math"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Validation error messages for GetProjectedCostRequest.
var (
	ErrProjectedCostRequestNil        = errors.New("request is required")
	ErrProjectedCostResourceNil       = errors.New("resource is required")
	ErrProjectedCostProviderEmpty     = errors.New("resource.provider is required")
	ErrProjectedCostResourceTypeEmpty = errors.New("resource.resource_type is required")
	ErrProjectedCostSkuEmpty          = errors.New("resource.sku is required (use mapping helpers)")
	ErrProjectedCostRegionEmpty       = errors.New("resource.region is required (use mapping helpers)")
	ErrUtilizationOutOfRange          = errors.New("utilization_percentage must be between 0.0 and 1.0")
	ErrMetricKindInvalid              = errors.New("invalid metric kind")
)

// Validation error messages for SupportsResponse.
var (
	ErrSupportsResponseNil = errors.New("response is required")
)

// Validation error messages for GetActualCostRequest.
var (
	ErrActualCostRequestNil       = errors.New("request is required")
	ErrActualCostResourceIDEmpty  = errors.New("resource_id is required")
	ErrActualCostStartTimeNil     = errors.New("start_time is required")
	ErrActualCostEndTimeNil       = errors.New("end_time is required")
	ErrActualCostTimeRangeInvalid = errors.New(
		"end_time must be strictly after start_time (equal timestamps not allowed)",
	)
)

// Validation error messages for EstimateCostResponse and GetProjectedCostResponse.
var (
	ErrEstimateCostResponseNil      = errors.New("response is required")
	ErrGetProjectedCostResponseNil  = errors.New("response is required")
	ErrSpotRiskScoreOutOfRange      = errors.New("spot_interruption_risk_score must be between 0.0 and 1.0")
	ErrSpotRiskScoreNaN             = errors.New("spot_interruption_risk_score cannot be NaN or Inf")
	ErrSpotRiskScoreInvalidCategory = errors.New(
		"spot_interruption_risk_score should only be non-zero when pricing_category is DYNAMIC",
	)
	ErrPredictionIntervalLowerMissing = errors.New(
		"GetProjectedCostResponse: prediction_interval_upper is set but prediction_interval_lower is missing",
	)
	ErrPredictionIntervalUpperMissing = errors.New(
		"GetProjectedCostResponse: prediction_interval_lower is set but prediction_interval_upper is missing",
	)
)

// spotRiskEpsilon is used for float comparison to handle floating-point representation errors.
// This small value (1e-9) is chosen to be well below the precision typically meaningful
// for risk scores while being large enough to catch representation errors.
const spotRiskEpsilon = 1e-9

// ValidateProjectedCostRequest validates a GetProjectedCostRequest for required fields.
// This function is designed for use in both:
//   - Core: Pre-flight validation before sending requests to plugins
//   - Plugins: Defense-in-depth validation upon receiving requests
//
// Validation order (fail-fast, structural before field values):
//  1. Request nil check
//  2. Resource nil check
//  3. Provider empty check
//  4. ResourceType empty check
//  5. SKU empty check (with mapping helper guidance)
//  6. Region empty check (with mapping helper guidance)
//  7. Global utilization range check (if non-zero)
//  8. Resource-level utilization range check (if provided)
//
// Performance: Zero allocations on the happy path (valid request returns nil).
// Error paths allocate for the error message.
//
// Returns nil if the request is valid, or an error describing the first validation failure.
func ValidateProjectedCostRequest(req *pbc.GetProjectedCostRequest) error {
	if req == nil {
		return ErrProjectedCostRequestNil
	}

	resource := req.GetResource()
	if resource == nil {
		return ErrProjectedCostResourceNil
	}

	if len(resource.GetProvider()) == 0 {
		return ErrProjectedCostProviderEmpty
	}

	if len(resource.GetResourceType()) == 0 {
		return ErrProjectedCostResourceTypeEmpty
	}

	if len(resource.GetSku()) == 0 {
		return ErrProjectedCostSkuEmpty
	}

	if len(resource.GetRegion()) == 0 {
		return ErrProjectedCostRegionEmpty
	}

	// Validate utilization values using centralized helper
	// Global utilization: non-zero values must be valid (protobuf3 default is 0.0)
	if u := req.GetUtilizationPercentage(); u != 0 && !IsUtilizationValid(u) {
		return ErrUtilizationOutOfRange
	}

	// Resource-level utilization: if explicitly set, must be valid
	if resource.UtilizationPercentage != nil && !IsUtilizationValid(resource.GetUtilizationPercentage()) {
		return ErrUtilizationOutOfRange
	}

	return nil
}

// ValidateSupportsResponse validates a SupportsResponse for correctness.
//
// Validation order:
//  1. Response nil check
//  2. Supported metrics validity check
//
// Returns nil if the response is valid, or an error describing the failure.
func ValidateSupportsResponse(res *pbc.SupportsResponse) error {
	if res == nil {
		return ErrSupportsResponseNil
	}

	for _, kind := range res.GetSupportedMetrics() {
		if !IsValidMetricKind(kind) {
			return ErrMetricKindInvalid
		}
	}

	return nil
}

// validMetricKinds contains all valid sustainability metric kinds for zero-allocation validation.
// This follows the pattern established in sdk/go/registry for optimized enum validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var validMetricKinds = []pbc.MetricKind{
	pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT,
	pbc.MetricKind_METRIC_KIND_ENERGY_CONSUMPTION,
	pbc.MetricKind_METRIC_KIND_WATER_USAGE,
}

// ValidMetricKinds returns all valid sustainability metric kinds.
// The returned slice is shared and must not be modified.
func ValidMetricKinds() []pbc.MetricKind {
	return validMetricKinds
}

// IsValidMetricKind returns true if the MetricKind is a recognized sustainability metric.
// METRIC_KIND_UNSPECIFIED is not in the valid list and returns false.
//
// Performance: Zero allocations, ~5-12 ns/op for small enum sets.
func IsValidMetricKind(kind pbc.MetricKind) bool {
	for _, valid := range validMetricKinds {
		if kind == valid {
			return true
		}
	}
	return false
}

// ValidateActualCostRequest validates a GetActualCostRequest for required fields.
// This function is designed for use in both:
//   - Core: Pre-flight validation before sending requests to plugins
//   - Plugins: Defense-in-depth validation upon receiving requests
//
// Validation order (fail-fast):
//  1. Request nil check
//  2. ResourceId empty check
//  3. StartTime nil check
//  4. EndTime nil check
//  5. TimeRange validation (EndTime must be after StartTime)
//
// Performance: Zero allocations on the happy path (valid request returns nil).
// Error paths allocate for the error message.
//
// Returns nil if the request is valid, or an error describing the first validation failure.
func ValidateActualCostRequest(req *pbc.GetActualCostRequest) error {
	if req == nil {
		return ErrActualCostRequestNil
	}

	if len(req.GetResourceId()) == 0 {
		return ErrActualCostResourceIDEmpty
	}

	startTime := req.GetStart()
	if startTime == nil {
		return ErrActualCostStartTimeNil
	}

	endTime := req.GetEnd()
	if endTime == nil {
		return ErrActualCostEndTimeNil
	}

	// Compare timestamps: end must be strictly after start
	// Using AsTime() for accurate comparison including nanoseconds
	if !endTime.AsTime().After(startTime.AsTime()) {
		return ErrActualCostTimeRangeInvalid
	}

	return nil
}

// validateSpotRiskScore validates the spot_interruption_risk_score field.
// Returns nil if score is effectively 0.0 (proto3 default) or a valid non-zero value.
//
// Validation checks (fail-fast):
//  1. NaN/Inf check (always performed for safety)
//  2. Fast path: skip further validation if score is effectively zero
//  3. Range check [0.0, 1.0] with epsilon tolerance
//  4. Semantic check: score > 0 requires DYNAMIC pricing category
//
// Float Comparison: Uses epsilon tolerance (1e-9) to handle floating-point representation
// errors that may occur from arithmetic operations.
//
// Performance: Zero allocations on the happy path.
func validateSpotRiskScore(score float64, category pbc.FocusPricingCategory) error {
	// Check for invalid float values first (NaN/Inf are more severe)
	// This check must come before epsilon comparison since NaN comparisons behave unexpectedly
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return fmt.Errorf("%w: got %v", ErrSpotRiskScoreNaN, score)
	}

	// Fast path: proto3 default (or effectively zero) is valid
	// Uses epsilon tolerance to handle floating-point representation errors
	if math.Abs(score) < spotRiskEpsilon {
		return nil
	}

	// Check range: epsilon tolerance only on lower bound (handles float errors near 0)
	// Upper bound is strict 1.0 - probability cannot exceed 100%
	if score < -spotRiskEpsilon || score > 1.0 {
		return fmt.Errorf("%w: got %f", ErrSpotRiskScoreOutOfRange, score)
	}

	// Semantic validation: enforce category/risk consistency
	if category != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC {
		return fmt.Errorf("%w: got score=%f with category=%s",
			ErrSpotRiskScoreInvalidCategory, score, category.String())
	}

	return nil
}

// validateConfidenceLevel validates the confidence_level field if set.
// Returns nil if confidence is nil (not set) or a valid value in range (0.0, 1.0].
func validateConfidenceLevel(confidence *float64) error {
	if confidence == nil {
		return nil
	}

	val := *confidence
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return fmt.Errorf("GetProjectedCostResponse: confidence_level is NaN/Inf: %v", val)
	}
	if val <= 0 || val > 1.0 {
		return fmt.Errorf(
			"GetProjectedCostResponse: confidence_level must be in range (0.0, 1.0], got %f",
			val,
		)
	}
	return nil
}

// validatePredictionInterval validates the prediction interval bounds for consistency.
// This helper reduces cognitive complexity of ValidateGetProjectedCostResponse.
//
// Returns nil if:
//   - Both bounds are unset (valid)
//   - Both bounds are set and satisfy all constraints
//
// Validation checks:
//  1. Both bounds must be present or both absent
//  2. Bounds must be finite (not NaN/Inf)
//  3. Lower bound must be non-negative
//  4. lower <= costPerMonth <= upper
//  5. lower <= upper
func validatePredictionInterval(
	lower, upper *float64,
	costPerMonth float64,
) error {
	lowerSet := lower != nil
	upperSet := upper != nil

	// Check that both bounds are either present or absent
	if lowerSet != upperSet {
		if lowerSet {
			return ErrPredictionIntervalUpperMissing
		}
		return ErrPredictionIntervalLowerMissing
	}

	// If neither bound is set, validation passes
	if !lowerSet {
		return nil
	}

	// Both bounds are set - validate them
	lowerVal := *lower
	upperVal := *upper

	// Validate bounds are finite (type safety before any value checks)
	if math.IsNaN(lowerVal) || math.IsInf(lowerVal, 0) {
		return fmt.Errorf("GetProjectedCostResponse: prediction_interval_lower is NaN/Inf: %v", lowerVal)
	}
	if math.IsNaN(upperVal) || math.IsInf(upperVal, 0) {
		return fmt.Errorf("GetProjectedCostResponse: prediction_interval_upper is NaN/Inf: %v", upperVal)
	}

	// Validate structural validity FIRST (lower <= upper)
	// Structural checks before value constraints ensures consistent error messages
	// regardless of which value constraint would also fail
	if lowerVal > upperVal {
		return fmt.Errorf(
			"GetProjectedCostResponse: prediction_interval_lower (%f) > prediction_interval_upper (%f)",
			lowerVal,
			upperVal,
		)
	}

	// Validate lower bound is non-negative (value constraint after structural)
	if lowerVal < 0 {
		return fmt.Errorf("GetProjectedCostResponse: prediction_interval_lower cannot be negative: %f", lowerVal)
	}

	// Zero-width interval (lower == upper) requires cost_per_month to equal the bounds
	// A zero-width interval [x, x] implies zero uncertainty, meaning the point estimate
	// must exactly match the bounds. This provides clearer error messages for edge cases
	// like [42, 42] with cost=50 instead of a generic "upper < cost" message.
	// This check comes after lower <= upper since zero-width requires lower == upper.
	if lowerVal == upperVal && lowerVal != costPerMonth {
		return fmt.Errorf(
			"GetProjectedCostResponse: zero-width prediction interval [%f, %f] "+
				"requires cost_per_month to equal bounds, got %f",
			lowerVal, upperVal, costPerMonth,
		)
	}

	// Validate cost is within the interval bounds
	if lowerVal > costPerMonth {
		return fmt.Errorf("GetProjectedCostResponse: prediction_interval_lower (%f) > cost_per_month (%f)",
			lowerVal, costPerMonth)
	}
	if upperVal < costPerMonth {
		return fmt.Errorf("GetProjectedCostResponse: prediction_interval_upper (%f) < cost_per_month (%f)",
			upperVal, costPerMonth)
	}

	return nil
}

// ValidateEstimateCostResponse validates an EstimateCostResponse for correctness.
// This function is designed for use in plugins before returning responses.
//
// Validation order (fail-fast):
//  1. Response nil check
//  2. Spot risk score validation (structural + semantic)
//
// Semantic rule enforced: spot_interruption_risk_score must only be non-zero
// when pricing_category is FOCUS_PRICING_CATEGORY_DYNAMIC.
//
// Performance: Zero allocations on the happy path (valid response returns nil).
// Error paths allocate for the error message.
//
// Returns nil if the response is valid, or an error describing the first validation failure.
func ValidateEstimateCostResponse(resp *pbc.EstimateCostResponse) error {
	if resp == nil {
		return ErrEstimateCostResponseNil
	}

	// Return sentinel errors directly (they already contain context)
	if err := validateSpotRiskScore(resp.GetSpotInterruptionRiskScore(), resp.GetPricingCategory()); err != nil {
		return err
	}

	return nil
}

// ValidateGetProjectedCostResponse validates a GetProjectedCostResponse for correctness.
// This function is designed for use in plugins before returning responses.
//
// Validation order (fail-fast):
//  1. Response nil check
//  2. CostPerMonth non-negative check
//  3. Prediction interval consistency (if set)
//  4. Confidence level range validation (if set)
//  5. Spot risk score validation (structural + semantic)
//
// Semantic rules enforced:
//   - spot_interruption_risk_score must only be non-zero when pricing_category is FOCUS_PRICING_CATEGORY_DYNAMIC
//   - prediction_interval_lower must be <= cost_per_month
//   - prediction_interval_upper must be >= cost_per_month
//   - prediction_interval_lower must be <= prediction_interval_upper
//   - confidence_level must be in range (0.0, 1.0] if set
//
// Performance: Zero allocations on the happy path (valid response returns nil).
// Error paths allocate for the error message.
//
// Returns nil if the response is valid, or an error describing the first validation failure.
func ValidateGetProjectedCostResponse(resp *pbc.GetProjectedCostResponse) error {
	if resp == nil {
		return ErrGetProjectedCostResponseNil
	}

	// Validate cost_per_month is finite and non-negative
	costPerMonth := resp.GetCostPerMonth()
	if math.IsNaN(costPerMonth) || math.IsInf(costPerMonth, 0) {
		return fmt.Errorf("GetProjectedCostResponse: cost_per_month is NaN/Inf: %v", costPerMonth)
	}
	if costPerMonth < 0 {
		return fmt.Errorf("GetProjectedCostResponse: cost_per_month cannot be negative: %f", costPerMonth)
	}

	// Validate prediction interval using extracted helper (reduces cognitive complexity)
	if err := validatePredictionInterval(
		resp.PredictionIntervalLower,
		resp.PredictionIntervalUpper,
		costPerMonth,
	); err != nil {
		return err
	}

	// Validate confidence level if set
	if err := validateConfidenceLevel(resp.ConfidenceLevel); err != nil {
		return err
	}

	// Return sentinel errors directly (they already contain context)
	if err := validateSpotRiskScore(resp.GetSpotInterruptionRiskScore(), resp.GetPricingCategory()); err != nil {
		return err
	}

	// Validate expires_at timestamp if set
	if ts := resp.GetExpiresAt(); ts != nil {
		if err := ts.CheckValid(); err != nil {
			return fmt.Errorf("GetProjectedCostResponse: expires_at is invalid: %w", err)
		}
	}

	return nil
}

// CheckSpotRiskConsistency checks semantic consistency between pricing_category and spot_interruption_risk_score.
// This is a warning-level validation that plugins can use for self-validation.
//
// Returns a slice of warning messages. Empty slice means no issues.
//
// Semantic rules checked:
//  1. spot_interruption_risk_score > 0 requires pricing_category = DYNAMIC
//  2. pricing_category = DYNAMIC with score = 0 may indicate missing risk data
//
// The second check is advisory - zero risk score with DYNAMIC pricing is valid
// (e.g., for dynamic pricing with no interruption risk), but plugins should
// confirm this is intentional rather than a data gap.
func CheckSpotRiskConsistency(category pbc.FocusPricingCategory, score float64) []string {
	var warnings []string

	// Check: non-zero risk score requires DYNAMIC pricing
	if score > spotRiskEpsilon && category != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC {
		warnings = append(warnings,
			fmt.Sprintf("spot_interruption_risk_score > 0.0 (%.4f) but pricing_category is %s, not DYNAMIC",
				score, category.String()))
	}

	// Check: DYNAMIC pricing with zero risk score may indicate missing data
	// This is advisory - zero risk is valid but unusual for spot/dynamic resources
	if category == pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC &&
		math.Abs(score) < spotRiskEpsilon {
		warnings = append(warnings,
			"pricing_category is DYNAMIC but spot_interruption_risk_score is 0.0 "+
				"(risk data may be unavailable or resource has unusually low interruption risk)")
	}

	return warnings
}
