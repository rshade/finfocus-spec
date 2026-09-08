package pluginsdk

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Deprecation warning rate-limiters to prevent log spam in high-volume plugins.
// Each warning fires at most once per process lifetime.
//
//nolint:gochecknoglobals // Intentional: rate-limiting requires package-level state
var (
	providerNameWarningOnce sync.Once
	publisherWarningOnce    sync.Once
)

// FocusRecordBuilder handles the construction of FOCUS cost records.
//
// # FOCUS Version Compatibility
//
// This builder supports both FOCUS 1.2 and FOCUS 1.3 specifications:
//   - FOCUS 1.2: Original columns (all existing methods)
//   - FOCUS 1.3: New columns for split cost allocation, provider identification, and contract commitments
//
// # Migration Guide: FOCUS 1.2 to FOCUS 1.3
//
// ## Deprecated Fields (FOCUS 1.3)
//
// The following fields are deprecated in FOCUS 1.3 but remain functional for backward compatibility:
//
//   - provider_name (field 1) → Use service_provider_name via WithServiceProvider()
//   - publisher (field 55) → Use host_provider_name via WithHostProvider()
//
// When both deprecated and replacement fields are set, a warning is logged and the
// FOCUS 1.3 field takes precedence. Existing code using WithIdentity() continues to work
// unchanged - it sets provider_name for backward compatibility.
//
// ## Migration Steps
//
//  1. Replace direct provider_name usage with WithServiceProvider(name)
//  2. Replace direct publisher usage with WithHostProvider(name)
//  3. For marketplace scenarios, set both service_provider_name (ISV/reseller) and
//     host_provider_name (underlying cloud platform)
//  4. Adopt new allocation methods (WithAllocation, WithAllocatedResource, WithAllocatedTags)
//     for split cost allocation scenarios
//  5. Link to contract commitments using WithContractApplied(commitmentId)
//
// ## Example Migration
//
//	// FOCUS 1.2 (still works - no changes required)
//	builder.WithIdentity("AWS", billingAccountID, billingAccountName)
//
//	// FOCUS 1.3 (preferred for new code)
//	builder.WithIdentity("AWS", billingAccountID, billingAccountName).
//	    WithServiceProvider("AWS").   // New FOCUS 1.3 field
//	    WithHostProvider("AWS")       // New FOCUS 1.3 field
//
// ## Marketplace Scenario Examples
//
// Before (FOCUS 1.2) - Limited provider visibility:
//
//	builder.WithIdentity("AWS", billingAccountID, billingAccountName).
//	    WithPublisher("Datadog")  // Only publisher, no service/host distinction
//
// After (FOCUS 1.3) - Clear provider chain:
//
//	builder.WithIdentity("AWS", billingAccountID, billingAccountName).
//	    WithServiceProvider("Datadog").  // ISV selling on AWS Marketplace
//	    WithHostProvider("AWS")          // Cloud platform hosting the service
//
// Azure Marketplace Example:
//
//	builder.WithIdentity("Azure", subscriptionID, subscriptionName).
//	    WithServiceProvider("Confluent"). // Kafka vendor on Azure Marketplace
//	    WithHostProvider("Azure")         // Azure hosts the underlying resources
//
// GCP Marketplace Example:
//
//	builder.WithIdentity("GCP", billingAccountID, projectName).
//	    WithServiceProvider("MongoDB").   // MongoDB Atlas via GCP Marketplace
//	    WithHostProvider("GCP")           // GCP hosts the infrastructure
//
// Direct Cloud Usage (no marketplace):
//
//	builder.WithIdentity("AWS", billingAccountID, billingAccountName).
//	    WithServiceProvider("AWS").       // AWS is both service provider
//	    WithHostProvider("AWS")           // and host provider
type FocusRecordBuilder struct {
	record *pbc.FocusCostRecord
}

// NewFocusRecordBuilder creates a new builder instance.
func NewFocusRecordBuilder() *FocusRecordBuilder {
	return &FocusRecordBuilder{
		record: &pbc.FocusCostRecord{
			ExtendedColumns: make(map[string]string),
			Tags:            make(map[string]string),
		},
	}
}

