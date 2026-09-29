package testing_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// =============================================================================
// FOCUS 1.3 Conformance Tests
// =============================================================================
//
// These tests validate backward compatibility and correct behavior of
// FOCUS 1.3 extensions in the pluginsdk package.

// buildValidFocusRecord creates a minimal valid FOCUS record with all required fields.
// This helper ensures tests focus on FOCUS 1.3 specific behavior.
func buildValidFocusRecord() *pluginsdk.FocusRecordBuilder {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	return pluginsdk.NewFocusRecordBuilder().
		WithIdentity("AWS", "123456789012", "Production").
		WithBillingPeriod(billingStart, billingEnd, "USD").
		WithChargePeriod(chargeStart, chargeEnd).
		WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2").
		WithChargeDetails(
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		).
		WithChargeClassification(
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			"EC2 usage",
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		).
		WithUsage(100, "Hours").
		WithFinancials(100.00, 100.00, 95.00, "USD", "INV-2024-001").
		WithContractedCost(95.00)
}

// TestFocus13_BackwardCompatibility_FOCUS12OnlyRecords validates that plugins
// can create valid FOCUS 1.2 records without any FOCUS 1.3 fields (T060, T061).
func TestFocus13_BackwardCompatibility_FOCUS12OnlyRecords(t *testing.T) {
	// Create a pure FOCUS 1.2 record with no FOCUS 1.3 fields
	record, err := buildValidFocusRecord().Build()

	if err != nil {
		t.Fatalf("FOCUS 1.2 record creation failed: %v", err)
	}

	// Verify FOCUS 1.3 fields are empty/zero (default values)
	if record.GetServiceProviderName() != "" {
		t.Errorf("Expected empty service_provider_name, got %q", record.GetServiceProviderName())
	}
	if record.GetHostProviderName() != "" {
		t.Errorf("Expected empty host_provider_name, got %q", record.GetHostProviderName())
	}
	if record.GetAllocatedMethodId() != "" {
		t.Errorf("Expected empty allocated_method_id, got %q", record.GetAllocatedMethodId())
	}
	if record.GetAllocatedResourceId() != "" {
		t.Errorf("Expected empty allocated_resource_id, got %q", record.GetAllocatedResourceId())
	}
	if record.GetContractApplied() != "" {
		t.Errorf("Expected empty contract_applied, got %q", record.GetContractApplied())
	}
	if len(record.GetAllocatedTags()) != 0 {
		t.Errorf("Expected empty allocated_tags, got %v", record.GetAllocatedTags())
	}

	// Verify FOCUS 1.2 fields are correctly populated
	//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
	if record.GetProviderName() != "AWS" {
		//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		t.Errorf("Expected provider_name='AWS', got %q", record.GetProviderName())
	}
	if record.GetBillingAccountId() != "123456789012" {
		t.Errorf("Expected billing_account_id='123456789012', got %q", record.GetBillingAccountId())
	}
}

// TestFocus13_BackwardCompatibility_DeprecatedFieldsWork validates that
// deprecated fields (provider_name, publisher) continue to function (T062).
func TestFocus13_BackwardCompatibility_DeprecatedFieldsWork(t *testing.T) {
	t.Run("provider_name continues to work", func(t *testing.T) {
		record, err := buildValidFocusRecord().Build()

		if err != nil {
			t.Fatalf("Record creation failed: %v", err)
		}

		//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		if record.GetProviderName() != "AWS" {
			//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
			t.Errorf("Expected provider_name='AWS', got %q", record.GetProviderName())
		}
	})

	t.Run("publisher continues to work", func(t *testing.T) {
		//nolint:staticcheck // SA1019: Testing deprecated WithPublisher backward compatibility
		record, err := buildValidFocusRecord().
			WithPublisher("AWS Inc.").
			Build()

		if err != nil {
			t.Fatalf("Record creation failed: %v", err)
		}

		//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
		if record.GetPublisher() != "AWS Inc." {
			//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
			t.Errorf("Expected publisher='AWS Inc.', got %q", record.GetPublisher())
		}
	})
}

