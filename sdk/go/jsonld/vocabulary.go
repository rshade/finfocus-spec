package jsonld

// FOCUS vocabulary namespace and term definitions for JSON-LD context.
//
// This package provides constants for the FOCUS-specific vocabulary namespace
// used in JSON-LD output. Terms without Schema.org equivalents use the
// FOCUS namespace for proper RDF semantics.
const (
	// JSON-LD keywords.
	JSONLDTypeKey = "@type"

	// FocusNamespace is the base IRI for FOCUS vocabulary terms.
	FocusNamespace = "https://focus.finops.org/v1#"

	// Record type identifiers.
	FocusCostRecordType    = "focus:FocusCostRecord"
	ContractCommitmentType = "focus:ContractCommitment"

	// Identity fields.
	BillingAccountID   = "focus:billingAccountId"
	BillingAccountName = "focus:billingAccountName"
	BillingAccountType = "focus:billingAccountType"
	SubAccountID       = "focus:subAccountId"
	SubAccountName     = "focus:subAccountName"
	SubAccountType     = "focus:subAccountType"

	// Period fields.
	BillingPeriodStart = "focus:billingPeriodStart"
	BillingPeriodEnd   = "focus:billingPeriodEnd"
	ChargePeriodStart  = "focus:chargePeriodStart"
	ChargePeriodEnd    = "focus:chargePeriodEnd"

	// Charge fields.
	ChargeCategory    = "focus:chargeCategory"
	ChargeClass       = "focus:chargeClass"
	ChargeDescription = "focus:chargeDescription"
	ChargeFrequency   = "focus:chargeFrequency"

	// Pricing fields.
	PricingCategory = "focus:pricingCategory"
	PricingQuantity = "focus:pricingQuantity"
	PricingUnit     = "focus:pricingUnit"
	ListUnitPrice   = "focus:listUnitPrice"

	// Service fields.
	ServiceCategory    = "focus:serviceCategory"
	ServiceName        = "focus:serviceName"
	ServiceSubcategory = "focus:serviceSubcategory"

	// Resource fields.
	ResourceID   = "focus:resourceId"
	ResourceName = "focus:resourceName"
	ResourceType = "focus:resourceType"

	// SKU fields.
	SkuID           = "focus:skuId"
	SkuPriceID      = "focus:skuPriceId"
	SkuMeter        = "focus:skuMeter"
	SkuPriceDetails = "focus:skuPriceDetails"

	// Region fields.
	RegionID         = "focus:regionId"
	RegionName       = "focus:regionName"
	AvailabilityZone = "focus:availabilityZone"

	// Cost fields.
	BilledCost          = "focus:billedCost"
	ListCost            = "focus:listCost"
	EffectiveCost       = "focus:effectiveCost"
	ContractedCost      = "focus:contractedCost"
	ContractedUnitPrice = "focus:contractedUnitPrice"

	// Consumption fields.
	ConsumedQuantity = "focus:consumedQuantity"
	ConsumedUnit     = "focus:consumedUnit"

	// Commitment discount fields.
	CommitmentDiscountCategory = "focus:commitmentDiscountCategory"
	CommitmentDiscountID       = "focus:commitmentDiscountId"
	CommitmentDiscountName     = "focus:commitmentDiscountName"
	CommitmentDiscountQuantity = "focus:commitmentDiscountQuantity"
	CommitmentDiscountStatus   = "focus:commitmentDiscountStatus"
	CommitmentDiscountType     = "focus:commitmentDiscountType"
	CommitmentDiscountUnit     = "focus:commitmentDiscountUnit"

	// Capacity reservation fields.
	CapacityReservationID     = "focus:capacityReservationId"
	CapacityReservationStatus = "focus:capacityReservationStatus"

	// Invoice fields.
	InvoiceID     = "focus:invoiceId"
	InvoiceIssuer = "focus:invoiceIssuer"

	// Map fields.
	Tags            = "focus:tags"
	ExtendedColumns = "focus:extendedColumns"

	// Deprecated fields (marked with schema:supersededBy).
	ProviderName = "focus:providerName"
	Publisher    = "focus:publisher"

	// FOCUS 1.3 service provider fields.
	ServiceProviderName = "focus:serviceProviderName"
	HostProviderName    = "focus:hostProviderName"

	// FOCUS 1.3 allocation fields.
	AllocatedMethodID      = "focus:allocatedMethodId"
	AllocatedMethodDetails = "focus:allocatedMethodDetails"
	AllocatedResourceID    = "focus:allocatedResourceId"
	AllocatedResourceName  = "focus:allocatedResourceName"
	AllocatedTags          = "focus:allocatedTags"

	// Contract reference.
	ContractApplied = "focus:contractApplied"

	// Contract commitment fields.
	ContractCommitmentID          = "focus:contractCommitmentId"
	ContractID                    = "focus:contractId"
	ContractCommitmentCategory    = "focus:contractCommitmentCategory"
	ContractCommitmentTypeEnum    = "focus:contractCommitmentType"
	ContractCommitmentPeriodStart = "focus:contractCommitmentPeriodStart"
	ContractCommitmentPeriodEnd   = "focus:contractCommitmentPeriodEnd"
	ContractPeriodStart           = "focus:contractPeriodStart"
	ContractPeriodEnd             = "focus:contractPeriodEnd"
	ContractCommitmentCost        = "focus:contractCommitmentCost"
	ContractCommitmentQuantity    = "focus:contractCommitmentQuantity"
	ContractCommitmentUnit        = "focus:contractCommitmentUnit"
	BillingCurrency               = "focus:billingCurrency"
)

//nolint:gochecknoglobals // Intentional optimization for zero-allocation lookup
var standardPrefixes = map[string]string{
	"schema": "https://schema.org/",
	"focus":  FocusNamespace,
	"xsd":    "http://www.w3.org/2001/XMLSchema#",
}

// StandardPrefixes returns a map of standard RDF prefixes for JSON-LD context.
// Uses zero-allocation lookup via package-level map.
//
// WARNING: The returned map is the package-level variable. Callers must not
// modify it. Modifications will affect all callers and may cause data races.
func StandardPrefixes() map[string]string {
	return standardPrefixes
}