// WithIdentity sets the identity and hierarchy fields per FOCUS 1.2 Section 2.1.
//
// FOCUS 1.3 Migration Note: The providerName parameter sets the deprecated provider_name field.
// For new FOCUS 1.3 code, prefer using WithServiceProvider() and WithHostProvider() instead.
// Existing code using this method continues to work for backward compatibility.
//
// See FocusRecordBuilder documentation for complete migration guidance.
func (b *FocusRecordBuilder) WithIdentity(
	providerName, billingAccountID, billingAccountName string,
) *FocusRecordBuilder {
	//nolint:staticcheck // SA1019: Intentionally setting deprecated field for backward compatibility
	b.record.ProviderName = providerName
	b.record.BillingAccountId = billingAccountID
	b.record.BillingAccountName = billingAccountName
	return b
}

// WithSubAccount sets the sub-account fields (e.g., AWS Account, Azure Subscription, GCP Project) per FOCUS 1.2 Section 2.1.
func (b *FocusRecordBuilder) WithSubAccount(subAccountID, subAccountName string) *FocusRecordBuilder {
	b.record.SubAccountId = subAccountID
	b.record.SubAccountName = subAccountName
	return b
}

// WithBillingPeriod sets the billing period start/end and currency per FOCUS 1.2 Section 2.2.
func (b *FocusRecordBuilder) WithBillingPeriod(start, end time.Time, currency string) *FocusRecordBuilder {
	b.record.BillingPeriodStart = timestamppb.New(start)
	b.record.BillingPeriodEnd = timestamppb.New(end)
	b.record.BillingCurrency = currency
	return b
}

// WithChargePeriod sets the start and end time of the charge per FOCUS 1.2 Section 2.3.
func (b *FocusRecordBuilder) WithChargePeriod(start, end time.Time) *FocusRecordBuilder {
	b.record.ChargePeriodStart = timestamppb.New(start)
	b.record.ChargePeriodEnd = timestamppb.New(end)
	return b
}

// WithServiceCategory sets the service category per FOCUS 1.2 Section 2.6.
func (b *FocusRecordBuilder) WithServiceCategory(category pbc.FocusServiceCategory) *FocusRecordBuilder {
	b.record.ServiceCategory = category
	return b
}

// WithChargeDetails sets the charge category and pricing category per FOCUS 1.2 Section 2.4.
func (b *FocusRecordBuilder) WithChargeDetails(
	chargeCat pbc.FocusChargeCategory,
	pricingCat pbc.FocusPricingCategory,
) *FocusRecordBuilder {
	b.record.ChargeCategory = chargeCat
	b.record.PricingCategory = pricingCat
	return b
}

// WithChargeClassification sets the charge class, description, and frequency per FOCUS 1.2 Section 2.4.
func (b *FocusRecordBuilder) WithChargeClassification(
	chargeClass pbc.FocusChargeClass,
	description string,
	frequency pbc.FocusChargeFrequency,
) *FocusRecordBuilder {
	b.record.ChargeClass = chargeClass
	b.record.ChargeDescription = description
	b.record.ChargeFrequency = frequency
	return b
}

// WithPricing sets the pricing quantity, unit, and list unit price per FOCUS 1.2 Section 2.5.
func (b *FocusRecordBuilder) WithPricing(quantity float64, unit string, listUnitPrice float64) *FocusRecordBuilder {
	b.record.PricingQuantity = quantity
	b.record.PricingUnit = unit
	b.record.ListUnitPrice = listUnitPrice
	return b
}

// WithFinancials sets the cost amounts and currency per FOCUS 1.2 Section 2.10.
func (b *FocusRecordBuilder) WithFinancials(
	billed, list, effective float64,
	currency, invoiceID string,
) *FocusRecordBuilder {
	b.record.BilledCost = billed
	b.record.ListCost = list
	b.record.EffectiveCost = effective
	b.record.BillingCurrency = currency
	b.record.InvoiceId = invoiceID
	return b
}

