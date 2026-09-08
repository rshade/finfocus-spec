// focus_example.go demonstrates a comprehensive FOCUS 1.2 cost record
// using all 57 columns defined in the FinOps FOCUS 1.2 specification.
//
// This example shows how to use the FocusRecordBuilder to construct
// a complete cost record with all mandatory, recommended, and conditional columns.
// It is not a gRPC plugin binary. Plugin mains should call pluginsdk.Run():
//
//	os.Exit(pluginsdk.Run(pluginsdk.ServeConfig{
//	    Plugin:     &MyPlugin{},
//	    PluginInfo: pluginsdk.NewPluginInfo("my-plugin", "v1.0.0"),
//	}))
//
// Reference: https://focus.finops.org/focus-specification/v1-2/
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Example cost values for demonstration.
// Note: ContractedCost MUST equal ContractedUnitPrice × PricingQuantity per FOCUS 1.2.
const (
	exampleBilledCost       = 73.00 // Monthly billed cost
	exampleListCost         = 80.00 // List price before discounts
	exampleEffectiveCost    = 70.00 // Effective cost after discounts
	exampleContractedCost   = 64.97 // Contracted cost (= 0.089 × 730.0)
	exampleListUnitPrice    = 0.10  // List price per hour
	exampleContractedPrice  = 0.089 // Contracted unit price
	examplePricingQuantity  = 730.0 // Hours in a month
	exampleConsumedQuantity = 720.0 // Actual hours consumed
	exampleDiscountQuantity = 730.0 // Commitment discount quantity
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create FOCUS 1.2 compliant cost record using the builder pattern
	record, err := buildCompleteFocusRecord()
	if err != nil {
		logger.Error("Failed to build record", "error", err)
		os.Exit(1)
	}

	//nolint:staticcheck // SA1019: Demonstrating deprecated provider_name access for backward compatibility example
	logger.Info("Generated Complete FOCUS 1.2 Record",
		"provider", record.GetProviderName(),
		"service", record.GetServiceName(),
		"resource_id", record.GetResourceId(),
		"billed_cost", record.GetBilledCost(),
		"effective_cost", record.GetEffectiveCost(),
		"contracted_cost", record.GetContractedCost(),
		"currency", record.GetBillingCurrency(),
		"charge_category", record.GetChargeCategory().String(),
		"pricing_category", record.GetPricingCategory().String(),
		"commitment_discount_status", record.GetCommitmentDiscountStatus().String(),
	)
}

// buildCompleteFocusRecord builds a complete FOCUS 1.2 record with all 57 columns.
func buildCompleteFocusRecord() (*pbc.FocusCostRecord, error) {
	builder := pluginsdk.NewFocusRecordBuilder()

	// Set mandatory columns (14 columns)
	setMandatoryColumns(builder)

	// Set conditional columns (42 columns)
	setConditionalColumns(builder)

	return builder.Build()
}

// setMandatoryColumns configures all 14 mandatory FOCUS 1.2 columns.
func setMandatoryColumns(builder *pluginsdk.FocusRecordBuilder) {
	now := time.Now()
	chargeStart := now.Add(-24 * time.Hour)
	chargeEnd := now
	billingStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	// Identity & Hierarchy (FOCUS 1.2 Section 2.1)
	builder.WithIdentity("AWS", "123456789012", "Production Account")

	// Billing Period (FOCUS 1.2 Section 2.2)
	builder.WithBillingPeriod(billingStart, billingEnd, "USD")

	// Charge Period (FOCUS 1.2 Section 2.3)
	builder.WithChargePeriod(chargeStart, chargeEnd)

	// Charge Details (FOCUS 1.2 Section 2.4)
	builder.WithChargeDetails(
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
	)

	// ChargeDescription, ChargeClass, ChargeFrequency (Section 2.4)
	builder.WithChargeClassification(
		pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		"Amazon EC2 m5.large instance usage - US East (N. Virginia)",
		pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
	)

	// Service Details (FOCUS 1.2 Section 2.6)
	builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")

	// Financial Amounts (FOCUS 1.2 Section 2.10)
	builder.WithFinancials(exampleBilledCost, exampleListCost, exampleEffectiveCost, "USD", "INV-2025-11-001")
	builder.WithContractedCost(exampleContractedCost)
}

// setConditionalColumns configures all 42 conditional FOCUS 1.2 columns.
func setConditionalColumns(builder *pluginsdk.FocusRecordBuilder) {
	// Identity - Conditional Fields
	builder.WithSubAccount("111122223333", "Development Workloads")
	builder.WithBillingAccountType("Organization")
	builder.WithSubAccountType("LinkedAccount")

	// Location (Section 2.9)
	builder.WithLocation("us-east-1", "US East (N. Virginia)", "us-east-1a")

	// Resource Details (Section 2.7)
	builder.WithResource("i-0abc123def456789", "web-server-prod-01", "m5.large")

	// SKU Details (Section 2.8)
	builder.WithSKU("DQ578CGN99KG6ECF", "DQ578CGN99KG6ECF.JRTCKXETXF.6YS6EN2CT7")
	builder.WithSkuDetails("BoxUsage:m5.large", "Linux/UNIX, US East (N. Virginia), m5.large")

	// Pricing Details (Section 2.5)
	builder.WithPricing(examplePricingQuantity, "Hours", exampleListUnitPrice)
	builder.WithContractedUnitPrice(exampleContractedPrice)
	builder.WithPricingCurrency("USD")
	builder.WithPricingCurrencyPrices(exampleContractedPrice, exampleEffectiveCost, exampleListUnitPrice)

	// Consumption/Usage (Section 2.11)
	builder.WithUsage(exampleConsumedQuantity, "Hours")

	// Service - Conditional Fields
	//nolint:staticcheck // SA1019: Demonstrating deprecated WithPublisher for backward compatibility example
	builder.WithPublisher("Amazon Web Services")
	builder.WithServiceSubcategory("Virtual Machine")

	// Commitment Discounts (Section 2.12)
	builder.WithCommitmentDiscount(
		pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_USAGE,
		"ri-0abc123456789def0",
		"EC2 m5.large 1-Year Reserved Instance",
	)
	builder.WithCommitmentDiscountDetails(
		exampleDiscountQuantity,
		pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
		"Standard Reserved Instance",
		"Hours",
	)

	// Capacity Reservation (Sections 3.6, 3.7)
	builder.WithCapacityReservation(
		"cr-0abc123456789def0",
		pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
	)

	// Invoice Details (Section 2.13)
	builder.WithInvoice("INV-2025-11-001", "Amazon Web Services, Inc.")

	// Metadata & Extension (Section 2.14)
	builder.WithTags(map[string]string{
		"Environment": "Production",
		"CostCenter":  "CC-Engineering-001",
		"Project":     "WebPlatform",
		"Owner":       "platform-team@example.com",
		"Application": "web-server",
		"ManagedBy":   "Pulumi",
	})

	// ExtendedColumns (The "Backpack" for provider-specific data)
	builder.WithExtension("aws:instance_lifecycle", "normal")
	builder.WithExtension("aws:tenancy", "default")
	builder.WithExtension("aws:purchase_option", "Reserved")
	builder.WithExtension("pulumi:stack", "production")
	builder.WithExtension("pulumi:project", "web-infrastructure")
}
