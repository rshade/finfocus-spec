package pluginsdk_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestFocusRecordBuilder_Build_HappyPath(t *testing.T) {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	builder := pluginsdk.NewFocusRecordBuilder()
	builder.WithIdentity("AWS", "acc-123", "My Account")
	builder.WithBillingPeriod(billingStart, billingEnd, "USD")
	builder.WithChargePeriod(chargeStart, chargeEnd)
	builder.WithChargeDetails(
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
	)
	builder.WithChargeClassification(
		pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		"EC2 Instance Usage",
		pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
	)
	builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")
	builder.WithFinancials(10.5, 12.0, 10.0, "USD", "inv-001")
	builder.WithUsage(1.0, "Hour")

	record, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if record.GetBillingAccountId() != "acc-123" {
		t.Errorf("Expected BillingAccountId 'acc-123', got %s", record.GetBillingAccountId())
	}
	if record.GetBilledCost() != 10.5 {
		t.Errorf("Expected BilledCost 10.5, got %f", record.GetBilledCost())
	}
	if record.GetServiceCategory() != pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE {
		t.Errorf("Expected Compute category, got %v", record.GetServiceCategory())
	}
}

func TestFocusRecordBuilder_Build_MissingFields(t *testing.T) {
	builder := pluginsdk.NewFocusRecordBuilder()
	// Only set some fields
	builder.WithIdentity("provider-aws", "acc-123", "My Account")

	_, err := builder.Build()
	if err == nil {
		t.Error("Expected error due to missing fields, got nil")
	}
}

func TestFocusRecordBuilder_WithExtension(t *testing.T) {
	builder := createValidBuilder()

	builder.WithExtension("MyKey", "MyValue")
	builder.WithExtension("AnotherKey", "AnotherValue")

	record, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if record.GetExtendedColumns()["MyKey"] != "MyValue" {
		t.Errorf("Expected MyKey=MyValue, got %s", record.GetExtendedColumns()["MyKey"])
	}
	if record.GetExtendedColumns()["AnotherKey"] != "AnotherValue" {
		t.Errorf("Expected AnotherKey=AnotherValue, got %s", record.GetExtendedColumns()["AnotherKey"])
	}
}

// =============================================================================
// FOCUS 1.2 New Column Builder Tests (Phase 5: US3)
// =============================================================================

// createValidBuilder creates a builder with all mandatory fields set for testing new methods.
// Updated to include all 14 FOCUS 1.2 mandatory fields.
func createValidBuilder() *pluginsdk.FocusRecordBuilder {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	builder := pluginsdk.NewFocusRecordBuilder()
	builder.WithIdentity("AWS", "acc-123", "Account Name")
	builder.WithBillingPeriod(billingStart, billingEnd, "USD")
	builder.WithChargePeriod(now.Add(-time.Hour), now)
	builder.WithChargeDetails(
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
	)
	builder.WithChargeClassification(
		pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		"EC2 Instance Usage",
		pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
	)
	builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")
	builder.WithFinancials(10.0, 12.0, 10.0, "USD", "inv-001")
	builder.WithUsage(1.0, "Hour")
	return builder
}

// TestFocusRecordBuilder_WithContractedCost tests the WithContractedCost builder method.
func TestFocusRecordBuilder_WithContractedCost(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		expected float64
	}{
		{"zero cost", 0.0, 0.0},
		{"positive cost", 100.50, 100.50},
		{"large cost", 999999.99, 999999.99},
		{"small cost", 0.001, 0.001},
		{"negative cost (credit)", -50.25, -50.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithContractedCost(tt.cost)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractedCost() != tt.expected {
				t.Errorf("Expected ContractedCost %f, got %f", tt.expected, record.GetContractedCost())
			}
		})
	}
}

// TestFocusRecordBuilder_WithBillingAccountType tests the WithBillingAccountType builder method.
func TestFocusRecordBuilder_WithBillingAccountType(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		expected    string
	}{
		{"empty type", "", ""},
		{"enterprise", "Enterprise", "Enterprise"},
		{"pay as you go", "PayAsYouGo", "PayAsYouGo"},
		{"linked account", "LinkedAccount", "LinkedAccount"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithBillingAccountType(tt.accountType)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetBillingAccountType() != tt.expected {
				t.Errorf("Expected BillingAccountType %q, got %q", tt.expected, record.GetBillingAccountType())
			}
		})
	}
}

// TestFocusRecordBuilder_WithSubAccountType tests the WithSubAccountType builder method.
func TestFocusRecordBuilder_WithSubAccountType(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		expected    string
	}{
		{"empty type", "", ""},
		{"subscription", "Subscription", "Subscription"},
		{"project", "Project", "Project"},
		{"account", "Account", "Account"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithSubAccountType(tt.accountType)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetSubAccountType() != tt.expected {
				t.Errorf("Expected SubAccountType %q, got %q", tt.expected, record.GetSubAccountType())
			}
		})
	}
}