// WithUsage sets the consumed quantity and unit per FOCUS 1.2 Section 2.11.
func (b *FocusRecordBuilder) WithUsage(quantity float64, unit string) *FocusRecordBuilder {
	b.record.ConsumedQuantity = quantity
	b.record.ConsumedUnit = unit
	return b
}

// WithResource sets the resource details per FOCUS 1.2 Section 2.7.
func (b *FocusRecordBuilder) WithResource(resourceID, resourceName, resourceType string) *FocusRecordBuilder {
	b.record.ResourceId = resourceID
	b.record.ResourceName = resourceName
	b.record.ResourceType = resourceType
	return b
}

// WithService sets the service details per FOCUS 1.2 Section 2.6.
func (b *FocusRecordBuilder) WithService(
	category pbc.FocusServiceCategory,
	serviceName string,
) *FocusRecordBuilder {
	b.record.ServiceCategory = category
	b.record.ServiceName = serviceName
	return b
}

// WithSKU sets the SKU details per FOCUS 1.2 Section 2.8.
func (b *FocusRecordBuilder) WithSKU(skuID, skuPriceID string) *FocusRecordBuilder {
	b.record.SkuId = skuID
	b.record.SkuPriceId = skuPriceID
	return b
}

// WithLocation sets the region and availability zone per FOCUS 1.2 Section 2.9.
func (b *FocusRecordBuilder) WithLocation(regionID, regionName, availabilityZone string) *FocusRecordBuilder {
	b.record.RegionId = regionID
	b.record.RegionName = regionName
	b.record.AvailabilityZone = availabilityZone
	return b
}

// WithCommitmentDiscount sets the commitment discount details per FOCUS 1.2 Section 2.12.
func (b *FocusRecordBuilder) WithCommitmentDiscount(
	category pbc.FocusCommitmentDiscountCategory,
	discountID, discountName string,
) *FocusRecordBuilder {
	b.record.CommitmentDiscountCategory = category
	b.record.CommitmentDiscountId = discountID
	b.record.CommitmentDiscountName = discountName
	return b
}

// WithInvoice sets the invoice details per FOCUS 1.2 Section 2.13.
func (b *FocusRecordBuilder) WithInvoice(invoiceID, invoiceIssuer string) *FocusRecordBuilder {
	b.record.InvoiceId = invoiceID
	b.record.InvoiceIssuer = invoiceIssuer
	return b
}

// WithTag adds a single tag key-value pair per FOCUS 1.2 Section 2.14.
func (b *FocusRecordBuilder) WithTag(key, value string) *FocusRecordBuilder {
	b.record.Tags[key] = value
	return b
}

// WithTags merges the provided tags into the record's tag map per FOCUS 1.2 Section 2.14.
//
// Copy Semantics (Zero-Allocation Pattern):
//   - Tags are copied into the builder's internal map, NOT assigned by reference.
//   - The input map can be safely modified after this call without affecting the record.
//   - For performance-critical code processing thousands of records, consider using
//     WithTag(key, value) to avoid map iteration overhead.
//
// Thread Safety: NOT thread-safe. Do not call from multiple goroutines.
func (b *FocusRecordBuilder) WithTags(tags map[string]string) *FocusRecordBuilder {
	for k, v := range tags {
		b.record.Tags[k] = v
	}
	return b
}

// WithExtension adds a key-value pair to the extended columns (Backpack) per FOCUS 1.2 Section 2.14.
func (b *FocusRecordBuilder) WithExtension(key, value string) *FocusRecordBuilder {
	b.record.ExtendedColumns[key] = value
	return b
}

// =============================================================================
// FOCUS 1.2 New Column Builder Methods (19 new columns)
// =============================================================================

// WithContractedCost sets the contracted cost per FOCUS 1.2 Section 3.20.
// This is a MANDATORY field representing the cost calculated by multiplying
// contracted unit price and pricing quantity.
func (b *FocusRecordBuilder) WithContractedCost(cost float64) *FocusRecordBuilder {
	b.record.ContractedCost = cost
	return b
}

