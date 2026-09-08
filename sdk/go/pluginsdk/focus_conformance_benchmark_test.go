//nolint:testpackage // Internal benchmark tests require access to unexported validateCostValues function
package pluginsdk

import (
	"math"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// createBenchmarkRecord creates a valid FocusCostRecord for benchmarking validateCostValues.
// All cost fields are set to valid finite values.
func createBenchmarkRecord() *pbc.FocusCostRecord {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	return &pbc.FocusCostRecord{
		ProviderName:        "AWS", //nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		BillingAccountId:    "123456789012",
		BillingAccountName:  "Production Account",
		BillingCurrency:     "USD",
		BillingPeriodStart:  timestamppb.New(billingStart),
		BillingPeriodEnd:    timestamppb.New(billingEnd),
		ChargePeriodStart:   timestamppb.New(now.Add(-24 * time.Hour)),
		ChargePeriodEnd:     timestamppb.New(now),
		ChargeCategory:      pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		ServiceCategory:     pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
		ServiceName:         "Amazon EC2",
		BilledCost:          100.50,
		EffectiveCost:       95.25,
		ListCost:            110.00,
		ContractedCost:      100.00,
		ContractedUnitPrice: 0.10,
		ListUnitPrice:       0.12,
		PricingQuantity:     10.0,
		ConsumedQuantity:    10.0,
		ConsumedUnit:        "Hours",
	}
}

// BenchmarkValidateCostValues_ValidRecord benchmarks validateCostValues with all valid cost fields.
// This is the typical happy path where all values are finite floats.
func BenchmarkValidateCostValues_ValidRecord(b *testing.B) {
	record := createBenchmarkRecord()
	opts := ValidationOptions{Mode: ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_AllFields benchmarks validateCostValues with all 8 cost fields populated.
// Verifies that validation scales linearly with the number of fields checked.
func BenchmarkValidateCostValues_AllFields(b *testing.B) {
	record := createBenchmarkRecord()
	// Ensure all 8 cost fields have values
	record.BilledCost = 100.50
	record.EffectiveCost = 95.25
	record.ListCost = 110.00
	record.ContractedCost = 100.00
	record.ContractedUnitPrice = 0.10
	record.ListUnitPrice = 0.12
	record.PricingQuantity = 10.0
	record.ConsumedQuantity = 10.0

	opts := ValidationOptions{Mode: ValidationModeAggregate}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_NaNDetection benchmarks the NaN detection path.
// This tests the error path when validateCostValues encounters a NaN value.
func BenchmarkValidateCostValues_NaNDetection(b *testing.B) {
	record := createBenchmarkRecord()
	record.BilledCost = math.NaN()
	opts := ValidationOptions{Mode: ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_InfDetection benchmarks the Inf detection path.
// This tests the error path when validateCostValues encounters a +Inf value.
func BenchmarkValidateCostValues_InfDetection(b *testing.B) {
	record := createBenchmarkRecord()
	record.BilledCost = math.Inf(1)
	opts := ValidationOptions{Mode: ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_NegativeInfDetection benchmarks -Inf detection.
// This tests the error path when validateCostValues encounters a -Inf value.
func BenchmarkValidateCostValues_NegativeInfDetection(b *testing.B) {
	record := createBenchmarkRecord()
	record.EffectiveCost = math.Inf(-1)
	opts := ValidationOptions{Mode: ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_AggregateMultipleErrors benchmarks aggregate mode with multiple errors.
// This tests performance when multiple cost fields have invalid values and all errors are collected.
func BenchmarkValidateCostValues_AggregateMultipleErrors(b *testing.B) {
	record := createBenchmarkRecord()
	record.BilledCost = math.NaN()
	record.EffectiveCost = math.Inf(1)
	record.ListCost = math.Inf(-1)
	opts := ValidationOptions{Mode: ValidationModeAggregate}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkValidateCostValues_ZeroValues benchmarks validation with zero cost values.
// Zero is a valid cost value (not Inf/NaN), should pass validation.
func BenchmarkValidateCostValues_ZeroValues(b *testing.B) {
	record := createBenchmarkRecord()
	record.BilledCost = 0.0
	record.EffectiveCost = 0.0
	record.ListCost = 0.0
	record.ContractedCost = 0.0
	opts := ValidationOptions{Mode: ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateCostValues(record, opts)
	}
}

// BenchmarkCheckCostValue benchmarks the individual checkCostValue function.
// This isolates the math.IsInf and math.IsNaN checks.
func BenchmarkCheckCostValue_ValidValue(b *testing.B) {
	val := 100.50
	name := "billed_cost"

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = checkCostValue(val, name)
	}
}

// BenchmarkCheckCostValue_NaN benchmarks checkCostValue with a NaN value.
func BenchmarkCheckCostValue_NaN(b *testing.B) {
	val := math.NaN()
	name := "billed_cost"

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = checkCostValue(val, name)
	}
}

// BenchmarkCheckCostValue_Inf benchmarks checkCostValue with an Inf value.
func BenchmarkCheckCostValue_Inf(b *testing.B) {
	val := math.Inf(1)
	name := "billed_cost"

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = checkCostValue(val, name)
	}
}

// =============================================================================
// Prediction Interval Validation Benchmarks
// =============================================================================

// BenchmarkValidateConfidenceLevel_Valid benchmarks confidence level validation for valid values.
func BenchmarkValidateConfidenceLevel_Valid(b *testing.B) {
	confidence := 0.95

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateConfidenceLevel(&confidence)
	}
}

// BenchmarkValidateConfidenceLevel_Nil benchmarks confidence level validation when not set.
func BenchmarkValidateConfidenceLevel_Nil(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateConfidenceLevel(nil)
	}
}

// BenchmarkValidateConfidenceLevel_Invalid benchmarks confidence level validation for invalid values.
func BenchmarkValidateConfidenceLevel_Invalid(b *testing.B) {
	confidence := 1.5 // Out of range

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateConfidenceLevel(&confidence)
	}
}

// BenchmarkValidateConfidenceLevel_NaN benchmarks NaN detection in confidence validation.
func BenchmarkValidateConfidenceLevel_NaN(b *testing.B) {
	confidence := math.NaN()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validateConfidenceLevel(&confidence)
	}
}

// BenchmarkValidatePredictionInterval_Valid benchmarks prediction interval validation for valid inputs.
func BenchmarkValidatePredictionInterval_Valid(b *testing.B) {
	lower := 30.0
	upper := 50.0
	costPerMonth := 40.0

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validatePredictionInterval(&lower, &upper, costPerMonth)
	}
}

// BenchmarkValidatePredictionInterval_Nil benchmarks prediction interval validation when bounds not set.
func BenchmarkValidatePredictionInterval_Nil(b *testing.B) {
	costPerMonth := 40.0

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validatePredictionInterval(nil, nil, costPerMonth)
	}
}

// BenchmarkValidatePredictionInterval_ZeroWidth benchmarks zero-width interval validation.
func BenchmarkValidatePredictionInterval_ZeroWidth(b *testing.B) {
	lower := 40.0
	upper := 40.0
	costPerMonth := 40.0 // Must match for zero-width

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validatePredictionInterval(&lower, &upper, costPerMonth)
	}
}

// BenchmarkValidatePredictionInterval_Invalid benchmarks prediction interval validation for invalid inputs.
func BenchmarkValidatePredictionInterval_Invalid(b *testing.B) {
	lower := 50.0
	upper := 60.0
	costPerMonth := 70.0 // Cost outside interval

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validatePredictionInterval(&lower, &upper, costPerMonth)
	}
}

// BenchmarkValidatePredictionInterval_NaN benchmarks NaN detection in interval validation.
func BenchmarkValidatePredictionInterval_NaN(b *testing.B) {
	lower := math.NaN()
	upper := 50.0
	costPerMonth := 40.0

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = validatePredictionInterval(&lower, &upper, costPerMonth)
	}
}