// TestFocus13_NewFieldsDefaultValues validates that all new FOCUS 1.3 fields
// have sensible default values that don't affect existing behavior (T063).
func TestFocus13_NewFieldsDefaultValues(t *testing.T) {
	record, err := buildValidFocusRecord().Build()

	if err != nil {
		t.Fatalf("Record creation with defaults failed: %v", err)
	}

	// All FOCUS 1.3 string fields should default to empty string
	focus13StringFields := map[string]string{
		"service_provider_name":    record.GetServiceProviderName(),
		"host_provider_name":       record.GetHostProviderName(),
		"allocated_method_id":      record.GetAllocatedMethodId(),
		"allocated_method_details": record.GetAllocatedMethodDetails(),
		"allocated_resource_id":    record.GetAllocatedResourceId(),
		"allocated_resource_name":  record.GetAllocatedResourceName(),
		"contract_applied":         record.GetContractApplied(),
	}

	for fieldName, value := range focus13StringFields {
		if value != "" {
			t.Errorf("FOCUS 1.3 field %s should default to empty, got %q", fieldName, value)
		}
	}

	// AllocatedTags should default to nil or empty map
	if tags := record.GetAllocatedTags(); len(tags) > 0 {
		t.Errorf("allocated_tags should be empty by default, got %v", tags)
	}
}

// TestFocus13_AllocationFields validates the new cost allocation builder methods.
func TestFocus13_AllocationFields(t *testing.T) {
	t.Run("allocation fields set correctly", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithAllocation("proportional-cpu", "Allocated by CPU usage percentage").
			WithAllocatedResource("workload-frontend", "Frontend Application").
			WithAllocatedTags(map[string]string{
				"team":        "frontend",
				"environment": "production",
			}).
			Build()

		if err != nil {
			t.Fatalf("Record creation failed: %v", err)
		}

		if record.GetAllocatedMethodId() != "proportional-cpu" {
			t.Errorf("Expected allocated_method_id='proportional-cpu', got %q",
				record.GetAllocatedMethodId())
		}
		if record.GetAllocatedMethodDetails() != "Allocated by CPU usage percentage" {
			t.Errorf("Expected allocated_method_details='Allocated by CPU usage percentage', got %q",
				record.GetAllocatedMethodDetails())
		}
		if record.GetAllocatedResourceId() != "workload-frontend" {
			t.Errorf("Expected allocated_resource_id='workload-frontend', got %q",
				record.GetAllocatedResourceId())
		}
		if record.GetAllocatedResourceName() != "Frontend Application" {
			t.Errorf("Expected allocated_resource_name='Frontend Application', got %q",
				record.GetAllocatedResourceName())
		}

		tags := record.GetAllocatedTags()
		if tags["team"] != "frontend" {
			t.Errorf("Expected allocated_tags['team']='frontend', got %q", tags["team"])
		}
		if tags["environment"] != "production" {
			t.Errorf("Expected allocated_tags['environment']='production', got %q", tags["environment"])
		}
	})
}

// TestFocus13_ProviderFields validates service_provider_name and host_provider_name.
func TestFocus13_ProviderFields(t *testing.T) {
	t.Run("marketplace scenario", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_MANAGEMENT, "Datadog APM").
			WithServiceProvider("Datadog"). // ISV selling via marketplace
			WithHostProvider("AWS").        // Cloud platform hosting
			Build()

		if err != nil {
			t.Fatalf("Marketplace record creation failed: %v", err)
		}

		if record.GetServiceProviderName() != "Datadog" {
			t.Errorf("Expected service_provider_name='Datadog', got %q",
				record.GetServiceProviderName())
		}
		if record.GetHostProviderName() != "AWS" {
			t.Errorf("Expected host_provider_name='AWS', got %q",
				record.GetHostProviderName())
		}
	})

	t.Run("same provider scenario", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithServiceProvider("AWS"). // AWS is both service and host provider
			WithHostProvider("AWS").
			Build()

		if err != nil {
			t.Fatalf("Same provider record creation failed: %v", err)
		}

		if record.GetServiceProviderName() != "AWS" {
			t.Errorf("Expected service_provider_name='AWS', got %q",
				record.GetServiceProviderName())
		}
		if record.GetHostProviderName() != "AWS" {
			t.Errorf("Expected host_provider_name='AWS', got %q",
				record.GetHostProviderName())
		}
	})
}

