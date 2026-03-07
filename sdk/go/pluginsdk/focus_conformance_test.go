package pluginsdk_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// FOCUS 1.2 Column Definitions for schema completeness verification.
// Reference: https://focus.finops.org/focus-specification/v1-2/.
// Total: 57 columns (14 mandatory, 1 recommended, 42 conditional).
//
//nolint:gochecknoglobals // Test data for FOCUS column validation.
var focus12Columns = map[string]struct {
	ProtoField string // Expected proto field name (snake_case)
	ProtoType  string // Expected Go type after generation
	Level      string // Mandatory, Recommended, or Conditional
}{
	// ==========================================================================
	// MANDATORY COLUMNS (14 total)
	// ==========================================================================
	"BilledCost":         {"billed_cost", "float64", "Mandatory"},
	"BillingAccountId":   {"billing_account_id", "string", "Mandatory"},
	"BillingAccountName": {"billing_account_name", "string", "Mandatory"},
	"BillingCurrency":    {"billing_currency", "string", "Mandatory"},
	"BillingPeriodEnd":   {"billing_period_end", "*timestamppb.Timestamp", "Mandatory"},
	"BillingPeriodStart": {"billing_period_start", "*timestamppb.Timestamp", "Mandatory"},
	"ChargeCategory":     {"charge_category", "FocusChargeCategory", "Mandatory"},
	"ChargeClass":        {"charge_class", "FocusChargeClass", "Mandatory"},
	"ChargeDescription":  {"charge_description", "string", "Mandatory"},
	"ChargePeriodEnd":    {"charge_period_end", "*timestamppb.Timestamp", "Mandatory"},
	"ChargePeriodStart":  {"charge_period_start", "*timestamppb.Timestamp", "Mandatory"},
	"ContractedCost":     {"contracted_cost", "float64", "Mandatory"},
	"ProviderName":       {"provider_name", "string", "Mandatory"},
	"ServiceName":        {"service_name", "string", "Mandatory"},

	// ==========================================================================
	// RECOMMENDED COLUMNS (1 total)
	// ==========================================================================
	"ChargeFrequency": {"charge_frequency", "FocusChargeFrequency", "Recommended"},

	// ==========================================================================
	// CONDITIONAL COLUMNS (42 total)
	// ==========================================================================

	// Account Types
	"BillingAccountType": {"billing_account_type", "string", "Conditional"},
	"SubAccountId":       {"sub_account_id", "string", "Conditional"},
	"SubAccountName":     {"sub_account_name", "string", "Conditional"},
	"SubAccountType":     {"sub_account_type", "string", "Conditional"},

	// Capacity Reservation
	"CapacityReservationId":     {"capacity_reservation_id", "string", "Conditional"},
	"CapacityReservationStatus": {"capacity_reservation_status", "FocusCapacityReservationStatus", "Conditional"},

	// Commitment Discounts
	"CommitmentDiscountCategory": {"commitment_discount_category", "FocusCommitmentDiscountCategory", "Conditional"},
	"CommitmentDiscountId":       {"commitment_discount_id", "string", "Conditional"},
	"CommitmentDiscountName":     {"commitment_discount_name", "string", "Conditional"},
	"CommitmentDiscountQuantity": {"commitment_discount_quantity", "float64", "Conditional"},
	"CommitmentDiscountStatus":   {"commitment_discount_status", "FocusCommitmentDiscountStatus", "Conditional"},
	"CommitmentDiscountType":     {"commitment_discount_type", "string", "Conditional"},
	"CommitmentDiscountUnit":     {"commitment_discount_unit", "string", "Conditional"},

	// Consumption/Usage
	"ConsumedQuantity": {"consumed_quantity", "float64", "Conditional"},
	"ConsumedUnit":     {"consumed_unit", "string", "Conditional"},

	// Financial
	"ContractedUnitPrice": {"contracted_unit_price", "float64", "Conditional"},
	"EffectiveCost":       {"effective_cost", "float64", "Conditional"},
	"ListCost":            {"list_cost", "float64", "Conditional"},

	// Invoice
	"InvoiceId":     {"invoice_id", "string", "Conditional"},
	"InvoiceIssuer": {"invoice_issuer", "string", "Conditional"},

	// Location
	"AvailabilityZone": {"availability_zone", "string", "Conditional"},
	"RegionId":         {"region_id", "string", "Conditional"},
	"RegionName":       {"region_name", "string", "Conditional"},

	// Metadata
	"Tags": {"tags", "map[string]string", "Conditional"},

	// Pricing
	"ListUnitPrice":                      {"list_unit_price", "float64", "Conditional"},
	"PricingCategory":                    {"pricing_category", "FocusPricingCategory", "Conditional"},
	"PricingCurrency":                    {"pricing_currency", "string", "Conditional"},
	"PricingCurrencyContractedUnitPrice": {"pricing_currency_contracted_unit_price", "float64", "Conditional"},
	"PricingCurrencyEffectiveCost":       {"pricing_currency_effective_cost", "float64", "Conditional"},
	"PricingCurrencyListUnitPrice":       {"pricing_currency_list_unit_price", "float64", "Conditional"},
	"PricingQuantity":                    {"pricing_quantity", "float64", "Conditional"},
	"PricingUnit":                        {"pricing_unit", "string", "Conditional"},

	// Resource
	"ResourceId":   {"resource_id", "string", "Conditional"},
	"ResourceName": {"resource_name", "string", "Conditional"},
	"ResourceType": {"resource_type", "string", "Conditional"},

	// Service
	"Publisher":          {"publisher", "string", "Conditional"},
	"ServiceCategory":    {"service_category", "FocusServiceCategory", "Conditional"},
	"ServiceSubcategory": {"service_subcategory", "string", "Conditional"},

	// SKU
	"SkuId":           {"sku_id", "string", "Conditional"},
	"SkuMeter":        {"sku_meter", "string", "Conditional"},
	"SkuPriceDetails": {"sku_price_details", "string", "Conditional"},
	"SkuPriceId":      {"sku_price_id", "string", "Conditional"},
}