// TestFocusRecordBuilder_WithCapacityReservation tests the WithCapacityReservation builder method.
func TestFocusRecordBuilder_WithCapacityReservation(t *testing.T) {
	tests := []struct {
		name           string
		reservationID  string
		status         pbc.FocusCapacityReservationStatus
		expectedID     string
		expectedStatus pbc.FocusCapacityReservationStatus
	}{
		{
			"empty reservation",
			"",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
			"",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNSPECIFIED,
		},
		{
			"used reservation",
			"cr-12345",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
			"cr-12345",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
		},
		{
			"unused reservation",
			"cr-67890",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNUSED,
			"cr-67890",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_UNUSED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithCapacityReservation(tt.reservationID, tt.status)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetCapacityReservationId() != tt.expectedID {
				t.Errorf("Expected CapacityReservationId %q, got %q", tt.expectedID, record.GetCapacityReservationId())
			}
			if record.GetCapacityReservationStatus() != tt.expectedStatus {
				t.Errorf(
					"Expected CapacityReservationStatus %v, got %v",
					tt.expectedStatus,
					record.GetCapacityReservationStatus(),
				)
			}
		})
	}
}

// TestFocusRecordBuilder_WithCommitmentDiscountDetails tests WithCommitmentDiscountDetails.
func TestFocusRecordBuilder_WithCommitmentDiscountDetails(t *testing.T) {
	tests := []struct {
		name             string
		quantity         float64
		status           pbc.FocusCommitmentDiscountStatus
		discountType     string
		unit             string
		expectedQuantity float64
		expectedStatus   pbc.FocusCommitmentDiscountStatus
		expectedType     string
		expectedUnit     string
	}{
		{
			"empty details",
			0.0,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			"",
			"",
			0.0,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED,
			"",
			"",
		},
		{
			"used savings plan",
			100.0,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			"SavingsPlan",
			"USD/Hour",
			100.0,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			"SavingsPlan",
			"USD/Hour",
		},
		{
			"unused reserved instance",
			50.5,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNUSED,
			"ReservedInstance",
			"Hours",
			50.5,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNUSED,
			"ReservedInstance",
			"Hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			// When setting a non-UNSPECIFIED status, also set CommitmentDiscountId (FR-004)
			if tt.status != pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_UNSPECIFIED {
				builder.WithCommitmentDiscount(
					pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_SPEND,
					"test-commitment-id",
					"TestDiscount",
				)
			}
			builder.WithCommitmentDiscountDetails(tt.quantity, tt.status, tt.discountType, tt.unit)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetCommitmentDiscountQuantity() != tt.expectedQuantity {
				t.Errorf(
					"Expected CommitmentDiscountQuantity %f, got %f",
					tt.expectedQuantity,
					record.GetCommitmentDiscountQuantity(),
				)
			}
			if record.GetCommitmentDiscountStatus() != tt.expectedStatus {
				t.Errorf(
					"Expected CommitmentDiscountStatus %v, got %v",
					tt.expectedStatus,
					record.GetCommitmentDiscountStatus(),
				)
			}
			if record.GetCommitmentDiscountType() != tt.expectedType {
				t.Errorf(
					"Expected CommitmentDiscountType %q, got %q",
					tt.expectedType,
					record.GetCommitmentDiscountType(),
				)
			}
			if record.GetCommitmentDiscountUnit() != tt.expectedUnit {
				t.Errorf(
					"Expected CommitmentDiscountUnit %q, got %q",
					tt.expectedUnit,
					record.GetCommitmentDiscountUnit(),
				)
			}
		})
	}
}

// TestFocusRecordBuilder_WithContractedUnitPrice tests the WithContractedUnitPrice builder method.
func TestFocusRecordBuilder_WithContractedUnitPrice(t *testing.T) {
	tests := []struct {
		name     string
		price    float64
		expected float64
	}{
		{"zero price", 0.0, 0.0},
		{"positive price", 0.0023, 0.0023},
		{"large price", 1500.99, 1500.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithContractedUnitPrice(tt.price)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractedUnitPrice() != tt.expected {
				t.Errorf("Expected ContractedUnitPrice %f, got %f", tt.expected, record.GetContractedUnitPrice())
			}
		})
	}
}

// TestFocusRecordBuilder_WithPricingCurrency tests the WithPricingCurrency builder method.
func TestFocusRecordBuilder_WithPricingCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		expected string
	}{
		{"empty currency", "", ""},
		{"USD", "USD", "USD"},
		{"EUR", "EUR", "EUR"},
		{"GBP", "GBP", "GBP"},
		{"JPY", "JPY", "JPY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithPricingCurrency(tt.currency)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetPricingCurrency() != tt.expected {
				t.Errorf("Expected PricingCurrency %q, got %q", tt.expected, record.GetPricingCurrency())
			}
		})
	}
}

// TestFocusRecordBuilder_WithPricingCurrencyPrices tests the WithPricingCurrencyPrices method.
func TestFocusRecordBuilder_WithPricingCurrencyPrices(t *testing.T) {
	tests := []struct {
		name                string
		contractedUnitPrice float64
		effectiveCost       float64
		listUnitPrice       float64
		expectedContracted  float64
		expectedEffective   float64
		expectedList        float64
	}{
		{"all zeros", 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{"all positive", 10.50, 100.00, 12.00, 10.50, 100.00, 12.00},
		{"mixed values", 0.023, 50.5, 0.025, 0.023, 50.5, 0.025},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithPricingCurrencyPrices(tt.contractedUnitPrice, tt.effectiveCost, tt.listUnitPrice)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetPricingCurrencyContractedUnitPrice() != tt.expectedContracted {
				t.Errorf(
					"Expected PricingCurrencyContractedUnitPrice %f, got %f",
					tt.expectedContracted,
					record.GetPricingCurrencyContractedUnitPrice(),
				)
			}
			if record.GetPricingCurrencyEffectiveCost() != tt.expectedEffective {
				t.Errorf(
					"Expected PricingCurrencyEffectiveCost %f, got %f",
					tt.expectedEffective,
					record.GetPricingCurrencyEffectiveCost(),
				)
			}
			if record.GetPricingCurrencyListUnitPrice() != tt.expectedList {
				t.Errorf(
					"Expected PricingCurrencyListUnitPrice %f, got %f",
					tt.expectedList,
					record.GetPricingCurrencyListUnitPrice(),
				)
			}
		})
	}
}