// WithBillingAccountType sets the billing account type per FOCUS 1.2 Section 3.3.
// This is a CONDITIONAL field representing the provider-assigned name to identify
// the type of billing account.
func (b *FocusRecordBuilder) WithBillingAccountType(accountType string) *FocusRecordBuilder {
	b.record.BillingAccountType = accountType
	return b
}

// WithSubAccountType sets the sub-account type per FOCUS 1.2 Section 3.45.
// This is a CONDITIONAL field representing the provider-assigned identifier
// for sub-account classification.
func (b *FocusRecordBuilder) WithSubAccountType(accountType string) *FocusRecordBuilder {
	b.record.SubAccountType = accountType
	return b
}

// WithCapacityReservation sets the capacity reservation details per FOCUS 1.2 Sections 3.6, 3.7.
// This includes the capacity reservation ID and its utilization status.
// Both fields are CONDITIONAL.
func (b *FocusRecordBuilder) WithCapacityReservation(
	reservationID string,
	status pbc.FocusCapacityReservationStatus,
) *FocusRecordBuilder {
	b.record.CapacityReservationId = reservationID
	b.record.CapacityReservationStatus = status
	return b
}

// WithCommitmentDiscountDetails sets extended commitment discount details per FOCUS 1.2.
// Includes quantity (Section 3.14), status (Section 3.17), type (Section 3.18),
// and unit (Section 3.19). All fields are CONDITIONAL.
func (b *FocusRecordBuilder) WithCommitmentDiscountDetails(
	quantity float64,
	status pbc.FocusCommitmentDiscountStatus,
	discountType string,
	unit string,
) *FocusRecordBuilder {
	b.record.CommitmentDiscountQuantity = quantity
	b.record.CommitmentDiscountStatus = status
	b.record.CommitmentDiscountType = discountType
	b.record.CommitmentDiscountUnit = unit
	return b
}

// WithContractedUnitPrice sets the contracted unit price per FOCUS 1.2 Section 3.21.
// This is a CONDITIONAL field representing the agreed-upon unit price per
// pricing unit for the associated SKU.
func (b *FocusRecordBuilder) WithContractedUnitPrice(price float64) *FocusRecordBuilder {
	b.record.ContractedUnitPrice = price
	return b
}

// WithPricingCurrency sets the pricing currency per FOCUS 1.2 Section 3.34.
// This is a CONDITIONAL field used when pricing is in a different currency
// than billing. Format: ISO 4217 currency code.
func (b *FocusRecordBuilder) WithPricingCurrency(currency string) *FocusRecordBuilder {
	b.record.PricingCurrency = currency
	return b
}

// WithPricingCurrencyPrices sets pricing currency amounts per FOCUS 1.2.
// Includes contracted unit price (Section 3.35), effective cost (Section 3.36),
// and list unit price (Section 3.37). All fields are CONDITIONAL.
func (b *FocusRecordBuilder) WithPricingCurrencyPrices(
	contractedUnitPrice float64,
	effectiveCost float64,
	listUnitPrice float64,
) *FocusRecordBuilder {
	b.record.PricingCurrencyContractedUnitPrice = contractedUnitPrice
	b.record.PricingCurrencyEffectiveCost = effectiveCost
	b.record.PricingCurrencyListUnitPrice = listUnitPrice
	return b
}

// WithPublisher sets the publisher per FOCUS 1.2 Section 3.39.
// This is a CONDITIONAL field representing the entity that published the
// service or product.
//
// Deprecated: FOCUS 1.3 deprecates publisher in favor of host_provider_name.
// Use WithHostProvider() for new code. This method remains for backward compatibility.
// If both publisher and host_provider_name are set, a warning is logged during Build().
func (b *FocusRecordBuilder) WithPublisher(publisher string) *FocusRecordBuilder {
	b.record.Publisher = publisher
	return b
}

// WithServiceSubcategory sets the service subcategory per FOCUS 1.2 Section 3.43.
// This is a CONDITIONAL field providing granular service classification
// supporting functional categorization.
func (b *FocusRecordBuilder) WithServiceSubcategory(subcategory string) *FocusRecordBuilder {
	b.record.ServiceSubcategory = subcategory
	return b
}