// TestFOCUS12ColumnCompleteness verifies all 57 FOCUS 1.2 columns are present
// in the FocusCostRecord proto message.
func TestFOCUS12ColumnCompleteness(t *testing.T) {
	// Get the type of FocusCostRecord
	recordType := reflect.TypeOf(pbc.FocusCostRecord{})

	missingColumns := []string{}
	presentColumns := []string{}

	for focusColumn, spec := range focus12Columns {
		// Convert proto field name to Go struct field name (PascalCase)
		goFieldName := snakeToPascal(spec.ProtoField)

		// Check if the field exists in the struct
		_, found := recordType.FieldByName(goFieldName)
		if !found {
			missingColumns = append(missingColumns, focusColumn)
		} else {
			presentColumns = append(presentColumns, focusColumn)
		}
	}

	// Report results
	t.Logf("FOCUS 1.2 Column Audit Results:")
	t.Logf("  Total columns defined: %d", len(focus12Columns))
	t.Logf("  Present in proto: %d", len(presentColumns))
	t.Logf("  Missing from proto: %d", len(missingColumns))

	if len(missingColumns) > 0 {
		t.Errorf("Missing FOCUS 1.2 columns: %v", missingColumns)
	}

	// Verify we have exactly 57 columns (FOCUS 1.2 specification)
	if len(focus12Columns) != 57 {
		t.Errorf("Expected 57 FOCUS 1.2 columns, but test defines %d", len(focus12Columns))
	}
}

// TestFOCUS12MandatoryColumns verifies all mandatory columns are present.
func TestFOCUS12MandatoryColumns(t *testing.T) {
	recordType := reflect.TypeOf(pbc.FocusCostRecord{})

	mandatoryColumns := []string{}
	for focusColumn, spec := range focus12Columns {
		if spec.Level == "Mandatory" {
			mandatoryColumns = append(mandatoryColumns, focusColumn)
		}
	}

	missingMandatory := []string{}
	for _, col := range mandatoryColumns {
		spec := focus12Columns[col]
		goFieldName := snakeToPascal(spec.ProtoField)
		if _, found := recordType.FieldByName(goFieldName); !found {
			missingMandatory = append(missingMandatory, col)
		}
	}

	t.Logf("Mandatory columns: %d", len(mandatoryColumns))

	if len(missingMandatory) > 0 {
		t.Errorf("Missing MANDATORY FOCUS 1.2 columns: %v", missingMandatory)
	}

	// FOCUS 1.2 has 14 mandatory columns
	if len(mandatoryColumns) != 14 {
		t.Errorf("Expected 14 mandatory columns, got %d", len(mandatoryColumns))
	}
}

// TestFOCUS12NewEnumTypes verifies the new enum types are properly defined.
func TestFOCUS12NewEnumTypes(t *testing.T) {
	// Test FocusCommitmentDiscountStatus enum
	t.Run("FocusCommitmentDiscountStatus", func(t *testing.T) {
		// Verify enum values exist
		if pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED != 0 {
			t.Error("UNSPECIFIED should be 0")
		}
		if pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED != 1 {
			t.Error("USED should be 1")
		}
		if pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNUSED != 2 {
			t.Error("UNUSED should be 2")
		}
	})

	// Test FocusCapacityReservationStatus enum
	t.Run("FocusCapacityReservationStatus", func(t *testing.T) {
		// Verify enum values exist
		if pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED != 0 {
			t.Error("UNSPECIFIED should be 0")
		}
		if pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED != 1 {
			t.Error("USED should be 1")
		}
		if pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNUSED != 2 {
			t.Error("UNUSED should be 2")
		}
	})
}

// TestFOCUS12NewFieldsAccessible verifies all 19 new fields can be accessed.
func TestFOCUS12NewFieldsAccessible(t *testing.T) {
	record := &pbc.FocusCostRecord{}

	// Test all 19 new fields are accessible (fields 41-58 + enums)
	newFields := map[string]interface{}{
		"ContractedCost":                     record.GetContractedCost(),
		"BillingAccountType":                 record.GetBillingAccountType(),
		"SubAccountType":                     record.GetSubAccountType(),
		"CapacityReservationId":              record.GetCapacityReservationId(),
		"CapacityReservationStatus":          record.GetCapacityReservationStatus(),
		"CommitmentDiscountQuantity":         record.GetCommitmentDiscountQuantity(),
		"CommitmentDiscountStatus":           record.GetCommitmentDiscountStatus(),
		"CommitmentDiscountType":             record.GetCommitmentDiscountType(),
		"CommitmentDiscountUnit":             record.GetCommitmentDiscountUnit(),
		"ContractedUnitPrice":                record.GetContractedUnitPrice(),
		"PricingCurrency":                    record.GetPricingCurrency(),
		"PricingCurrencyContractedUnitPrice": record.GetPricingCurrencyContractedUnitPrice(),
		"PricingCurrencyEffectiveCost":       record.GetPricingCurrencyEffectiveCost(),
		"PricingCurrencyListUnitPrice":       record.GetPricingCurrencyListUnitPrice(),
		"Publisher":                          record.GetPublisher(), //nolint:staticcheck // SA1019: deprecated field test
		"ServiceSubcategory":                 record.GetServiceSubcategory(),
		"SkuMeter":                           record.GetSkuMeter(),
		"SkuPriceDetails":                    record.GetSkuPriceDetails(),
	}

	t.Logf("Verified %d new FOCUS 1.2 fields are accessible", len(newFields))

	// All should return zero values without panicking
	for name, value := range newFields {
		if value == nil {
			t.Logf("  %s: nil (pointer type)", name)
		} else {
			t.Logf("  %s: %v (zero value)", name, value)
		}
	}
}