// TestFocusRecordBuilder_WithPublisher tests the WithPublisher builder method.
func TestFocusRecordBuilder_WithPublisher(t *testing.T) {
	tests := []struct {
		name      string
		publisher string
		expected  string
	}{
		{"empty publisher", "", ""},
		{"AWS", "Amazon Web Services", "Amazon Web Services"},
		{"Azure", "Microsoft", "Microsoft"},
		{"GCP", "Google", "Google"},
		{"third party", "Datadog", "Datadog"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithPublisher(tt.publisher)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			//nolint:staticcheck // SA1019: Testing deprecated WithPublisher backward compatibility
			if record.GetPublisher() != tt.expected {
				//nolint:staticcheck // SA1019: Testing deprecated WithPublisher backward compatibility
				t.Errorf("Expected Publisher %q, got %q", tt.expected, record.GetPublisher())
			}
		})
	}
}

// TestFocusRecordBuilder_WithServiceSubcategory tests the WithServiceSubcategory builder method.
func TestFocusRecordBuilder_WithServiceSubcategory(t *testing.T) {
	tests := []struct {
		name        string
		subcategory string
		expected    string
	}{
		{"empty subcategory", "", ""},
		{"virtual machine", "Virtual Machine", "Virtual Machine"},
		{"container", "Container", "Container"},
		{"serverless", "Serverless", "Serverless"},
		{"kubernetes", "Kubernetes", "Kubernetes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithServiceSubcategory(tt.subcategory)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetServiceSubcategory() != tt.expected {
				t.Errorf("Expected ServiceSubcategory %q, got %q", tt.expected, record.GetServiceSubcategory())
			}
		})
	}
}

// TestFocusRecordBuilder_WithSkuDetails tests the WithSkuDetails builder method.
func TestFocusRecordBuilder_WithSkuDetails(t *testing.T) {
	tests := []struct {
		name            string
		meter           string
		priceDetails    string
		expectedMeter   string
		expectedDetails string
	}{
		{"empty details", "", "", "", ""},
		{"compute meter", "compute-hours", "On-Demand", "compute-hours", "On-Demand"},
		{"storage meter", "storage-gb-month", "Standard tier", "storage-gb-month", "Standard tier"},
		{"network meter", "data-transfer-gb", "Outbound", "data-transfer-gb", "Outbound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithSkuDetails(tt.meter, tt.priceDetails)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetSkuMeter() != tt.expectedMeter {
				t.Errorf("Expected SkuMeter %q, got %q", tt.expectedMeter, record.GetSkuMeter())
			}
			if record.GetSkuPriceDetails() != tt.expectedDetails {
				t.Errorf("Expected SkuPriceDetails %q, got %q", tt.expectedDetails, record.GetSkuPriceDetails())
			}
		})
	}
}

// TestFocusRecordBuilder_ChainNewMethods tests chaining multiple new builder methods.
func TestFocusRecordBuilder_ChainNewMethods(t *testing.T) {
	builder := createValidBuilder()

	// Chain all new methods
	// Note: WithCommitmentDiscount must be called before WithCommitmentDiscountDetails (FR-004)
	record, err := builder.
		WithContractedCost(100.50).
		WithBillingAccountType("Enterprise").
		WithSubAccountType("Subscription").
		WithCapacityReservation("cr-123", pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED).
		WithCommitmentDiscount(pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_SPEND, "cd-123", "TestDiscount").
		WithCommitmentDiscountDetails(50.0, pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED, "SavingsPlan", "USD/Hour").
		WithContractedUnitPrice(0.05).
		WithPricingCurrency("EUR").
		WithPricingCurrencyPrices(0.045, 90.0, 0.055).
		WithPublisher("Amazon Web Services").
		WithServiceSubcategory("Virtual Machine").
		WithSkuDetails("compute-hours", "On-Demand").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify all fields were set
	if record.GetContractedCost() != 100.50 {
		t.Errorf("Expected ContractedCost 100.50, got %f", record.GetContractedCost())
	}
	if record.GetBillingAccountType() != "Enterprise" {
		t.Errorf("Expected BillingAccountType 'Enterprise', got %q", record.GetBillingAccountType())
	}
	if record.GetSubAccountType() != "Subscription" {
		t.Errorf("Expected SubAccountType 'Subscription', got %q", record.GetSubAccountType())
	}
	if record.GetCapacityReservationId() != "cr-123" {
		t.Errorf("Expected CapacityReservationId 'cr-123', got %q", record.GetCapacityReservationId())
	}
	if record.GetCommitmentDiscountQuantity() != 50.0 {
		t.Errorf("Expected CommitmentDiscountQuantity 50.0, got %f", record.GetCommitmentDiscountQuantity())
	}
	if record.GetContractedUnitPrice() != 0.05 {
		t.Errorf("Expected ContractedUnitPrice 0.05, got %f", record.GetContractedUnitPrice())
	}
	if record.GetPricingCurrency() != "EUR" {
		t.Errorf("Expected PricingCurrency 'EUR', got %q", record.GetPricingCurrency())
	}
	//nolint:staticcheck // SA1019: Testing deprecated WithPublisher backward compatibility
	if record.GetPublisher() != "Amazon Web Services" {
		//nolint:staticcheck // SA1019: Testing deprecated WithPublisher backward compatibility
		t.Errorf("Expected Publisher 'Amazon Web Services', got %q", record.GetPublisher())
	}
	if record.GetServiceSubcategory() != "Virtual Machine" {
		t.Errorf("Expected ServiceSubcategory 'Virtual Machine', got %q", record.GetServiceSubcategory())
	}
	if record.GetSkuMeter() != "compute-hours" {
		t.Errorf("Expected SkuMeter 'compute-hours', got %q", record.GetSkuMeter())
	}
}