// WithSkuDetails sets SKU meter and price details per FOCUS 1.2 Sections 3.46, 3.48.
// SkuMeter is the provider-assigned meter identifier, and SkuPriceDetails contains
// additional pricing information. Both fields are CONDITIONAL.
func (b *FocusRecordBuilder) WithSkuDetails(meter, priceDetails string) *FocusRecordBuilder {
	b.record.SkuMeter = meter
	b.record.SkuPriceDetails = priceDetails
	return b
}

// =============================================================================
// FOCUS 1.3 Split Cost Allocation Builder Methods
// =============================================================================

// WithAllocation sets the cost allocation methodology per FOCUS 1.3.
// AllocatedMethodId identifies the allocation methodology used (e.g., "proportional-cpu").
// AllocatedMethodDetails provides a human-readable description of how costs were split.
// Validation: If methodId is set, WithAllocatedResource MUST also be called.
// FOCUS 1.3 Section: Allocated Method ID, Allocated Method Details.
func (b *FocusRecordBuilder) WithAllocation(
	methodID, methodDetails string,
) *FocusRecordBuilder {
	b.record.AllocatedMethodId = methodID
	b.record.AllocatedMethodDetails = methodDetails
	return b
}

// WithAllocatedResource sets the target resource receiving allocated cost per FOCUS 1.3.
// AllocatedResourceId is the identifier of the resource receiving the allocated cost.
// AllocatedResourceName is the display name of that resource.
// FOCUS 1.3 Section: Allocated Resource ID, Allocated Resource Name.
func (b *FocusRecordBuilder) WithAllocatedResource(
	resourceID, resourceName string,
) *FocusRecordBuilder {
	b.record.AllocatedResourceId = resourceID
	b.record.AllocatedResourceName = resourceName
	return b
}

// WithAllocatedTags sets tags associated with the allocated resource per FOCUS 1.3.
// Copies tags to avoid external modification of builder state.
// Follows the same map<string, string> pattern as the existing WithTags method.
// FOCUS 1.3 Section: Allocated Tags.
//
// Performance note: This method copies the input map (~130 ns/op overhead).
// For high-volume plugins processing thousands of records/second, consider
// pre-allocating tags or using WithAllocatedResource for simpler cases.
func (b *FocusRecordBuilder) WithAllocatedTags(
	tags map[string]string,
) *FocusRecordBuilder {
	if len(tags) == 0 {
		return b // No-op for nil/empty
	}
	if b.record.AllocatedTags == nil {
		b.record.AllocatedTags = make(map[string]string, len(tags)) // Pre-size
	}
	// Copy to avoid external mutation
	for k, v := range tags {
		b.record.AllocatedTags[k] = v
	}
	return b
}

// =============================================================================
// FOCUS 1.3 Provider Identification Builder Methods
// =============================================================================

// WithServiceProvider sets the service provider name per FOCUS 1.3.
// This identifies the entity that makes the service available for purchase.
// In reseller/marketplace scenarios, this is the ISV or reseller.
// Replaces deprecated provider_name field.
// FOCUS 1.3 Section: Service Provider Name.
func (b *FocusRecordBuilder) WithServiceProvider(
	name string,
) *FocusRecordBuilder {
	b.record.ServiceProviderName = name
	return b
}

// WithHostProvider sets the host provider name per FOCUS 1.3.
// This identifies the entity that hosts the underlying resource or service.
// This is where the workload actually runs (e.g., AWS, Azure, GCP).
// Replaces deprecated publisher field.
// FOCUS 1.3 Section: Host Provider Name.
func (b *FocusRecordBuilder) WithHostProvider(
	name string,
) *FocusRecordBuilder {
	b.record.HostProviderName = name
	return b
}

// =============================================================================
// FOCUS 1.3 Contract Commitment Link Builder Method
// =============================================================================

