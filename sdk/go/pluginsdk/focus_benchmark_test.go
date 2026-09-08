package pluginsdk_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func BenchmarkFocusRecordBuilder(b *testing.B) {
	start := time.Now()
	end := start.Add(time.Hour)

	b.ResetTimer()
	for range b.N {
		builder := pluginsdk.NewFocusRecordBuilder()
		builder.WithIdentity("aws", "acc-123", "My Account")
		builder.WithChargePeriod(start, end)
		builder.WithServiceCategory(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE)
		builder.WithChargeDetails(
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		)
		builder.WithFinancials(10.5, 12.0, 10.0, "USD", "inv-001")
		builder.WithUsage(1.0, "Hour")
		builder.WithExtension("Env", "Prod")
		_, _ = builder.Build()
	}
}

// =============================================================================
// Contextual FinOps Validation Benchmarks (Feature 027-finops-validation)
// =============================================================================
//
// These benchmarks verify SC-001: <100ns, 0 allocs on valid records.
// The sentinel error pattern enables zero-allocation validation.

// createValidBenchmarkRecord creates a valid FocusCostRecord for benchmarking.
// This record passes all validation rules with realistic field values.
func createValidBenchmarkRecord() *pbc.FocusCostRecord {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	return &pbc.FocusCostRecord{
		//nolint:staticcheck // SA1019: provider_name is mandatory for FOCUS 1.2 conformance (see validateMandatoryFields)
		ProviderName:       "AWS",
		BillingAccountId:   "123456789012",
		BillingAccountName: "Production Account",
		BillingCurrency:    "USD",
		BillingPeriodStart: timestamppb.New(billingStart),
		BillingPeriodEnd:   timestamppb.New(billingEnd),
		ChargePeriodStart:  timestamppb.New(chargeStart),
		ChargePeriodEnd:    timestamppb.New(chargeEnd),
		ChargeCategory:     pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		ChargeClass:        pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		ChargeDescription:  "EC2 Instance Usage",
		ServiceCategory:    pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
		ServiceName:        "Amazon EC2",
		BilledCost:         100.0,
		EffectiveCost:      80.0,
		ListCost:           120.0,
		ContractedCost:     100.0,
		ConsumedQuantity:   1.0,
		ConsumedUnit:       "Hours",
	}
}

// BenchmarkValidateFocusRecord_ValidRecord measures validation performance
// on a valid record (happy path). This should achieve 0 allocations.
func BenchmarkValidateFocusRecord_ValidRecord(b *testing.B) {
	record := createValidBenchmarkRecord()

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecordWithOptions_FailFast measures fail-fast mode
// on a valid record. This should achieve 0 allocations on success path.
func BenchmarkValidateFocusRecordWithOptions_FailFast(b *testing.B) {
	record := createValidBenchmarkRecord()
	opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeFailFast}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecordWithOptions(record, opts)
	}
}

// BenchmarkValidateFocusRecordWithOptions_Aggregate measures aggregate mode
// on a valid record. This should achieve 0 allocations on success path.
func BenchmarkValidateFocusRecordWithOptions_Aggregate(b *testing.B) {
	record := createValidBenchmarkRecord()
	opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecordWithOptions(record, opts)
	}
}

// BenchmarkValidateFocusRecord_CostHierarchyValid measures cost hierarchy
// validation specifically (FR-001, FR-002 checks pass).
func BenchmarkValidateFocusRecord_CostHierarchyValid(b *testing.B) {
	record := createValidBenchmarkRecord()
	// Set proper cost hierarchy: ListCost >= BilledCost >= EffectiveCost
	record.ListCost = 150.0
	record.BilledCost = 100.0
	record.EffectiveCost = 80.0

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecord_WithCommitmentDiscount measures validation
// with commitment discount fields properly set.
func BenchmarkValidateFocusRecord_WithCommitmentDiscount(b *testing.B) {
	record := createValidBenchmarkRecord()
	record.CommitmentDiscountId = "ri-12345"
	record.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecord_WithCapacityReservation measures validation
// with capacity reservation fields properly set.
func BenchmarkValidateFocusRecord_WithCapacityReservation(b *testing.B) {
	record := createValidBenchmarkRecord()
	record.CapacityReservationId = "cr-12345"
	record.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecord_WithPricing measures validation
// with pricing fields properly set (FR-006 validation).
func BenchmarkValidateFocusRecord_WithPricing(b *testing.B) {
	record := createValidBenchmarkRecord()
	record.PricingQuantity = 10.0
	record.PricingUnit = "Hours"

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecord_FullRecord measures validation on a record
// with all contextual FinOps fields populated.
func BenchmarkValidateFocusRecord_FullRecord(b *testing.B) {
	record := createValidBenchmarkRecord()
	// Add all optional fields that affect contextual validation
	record.ListCost = 150.0
	record.BilledCost = 100.0
	record.EffectiveCost = 80.0
	record.CommitmentDiscountId = "ri-12345"
	record.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED
	record.CapacityReservationId = "cr-12345"
	record.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED
	record.PricingQuantity = 10.0
	record.PricingUnit = "Hours"

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}

// BenchmarkValidateFocusRecord_ErrorPath measures validation when an error
// is returned. This path may allocate for the error slice.
func BenchmarkValidateFocusRecord_ErrorPath(b *testing.B) {
	record := createValidBenchmarkRecord()
	// Trigger FR-001: EffectiveCost > BilledCost
	record.EffectiveCost = 200.0
	record.BilledCost = 100.0

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateFocusRecord(record)
	}
}