// =============================================================================
// Benchmarks for Builder Operations
// =============================================================================

// BenchmarkNewFocusRecordBuilder measures builder creation performance.
func BenchmarkNewFocusRecordBuilder(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.NewFocusRecordBuilder()
	}
}

// BenchmarkFocusRecordBuilder_Build measures the full build cycle performance.
func BenchmarkFocusRecordBuilder_Build(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		builder := createValidBuilder()
		_, _ = builder.Build()
	}
}

// BenchmarkFocusRecordBuilder_ChainedBuild measures chained method performance.
func BenchmarkFocusRecordBuilder_ChainedBuild(b *testing.B) {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, _ = pluginsdk.NewFocusRecordBuilder().
			WithIdentity("AWS", "acc-123", "Account").
			WithBillingPeriod(billingStart, billingEnd, "USD").
			WithChargePeriod(now.Add(-time.Hour), now).
			WithChargeDetails(
				pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
				pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			).
			WithChargeClassification(
				pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
				"Usage",
				pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
			).
			WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "EC2").
			WithFinancials(10.0, 12.0, 10.0, "USD", "inv-001").
			WithUsage(1.0, "Hour").
			Build()
	}
}

// BenchmarkFocusRecordBuilder_FullRecord measures building a complete 57-column record.
func BenchmarkFocusRecordBuilder_FullRecord(b *testing.B) {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		builder := pluginsdk.NewFocusRecordBuilder()

		// Mandatory columns
		builder.WithIdentity("AWS", "acc-123", "Account")
		builder.WithBillingPeriod(billingStart, billingEnd, "USD")
		builder.WithChargePeriod(now.Add(-time.Hour), now)
		builder.WithChargeDetails(
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		)
		builder.WithChargeClassification(
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			"Usage",
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		)
		builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "EC2")
		builder.WithFinancials(10.0, 12.0, 10.0, "USD", "inv-001")
		builder.WithUsage(1.0, "Hour")

		// Conditional columns
		builder.WithSubAccount("sub-123", "SubAccount")
		builder.WithBillingAccountType("Enterprise")
		builder.WithSubAccountType("Subscription")
		builder.WithLocation("us-east-1", "US East", "us-east-1a")
		builder.WithResource("i-123", "instance", "m5.large")
		builder.WithSKU("sku-123", "price-123")
		builder.WithSkuDetails("meter", "details")
		builder.WithPricing(100.0, "Hours", 0.10)
		builder.WithContractedCost(10.0)
		builder.WithContractedUnitPrice(0.10)
		builder.WithPricingCurrency("USD")
		builder.WithPricingCurrencyPrices(0.10, 10.0, 0.12)
		builder.WithPublisher("AWS")
		builder.WithServiceSubcategory("VM")
		builder.WithCommitmentDiscount(
			pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_USAGE,
			"ri-123", "Reserved Instance",
		)
		builder.WithCommitmentDiscountDetails(
			100.0,
			pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
			"Standard RI", "Hours",
		)
		builder.WithCapacityReservation(
			"cr-123",
			pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
		)
		builder.WithInvoice("inv-001", "AWS Inc")
		builder.WithTags(map[string]string{"env": "prod", "team": "platform"})
		builder.WithExtension("custom", "value")

		_, _ = builder.Build()
	}
}

// =============================================================================
// FOCUS 1.3 New Column Builder Tests
// =============================================================================

// TestFocusRecordBuilder_WithAllocation tests the WithAllocation builder method.
func TestFocusRecordBuilder_WithAllocation(t *testing.T) {
	tests := []struct {
		name                  string
		methodID              string
		methodDetails         string
		expectedMethodID      string
		expectedMethodDetails string
	}{
		{"empty allocation", "", "", "", ""},
		{
			"cpu weighted",
			"proportional-cpu",
			"CPU-weighted proportional",
			"proportional-cpu",
			"CPU-weighted proportional",
		},
		{"memory based", "memory-split", "Memory usage based", "memory-split", "Memory usage based"},
		{"even split", "even-divide", "Equal distribution", "even-divide", "Equal distribution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			// Must also set resource ID to satisfy validation
			builder.WithAllocation(tt.methodID, tt.methodDetails)
			if tt.methodID != "" {
				builder.WithAllocatedResource("resource-123", "Frontend Pod")
			}
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetAllocatedMethodId() != tt.expectedMethodID {
				t.Errorf("Expected AllocatedMethodId %q, got %q", tt.expectedMethodID, record.GetAllocatedMethodId())
			}
			if record.GetAllocatedMethodDetails() != tt.expectedMethodDetails {
				t.Errorf(
					"Expected AllocatedMethodDetails %q, got %q",
					tt.expectedMethodDetails,
					record.GetAllocatedMethodDetails(),
				)
			}
		})
	}
}