// WithContractApplied sets the contract commitment reference per FOCUS 1.3.
// This links the cost record to a ContractCommitmentId in the Contract
// Commitment supplemental dataset. Treated as an opaque reference (no
// cross-dataset validation is performed).
// FOCUS 1.3 Section: Contract Applied.
func (b *FocusRecordBuilder) WithContractApplied(
	commitmentID string,
) *FocusRecordBuilder {
	b.record.ContractApplied = commitmentID
	return b
}

// WithProfileDefaults applies usage profile-based defaults to fields that haven't been set.
// This is a convenience method for plugins implementing profile-aware cost estimation.
//
// The method only sets defaults for fields that are currently unset (zero values):
//   - PROD: No changes (standard assumptions apply)
//   - DEV: Sets extended_columns["usage_profile"] = "dev" for tracking
//   - BURST: Sets extended_columns["usage_profile"] = "burst" for tracking
//   - UNSPECIFIED: No changes (intentionally excluded from extended_columns to distinguish
//     between "no profile specified" vs "explicitly profiled workload")
//
// Plugins should call this after setting other fields but before Build().
// This method does NOT modify cost values - that logic belongs in the plugin's
// cost calculation based on the profile-specific hours (see DefaultMonthlyHours).
//
// Example:
//
//	builder.WithIdentity("AWS", accountID, accountName).
//	    WithResource(resourceID, name, resourceType).
//	    WithProfileDefaults(req.GetUsageProfile()).
//	    Build()
func (b *FocusRecordBuilder) WithProfileDefaults(profile pbc.UsageProfile) *FocusRecordBuilder {
	// Normalize unknown profiles to UNSPECIFIED
	normalizedProfile := NormalizeUsageProfile(profile)

	// Set extended column to track which profile was applied
	profileStr := UsageProfileString(normalizedProfile)
	if profileStr != usageProfileUnspecifiedLabel {
		if b.record.ExtendedColumns == nil {
			b.record.ExtendedColumns = make(map[string]string)
		}
		if _, exists := b.record.GetExtendedColumns()["usage_profile"]; !exists {
			b.record.ExtendedColumns["usage_profile"] = profileStr
		}
	}

	return b
}

// Build validates and returns the constructed FocusCostRecord.
// FOCUS 1.3 deprecation warnings are logged when deprecated fields are used
// alongside their replacement fields.
func (b *FocusRecordBuilder) Build() (*pbc.FocusCostRecord, error) {
	// Log deprecation warnings for FOCUS 1.3 field migrations.
	b.logDeprecationWarnings()

	if err := ValidateFocusRecord(b.record); err != nil {
		return nil, err
	}
	return b.record, nil
}

// logDeprecationWarnings logs warnings when deprecated fields are used alongside
// their FOCUS 1.3 replacement fields. The new fields take precedence.
//
// Rate-limiting: Each warning type fires at most once per process lifetime to
// prevent log spam in high-volume plugins (thousands of records/second).
func (b *FocusRecordBuilder) logDeprecationWarnings() {
	// provider_name (deprecated) -> service_provider_name (FOCUS 1.3)
	//nolint:staticcheck // SA1019: Intentionally accessing deprecated field to detect and warn about dual usage
	if b.record.GetProviderName() != "" && b.record.GetServiceProviderName() != "" {
		providerNameWarningOnce.Do(func() {
			log.Warn().
				Str("deprecated_field", "provider_name").
				Str("replacement_field", "service_provider_name").
				Msg("FOCUS 1.3: provider_name is deprecated, using service_provider_name (this warning shown once)")
		})
	}

	// publisher (deprecated) -> host_provider_name (FOCUS 1.3)
	//nolint:staticcheck // SA1019: Intentionally accessing deprecated field to detect and warn about dual usage
	if b.record.GetPublisher() != "" && b.record.GetHostProviderName() != "" {
		publisherWarningOnce.Do(func() {
			log.Warn().
				Str("deprecated_field", "publisher").
				Str("replacement_field", "host_provider_name").
				Msg("FOCUS 1.3: publisher is deprecated, using host_provider_name (this warning shown once)")
		})
	}
}