// TestFocus13_ContractApplied validates the ContractApplied field linking.
func TestFocus13_ContractApplied(t *testing.T) {
	t.Run("links to contract commitment", func(t *testing.T) {
		// First create a contract commitment
		commitment, err := pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commit-ri-001", "contract-ea-2025").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
			WithType("Reserved Instance").
			WithFinancials(10000.00, 0, "", "USD").
			WithBaselineTerms(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build()

		if err != nil {
			t.Fatalf("Contract commitment creation failed: %v", err)
		}

		// Then link a cost record to it
		record, err := buildValidFocusRecord(). //nolint:staticcheck // SA1019: bare-ID setter keeps the legacy wire value
							WithContractApplied(commitment.GetContractCommitmentId()).
							Build()

		if err != nil {
			t.Fatalf("Record with contract applied failed: %v", err)
		}

		if record.GetContractApplied() != "commit-ri-001" {
			t.Errorf("Expected contract_applied='commit-ri-001', got %q",
				record.GetContractApplied())
		}
	})

	t.Run("accepts any string as opaque reference", func(t *testing.T) {
		// ContractApplied accepts any string - no validation against commitment dataset
		record, err := buildValidFocusRecord(). //nolint:staticcheck // SA1019: bare-ID setter keeps the legacy wire value
							WithContractApplied("any-arbitrary-commitment-id").
							Build()

		if err != nil {
			t.Fatalf("Record with arbitrary contract_applied failed: %v", err)
		}

		if record.GetContractApplied() != "any-arbitrary-commitment-id" {
			t.Errorf("Expected contract_applied='any-arbitrary-commitment-id', got %q",
				record.GetContractApplied())
		}
	})
}

// TestFocus13_ContractCommitmentBuilder_SPEND validates SPEND category commitments.
func TestFocus13_ContractCommitmentBuilder_SPEND(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	commitment, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commit-sp-001", "aws-ea-2025").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
		WithType("Savings Plan").
		WithCommitmentPeriod(start, end).
		WithContractPeriod(start, end.AddDate(2, 0, 0)).
		WithFinancials(100000.00, 0, "", "USD").
		WithBaselineTerms(start).
		Build()

	if err != nil {
		t.Fatalf("SPEND commitment creation failed: %v", err)
	}

	if commitment.GetContractCommitmentId() != "commit-sp-001" {
		t.Errorf("Expected contract_commitment_id='commit-sp-001', got %q",
			commitment.GetContractCommitmentId())
	}
	if commitment.GetContractId() != "aws-ea-2025" {
		t.Errorf("Expected contract_id='aws-ea-2025', got %q",
			commitment.GetContractId())
	}
	if commitment.GetContractCommitmentCategory() !=
		pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND {
		t.Errorf("Expected SPEND category, got %v",
			commitment.GetContractCommitmentCategory())
	}
	if commitment.GetContractCommitmentCost() != 100000.00 {
		t.Errorf("Expected cost=100000.00, got %f",
			commitment.GetContractCommitmentCost())
	}
}

// TestFocus13_ContractCommitmentBuilder_USAGE validates USAGE category commitments.
func TestFocus13_ContractCommitmentBuilder_USAGE(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	commitment, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commit-cud-001", "gcp-cud-2025").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
		WithType("Committed Use Discount").
		WithCommitmentPeriod(start, end).
		WithFinancials(0, 10000, "vCPU-Hours", "USD").
		WithBaselineTerms(start).
		Build()

	if err != nil {
		t.Fatalf("USAGE commitment creation failed: %v", err)
	}

	if commitment.GetContractCommitmentCategory() !=
		pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE {
		t.Errorf("Expected USAGE category, got %v",
			commitment.GetContractCommitmentCategory())
	}
	if commitment.GetContractCommitmentQuantity() != 10000 {
		t.Errorf("Expected quantity=10000, got %f",
			commitment.GetContractCommitmentQuantity())
	}
	if commitment.GetContractCommitmentUnit() != "vCPU-Hours" {
		t.Errorf("Expected unit='vCPU-Hours', got %q",
			commitment.GetContractCommitmentUnit())
	}
}

// TestFocus13_ContractCommitmentBuilder_Validation validates builder error handling.
func TestFocus13_ContractCommitmentBuilder_Validation(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		_, err := pluginsdk.NewContractCommitmentBuilder().
			WithFinancials(100.0, 0, "", "USD").
			Build()

		if err == nil {
			t.Error("Expected error for missing contract_commitment_id")
		}
	})

	t.Run("invalid currency", func(t *testing.T) {
		_, err := pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commit-001", "contract-001").
			WithFinancials(100.0, 0, "", "ZZZ"). // Invalid currency
			Build()

		if err == nil {
			t.Error("Expected error for invalid currency")
		}
	})

	t.Run("invalid period", func(t *testing.T) {
		start := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		_, err := pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commit-001", "contract-001").
			WithCommitmentPeriod(start, end). // End before start
			WithFinancials(100.0, 0, "", "USD").
			Build()

		if err == nil {
			t.Error("Expected error for invalid commitment period")
		}
	})

	t.Run("negative cost", func(t *testing.T) {
		_, err := pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commit-001", "contract-001").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
			WithFinancials(-100.0, 0, "", "USD"). // Negative cost
			Build()

		if err == nil {
			t.Error("Expected error for negative cost")
		}
	})

	t.Run("negative quantity", func(t *testing.T) {
		_, err := pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commit-001", "contract-001").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
			WithFinancials(0, -100.0, "vCPU-Hours", "USD"). // Negative quantity
			Build()

		if err == nil {
			t.Error("Expected error for negative quantity")
		}
	})
}