// TestFocusRecordBuilder_WithAllocatedResource tests the WithAllocatedResource builder method.
func TestFocusRecordBuilder_WithAllocatedResource(t *testing.T) {
	tests := []struct {
		name                 string
		resourceID           string
		resourceName         string
		expectedResourceID   string
		expectedResourceName string
	}{
		{"empty resource", "", "", "", ""},
		{"k8s pod", "pod-frontend-abc123", "frontend-service", "pod-frontend-abc123", "frontend-service"},
		{"database instance", "db-instance-xyz", "production-db", "db-instance-xyz", "production-db"},
		{"compute instance", "vm-123", "web-server", "vm-123", "web-server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithAllocatedResource(tt.resourceID, tt.resourceName)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetAllocatedResourceId() != tt.expectedResourceID {
				t.Errorf(
					"Expected AllocatedResourceId %q, got %q",
					tt.expectedResourceID,
					record.GetAllocatedResourceId(),
				)
			}
			if record.GetAllocatedResourceName() != tt.expectedResourceName {
				t.Errorf(
					"Expected AllocatedResourceName %q, got %q",
					tt.expectedResourceName,
					record.GetAllocatedResourceName(),
				)
			}
		})
	}
}

// TestFocusRecordBuilder_WithAllocatedTags tests the WithAllocatedTags builder method.
func TestFocusRecordBuilder_WithAllocatedTags(t *testing.T) {
	tests := []struct {
		name         string
		tags         map[string]string
		expectedTags map[string]string
	}{
		{"nil tags", nil, nil},
		{"empty tags", map[string]string{}, map[string]string{}},
		{"single tag", map[string]string{"team": "platform"}, map[string]string{"team": "platform"}},
		{
			"multiple tags",
			map[string]string{"team": "platform", "env": "production", "cost-center": "123"},
			map[string]string{"team": "platform", "env": "production", "cost-center": "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithAllocatedTags(tt.tags)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			allocatedTags := record.GetAllocatedTags()
			if tt.expectedTags == nil {
				if len(allocatedTags) != 0 {
					t.Errorf("Expected nil/empty AllocatedTags, got %v", allocatedTags)
				}
			} else {
				for k, v := range tt.expectedTags {
					if allocatedTags[k] != v {
						t.Errorf("Expected AllocatedTags[%q]=%q, got %q", k, v, allocatedTags[k])
					}
				}
			}
		})
	}
}

// TestFocusRecordBuilder_AllocationValidation tests that AllocatedMethodId requires AllocatedResourceId.
//
//nolint:gocognit // table-driven test with structured error assertions requires nested checks
func TestFocusRecordBuilder_AllocationValidation(t *testing.T) {
	tests := []struct {
		name          string
		methodID      string
		resourceID    string
		expectError   bool
		errorContains string
		expectedField string
	}{
		{"no allocation - valid", "", "", false, "", ""},
		{
			"method only - invalid", "method-123", "", true,
			"allocated_resource_id: required when allocated_method_id set",
			"allocated_resource_id",
		},
		{"resource only - valid", "", "resource-123", false, "", ""},
		{"both set - valid", "method-123", "resource-123", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			if tt.methodID != "" {
				builder.WithAllocation(tt.methodID, "Some method details")
			}
			if tt.resourceID != "" {
				builder.WithAllocatedResource(tt.resourceID, "Some resource")
			}

			_, err := builder.Build()

			if !tt.expectError {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
			}
			var valErr *pluginsdk.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("Expected *pluginsdk.ValidationError, got %T", err)
			}
			if valErr.FieldName != tt.expectedField {
				t.Errorf("Expected FieldName %q, got %q", tt.expectedField, valErr.FieldName)
			}
		})
	}
}

// TestFocusRecordBuilder_ChainFocus13Allocation tests chaining FOCUS 1.3 allocation methods.
func TestFocusRecordBuilder_ChainFocus13Allocation(t *testing.T) {
	tags := map[string]string{"team": "platform", "environment": "production"}

	record, err := createValidBuilder().
		WithAllocation("proportional-cpu", "CPU-weighted proportional allocation").
		WithAllocatedResource("pod-frontend-abc123", "frontend-service").
		WithAllocatedTags(tags).
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if record.GetAllocatedMethodId() != "proportional-cpu" {
		t.Errorf("Expected AllocatedMethodId 'proportional-cpu', got %q", record.GetAllocatedMethodId())
	}
	if record.GetAllocatedMethodDetails() != "CPU-weighted proportional allocation" {
		t.Errorf("Expected AllocatedMethodDetails, got %q", record.GetAllocatedMethodDetails())
	}
	if record.GetAllocatedResourceId() != "pod-frontend-abc123" {
		t.Errorf("Expected AllocatedResourceId 'pod-frontend-abc123', got %q", record.GetAllocatedResourceId())
	}
	if record.GetAllocatedResourceName() != "frontend-service" {
		t.Errorf("Expected AllocatedResourceName 'frontend-service', got %q", record.GetAllocatedResourceName())
	}
	if record.GetAllocatedTags()["team"] != "platform" {
		t.Errorf("Expected AllocatedTags[team]='platform', got %q", record.GetAllocatedTags()["team"])
	}
}