// snakeToPascal converts snake_case to PascalCase for Go struct field lookup.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// TestFOCUS12TypeMappings verifies proto types match FOCUS data type requirements.
// FOCUS types: String -> string, Decimal -> float64, DateTime -> *timestamppb.Timestamp.
func TestFOCUS12TypeMappings(t *testing.T) {
	recordType := reflect.TypeOf(pbc.FocusCostRecord{})

	// Define expected type mappings based on FOCUS specification
	typeMappings := map[string]string{
		// Decimal fields -> float64
		"BilledCost":                         "float64",
		"ContractedCost":                     "float64",
		"ContractedUnitPrice":                "float64",
		"EffectiveCost":                      "float64",
		"ListCost":                           "float64",
		"ListUnitPrice":                      "float64",
		"PricingQuantity":                    "float64",
		"ConsumedQuantity":                   "float64",
		"CommitmentDiscountQuantity":         "float64",
		"PricingCurrencyContractedUnitPrice": "float64",
		"PricingCurrencyEffectiveCost":       "float64",
		"PricingCurrencyListUnitPrice":       "float64",

		// DateTime fields -> *timestamppb.Timestamp
		"BillingPeriodStart": "*timestamppb.Timestamp",
		"BillingPeriodEnd":   "*timestamppb.Timestamp",
		"ChargePeriodStart":  "*timestamppb.Timestamp",
		"ChargePeriodEnd":    "*timestamppb.Timestamp",

		// String fields -> string
		"ProviderName":       "string",
		"BillingAccountId":   "string",
		"BillingAccountName": "string",
		"BillingCurrency":    "string",
		"ChargeDescription":  "string",
		"ServiceName":        "string",
	}

	errors := []string{}
	for fieldName, expectedType := range typeMappings {
		field, found := recordType.FieldByName(fieldName)
		if !found {
			errors = append(errors, fmt.Sprintf("%s: field not found", fieldName))
			continue
		}

		actualType := field.Type.String()
		// Normalize type names for comparison
		if strings.Contains(actualType, "timestamp") {
			actualType = "*timestamppb.Timestamp"
		}

		if actualType != expectedType {
			errors = append(errors, fmt.Sprintf("%s: expected %s, got %s", fieldName, expectedType, actualType))
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("Type mismatch: %s", err)
		}
	}

	t.Logf("Verified %d type mappings", len(typeMappings))
}

// =============================================================================
// Validation Enhancement Tests
// =============================================================================

// TestValidateFocusRecord_MandatoryFields tests that all 14 mandatory fields are validated.
func TestValidateFocusRecord_MandatoryFields(t *testing.T) {
	tests := []struct {
		name          string
		modifyFunc    func(*pbc.FocusCostRecord)
		errorContains string
		expectedField string
	}{
		{
			name: "missing provider_name",
			//nolint:staticcheck // SA1019: Testing validation of deprecated provider_name field
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.ProviderName = "" },
			errorContains: "provider_name",
			expectedField: "provider_name",
		},
		{
			name:          "missing billing_account_id",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.BillingAccountId = "" },
			errorContains: "billing_account_id",
			expectedField: "billing_account_id",
		},
		{
			name:          "missing billing_period_start",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.BillingPeriodStart = nil },
			errorContains: "billing_period.start",
			expectedField: "billing_period.start",
		},
		{
			name:          "missing billing_period_end",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.BillingPeriodEnd = nil },
			errorContains: "billing_period.end",
			expectedField: "billing_period.end",
		},
		{
			name:          "missing charge_period_start",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.ChargePeriodStart = nil },
			errorContains: "charge_period.start",
			expectedField: "charge_period.start",
		},
		{
			name:          "missing charge_period_end",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.ChargePeriodEnd = nil },
			errorContains: "charge_period.end",
			expectedField: "charge_period.end",
		},
		{
			name: "missing charge_category",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_UNSPECIFIED
			},
			errorContains: "charge_category",
			expectedField: "charge_category",
		},
		{
			name: "missing charge_class",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.ChargeClass = pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_UNSPECIFIED
			},
			errorContains: "charge_class",
			expectedField: "charge_class",
		},
		{
			name:          "missing charge_description",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.ChargeDescription = "" },
			errorContains: "charge_description",
			expectedField: "charge_description",
		},
		{
			name: "missing service_category",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.ServiceCategory = pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_UNSPECIFIED
			},
			errorContains: "service_category",
			expectedField: "service_category",
		},
		{
			name:          "missing service_name",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.ServiceName = "" },
			errorContains: "service_name",
			expectedField: "service_name",
		},
		{
			name:          "missing billing_currency",
			modifyFunc:    func(r *pbc.FocusCostRecord) { r.BillingCurrency = "" },
			errorContains: "billing_currency",
			expectedField: "billing_currency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			tt.modifyFunc(record)

			err := pluginsdk.ValidateFocusRecord(record)
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.errorContains)
				return
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
			}
			// Verify structured error extraction
			var valErr *pluginsdk.ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("errors.As failed: expected *ValidationError, got %T", err)
				return
			}
			if valErr.FieldName != tt.expectedField {
				t.Errorf("FieldName = %q, want %q", valErr.FieldName, tt.expectedField)
			}
		})
	}
}