// TestFocus13_MixedVersionRecords validates records with both FOCUS 1.2 and 1.3 fields.
func TestFocus13_MixedVersionRecords(t *testing.T) {
	t.Run("all FOCUS 1.3 fields populated", func(t *testing.T) {
		record, err := buildValidFocusRecord(). //nolint:staticcheck // SA1019: bare-ID setter keeps the legacy wire value
			// FOCUS 1.3 fields
			WithServiceProvider("AWS").
			WithHostProvider("AWS").
			WithAllocation("proportional-cost", "Cost split by resource usage").
			WithAllocatedResource("app-backend", "Backend Service").
			WithAllocatedTags(map[string]string{
				"cost-center": "engineering",
				"project":     "infrastructure",
			}).
			WithContractApplied("commit-ri-backend-001").
			Build()

		if err != nil {
			t.Fatalf("Mixed version record creation failed: %v", err)
		}

		// Verify both 1.2 and 1.3 fields are populated
		//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		if record.GetProviderName() != "AWS" {
			t.Error("FOCUS 1.2 provider_name not set")
		}
		if record.GetServiceProviderName() != "AWS" {
			t.Error("FOCUS 1.3 service_provider_name not set")
		}
		if record.GetAllocatedMethodId() != "proportional-cost" {
			t.Error("FOCUS 1.3 allocated_method_id not set")
		}
		if record.GetContractApplied() != "commit-ri-backend-001" {
			t.Error("FOCUS 1.3 contract_applied not set")
		}
	})
}

// =============================================================================
// Contextual FinOps Validation Conformance Tests (Feature 027-finops-validation)
// =============================================================================
//
// These tests verify the contextual FinOps validation rules are correctly
// enforced for FOCUS cost records. They test the validation API from the
// pluginsdk package.

// TestContextualValidation_CostHierarchy validates FR-001 and FR-002.
func TestContextualValidation_CostHierarchy(t *testing.T) {
	t.Run("valid cost hierarchy passes", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithFinancials(100.0, 120.0, 80.0, "USD", "INV-001"). // ListCost >= BilledCost >= EffectiveCost
			Build()
		require.NoError(t, err, "Record creation failed")
		require.NoError(t, pluginsdk.ValidateFocusRecord(record), "Valid cost hierarchy should pass")
	})

	t.Run("invalid cost hierarchies fail", func(t *testing.T) {
		tests := []struct {
			name          string
			billedCost    float64
			effectiveCost float64
			listCost      float64
		}{
			{"effectiveCost > billedCost", 100.0, 120.0, 150.0},
			{"listCost < effectiveCost", 100.0, 80.0, 50.0},
			{"listCost < billedCost < effectiveCost", 100.0, 120.0, 50.0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				record, err := buildValidFocusRecord().Build()
				require.NoError(t, err, "Record creation failed")
				record.BilledCost = tt.billedCost
				record.EffectiveCost = tt.effectiveCost
				record.ListCost = tt.listCost
				require.Error(t, pluginsdk.ValidateFocusRecord(record), "Invalid hierarchy should fail validation")
			})
		}
	})

	t.Run("correction charges exempt from hierarchy", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithChargeClassification(
				pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION,
				"Billing correction",
				pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_ONE_TIME,
			).
			WithFinancials(50.0, 100.0, 200.0, "USD", "INV-001"). // Would fail without CORRECTION
			Build()
		require.NoError(t, err, "Record creation failed")
		require.NoError(t, pluginsdk.ValidateFocusRecord(record), "Correction charges should be exempt")
	})
}