// =============================================================================
// FOCUS 1.3 Allocation Benchmarks
// =============================================================================

// BenchmarkFocusRecordBuilder_WithAllocation measures allocation method performance.
func BenchmarkFocusRecordBuilder_WithAllocation(b *testing.B) {
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithAllocation("method-id", "method details")
	}
}

// BenchmarkFocusRecordBuilder_WithAllocatedResource measures allocated resource method performance.
func BenchmarkFocusRecordBuilder_WithAllocatedResource(b *testing.B) {
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithAllocatedResource("resource-id", "resource name")
	}
}

// BenchmarkFocusRecordBuilder_WithAllocatedTags measures allocated tags method performance.
func BenchmarkFocusRecordBuilder_WithAllocatedTags(b *testing.B) {
	tags := map[string]string{"team": "platform", "env": "prod"}
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithAllocatedTags(tags)
	}
}

// =============================================================================
// FOCUS 1.3 Service/Host Provider Tests (Phase 4 - User Story 2)
// =============================================================================

// TestFocusRecordBuilder_WithServiceProvider tests the WithServiceProvider builder method.
func TestFocusRecordBuilder_WithServiceProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{"empty provider", ""},
		{"aws", "Amazon Web Services"},
		{"azure marketplace isv", "Datadog Inc"},
		{"gcp marketplace isv", "MongoDB Atlas"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithServiceProvider(tt.provider)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetServiceProviderName() != tt.provider {
				t.Errorf("Expected ServiceProviderName %q, got %q", tt.provider, record.GetServiceProviderName())
			}
		})
	}
}

// TestFocusRecordBuilder_WithHostProvider tests the WithHostProvider builder method.
func TestFocusRecordBuilder_WithHostProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{"empty provider", ""},
		{"aws", "Amazon Web Services"},
		{"azure", "Microsoft Azure"},
		{"gcp", "Google Cloud Platform"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithHostProvider(tt.provider)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetHostProviderName() != tt.provider {
				t.Errorf("Expected HostProviderName %q, got %q", tt.provider, record.GetHostProviderName())
			}
		})
	}
}

// TestFocusRecordBuilder_ProviderDeprecation tests deprecation warning behavior
// when both old (provider_name) and new (service_provider_name) fields are set.
func TestFocusRecordBuilder_ProviderDeprecation(t *testing.T) {
	tests := []struct {
		name                string
		providerName        string // deprecated field via WithIdentity
		serviceProviderName string // new field via WithServiceProvider
		expectedProvider    string // what should be in service_provider_name
	}{
		{
			name:                "only deprecated provider_name set",
			providerName:        "AWS",
			serviceProviderName: "",
			expectedProvider:    "", // service_provider_name stays empty (backward compat)
		},
		{
			name:                "only new service_provider_name set",
			providerName:        "AWS",
			serviceProviderName: "Amazon Web Services",
			expectedProvider:    "Amazon Web Services",
		},
		{
			name:                "both set - new field takes precedence",
			providerName:        "AWS",
			serviceProviderName: "Amazon Web Services Inc",
			expectedProvider:    "Amazon Web Services Inc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			// WithIdentity sets the deprecated provider_name
			builder.WithIdentity(tt.providerName, "billing-123", "Test Account")
			if tt.serviceProviderName != "" {
				builder.WithServiceProvider(tt.serviceProviderName)
			}
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetServiceProviderName() != tt.expectedProvider {
				t.Errorf(
					"Expected ServiceProviderName %q, got %q",
					tt.expectedProvider,
					record.GetServiceProviderName(),
				)
			}
			// deprecated field should still be set for backward compatibility
			//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
			if record.GetProviderName() != tt.providerName {
				//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
				t.Errorf("Expected ProviderName (deprecated) %q, got %q", tt.providerName, record.GetProviderName())
			}
		})
	}
}

// TestFocusRecordBuilder_PublisherDeprecation tests deprecation warning behavior
// when both old (publisher) and new (host_provider_name) fields are set.
func TestFocusRecordBuilder_PublisherDeprecation(t *testing.T) {
	tests := []struct {
		name             string
		publisher        string // deprecated field via WithPublisher
		hostProviderName string // new field via WithHostProvider
		expectedHost     string // what should be in host_provider_name
	}{
		{
			name:             "only deprecated publisher set",
			publisher:        "AWS",
			hostProviderName: "",
			expectedHost:     "", // host_provider_name stays empty (backward compat)
		},
		{
			name:             "only new host_provider_name set",
			publisher:        "",
			hostProviderName: "Amazon Web Services",
			expectedHost:     "Amazon Web Services",
		},
		{
			name:             "both set - new field takes precedence",
			publisher:        "AWS",
			hostProviderName: "Amazon Web Services",
			expectedHost:     "Amazon Web Services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			if tt.publisher != "" {
				builder.WithPublisher(tt.publisher)
			}
			if tt.hostProviderName != "" {
				builder.WithHostProvider(tt.hostProviderName)
			}
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetHostProviderName() != tt.expectedHost {
				t.Errorf("Expected HostProviderName %q, got %q", tt.expectedHost, record.GetHostProviderName())
			}
			//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
			if record.GetPublisher() != tt.publisher {
				//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
				t.Errorf("Expected Publisher (deprecated) %q, got %q", tt.publisher, record.GetPublisher())
			}
		})
	}
}