// TestValidateFocusRecord_ISO4217Currency tests ISO 4217 currency validation.
func TestValidateFocusRecord_ISO4217Currency(t *testing.T) {
	tests := []struct {
		name          string
		billingCurr   string
		pricingCurr   string
		expectError   bool
		errorContains string
	}{
		{"valid USD", "USD", "", false, ""},
		{"valid EUR", "EUR", "", false, ""},
		{"valid GBP", "GBP", "", false, ""},
		{"valid JPY", "JPY", "", false, ""},
		{"valid CNY", "CNY", "", false, ""},
		{"valid CAD", "CAD", "", false, ""},
		{"valid AUD", "AUD", "", false, ""},
		{"invalid billing currency", "ABC", "", true, "billing_currency: must be valid ISO 4217"},
		{"invalid lowercase", "usd", "", true, "billing_currency: must be valid ISO 4217"},
		{"empty pricing currency ok", "USD", "", false, ""},
		{"valid pricing currency", "USD", "EUR", false, ""},
		{"invalid pricing currency", "USD", "ZZZ", true, "pricing_currency: must be valid ISO 4217"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.BillingCurrency = tt.billingCurr
			record.PricingCurrency = tt.pricingCurr

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_ContractedCostRule tests the ContractedCost business rule.
func TestValidateFocusRecord_ContractedCostRule(t *testing.T) {
	tests := []struct {
		name                string
		contractedCost      float64
		contractedUnitPrice float64
		pricingQuantity     float64
		chargeClass         pbc.FocusChargeClass
		expectError         bool
		errorContains       string
	}{
		{
			name:                "valid calculation",
			contractedCost:      100.0,
			contractedUnitPrice: 10.0,
			pricingQuantity:     10.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         false,
		},
		{
			name:                "valid with decimals",
			contractedCost:      73.0,
			contractedUnitPrice: 0.10,
			pricingQuantity:     730.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         false,
		},
		{
			name:                "skip when unit price is zero",
			contractedCost:      100.0,
			contractedUnitPrice: 0.0,
			pricingQuantity:     10.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         false,
		},
		{
			name:                "skip when quantity is zero",
			contractedCost:      100.0,
			contractedUnitPrice: 10.0,
			pricingQuantity:     0.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         false,
		},
		{
			name:                "skip for correction charges",
			contractedCost:      999.0, // Doesn't match calculation
			contractedUnitPrice: 10.0,
			pricingQuantity:     10.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION,
			expectError:         false,
		},
		{
			name:                "invalid mismatch",
			contractedCost:      50.0, // Should be 100.0
			contractedUnitPrice: 10.0,
			pricingQuantity:     10.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         true,
			errorContains:       "contracted_cost",
		},
		{
			name:                "within tolerance",
			contractedCost:      100.001, // Slightly off but within 0.01% tolerance
			contractedUnitPrice: 10.0,
			pricingQuantity:     10.0,
			chargeClass:         pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.ContractedCost = tt.contractedCost
			record.ContractedUnitPrice = tt.contractedUnitPrice
			record.PricingQuantity = tt.pricingQuantity
			record.ChargeClass = tt.chargeClass
			// Set PricingUnit when PricingQuantity > 0 to satisfy FR-006 validation
			if tt.pricingQuantity > 0 {
				record.PricingUnit = "Hours"
			}

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_UsageQuantityRule tests consumed quantity validation for usage charges.
func TestValidateFocusRecord_UsageQuantityRule(t *testing.T) {
	tests := []struct {
		name             string
		chargeCategory   pbc.FocusChargeCategory
		consumedQuantity float64
		expectError      bool
	}{
		{"usage with quantity", pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE, 1.0, false},
		{"usage without quantity", pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE, 0.0, true},
		{"usage with negative quantity", pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE, -1.0, true},
		{"purchase without quantity ok", pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_PURCHASE, 0.0, false},
		{"credit without quantity ok", pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_CREDIT, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.ChargeCategory = tt.chargeCategory
			record.ConsumedQuantity = tt.consumedQuantity

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), "consumed_quantity") {
					t.Errorf("Expected consumed_quantity error, got %q", err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// createValidFocusRecord creates a valid FocusCostRecord for testing validation.
// Uses realistic distinct timestamps for billing and charge periods.
func createValidFocusRecord() *pbc.FocusCostRecord {
	now := time.Now()
	// Billing period: first of current month to first of next month
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	// Charge period: 24 hours ending now
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	return &pbc.FocusCostRecord{
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
		BilledCost:         10.0,
		ContractedCost:     10.0,
		ConsumedQuantity:   1.0,
		ConsumedUnit:       "Hours",
	}
}

// =============================================================================
// FOCUS 1.3 Validation Tests
// =============================================================================

// TestValidateFocusRecord_FOCUS13AllocationRule tests the allocation field dependency rules.
// FOCUS 1.3: allocated_method_id requires allocated_resource_id, but NOT vice versa.
func TestValidateFocusRecord_FOCUS13AllocationRule(t *testing.T) {
	tests := []struct {
		name                string
		allocatedMethodID   string
		allocatedResourceID string
		expectError         bool
		errorContains       string
	}{
		{
			name:                "valid: both method and resource set",
			allocatedMethodID:   "proportional-cpu",
			allocatedResourceID: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			expectError:         false,
		},
		{
			name:                "valid: neither method nor resource set",
			allocatedMethodID:   "",
			allocatedResourceID: "",
			expectError:         false,
		},
		{
			name:                "valid: resource set without method (resource tagged for allocation, method TBD)",
			allocatedMethodID:   "",
			allocatedResourceID: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			expectError:         false,
		},
		{
			name:                "invalid: method set without resource",
			allocatedMethodID:   "proportional-cpu",
			allocatedResourceID: "",
			expectError:         true,
			errorContains:       "allocated_resource_id: required when allocated_method_id set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.AllocatedMethodId = tt.allocatedMethodID
			record.AllocatedResourceId = tt.allocatedResourceID

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestFOCUS12EnumCompleteness verifies all enum types have expected values.
func TestFOCUS12EnumCompleteness(t *testing.T) {
	// FocusChargeCategory - FOCUS 1.2 charge categories
	t.Run("FocusChargeCategory", func(t *testing.T) {
		expectedValues := []pbc.FocusChargeCategory{
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_UNSPECIFIED,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_PURCHASE,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_CREDIT,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_TAX,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_REFUND,
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_ADJUSTMENT,
		}
		if len(pbc.FocusChargeCategory_name) < len(expectedValues) {
			t.Errorf("FocusChargeCategory missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusChargeCategory_name))
		}
	})

	// FocusChargeClass - FOCUS 1.2 charge classes
	t.Run("FocusChargeClass", func(t *testing.T) {
		expectedValues := []pbc.FocusChargeClass{
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_UNSPECIFIED,
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION,
		}
		if len(pbc.FocusChargeClass_name) < len(expectedValues) {
			t.Errorf("FocusChargeClass missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusChargeClass_name))
		}
	})

	// FocusChargeFrequency - FOCUS 1.2 charge frequencies
	t.Run("FocusChargeFrequency", func(t *testing.T) {
		expectedValues := []pbc.FocusChargeFrequency{
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_UNSPECIFIED,
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_ONE_TIME,
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_RECURRING,
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		}
		if len(pbc.FocusChargeFrequency_name) < len(expectedValues) {
			t.Errorf("FocusChargeFrequency missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusChargeFrequency_name))
		}
	})

	// FocusPricingCategory - FOCUS 1.2 pricing categories
	t.Run("FocusPricingCategory", func(t *testing.T) {
		expectedValues := []pbc.FocusPricingCategory{
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_OTHER,
		}
		if len(pbc.FocusPricingCategory_name) < len(expectedValues) {
			t.Errorf("FocusPricingCategory missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusPricingCategory_name))
		}
	})

	// FocusServiceCategory - FOCUS 1.2 service categories
	t.Run("FocusServiceCategory", func(t *testing.T) {
		expectedValues := []pbc.FocusServiceCategory{
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_UNSPECIFIED,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_STORAGE,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_NETWORK,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DATABASE,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_ANALYTICS,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_MACHINE_LEARNING,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_MANAGEMENT,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_SECURITY,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_DEVELOPER_TOOLS,
			pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_OTHER,
		}
		if len(pbc.FocusServiceCategory_name) < len(expectedValues) {
			t.Errorf("FocusServiceCategory missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusServiceCategory_name))
		}
	})

	// FocusCommitmentDiscountCategory - FOCUS 1.2 commitment discount categories
	t.Run("FocusCommitmentDiscountCategory", func(t *testing.T) {
		expectedValues := []pbc.FocusCommitmentDiscountCategory{
			pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_UNSPECIFIED,
			pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_SPEND,
			pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_USAGE,
		}
		if len(pbc.FocusCommitmentDiscountCategory_name) < len(expectedValues) {
			t.Errorf("FocusCommitmentDiscountCategory missing values: expected %d, got %d",
				len(expectedValues), len(pbc.FocusCommitmentDiscountCategory_name))
		}
	})

	// FocusCommitmentDiscountStatus - NEW in this feature
	t.Run("FocusCommitmentDiscountStatus", func(t *testing.T) {
		expectedValues := []pbc.FocusCommitmentDiscountStatus{
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNUSED,
		}
		if len(pbc.FocusCommitmentDiscountStatus_name) != len(expectedValues) {
			t.Errorf("FocusCommitmentDiscountStatus: expected %d values, got %d",
				len(expectedValues), len(pbc.FocusCommitmentDiscountStatus_name))
		}
	})

	// FocusCapacityReservationStatus - NEW in this feature
	t.Run("FocusCapacityReservationStatus", func(t *testing.T) {
		expectedValues := []pbc.FocusCapacityReservationStatus{
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNUSED,
		}
		if len(pbc.FocusCapacityReservationStatus_name) != len(expectedValues) {
			t.Errorf("FocusCapacityReservationStatus: expected %d values, got %d",
				len(expectedValues), len(pbc.FocusCapacityReservationStatus_name))
		}
	})
}

// =============================================================================
// Contextual FinOps Validation Tests (Feature 027-finops-validation)
// =============================================================================

// TestValidateFocusRecord_CostHierarchy tests the cost relationship validation (FR-001, FR-002).
// Cost hierarchy: ListCost >= BilledCost >= EffectiveCost (when all positive, non-correction).
func TestValidateFocusRecord_CostHierarchy(t *testing.T) {
	tests := []struct {
		name          string
		billedCost    float64
		effectiveCost float64
		listCost      float64
		chargeClass   pbc.FocusChargeClass
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid: all costs equal",
			billedCost:    100.0,
			effectiveCost: 100.0,
			listCost:      100.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   false,
		},
		{
			name:          "valid: proper hierarchy with discounts",
			billedCost:    100.0,
			effectiveCost: 80.0,
			listCost:      120.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   false,
		},
		{
			name:          "valid: zero costs (free tier)",
			billedCost:    0.0,
			effectiveCost: 0.0,
			listCost:      0.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   false,
		},
		{
			name:          "valid: negative effectiveCost (credits exempt)",
			billedCost:    100.0,
			effectiveCost: -50.0,
			listCost:      100.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   false,
		},
		{
			name:          "valid: correction charge exempt from hierarchy",
			billedCost:    50.0,
			effectiveCost: 150.0, // Would fail without CORRECTION exemption
			listCost:      100.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_CORRECTION,
			expectError:   false,
		},
		{
			name:          "invalid: effectiveCost exceeds billedCost (FR-001)",
			billedCost:    100.0,
			effectiveCost: 150.0,
			listCost:      200.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   true,
			errorContains: "effective_cost: must not exceed billed_cost",
		},
		{
			name:          "invalid: listCost less than effectiveCost (FR-002)",
			billedCost:    100.0,
			effectiveCost: 100.0,
			listCost:      50.0,
			chargeClass:   pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			expectError:   true,
			errorContains: "list_cost: must be >= effective_cost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.BilledCost = tt.billedCost
			record.EffectiveCost = tt.effectiveCost
			record.ListCost = tt.listCost
			record.ChargeClass = tt.chargeClass

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_CommitmentDiscountConsistency tests FR-003 and FR-004.
func TestValidateFocusRecord_CommitmentDiscountConsistency(t *testing.T) {
	tests := []struct {
		name                     string
		commitmentDiscountID     string
		commitmentDiscountStatus pbc.FocusCommitmentDiscountStatus
		chargeCategory           pbc.FocusChargeCategory
		expectError              bool
		errorContains            string
	}{
		{
			name:                     "valid: no commitment discount",
			commitmentDiscountID:     "",
			commitmentDiscountStatus: pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			chargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:              false,
		},
		{
			name:                     "valid: commitment with status for usage",
			commitmentDiscountID:     "ri-12345",
			commitmentDiscountStatus: pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			chargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:              false,
		},
		{
			name:                     "valid: commitment ID without status for purchase (FR-003 exception)",
			commitmentDiscountID:     "ri-12345",
			commitmentDiscountStatus: pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			chargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_PURCHASE,
			expectError:              false,
		},
		{
			name:                     "invalid: commitment ID set + Usage but no status (FR-003)",
			commitmentDiscountID:     "ri-12345",
			commitmentDiscountStatus: pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			chargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:              true,
			errorContains:            "commitment_discount_status: required when",
		},
		{
			name:                     "invalid: status set without ID (FR-004)",
			commitmentDiscountID:     "",
			commitmentDiscountStatus: pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			chargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:              true,
			errorContains:            "commitment_discount_id: required when",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.CommitmentDiscountId = tt.commitmentDiscountID
			record.CommitmentDiscountStatus = tt.commitmentDiscountStatus
			record.ChargeCategory = tt.chargeCategory

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_PricingConsistency tests FR-006.
func TestValidateFocusRecord_PricingConsistency(t *testing.T) {
	tests := []struct {
		name            string
		pricingQuantity float64
		pricingUnit     string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "valid: quantity with unit",
			pricingQuantity: 10.0,
			pricingUnit:     "Hours",
			expectError:     false,
		},
		{
			name:            "valid: zero quantity without unit",
			pricingQuantity: 0.0,
			pricingUnit:     "",
			expectError:     false,
		},
		{
			name:            "valid: zero quantity with unit",
			pricingQuantity: 0.0,
			pricingUnit:     "Hours",
			expectError:     false,
		},
		{
			name:            "invalid: quantity > 0 without unit (FR-006)",
			pricingQuantity: 10.0,
			pricingUnit:     "",
			expectError:     true,
			errorContains:   "pricing_unit: required when pricing_quantity > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.PricingQuantity = tt.pricingQuantity
			record.PricingUnit = tt.pricingUnit

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_CapacityReservationConsistency tests FR-005.
func TestValidateFocusRecord_CapacityReservationConsistency(t *testing.T) {
	tests := []struct {
		name                      string
		capacityReservationID     string
		capacityReservationStatus pbc.FocusCapacityReservationStatus
		chargeCategory            pbc.FocusChargeCategory
		expectError               bool
		errorContains             string
	}{
		{
			name:                      "valid: no capacity reservation",
			capacityReservationID:     "",
			capacityReservationStatus: pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
			chargeCategory:            pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:               false,
		},
		{
			name:                      "valid: capacity reservation with status for usage",
			capacityReservationID:     "cr-12345",
			capacityReservationStatus: pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
			chargeCategory:            pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:               false,
		},
		{
			name:                      "valid: capacity ID for purchase without status",
			capacityReservationID:     "cr-12345",
			capacityReservationStatus: pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
			chargeCategory:            pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_PURCHASE,
			expectError:               false,
		},
		{
			name:                      "invalid: capacity ID + Usage but no status (FR-005)",
			capacityReservationID:     "cr-12345",
			capacityReservationStatus: pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
			chargeCategory:            pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:               true,
			errorContains:             "capacity_reservation_status: required when",
		},
		{
			name:                      "invalid: status set without ID (FR-005 bidirectional)",
			capacityReservationID:     "",
			capacityReservationStatus: pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
			chargeCategory:            pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			expectError:               true,
			errorContains:             "capacity_reservation_id: required when",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			record.CapacityReservationId = tt.capacityReservationID
			record.CapacityReservationStatus = tt.capacityReservationStatus
			record.ChargeCategory = tt.chargeCategory

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecordWithOptions_AggregateMode tests FR-009 aggregate error collection.
func TestValidateFocusRecordWithOptions_AggregateMode(t *testing.T) {
	t.Run("aggregate mode returns all errors", func(t *testing.T) {
		record := createValidFocusRecord()
		// Set up multiple validation failures
		record.EffectiveCost = 150.0 // Exceeds BilledCost (10.0) - FR-001
		record.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED
		// No CommitmentDiscountId - FR-004
		record.PricingQuantity = 10.0
		record.PricingUnit = "" // FR-006

		opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
		errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

		// Should have exactly 3 errors: cost hierarchy (FR-001) + commitment ID missing (FR-004) + pricing unit missing (FR-006)
		if len(errs) != 3 {
			t.Errorf("Expected 3 errors in aggregate mode, got %d: %v", len(errs), errs)
		}
	})

	t.Run("fail-fast mode returns first error only", func(t *testing.T) {
		record := createValidFocusRecord()
		// Set up multiple validation failures
		record.EffectiveCost = 150.0 // Exceeds BilledCost
		record.PricingQuantity = 10.0
		record.PricingUnit = ""

		opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeFailFast}
		errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

		if len(errs) != 1 {
			t.Errorf("Expected exactly 1 error in fail-fast mode, got %d: %v", len(errs), errs)
		}
	})

	t.Run("aggregate mode with valid record returns empty slice", func(t *testing.T) {
		record := createValidFocusRecord()

		opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
		errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

		if len(errs) != 0 {
			t.Errorf("Expected no errors for valid record, got %d: %v", len(errs), errs)
		}
	})
}

// TestValidateFocusRecordWithOptions_AggregateMode_ErrorsAs verifies all errors in aggregate
// mode are *ValidationError instances with distinct FieldName values.
func TestValidateFocusRecordWithOptions_AggregateMode_ErrorsAs(t *testing.T) {
	record := createValidFocusRecord()
	// Set up 3+ distinct violations
	record.EffectiveCost = 150.0 // Exceeds BilledCost (10.0) - FR-001
	record.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED
	// No CommitmentDiscountId - FR-004
	record.PricingQuantity = 10.0
	record.PricingUnit = "" // FR-006

	opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
	errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

	if len(errs) < 3 {
		t.Fatalf("Expected at least 3 errors, got %d: %v", len(errs), errs)
	}

	// Every error should be extractable as *ValidationError
	fieldNames := make(map[string]bool)
	for i, err := range errs {
		var valErr *pluginsdk.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("errs[%d]: errors.As failed: error %T does not contain *ValidationError", i, err)
			continue
		}
		if valErr.FieldName == "" {
			t.Errorf("errs[%d]: FieldName is empty", i)
		}
		fieldNames[valErr.FieldName] = true
	}

	// Verify distinct field names
	if len(fieldNames) < 3 {
		t.Errorf("Expected at least 3 distinct field names, got %d: %v", len(fieldNames), fieldNames)
	}
}

// TestValidateFocusRecord_ErrorsIs_AllSentinels verifies errors.Is(err, sentinel) returns true
// for all 7 sentinel errors when returned by ValidateFocusRecord through *ValidationError wrapping.
func TestValidateFocusRecord_ErrorsIs_AllSentinels(t *testing.T) {
	tests := []struct {
		name       string
		modifyFunc func(*pbc.FocusCostRecord)
		sentinel   error
	}{
		{
			name: "ErrEffectiveCostExceedsBilledCost",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.BilledCost = 100.0
				r.EffectiveCost = 150.0
				r.ListCost = 200.0
			},
			sentinel: pluginsdk.ErrEffectiveCostExceedsBilledCost,
		},
		{
			name: "ErrListCostLessThanEffectiveCost",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.BilledCost = 100.0
				r.EffectiveCost = 100.0
				r.ListCost = 50.0
			},
			sentinel: pluginsdk.ErrListCostLessThanEffectiveCost,
		},
		{
			name: "ErrCommitmentStatusMissing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CommitmentDiscountId = "ri-12345"
				r.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED
				r.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE
			},
			sentinel: pluginsdk.ErrCommitmentStatusMissing,
		},
		{
			name: "ErrCommitmentIDMissingForStatus",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CommitmentDiscountId = ""
				r.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED
			},
			sentinel: pluginsdk.ErrCommitmentIDMissingForStatus,
		},
		{
			name: "ErrCapacityReservationStatusMissing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CapacityReservationId = "cr-12345"
				r.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED
				r.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE
			},
			sentinel: pluginsdk.ErrCapacityReservationStatusMissing,
		},
		{
			name: "ErrCapacityReservationIDMissing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CapacityReservationId = ""
				r.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED
			},
			sentinel: pluginsdk.ErrCapacityReservationIDMissing,
		},
		{
			name: "ErrPricingUnitMissing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.PricingQuantity = 10.0
				r.PricingUnit = ""
			},
			sentinel: pluginsdk.ErrPricingUnitMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			tt.modifyFunc(record)

			err := pluginsdk.ValidateFocusRecord(record)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !errors.Is(err, tt.sentinel) {
				t.Errorf("errors.Is(err, %s) = false, want true", tt.name)
			}
		})
	}
}

// TestSentinelErrors verifies sentinel errors can be checked with errors.Is.
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrEffectiveCostExceedsBilledCost", pluginsdk.ErrEffectiveCostExceedsBilledCost},
		{"ErrListCostLessThanEffectiveCost", pluginsdk.ErrListCostLessThanEffectiveCost},
		{"ErrCommitmentStatusMissing", pluginsdk.ErrCommitmentStatusMissing},
		{"ErrCommitmentIDMissingForStatus", pluginsdk.ErrCommitmentIDMissingForStatus},
		{"ErrCapacityReservationStatusMissing", pluginsdk.ErrCapacityReservationStatusMissing},
		{"ErrCapacityReservationIDMissing", pluginsdk.ErrCapacityReservationIDMissing},
		{"ErrPricingUnitMissing", pluginsdk.ErrPricingUnitMissing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify sentinel error is not nil
			if tt.sentinel == nil {
				t.Errorf("Sentinel error %s is nil", tt.name)
			}
			// Verify error message is not empty
			if tt.sentinel.Error() == "" {
				t.Errorf("Sentinel error %s has empty message", tt.name)
			}
			// Verify errors.Is works with sentinel (primary use case)
			if !errors.Is(tt.sentinel, tt.sentinel) {
				t.Errorf("errors.Is failed for sentinel %s", tt.name)
			}
		})
	}
}

// TestValidateFocusRecord_ExtremeValues tests handling of IEEE 754 special values (Inf, NaN).
func TestValidateFocusRecord_ExtremeValues(t *testing.T) {
	tests := []struct {
		name          string
		setField      func(*pbc.FocusCostRecord, float64)
		fieldValue    float64
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid billed_cost",
			setField:    func(r *pbc.FocusCostRecord, v float64) { r.BilledCost = v },
			fieldValue:  100.0,
			expectError: false,
		},
		{
			name:          "billed_cost positive infinity",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.BilledCost = v },
			fieldValue:    math.Inf(1),
			expectError:   true,
			errorContains: "must be finite",
		},
		{
			name:          "effective_cost NaN",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.EffectiveCost = v },
			fieldValue:    math.NaN(),
			expectError:   true,
			errorContains: "must be a number",
		},
		{
			name:          "list_cost positive infinity",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.ListCost = v },
			fieldValue:    math.Inf(1),
			expectError:   true,
			errorContains: "must be finite",
		},
		{
			name:          "contracted_cost NaN",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.ContractedCost = v },
			fieldValue:    math.NaN(),
			expectError:   true,
			errorContains: "must be a number",
		},
		{
			name:          "contracted_unit_price negative infinity",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.ContractedUnitPrice = v },
			fieldValue:    math.Inf(-1),
			expectError:   true,
			errorContains: "must be finite",
		},
		{
			name:          "list_unit_price NaN",
			setField:      func(r *pbc.FocusCostRecord, v float64) { r.ListUnitPrice = v },
			fieldValue:    math.NaN(),
			expectError:   true,
			errorContains: "must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			tt.setField(record, tt.fieldValue)

			if !tt.expectError {
				// Sync other cost fields to avoid hierarchy validation errors (FR-001/FR-002)
				// since ValidateFocusRecord checks extreme values before hierarchy.
				record.EffectiveCost = tt.fieldValue
				record.ListCost = tt.fieldValue
				record.ContractedCost = tt.fieldValue
				record.BilledCost = tt.fieldValue
			}

			err := pluginsdk.ValidateFocusRecord(record)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestValidateFocusRecord_ErrorsAs_SentinelErrors verifies errors.As extracts *ValidationError
// with correct FieldName for all 7 sentinel error cases.
func TestValidateFocusRecord_ErrorsAs_SentinelErrors(t *testing.T) {
	tests := []struct {
		name              string
		modifyFunc        func(*pbc.FocusCostRecord)
		expectedFieldName string
		expectedSentinel  error
	}{
		{
			name: "effective_cost exceeds billed_cost",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.BilledCost = 100.0
				r.EffectiveCost = 150.0
				r.ListCost = 200.0
			},
			expectedFieldName: "effective_cost",
			expectedSentinel:  pluginsdk.ErrEffectiveCostExceedsBilledCost,
		},
		{
			name: "list_cost less than effective_cost",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.BilledCost = 100.0
				r.EffectiveCost = 100.0
				r.ListCost = 50.0
			},
			expectedFieldName: "list_cost",
			expectedSentinel:  pluginsdk.ErrListCostLessThanEffectiveCost,
		},
		{
			name: "commitment_discount_status missing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CommitmentDiscountId = "ri-12345"
				r.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED
				r.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE
			},
			expectedFieldName: "commitment_discount_status",
			expectedSentinel:  pluginsdk.ErrCommitmentStatusMissing,
		},
		{
			name: "commitment_discount_id missing for status",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CommitmentDiscountId = ""
				r.CommitmentDiscountStatus = pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED
			},
			expectedFieldName: "commitment_discount_id",
			expectedSentinel:  pluginsdk.ErrCommitmentIDMissingForStatus,
		},
		{
			name: "capacity_reservation_status missing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CapacityReservationId = "cr-12345"
				r.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED
				r.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE
			},
			expectedFieldName: "capacity_reservation_status",
			expectedSentinel:  pluginsdk.ErrCapacityReservationStatusMissing,
		},
		{
			name: "capacity_reservation_id missing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.CapacityReservationId = ""
				r.CapacityReservationStatus = pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED
			},
			expectedFieldName: "capacity_reservation_id",
			expectedSentinel:  pluginsdk.ErrCapacityReservationIDMissing,
		},
		{
			name: "pricing_unit missing",
			modifyFunc: func(r *pbc.FocusCostRecord) {
				r.PricingQuantity = 10.0
				r.PricingUnit = ""
			},
			expectedFieldName: "pricing_unit",
			expectedSentinel:  pluginsdk.ErrPricingUnitMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := createValidFocusRecord()
			tt.modifyFunc(record)

			err := pluginsdk.ValidateFocusRecord(record)
			if err == nil {
				t.Fatal("Expected validation error, got nil")
			}

			// errors.As should extract *ValidationError with correct FieldName
			var valErr *pluginsdk.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("errors.As failed: error %T does not contain *ValidationError", err)
			}
			if valErr.FieldName != tt.expectedFieldName {
				t.Errorf("FieldName = %q, want %q", valErr.FieldName, tt.expectedFieldName)
			}

			// errors.Is should still find the sentinel through Unwrap()
			if !errors.Is(err, tt.expectedSentinel) {
				t.Errorf("errors.Is failed: error does not wrap %v", tt.expectedSentinel)
			}
		})
	}
}

// TestValidateFocusRecord_ErrorsAs_AdHocErrors verifies errors.As extracts *ValidationError
// for ad-hoc errors (mandatory fields, currency, NaN/Inf, nil record).
func TestValidateFocusRecord_ErrorsAs_AdHocErrors(t *testing.T) {
	tests := []struct {
		name              string
		record            *pbc.FocusCostRecord
		expectedFieldName string
	}{
		{
			name:              "nil record",
			record:            nil,
			expectedFieldName: "record",
		},
		{
			name: "mandatory field (provider_name)",
			record: func() *pbc.FocusCostRecord {
				r := createValidFocusRecord()
				//nolint:staticcheck // SA1019: Testing deprecated field
				r.ProviderName = ""
				return r
			}(),
			expectedFieldName: "provider_name",
		},
		{
			name: "currency validation",
			record: func() *pbc.FocusCostRecord {
				r := createValidFocusRecord()
				r.BillingCurrency = "INVALID"
				return r
			}(),
			expectedFieldName: "billing_currency",
		},
		{
			name: "NaN cost",
			record: func() *pbc.FocusCostRecord {
				r := createValidFocusRecord()
				r.BilledCost = math.NaN()
				return r
			}(),
			expectedFieldName: "billed_cost",
		},
		{
			name: "Inf cost",
			record: func() *pbc.FocusCostRecord {
				r := createValidFocusRecord()
				r.BilledCost = math.Inf(1)
				return r
			}(),
			expectedFieldName: "billed_cost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pluginsdk.ValidateFocusRecord(tt.record)
			if err == nil {
				t.Fatal("Expected validation error, got nil")
			}

			var valErr *pluginsdk.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("errors.As failed: error %T does not contain *ValidationError", err)
			}
			if valErr.FieldName != tt.expectedFieldName {
				t.Errorf("FieldName = %q, want %q", valErr.FieldName, tt.expectedFieldName)
			}
		})
	}
}