// TestContextualValidation_CommitmentDiscountConsistency validates FR-003 and FR-004.
func TestContextualValidation_CommitmentDiscountConsistency(t *testing.T) {
	t.Run("commitment ID with status passes for usage", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithCommitmentDiscount(
				pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_SPEND,
				"ri-12345",
				"Reserved Instance",
			).
			WithCommitmentDiscountDetails(
				100.0, // quantity
				pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
				"EC2 1yr RI", // discountType
				"Hours",      // unit
			).
			Build()
		require.NoError(t, err, "Record creation failed")
		require.NoError(t, pluginsdk.ValidateFocusRecord(record), "Valid commitment discount should pass")
	})

	t.Run("commitment ID without status fails for usage", func(t *testing.T) {
		// Build valid record first, then modify to create invalid state
		record, err := buildValidFocusRecord().Build()
		require.NoError(t, err, "Record creation failed")
		// Set CommitmentDiscountId without status (status stays UNSPECIFIED)
		record.CommitmentDiscountId = "ri-12345"

		require.Error(t, pluginsdk.ValidateFocusRecord(record),
			"Commitment ID without status should fail for usage charges")
	})
}

// TestContextualValidation_PricingConsistency validates FR-006.
func TestContextualValidation_PricingConsistency(t *testing.T) {
	t.Run("pricing quantity with unit passes", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithPricing(10.0, "Hours", 100.0). // quantity, unit, listUnitPrice
			Build()
		require.NoError(t, err, "Record creation failed")
		require.NoError(t, pluginsdk.ValidateFocusRecord(record), "Valid pricing should pass")
	})

	t.Run("pricing quantity without unit fails", func(t *testing.T) {
		// Build valid record first, then modify to create invalid state
		record, err := buildValidFocusRecord().Build()
		require.NoError(t, err, "Record creation failed")
		// Set pricing quantity > 0 without unit
		record.PricingQuantity = 10.0
		record.PricingUnit = ""

		require.Error(t, pluginsdk.ValidateFocusRecord(record), "Pricing quantity > 0 without unit should fail")
	})
}

// TestContextualValidation_CapacityReservationConsistency validates FR-005.
func TestContextualValidation_CapacityReservationConsistency(t *testing.T) {
	t.Run("capacity reservation with status passes", func(t *testing.T) {
		record, err := buildValidFocusRecord().
			WithCapacityReservation(
				"cr-12345",
				pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
			).
			Build()
		require.NoError(t, err, "Record creation failed")
		require.NoError(t, pluginsdk.ValidateFocusRecord(record), "Valid capacity reservation should pass")
	})

	t.Run("capacity reservation without status fails for usage", func(t *testing.T) {
		record, err := buildValidFocusRecord().Build()
		require.NoError(t, err, "Record creation failed")
		// Set capacity ID but not status
		record.CapacityReservationId = "cr-12345"
		// Status remains UNSPECIFIED

		require.Error(t, pluginsdk.ValidateFocusRecord(record),
			"Capacity ID without status should fail for usage charges")
	})
}

// TestContextualValidation_AggregateMode validates aggregate error collection.
func TestContextualValidation_AggregateMode(t *testing.T) {
	t.Run("aggregate mode collects all errors", func(t *testing.T) {
		// Build valid record first, then modify to create multiple violations
		record, err := buildValidFocusRecord().Build()
		require.NoError(t, err, "Record creation failed")
		// Add FR-001 violation: EffectiveCost > BilledCost
		record.BilledCost = 100.0
		record.EffectiveCost = 120.0
		// Add FR-006 violation: PricingQuantity > 0 without unit
		record.PricingQuantity = 10.0
		record.PricingUnit = ""

		opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
		errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

		require.GreaterOrEqual(t, len(errs), 2, "Expected multiple errors in aggregate mode")
	})

	t.Run("valid record returns empty slice", func(t *testing.T) {
		record, err := buildValidFocusRecord().Build()
		require.NoError(t, err, "Record creation failed")

		opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
		errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

		require.Empty(t, errs, "Valid record should return no errors")
	})
}