// TestFocusRecordBuilder_ChainFocus13Providers tests method chaining for provider fields.
func TestFocusRecordBuilder_ChainFocus13Providers(t *testing.T) {
	builder := createValidBuilder()
	record, err := builder.
		WithServiceProvider("Datadog Inc").
		WithHostProvider("Amazon Web Services").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if record.GetServiceProviderName() != "Datadog Inc" {
		t.Errorf("Expected ServiceProviderName %q, got %q", "Datadog Inc", record.GetServiceProviderName())
	}
	if record.GetHostProviderName() != "Amazon Web Services" {
		t.Errorf("Expected HostProviderName %q, got %q", "Amazon Web Services", record.GetHostProviderName())
	}
}

// =============================================================================
// FOCUS 1.3 Provider Benchmarks
// =============================================================================

// BenchmarkFocusRecordBuilder_WithServiceProvider measures service provider method performance.
func BenchmarkFocusRecordBuilder_WithServiceProvider(b *testing.B) {
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithServiceProvider("Amazon Web Services")
	}
}

// BenchmarkFocusRecordBuilder_WithHostProvider measures host provider method performance.
func BenchmarkFocusRecordBuilder_WithHostProvider(b *testing.B) {
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithHostProvider("Amazon Web Services")
	}
}

// =============================================================================
// FOCUS 1.3 Contract Applied Tests (Phase 6 - User Story 4)
// =============================================================================

// TestFocusRecordBuilder_WithContractApplied tests the WithContractApplied builder method.
func TestFocusRecordBuilder_WithContractApplied(t *testing.T) {
	tests := []struct {
		name         string
		commitmentID string
	}{
		{"empty commitment id", ""},
		{"reserved instance", "commitment-ri-123456"},
		{"savings plan", "commitment-sp-789"},
		{"enterprise agreement", "contract-ea-2024-001"},
		{"arbitrary string", "any-opaque-reference-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithContractApplied(tt.commitmentID)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractApplied() != tt.commitmentID {
				t.Errorf("Expected ContractApplied %q, got %q", tt.commitmentID, record.GetContractApplied())
			}
		})
	}
}

// TestFocusRecordBuilder_ContractApplied_NoValidation verifies no cross-dataset validation.
// ContractApplied accepts any string value without verifying it exists in the
// ContractCommitment dataset (it's an opaque reference).
func TestFocusRecordBuilder_ContractApplied_NoValidation(t *testing.T) {
	tests := []struct {
		name         string
		commitmentID string
	}{
		{"non-existent commitment", "does-not-exist-in-any-dataset"},
		{"garbage string", "🚀💰📊"},
		{"uuid format", "550e8400-e29b-41d4-a716-446655440000"},
		{"numeric", "12345678901234567890"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidBuilder()
			builder.WithContractApplied(tt.commitmentID)
			record, err := builder.Build()
			// No validation error should occur - any string is accepted
			if err != nil {
				t.Errorf("Unexpected error for ContractApplied %q: %v", tt.commitmentID, err)
			}
			if record.GetContractApplied() != tt.commitmentID {
				t.Errorf("Expected ContractApplied %q, got %q", tt.commitmentID, record.GetContractApplied())
			}
		})
	}
}

// TestFocusRecordBuilder_ChainFocus13ContractApplied tests method chaining with contract applied.
func TestFocusRecordBuilder_ChainFocus13ContractApplied(t *testing.T) {
	builder := createValidBuilder()
	record, err := builder.
		WithContractApplied("commitment-enterprise-2024").
		WithServiceProvider("AWS").
		WithHostProvider("Amazon Web Services").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if record.GetContractApplied() != "commitment-enterprise-2024" {
		t.Errorf("Expected ContractApplied %q, got %q", "commitment-enterprise-2024", record.GetContractApplied())
	}
}

// BenchmarkFocusRecordBuilder_WithContractApplied measures contract applied method performance.
func BenchmarkFocusRecordBuilder_WithContractApplied(b *testing.B) {
	builder := createValidBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithContractApplied("commitment-ri-123456")
	}
}

// =============================================================================
// FOCUS 1.3 Backward Compatibility Tests (Phase 7 - User Story 5)
// =============================================================================

// TestFocusRecordBuilder_BackwardCompatibility_Focus12Only verifies that
// FOCUS 1.2-only records (without any FOCUS 1.3 fields) still validate and work correctly.
func TestFocusRecordBuilder_BackwardCompatibility_Focus12Only(t *testing.T) {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	// Build a pure FOCUS 1.2 record - no new FOCUS 1.3 fields
	record, err := pluginsdk.NewFocusRecordBuilder().
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
			"EC2 m5.large usage",
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		).
		WithUsage(100, "Hours").
		WithFinancials(100.00, 100.00, 95.00, "USD", "INV-2024-001").
		WithContractedCost(95.00).
		Build()

	if err != nil {
		t.Fatalf("FOCUS 1.2-only record should validate successfully: %v", err)
	}

	// Verify FOCUS 1.3 fields are empty/zero (default values)
	if record.GetServiceProviderName() != "" {
		t.Errorf("Expected empty ServiceProviderName for FOCUS 1.2 record, got %q", record.GetServiceProviderName())
	}
	if record.GetHostProviderName() != "" {
		t.Errorf("Expected empty HostProviderName for FOCUS 1.2 record, got %q", record.GetHostProviderName())
	}
	if record.GetAllocatedMethodId() != "" {
		t.Errorf("Expected empty AllocatedMethodId for FOCUS 1.2 record, got %q", record.GetAllocatedMethodId())
	}
	if record.GetContractApplied() != "" {
		t.Errorf("Expected empty ContractApplied for FOCUS 1.2 record, got %q", record.GetContractApplied())
	}

	// Verify FOCUS 1.2 fields are correctly set
	//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
	if record.GetProviderName() != "AWS" {
		//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		t.Errorf("Expected ProviderName %q, got %q", "AWS", record.GetProviderName())
	}
	if record.GetBilledCost() != 100.00 {
		t.Errorf("Expected BilledCost 100.00, got %f", record.GetBilledCost())
	}
}

// TestFocusRecordBuilder_BackwardCompatibility_DeprecatedFieldsWork verifies
// that deprecated fields (provider_name, publisher) continue to work.
func TestFocusRecordBuilder_BackwardCompatibility_DeprecatedFieldsWork(t *testing.T) {
	// Test that WithIdentity still sets provider_name
	builder := createValidBuilder()
	builder.WithIdentity("AWS", "123", "Test")
	record, err := builder.Build()
	if err != nil {
		t.Fatalf("Build with deprecated provider_name failed: %v", err)
	}
	//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
	if record.GetProviderName() != "AWS" {
		//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
		t.Errorf("Expected ProviderName (deprecated) %q, got %q", "AWS", record.GetProviderName())
	}

	// Test that WithPublisher still sets publisher
	builder2 := createValidBuilder()
	builder2.WithPublisher("Microsoft")
	record2, err := builder2.Build()
	if err != nil {
		t.Fatalf("Build with deprecated publisher failed: %v", err)
	}
	//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
	if record2.GetPublisher() != "Microsoft" {
		//nolint:staticcheck // SA1019: Testing deprecated publisher backward compatibility
		t.Errorf("Expected Publisher (deprecated) %q, got %q", "Microsoft", record2.GetPublisher())
	}
}

// TestFocusRecordBuilder_BackwardCompatibility_MixedFocus12And13 verifies
// that records can use both FOCUS 1.2 and 1.3 fields together.
func TestFocusRecordBuilder_BackwardCompatibility_MixedFocus12And13(t *testing.T) {
	now := time.Now()
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now

	// Build a record with both FOCUS 1.2 and 1.3 fields
	record, err := pluginsdk.NewFocusRecordBuilder().
		// FOCUS 1.2 fields
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
			"EC2 m5.large usage",
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		).
		WithUsage(100, "Hours").
		WithFinancials(100.00, 100.00, 95.00, "USD", "INV-2024-001").
		WithContractedCost(95.00).
		// FOCUS 1.3 fields
		WithServiceProvider("Amazon Web Services").
		WithHostProvider("Amazon Web Services").
		WithAllocation("proportional-cpu", "CPU-weighted").
		WithAllocatedResource("pod-123", "frontend").
		WithContractApplied("commitment-ri-123").
		Build()

	if err != nil {
		t.Fatalf("Mixed FOCUS 1.2/1.3 record should validate: %v", err)
	}

	// Verify both FOCUS 1.2 and 1.3 fields are set
	//nolint:staticcheck // SA1019: Testing deprecated provider_name backward compatibility
	if record.GetProviderName() != "AWS" {
		t.Errorf("FOCUS 1.2 ProviderName missing")
	}
	if record.GetServiceProviderName() != "Amazon Web Services" {
		t.Errorf("FOCUS 1.3 ServiceProviderName missing")
	}
	if record.GetAllocatedMethodId() != "proportional-cpu" {
		t.Errorf("FOCUS 1.3 AllocatedMethodId missing")
	}
}

// TestFocusRecordBuilder_BackwardCompatibility_NewFieldsDefaultValues verifies
// that all new FOCUS 1.3 fields have sensible default/zero values.
func TestFocusRecordBuilder_BackwardCompatibility_NewFieldsDefaultValues(t *testing.T) {
	builder := createValidBuilder()
	record, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// All FOCUS 1.3 fields should have zero/empty values by default
	tests := []struct {
		name  string
		value string
	}{
		{"ServiceProviderName", record.GetServiceProviderName()},
		{"HostProviderName", record.GetHostProviderName()},
		{"AllocatedMethodId", record.GetAllocatedMethodId()},
		{"AllocatedMethodDetails", record.GetAllocatedMethodDetails()},
		{"AllocatedResourceId", record.GetAllocatedResourceId()},
		{"AllocatedResourceName", record.GetAllocatedResourceName()},
		{"ContractApplied", record.GetContractApplied()},
	}

	for _, tt := range tests {
		if tt.value != "" {
			t.Errorf("%s should be empty by default, got %q", tt.name, tt.value)
		}
	}

	// AllocatedTags should be nil or empty map
	if len(record.GetAllocatedTags()) != 0 {
		t.Errorf("AllocatedTags should be empty by default, got %v", record.GetAllocatedTags())
	}
}
